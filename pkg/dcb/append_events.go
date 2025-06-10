package dcb

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// validateQueryTags validates the query tags and returns a ValidationError if invalid
func validateQueryTags(query Query) error {
	// Validate individual tags if present
	for i, t := range query.Tags {
		if t.Key == "" {
			return &ValidationError{
				EventStoreError: EventStoreError{
					Op:  "validateQueryTags",
					Err: fmt.Errorf("empty key in tag %d", i),
				},
				Field: "tag.key",
				Value: fmt.Sprintf("tag[%d]", i),
			}
		}
		if t.Value == "" {
			return &ValidationError{
				EventStoreError: EventStoreError{
					Op:  "validateQueryTags",
					Err: fmt.Errorf("empty value for key %s in tag %d", t.Key, i),
				},
				Field: fmt.Sprintf("tag[%d].value", i),
				Value: t.Key,
			}
		}
	}

	// Validate event types if present
	for i, eventType := range query.EventTypes {
		if eventType == "" {
			return &ValidationError{
				EventStoreError: EventStoreError{
					Op:  "validateQueryTags",
					Err: fmt.Errorf("empty event type at index %d", i),
				},
				Field: "eventTypes",
				Value: fmt.Sprintf("index[%d]", i),
			}
		}
	}

	return nil
}

// validateEvent validates a single event and returns a ValidationError if invalid
func validateEvent(e InputEvent, index int) error {
	// Validate event type
	if e.Type == "" {
		return &ValidationError{
			EventStoreError: EventStoreError{
				Op:  "validateEvent",
				Err: fmt.Errorf("empty type in event %d", index),
			},
			Field: "type",
			Value: fmt.Sprintf("event[%d]", index),
		}
	}

	// Validate event tags
	if len(e.Tags) == 0 {
		return &ValidationError{
			EventStoreError: EventStoreError{
				Op:  "validateEvent",
				Err: fmt.Errorf("empty tags in event %d", index),
			},
			Field: "tags",
			Value: fmt.Sprintf("event[%d]", index),
		}
	}
	for j, t := range e.Tags {
		if t.Key == "" {
			return &ValidationError{
				EventStoreError: EventStoreError{
					Op:  "validateEvent",
					Err: fmt.Errorf("empty key in tag %d of event %d", j, index),
				},
				Field: fmt.Sprintf("event[%d].tag[%d].key", index, j),
				Value: fmt.Sprintf("tag[%d]", j),
			}
		}
		if t.Value == "" {
			return &ValidationError{
				EventStoreError: EventStoreError{
					Op:  "validateEvent",
					Err: fmt.Errorf("empty value for key %s in tag %d of event %d", t.Key, j, index),
				},
				Field: fmt.Sprintf("event[%d].tag[%d].value", index, j),
				Value: t.Key,
			}
		}
	}

	// Validate Data as JSON
	if !json.Valid(e.Data) {
		return &ValidationError{
			EventStoreError: EventStoreError{
				Op:  "validateEvent",
				Err: fmt.Errorf("invalid JSON data in event %d", index),
			},
			Field: "data",
			Value: fmt.Sprintf("event[%d]", index),
		}
	}

	return nil
}

// validateBatchSize validates that the batch size is within limits
func (es *eventStore) validateBatchSize(events []InputEvent, operation string) error {
	if len(events) > es.maxBatchSize {
		return &ValidationError{
			EventStoreError: EventStoreError{
				Op:  operation,
				Err: fmt.Errorf("batch size %d exceeds maximum %d", len(events), es.maxBatchSize),
			},
			Field: "batchSize",
			Value: fmt.Sprintf("%d", len(events)),
		}
	}
	return nil
}

// validateEvents validates all events in a batch
func validateEvents(events []InputEvent) error {
	for i, event := range events {
		if err := validateEvent(event, i); err != nil {
			return err
		}
	}
	return nil
}

// convertTagsToJSON converts a slice of tags to JSON bytes
func convertTagsToJSON(tags []Tag) ([]byte, error) {
	tagMap := make(map[string]string)
	for _, t := range tags {
		tagMap[t.Key] = t.Value
	}
	return json.Marshal(tagMap)
}

// prepareEventBatch prepares arrays for batch insert from events
func prepareEventBatch(events []InputEvent) ([]pgtype.UUID, []string, [][]byte, [][]byte, []pgtype.UUID, []pgtype.UUID, error) {
	ids := make([]pgtype.UUID, len(events))
	types := make([]string, len(events))
	tagsJSON := make([][]byte, len(events))
	data := make([][]byte, len(events))
	causationIDs := make([]pgtype.UUID, len(events))
	correlationIDs := make([]pgtype.UUID, len(events))

	for i, e := range events {
		// Generate UUID for event (UUIDv7)
		uuidVal, err := uuid.NewV7()
		if err != nil {
			log.Printf("Failed to generate UUID for event %d: %v", i, err)
			return nil, nil, nil, nil, nil, nil, fmt.Errorf("failed to generate UUID for event %d: %w", i, err)
		}
		pgUUID := pgtype.UUID{}
		err = pgUUID.Scan(uuidVal.String())
		if err != nil {
			log.Printf("Failed to parse UUID for event %d: %v", i, err)
			return nil, nil, nil, nil, nil, nil, fmt.Errorf("failed to parse UUID for event %d: %w", i, err)
		}
		ids[i] = pgUUID

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
			causationIDs[i] = ids[i-1] // Previous event's ID
		} else {
			// For first event, set causation_id to its own ID (self-caused)
			causationIDs[i] = pgUUID
		}

		// Set correlation_id
		if i == 0 {
			// For first event, set correlation_id to its own ID
			correlationIDs[i] = pgUUID
		} else {
			// For subsequent events, use the correlation_id of the first event
			correlationIDs[i] = correlationIDs[0]
		}

		// Log event relationships
		causationIDStr := causationIDs[i].String()
		correlationIDStr := correlationIDs[i].String()
		log.Printf("Appending event %d: ID=%s, CausationID=%s, CorrelationID=%s", i, uuidVal.String(), causationIDStr, correlationIDStr)
	}

	return ids, types, tagsJSON, data, causationIDs, correlationIDs, nil
}

// executeBatchInsert executes the batch insert and returns positions
func executeBatchInsert(ctx context.Context, tx pgx.Tx, events []InputEvent, ids []pgtype.UUID, types []string, tagsJSON [][]byte, data [][]byte, causationIDs []pgtype.UUID, correlationIDs []pgtype.UUID) ([]int64, error) {
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
	if len(query.Tags) == 0 {
		return nil // No query tags, no conflict check needed
	}

	var exists bool
	checkQuery := `
		SELECT EXISTS (
			SELECT 1
			FROM events
			WHERE position > $1
			  AND tags @> $2::jsonb
			  AND ($3::text[] IS NULL OR
				   array_length($3::text[], 1) = 0 OR
				   type = ANY($3::text[]))
		)
	`
	err := tx.QueryRow(ctx, checkQuery, latestPosition, queryTagsJSON, query.EventTypes).Scan(&exists)
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
				Err: fmt.Errorf("Consistency violation: new events match query since position %d", latestPosition),
			},
			ExpectedPosition: latestPosition,
			ActualPosition:   latestPosition + 1, // Since we found a conflicting event, it must be at the next position
		}
	}
	return nil
}

// checkForMatchingEvents checks for matching events in conditional append
func checkForMatchingEvents(ctx context.Context, tx pgx.Tx, condition AppendCondition, queryTagsJSON []byte) error {
	// Fix: If EventTypes is empty, pass nil to the query so the type filter is ignored
	var eventTypesParam interface{}
	if len(condition.FailIfEventsMatch.EventTypes) == 0 {
		eventTypesParam = nil
	} else {
		eventTypesParam = condition.FailIfEventsMatch.EventTypes
	}

	var exists bool
	checkQuery := `
		SELECT EXISTS (
			SELECT 1
			FROM events
			WHERE tags @> $1::jsonb
			  AND ($2::text[] IS NULL OR type = ANY($2::text[]))
			  AND ($3::bigint IS NULL OR position > $3::bigint)
		)
	`
	err := tx.QueryRow(ctx, checkQuery, queryTagsJSON, eventTypesParam, condition.After).Scan(&exists)
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
				Err: fmt.Errorf("append condition failed: matching events found"),
			},
			ExpectedPosition: 0, // Not applicable in this case
			ActualPosition:   0, // Not applicable in this case
		}
	}
	return nil
}

// AppendEvents adds multiple events to the stream and returns the latest position.
func (es *eventStore) AppendEvents(ctx context.Context, events []InputEvent, query Query, latestPosition int64) (int64, error) {
	// Validate batch size
	if err := es.validateBatchSize(events, "AppendEvents"); err != nil {
		return 0, err
	}
	if len(events) == 0 {
		return latestPosition, nil
	}

	// Validate query tags
	if err := validateQueryTags(query); err != nil {
		return 0, err
	}

	// Validate all events before proceeding
	if err := validateEvents(events); err != nil {
		return 0, err
	}

	// Convert query tags to JSONB
	queryTagsJSON, err := convertTagsToJSON(query.Tags)
	if err != nil {
		log.Printf("Failed to marshal query tags: %v", err)
		return 0, fmt.Errorf("failed to marshal query tags: %w", err)
	}

	// Start transaction
	tx, err := es.pool.Begin(ctx)
	if err != nil {
		return 0, &EventStoreError{
			Op:  "AppendEvents",
			Err: fmt.Errorf("failed to begin transaction: %w", err),
		}
	}
	defer tx.Rollback(ctx) // Rollback if not committed

	// Check for conflicting events (optimistic locking)
	if err := checkForConflictingEvents(ctx, tx, query, queryTagsJSON, latestPosition); err != nil {
		return 0, err
	}

	// Prepare arrays for batch insert
	ids, types, tagsJSON, data, causationIDs, correlationIDs, err := prepareEventBatch(events)
	if err != nil {
		return 0, err
	}

	// Execute batch insert
	positions, err := executeBatchInsert(ctx, tx, events, ids, types, tagsJSON, data, causationIDs, correlationIDs)
	if err != nil {
		return 0, err
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return 0, &EventStoreError{
			Op:  "AppendEvents",
			Err: fmt.Errorf("failed to commit transaction: %w", err),
		}
	}

	// Log successful append
	log.Printf("Appended %d events, positions: %v", len(events), positions)

	// Return the latest position
	if len(positions) > 0 {
		return positions[len(positions)-1], nil
	}
	return latestPosition, nil // Fallback, though unlikely
}

// AppendEventsIfNotExists appends events only if no events match the append condition.
// It uses the condition to enforce consistency by failing if any events match the query
// after the specified position (if any).
func (es *eventStore) AppendEventsIf(ctx context.Context, events []InputEvent, condition AppendCondition) (int64, error) {
	// Validate batch size
	if err := es.validateBatchSize(events, "AppendEventsIf"); err != nil {
		return 0, err
	}
	if len(events) == 0 {
		return 0, &ValidationError{
			EventStoreError: EventStoreError{
				Op:  "AppendEventsIf",
				Err: fmt.Errorf("events slice cannot be empty"),
			},
			Field: "events",
			Value: "[]",
		}
	}

	// Validate query tags
	if err := validateQueryTags(condition.FailIfEventsMatch); err != nil {
		return 0, err
	}

	// Validate all events before proceeding
	if err := validateEvents(events); err != nil {
		return 0, err
	}

	// Start transaction
	tx, err := es.pool.Begin(ctx)
	if err != nil {
		return 0, &EventStoreError{
			Op:  "AppendEventsIf",
			Err: fmt.Errorf("failed to begin transaction: %w", err),
		}
	}
	defer tx.Rollback(ctx) // Rollback if not committed

	// Convert query tags to JSONB
	queryTagsJSON, err := convertTagsToJSON(condition.FailIfEventsMatch.Tags)
	if err != nil {
		log.Printf("Failed to marshal query tags: %v", err)
		return 0, fmt.Errorf("failed to marshal query tags: %w", err)
	}

	// Check for matching events
	if err := checkForMatchingEvents(ctx, tx, condition, queryTagsJSON); err != nil {
		return 0, err
	}

	// Prepare arrays for batch insert
	ids, types, tagsJSON, data, causationIDs, correlationIDs, err := prepareEventBatch(events)
	if err != nil {
		return 0, err
	}

	// Execute batch insert
	positions, err := executeBatchInsert(ctx, tx, events, ids, types, tagsJSON, data, causationIDs, correlationIDs)
	if err != nil {
		return 0, err
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return 0, &EventStoreError{
			Op:  "AppendEventsIf",
			Err: fmt.Errorf("failed to commit transaction: %w", err),
		}
	}

	// Log successful append
	log.Printf("Appended %d events, positions: %v", len(events), positions)

	// Return the latest position
	if len(positions) > 0 {
		return positions[len(positions)-1], nil
	}
	return 0, nil // This should never happen due to validation at the start
}
