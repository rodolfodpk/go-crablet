package dcb

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

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
		SELECT type, position, tags, data
		FROM events
		ORDER BY position
	`)
	Expect(err).NotTo(HaveOccurred())
	defer rows.Close()
	type EventRecord struct {
		Type     string      `json:"type"`
		Position int64       `json:"position"`
		Tags     interface{} `json:"tags"`
		Data     interface{} `json:"data"`
	}
	events := []EventRecord{}
	for rows.Next() {
		var (
			type_     string
			position  int64
			tagsArray []string
			dataBytes []byte
		)
		err := rows.Scan(&type_, &position, &tagsArray, &dataBytes)
		Expect(err).NotTo(HaveOccurred())

		// Convert tags array to map for JSON output
		tagsMap := make(map[string]string)
		for _, tagItem := range tagsArray {
			if parts := strings.SplitN(tagItem, ":", 2); len(parts) == 2 {
				tagsMap[parts[0]] = parts[1]
			}
		}

		var data interface{}
		err = json.Unmarshal(dataBytes, &data)
		Expect(err).NotTo(HaveOccurred())
		events = append(events, EventRecord{
			Type:     type_,
			Position: position,
			Tags:     tagsMap,
			Data:     data,
		})
	}
	Expect(rows.Err()).NotTo(HaveOccurred())
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

// NewEventStoreFromPool creates a new EventStore from an existing pool without connection testing
// This is used for tests that share a PostgreSQL container
func NewEventStoreFromPool(pool *pgxpool.Pool) EventStore {
	return &eventStore{
		pool:         pool,
		maxBatchSize: 1000, // Default maximum batch size
	}
}

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
