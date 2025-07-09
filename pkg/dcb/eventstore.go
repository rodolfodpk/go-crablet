package dcb

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// eventStore implements the EventStore interface using PostgreSQL
type eventStore struct {
	pool         *pgxpool.Pool
	maxBatchSize int
	lockTimeout  int // Lock timeout in milliseconds for advisory locks
	streamBuffer int // Channel buffer size for streaming operations
}

// newEventStore creates a new eventStore instance
func newEventStore(ctx context.Context, pool *pgxpool.Pool) (*eventStore, error) {
	// Test the connection
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &eventStore{
		pool:         pool,
		maxBatchSize: 1000, // Default maximum batch size
		lockTimeout:  5000, // Default 5 second lock timeout
		streamBuffer: 100,  // Default channel buffer size
	}, nil
}

// GetLockTimeout returns the lock timeout in milliseconds for advisory locks
func (es *eventStore) GetLockTimeout() int {
	return es.lockTimeout
}

// Remove ReadWithOptions and Read methods (now in read.go)
