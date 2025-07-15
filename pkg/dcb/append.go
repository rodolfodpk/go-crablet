package dcb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// withTimeout creates a new context with timeout, respecting caller's timeout if set
// If caller provides context with deadline: use caller's timeout
// If caller provides context without deadline: use default from config
func (es *eventStore) withTimeout(ctx context.Context, defaultTimeoutMs int) (context.Context, context.CancelFunc) {
	if deadline, ok := ctx.Deadline(); ok {
		// Caller already set a timeout, use it
		// Use context.Background() as parent to avoid inheriting cancellation from original context
		return context.WithDeadline(context.Background(), deadline)
	}
	// No caller timeout, use default
	// Use context.Background() as parent to avoid inheriting cancellation from original context
	return context.WithTimeout(context.Background(), time.Duration(defaultTimeoutMs)*time.Millisecond)
}

// Append appends events to the store with optional condition
// condition == nil: unconditional append
// condition != nil: conditional append (optimistic locking)
func (es *eventStore) Append(ctx context.Context, events []InputEvent, condition *AppendCondition) error {
	// Validate and prepare condition FIRST (fail early)
	var conditionJSON []byte
	if condition != nil {
		var err error
		conditionJSON, err = json.Marshal(condition)
		if err != nil {
			return &ResourceError{
				EventStoreError: EventStoreError{
					Op:  "append",
					Err: fmt.Errorf("failed to marshal condition: %w", err),
				},
				Resource: "json",
			}
		}
	}

	// Validate events
	if len(events) == 0 {
		return &ValidationError{
			EventStoreError: EventStoreError{
				Op:  "append",
				Err: fmt.Errorf("events slice cannot be empty"),
			},
			Field: "events",
			Value: "empty",
		}
	}

	// Start transaction with hybrid timeout (respects caller timeout if set, otherwise uses default)
	appendCtx, cancel := es.withTimeout(ctx, es.config.AppendTimeout)
	defer cancel()

	tx, err := es.pool.BeginTx(appendCtx, pgx.TxOptions{
		IsoLevel: toPgxIsoLevel(es.config.DefaultAppendIsolation),
	})
	if err != nil {
		return &ResourceError{
			EventStoreError: EventStoreError{
				Op:  "append",
				Err: fmt.Errorf("failed to begin transaction: %w", err),
			},
			Resource: "database",
		}
	}
	defer tx.Rollback(ctx)

	// Use conditional or unconditional append based on condition parameter
	if condition != nil {
		err = es.appendInTx(ctx, tx, events, *condition, conditionJSON)
	} else {
		err = es.appendInTx(ctx, tx, events, nil, nil)
	}
	if err != nil {
		return err
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return &ResourceError{
			EventStoreError: EventStoreError{
				Op:  "append",
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

	// Tags are already in "key:value" format from TagsToArray
	// We need to properly escape and quote each tag
	quotedTags := make([]string, len(tags))
	for i, tag := range tags {
		// Escape any double quotes in the tag and wrap in quotes
		quotedTags[i] = `"` + strings.ReplaceAll(tag, `"`, `\\"`) + `"`
	}
	return "{" + strings.Join(quotedTags, ",") + "}"
}

// Helper to convert our IsolationLevel to pgx.TxIsoLevel
func toPgxIsoLevel(level IsolationLevel) pgx.TxIsoLevel {
	// Map our enum to pgx
	switch level {
	case IsolationLevelReadCommitted:
		return pgx.ReadCommitted
	case IsolationLevelRepeatableRead:
		return pgx.RepeatableRead
	case IsolationLevelSerializable:
		return pgx.Serializable
	default:
		return pgx.ReadCommitted
	}
}

// appendInTx appends events within an existing transaction
// This is the internal method that does the actual work without managing transactions
func (es *eventStore) appendInTx(ctx context.Context, tx pgx.Tx, events []InputEvent, condition AppendCondition, conditionJSON []byte) error {
	// Validate events
	if len(events) == 0 {
		return &ValidationError{
			EventStoreError: EventStoreError{
				Op:  "appendInTx",
				Err: fmt.Errorf("events slice cannot be empty"),
			},
			Field: "events",
			Value: "empty",
		}
	}

	if len(events) > es.config.MaxBatchSize {
		return &ValidationError{
			EventStoreError: EventStoreError{
				Op:  "appendInTx",
				Err: fmt.Errorf("batch size %d exceeds maximum of %d", len(events), es.config.MaxBatchSize),
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
					Op:  "appendInTx",
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
						Op:  "appendInTx",
						Err: fmt.Errorf("empty tag key at index %d", j),
					},
					Field: "tag.key",
					Value: "empty",
				}
			}
			if tag.GetValue() == "" {
				return &ValidationError{
					EventStoreError: EventStoreError{
						Op:  "appendInTx",
						Err: fmt.Errorf("empty tag value for key %s", tag.GetKey()),
					},
					Field: "tag.value",
					Value: "empty",
				}
			}
			if tagKeys[tag.GetKey()] {
				return &ValidationError{
					EventStoreError: EventStoreError{
						Op:  "appendInTx",
						Err: fmt.Errorf("event at index %d has duplicate tag key: %s", i, tag.GetKey()),
					},
					Field: "tag.key",
					Value: tag.GetKey(),
				}
			}
			tagKeys[tag.GetKey()] = true
		}
	}

	// Check if any events have lock: tags to determine which SQL function to use
	hasLockTags := false
	for _, event := range events {
		for _, tag := range event.GetTags() {
			if strings.HasPrefix(tag.GetKey(), "lock:") {
				hasLockTags = true
				break
			}
		}
		if hasLockTags {
			break
		}
	}

	// Prepare data for batch insert
	types := make([]string, len(events))
	tags := make([]string, len(events))     // array literal strings for storage
	lockTags := make([]string, len(events)) // array literal strings for advisory locks
	data := make([][]byte, len(events))

	for i, event := range events {
		types[i] = event.GetType()
		data[i] = event.GetData()

		// Separate lock tags from regular tags
		var regularTags []string
		var lockKeys []string

		for _, tag := range event.GetTags() {
			if strings.HasPrefix(tag.GetKey(), "lock:") {
				// Extract the lock key (remove "lock:" prefix)
				lockKey := strings.TrimPrefix(tag.GetKey(), "lock:")
				lockKeys = append(lockKeys, lockKey)
			} else {
				// Regular tag for storage
				regularTags = append(regularTags, tag.GetKey()+":"+tag.GetValue())
			}
		}

		// Encode tags for storage (without lock: prefix)
		tags[i] = encodeTagsArrayLiteral(regularTags)

		// Encode lock keys for advisory locks (without lock: prefix)
		lockTags[i] = encodeTagsArrayLiteral(lockKeys)

		// Debug logging removed for performance
	}

	// Execute append operation using appropriate PostgreSQL function
	// Use advisory locks if any lock: tags are present, otherwise use regular functions
	var result []byte
	var err error
	if hasLockTags {
		// Use advisory lock function with separate lock tags parameter
		if condition != nil {
			err = tx.QueryRow(ctx, `
				SELECT append_events_with_advisory_locks($1, $2, $3, $4, $5, $6)
			`, types, tags, data, lockTags, conditionJSON, es.config.LockTimeout).Scan(&result)
		} else {
			err = tx.QueryRow(ctx, `
				SELECT append_events_with_advisory_locks($1, $2, $3, $4, $5, $6)
			`, types, tags, data, lockTags, nil, es.config.LockTimeout).Scan(&result)
		}
	} else {
		// Use regular functions (no advisory locks)
		if condition != nil {
			err = tx.QueryRow(ctx, `
				SELECT append_events_with_condition($1, $2, $3, $4)
			`, types, tags, data, conditionJSON).Scan(&result)
		} else {
			_, err = tx.Exec(ctx, `SELECT append_events_batch($1, $2, $3)`, types, tags, data)
		}
	}

	if err != nil {
		return &ResourceError{
			EventStoreError: EventStoreError{
				Op:  "appendInTx",
				Err: fmt.Errorf("failed to append events: %w", err),
			},
			Resource: "database",
		}
	}

	// Check result for conditional append or advisory lock operations
	if (condition != nil || hasLockTags) && len(result) > 0 {
		var resultMap map[string]interface{}
		if err := json.Unmarshal(result, &resultMap); err != nil {
			return &ResourceError{
				EventStoreError: EventStoreError{
					Op:  "appendInTx",
					Err: fmt.Errorf("failed to parse append result: %w", err),
				},
				Resource: "json",
			}
		}

		// Check if the operation was successful
		if success, ok := resultMap["success"].(bool); !ok || !success {
			// This is a concurrency violation or lock acquisition failure
			return &ConcurrencyError{
				EventStoreError: EventStoreError{
					Op:  "appendInTx",
					Err: fmt.Errorf("append condition violated: %v", resultMap["message"]),
				},
			}
		}
	}

	return nil
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
