package dcb

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// eventStore implements the EventStore interface
type eventStore struct {
	pool         *pgxpool.Pool // Database connection pool
	maxBatchSize int           // Maximum number of events in a single batch operation
}

// NewEventStore creates a new event store instance
func NewEventStore(ctx context.Context, pool *pgxpool.Pool) (EventStore, error) {
	if pool == nil {
		return nil, &ValidationError{
			EventStoreError: EventStoreError{
				Op:  "NewEventStore",
				Err: fmt.Errorf("pool cannot be nil"),
			},
			Field: "pool",
			Value: "nil",
		}
	}

	return &eventStore{
		pool:         pool,
		maxBatchSize: 1000, // Default batch size
	}, nil
}

// Read reads events matching a query, optionally starting from a specified sequence position
// This matches the DCB specification exactly
func (es *eventStore) Read(ctx context.Context, query Query, options *ReadOptions) (SequencedEvents, error) {
	if len(query.getItems()) == 0 {
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

	// Execute the query with timeout
	queryCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Execute the query
	rows, err := es.pool.Query(queryCtx, sqlQuery, args...)
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
func (es *eventStore) Append(ctx context.Context, events []InputEvent, condition AppendCondition) error {
	if len(events) == 0 {
		return &ValidationError{
			EventStoreError: EventStoreError{
				Op:  "append",
				Err: fmt.Errorf("events must not be empty"),
			},
			Field: "events",
			Value: "empty",
		}
	}

	if len(events) > es.maxBatchSize {
		return &ValidationError{
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
		if event.GetType() == "" {
			return &ValidationError{
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
		for _, tag := range event.GetTags() {
			if tag.Key == "" {
				return &ValidationError{
					EventStoreError: EventStoreError{
						Op:  "append",
						Err: fmt.Errorf("event at index %d has tag with empty key", i),
					},
					Field: "tag.key",
					Value: "empty",
				}
			}
			if tag.Value == "" {
				return &ValidationError{
					EventStoreError: EventStoreError{
						Op:  "append",
						Err: fmt.Errorf("event at index %d has tag with empty value", i),
					},
					Field: "tag.value",
					Value: "empty",
				}
			}
			if tagKeys[tag.Key] {
				return &ValidationError{
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
func (es *eventStore) appendEventsWithCondition(ctx context.Context, events []InputEvent, condition AppendCondition) error {
	// Prepare data for batch insert
	types := make([]string, len(events))
	tags := make([]string, len(events)) // now []string of array literals
	data := make([][]byte, len(events))

	for i, event := range events {
		types[i] = event.GetType()
		tags[i] = encodeTagsArrayLiteral(TagsToArray(event.GetTags()))
		data[i] = event.GetData()
	}

	// Convert condition to JSONB for PostgreSQL function
	var conditionJSON []byte
	var err error
	if condition != nil {
		conditionJSON, err = json.Marshal(condition)
		if err != nil {
			return &ResourceError{
				EventStoreError: EventStoreError{
					Op:  "appendEventsWithCondition",
					Err: fmt.Errorf("failed to marshal condition: %w", err),
				},
				Resource: "json",
			}
		}
	}

	// Execute PostgreSQL function for atomic append
	_, err = es.pool.Exec(ctx, `
		SELECT append_events_with_condition($1, $2, $3, $4)
	`, types, tags, data, conditionJSON)

	if err != nil {
		// Check if it's a condition violation error
		if strings.Contains(err.Error(), "append condition violated") {
			return &ConcurrencyError{
				EventStoreError: EventStoreError{
					Op:  "append",
					Err: fmt.Errorf("append condition violated: %w", err),
				},
			}
		}

		return &ResourceError{
			EventStoreError: EventStoreError{
				Op:  "appendEventsWithCondition",
				Err: fmt.Errorf("failed to append events: %w", err),
			},
			Resource: "database",
		}
	}

	return nil
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
