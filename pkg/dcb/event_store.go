package dcb

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// eventStore implements EventStore.
type eventStore struct {
	pool         *pgxpool.Pool // Database connection pool
	maxBatchSize int           // Maximum number of events in a single batch operation
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

// GetCurrentPosition returns the current position for the given query.
func (es *eventStore) GetCurrentPosition(ctx context.Context, query Query) (int64, error) {
	// Convert query tags to JSONB
	queryTagMap := make(map[string]string)
	for _, t := range query.Tags {
		queryTagMap[t.Key] = t.Value
	}
	queryTagsJSON, err := json.Marshal(queryTagMap)
	if err != nil {
		return 0, &EventStoreError{
			Op:  "GetCurrentPosition",
			Err: fmt.Errorf("failed to marshal query tags: %w", err),
		}
	}

	// Query for the latest position
	var position int64
	err = es.pool.QueryRow(ctx, `
		SELECT COALESCE(MAX(position), 0)
		FROM events
		WHERE tags @> $1::jsonb
		  AND ($2::text[] IS NULL OR
			   array_length($2::text[], 1) = 0 OR
			   type = ANY($2::text[]))
	`, queryTagsJSON, query.EventTypes).Scan(&position)
	if err != nil {
		return 0, &EventStoreError{
			Op:  "GetCurrentPosition",
			Err: fmt.Errorf("failed to get current position: %w", err),
		}
	}

	return position, nil
}
