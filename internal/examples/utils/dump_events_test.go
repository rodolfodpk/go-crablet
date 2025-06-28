package utils

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// cleanupEvents truncates the events table to ensure test isolation
func cleanupEvents(t *testing.T, pool *pgxpool.Pool) {
	ctx := context.Background()
	_, err := pool.Exec(ctx, "TRUNCATE TABLE events RESTART IDENTITY CASCADE")
	require.NoError(t, err)
}

func TestDumpEvents(t *testing.T) {
	ctx := context.Background()

	// Connect to test database
	pool, err := pgxpool.New(ctx, "postgres://postgres:postgres@localhost:5432/dcb_app?sslmode=disable")
	require.NoError(t, err)
	defer pool.Close()

	t.Run("Empty Database", func(t *testing.T) {
		cleanupEvents(t, pool)

		// DumpEvents should not panic on empty database
		assert.NotPanics(t, func() {
			DumpEvents(ctx, pool)
		})
	})

	t.Run("Single Event", func(t *testing.T) {
		cleanupEvents(t, pool)

		// Insert a test event
		_, err := pool.Exec(ctx, `
			INSERT INTO events (type, tags, data) 
			VALUES ($1, $2, $3)
		`, "UserCreated", []string{"user_id:123"}, `{"name": "John Doe"}`)
		require.NoError(t, err)

		// DumpEvents should not panic with single event
		assert.NotPanics(t, func() {
			DumpEvents(ctx, pool)
		})
	})

	t.Run("Multiple Events", func(t *testing.T) {
		cleanupEvents(t, pool)

		// Insert multiple test events
		events := []struct {
			eventType string
			tags      []string
			data      string
		}{
			{"UserCreated", []string{"user_id:123"}, `{"name": "John Doe"}`},
			{"UserUpdated", []string{"user_id:123"}, `{"name": "John Smith"}`},
			{"OrderCreated", []string{"order_id:456", "user_id:123"}, `{"total": 100.50}`},
		}

		for _, event := range events {
			_, err := pool.Exec(ctx, `
				INSERT INTO events (type, tags, data) 
				VALUES ($1, $2, $3)
			`, event.eventType, event.tags, event.data)
			require.NoError(t, err)
		}

		// DumpEvents should not panic with multiple events
		assert.NotPanics(t, func() {
			DumpEvents(ctx, pool)
		})
	})

	t.Run("Long Tags and Data", func(t *testing.T) {
		cleanupEvents(t, pool)

		// Insert event with long tags and data
		longTags := []string{"very_long_tag_key_that_exceeds_limit", "another_long_tag_value_here"}
		longData := `{"very_long_field_name": "This is a very long data value that should be truncated when displayed in the output"}`

		_, err := pool.Exec(ctx, `
			INSERT INTO events (type, tags, data) 
			VALUES ($1, $2, $3)
		`, "TestEvent", longTags, longData)
		require.NoError(t, err)

		// DumpEvents should not panic with long content
		assert.NotPanics(t, func() {
			DumpEvents(ctx, pool)
		})
	})

	t.Run("Special Characters in Data", func(t *testing.T) {
		cleanupEvents(t, pool)

		// Insert event with special characters
		specialData := `{"message": "Hello\nWorld\twith\"quotes\"and'single'quotes"}`

		_, err := pool.Exec(ctx, `
			INSERT INTO events (type, tags, data) 
			VALUES ($1, $2, $3)
		`, "SpecialEvent", []string{"test:value"}, specialData)
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
