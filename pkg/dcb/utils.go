package dcb

import (
	"context"
	"fmt"
)

// TruncateEvents truncates the events table and resets the position sequence
// This is intended for testing and benchmarking purposes only
func TruncateEvents(ctx context.Context, store EventStore) error {
	// Type assert to get access to the underlying pool
	// This is safe because we control the implementation
	es, ok := store.(*eventStore)
	if !ok {
		return fmt.Errorf("store is not the expected implementation type")
	}

	// Truncate the events table and reset the sequence
	_, err := es.pool.Exec(ctx, "TRUNCATE TABLE events RESTART IDENTITY CASCADE")
	if err != nil {
		return fmt.Errorf("failed to truncate events table: %w", err)
	}

	return nil
}
