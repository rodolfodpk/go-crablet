package dcb

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

// =============================================================================
// APPEND-RELATED TYPES
// =============================================================================

// AppendCondition represents conditions for DCB concurrency control during append operations
// This is opaque to consumers - they can only construct it via helper functions
type AppendCondition interface {
	// isAppendCondition is a marker method to make this interface unexported
	isAppendCondition()
	// setAfterCursor sets the after cursor for proper (transaction_id, position) tracking
	setAfterCursor(after *Cursor)
	// getFailIfEventsMatch returns the internal query (used by event store)
	getFailIfEventsMatch() *Query
	// getAfterCursor returns the internal after cursor (used by event store)
	getAfterCursor() *Cursor
}

// InputEvent represents an event to be appended to the store
// This is now an opaque type: construct only via NewInputEvent
// and access fields only via methods
type InputEvent interface {
	isInputEvent()
	GetType() string
	GetTags() []Tag
	GetData() []byte
}

// appendCondition is the internal implementation
type appendCondition struct {
	FailIfEventsMatch *query  `json:"fail_if_events_match"`
	AfterCursor       *Cursor `json:"after_cursor"`
}

// isAppendCondition implements AppendCondition
func (ac *appendCondition) isAppendCondition() {}

// setAfterCursor sets the after cursor for proper (transaction_id, position) tracking
func (ac *appendCondition) setAfterCursor(after *Cursor) {
	ac.AfterCursor = after
}

// getFailIfEventsMatch returns the internal query (used by event store)
func (ac *appendCondition) getFailIfEventsMatch() *Query {
	if ac.FailIfEventsMatch == nil {
		return nil
	}
	var q Query = ac.FailIfEventsMatch
	return &q
}

// getAfterCursor returns the internal after cursor (used by event store)
func (ac *appendCondition) getAfterCursor() *Cursor {
	return ac.AfterCursor
}

// inputEvent is the internal implementation
type inputEvent struct {
	eventType string
	tags      []Tag
	data      []byte
}

func (e *inputEvent) isInputEvent()   {}
func (e *inputEvent) GetType() string { return e.eventType }
func (e *inputEvent) GetTags() []Tag  { return e.tags }
func (e *inputEvent) GetData() []byte { return e.data }

// Append appends events to the store with optional condition
// Append appends events to the store without any consistency/concurrency checks
// Use this only when there are no business rules or consistency requirements
func (es *eventStore) Append(ctx context.Context, events []InputEvent) error {
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

	// Start transaction using caller's context (caller controls timeout)
	tx, err := es.pool.BeginTx(ctx, pgx.TxOptions{
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

	// Use unconditional append (no consistency checks)
	err = es.appendInTx(ctx, tx, events, nil, nil)
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

// AppendIf appends events to the store with explicit DCB concurrency control
// This method makes it clear when consistency/concurrency checks are required
// Note: DCB uses its own concurrency control mechanism via AppendCondition
func (es *eventStore) AppendIf(ctx context.Context, events []InputEvent, condition AppendCondition) error {
	// Validate and prepare condition FIRST (fail early)
	conditionJSON, err := json.Marshal(condition)
	if err != nil {
		return &ResourceError{
			EventStoreError: EventStoreError{
				Op:  "appendIf",
				Err: fmt.Errorf("failed to marshal condition: %w", err),
			},
			Resource: "json",
		}
	}

	// Validate events
	if len(events) == 0 {
		return &ValidationError{
			EventStoreError: EventStoreError{
				Op:  "appendIf",
				Err: fmt.Errorf("events slice cannot be empty"),
			},
			Field: "events",
			Value: "empty",
		}
	}

	// Start transaction using caller's context (caller controls timeout)
	tx, err := es.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: toPgxIsoLevel(es.config.DefaultAppendIsolation),
	})
	if err != nil {
		return &ResourceError{
			EventStoreError: EventStoreError{
				Op:  "appendIf",
				Err: fmt.Errorf("failed to begin transaction: %w", err),
			},
			Resource: "database",
		}
	}
	defer tx.Rollback(ctx)

	// Use conditional append with DCB concurrency control
	err = es.appendInTx(ctx, tx, events, condition, conditionJSON)
	if err != nil {
		return err
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return &ResourceError{
			EventStoreError: EventStoreError{
				Op:  "appendIf",
				Err: fmt.Errorf("failed to commit transaction: %w", err),
			},
			Resource: "database",
		}
	}

	return nil
}

// extractConditionPrimitives extracts primitive values from AppendCondition for optimized PostgreSQL function
func extractConditionPrimitives(condition AppendCondition) ([]string, []string, *uint64, *int64) {
	var eventTypes []string
	var conditionTags []string
	var afterCursorTxID *uint64
	var afterCursorPosition *int64

	// Extract after cursor if present
	if afterCursor := condition.getAfterCursor(); afterCursor != nil {
		afterCursorTxID = &afterCursor.TransactionID
		afterCursorPosition = &afterCursor.Position
	}

	// Extract fail condition if present
	if failQuery := condition.getFailIfEventsMatch(); failQuery != nil {
		items := (*failQuery).GetItems()
		if len(items) > 0 {
			// Extract event types from first item
			eventTypes = items[0].GetEventTypes()

			// Extract tags from first item
			tags := items[0].GetTags()
			for _, tag := range tags {
				conditionTags = append(conditionTags, tag.GetKey()+":"+tag.GetValue())
			}
		}
	}

	return eventTypes, conditionTags, afterCursorTxID, afterCursorPosition
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

	// Validate batch size
	if err := es.validateBatchSize(events, "appendInTx"); err != nil {
		return err
	}

	// Validate each event
	for i, event := range events {
		if err := validateEvent(event, i); err != nil {
			return err
		}
	}

	// Prepare data for batch insert
	types := make([]string, len(events))
	tags := make([]string, len(events)) // array literal strings for storage
	data := make([][]byte, len(events))

	for i, event := range events {
		types[i] = event.GetType()
		data[i] = event.GetData()

		// Encode tags for storage
		var tagStrings []string
		for _, tag := range event.GetTags() {
			tagStrings = append(tagStrings, tag.GetKey()+":"+tag.GetValue())
		}
		tags[i] = encodeTagsArrayLiteral(tagStrings)

		// Debug logging removed for performance
	}

	// Execute append operation using appropriate PostgreSQL function
	var result []byte
	var err error
	if condition != nil {
		// Extract primitive values from condition for optimized function
		eventTypes, conditionTags, afterCursorTxID, afterCursorPosition := extractConditionPrimitives(condition)

		err = tx.QueryRow(ctx, `
			SELECT append_events_with_condition_optimized($1, $2, $3, $4, $5, $6, $7)
		`, types, tags, data, eventTypes, conditionTags, afterCursorTxID, afterCursorPosition).Scan(&result)
	} else {
		_, err = tx.Exec(ctx, `SELECT append_events_batch($1, $2, $3)`, types, tags, data)
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

	// Check result for conditional append operations
	if condition != nil && len(result) > 0 {
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
			// This is a concurrency violation
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
