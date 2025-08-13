package dcb

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

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

	// Execute query using caller's context (caller controls timeout)
	rows, err := es.pool.Query(ctx, sqlQuery, args...)
	if err != nil {
		return nil, &EventStoreError{
			Op:  "query",
			Err: fmt.Errorf("failed to execute query: %w", err),
		}
	}
	defer rows.Close()

	// Scan results
	var events []Event
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
			return nil, &EventStoreError{
				Op:  "query",
				Err: fmt.Errorf("failed to scan event: %w", err),
			}
		}
		events = append(events, convertRowToEvent(row))
	}

	if err := rows.Err(); err != nil {
		return nil, &EventStoreError{
			Op:  "query",
			Err: fmt.Errorf("error iterating over rows: %w", err),
		}
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
