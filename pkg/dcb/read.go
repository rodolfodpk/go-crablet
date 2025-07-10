package dcb

import (
	"context"
	"fmt"
	"log"
	"time"
)

// Read reads events matching the query (no options)
func (es *eventStore) Read(ctx context.Context, query Query) ([]Event, error) {
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
	sqlQuery, args, err := es.buildReadQuerySQL(query, nil, nil)
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
		var row rowEvent

		if err := rows.Scan(&row.Type, &row.Tags, &row.Data, &row.Position, &row.TransactionID, &row.CreatedAt); err != nil {
			return nil, &ResourceError{
				EventStoreError: EventStoreError{
					Op:  "read",
					Err: fmt.Errorf("failed to scan event row: %w", err),
				},
				Resource: "database",
			}
		}

		// Convert row to Event using the helper function
		event := convertRowToEvent(row)
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

// ReadStream creates a channel-based stream of events matching a query
// This replaces ReadWithOptions functionality - the caller manages complexity
// like limits and cursors through channel consumption patterns
// This is optimized for small to medium datasets (< 500 events) and provides
// a more Go-idiomatic interface using channels
func (es *eventStore) ReadStream(ctx context.Context, query Query) (<-chan Event, error) {
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

		// Build SQL query
		sqlQuery, args, err := es.buildReadQuerySQL(query, nil, nil)
		if err != nil {
			log.Printf("Error building SQL query in ReadStream: %v", err)
			return
		}

		// Execute query
		rows, err := es.pool.Query(ctx, sqlQuery, args...)
		if err != nil {
			log.Printf("Error executing query in ReadStream: %v", err)
			return
		}
		defer rows.Close()

		// Stream events through channel
		for rows.Next() {
			var row rowEvent
			err := rows.Scan(&row.Type, &row.Tags, &row.Data, &row.Position, &row.TransactionID, &row.CreatedAt)
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
