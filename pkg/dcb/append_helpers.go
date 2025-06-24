package dcb

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"
)

// prepareEventBatch prepares arrays for batch insert from events
func prepareEventBatch(events []InputEvent) ([]string, [][]string, [][]byte, error) {
	types := make([]string, len(events))
	tags := make([][]string, len(events))
	data := make([][]byte, len(events))

	for i, e := range events {
		types[i] = e.Type
		data[i] = e.Data

		// Convert tags to TEXT[] format
		tagStrings := make([]string, len(e.Tags))
		for j, tag := range e.Tags {
			tagStrings[j] = tag.Key + ":" + tag.Value
		}
		tags[i] = tagStrings

		// Log event details
		log.Printf("Appending event %d: Type=%s", i, e.Type)
	}

	return types, tags, data, nil
}

// executeBatchInsert executes the batch insert and returns positions
func executeBatchInsert(ctx context.Context, tx pgx.Tx, events []InputEvent, types []string, tags [][]string, data [][]byte) ([]int64, error) {
	// Pre-allocate batch and results for better performance
	batch := &pgx.Batch{}
	positions := make([]int64, len(events))

	// Pre-allocate batch with known size
	batch.Queue("BEGIN")

	// Add insert statements to batch efficiently
	for i := range events {
		batch.Queue(`
			INSERT INTO events (type, tags, data)
			VALUES ($1, $2, $3)
			RETURNING position
		`, types[i], tags[i], data[i])
	}

	batch.Queue("COMMIT")

	// Execute batch
	br := tx.SendBatch(ctx, batch)
	defer br.Close()

	// Get results efficiently
	for i := range events {
		err := br.QueryRow().Scan(&positions[i])
		if err != nil {
			return nil, &EventStoreError{
				Op:  "executeBatchInsert",
				Err: fmt.Errorf("failed to insert event %d: %w", i, err),
			}
		}
	}

	return positions, nil
}

// checkForConflictingEvents checks for conflicting events in optimistic locking
func checkForConflictingEvents(ctx context.Context, tx pgx.Tx, query Query, latestPosition int64) error {
	if len(query.Items) == 0 {
		return nil // No query items, no conflict check needed
	}

	// For optimistic locking, we only check the first query item
	// This maintains backward compatibility while supporting the new structure
	item := query.Items[0]

	// Convert item tags to TEXT[] format
	itemTagsArray := TagsToArray(item.Tags)

	var exists bool
	checkQuery := `
		SELECT EXISTS(
			SELECT 1 FROM events 
			WHERE position > $1 
			  AND tags @> $2::text[]
			  AND ($3::text[] IS NULL OR
				   array_length($3::text[], 1) = 0 OR
				   type = ANY($3::text[]))
		)
	`
	err := tx.QueryRow(ctx, checkQuery, latestPosition, itemTagsArray, item.EventTypes).Scan(&exists)
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
func checkForMatchingEvents(ctx context.Context, tx pgx.Tx, condition AppendCondition) error {
	failIfEventsMatch := condition.getFailIfEventsMatch()
	if failIfEventsMatch == nil || len(failIfEventsMatch.Items) == 0 {
		return nil // No query items, no check needed
	}

	// For append conditions, we only check the first query item
	// This maintains backward compatibility while supporting the new structure
	item := failIfEventsMatch.Items[0]

	// Convert item tags to TEXT[] format
	itemTagsArray := TagsToArray(item.Tags)

	var exists bool
	checkQuery := `
		SELECT EXISTS(
			SELECT 1 FROM events 
			WHERE tags @> $1::text[]
	`
	args := []interface{}{itemTagsArray}
	argIndex := 2

	// Add position filtering if specified
	after := condition.getAfter()
	if after != nil {
		checkQuery += fmt.Sprintf(" AND position > $%d", argIndex)
		args = append(args, *after)
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
