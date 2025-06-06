package dcb

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/testcontainers/testcontainers-go"
	"io"
	"os"
	"testing"
	"time"
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

		if store != nil {
			store.Close()
		}
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

	pool, postgresC, err = setupPostgresContainer(ctx)
	Expect(err).NotTo(HaveOccurred())

	Eventually(func() error {
		return pool.Ping(ctx)
	}, 10*time.Second, 200*time.Millisecond).Should(Succeed())

	// Use go:embed or another more robust path resolution approach
	projectRoot := "../.." // Go up two levels from internal/dcb to the project root
	schemaPath := projectRoot + "/docker-entrypoint-initdb.d/schema.sql"
	schema, err := os.ReadFile(schemaPath)
	Expect(err).NotTo(HaveOccurred())

	_, err = pool.Exec(ctx, string(schema))
	Expect(err).NotTo(HaveOccurred())

	store, err = NewEventStore(ctx, pool)
	Expect(err).NotTo(HaveOccurred())
	setupTeardown()
})

var _ = AfterSuite(func() {
	if teardown != nil {
		teardown()
	}
})
