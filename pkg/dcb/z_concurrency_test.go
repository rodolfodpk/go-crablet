package dcb

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Concurrency and Locking", func() {
	var (
		store EventStore
		ctx   context.Context
	)

	BeforeEach(func() {
		// Use shared PostgreSQL container and truncate events between tests
		store = NewEventStoreFromPool(pool)

		// Create context with timeout for each test
		ctx, _ = context.WithTimeout(context.Background(), 30*time.Second)

		// Truncate events table before each test
		err := truncateEventsTable(ctx, pool)
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("Optimistic Locking", func() {
		It("should enforce optimistic locking under true concurrency", func() {
			// Use a unique key for this test run
			key := fmt.Sprintf("concurrent-%d", time.Now().UnixNano())

			// First, create an existing event that will be used in the condition
			existingEvent := NewInputEvent("ExistingEvent", NewTags("key", key), toJSON(map[string]string{"data": "existing"}))
			err := store.Append(ctx, []InputEvent{existingEvent})
			Expect(err).NotTo(HaveOccurred())

			// Create a condition that looks for this existing event
			query := NewQuery(NewTags("key", key), "ExistingEvent")
			condition := NewAppendCondition(query)

			// Create two different events to append - both should fail because the condition matches
			event1 := NewInputEvent("TestEvent1", NewTags("key", key), toJSON(map[string]string{"data": "concurrent1"}))
			event2 := NewInputEvent("TestEvent2", NewTags("key", key), toJSON(map[string]string{"data": "concurrent2"}))

			// Barrier to synchronize goroutines
			start := make(chan struct{})
			results := make(chan error, 2)

			appendFn := func(event InputEvent) {
				<-start
				err := store.AppendIf(ctx, []InputEvent{event}, condition)
				results <- err
			}

			go appendFn(event1)
			go appendFn(event2)
			time.Sleep(100 * time.Millisecond) // Let goroutines get ready
			close(start)

			err1 := <-results
			err2 := <-results

			// Both should fail with concurrency error because the condition matches an existing event
			Expect(err1).To(HaveOccurred())
			Expect(err1.Error()).To(ContainSubstring("append condition violated"))
			Expect(err2).To(HaveOccurred())
			Expect(err2.Error()).To(ContainSubstring("append condition violated"))
		})
	})
})
