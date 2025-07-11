package dcb_test

import (
	"context"
	"fmt"
	"time"

	"go-crablet/pkg/dcb"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Concurrency and Locking", func() {
	var (
		store dcb.EventStore
		ctx   context.Context
	)

	BeforeEach(func() {
		store = dcb.NewEventStoreFromPool(pool)
		ctx, _ = context.WithTimeout(context.Background(), 30*time.Second)
		err := truncateEventsTable(ctx, pool)
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("Optimistic Locking", func() {
		It("should enforce optimistic locking under true concurrency", func() {
			// Use a unique key for this test run
			key := fmt.Sprintf("concurrent-%d", time.Now().UnixNano())

			// First, create an existing event that will be used in the condition
			existingEvent := dcb.NewInputEvent("ExistingEvent", dcb.NewTags("key", key), dcb.ToJSON(map[string]string{"data": "existing"}))
			err := store.Append(ctx, []dcb.InputEvent{existingEvent}, nil)
			Expect(err).NotTo(HaveOccurred())

			// Create a condition that looks for this existing event
			query := dcb.NewQuery(dcb.NewTags("key", key), "ExistingEvent")
			condition := dcb.NewAppendCondition(query)

			// Create two different events to append - both should fail because the condition matches
			event1 := dcb.NewInputEvent("TestEvent1", dcb.NewTags("key", key), dcb.ToJSON(map[string]string{"data": "concurrent1"}))
			event2 := dcb.NewInputEvent("TestEvent2", dcb.NewTags("key", key), dcb.ToJSON(map[string]string{"data": "concurrent2"}))

			// Barrier to synchronize goroutines
			start := make(chan struct{})
			results := make(chan error, 2)

			appendFn := func(event dcb.InputEvent) {
				<-start
				err := store.Append(ctx, []dcb.InputEvent{event}, &condition)
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
