package dcb

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ReadEvents", func() {
	var (
		ctx   context.Context
		pool  *pgxpool.Pool
		store EventStore
	)

	BeforeEach(func() {
		ctx = context.Background()
		var err error

		Eventually(func() error {
			pool, _, err = setupPostgresContainer(ctx)
			if err != nil {
				return fmt.Errorf("failed to setup postgres container: %w", err)
			}
			return nil
		}, 30*time.Second, 1*time.Second).Should(Succeed(), "Failed to setup postgres container after multiple attempts")

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
		}, 10*time.Second, 100*time.Millisecond).Should(Succeed(), "Database connectivity test failed")

		store, err = NewEventStore(ctx, pool)
		Expect(err).NotTo(HaveOccurred())

		// Load schema
		schema, err := os.ReadFile("../../docker-entrypoint-initdb.d/schema.sql")
		Expect(err).NotTo(HaveOccurred())
		_, err = pool.Exec(ctx, string(schema))
		Expect(err).NotTo(HaveOccurred())

		// Truncate events table before each test
		err = truncateEventsTable(ctx, pool)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		if pool != nil {
			pool.Close()
		}
	})

	Describe("ReadEvents", func() {
		It("should read events with simple query", func() {
			// Append some test events
			events := []InputEvent{
				NewInputEvent("TestEvent", NewTags("test", "value1"), []byte(`{"data": "test1"}`)),
				NewInputEvent("TestEvent", NewTags("test", "value2"), []byte(`{"data": "test2"}`)),
				NewInputEvent("OtherEvent", NewTags("test", "value1"), []byte(`{"data": "other"}`)),
			}

			query := NewQuery(NewTags("test", "value1"))
			position, err := store.GetCurrentPosition(ctx, query)
			Expect(err).NotTo(HaveOccurred())

			_, err = store.AppendEvents(ctx, events, query, position)
			Expect(err).NotTo(HaveOccurred())

			// Read events
			iterator, err := store.ReadEvents(ctx, query, nil)
			Expect(err).NotTo(HaveOccurred())
			defer iterator.Close()

			// Read all events
			var readEvents []Event
			for {
				event, err := iterator.Next()
				if err != nil {
					break
				}
				if event == nil {
					break
				}
				readEvents = append(readEvents, *event)
			}

			Expect(readEvents).To(HaveLen(2))
			Expect(readEvents[0].Type).To(Equal("TestEvent"))
			Expect(readEvents[1].Type).To(Equal("OtherEvent"))
		})

		It("should read events with limit", func() {
			// Append some test events
			events := []InputEvent{
				NewInputEvent("TestEvent", NewTags("test", "value"), []byte(`{"data": "test1"}`)),
				NewInputEvent("TestEvent", NewTags("test", "value"), []byte(`{"data": "test2"}`)),
				NewInputEvent("TestEvent", NewTags("test", "value"), []byte(`{"data": "test3"}`)),
			}

			query := NewQuery(NewTags("test", "value"))
			position, err := store.GetCurrentPosition(ctx, query)
			Expect(err).NotTo(HaveOccurred())

			_, err = store.AppendEvents(ctx, events, query, position)
			Expect(err).NotTo(HaveOccurred())

			// Read events with limit
			options := &ReadOptions{
				Limit:   2,
				OrderBy: "asc",
			}
			iterator, err := store.ReadEvents(ctx, query, options)
			Expect(err).NotTo(HaveOccurred())
			defer iterator.Close()

			// Read events
			var readEvents []Event
			for {
				event, err := iterator.Next()
				if err != nil {
					break
				}
				if event == nil {
					break
				}
				readEvents = append(readEvents, *event)
			}

			Expect(readEvents).To(HaveLen(2))
		})

		It("should read events from specific position", func() {
			// Append some test events
			events := []InputEvent{
				NewInputEvent("TestEvent", NewTags("test", "value"), []byte(`{"data": "test1"}`)),
				NewInputEvent("TestEvent", NewTags("test", "value"), []byte(`{"data": "test2"}`)),
				NewInputEvent("TestEvent", NewTags("test", "value"), []byte(`{"data": "test3"}`)),
			}

			query := NewQuery(NewTags("test", "value"))
			position, err := store.GetCurrentPosition(ctx, query)
			Expect(err).NotTo(HaveOccurred())

			_, err = store.AppendEvents(ctx, events, query, position)
			Expect(err).NotTo(HaveOccurred())

			// Read events from position 2
			options := &ReadOptions{
				FromPosition: 2,
				OrderBy:      "asc",
			}
			iterator, err := store.ReadEvents(ctx, query, options)
			Expect(err).NotTo(HaveOccurred())
			defer iterator.Close()

			// Read events
			var readEvents []Event
			for {
				event, err := iterator.Next()
				if err != nil {
					break
				}
				if event == nil {
					break
				}
				readEvents = append(readEvents, *event)
			}

			Expect(readEvents).To(HaveLen(2))
			Expect(readEvents[0].Position).To(Equal(int64(2)))
			Expect(readEvents[1].Position).To(Equal(int64(3)))
		})

		It("should read events in descending order", func() {
			// Append some test events
			events := []InputEvent{
				NewInputEvent("TestEvent", NewTags("test", "value"), []byte(`{"data": "test1"}`)),
				NewInputEvent("TestEvent", NewTags("test", "value"), []byte(`{"data": "test2"}`)),
				NewInputEvent("TestEvent", NewTags("test", "value"), []byte(`{"data": "test3"}`)),
			}

			query := NewQuery(NewTags("test", "value"))
			position, err := store.GetCurrentPosition(ctx, query)
			Expect(err).NotTo(HaveOccurred())

			_, err = store.AppendEvents(ctx, events, query, position)
			Expect(err).NotTo(HaveOccurred())

			// Read events in descending order
			options := &ReadOptions{
				OrderBy: "desc",
			}
			iterator, err := store.ReadEvents(ctx, query, options)
			Expect(err).NotTo(HaveOccurred())
			defer iterator.Close()

			// Read events
			var readEvents []Event
			for {
				event, err := iterator.Next()
				if err != nil {
					break
				}
				if event == nil {
					break
				}
				readEvents = append(readEvents, *event)
			}

			Expect(readEvents).To(HaveLen(3))
			Expect(readEvents[0].Position).To(Equal(int64(3)))
			Expect(readEvents[1].Position).To(Equal(int64(2)))
			Expect(readEvents[2].Position).To(Equal(int64(1)))
		})

		It("should read all events with empty query", func() {
			// Append some test events
			events := []InputEvent{
				NewInputEvent("TestEvent", NewTags("test", "value1"), []byte(`{"data": "test1"}`)),
				NewInputEvent("OtherEvent", NewTags("test", "value2"), []byte(`{"data": "test2"}`)),
			}

			query := NewQuery(NewTags("test", "value1"))
			position, err := store.GetCurrentPosition(ctx, query)
			Expect(err).NotTo(HaveOccurred())

			_, err = store.AppendEvents(ctx, events, query, position)
			Expect(err).NotTo(HaveOccurred())

			// Read all events with empty query
			emptyQuery := Query{Items: []QueryItem{}}
			iterator, err := store.ReadEvents(ctx, emptyQuery, nil)
			Expect(err).NotTo(HaveOccurred())
			defer iterator.Close()

			// Read events
			var readEvents []Event
			for {
				event, err := iterator.Next()
				if err != nil {
					break
				}
				if event == nil {
					break
				}
				readEvents = append(readEvents, *event)
			}

			Expect(readEvents).To(HaveLen(2))
		})

		It("should handle complex query with multiple items", func() {
			// Append some test events
			events := []InputEvent{
				NewInputEvent("Event1", NewTags("type", "A"), []byte(`{"data": "A1"}`)),
				NewInputEvent("Event2", NewTags("type", "B"), []byte(`{"data": "B1"}`)),
				NewInputEvent("Event3", NewTags("type", "C"), []byte(`{"data": "C1"}`)),
			}

			query := NewQuery(NewTags("type", "A"))
			position, err := store.GetCurrentPosition(ctx, query)
			Expect(err).NotTo(HaveOccurred())

			_, err = store.AppendEvents(ctx, events, query, position)
			Expect(err).NotTo(HaveOccurred())

			// Create complex query with multiple items
			complexQuery := NewQueryFromItems(
				NewQueryItem([]string{"Event1"}, NewTags("type", "A")),
				NewQueryItem([]string{"Event2"}, NewTags("type", "B")),
			)

			iterator, err := store.ReadEvents(ctx, complexQuery, nil)
			Expect(err).NotTo(HaveOccurred())
			defer iterator.Close()

			// Read events
			var readEvents []Event
			for {
				event, err := iterator.Next()
				if err != nil {
					break
				}
				if event == nil {
					break
				}
				readEvents = append(readEvents, *event)
			}

			Expect(readEvents).To(HaveLen(2))
		})

		It("should validate options", func() {
			query := NewQuery(NewTags("test", "value"))

			// Test negative limit
			options := &ReadOptions{Limit: -1}
			_, err := store.ReadEvents(ctx, query, options)
			Expect(err).To(HaveOccurred())
			Expect(err).To(BeAssignableToTypeOf(&ValidationError{}))

			// Test invalid orderBy
			options = &ReadOptions{OrderBy: "invalid"}
			_, err = store.ReadEvents(ctx, query, options)
			Expect(err).To(HaveOccurred())
			Expect(err).To(BeAssignableToTypeOf(&ValidationError{}))
		})

		It("should handle iterator position tracking", func() {
			// Append some test events
			events := []InputEvent{
				NewInputEvent("TestEvent", NewTags("test", "value"), []byte(`{"data": "test1"}`)),
				NewInputEvent("TestEvent", NewTags("test", "value"), []byte(`{"data": "test2"}`)),
			}

			query := NewQuery(NewTags("test", "value"))
			position, err := store.GetCurrentPosition(ctx, query)
			Expect(err).NotTo(HaveOccurred())

			_, err = store.AppendEvents(ctx, events, query, position)
			Expect(err).NotTo(HaveOccurred())

			iterator, err := store.ReadEvents(ctx, query, nil)
			Expect(err).NotTo(HaveOccurred())
			defer iterator.Close()

			// Read first event
			event, err := iterator.Next()
			Expect(err).NotTo(HaveOccurred())
			Expect(event).NotTo(BeNil())
			Expect(iterator.Position()).To(Equal(int64(1)))

			// Read second event
			event, err = iterator.Next()
			Expect(err).NotTo(HaveOccurred())
			Expect(event).NotTo(BeNil())
			Expect(iterator.Position()).To(Equal(int64(2)))

			// No more events
			event, err = iterator.Next()
			Expect(err).NotTo(HaveOccurred())
			Expect(event).To(BeNil())
		})

		It("should handle closed iterator", func() {
			query := NewQuery(NewTags("test", "value"))
			iterator, err := store.ReadEvents(ctx, query, nil)
			Expect(err).NotTo(HaveOccurred())

			// Close iterator
			err = iterator.Close()
			Expect(err).NotTo(HaveOccurred())

			// Try to read from closed iterator
			_, err = iterator.Next()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("iterator is closed"))

			// Close again should not error
			err = iterator.Close()
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
