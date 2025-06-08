package dcb

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/testcontainers/testcontainers-go"
)

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

var _ = Describe("AppendEventsIfNotExists", func() {
	var (
		ctx       context.Context
		store     EventStore
		pool      *pgxpool.Pool
		postgresC testcontainers.Container
	)

	BeforeEach(func() {
		ctx = context.Background()
		var err error
		pool, postgresC, err = setupPostgresContainer(ctx)
		Expect(err).NotTo(HaveOccurred())

		store, err = NewEventStore(ctx, pool)
		Expect(err).NotTo(HaveOccurred())

		// Load schema
		schema, err := os.ReadFile("../../docker-entrypoint-initdb.d/schema.sql")
		Expect(err).NotTo(HaveOccurred())
		_, err = pool.Exec(ctx, string(schema))
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		if pool != nil {
			pool.Close()
		}
		if postgresC != nil {
			Expect(postgresC.Terminate(ctx)).To(Succeed())
		}
	})

	It("should append events when no matching events exist", func() {
		// Create test events
		events := []InputEvent{
			NewInputEvent("TestEvent", NewTags("test", "value"), []byte(`{"data": "test1"}`)),
			NewInputEvent("TestEvent", NewTags("test", "value"), []byte(`{"data": "test2"}`)),
		}

		// Create append condition that should not match any events
		condition := AppendCondition{
			FailIfEventsMatch: NewQuery(NewTags("nonexistent", "value")),
		}

		// Append events
		position, err := store.AppendEventsIfNotExists(ctx, events, condition)
		Expect(err).NotTo(HaveOccurred())
		Expect(position).To(Equal(int64(2))) // Should be the position of the last event

		// Verify events were appended
		rows, err := pool.Query(ctx, "SELECT type, tags, data FROM events ORDER BY position")
		Expect(err).NotTo(HaveOccurred())
		defer rows.Close()

		var count int
		for rows.Next() {
			var eventType string
			var tags, data []byte
			err := rows.Scan(&eventType, &tags, &data)
			Expect(err).NotTo(HaveOccurred())
			Expect(eventType).To(Equal("TestEvent"))
			count++
		}
		Expect(count).To(Equal(2))
	})

	It("should fail when matching events exist", func() {
		// First append some events
		firstEvents := []InputEvent{
			NewInputEvent("TestEvent", NewTags("test", "value"), []byte(`{"data": "test1"}`)),
		}
		firstCondition := AppendCondition{
			FailIfEventsMatch: NewQuery(NewTags("nonexistent", "value")),
		}
		_, err := store.AppendEventsIfNotExists(ctx, firstEvents, firstCondition)
		Expect(err).NotTo(HaveOccurred())

		// Try to append more events with a condition that matches existing events
		secondEvents := []InputEvent{
			NewInputEvent("TestEvent", NewTags("test", "value"), []byte(`{"data": "test2"}`)),
		}
		secondCondition := AppendCondition{
			FailIfEventsMatch: NewQuery(NewTags("test", "value")),
		}
		_, err = store.AppendEventsIfNotExists(ctx, secondEvents, secondCondition)
		Expect(err).To(HaveOccurred())
		Expect(err).To(BeAssignableToTypeOf(&ConcurrencyError{}))
	})

	It("should respect the After position in append condition", func() {
		// First append some events
		firstEvents := []InputEvent{
			NewInputEvent("TestEvent", NewTags("test", "value"), []byte(`{"data": "test1"}`)),
		}
		firstCondition := AppendCondition{
			FailIfEventsMatch: NewQuery(NewTags("nonexistent", "value")),
		}
		position, err := store.AppendEventsIfNotExists(ctx, firstEvents, firstCondition)
		Expect(err).NotTo(HaveOccurred())

		// Try to append more events with a condition that matches existing events but after the first position
		secondEvents := []InputEvent{
			NewInputEvent("TestEvent", NewTags("test", "value"), []byte(`{"data": "test2"}`)),
		}
		secondCondition := AppendCondition{
			FailIfEventsMatch: NewQuery(NewTags("test", "value")),
			After:             &position, // Only check for matches after the first event
		}
		_, err = store.AppendEventsIfNotExists(ctx, secondEvents, secondCondition)
		Expect(err).NotTo(HaveOccurred()) // Should succeed because we're only checking after the first event
	})

	It("should handle empty event types in append condition", func() {
		// Create test events
		events := []InputEvent{
			NewInputEvent("TestEvent", NewTags("test", "value"), []byte(`{"data": "test1"}`)),
		}

		// Create append condition with empty event types
		condition := AppendCondition{
			FailIfEventsMatch: Query{
				Tags:       NewTags("test", "value"),
				EventTypes: []string{}, // Empty event types should match any type
			},
		}

		// First append should succeed
		_, err := store.AppendEventsIfNotExists(ctx, events, condition)
		Expect(err).NotTo(HaveOccurred())

		// Second append should fail because events match the condition
		_, err = store.AppendEventsIfNotExists(ctx, events, condition)
		Expect(err).To(HaveOccurred())
		Expect(err).To(BeAssignableToTypeOf(&ConcurrencyError{}))
	})

	It("should handle nil After position in append condition", func() {
		// Create test events
		events := []InputEvent{
			NewInputEvent("TestEvent", NewTags("test", "value"), []byte(`{"data": "test1"}`)),
		}

		// Create append condition with nil After position
		condition := AppendCondition{
			FailIfEventsMatch: NewQuery(NewTags("test", "value")),
			After:             nil, // Should check all events
		}

		// First append should succeed
		_, err := store.AppendEventsIfNotExists(ctx, events, condition)
		Expect(err).NotTo(HaveOccurred())

		// Second append should fail because events match the condition
		_, err = store.AppendEventsIfNotExists(ctx, events, condition)
		Expect(err).To(HaveOccurred())
		Expect(err).To(BeAssignableToTypeOf(&ConcurrencyError{}))
	})

	It("should validate input events", func() {
		// Test empty events slice
		condition := AppendCondition{
			FailIfEventsMatch: NewQuery(NewTags("test", "value")),
		}
		_, err := store.AppendEventsIfNotExists(ctx, []InputEvent{}, condition)
		Expect(err).To(HaveOccurred())
		Expect(err).To(BeAssignableToTypeOf(&ValidationError{}))

		// Test events exceeding max batch size
		largeEvents := make([]InputEvent, 1001) // Max batch size is 1000
		for i := range largeEvents {
			largeEvents[i] = NewInputEvent("TestEvent", NewTags("test", "value"), []byte(`{"data": "test"}`))
		}
		_, err = store.AppendEventsIfNotExists(ctx, largeEvents, condition)
		Expect(err).To(HaveOccurred())
		Expect(err).To(BeAssignableToTypeOf(&ValidationError{}))
	})

	It("should maintain event relationships (causation and correlation)", func() {
		// Create test events
		events := []InputEvent{
			NewInputEvent("TestEvent1", NewTags("test", "value"), []byte(`{"data": "test1"}`)),
			NewInputEvent("TestEvent2", NewTags("test", "value"), []byte(`{"data": "test2"}`)),
		}

		condition := AppendCondition{
			FailIfEventsMatch: NewQuery(NewTags("nonexistent", "value")),
		}

		// Append events
		_, err := store.AppendEventsIfNotExists(ctx, events, condition)
		Expect(err).NotTo(HaveOccurred())

		// Verify event relationships
		rows, err := pool.Query(ctx, `
			SELECT id, causation_id, correlation_id 
			FROM events 
			ORDER BY position
		`)
		Expect(err).NotTo(HaveOccurred())
		defer rows.Close()

		var firstID, firstCausationID, firstCorrelationID string
		var secondID, secondCausationID, secondCorrelationID string

		Expect(rows.Next()).To(BeTrue())
		err = rows.Scan(&firstID, &firstCausationID, &firstCorrelationID)
		Expect(err).NotTo(HaveOccurred())

		Expect(rows.Next()).To(BeTrue())
		err = rows.Scan(&secondID, &secondCausationID, &secondCorrelationID)
		Expect(err).NotTo(HaveOccurred())

		// First event should be self-caused and self-correlated
		Expect(firstCausationID).To(Equal(firstID))
		Expect(firstCorrelationID).To(Equal(firstID))

		// Second event should be caused by first event and correlated with first event
		Expect(secondCausationID).To(Equal(firstID))
		Expect(secondCorrelationID).To(Equal(firstID))
	})
})
