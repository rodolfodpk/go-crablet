package dcb

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/rodolfodpk/go-crablet/pkg/dcb"

	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// Test globals
var (
	ctx       context.Context
	cancel    context.CancelFunc
	pool      *pgxpool.Pool
	store     dcb.EventStore
	container testcontainers.Container
)

// Test setup and teardown
var _ = BeforeSuite(func() {
	// Create context with timeout for test setup
	ctx, cancel = context.WithTimeout(context.Background(), 120*time.Second)
	// Don't defer cancel here - we need the context for tests
	// The context will be cleaned up in AfterSuite

	// Initialize test database using testcontainers
	var err error
	pool, container, err = setupPostgresContainer(context.Background()) // Use context.Background() for pool creation
	Expect(err).NotTo(HaveOccurred())

	// Wait a bit for the database to be fully ready
	time.Sleep(2 * time.Second)

	// Read and execute schema.sql (path from pkg/dcb/tests to root)
	schemaSQL, err := os.ReadFile("../../../docker-entrypoint-initdb.d/schema.sql")
	Expect(err).NotTo(HaveOccurred())

	// Filter out psql meta-commands that don't work with Go's database driver
	filteredSQL := filterPsqlCommands(string(schemaSQL))

	// Debug: print the filtered SQL
	fmt.Printf("Filtered SQL:\n%s\n", filteredSQL)

	// Execute schema with retry logic
	for i := 0; i < 3; i++ {
		_, err = pool.Exec(ctx, filteredSQL)
		if err == nil {
			break
		}
		time.Sleep(time.Duration(1<<uint(i)) * time.Second)
	}
	Expect(err).NotTo(HaveOccurred())

	// Create event store
	store, err = dcb.NewEventStore(ctx, pool)
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
	if cancel != nil {
		cancel() // Cancel the context
	}
	if pool != nil {
		pool.Close()
	}
	if container != nil {
		container.Terminate(context.Background()) // Use background context for cleanup
	}
})

// Helper functions

// toJSON marshals a struct to JSON bytes, panicking on error (for test convenience)
func toJSON(v any) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal to JSON: %v", err))
	}
	return data
}

// generateRandomPassword creates a random password string
func generateRandomPassword(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes)[:length], nil
}

// setupPostgresContainer creates and configures a Postgres test container
func setupPostgresContainer(ctx context.Context) (*pgxpool.Pool, testcontainers.Container, error) {
	// Generate a random password
	password, err := generateRandomPassword(16)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate password: %w", err)
	}

	req := testcontainers.ContainerRequest{
		Image:        "postgres:16.10",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_PASSWORD": password,
			"POSTGRES_USER":     "postgres",
			"POSTGRES_DB":       "postgres",
		},
		WaitingFor: wait.ForListeningPort("5432/tcp").WithStartupTimeout(60 * time.Second),
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

	// Add connection timeout and retry logic
	poolConfig.ConnConfig.ConnectTimeout = 30 * time.Second
	poolConfig.MaxConns = 10
	poolConfig.MinConns = 2

	// Retry connection with exponential backoff
	var pool *pgxpool.Pool
	for i := 0; i < 5; i++ {
		pool, err = pgxpool.NewWithConfig(ctx, poolConfig)
		if err == nil {
			break
		}
		
		// Wait before retry with exponential backoff
		waitTime := time.Duration(1<<uint(i)) * time.Second
		time.Sleep(waitTime)
	}
	
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect after retries: %w", err)
	}

	return pool, postgresC, nil
}

// truncateEventsTable resets the events table before each test
func truncateEventsTable(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, "TRUNCATE TABLE events RESTART IDENTITY CASCADE")
	return err
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
		filteredLines = append(filteredLines, line)
	}

	return strings.Join(filteredLines, "\n")
}

// TestDCB is the main test entry point for the Ginkgo test suite
func TestDCB(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "DCB Test Suite")
}
