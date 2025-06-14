package dcb

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"
)

// convertTagsToJSON converts a slice of tags to JSON bytes
func convertTagsToJSON(tags []Tag) ([]byte, error) {
	tagMap := make(map[string]string)
	for _, t := range tags {
		tagMap[t.Key] = t.Value
	}
	return json.Marshal(tagMap)
}

// prepareEventBatch prepares arrays for batch insert from events
func prepareEventBatch(events []InputEvent) ([]string, []string, [][]byte, [][]byte, []string, []string, error) {
	ids := make([]string, len(events))
	types := make([]string, len(events))
	tagsJSON := make([][]byte, len(events))
	data := make([][]byte, len(events))
	causationIDs := make([]string, len(events))
	correlationIDs := make([]string, len(events))

	for i, e := range events {
		// Generate TypeID for event based on sorted tag keys
		eventID := generateTagBasedTypeID(e.Tags)
		ids[i] = eventID

		types[i] = e.Type
		data[i] = e.Data

		// Convert tags to JSONB
		jsonBytes, err := convertTagsToJSON(e.Tags)
		if err != nil {
			log.Printf("Failed to marshal tags for event %d: %v", i, err)
			return nil, nil, nil, nil, nil, nil, fmt.Errorf("failed to marshal tags for event %d: %w", i, err)
		}
		tagsJSON[i] = jsonBytes

		// Set causation_id
		if i > 0 {
			causationIDs[i] = ids[i-1] // Previous event's TypeID
		} else {
			causationIDs[i] = eventID // Self-caused
		}

		// Set correlation_id
		if i == 0 {
			correlationIDs[i] = eventID // Root event
		} else {
			correlationIDs[i] = correlationIDs[0] // Same correlation chain
		}

		// Log event relationships
		log.Printf("Appending event %d: ID=%s, CausationID=%s, CorrelationID=%s", i, eventID, causationIDs[i], correlationIDs[i])
	}

	return ids, types, tagsJSON, data, causationIDs, correlationIDs, nil
}

// executeBatchInsert executes the batch insert and returns positions
func executeBatchInsert(ctx context.Context, tx pgx.Tx, events []InputEvent, ids []string, types []string, tagsJSON [][]byte, data [][]byte, causationIDs []string, correlationIDs []string) ([]int64, error) {
	batch := &pgx.Batch{}
	positions := make([]int64, len(events))

	// Add insert statements to batch
	for i := range events {
		batch.Queue(`
			INSERT INTO events (id, type, tags, data, causation_id, correlation_id)
			VALUES ($1, $2, $3::jsonb, $4::jsonb, $5, $6)
			RETURNING position
		`, ids[i], types[i], tagsJSON[i], data[i], causationIDs[i], correlationIDs[i])
	}

	// Execute batch
	br := tx.SendBatch(ctx, batch)
	defer br.Close()

	// Get results
	for i := range events {
		err := br.QueryRow().Scan(&positions[i])
		if err != nil {
			if err.Error() == "ERROR: insert or update on table \"events\" violates foreign key constraint \"events_causation_id_fkey\" (SQLSTATE 23503)" ||
				err.Error() == "ERROR: insert or update on table \"events\" violates foreign key constraint \"events_correlation_id_fkey\" (SQLSTATE 23503)" {
				return nil, &ValidationError{
					EventStoreError: EventStoreError{
						Op:  "executeBatchInsert",
						Err: fmt.Errorf("foreign key violation: one or more causation_id or correlation_id values are invalid"),
					},
					Field: "causation_id/correlation_id",
					Value: "batch",
				}
			}
			return nil, &EventStoreError{
				Op:  "executeBatchInsert",
				Err: fmt.Errorf("failed to insert event %d: %w", i, err),
			}
		}
	}

	return positions, nil
}

// checkForConflictingEvents checks for conflicting events in optimistic locking
func checkForConflictingEvents(ctx context.Context, tx pgx.Tx, query Query, queryTagsJSON []byte, latestPosition int64) error {
	if len(query.Items) == 0 {
		return nil // No query items, no conflict check needed
	}

	// For optimistic locking, we only check the first query item
	// This maintains backward compatibility while supporting the new structure
	item := query.Items[0]

	// Convert item tags to JSONB
	itemTagMap := make(map[string]string)
	for _, t := range item.Tags {
		itemTagMap[t.Key] = t.Value
	}
	itemTagsJSON, err := json.Marshal(itemTagMap)
	if err != nil {
		return &EventStoreError{
			Op:  "checkForConflictingEvents",
			Err: fmt.Errorf("failed to marshal query tags: %w", err),
		}
	}

	var exists bool
	checkQuery := `
		SELECT EXISTS(
			SELECT 1 FROM events 
			WHERE position > $1 
			  AND tags @> $2::jsonb
			  AND ($3::text[] IS NULL OR
				   array_length($3::text[], 1) = 0 OR
				   type = ANY($3::text[]))
		)
	`
	err = tx.QueryRow(ctx, checkQuery, latestPosition, itemTagsJSON, item.EventTypes).Scan(&exists)
	if err != nil {
		return &EventStoreError{
			Op:  "checkForConflictingEvents",
			Err: fmt.Errorf("failed to check for conflicting events: %w", err),
		}
	}

	if exists {
		return &ConcurrencyError{
			EventStoreError: EventStoreError{
				Op:  "checkForConflictingEvents",
				Err: fmt.Errorf("conflicting events found after position %d", latestPosition),
			},
			ExpectedPosition: latestPosition,
			ActualPosition:   latestPosition + 1, // Since we found a conflicting event, it must be at the next position
		}
	}

	return nil
}

// checkForMatchingEvents checks if any events match the append condition
func checkForMatchingEvents(ctx context.Context, tx pgx.Tx, condition AppendCondition, queryTagsJSON []byte) error {
	if len(condition.FailIfEventsMatch.Items) == 0 {
		return nil // No query items, no check needed
	}

	// For append conditions, we only check the first query item
	// This maintains backward compatibility while supporting the new structure
	item := condition.FailIfEventsMatch.Items[0]

	// Use the passed queryTagsJSON instead of re-converting
	// This ensures consistency with the calling function
	itemTagsJSON := queryTagsJSON

	var exists bool
	checkQuery := `
		SELECT EXISTS(
			SELECT 1 FROM events 
			WHERE tags @> $1::jsonb
	`
	args := []interface{}{itemTagsJSON}
	argIndex := 2

	// Add position filtering if specified
	if condition.After != nil {
		checkQuery += fmt.Sprintf(" AND position > $%d", argIndex)
		args = append(args, *condition.After)
		argIndex++
	}

	// Add event type filtering if specified
	if len(item.EventTypes) > 0 {
		checkQuery += fmt.Sprintf(" AND type = ANY($%d)", argIndex)
		args = append(args, item.EventTypes)
		argIndex++
	}

	checkQuery += ")"

	err := tx.QueryRow(ctx, checkQuery, args...).Scan(&exists)
	if err != nil {
		return &EventStoreError{
			Op:  "checkForMatchingEvents",
			Err: fmt.Errorf("failed to check for matching events: %w", err),
		}
	}

	if exists {
		return &ConcurrencyError{
			EventStoreError: EventStoreError{
				Op:  "checkForMatchingEvents",
				Err: fmt.Errorf("events matching condition found"),
			},
			ExpectedPosition: 0,
			ActualPosition:   0,
		}
	}

	return nil
}
