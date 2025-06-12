package dcb

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// eventStore implements EventStore.
type eventStore struct {
	pool         *pgxpool.Pool // Database connection pool
	maxBatchSize int           // Maximum number of events in a single batch operation
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

// Read reads events matching a query, optionally starting from a specified sequence position
// This matches the DCB specification exactly
func (es *eventStore) Read(ctx context.Context, query Query, options *ReadOptions) (SequencedEvents, error) {
	if len(query.Items) == 0 {
		return SequencedEvents{}, &ValidationError{
			EventStoreError: EventStoreError{
				Op:  "read",
				Err: fmt.Errorf("query must contain at least one item"),
			},
			Field: "query",
			Value: "empty",
		}
	}

	// Build SQL query based on query items
	sqlQuery, args, err := es.buildReadQuerySQL(query, options)
	if err != nil {
		return SequencedEvents{}, err
	}

	// Execute the query
	rows, err := es.pool.Query(ctx, sqlQuery, args...)
	if err != nil {
		return SequencedEvents{}, &ResourceError{
			EventStoreError: EventStoreError{
				Op:  "read",
				Err: fmt.Errorf("failed to execute read query: %w", err),
			},
			Resource: "database",
		}
	}
	defer rows.Close()

	// Collect events
	var events []Event
	var lastPosition int64

	for rows.Next() {
		var row struct {
			ID            string
			Type          string
			Tags          []byte
			Data          []byte
			Position      int64
			CausationID   string
			CorrelationID string
		}

		if err := rows.Scan(&row.ID, &row.Type, &row.Tags, &row.Data, &row.Position, &row.CausationID, &row.CorrelationID); err != nil {
			return SequencedEvents{}, &ResourceError{
				EventStoreError: EventStoreError{
					Op:  "read",
					Err: fmt.Errorf("failed to scan event row: %w", err),
				},
				Resource: "database",
			}
		}

		// Convert row to Event
		event := Event{
			ID:            row.ID,
			Type:          row.Type,
			Data:          row.Data,
			Position:      row.Position,
			CausationID:   row.CausationID,
			CorrelationID: row.CorrelationID,
		}

		// Parse tags
		var tagMap map[string]string
		if err := json.Unmarshal(row.Tags, &tagMap); err != nil {
			return SequencedEvents{}, &EventStoreError{
				Op:  "read",
				Err: fmt.Errorf("failed to unmarshal tags for event %s: %w", row.ID, err),
			}
		}

		for k, v := range tagMap {
			event.Tags = append(event.Tags, Tag{Key: k, Value: v})
		}

		events = append(events, event)
		lastPosition = row.Position
	}

	if err := rows.Err(); err != nil {
		return SequencedEvents{}, &ResourceError{
			EventStoreError: EventStoreError{
				Op:  "read",
				Err: fmt.Errorf("error iterating over events: %w", err),
			},
			Resource: "database",
		}
	}

	return SequencedEvents{
		Events:   events,
		Position: lastPosition,
	}, nil
}

// Append atomically persists one or more events, optionally with an append condition
// This matches the DCB specification exactly
func (es *eventStore) Append(ctx context.Context, events []InputEvent, condition *AppendCondition) (int64, error) {
	if len(events) == 0 {
		return 0, &ValidationError{
			EventStoreError: EventStoreError{
				Op:  "append",
				Err: fmt.Errorf("events must not be empty"),
			},
			Field: "events",
			Value: "empty",
		}
	}

	if len(events) > es.maxBatchSize {
		return 0, &ValidationError{
			EventStoreError: EventStoreError{
				Op:  "append",
				Err: fmt.Errorf("batch size %d exceeds maximum of %d", len(events), es.maxBatchSize),
			},
			Field: "events",
			Value: fmt.Sprintf("count:%d", len(events)),
		}
	}

	// Validate events
	for i, event := range events {
		if event.Type == "" {
			return 0, &ValidationError{
				EventStoreError: EventStoreError{
					Op:  "append",
					Err: fmt.Errorf("event at index %d has empty type", i),
				},
				Field: "type",
				Value: "empty",
			}
		}

		// Validate tags
		tagKeys := make(map[string]bool)
		for _, tag := range event.Tags {
			if tag.Key == "" {
				return 0, &ValidationError{
					EventStoreError: EventStoreError{
						Op:  "append",
						Err: fmt.Errorf("event at index %d has tag with empty key", i),
					},
					Field: "tag.key",
					Value: "empty",
				}
			}
			if tag.Value == "" {
				return 0, &ValidationError{
					EventStoreError: EventStoreError{
						Op:  "append",
						Err: fmt.Errorf("event at index %d has tag with empty value", i),
					},
					Field: "tag.value",
					Value: "empty",
				}
			}
			if tagKeys[tag.Key] {
				return 0, &ValidationError{
					EventStoreError: EventStoreError{
						Op:  "append",
						Err: fmt.Errorf("event at index %d has duplicate tag key: %s", i, tag.Key),
					},
					Field: "tag.key",
					Value: tag.Key,
				}
			}
			tagKeys[tag.Key] = true
		}
	}

	// Check append condition if provided
	if condition != nil {
		if err := es.checkAppendCondition(ctx, *condition); err != nil {
			return 0, err
		}
	}

	// Insert events and return the position of the last event
	return es.insertEvents(ctx, events)
}

// buildReadQuerySQL builds the SQL query for reading events
func (es *eventStore) buildReadQuerySQL(query Query, options *ReadOptions) (string, []interface{}, error) {
	baseQuery := "SELECT id, type, tags, data, position, causation_id, correlation_id FROM events"
	var conditions []string
	var args []interface{}
	argIndex := 1

	// Add query conditions
	if len(query.Items) > 0 {
		var orConditions []string
		for _, item := range query.Items {
			var andConditions []string

			// Add event type conditions
			if len(item.EventTypes) > 0 {
				orConditions = append(orConditions, fmt.Sprintf("type = ANY($%d::text[])", argIndex))
				args = append(args, item.EventTypes)
				argIndex++
			}

			// Add tag conditions
			if len(item.Tags) > 0 {
				tagMap := make(map[string]string)
				for _, tag := range item.Tags {
					tagMap[tag.Key] = tag.Value
				}
				tagJSON, err := json.Marshal(tagMap)
				if err != nil {
					return "", nil, &EventStoreError{
						Op:  "buildReadQuerySQL",
						Err: fmt.Errorf("failed to marshal tags: %w", err),
					}
				}
				orConditions = append(orConditions, fmt.Sprintf("tags @> $%d::jsonb", argIndex))
				args = append(args, tagJSON)
				argIndex++
			}

			if len(andConditions) > 0 {
				orConditions = append(orConditions, "("+strings.Join(andConditions, " AND ")+")")
			}
		}

		if len(orConditions) > 0 {
			conditions = append(conditions, "("+strings.Join(orConditions, " OR ")+")")
		}
	}

	// Add position conditions
	if options != nil && options.FromPosition != nil {
		conditions = append(conditions, fmt.Sprintf("position > $%d", argIndex))
		args = append(args, *options.FromPosition)
		argIndex++
	}

	// Build final query
	if len(conditions) > 0 {
		baseQuery += " WHERE " + strings.Join(conditions, " AND ")
	}

	baseQuery += " ORDER BY position ASC"

	if options != nil && options.Limit != nil {
		baseQuery += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, *options.Limit)
	}

	return baseQuery, args, nil
}

// checkAppendCondition checks if the append condition is satisfied
func (es *eventStore) checkAppendCondition(ctx context.Context, condition AppendCondition) error {
	// Build query to check for conflicting events
	sqlQuery, args, err := es.buildReadQuerySQL(condition.FailIfEventsMatch, &ReadOptions{
		FromPosition: condition.After,
		Limit:        &[]int{1}[0], // Just need to know if any exist
	})
	if err != nil {
		return err
	}

	// Execute the query
	rows, err := es.pool.Query(ctx, sqlQuery, args...)
	if err != nil {
		return &ResourceError{
			EventStoreError: EventStoreError{
				Op:  "checkAppendCondition",
				Err: fmt.Errorf("failed to check append condition: %w", err),
			},
			Resource: "database",
		}
	}
	defer rows.Close()

	// If we get any rows, the condition is violated
	if rows.Next() {
		return &ConcurrencyError{
			EventStoreError: EventStoreError{
				Op:  "append",
				Err: fmt.Errorf("append condition violated: events matching query already exist"),
			},
		}
	}

	return nil
}

// insertEvents inserts the events into the database
func (es *eventStore) insertEvents(ctx context.Context, events []InputEvent) (int64, error) {
	// Start a transaction
	tx, err := es.pool.Begin(ctx)
	if err != nil {
		return 0, &ResourceError{
			EventStoreError: EventStoreError{
				Op:  "insertEvents",
				Err: fmt.Errorf("failed to begin transaction: %w", err),
			},
			Resource: "database",
		}
	}
	defer tx.Rollback(ctx)

	// Check if this is the first event in the store
	var firstEventID string
	var firstEventCorrelationID string
	err = tx.QueryRow(ctx, "SELECT id, correlation_id FROM events ORDER BY position ASC LIMIT 1").Scan(&firstEventID, &firstEventCorrelationID)
	if err != nil && err != pgx.ErrNoRows {
		return 0, &ResourceError{
			EventStoreError: EventStoreError{
				Op:  "insertEvents",
				Err: fmt.Errorf("failed to check for existing events: %w", err),
			},
			Resource: "database",
		}
	}

	// Insert each event
	for i, event := range events {
		// Convert tags to JSON
		tagMap := make(map[string]string)
		for _, tag := range event.Tags {
			tagMap[tag.Key] = tag.Value
		}
		tagJSON, err := json.Marshal(tagMap)
		if err != nil {
			return 0, &EventStoreError{
				Op:  "insertEvents",
				Err: fmt.Errorf("failed to marshal tags: %w", err),
			}
		}

		// Generate event ID using TypeID with tag-based prefix
		eventID := generateTagBasedTypeID(event.Tags)

		// Determine causation and correlation IDs
		var causationID, correlationID string
		if firstEventID == "" {
			// This is the first event in the store
			causationID = eventID
			correlationID = eventID
		} else {
			// Use the first event's ID as causation ID and first event's correlation ID as correlation ID
			causationID = firstEventID
			correlationID = firstEventCorrelationID
		}

		// Insert event
		_, err = tx.Exec(ctx, `
			INSERT INTO events (id, type, tags, data, position, causation_id, correlation_id)
			VALUES ($1, $2, $3, $4, nextval('events_position_seq'), $5, $6)
		`, eventID, event.Type, tagJSON, event.Data, causationID, correlationID)
		if err != nil {
			return 0, &ResourceError{
				EventStoreError: EventStoreError{
					Op:  "insertEvents",
					Err: fmt.Errorf("failed to insert event: %w", err),
				},
				Resource: "database",
			}
		}

		// Update first event info for subsequent events in this batch
		if i == 0 {
			firstEventID = eventID
			firstEventCorrelationID = correlationID
		}
	}

	// Get the position of the last event before committing
	var position int64
	err = tx.QueryRow(ctx, "SELECT COALESCE(MAX(position), 0) FROM events").Scan(&position)
	if err != nil {
		return 0, &EventStoreError{
			Op:  "insertEvents",
			Err: fmt.Errorf("failed to get current position: %w", err),
		}
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return 0, &ResourceError{
			EventStoreError: EventStoreError{
				Op:  "insertEvents",
				Err: fmt.Errorf("failed to commit transaction: %w", err),
			},
			Resource: "database",
		}
	}

	return position, nil
}
