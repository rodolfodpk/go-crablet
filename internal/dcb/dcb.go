package dcb

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// eventStore implements EventStore.
type eventStore struct {
	pool         *pgxpool.Pool // Database connection pool
	mu           sync.RWMutex  // Changed to RWMutex for better concurrency
	closed       bool          // Indicates if the store has been closed
	maxBatchSize int           // Maximum number of events in a single batch operation
	cleanupOnce  sync.Once     // Ensures cleanup happens only once
}

// NewEventStore creates a new EventStore using the provided PostgreSQL connection pool.
// It uses a default maximum batch size of 1000 events.
func NewEventStore(ctx context.Context, pool *pgxpool.Pool) (EventStore, error) {
	// Test the connection with context timeout
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := pool.Ping(ctx); err != nil {
		return nil, &ResourceError{
			EventStoreError: EventStoreError{
				Op:  "NewEventStore",
				Err: fmt.Errorf("unable to connect to database: %w", err),
			},
			Resource: "database",
		}
	}

	return &eventStore{
		pool:         pool,
		maxBatchSize: 1000, // Default maximum batch size
	}, nil
}

// Close closes the event store's connection pool.
// It is safe to call Close multiple times.
func (es *eventStore) Close() {
	es.cleanupOnce.Do(func() {
		es.mu.Lock()
		defer es.mu.Unlock()

		if !es.closed {
			es.closed = true
			// Close the pool in a separate goroutine to avoid blocking
			go func() {
				// Use a timeout context for pool closure
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				// Gracefully close the pool with timeout
				done := make(chan struct{})
				go func() {
					es.pool.Close()
					close(done)
				}()

				select {
				case <-ctx.Done():
					// Context timed out, but pool.Close() will still run in background
					return
				case <-done:
					// Pool closed successfully
					return
				}
			}()
		}
	})
}

// isClosed checks if the event store is closed
func (es *eventStore) isClosed() bool {
	es.mu.RLock()
	defer es.mu.RUnlock()
	return es.closed
}

func (es *eventStore) AppendEventsIfNotExists(ctx context.Context, events []InputEvent, query Query, latestPosition int64, reducer StateReducer) (int64, error) {
	position, state, err := es.ReadStateUpTo(ctx, query, reducer, latestPosition)
	if err == nil && state != nil {
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

// rowEvent is a helper struct for scanning database rows.
type rowEvent struct {
	ID            pgtype.UUID
	Type          string
	Tags          []byte
	Data          []byte
	Position      int64
	CausationID   pgtype.UUID
	CorrelationID pgtype.UUID
}

// convertRowToEvent converts a database row to an Event
func convertRowToEvent(row rowEvent) Event {
	var e Event
	if !row.ID.Valid {
		panic(fmt.Sprintf("invalid UUID for id at position %d", row.Position))
	}
	e.ID = row.ID.String()
	e.Type = row.Type
	var tagMap map[string]string
	if err := json.Unmarshal(row.Tags, &tagMap); err != nil {
		panic(fmt.Sprintf("failed to unmarshal tags at position %d: %v", row.Position, err))
	}
	for k, v := range tagMap {
		e.Tags = append(e.Tags, Tag{Key: k, Value: v})
	}
	e.Data = row.Data
	e.Position = row.Position
	if row.CausationID.Valid {
		e.CausationID = row.CausationID.String()
	}
	if row.CorrelationID.Valid {
		e.CorrelationID = row.CorrelationID.String()
	}
	return e
}

// ReadState computes a state by streaming events matching the query, up to maxPosition.
func (es *eventStore) ReadState(ctx context.Context, query Query, stateReducer StateReducer) (int64, any, error) {
	return es.ReadStateUpTo(ctx, query, stateReducer, -1)
}

// ReadStateUpTo computes a state by streaming events matching the query, up to maxPosition.
func (es *eventStore) ReadStateUpTo(ctx context.Context, query Query, stateReducer StateReducer, maxPosition int64) (int64, any, error) {
	if stateReducer.ReducerFn == nil {
		return 0, stateReducer.InitialState, &ValidationError{
			EventStoreError: EventStoreError{
				Op:  "ReadStateUpTo",
				Err: fmt.Errorf("reducer function cannot be nil"),
			},
			Field: "reducer",
			Value: "nil",
		}
	}

	// Build JSONB query condition with proper error handling
	tagMap := make(map[string]string)
	for _, t := range query.Tags {
		tagMap[t.Key] = t.Value
	}
	queryTags, err := json.Marshal(tagMap)
	if err != nil {
		return 0, stateReducer.InitialState, &EventStoreError{
			Op:  "ReadStateUpTo",
			Err: fmt.Errorf("failed to marshal query tags %v: %w", tagMap, err),
		}
	}

	// Construct SQL query with proper error context
	sqlQuery := "SELECT id, type, tags, data, position, causation_id, correlation_id FROM events WHERE tags @> $1"
	args := []interface{}{queryTags}

	// Add event type filtering if specified
	if len(query.EventTypes) > 0 {
		sqlQuery += fmt.Sprintf(" AND type = ANY($%d)", len(args)+1)
		args = append(args, query.EventTypes)
	}

	if maxPosition > 0 {
		sqlQuery += fmt.Sprintf(" AND position <= $%d", len(args)+1)
		args = append(args, maxPosition)
	}

	// Query and stream rows with proper error handling
	rows, err := es.pool.Query(ctx, sqlQuery, args...)
	if err != nil {
		return 0, stateReducer.InitialState, &ResourceError{
			EventStoreError: EventStoreError{
				Op:  "ReadStateUpTo",
				Err: fmt.Errorf("failed to execute query for tags %v: %w", tagMap, err),
			},
			Resource: "database",
		}
	}
	defer rows.Close()

	// Initialize state
	state := stateReducer.InitialState
	position := int64(0)

	// Process events with proper error handling
	for rows.Next() {
		var row rowEvent
		if err := rows.Scan(&row.ID, &row.Type, &row.Tags, &row.Data, &row.Position, &row.CausationID, &row.CorrelationID); err != nil {
			return 0, stateReducer.InitialState, &ResourceError{
				EventStoreError: EventStoreError{
					Op:  "ReadStateUpTo",
					Err: fmt.Errorf("failed to scan event row at position %d: %w", position, err),
				},
				Resource: "database",
			}
		}

		// Convert row to Event with panic recovery
		var event Event
		func() {
			defer func() {
				if r := recover(); r != nil {
					err = &EventStoreError{
						Op:  "ReadStateUpTo",
						Err: fmt.Errorf("panic converting row to event at position %d: %v", row.Position, r),
					}
				}
			}()
			event = convertRowToEvent(row)
		}()
		if err != nil {
			return 0, stateReducer.InitialState, err
		}

		// Apply reducer with panic recovery
		func() {
			defer func() {
				if r := recover(); r != nil {
					err = &EventStoreError{
						Op:  "ReadStateUpTo",
						Err: fmt.Errorf("panic in reducer for event type %s at position %d: %v", event.Type, event.Position, r),
					}
				}
			}()
			state = stateReducer.ReducerFn(state, event)
		}()
		if err != nil {
			return 0, stateReducer.InitialState, err
		}

		position = row.Position
	}

	// Check for errors from iterating over rows
	if err := rows.Err(); err != nil {
		return 0, stateReducer.InitialState, &ResourceError{
			EventStoreError: EventStoreError{
				Op:  "ReadStateUpTo",
				Err: fmt.Errorf("error iterating over events: %w", err),
			},
			Resource: "database",
		}
	}

	return position, state, nil
}
