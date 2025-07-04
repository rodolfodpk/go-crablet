package utils

import (
	"context"
	"testing"
	"time"

	"go-crablet/pkg/dcb"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// cleanupEvents truncates the events table to ensure test isolation
func cleanupEvents(t *testing.T, pool *pgxpool.Pool) {
	// Create context with timeout for cleanup
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_, err := pool.Exec(ctx, "TRUNCATE TABLE events RESTART IDENTITY CASCADE")
	require.NoError(t, err)
}

func TestDumpEvents(t *testing.T) {
	// Create context with timeout for the entire test
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Connect to test database
	pool, err := pgxpool.New(ctx, "postgres://postgres:postgres@localhost:5432/dcb_app?sslmode=disable")
	require.NoError(t, err)
	defer pool.Close()

	// Create event store
	store := dcb.NewEventStoreFromPool(pool)

	t.Run("Empty Database", func(t *testing.T) {
		cleanupEvents(t, pool)

		// DumpEvents should not panic on empty database
		assert.NotPanics(t, func() {
			DumpEvents(ctx, pool)
		})
	})

	t.Run("Single Event", func(t *testing.T) {
		cleanupEvents(t, pool)

		// Insert a test event using EventStore API
		event := dcb.NewInputEvent("UserCreated", dcb.NewTags("user_id", "123"), []byte(`{"name": "John Doe"}`))
		err := store.Append(ctx, []dcb.InputEvent{event})
		require.NoError(t, err)

		// DumpEvents should not panic with single event
		assert.NotPanics(t, func() {
			DumpEvents(ctx, pool)
		})
	})

	t.Run("Multiple Events", func(t *testing.T) {
		cleanupEvents(t, pool)

		// Insert multiple test events using EventStore API
		events := []dcb.InputEvent{
			dcb.NewInputEvent("UserCreated", dcb.NewTags("user_id", "123"), []byte(`{"name": "John Doe"}`)),
			dcb.NewInputEvent("UserUpdated", dcb.NewTags("user_id", "123"), []byte(`{"name": "John Smith"}`)),
			dcb.NewInputEvent("OrderCreated", dcb.NewTags("order_id", "456", "user_id", "123"), []byte(`{"total": 100.50}`)),
		}

		err := store.Append(ctx, events)
		require.NoError(t, err)

		// DumpEvents should not panic with multiple events
		assert.NotPanics(t, func() {
			DumpEvents(ctx, pool)
		})
	})

	t.Run("Long Tags and Data", func(t *testing.T) {
		cleanupEvents(t, pool)

		// Insert event with long tags and data using EventStore API
		longData := `{"very_long_field_name": "This is a very long data value that should be truncated when displayed in the output"}`
		event := dcb.NewInputEvent("TestEvent", dcb.NewTags("very_long_tag_key_that_exceeds_limit", "another_long_tag_value_here"), []byte(longData))

		err := store.Append(ctx, []dcb.InputEvent{event})
		require.NoError(t, err)

		// DumpEvents should not panic with long content
		assert.NotPanics(t, func() {
			DumpEvents(ctx, pool)
		})
	})

	t.Run("Special Characters in Data", func(t *testing.T) {
		cleanupEvents(t, pool)

		// Insert event with special characters using EventStore API
		specialData := `{"message": "Hello\nWorld\twith\"quotes\"and'single'quotes"}`
		event := dcb.NewInputEvent("SpecialEvent", dcb.NewTags("test", "value"), []byte(specialData))

		err := store.Append(ctx, []dcb.InputEvent{event})
		require.NoError(t, err)

		// DumpEvents should not panic with special characters
		assert.NotPanics(t, func() {
			DumpEvents(ctx, pool)
		})
	})

	t.Run("Database Connection Error", func(t *testing.T) {
		// Create an invalid pool to test error handling
		invalidPool, err := pgxpool.New(ctx, "postgres://invalid:invalid@localhost:9999/invalid?sslmode=disable")
		require.NoError(t, err) // Pool creation might succeed, but query will fail
		defer invalidPool.Close()

		// DumpEvents should not panic and should handle the error gracefully
		assert.NotPanics(t, func() {
			DumpEvents(ctx, invalidPool)
		})
	})
}
