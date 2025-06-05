package dcb

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func (es *eventStore) AppendEventsIfNotExists(ctx context.Context, events []InputEvent, query Query, latestPosition int64, projector StateProjector) (int64, error) {
	position, state, err := es.ProjectStateUpTo(ctx, query, projector, latestPosition) // TODO this should be a boolean function
	if err != nil {
		return 0, fmt.Errorf("failed to project state: %w", err)
	}

	if state != nil {
		log.Printf("Events already exist for query: %v", query)
		return position, nil
	}

	return es.AppendEvents(ctx, events, query, latestPosition)
}

// validateQueryTags validates the query tags and returns a ValidationError if invalid
func validateQueryTags(query Query) error {
	// Empty Tags or EventTypes are allowed

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

	// Validate event types (optional)
	for i, eventType := range query.EventTypes {
		if eventType == "" {
			return &ValidationError{
				EventStoreError: EventStoreError{
					Op:  "validateQueryTags",
					Err: fmt.Errorf("empty event type at index %d", i),
				},
				Field: "eventType",
				Value: fmt.Sprintf("type[%d]", i),
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

// AppendEvents adds multiple events to the stream and returns the latest position.
func (es *eventStore) AppendEvents(ctx context.Context, events []InputEvent, query Query, latestPosition int64) (int64, error) {
	es.mu.Lock()
	defer es.mu.Unlock()

	if es.closed {
		return 0, &ResourceError{
			EventStoreError: EventStoreError{
				Op:  "AppendEvents",
				Err: fmt.Errorf("event store is closed"),
			},
			Resource: "eventStore",
		}
	}

	if len(events) > es.maxBatchSize {
		return 0, &ValidationError{
			EventStoreError: EventStoreError{
				Op:  "AppendEvents",
				Err: fmt.Errorf("batch size %d exceeds maximum %d", len(events), es.maxBatchSize),
			},
			Field: "batchSize",
			Value: fmt.Sprintf("%d", len(events)),
		}
	}
	if len(events) == 0 {
		return latestPosition, nil
	}

	// Validate query tags
	if err := validateQueryTags(query); err != nil {
		return 0, err
	}

	// Validate all events before proceeding
	for i, event := range events {
		if err := validateEvent(event, i); err != nil {
			return 0, err
		}
	}

	// Prepare arrays for PL/pgSQL
	ids := make([]pgtype.UUID, len(events))
	types := make([]string, len(events))
	tagsJSON := make([][]byte, len(events)) // Changed to [][]byte for JSONB
	data := make([][]byte, len(events))     // Changed to [][]byte for JSONB
	causationIDs := make([]pgtype.UUID, len(events))
	correlationIDs := make([]pgtype.UUID, len(events))

	for i, e := range events {
		// Generate UUID for event (UUIDv7)
		uuidVal, err := uuid.NewV7()
		if err != nil {
			log.Printf("Failed to generate UUID for event %d: %v", i, err)
			return 0, fmt.Errorf("failed to generate UUID for event %d: %w", i, err)
		}
		pgUUID := pgtype.UUID{}
		err = pgUUID.Scan(uuidVal.String())
		if err != nil {
			log.Printf("Failed to parse UUID for event %d: %v", i, err)
			return 0, fmt.Errorf("failed to parse UUID for event %d: %w", i, err)
		}
		ids[i] = pgUUID

		types[i] = e.Type
		data[i] = e.Data // Store as []byte for JSONB

		// Convert tags to JSONB
		tagMap := make(map[string]string)
		for _, t := range e.Tags {
			tagMap[t.Key] = t.Value
		}
		jsonBytes, err := json.Marshal(tagMap)
		if err != nil {
			log.Printf("Failed to marshal tags for event %d: %v", i, err)
			return 0, fmt.Errorf("failed to marshal tags for event %d: %w", i, err)
		}
		tagsJSON[i] = jsonBytes // Store as []byte for JSONB

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

	// Convert query tags to JSONB
	queryTagMap := make(map[string]string)
	for _, t := range query.Tags {
		queryTagMap[t.Key] = t.Value
	}
	queryTagsJSON, err := json.Marshal(queryTagMap)
	if err != nil {
		log.Printf("Failed to marshal query tags: %v", err)
		return 0, fmt.Errorf("failed to marshal query tags: %w", err)
	}

	// Append new events
	var pgPositions pgtype.Array[int64]
	err = es.pool.QueryRow(ctx, "SELECT append_events_batch($1, $2, $3::jsonb[], $4::jsonb[], $5::jsonb, $6, $7, $8, $9)",
		ids, types, tagsJSON, data, queryTagsJSON, latestPosition, causationIDs, correlationIDs, query.EventTypes,
	).Scan(&pgPositions)
	if err != nil {
		if err.Error() == "ERROR: Foreign key violation: invalid causation_id or correlation_id in batch (SQLSTATE P0001)" {
			return 0, &ValidationError{
				EventStoreError: EventStoreError{
					Op:  "AppendEvents",
					Err: fmt.Errorf("foreign key violation: one or more causation_id or correlation_id values are invalid"),
				},
				Field: "causation_id/correlation_id",
				Value: "batch",
			}
		}
		return 0, &EventStoreError{
			Op:  "AppendEvents",
			Err: fmt.Errorf("failed to append events: %w", err),
		}
	}

	// Extract positions from pgtype.Array[int64]
	positions := pgPositions.Elements
	// Log successful append
	log.Printf("Appended %d events, positions: %v", len(events), positions)

	// Return the latest position
	if len(positions) > 0 {
		return positions[len(positions)-1], nil
	}
	return latestPosition, nil // Fallback, though unlikely
}
