package dcb

import (
	"context"
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

	// Validate query items
	if err := validateQueryTags(query); err != nil {
		return SequencedEvents{}, err
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
			Tags          []string
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
			Tags:          ParseTagsArray(row.Tags),
			Data:          row.Data,
			Position:      row.Position,
			CausationID:   row.CausationID,
			CorrelationID: row.CorrelationID,
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
		// Check if After position exists
		if condition.After != nil {
			var exists bool
			err := es.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM events WHERE position = $1)", *condition.After).Scan(&exists)
			if err != nil {
				return 0, &ResourceError{
					EventStoreError: EventStoreError{
						Op:  "append",
						Err: fmt.Errorf("failed to check After position existence: %w", err),
					},
					Resource: "database",
				}
			}
			if !exists {
				return 0, &ConcurrencyError{
					EventStoreError: EventStoreError{
						Op:  "append",
						Err: fmt.Errorf("optimistic concurrency conflict: After position %d does not exist", *condition.After),
					},
					ExpectedPosition: *condition.After,
					ActualPosition:   0,
				}
			}
		}
		if err := es.checkAppendCondition(ctx, *condition); err != nil {
			return 0, err
		}
	}

	// Insert events and return the position of the last event
	return es.insertEvents(ctx, events)
}

// buildReadQuerySQL builds the SQL query for reading events
func (es *eventStore) buildReadQuerySQL(query Query, options *ReadOptions) (string, []interface{}, error) {
	// Pre-allocate slices with reasonable capacity
	conditions := make([]string, 0, 4) // Usually 1-4 conditions
	args := make([]interface{}, 0, 8)  // Usually 2-8 args
	argIndex := 1

	// Add query conditions
	if len(query.Items) > 0 {
		orConditions := make([]string, 0, len(query.Items))

		for _, item := range query.Items {
			andConditions := make([]string, 0, 2) // Usually 1-2 conditions per item

			// Add event type conditions
			if len(item.EventTypes) > 0 {
				andConditions = append(andConditions, fmt.Sprintf("type = ANY($%d::text[])", argIndex))
				args = append(args, item.EventTypes)
				argIndex++
			}

			// Add tag conditions - use contains operator for DCB semantics
			if len(item.Tags) > 0 {
				tagsArray := TagsToArray(item.Tags)
				andConditions = append(andConditions, fmt.Sprintf("tags @> $%d::text[]", argIndex))
				args = append(args, tagsArray)
				argIndex++
			}

			// Combine AND conditions for this item
			if len(andConditions) > 0 {
				orConditions = append(orConditions, "("+strings.Join(andConditions, " AND ")+")")
			}
		}

		// Combine OR conditions for all items
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

	// Build final query efficiently
	var sqlQuery strings.Builder
	sqlQuery.WriteString("SELECT id, type, tags, data, position, causation_id, correlation_id FROM events")

	if len(conditions) > 0 {
		sqlQuery.WriteString(" WHERE ")
		sqlQuery.WriteString(strings.Join(conditions, " AND "))
	}

	sqlQuery.WriteString(" ORDER BY position ASC")

	if options != nil && options.Limit != nil {
		sqlQuery.WriteString(fmt.Sprintf(" LIMIT $%d", argIndex))
		args = append(args, *options.Limit)
	}

	return sqlQuery.String(), args, nil
}

// checkAppendCondition checks if the append condition is satisfied
func (es *eventStore) checkAppendCondition(ctx context.Context, condition AppendCondition) error {
	// Check FailIfEventsMatch condition if provided
	if condition.FailIfEventsMatch != nil {
		// Build query to check for conflicting events
		sqlQuery, args, err := es.buildReadQuerySQL(*condition.FailIfEventsMatch, &ReadOptions{
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
	}

	// Check After field for optimistic locking if provided
	if condition.After != nil {
		// For optimistic locking with After field, we need to check if any events exist
		// after the specified position that match the same query scope as the projection.
		// Since we don't have the original query here, we use the FailIfEventsMatch query
		// as a proxy for the query scope, or check for any events if no query is provided.
		if condition.FailIfEventsMatch != nil {
			// Use the same query scope as the FailIfEventsMatch condition
			sqlQuery, args, err := es.buildReadQuerySQL(*condition.FailIfEventsMatch, &ReadOptions{
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
						Err: fmt.Errorf("failed to check optimistic locking condition: %w", err),
					},
					Resource: "database",
				}
			}
			defer rows.Close()

			// If we get any rows, there are events after the specified position
			if rows.Next() {
				return &ConcurrencyError{
					EventStoreError: EventStoreError{
						Op:  "append",
						Err: fmt.Errorf("optimistic concurrency conflict: events exist after position %d", *condition.After),
					},
					ExpectedPosition: *condition.After,
					ActualPosition:   *condition.After + 1, // Since we found events after, they must be at next position
				}
			}
		} else {
			// If no query scope is provided, check for any events after the position (global optimistic locking)
			var exists bool
			checkQuery := `SELECT EXISTS(SELECT 1 FROM events WHERE position > $1)`
			err := es.pool.QueryRow(ctx, checkQuery, *condition.After).Scan(&exists)
			if err != nil {
				return &ResourceError{
					EventStoreError: EventStoreError{
						Op:  "checkAppendCondition",
						Err: fmt.Errorf("failed to check optimistic locking condition: %w", err),
					},
					Resource: "database",
				}
			}

			if exists {
				return &ConcurrencyError{
					EventStoreError: EventStoreError{
						Op:  "append",
						Err: fmt.Errorf("optimistic concurrency conflict: events exist after position %d", *condition.After),
					},
					ExpectedPosition: *condition.After,
					ActualPosition:   *condition.After + 1, // Since we found events after, they must be at next position
				}
			}
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
		// Convert tags to TEXT[] array
		tagsArray := TagsToArray(event.Tags)

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
		`, eventID, event.Type, tagsArray, event.Data, causationID, correlationID)
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
