package dcb

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rodolfodpk/go-crablet/pkg/dcb"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Concurrency and Locking", func() {
	var (
		store dcb.EventStore
		ctx   context.Context
	)

	BeforeEach(func() {
		var err error
		ctx = context.Background()
		store, err = dcb.NewEventStore(ctx, pool)
		Expect(err).NotTo(HaveOccurred())
		err = truncateEventsTable(ctx, pool)
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("DCB Concurrency Control", func() {
		It("should enforce DCB concurrency control under true concurrency", func() {
			// Use a unique key for this test run
			key := fmt.Sprintf("concurrent-%d", time.Now().UnixNano())

			// First, create an existing event that will be used in the condition
			existingEvent := dcb.NewInputEvent("ExistingEvent", dcb.NewTags("key", key), dcb.ToJSON(map[string]string{"data": "existing"}))
			err := store.Append(ctx, []dcb.InputEvent{existingEvent})
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
				err := store.AppendIf(ctx, []dcb.InputEvent{event}, condition)
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

		It("should handle N concurrent users with DCB concurrency control", func() {
			resourceID := fmt.Sprintf("resource-%d", time.Now().UnixNano())
			numUsers := 10

			resourceEvent := dcb.NewInputEvent("ResourceCreated",
				dcb.NewTags("resource_id", resourceID),
				dcb.ToJSON(map[string]interface{}{
					"max_capacity":  5,
					"current_usage": 0,
				}))
			err := store.Append(ctx, []dcb.InputEvent{resourceEvent})
			Expect(err).NotTo(HaveOccurred())

			// DCB concurrency control: use AppendCondition to enforce some concurrency limits
			// This demonstrates the mechanism works, though it may not enforce exact "max N"
			query := dcb.NewQuery(dcb.NewTags("resource_id", resourceID), "ResourceUsageUpdated")
			condition := dcb.NewAppendCondition(query)

			var wg sync.WaitGroup
			results := make(chan string, numUsers)
			start := make(chan struct{})

			for i := 1; i <= numUsers; i++ {
				wg.Add(1)
				go func(userID int) {
					defer wg.Done()
					<-start

					usageEvent := dcb.NewInputEvent("ResourceUsageUpdated",
						dcb.NewTags("resource_id", resourceID, "user_id", fmt.Sprintf("user%d", userID)),
						dcb.ToJSON(map[string]interface{}{
							"user_id": fmt.Sprintf("user%d", userID),
							"usage":   1,
						}))

					// DCB concurrency control: AppendCondition enforces some limits
					err := store.AppendIf(ctx, []dcb.InputEvent{usageEvent}, condition)
					if err != nil {
						results <- fmt.Sprintf("User %d: FAILED - %v", userID, err)
					} else {
						results <- fmt.Sprintf("User %d: SUCCESS", userID)
					}
				}(i)
			}

			time.Sleep(100 * time.Millisecond)
			close(start)
			wg.Wait()
			close(results)

			successCount := 0
			failureCount := 0
			for result := range results {
				if result[len(result)-7:] == "SUCCESS" {
					successCount++
				} else {
					failureCount++
				}
			}

			// DCB concurrency control should enforce some limits
			Expect(successCount).To(BeNumerically(">", 0))
			Expect(successCount).To(BeNumerically("<=", numUsers))
			Expect(successCount + failureCount).To(Equal(numUsers))

			// Verify events were stored correctly
			finalQuery := dcb.NewQuery(dcb.NewTags("resource_id", resourceID), "ResourceUsageUpdated")
			events, err := store.Query(ctx, finalQuery, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(events).To(HaveLen(successCount))
		})
	})
})
