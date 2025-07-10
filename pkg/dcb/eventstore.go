package dcb

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// eventStore implements the EventStore interface using PostgreSQL
type eventStore struct {
	pool   *pgxpool.Pool
	config EventStoreConfig
}

// newEventStore creates a new eventStore instance
func newEventStore(ctx context.Context, pool *pgxpool.Pool, config *EventStoreConfig) (*eventStore, error) {
	// Test the connection
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}
	var cfg EventStoreConfig
	if config != nil {
		cfg = *config
	} else {
		cfg = EventStoreConfig{
			MaxBatchSize:           1000,
			LockTimeout:            5000,
			StreamBuffer:           1000,
			DefaultAppendIsolation: IsolationLevelReadCommitted,
			ReadTimeout:            15000, // 15 seconds default
		}
	}
	return &eventStore{
		pool:   pool,
		config: cfg,
	}, nil
}

// Remove GetLockTimeout method - lock timeout is now accessed via GetConfig().LockTimeout

// GetConfig returns the current EventStore configuration
func (es *eventStore) GetConfig() EventStoreConfig {
	return es.config
}

// Remove ReadWithOptions and Read methods (now in read.go)
