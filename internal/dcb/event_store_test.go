package dcb

import (
	"context"
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/testcontainers/testcontainers-go"
)

func TestEventStore(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "EventStore Integration Suite")
}

var (
	ctx       context.Context
	pool      *pgxpool.Pool
	postgresC testcontainers.Container
	teardown  func()
	store     EventStore
)

// Define teardown function at package level
func setupTeardown() {
	teardown = func() {
		// Attempt to retrieve and print container logs
		if postgresC != nil {
			logsReader, err := postgresC.Logs(ctx)
			if err == nil {
				defer logsReader.Close()
				logBytes, readErr := io.ReadAll(logsReader)
				if readErr == nil && len(logBytes) > 0 {
					GinkgoWriter.Printf("--- PostgreSQL Container Logs ---\n%s\n-------------------------------\n", string(logBytes))
				} else if readErr != nil {
					GinkgoWriter.Printf("--- Error reading PostgreSQL Container Logs: %v ---\n", readErr)
				} else {
					GinkgoWriter.Println("--- PostgreSQL Container Logs: No logs produced. ---")
				}
			} else {
				GinkgoWriter.Printf("--- Error retrieving PostgreSQL Container Logs stream: %v ---\n", err)
			}
		}

		// Only close the pool, not the store
		if pool != nil {
			pool.Close()
		}
		if postgresC != nil {
			err := postgresC.Terminate(ctx)
			if err != nil {
				GinkgoWriter.Printf("--- Error terminating PostgreSQL Container: %v ---\n", err)
			}
		}
	}
}

var _ = BeforeSuite(func() {
	ctx = context.Background()
	var err error

	// Setup database container with retries
	Eventually(func() error {
		pool, postgresC, err = setupPostgresContainer(ctx)
		if err != nil {
			return fmt.Errorf("failed to setup postgres container: %w", err)
		}
		return nil
	}, 30*time.Second, 1*time.Second).Should(Succeed(), "Failed to setup postgres container after multiple attempts")

	// Wait for basic database connectivity
	Eventually(func() error {
		// Check basic connectivity
		if err := pool.Ping(ctx); err != nil {
			return fmt.Errorf("database ping failed: %w", err)
		}

		// Verify we can execute queries
		var result int
		if err := pool.QueryRow(ctx, "SELECT 1").Scan(&result); err != nil {
			return fmt.Errorf("database query test failed: %w", err)
		}
		if result != 1 {
			return fmt.Errorf("unexpected query result: %d", result)
		}

		return nil
	}, 30*time.Second, 1*time.Second).Should(Succeed(), "Basic database connectivity check failed")

	// Load and apply schema
	projectRoot := "../.." // Go up two levels from internal/dcb to the project root
	schemaPath := projectRoot + "/docker-entrypoint-initdb.d/schema.sql"
	schema, err := os.ReadFile(schemaPath)
	Expect(err).NotTo(HaveOccurred(), "Failed to read schema file")

	// Apply schema with retry
	Eventually(func() error {
		_, err = pool.Exec(ctx, string(schema))
		if err != nil {
			return fmt.Errorf("failed to apply schema: %w", err)
		}
		return nil
	}, 30*time.Second, 1*time.Second).Should(Succeed(), "Schema application failed")

	// Verify schema was applied correctly
	Eventually(func() error {
		// Check if events table exists
		var exists bool
		err = pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT FROM information_schema.tables 
				WHERE table_schema = 'public' 
				AND table_name = 'events'
			)
		`).Scan(&exists)
		if err != nil {
			return fmt.Errorf("failed to check events table: %w", err)
		}
		if !exists {
			return fmt.Errorf("events table does not exist")
		}

		// Verify table structure
		var count int
		err = pool.QueryRow(ctx, `
			SELECT COUNT(*) 
			FROM information_schema.columns 
			WHERE table_schema = 'public' 
			AND table_name = 'events'
		`).Scan(&count)
		if err != nil {
			return fmt.Errorf("failed to verify schema: %w", err)
		}
		if count == 0 {
			return fmt.Errorf("events table has no columns after schema application")
		}

		return nil
	}, 30*time.Second, 1*time.Second).Should(Succeed(), "Schema verification failed")

	// Initialize event store with retry
	Eventually(func() error {
		store, err = NewEventStore(ctx, pool)
		if err != nil {
			return fmt.Errorf("failed to create event store: %w", err)
		}
		return nil
	}, 10*time.Second, 1*time.Second).Should(Succeed(), "Event store initialization failed")

	setupTeardown()
})

var _ = AfterSuite(func() {
	if teardown != nil {
		teardown()
	}
})
