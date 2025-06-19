package dcb

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/jackc/pgx/v5"
)

// CursorEventIterator implements EventIterator for true streaming events from PostgreSQL using cursors
type CursorEventIterator struct {
	tx           pgx.Tx
	cursorName   string
	batchSize    int
	currentBatch pgx.Rows
	event        Event
	err          error
	done         bool
	sqlQuery     string
	args         []interface{}
}

// ReadStream returns a pure event iterator for streaming events from PostgreSQL using server-side cursors
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

	// Validate query items
	if err := validateQueryTags(query); err != nil {
		return nil, err
	}

	// Build SQL query based on query items
	sqlQuery, args, err := es.buildReadQuerySQL(query, options)
	if err != nil {
		return nil, err
	}

	// Start transaction for cursor
	tx, err := es.pool.Begin(ctx)
	if err != nil {
		return nil, &ResourceError{
			EventStoreError: EventStoreError{
				Op:  "ReadStream",
				Err: fmt.Errorf("failed to start transaction: %w", err),
			},
			Resource: "database",
		}
	}

	// Generate unique cursor name
	cursorName := fmt.Sprintf("cursor_%d_%d", time.Now().UnixNano(), rand.Int63())

	// Set batch size (default 1000)
	batchSize := 1000
	if options != nil && options.BatchSize != nil && *options.BatchSize > 0 {
		batchSize = *options.BatchSize
	}

	// Declare cursor
	declareSQL := fmt.Sprintf("DECLARE %s CURSOR FOR %s", cursorName, sqlQuery)
	_, err = tx.Exec(ctx, declareSQL, args...)
	if err != nil {
		tx.Rollback(ctx)
		return nil, &ResourceError{
			EventStoreError: EventStoreError{
				Op:  "ReadStream",
				Err: fmt.Errorf("failed to declare cursor: %w", err),
			},
			Resource: "database",
		}
	}

	return &CursorEventIterator{
		tx:         tx,
		cursorName: cursorName,
		batchSize:  batchSize,
		sqlQuery:   sqlQuery,
		args:       args,
	}, nil
}

// Next processes the next event
func (it *CursorEventIterator) Next() bool {
	if it.done {
		return false
	}

	// If we have a current batch, try to get next row from it
	if it.currentBatch != nil {
		if it.currentBatch.Next() {
			it.scanCurrentRow()
			return true
		}
		// Current batch exhausted, close it
		it.currentBatch.Close()
		it.currentBatch = nil
	}

	// Fetch next batch
	if !it.fetchNextBatch() {
		return false
	}

	// Try to get first row from new batch
	if it.currentBatch.Next() {
		it.scanCurrentRow()
		return true
	}

	// No more data
	it.done = true
	return false
}

// fetchNextBatch fetches the next batch of rows from the cursor
func (it *CursorEventIterator) fetchNextBatch() bool {
	fetchSQL := fmt.Sprintf("FETCH %d FROM %s", it.batchSize, it.cursorName)
	rows, err := it.tx.Query(context.Background(), fetchSQL)
	if err != nil {
		it.err = &ResourceError{
			EventStoreError: EventStoreError{
				Op:  "CursorEventIterator.fetchNextBatch",
				Err: fmt.Errorf("failed to fetch from cursor: %w", err),
			},
			Resource: "database",
		}
		return false
	}

	it.currentBatch = rows
	return true
}

// scanCurrentRow scans the current row from the batch
func (it *CursorEventIterator) scanCurrentRow() {
	var row rowEvent
	if err := it.currentBatch.Scan(&row.Type, &row.Tags, &row.Data, &row.Position); err != nil {
		it.err = &ResourceError{
			EventStoreError: EventStoreError{
				Op:  "CursorEventIterator.scanCurrentRow",
				Err: fmt.Errorf("failed to scan event row: %w", err),
			},
			Resource: "database",
		}
		return
	}

	// Convert row to Event
	it.event = convertRowToEvent(row)
}

// Event returns the current event
func (it *CursorEventIterator) Event() Event {
	return it.event
}

// Err returns any error that occurred during iteration
func (it *CursorEventIterator) Err() error {
	if it.err != nil {
		return it.err
	}
	if it.currentBatch != nil {
		return it.currentBatch.Err()
	}
	return nil
}

// Close closes the iterator and releases resources
func (it *CursorEventIterator) Close() error {
	// Close current batch if open
	if it.currentBatch != nil {
		it.currentBatch.Close()
		it.currentBatch = nil
	}

	// Close cursor
	if it.tx != nil {
		closeSQL := fmt.Sprintf("CLOSE %s", it.cursorName)
		it.tx.Exec(context.Background(), closeSQL)

		// Rollback transaction (cursors are automatically cleaned up)
		it.tx.Rollback(context.Background())
	}

	return nil
}
