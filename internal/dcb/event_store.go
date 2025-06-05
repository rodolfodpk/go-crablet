package dcb

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// eventStore implements EventStore.
type eventStore struct {
	pool         *pgxpool.Pool // Database connection pool
	mu           sync.RWMutex  // Changed to RWMutex for better concurrency
	closed       bool          // Indicates if the store has been closed
	maxBatchSize int           // Maximum number of events in a single batch operation
	cleanupOnce  sync.Once     // Ensures cleanup happens only once
}

// NewEventStore creates a new EventStore using the provided PostgreSQL connection pool.
// It uses a default maximum batch size of 1000 events.
func NewEventStore(ctx context.Context, pool *pgxpool.Pool) (EventStore, error) {
	// Test the connection with context timeout
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := pool.Ping(ctx); err != nil {
		return nil, &ResourceError{
			EventStoreError: EventStoreError{
				Op:  "NewEventStore",
				Err: fmt.Errorf("unable to connect to database: %w", err),
			},
			Resource: "database",
		}
	}

	return &eventStore{
		pool:         pool,
		maxBatchSize: 1000, // Default maximum batch size
	}, nil
}

// Close closes the event store's connection pool.
// It is safe to call Close multiple times.
func (es *eventStore) Close() {
	es.cleanupOnce.Do(func() {
		es.mu.Lock()
		defer es.mu.Unlock()

		if !es.closed {
			es.closed = true
			// Close the pool in a separate goroutine to avoid blocking
			go func() {
				// Use a timeout context for pool closure
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				// Gracefully close the pool with timeout
				done := make(chan struct{})
				go func() {
					es.pool.Close()
					close(done)
				}()

				select {
				case <-ctx.Done():
					// Context timed out, but pool.Close() will still run in background
					return
				case <-done:
					// Pool closed successfully
					return
				}
			}()
		}
	})
}

// isClosed checks if the event store is closed
func (es *eventStore) isClosed() bool {
	es.mu.RLock()
	defer es.mu.RUnlock()
	return es.closed
}
