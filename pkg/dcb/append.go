package dcb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Append appends events to the store (always succeeds if no validation errors)
func (es *eventStore) Append(ctx context.Context, events []InputEvent) error {
	return es.appendEventsBatch(ctx, events)
}

// AppendIf appends events to the store only if the condition is met
// Uses REPEATABLE READ isolation level for better consistency
func (es *eventStore) AppendIf(ctx context.Context, events []InputEvent, condition AppendCondition) error {
	return es.appendEventsWithCondition(ctx, events, condition, pgx.RepeatableRead)
}

// AppendIfIsolated appends events to the store only if the condition is met
// Uses SERIALIZABLE isolation level for strongest consistency
func (es *eventStore) AppendIfIsolated(ctx context.Context, events []InputEvent, condition AppendCondition) error {
	return es.appendEventsWithCondition(ctx, events, condition, pgx.Serializable)
}

// appendEventsWithCondition uses PostgreSQL functions for atomic append with condition checking
func (es *eventStore) appendEventsWithCondition(ctx context.Context, events []InputEvent, condition AppendCondition, isolationLevel pgx.TxIsoLevel) error {
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

	// Start transaction with specified isolation level
	tx, err := es.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: isolationLevel,
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

// isConcurrencyError checks if the error is a concurrency error raised by SQL
func isConcurrencyError(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		// Check for our custom error code DCB01 which indicates concurrency violations
		return pgErr.Code == "DCB01"
	}

	// Fallback: check for the error message pattern (for backward compatibility)
	// This can be removed once we're confident all deployments use the new error codes
	return err != nil && strings.Contains(err.Error(), "append condition violated:")
}
