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

var _ = Describe("Helper Functions", func() {
	Describe("NewQueryFromItems", func() {
		It("should create query from multiple items", func() {
			item1 := QueryItem{
				Types: []string{"Event1", "Event2"},
				Tags:  []Tag{{Key: "key1", Value: "value1"}},
			}
			item2 := QueryItem{
				Types: []string{"Event3"},
				Tags:  []Tag{{Key: "key2", Value: "value2"}},
			}

			query := NewQueryFromItems(item1, item2)

			Expect(query.Items).To(HaveLen(2))
			Expect(query.Items[0]).To(Equal(item1))
			Expect(query.Items[1]).To(Equal(item2))
		})

		It("should create empty query when no items provided", func() {
			query := NewQueryFromItems()

			Expect(query.Items).To(BeEmpty())
		})
	})

	Describe("NewQueryAll", func() {
		It("should create query that matches all events", func() {
			query := NewQueryAll()

			Expect(query.Items).To(HaveLen(1))
			Expect(query.Items[0].Types).To(BeEmpty())
			Expect(query.Items[0].Tags).To(BeEmpty())
		})
	})

	Describe("NewQueryItem", func() {
		It("should create query item with types and tags", func() {
			types := []string{"Event1", "Event2"}
			tags := []Tag{{Key: "key1", Value: "value1"}}

			item := NewQueryItem(types, tags)

			Expect(item.Types).To(Equal(types))
			Expect(item.Tags).To(Equal(tags))
		})

		It("should create query item with empty slices", func() {
			item := NewQueryItem([]string{}, []Tag{})

			Expect(item.Types).To(BeEmpty())
			Expect(item.Tags).To(BeEmpty())
		})
	})

	Describe("ToLegacyQuery", func() {
		It("should convert query to legacy query", func() {
			query := Query{
				Items: []QueryItem{
					{
						Types: []string{"Event1", "Event2"},
						Tags:  []Tag{{Key: "key1", Value: "value1"}},
					},
				},
			}

			legacy := query.ToLegacyQuery()

			Expect(legacy.EventTypes).To(Equal([]string{"Event1", "Event2"}))
			Expect(legacy.Tags).To(Equal([]Tag{{Key: "key1", Value: "value1"}}))
		})

		It("should return empty legacy query when query has no items", func() {
			query := Query{Items: []QueryItem{}}

			legacy := query.ToLegacyQuery()

			Expect(legacy.EventTypes).To(BeNil())
			Expect(legacy.Tags).To(BeNil())
		})

		It("should use first item when query has multiple items", func() {
			query := Query{
				Items: []QueryItem{
					{
						Types: []string{"Event1"},
						Tags:  []Tag{{Key: "key1", Value: "value1"}},
					},
					{
						Types: []string{"Event2"},
						Tags:  []Tag{{Key: "key2", Value: "value2"}},
					},
				},
			}

			legacy := query.ToLegacyQuery()

			Expect(legacy.EventTypes).To(Equal([]string{"Event1"}))
			Expect(legacy.Tags).To(Equal([]Tag{{Key: "key1", Value: "value1"}}))
		})
	})

	Describe("FromLegacyQuery", func() {
		It("should convert legacy query to query", func() {
			legacy := LegacyQuery{
				EventTypes: []string{"Event1", "Event2"},
				Tags:       []Tag{{Key: "key1", Value: "value1"}},
			}

			query := FromLegacyQuery(legacy)

			Expect(query.Items).To(HaveLen(1))
			Expect(query.Items[0].Types).To(Equal([]string{"Event1", "Event2"}))
			Expect(query.Items[0].Tags).To(Equal([]Tag{{Key: "key1", Value: "value1"}}))
		})

		It("should handle empty legacy query", func() {
			legacy := LegacyQuery{}

			query := FromLegacyQuery(legacy)

			Expect(query.Items).To(HaveLen(1))
			Expect(query.Items[0].Types).To(BeNil())
			Expect(query.Items[0].Tags).To(BeNil())
		})
	})
})
