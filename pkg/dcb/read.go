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

// ReadChannel creates a channel-based stream of events matching a query
// This is optimized for small to medium datasets (< 500 events) and provides
// a more Go-idiomatic interface using channels
func (es *eventStore) ReadChannel(ctx context.Context, query Query) (<-chan Event, error) {
	// Validate that the query is not empty (same validation as Read method)
	if len(query.getItems()) == 0 {
		return nil, &ValidationError{
			EventStoreError: EventStoreError{
				Op:  "ReadChannel",
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

	// Build the SQL query
	sqlQuery, args, err := es.buildReadQuerySQL(query, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	// Use the caller's context directly for streaming
	queryCtx := ctx

	// Execute the query
	rows, err := es.pool.Query(queryCtx, sqlQuery, args...)
	if err != nil {
		return nil, &ResourceError{
			EventStoreError: EventStoreError{
				Op:  "ReadChannel",
				Err: fmt.Errorf("query failed: %w", err),
			},
			Resource: "database",
		}
	}

	// Create result channel with configurable buffer
	resultChan := make(chan Event, es.config.StreamBuffer)

	// Start streaming events in a goroutine
	go func() {
		// Ensure rows are always closed, even if goroutine panics
		defer func() {
			if r := recover(); r != nil {
				log.Printf("ReadChannel panic recovered: %v", r)
			}
			rows.Close()
			close(resultChan)
		}()

		for rows.Next() {
			select {
			case <-ctx.Done():
				// Context cancelled - exit cleanly
				return
			case resultChan <- func() Event {
				var row rowEvent
				err := rows.Scan(
					&row.Type,
					&row.Tags,
					&row.Data,
					&row.Position,
					&row.TransactionID,
					&row.CreatedAt,
				)
				if err != nil {
					// Log error and return a sentinel event that consumers can detect
					log.Printf("Error scanning row in ReadChannel: %v", err)
					// Return an event with empty type to indicate error
					return Event{
						Type: "", // Empty type indicates scan error
						Tags: []Tag{},
						Data: []byte{},
					}
				}

				return convertRowToEvent(row)
			}():
				// Event sent successfully
			}
		}

		// Check for row iteration errors
		if err := rows.Err(); err != nil {
			log.Printf("Row iteration error in ReadChannel: %v", err)
		}
	}()

	return resultChan, nil
}
