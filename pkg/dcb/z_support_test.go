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
	pool      *pgxpool.Pool
	store     EventStore
	container testcontainers.Container
)

// Test setup and teardown
var _ = BeforeSuite(func() {
	// Create context with timeout for test setup
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Initialize test database using testcontainers
	var err error
	pool, container, err = setupPostgresContainer(ctx)
	Expect(err).NotTo(HaveOccurred())

	// Read and execute schema.sql (path from pkg/dcb to root)
	schemaSQL, err := os.ReadFile("../../docker-entrypoint-initdb.d/schema.sql")
	Expect(err).NotTo(HaveOccurred())

	// Filter out psql meta-commands that don't work with Go's database driver
	filteredSQL := filterPsqlCommands(string(schemaSQL))

	// Debug: print the filtered SQL
	fmt.Printf("Filtered SQL:\n%s\n", filteredSQL)

	// Execute schema
	_, err = pool.Exec(ctx, filteredSQL)
	Expect(err).NotTo(HaveOccurred())

	// Create event store
	store, err = NewEventStore(ctx, pool)
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
	if pool != nil {
		pool.Close()
	}
	if container != nil {
		container.Terminate(ctx)
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

	return pool, postgresC, nil
}

// truncateEventsTable resets the events table before each test
func truncateEventsTable(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, "TRUNCATE TABLE events RESTART IDENTITY CASCADE")
	return err
}

// dumpEvents queries the events table and prints the results as JSON
func dumpEvents(pool *pgxpool.Pool) {
	rows, err := pool.Query(ctx, `
		SELECT type, transaction_id, position, tags, data
		FROM events 
		ORDER BY transaction_id, position
	`)
	if err != nil {
		return
	}
	defer rows.Close()

	// Event structure for scanning
	type Event struct {
		Type          string          `json:"type"`
		TransactionID uint64          `json:"transaction_id"`
		Position      int64           `json:"position"`
		Tags          []string        `json:"tags"`
		Data          json.RawMessage `json:"data"`
	}

	var events []Event
	for rows.Next() {
		var event Event
		var tagsArray []string
		var dataBytes []byte

		err := rows.Scan(&event.Type, &event.TransactionID, &event.Position, &tagsArray, &dataBytes)
		if err != nil {
			return
		}

		event.Tags = tagsArray
		event.Data = dataBytes

		events = append(events, event)
	}

	jsonData, err := json.MarshalIndent(events, "", "  ")
	Expect(err).NotTo(HaveOccurred())
	GinkgoWriter.Println("--- Events Table Contents (JSON) ---")
	GinkgoWriter.Println(string(jsonData))
	GinkgoWriter.Printf("Total events: %d\n", len(events))
	GinkgoWriter.Println("------------------------------------")
	fmt.Println("--- Events Table Contents (JSON) ---")
	fmt.Println(string(jsonData))
	fmt.Printf("Total events: %d\n", len(events))
	fmt.Println("------------------------------------")
}

// filterPsqlCommands removes psql meta-commands and psql-only SQL from schema.sql
func filterPsqlCommands(sql string) string {
	lines := strings.Split(sql, "\n")
	var filteredLines []string

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// Remove lines that are psql meta-commands or psql-only SQL
		if strings.HasPrefix(trimmedLine, "\\") {
			continue
		}
		if strings.Contains(trimmedLine, "\\gexec") {
			continue
		}
		if strings.Contains(trimmedLine, "SELECT 'CREATE DATABASE") {
			continue
		}

		// Skip empty lines after filtering
		if trimmedLine == "" {
			continue
		}

		filteredLines = append(filteredLines, trimmedLine)
	}

	return strings.Join(filteredLines, "\n")
}
