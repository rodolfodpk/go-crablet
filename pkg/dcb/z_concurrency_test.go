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

			// Create a condition that looks for a specific event type that doesn't exist yet
			query := NewQuery(NewTags("key", key), "ConcurrentEvent")
			condition := NewAppendCondition(query)

			// Create two different events to append - both should succeed initially
			event1 := NewInputEvent("ConcurrentEvent", NewTags("key", key), toJSON(map[string]string{"data": "concurrent1"}))
			event2 := NewInputEvent("ConcurrentEvent", NewTags("key", key), toJSON(map[string]string{"data": "concurrent2"}))

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

			// One should succeed, one should fail with concurrency error
			// Both are trying to append the same event type with the same condition
			// The first one will succeed (no existing events), the second will fail
			if err1 == nil {
				Expect(err2).To(HaveOccurred())
				Expect(err2.Error()).To(SatisfyAny(
					ContainSubstring("append condition violated"),
					ContainSubstring("SQLSTATE 40001"),
				))
			} else {
				Expect(err1.Error()).To(SatisfyAny(
					ContainSubstring("append condition violated"),
					ContainSubstring("SQLSTATE 40001"),
				))
				Expect(err2).To(BeNil())
			}
		})
	})
})
