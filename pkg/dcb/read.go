package dcb

import (
	"context"
	"fmt"
	"log"
	"time"
)

// Read reads events matching the query with optional cursor
// after == nil: read from beginning of stream
// after != nil: read from specified cursor position (EXCLUSIVE - events after cursor, not including cursor)
func (es *eventStore) Read(ctx context.Context, query Query, after *Cursor) ([]Event, error) {
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

	// Build SQL query based on query items with cursor
	sqlQuery, args, err := es.buildReadQuerySQL(query, after, nil)
	if err != nil {
		return nil, err
	}

	// Execute query with timeout
	ctx, cancel := context.WithTimeout(ctx, time.Duration(es.config.ReadTimeout)*time.Millisecond)
	defer cancel()

	rows, err := es.pool.Query(ctx, sqlQuery, args...)
	if err != nil {
		return nil, &EventStoreError{
			Op:  "read",
			Err: fmt.Errorf("failed to execute read query: %w", err),
		}
	}
	defer rows.Close()

	// Scan results
	var events []Event
	for rows.Next() {
		var row rowEvent
		err := rows.Scan(&row.Type, &row.Tags, &row.Data, &row.TransactionID, &row.Position, &row.CreatedAt)
		if err != nil {
			return nil, &EventStoreError{
				Op:  "read",
				Err: fmt.Errorf("failed to scan event row: %w", err),
			}
		}

		// Convert rowEvent to Event
		event := convertRowToEvent(row)
		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, &EventStoreError{
			Op:  "read",
			Err: fmt.Errorf("error iterating over rows: %w", err),
		}
	}

	return events, nil
}

// ReadStream creates a channel-based stream of events matching a query with optional cursor
// after == nil: stream from beginning of stream
// after != nil: stream from specified cursor position (EXCLUSIVE - events after cursor, not including cursor)
// This is optimized for large datasets and provides backpressure through channels
// for efficient memory usage and Go-idiomatic streaming
func (es *eventStore) ReadStream(ctx context.Context, query Query, after *Cursor) (<-chan Event, error) {
	// Validate query
	if query == nil {
		return nil, &ValidationError{
			EventStoreError: EventStoreError{
				Op:  "ReadStream",
				Err: fmt.Errorf("query cannot be nil"),
			},
			Field: "query",
			Value: "nil",
		}
	}

	// Validate query items
	if len(query.getItems()) == 0 {
		return nil, &ValidationError{
			EventStoreError: EventStoreError{
				Op:  "ReadStream",
				Err: fmt.Errorf("query must have at least one item"),
			},
			Field: "query.items",
			Value: "empty",
		}
	}

	// Create channel with buffer size from config
	eventChan := make(chan Event, es.config.StreamBuffer)

	// Start goroutine to handle the streaming
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("ReadStream panic recovered: %v", r)
			}
			close(eventChan)
		}()

		// Build SQL query with cursor
		sqlQuery, args, err := es.buildReadQuerySQL(query, after, nil)
		if err != nil {
			log.Printf("Error building SQL query in ReadStream: %v", err)
			return
		}

		// Execute query using caller's context (caller controls timeout)
		rows, err := es.pool.Query(ctx, sqlQuery, args...)
		if err != nil {
			log.Printf("Error executing query in ReadStream: %v", err)
			return
		}
		defer rows.Close()

		// Stream events through channel
		for rows.Next() {
			var row rowEvent
			err := rows.Scan(&row.Type, &row.Tags, &row.Data, &row.TransactionID, &row.Position, &row.CreatedAt)
			if err != nil {
				log.Printf("Error scanning row in ReadStream: %v", err)
				continue
			}

			// Convert row to event
			event := convertRowToEvent(row)

			// Send event through channel (check for context cancellation)
			select {
			case eventChan <- event:
				// Event sent successfully
			case <-ctx.Done():
				// Context cancelled, stop streaming
				return
			}
		}

		// Check for errors during iteration
		if err := rows.Err(); err != nil {
			log.Printf("Row iteration error in ReadStream: %v", err)
		}
	}()

	return eventChan, nil
}
