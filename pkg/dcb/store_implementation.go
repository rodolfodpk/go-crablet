package dcb

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

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
			Type     string
			Tags     []string
			Data     []byte
			Position int64
		}

		if err := rows.Scan(&row.Type, &row.Tags, &row.Data, &row.Position); err != nil {
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
			Type:     row.Type,
			Tags:     ParseTagsArray(row.Tags),
			Data:     row.Data,
			Position: row.Position,
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
func (es *eventStore) Append(ctx context.Context, events []InputEvent, condition AppendCondition) (int64, error) {
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

	// Use PostgreSQL function for atomic append with condition checking
	return es.appendEventsWithCondition(ctx, events, condition)
}

// appendEventsWithCondition uses PostgreSQL functions for atomic append with condition checking
func (es *eventStore) appendEventsWithCondition(ctx context.Context, events []InputEvent, condition AppendCondition) (int64, error) {
	// Prepare data for batch insert
	types := make([]string, len(events))
	tags := make([]string, len(events)) // now []string of array literals
	data := make([][]byte, len(events))

	for i, event := range events {
		types[i] = event.Type
		tags[i] = encodeTagsArrayLiteral(TagsToArray(event.Tags))
		data[i] = event.Data
	}

	// Convert condition to JSONB for PostgreSQL function
	var conditionJSON []byte
	var err error
	if condition != nil {
		conditionJSON, err = json.Marshal(condition)
		if err != nil {
			return 0, &ResourceError{
				EventStoreError: EventStoreError{
					Op:  "appendEventsWithCondition",
					Err: fmt.Errorf("failed to marshal condition: %w", err),
				},
				Resource: "json",
			}
		}
	}

	// Execute PostgreSQL function for atomic append
	var lastPosition int64
	err = es.pool.QueryRow(ctx, `
		SELECT append_events_with_condition($1, $2, $3, $4)
	`, types, tags, data, conditionJSON).Scan(&lastPosition)

	if err != nil {
		// Check if it's a condition violation error
		if strings.Contains(err.Error(), "append condition violated") {
			return 0, &ConcurrencyError{
				EventStoreError: EventStoreError{
					Op:  "append",
					Err: fmt.Errorf("append condition violated: %w", err),
				},
			}
		}

		return 0, &ResourceError{
			EventStoreError: EventStoreError{
				Op:  "appendEventsWithCondition",
				Err: fmt.Errorf("failed to append events: %w", err),
			},
			Resource: "database",
		}
	}

	return lastPosition, nil
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
	sqlQuery.WriteString("SELECT type, tags, data, position FROM events")

	if len(conditions) > 0 {
		sqlQuery.WriteString(" WHERE ")
		sqlQuery.WriteString(strings.Join(conditions, " AND "))
	}

	sqlQuery.WriteString(" ORDER BY position ASC")

	// Add limit if specified
	if options != nil && options.Limit != nil {
		sqlQuery.WriteString(fmt.Sprintf(" LIMIT %d", *options.Limit))
	}

	return sqlQuery.String(), args, nil
}

// Add helper function to encode tags as Postgres array literal
func encodeTagsArrayLiteral(tags []string) string {
	if len(tags) == 0 {
		return "{}"
	}
	for i, t := range tags {
		tags[i] = `"` + strings.ReplaceAll(t, `"`, `\\"`) + `"`
	}
	return "{" + strings.Join(tags, ",") + "}"
}
