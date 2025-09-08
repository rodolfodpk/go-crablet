package dcb

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
)

// =============================================================================
// QUERY-RELATED TYPES
// =============================================================================

// Query represents a composite query with multiple conditions combined with OR logic
// This is opaque to consumers - they can only construct it via helper functions
// Now exposes GetItems for public access
type Query interface {
	// isQuery is a marker method to make this interface unexported
	isQuery()
	// GetItems returns the internal query items (used by event store)
	GetItems() []QueryItem
}

// QueryItem represents a single atomic query condition
// This is opaque to consumers - they can only construct it via helper functions
// Now exposes GetEventTypes and GetTags for public access
type QueryItem interface {
	// isQueryItem is a marker method to make this interface unexported
	isQueryItem()
	// GetEventTypes returns the internal event types (used by event store)
	GetEventTypes() []string
	// GetTags returns the internal tags (used by event store)
	GetTags() []Tag
}

// query is the internal implementation
type query struct {
	Items []QueryItem `json:"items"`
}

// isQuery implements Query
func (q *query) isQuery() {}

// GetItems returns the internal query items (used by event store)
func (q *query) GetItems() []QueryItem {
	return q.Items
}

// queryItem is the internal implementation
type queryItem struct {
	EventTypes []string `json:"event_types"`
	Tags       []Tag    `json:"tags"`
}

// isQueryItem implements QueryItem
func (qi *queryItem) isQueryItem() {}

// GetEventTypes returns the internal event types (used by event store)
func (qi *queryItem) GetEventTypes() []string {
	return qi.EventTypes
}

// GetTags returns the internal tags (used by event store)
func (qi *queryItem) GetTags() []Tag {
	return qi.Tags
}

// Query reads events matching the query with optional cursor
// cursor == nil: query from beginning of stream
// cursor != nil: query from specified cursor position
func (es *eventStore) Query(ctx context.Context, query Query, after *Cursor) ([]Event, error) {
	if len(query.GetItems()) == 0 {
		return nil, &ValidationError{
			EventStoreError: EventStoreError{
				Op:  "query",
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

	// Build SQL query based on query items with cursor
	sqlQuery, args, err := es.buildReadQuerySQL(query, after, nil)
	if err != nil {
		return nil, &EventStoreError{
			Op:  "query",
			Err: fmt.Errorf("failed to build SQL query: %w", err),
		}
	}

	// Execute query within a transaction for consistency
	var events []Event
	err = es.executeReadInTx(ctx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, sqlQuery, args...)
		if err != nil {
			return &EventStoreError{
				Op:  "query",
				Err: fmt.Errorf("failed to execute query: %w", err),
			}
		}
		defer rows.Close()

		// Scan results
		for rows.Next() {
			var row rowEvent
			err := rows.Scan(
				&row.Type,
				&row.Tags,
				&row.Data,
				&row.TransactionID,
				&row.Position,
				&row.OccurredAt,
			)
			if err != nil {
				return &EventStoreError{
					Op:  "query",
					Err: fmt.Errorf("failed to scan event: %w", err),
				}
			}
			events = append(events, convertRowToEvent(row))
		}

		if err := rows.Err(); err != nil {
			return &EventStoreError{
				Op:  "query",
				Err: fmt.Errorf("error iterating over rows: %w", err),
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return events, nil
}

// QueryStream creates a channel-based stream of events matching a query with optional cursor
// cursor == nil: stream from beginning of stream
// cursor != nil: stream from specified cursor position
// This is optimized for large datasets and provides backpressure through channels
// for efficient memory usage and Go-idiomatic streaming
func (es *eventStore) QueryStream(ctx context.Context, query Query, after *Cursor) (<-chan Event, error) {
	if len(query.GetItems()) == 0 {
		return nil, &ValidationError{
			EventStoreError: EventStoreError{
				Op:  "query_stream",
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

	// Create event channel
	eventChan := make(chan Event, es.config.StreamBuffer)

	// Start goroutine to stream events
	go func() {
		defer close(eventChan)

		// Build SQL query with cursor
		sqlQuery, args, err := es.buildReadQuerySQL(query, after, nil)
		if err != nil {
			// Send error through channel (caller should handle)
			return
		}

		// Execute query using caller's context (caller controls timeout)
		rows, err := es.pool.Query(ctx, sqlQuery, args...)
		if err != nil {
			return
		}
		defer rows.Close()

		// Stream events
		for rows.Next() {
			var row rowEvent
			err := rows.Scan(
				&row.Type,
				&row.Tags,
				&row.Data,
				&row.TransactionID,
				&row.Position,
				&row.OccurredAt,
			)
			if err != nil {
				return
			}

			// Convert row to event and send through channel
			event := convertRowToEvent(row)
			select {
			case eventChan <- event:
			case <-ctx.Done():
				return
			}
		}

		if err := rows.Err(); err != nil {
			return
		}
	}()

	return eventChan, nil
}

// TagsToArray converts a slice of Tags to a PostgreSQL TEXT[] array
func TagsToArray(tags []Tag) []string {
	if len(tags) == 0 {
		return []string{}
	}

	result := make([]string, len(tags))
	for i, tag := range tags {
		result[i] = tag.GetKey() + ":" + tag.GetValue()
	}

	// Sort for consistent ordering
	sort.Strings(result)
	return result
}

// ParseTagsArray converts a PostgreSQL TEXT[] array back to a slice of Tags
func ParseTagsArray(arr []string) []Tag {
	if len(arr) == 0 {
		return []Tag{}
	}

	tags := make([]Tag, 0, len(arr))
	for _, item := range arr {
		if item == "" {
			continue
		}

		// Split on first ":" only to handle values with colons
		parts := strings.SplitN(item, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := parts[1] // Keep original value (including colons)
			if key != "" {
				tags = append(tags, NewTag(key, value))
			}
		}
	}
	return tags
}

// TagsToString returns a slice of string representations of tags
func TagsToString(tags []Tag) []string {
	result := make([]string, len(tags))
	for i, tag := range tags {
		result[i] = tag.GetKey() + ":" + tag.GetValue()
	}
	return result
}
