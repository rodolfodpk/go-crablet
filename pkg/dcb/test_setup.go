package dcb

import (
	"context"
	"fmt"
	"os"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = BeforeSuite(func() {
	ctx = context.Background()

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
