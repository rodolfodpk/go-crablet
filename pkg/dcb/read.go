package dcb

import (
	"context"
	"fmt"
	"log"
	"time"
)

// ReadWithOptions reads events matching the query with additional options
func (es *eventStore) ReadWithOptions(ctx context.Context, query Query, options ReadOptions) ([]Event, error) {
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

// Read reads events matching the query (no options)
func (es *eventStore) Read(ctx context.Context, query Query) ([]Event, error) {
	return es.ReadWithOptions(ctx, query, ReadOptions{})
}

// ReadStreamChannel creates a channel-based stream of events matching a query
// This is optimized for small to medium datasets (< 500 events) and provides
// a more Go-idiomatic interface using channels
func (es *eventStore) ReadStreamChannel(ctx context.Context, query Query) (<-chan Event, *Cursor, error) {
	// Validate that the query is not empty (same validation as Read method)
	if len(query.getItems()) == 0 {
		return nil, nil, &ValidationError{
			EventStoreError: EventStoreError{
				Op:  "ReadStreamChannel",
				Err: fmt.Errorf("query must contain at least one item"),
			},
			Field: "query",
			Value: "empty",
		}
	}

	// Build the SQL query
	sqlQuery, args, err := es.buildReadQuerySQL(query, ReadOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build query: %w", err)
	}

	// Execute the query
	rows, err := es.pool.Query(ctx, sqlQuery, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("query failed: %w", err)
	}

	// Create result channel
	resultChan := make(chan Event, 100)

	// Track latest cursor
	var latestCursor *Cursor

	// Start streaming events in a goroutine
	go func() {
		// Ensure rows are always closed, even if goroutine panics
		defer func() {
			if r := recover(); r != nil {
				log.Printf("ReadStreamChannel panic recovered: %v", r)
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
					// Log error but continue processing
					log.Printf("Error scanning row in ReadStreamChannel: %v", err)
					return Event{} // Return empty event, will be filtered out
				}

				// Update latest cursor (events are ordered by transaction_id ASC, position ASC)
				latestCursor = &Cursor{
					TransactionID: row.TransactionID,
					Position:      row.Position,
				}

				return convertRowToEvent(row)
			}():
				// Event sent successfully
			}
		}

		// Check for row iteration errors
		if err := rows.Err(); err != nil {
			log.Printf("Row iteration error in ReadStreamChannel: %v", err)
		}
	}()

	return resultChan, latestCursor, nil
}
