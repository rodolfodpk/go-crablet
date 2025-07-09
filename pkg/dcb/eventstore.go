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
	}, nil
}

// Remove ReadWithOptions and Read methods (now in read.go)
