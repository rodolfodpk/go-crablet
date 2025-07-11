package dcb

import (
	"context"
	"fmt"
	"time"
)

// Query reads events matching the query with optional cursor
// cursor == nil: query from beginning of stream
// cursor != nil: query from specified cursor position
func (es *eventStore) Query(ctx context.Context, query Query, after *Cursor) ([]Event, error) {
	if len(query.getItems()) == 0 {
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

	// Execute query with timeout
	ctx, cancel := context.WithTimeout(ctx, time.Duration(es.config.QueryTimeout)*time.Millisecond)
	defer cancel()

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
			&row.CreatedAt,
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
	if len(query.getItems()) == 0 {
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

		// Execute query
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
				&row.CreatedAt,
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
