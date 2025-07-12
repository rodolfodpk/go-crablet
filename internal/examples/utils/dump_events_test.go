package utils

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"go-crablet/pkg/dcb"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// cleanupEvents truncates the events table to ensure test isolation
func cleanupEvents(t *testing.T, pool *pgxpool.Pool) {
	// Create context with timeout for cleanup
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_, err := pool.Exec(ctx, "TRUNCATE TABLE events RESTART IDENTITY CASCADE")
	require.NoError(t, err)
}

// setupTestDatabase creates a test database using testcontainers
func setupTestDatabase(ctx context.Context) (*pgxpool.Pool, testcontainers.Container, error) {
	// Generate a random password
	password, err := generateRandomPassword(16)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate password: %w", err)
	}

	req := testcontainers.ContainerRequest{
		Image:        "postgres:17.5-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_PASSWORD": password,
		},
		WaitingFor: wait.ForListeningPort("5432/tcp"),
	}

	postgresC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, nil, err
	}

	host, err := postgresC.Host(ctx)
	if err != nil {
		return nil, nil, err
	}

	port, err := postgresC.MappedPort(ctx, "5432")
	if err != nil {
		return nil, nil, err
	}

	dsn := fmt.Sprintf("postgres://postgres:%s@%s:%s/postgres?sslmode=disable", password, host, port.Port())
	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, nil, err
	}

	// Configure prepared statement cache settings
	poolConfig.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeCacheDescribe
	poolConfig.ConnConfig.StatementCacheCapacity = 100

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, nil, err
	}

	// Read and execute schema.sql
	schemaSQL, err := os.ReadFile("../../../docker-entrypoint-initdb.d/schema.sql")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read schema: %w", err)
	}

	// Filter out psql meta-commands that don't work with Go's database driver
	filteredSQL := filterPsqlCommands(string(schemaSQL))

	// Execute schema
	_, err = pool.Exec(ctx, filteredSQL)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to execute schema: %w", err)
	}

	return pool, postgresC, nil
}

// generateRandomPassword creates a random password string
func generateRandomPassword(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes)[:length], nil
}

// filterPsqlCommands removes psql meta-commands and psql-only SQL from schema.sql
func filterPsqlCommands(sql string) string {
	lines := strings.Split(sql, "\n")
	var filteredLines []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Skip psql meta-commands (lines starting with \) and empty lines
		if strings.HasPrefix(trimmed, "\\") || trimmed == "" {
			continue
		}
		// Skip lines that contain \gexec (psql command)
		if strings.Contains(line, "\\gexec") {
			continue
		}
		// Skip comment lines
		if strings.HasPrefix(trimmed, "--") {
			continue
		}
		filteredLines = append(filteredLines, line)
	}

	return strings.Join(filteredLines, "\n")
}

func TestDumpEvents(t *testing.T) {
	// Create context with timeout for the entire test
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Set up test database using testcontainers
	pool, container, err := setupTestDatabase(ctx)
	require.NoError(t, err)
	defer pool.Close()
	defer container.Terminate(ctx)

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
		err := store.Append(ctx, []dcb.InputEvent{event}, nil)
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

		err := store.Append(ctx, events, nil)
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

		err := store.Append(ctx, []dcb.InputEvent{event}, nil)
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

		err := store.Append(ctx, []dcb.InputEvent{event}, nil)
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
