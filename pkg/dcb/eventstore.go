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

// Internal types for isolation levels and append options
type isolationLevel string

const (
	readUncommitted isolationLevel = "READ UNCOMMITTED"
	readCommitted   isolationLevel = "READ COMMITTED"
	repeatableRead  isolationLevel = "REPEATABLE READ"
	serializable    isolationLevel = "SERIALIZABLE"
)

type appendOptions struct {
	IsolationLevel *isolationLevel `json:"isolation_level,omitempty"`
}

// eventStore implements the EventStore interface using PostgreSQL
type eventStore struct {
	pool         *pgxpool.Pool
	maxBatchSize int
}

// newEventStore creates a new eventStore instance
func newEventStore(ctx context.Context, pool *pgxpool.Pool) (*eventStore, error) {
	// Test the connection
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &eventStore{
		pool:         pool,
		maxBatchSize: 1000, // Default maximum batch size
	}, nil
}

// ReadWithOptions reads events matching the query with additional options
func (es *eventStore) ReadWithOptions(ctx context.Context, query Query, options *ReadOptions) ([]Event, error) {
	if len(query.getItems()) == 0 {
		return nil, &ValidationError{
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
		return nil, err
	}

	// Build SQL query based on query items
	sqlQuery, args, err := es.buildReadQuerySQL(query, options)
	if err != nil {
		return nil, err
	}

	// Execute the query with timeout
	queryCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Execute the query
	rows, err := es.pool.Query(queryCtx, sqlQuery, args...)
	if err != nil {
		return nil, &ResourceError{
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
	for rows.Next() {
		var row struct {
			Type     string
			Tags     []string
			Data     []byte
			Position int64
		}

		if err := rows.Scan(&row.Type, &row.Tags, &row.Data, &row.Position); err != nil {
			return nil, &ResourceError{
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
	}

	if err := rows.Err(); err != nil {
		return nil, &ResourceError{
			EventStoreError: EventStoreError{
				Op:  "read",
				Err: fmt.Errorf("error iterating over events: %w", err),
			},
			Resource: "database",
		}
	}

	return events, nil
}

// Read reads events matching the query (no options)
func (es *eventStore) Read(ctx context.Context, query Query) ([]Event, error) {
	return es.ReadWithOptions(ctx, query, nil)
}

// Append appends events to the store (always succeeds if no validation errors)
func (es *eventStore) Append(ctx context.Context, events []InputEvent) error {
	return es.appendEventsBatch(ctx, events)
}

// AppendIf appends events to the store only if the condition is met
// Uses READ COMMITTED isolation level by default
func (es *eventStore) AppendIf(ctx context.Context, events []InputEvent, condition AppendCondition) error {
	return es.appendEventsWithCondition(ctx, events, condition, &appendOptions{})
}

// AppendIfSerializable appends events to the store only if the condition is met
// with SERIALIZABLE isolation level for maximum consistency
func (es *eventStore) AppendIfSerializable(ctx context.Context, events []InputEvent, condition AppendCondition) error {
	ser := serializable
	return es.appendEventsWithCondition(ctx, events, condition, &appendOptions{
		IsolationLevel: &ser,
	})
}

// AppendWithTxLevel atomically persists one or more events, optionally with an append condition and transaction options
// This matches the DCB specification exactly
func (es *eventStore) AppendWithTxLevel(ctx context.Context, events []InputEvent, condition AppendCondition, options *appendOptions) error {
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
		for j, tag := range event.GetTags() {
			if tag.GetKey() == "" {
				return &ValidationError{
					EventStoreError: EventStoreError{
						Op:  "append",
						Err: fmt.Errorf("empty tag key at index %d", j),
					},
					Field: "tag.key",
					Value: "empty",
				}
			}
			if tag.GetValue() == "" {
				return &ValidationError{
					EventStoreError: EventStoreError{
						Op:  "append",
						Err: fmt.Errorf("empty tag value for key %s", tag.GetKey()),
					},
					Field: "tag.value",
					Value: "empty",
				}
			}
			if tagKeys[tag.GetKey()] {
				return &ValidationError{
					EventStoreError: EventStoreError{
						Op:  "append",
						Err: fmt.Errorf("event at index %d has duplicate tag key: %s", i, tag.GetKey()),
					},
					Field: "tag.key",
					Value: tag.GetKey(),
				}
			}
			tagKeys[tag.GetKey()] = true
		}
	}

	// Use PostgreSQL function for atomic append with condition checking
	return es.appendEventsWithCondition(ctx, events, condition, options)
}

// appendEventsWithCondition uses PostgreSQL functions for atomic append with condition checking
func (es *eventStore) appendEventsWithCondition(ctx context.Context, events []InputEvent, condition AppendCondition, options *appendOptions) error {
	// Validate events
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

	// Validate individual events
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
		for j, tag := range event.GetTags() {
			if tag.GetKey() == "" {
				return &ValidationError{
					EventStoreError: EventStoreError{
						Op:  "append",
						Err: fmt.Errorf("empty tag key at index %d", j),
					},
					Field: "tag.key",
					Value: "empty",
				}
			}
			if tag.GetValue() == "" {
				return &ValidationError{
					EventStoreError: EventStoreError{
						Op:  "append",
						Err: fmt.Errorf("empty tag value for key %s", tag.GetKey()),
					},
					Field: "tag.value",
					Value: "empty",
				}
			}
			if tagKeys[tag.GetKey()] {
				return &ValidationError{
					EventStoreError: EventStoreError{
						Op:  "append",
						Err: fmt.Errorf("event at index %d has duplicate tag key: %s", i, tag.GetKey()),
					},
					Field: "tag.key",
					Value: tag.GetKey(),
				}
			}
			tagKeys[tag.GetKey()] = true
		}
	}

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

	// Determine isolation level
	var isoLevel pgx.TxIsoLevel
	if options != nil && options.IsolationLevel != nil {
		switch *options.IsolationLevel {
		case readUncommitted:
			isoLevel = pgx.ReadUncommitted
		case readCommitted:
			isoLevel = pgx.ReadCommitted
		case repeatableRead:
			isoLevel = pgx.RepeatableRead
		case serializable:
			isoLevel = pgx.Serializable
		default:
			isoLevel = pgx.ReadCommitted // fallback to default
		}
	} else {
		isoLevel = pgx.ReadCommitted // default
	}

	// Execute PostgreSQL function for atomic append
	for i, tagLiteral := range tags {
		if len(tagLiteral) < 2 || tagLiteral[0] != '{' || tagLiteral[len(tagLiteral)-1] != '}' {
			return fmt.Errorf("event %d: tag is not a valid Postgres array literal: %q", i, tagLiteral)
		}
	}

	// Start transaction with specified isolation level
	tx, err := es.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: isoLevel,
	})
	if err != nil {
		return &ResourceError{
			EventStoreError: EventStoreError{
				Op:  "appendEventsWithCondition",
				Err: fmt.Errorf("failed to begin transaction: %w", err),
			},
			Resource: "database",
		}
	}
	defer tx.Rollback(ctx)

	// Execute PostgreSQL function within transaction
	_, err = tx.Exec(ctx, `
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

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return &ResourceError{
			EventStoreError: EventStoreError{
				Op:  "appendEventsWithCondition",
				Err: fmt.Errorf("failed to commit transaction: %w", err),
			},
			Resource: "database",
		}
	}

	return nil
}

// appendEventsBatch uses the PostgreSQL append_events_batch function (no condition)
func (es *eventStore) appendEventsBatch(ctx context.Context, events []InputEvent) error {
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

	// Validate individual events
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
		for j, tag := range event.GetTags() {
			if tag.GetKey() == "" {
				return &ValidationError{
					EventStoreError: EventStoreError{
						Op:  "append",
						Err: fmt.Errorf("empty tag key at index %d", j),
					},
					Field: "tag.key",
					Value: "empty",
				}
			}
			if tag.GetValue() == "" {
				return &ValidationError{
					EventStoreError: EventStoreError{
						Op:  "append",
						Err: fmt.Errorf("empty tag value for key %s", tag.GetKey()),
					},
					Field: "tag.value",
					Value: "empty",
				}
			}
			if tagKeys[tag.GetKey()] {
				return &ValidationError{
					EventStoreError: EventStoreError{
						Op:  "append",
						Err: fmt.Errorf("event at index %d has duplicate tag key: %s", i, tag.GetKey()),
					},
					Field: "tag.key",
					Value: tag.GetKey(),
				}
			}
			tagKeys[tag.GetKey()] = true
		}
	}

	// Prepare data for batch insert
	types := make([]string, len(events))
	tags := make([]string, len(events)) // now []string of array literals
	data := make([][]byte, len(events))

	for i, event := range events {
		types[i] = event.GetType()
		tags[i] = encodeTagsArrayLiteral(TagsToArray(event.GetTags()))
		data[i] = event.GetData()
	}

	// Execute PostgreSQL function for batch insert
	tx, err := es.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.ReadCommitted, // Always use default for batch
	})
	if err != nil {
		return &ResourceError{
			EventStoreError: EventStoreError{
				Op:  "appendEventsBatch",
				Err: fmt.Errorf("failed to begin transaction: %w", err),
			},
			Resource: "database",
		}
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `SELECT append_events_batch($1, $2, $3)`, types, tags, data)
	if err != nil {
		return &ResourceError{
			EventStoreError: EventStoreError{
				Op:  "appendEventsBatch",
				Err: fmt.Errorf("failed to append events: %w", err),
			},
			Resource: "database",
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return &ResourceError{
			EventStoreError: EventStoreError{
				Op:  "appendEventsBatch",
				Err: fmt.Errorf("failed to commit transaction: %w", err),
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
