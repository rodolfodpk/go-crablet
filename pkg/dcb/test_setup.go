package dcb

import (
	"context"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = BeforeSuite(func() {
	ctx = context.Background()

	// Initialize test database
	var err error
	pool, container, err = setupPostgresContainer(ctx)
	Expect(err).NotTo(HaveOccurred())

	// Read and execute schema.sql (path from pkg/dcb to root)
	schemaSQL, err := os.ReadFile("../../docker-entrypoint-initdb.d/schema.sql")
	Expect(err).NotTo(HaveOccurred())

	// Execute schema
	_, err = pool.Exec(ctx, string(schemaSQL))
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
