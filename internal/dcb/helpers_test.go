package dcb

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

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
		Image:        "postgres:15-alpine",
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
		SELECT id, type, position, tags, data, causation_id, correlation_id
		FROM events
		ORDER BY position
	`)
	Expect(err).NotTo(HaveOccurred())
	defer rows.Close()

	type EventRecord struct {
		ID            string      `json:"id"`
		Type          string      `json:"type"`
		Position      int64       `json:"position"`
		Tags          interface{} `json:"tags"`
		Data          interface{} `json:"data"`
		CausationID   string      `json:"causation_id"`
		CorrelationID string      `json:"correlation_id"`
	}

	events := []EventRecord{}
	for rows.Next() {
		var (
			id            string
			eventType     string
			position      int64
			tagsBytes     []byte
			dataBytes     []byte
			causationID   string
			correlationID string
		)

		err := rows.Scan(&id, &eventType, &position, &tagsBytes, &dataBytes, &causationID, &correlationID)
		Expect(err).NotTo(HaveOccurred())

		var tags interface{}
		err = json.Unmarshal(tagsBytes, &tags)
		Expect(err).NotTo(HaveOccurred())

		var data interface{}
		err = json.Unmarshal(dataBytes, &data)
		Expect(err).NotTo(HaveOccurred())

		events = append(events, EventRecord{
			ID:            id,
			Type:          eventType,
			Position:      position,
			Tags:          tags,
			Data:          data,
			CausationID:   causationID,
			CorrelationID: correlationID,
		})
	}
	Expect(rows.Err()).NotTo(HaveOccurred())

	// Convert events to JSON
	jsonData, err := json.MarshalIndent(events, "", "  ")
	Expect(err).NotTo(HaveOccurred())

	// Print the JSON data
	GinkgoWriter.Println("--- Events Table Contents (JSON) ---")
	GinkgoWriter.Println(string(jsonData))
	GinkgoWriter.Printf("Total events: %d\n", len(events))
	GinkgoWriter.Println("------------------------------------")

	// Also print to standard output to ensure visibility
	fmt.Println("--- Events Table Contents (JSON) ---")
	fmt.Println(string(jsonData))
	fmt.Printf("Total events: %d\n", len(events))
	fmt.Println("------------------------------------")

}
