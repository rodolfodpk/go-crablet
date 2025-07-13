package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/rodolfodpk/go-crablet/pkg/dcb"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

var _ = Describe("BatchExample", func() {
	var (
		ctx       context.Context
		cancel    context.CancelFunc
		pool      *pgxpool.Pool
		container testcontainers.Container
		store     dcb.EventStore
	)

	BeforeSuite(func() {
		ctx, cancel = context.WithTimeout(context.Background(), 2*time.Minute)
		var err error
		pool, container, err = setupTestDatabase(ctx)
		Expect(err).NotTo(HaveOccurred())
		store, err = dcb.NewEventStore(ctx, pool)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterSuite(func() {
		if pool != nil {
			pool.Close()
		}
		if container != nil {
			container.Terminate(ctx)
		}
		if cancel != nil {
			cancel()
		}
	})

	cleanupEvents := func() {
		_, err := pool.Exec(ctx, "TRUNCATE TABLE events RESTART IDENTITY CASCADE")
		Expect(err).NotTo(HaveOccurred())
	}

	Describe("BatchExample", func() {
		BeforeEach(func() {
			cleanupEvents()
		})

		It("should create a user", func() {
			createUserCmd := CreateUserCommand{
				UserID:   "test_user123",
				Username: "john_doe",
				Email:    "john@example.com",
			}
			err := handleCreateUser(ctx, store, createUserCmd)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should create an order after creating a user", func() {
			createUserCmd := CreateUserCommand{
				UserID:   "test_user123",
				Username: "john_doe",
				Email:    "john@example.com",
			}
			err := handleCreateUser(ctx, store, createUserCmd)
			Expect(err).NotTo(HaveOccurred())

			createOrderCmd := CreateOrderCommand{
				OrderID: "test_order456",
				UserID:  "test_user123",
				Items: []OrderItem{
					{ProductID: "prod1", Quantity: 2, Price: 29.99},
					{ProductID: "prod2", Quantity: 1, Price: 49.99},
				},
			}
			err = handleCreateOrder(ctx, store, createOrderCmd)
			Expect(err).NotTo(HaveOccurred())
		})

		// Test business rules
		Describe("Business Rules", func() {
			BeforeEach(func() {
				cleanupEvents()
			})

			It("should prevent creating user with same ID", func() {
				createUserCmd := CreateUserCommand{
					UserID:   "test_user123",
					Username: "john_doe",
					Email:    "john@example.com",
				}
				err := handleCreateUser(ctx, store, createUserCmd)
				Expect(err).NotTo(HaveOccurred())

				duplicateCmd := CreateUserCommand{
					UserID:   "test_user123", // Same ID as existing user
					Username: "jane_doe",
					Email:    "jane@example.com",
				}
				err = handleCreateUser(ctx, store, duplicateCmd)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("already exists"))
			})

			It("should prevent creating user with same email", func() {
				createUserCmd := CreateUserCommand{
					UserID:   "test_user123",
					Username: "john_doe",
					Email:    "john@example.com",
				}
				err := handleCreateUser(ctx, store, createUserCmd)
				Expect(err).NotTo(HaveOccurred())

				duplicateEmailCmd := CreateUserCommand{
					UserID:   "test_user456",
					Username: "jane_doe",
					Email:    "john@example.com", // Same email as existing user
				}
				err = handleCreateUser(ctx, store, duplicateEmailCmd)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("already exists"))
			})

			It("should prevent creating order with same ID", func() {
				createUserCmd := CreateUserCommand{
					UserID:   "test_user123",
					Username: "john_doe",
					Email:    "john@example.com",
				}
				err := handleCreateUser(ctx, store, createUserCmd)
				Expect(err).NotTo(HaveOccurred())

				createOrderCmd := CreateOrderCommand{
					OrderID: "test_order456",
					UserID:  "test_user123",
					Items: []OrderItem{
						{ProductID: "prod1", Quantity: 2, Price: 29.99},
					},
				}
				err = handleCreateOrder(ctx, store, createOrderCmd)
				Expect(err).NotTo(HaveOccurred())

				duplicateOrderCmd := CreateOrderCommand{
					OrderID: "test_order456", // Same ID as existing order
					UserID:  "test_user123",
					Items: []OrderItem{
						{ProductID: "prod3", Quantity: 1, Price: 19.99},
					},
				}
				err = handleCreateOrder(ctx, store, duplicateOrderCmd)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("already exists"))
			})

			It("should prevent creating order for non-existent user", func() {
				nonExistentUserCmd := CreateOrderCommand{
					OrderID: "test_order789",
					UserID:  "non_existent_user",
					Items: []OrderItem{
						{ProductID: "prod1", Quantity: 1, Price: 29.99},
					},
				}
				err := handleCreateOrder(ctx, store, nonExistentUserCmd)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("does not exist"))
			})
		})

		// Test batch operations
		Describe("Batch Operations", func() {
			BeforeEach(func() {
				cleanupEvents()
			})

			It("should batch create users", func() {
				users := []CreateUserCommand{
					{UserID: "batch_user1", Username: "batch_user1", Email: "batch1@example.com"},
					{UserID: "batch_user2", Username: "batch_user2", Email: "batch2@example.com"},
				}
				err := handleBatchCreateUsers(ctx, store, users)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should batch create orders", func() {
				users := []CreateUserCommand{
					{UserID: "batch_user1", Username: "batch_user1", Email: "batch1@example.com"},
					{UserID: "batch_user2", Username: "batch_user2", Email: "batch2@example.com"},
				}
				err := handleBatchCreateUsers(ctx, store, users)
				Expect(err).NotTo(HaveOccurred())

				orders := []CreateOrderCommand{
					{
						OrderID: "batch_order1",
						UserID:  "batch_user1",
						Items: []OrderItem{
							{ProductID: "prod1", Quantity: 1, Price: 29.99},
						},
					},
					{
						OrderID: "batch_order2",
						UserID:  "batch_user2",
						Items: []OrderItem{
							{ProductID: "prod2", Quantity: 2, Price: 49.99},
						},
					},
				}
				err = handleBatchCreateOrders(ctx, store, orders)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should batch validation - one user already exists", func() {
				createUserCmd := CreateUserCommand{
					UserID:   "batch_user1",
					Username: "batch_user1",
					Email:    "batch1@example.com",
				}
				err := handleCreateUser(ctx, store, createUserCmd)
				Expect(err).NotTo(HaveOccurred())

				users := []CreateUserCommand{
					{UserID: "batch_user3", Username: "batch_user3", Email: "batch3@example.com"},
					{UserID: "batch_user1", Username: "batch_user1_duplicate", Email: "batch1_duplicate@example.com"}, // Already exists
				}
				err = handleBatchCreateUsers(ctx, store, users)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("already exists"))
			})

			It("should batch validation - one order already exists", func() {
				createUserCmd := CreateUserCommand{
					UserID:   "batch_user3",
					Username: "batch_user3",
					Email:    "batch3@example.com",
				}
				err := handleCreateUser(ctx, store, createUserCmd)
				Expect(err).NotTo(HaveOccurred())

				createOrderCmd := CreateOrderCommand{
					OrderID: "batch_order1",
					UserID:  "batch_user3",
					Items: []OrderItem{
						{ProductID: "prod1", Quantity: 1, Price: 29.99},
					},
				}
				err = handleCreateOrder(ctx, store, createOrderCmd)
				Expect(err).NotTo(HaveOccurred())

				orders := []CreateOrderCommand{
					{
						OrderID: "batch_order3",
						UserID:  "batch_user3",
						Items: []OrderItem{
							{ProductID: "prod3", Quantity: 1, Price: 19.99},
						},
					},
					{
						OrderID: "batch_order1", // Already exists
						UserID:  "batch_user3",
						Items: []OrderItem{
							{ProductID: "prod4", Quantity: 1, Price: 39.99},
						},
					},
				}
				err = handleBatchCreateOrders(ctx, store, orders)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("order batch_order1 already exists"))
			})
		})
	})
})

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
