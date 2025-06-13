package dcb

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// SimpleEventIterator implements EventIterator for streaming events from PostgreSQL
type SimpleEventIterator struct {
	rows  pgx.Rows
	err   error
	event Event
}

// ReadStream returns a pure event iterator for streaming events from PostgreSQL
func (es *eventStore) ReadStream(ctx context.Context, query Query, options *ReadOptions) (EventIterator, error) {
	if len(query.Items) == 0 {
		return nil, &ValidationError{
			EventStoreError: EventStoreError{
				Op:  "ReadStream",
				Err: fmt.Errorf("query must contain at least one item"),
			},
			Field: "query",
			Value: "empty",
		}
	}

	// Build SQL query based on query items
	sqlQuery, args, err := es.buildReadQuerySQL(query, options)
	if err != nil {
		return nil, err
	}

	// Execute the query - this starts the streaming
	rows, err := es.pool.Query(ctx, sqlQuery, args...)
	if err != nil {
		return nil, &ResourceError{
			EventStoreError: EventStoreError{
				Op:  "ReadStream",
				Err: fmt.Errorf("failed to execute read query: %w", err),
			},
			Resource: "database",
		}
	}

	return &SimpleEventIterator{rows: rows}, nil
}

// Next processes the next event
func (it *SimpleEventIterator) Next() bool {
	if !it.rows.Next() {
		return false
	}

	var row rowEvent
	if err := it.rows.Scan(&row.ID, &row.Type, &row.Tags, &row.Data, &row.Position, &row.CausationID, &row.CorrelationID); err != nil {
		it.err = &ResourceError{
			EventStoreError: EventStoreError{
				Op:  "SimpleEventIterator.Next",
				Err: fmt.Errorf("failed to scan event row: %w", err),
			},
			Resource: "database",
		}
		return false
	}

	// Convert row to Event
	it.event = convertRowToEvent(row)
	return true
}

// Event returns the current event
func (it *SimpleEventIterator) Event() Event {
	return it.event
}

// Err returns any error that occurred during iteration
func (it *SimpleEventIterator) Err() error {
	if it.err != nil {
		return it.err
	}
	return it.rows.Err()
}

// Close closes the iterator and releases resources
func (it *SimpleEventIterator) Close() error {
	it.rows.Close()
	return nil
}
