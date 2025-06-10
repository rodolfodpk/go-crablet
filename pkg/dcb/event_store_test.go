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

var _ = Describe("GetCurrentPosition", func() {
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

	It("should return 0 for empty database", func() {
		query := NewQuery([]Tag{{Key: "test", Value: "value"}}, "TestEvent")
		position, err := store.GetCurrentPosition(ctx, query)
		Expect(err).NotTo(HaveOccurred())
		Expect(position).To(Equal(int64(0)))
	})

	It("should return latest position for matching events", func() {
		// Append some events
		events := []InputEvent{
			NewInputEvent("TestEvent", NewTags("test", "value"), []byte(`{"data": "test1"}`)),
			NewInputEvent("TestEvent", NewTags("test", "value"), []byte(`{"data": "test2"}`)),
			NewInputEvent("OtherEvent", NewTags("test", "value"), []byte(`{"data": "other"}`)),
		}

		query := NewQuery([]Tag{{Key: "test", Value: "value"}}, "TestEvent")
		_, err := store.AppendEvents(ctx, events, query, 0)
		Expect(err).NotTo(HaveOccurred())

		// Get current position
		position, err := store.GetCurrentPosition(ctx, query)
		Expect(err).NotTo(HaveOccurred())
		Expect(position).To(Equal(int64(2))) // Should return position of last TestEvent
	})

	It("should filter by event types", func() {
		// Append events with different types
		events := []InputEvent{
			NewInputEvent("TypeA", NewTags("category", "test"), []byte(`{"data": "a"}`)),
			NewInputEvent("TypeB", NewTags("category", "test"), []byte(`{"data": "b"}`)),
			NewInputEvent("TypeC", NewTags("category", "test"), []byte(`{"data": "c"}`)),
		}

		query := NewQuery([]Tag{{Key: "category", Value: "test"}}, "TypeA", "TypeB")
		_, err := store.AppendEvents(ctx, events, query, 0)
		Expect(err).NotTo(HaveOccurred())

		// Get current position for TypeA and TypeB only
		position, err := store.GetCurrentPosition(ctx, query)
		Expect(err).NotTo(HaveOccurred())
		Expect(position).To(Equal(int64(2))) // Should return position of last TypeB event
	})

	It("should handle multiple tags", func() {
		// Append events with multiple tags
		events := []InputEvent{
			NewInputEvent("MultiTag", []Tag{
				{Key: "category", Value: "test"},
				{Key: "priority", Value: "high"},
			}, []byte(`{"data": "high"}`)),
			NewInputEvent("MultiTag", []Tag{
				{Key: "category", Value: "test"},
				{Key: "priority", Value: "low"},
			}, []byte(`{"data": "low"}`)),
		}

		query := NewQuery([]Tag{
			{Key: "category", Value: "test"},
			{Key: "priority", Value: "high"},
		}, "MultiTag")
		_, err := store.AppendEvents(ctx, events, query, 0)
		Expect(err).NotTo(HaveOccurred())

		// Get current position for high priority events
		position, err := store.GetCurrentPosition(ctx, query)
		Expect(err).NotTo(HaveOccurred())
		Expect(position).To(Equal(int64(1))) // Should return position of high priority event
	})

	It("should handle empty event types", func() {
		// Append events
		events := []InputEvent{
			NewInputEvent("AnyType", NewTags("test", "value"), []byte(`{"data": "test"}`)),
		}

		query := NewQuery([]Tag{{Key: "test", Value: "value"}}) // No event types specified
		_, err := store.AppendEvents(ctx, events, query, 0)
		Expect(err).NotTo(HaveOccurred())

		// Get current position
		position, err := store.GetCurrentPosition(ctx, query)
		Expect(err).NotTo(HaveOccurred())
		Expect(position).To(Equal(int64(1)))
	})
})

var _ = Describe("AppendEventsIf", func() {
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

		// Create append condition
		condition := AppendCondition{
			FailIfEventsMatch: NewQuery([]Tag{{Key: "test", Value: "value"}}, "TestEvent"),
			After:             nil, // Check all events
		}

		// Append events
		position, err := store.AppendEventsIf(ctx, events, condition)
		Expect(err).NotTo(HaveOccurred())
		Expect(position).To(Equal(int64(2)))

		// Verify events were appended by checking the database directly
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
		// First, append some events
		events1 := []InputEvent{
			NewInputEvent("TestEvent", NewTags("test", "value"), []byte(`{"data": "existing"}`)),
		}

		condition1 := AppendCondition{
			FailIfEventsMatch: NewQuery([]Tag{{Key: "test", Value: "value"}}, "TestEvent"),
			After:             nil,
		}

		_, err := store.AppendEventsIf(ctx, events1, condition1)
		Expect(err).NotTo(HaveOccurred())

		// Now try to append more events with the same condition
		events2 := []InputEvent{
			NewInputEvent("TestEvent", NewTags("test", "value"), []byte(`{"data": "new"}`)),
		}

		condition2 := AppendCondition{
			FailIfEventsMatch: NewQuery([]Tag{{Key: "test", Value: "value"}}, "TestEvent"),
			After:             nil,
		}

		_, err = store.AppendEventsIf(ctx, events2, condition2)
		Expect(err).To(HaveOccurred())
		Expect(err).To(BeAssignableToTypeOf(&ConcurrencyError{}))
	})

	It("should succeed when no matching events exist after specified position", func() {
		// First, append some events
		events1 := []InputEvent{
			NewInputEvent("TestEvent", NewTags("test", "value"), []byte(`{"data": "existing"}`)),
		}

		condition1 := AppendCondition{
			FailIfEventsMatch: NewQuery([]Tag{{Key: "test", Value: "value"}}, "TestEvent"),
			After:             nil,
		}

		position1, err := store.AppendEventsIf(ctx, events1, condition1)
		Expect(err).NotTo(HaveOccurred())

		// Now append more events, but only fail if matching events exist after position1
		events2 := []InputEvent{
			NewInputEvent("TestEvent", NewTags("test", "value"), []byte(`{"data": "new"}`)),
		}

		condition2 := AppendCondition{
			FailIfEventsMatch: NewQuery([]Tag{{Key: "test", Value: "value"}}, "TestEvent"),
			After:             &position1,
		}

		position2, err := store.AppendEventsIf(ctx, events2, condition2)
		Expect(err).NotTo(HaveOccurred())
		Expect(position2).To(Equal(int64(2)))
	})

	It("should handle empty event types in condition", func() {
		// Create test events
		events := []InputEvent{
			NewInputEvent("TestEvent", NewTags("test", "value"), []byte(`{"data": "test"}`)),
		}

		// Create append condition with empty event types
		condition := AppendCondition{
			FailIfEventsMatch: NewQuery([]Tag{{Key: "test", Value: "value"}}), // No event types specified
			After:             nil,
		}

		// Append events
		position, err := store.AppendEventsIf(ctx, events, condition)
		Expect(err).NotTo(HaveOccurred())
		Expect(position).To(Equal(int64(1)))
	})

	It("should handle multiple tags in condition", func() {
		// Create test events
		events := []InputEvent{
			NewInputEvent("TestEvent", []Tag{
				{Key: "test", Value: "value"},
				{Key: "category", Value: "important"},
			}, []byte(`{"data": "test"}`)),
		}

		// Create append condition with multiple tags
		condition := AppendCondition{
			FailIfEventsMatch: NewQuery([]Tag{
				{Key: "test", Value: "value"},
				{Key: "category", Value: "important"},
			}, "TestEvent"),
			After: nil,
		}

		// Append events
		position, err := store.AppendEventsIf(ctx, events, condition)
		Expect(err).NotTo(HaveOccurred())
		Expect(position).To(Equal(int64(1)))
	})

	It("should fail with empty events slice", func() {
		condition := AppendCondition{
			FailIfEventsMatch: NewQuery([]Tag{{Key: "test", Value: "value"}}, "TestEvent"),
			After:             nil,
		}

		_, err := store.AppendEventsIf(ctx, []InputEvent{}, condition)
		Expect(err).To(HaveOccurred())
		Expect(err).To(BeAssignableToTypeOf(&ValidationError{}))
	})

	It("should fail with events exceeding max batch size", func() {
		// Create events exceeding max batch size (1000)
		largeEvents := make([]InputEvent, 1001)
		for i := range largeEvents {
			largeEvents[i] = NewInputEvent("TestEvent", NewTags("test", "value"), []byte(`{"data": "test"}`))
		}

		condition := AppendCondition{
			FailIfEventsMatch: NewQuery([]Tag{{Key: "test", Value: "value"}}, "TestEvent"),
			After:             nil,
		}

		_, err := store.AppendEventsIf(ctx, largeEvents, condition)
		Expect(err).To(HaveOccurred())
		Expect(err).To(BeAssignableToTypeOf(&ValidationError{}))
	})

	It("should fail with invalid event data", func() {
		// Create event with invalid JSON data
		events := []InputEvent{
			NewInputEvent("TestEvent", NewTags("test", "value"), []byte(`invalid json`)),
		}

		condition := AppendCondition{
			FailIfEventsMatch: NewQuery([]Tag{{Key: "test", Value: "value"}}, "TestEvent"),
			After:             nil,
		}

		_, err := store.AppendEventsIf(ctx, events, condition)
		Expect(err).To(HaveOccurred())
		Expect(err).To(BeAssignableToTypeOf(&ValidationError{}))
	})

	It("should fail with empty event type", func() {
		// Create event with empty type
		events := []InputEvent{
			NewInputEvent("", NewTags("test", "value"), []byte(`{"data": "test"}`)),
		}

		condition := AppendCondition{
			FailIfEventsMatch: NewQuery([]Tag{{Key: "test", Value: "value"}}, "TestEvent"),
			After:             nil,
		}

		_, err := store.AppendEventsIf(ctx, events, condition)
		Expect(err).To(HaveOccurred())
		Expect(err).To(BeAssignableToTypeOf(&ValidationError{}))
	})

	It("should fail with empty tags", func() {
		// Create event with empty tags
		events := []InputEvent{
			NewInputEvent("TestEvent", []Tag{}, []byte{}),
		}

		condition := AppendCondition{
			FailIfEventsMatch: NewQuery([]Tag{{Key: "test", Value: "value"}}, "TestEvent"),
			After:             nil,
		}

		_, err := store.AppendEventsIf(ctx, events, condition)
		Expect(err).To(HaveOccurred())
		Expect(err).To(BeAssignableToTypeOf(&ValidationError{}))
	})

	It("should fail with invalid query tags", func() {
		// Create valid events
		events := []InputEvent{
			NewInputEvent("TestEvent", NewTags("test", "value"), []byte(`{"data": "test"}`)),
		}

		// Create condition with invalid query (empty key)
		condition := AppendCondition{
			FailIfEventsMatch: NewLegacyQuery([]Tag{{Key: "", Value: "value"}}, []string{}),
			After:             nil,
		}

		_, err := store.AppendEventsIf(ctx, events, condition)
		Expect(err).To(HaveOccurred())
		Expect(err).To(BeAssignableToTypeOf(&ValidationError{}))
	})
})

var _ = Describe("Helper Functions", func() {
	Describe("NewTags", func() {
		It("should create empty tags slice", func() {
			tags := NewTags()
			Expect(tags).NotTo(BeNil())
			Expect(tags).To(HaveLen(0))
		})

		It("should create single tag", func() {
			tags := NewTags("key", "value")
			Expect(tags).To(HaveLen(1))
			Expect(tags[0].Key).To(Equal("key"))
			Expect(tags[0].Value).To(Equal("value"))
		})

		It("should create multiple tags", func() {
			tags := NewTags("key1", "value1", "key2", "value2")
			Expect(tags).To(HaveLen(2))
			Expect(tags[0].Key).To(Equal("key1"))
			Expect(tags[0].Value).To(Equal("value1"))
			Expect(tags[1].Key).To(Equal("key2"))
			Expect(tags[1].Value).To(Equal("value2"))
		})

		It("should handle odd number of arguments", func() {
			Expect(func() {
				NewTags("key1", "value1", "key2")
			}).To(PanicWith("NewTags: odd number of arguments"))
		})
	})

	Describe("NewQuery", func() {
		It("should create query with tags only", func() {
			NewQuery([]Tag{{Key: "test", Value: "value"}})
		})

		It("should create query with tags and event types", func() {
			NewQuery([]Tag{{Key: "test", Value: "value"}}, "Event1", "Event2")
		})

		It("should create query with empty tags", func() {
			NewQuery([]Tag{})
		})
	})

	Describe("NewInputEvent", func() {
		It("should create input event", func() {
			event := NewInputEvent("TestEvent", NewTags("test", "value"), []byte(`{"data": "test"}`))
			Expect(event.Type).To(Equal("TestEvent"))
			Expect(event.Tags).To(HaveLen(1))
			Expect(event.Tags[0].Key).To(Equal("test"))
			Expect(event.Tags[0].Value).To(Equal("value"))
			Expect(event.Data).To(Equal([]byte(`{"data": "test"}`)))
		})
	})
})
