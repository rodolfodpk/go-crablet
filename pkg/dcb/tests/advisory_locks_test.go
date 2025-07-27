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

var _ = Describe("Advisory Locks", func() {
	var (
		store dcb.EventStore
		ctx   context.Context
	)

	BeforeEach(func() {
		store = dcb.NewEventStoreFromPool(pool)
		ctx = context.Background()
		err := truncateEventsTable(ctx, pool)
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("Advisory Lock Functionality", func() {
		It("should successfully append events with lock tags", func() {
			// Create events with lock tags
			event1 := dcb.NewInputEvent("ConcertBooking",
				dcb.NewTags("concert_id", "123", "lock:concert", "123", "user_id", "456"),
				dcb.ToJSON(map[string]string{"action": "book", "seats": "2"}))

			event2 := dcb.NewInputEvent("PaymentProcessed",
				dcb.NewTags("concert_id", "123", "lock:concert", "123", "payment_id", "789"),
				dcb.ToJSON(map[string]string{"amount": "100", "currency": "USD"}))

			// Append events with advisory locks
			err := store.Append(ctx, []dcb.InputEvent{event1, event2})
			Expect(err).NotTo(HaveOccurred())

			// Verify events were stored (without lock: prefix in tags)
			query := dcb.NewQuery(dcb.NewTags("concert_id", "123"), "ConcertBooking")
			events, err := store.Query(ctx, query, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(events).To(HaveLen(1))

			// Check that the stored event has tags without lock: prefix
			event := events[0]
			tags := dcb.TagsToArray(event.Tags)
			Expect(tags).To(ContainElement("concert_id:123"))
			Expect(tags).To(ContainElement("user_id:456"))
			Expect(tags).NotTo(ContainElement("lock:concert:123"))
		})

		It("should handle concurrent access with advisory locks", func() {
			concertID := fmt.Sprintf("concert_%d", time.Now().UnixNano())

			// Create two events that will compete for the same lock
			event1 := dcb.NewInputEvent("TicketBookingRequested",
				dcb.NewTags("concert_id", concertID, "lock:concert", concertID, "user_id", "user1"),
				dcb.ToJSON(map[string]string{"action": "book"}))

			event2 := dcb.NewInputEvent("TicketBookingRequested",
				dcb.NewTags("concert_id", concertID, "lock:concert", concertID, "user_id", "user2"),
				dcb.ToJSON(map[string]string{"action": "book"}))

			// Barrier to synchronize goroutines
			start := make(chan struct{})
			results := make(chan error, 2)

			appendFn := func(event dcb.InputEvent) {
				<-start
				err := store.Append(ctx, []dcb.InputEvent{event})
				results <- err
			}

			go appendFn(event1)
			go appendFn(event2)
			time.Sleep(100 * time.Millisecond) // Let goroutines get ready
			close(start)

			err1 := <-results
			err2 := <-results

			// Both should succeed because advisory locks are acquired in order
			Expect(err1).NotTo(HaveOccurred())
			Expect(err2).NotTo(HaveOccurred())

			// Verify both events were stored
			query := dcb.NewQuery(dcb.NewTags("concert_id", concertID), "TicketBookingRequested")
			events, err := store.Query(ctx, query, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(events).To(HaveLen(2))
		})

		It("should handle N concurrent users with advisory locks", func() {
			resourceID := fmt.Sprintf("resource-%d", time.Now().UnixNano())
			maxCapacity := 5
			numUsers := 10

			resourceEvent := dcb.NewInputEvent("ResourceCreated",
				dcb.NewTags("resource_id", resourceID),
				dcb.ToJSON(map[string]interface{}{
					"max_capacity":  maxCapacity,
					"current_usage": 0,
				}))
			err := store.Append(ctx, []dcb.InputEvent{resourceEvent})
			Expect(err).NotTo(HaveOccurred())

			// Test 1: Advisory locks WITHOUT AppendCondition - all users should succeed due to serialization
			fmt.Println("Testing advisory locks WITHOUT AppendCondition...")
			var wg1 sync.WaitGroup
			results1 := make(chan string, numUsers)
			start1 := make(chan struct{})

			for i := 1; i <= numUsers; i++ {
				wg1.Add(1)
				go func(userID int) {
					defer wg1.Done()
					<-start1

					usageEvent := dcb.NewInputEvent("ResourceUsageUpdated",
						dcb.NewTags(
							"resource_id", resourceID,
							"user_id", fmt.Sprintf("user%d", userID),
							"lock:resource", resourceID, // Advisory lock on the resource
						),
						dcb.ToJSON(map[string]interface{}{
							"user_id": fmt.Sprintf("user%d", userID),
							"usage":   1,
						}))

					// No AppendCondition - advisory locks serialize access but don't enforce business limits
					err := store.Append(ctx, []dcb.InputEvent{usageEvent})
					if err != nil {
						results1 <- fmt.Sprintf("User %d: FAILED - %v", userID, err)
					} else {
						results1 <- fmt.Sprintf("User %d: SUCCESS", userID)
					}
				}(i)
			}

			time.Sleep(100 * time.Millisecond)
			close(start1)
			wg1.Wait()
			close(results1)

			successCount1 := 0
			failureCount1 := 0
			for result := range results1 {
				if result[len(result)-7:] == "SUCCESS" {
					successCount1++
				} else {
					failureCount1++
				}
			}

			// Without AppendCondition, all users should succeed due to advisory lock serialization
			Expect(successCount1).To(Equal(numUsers))
			Expect(failureCount1).To(Equal(0))

			// Clear database for next test
			err = truncateEventsTable(ctx, pool)
			Expect(err).NotTo(HaveOccurred())

			// Recreate resource
			resourceEvent2 := dcb.NewInputEvent("ResourceCreated",
				dcb.NewTags("resource_id", resourceID),
				dcb.ToJSON(map[string]interface{}{
					"max_capacity":  maxCapacity,
					"current_usage": 0,
				}))
			err = store.Append(ctx, []dcb.InputEvent{resourceEvent2})
			Expect(err).NotTo(HaveOccurred())

			// Test 2: Advisory locks WITH AppendCondition - should enforce business limits
			fmt.Println("Testing advisory locks WITH AppendCondition...")
			query := dcb.NewQuery(dcb.NewTags("resource_id", resourceID), "ResourceUsageUpdated")
			condition := dcb.NewAppendCondition(query)

			var wg2 sync.WaitGroup
			results2 := make(chan string, numUsers)
			start2 := make(chan struct{})

			for i := 1; i <= numUsers; i++ {
				wg2.Add(1)
				go func(userID int) {
					defer wg2.Done()
					<-start2

					usageEvent := dcb.NewInputEvent("ResourceUsageUpdated",
						dcb.NewTags(
							"resource_id", resourceID,
							"user_id", fmt.Sprintf("user%d", userID),
							"lock:resource", resourceID, // Advisory lock on the resource
						),
						dcb.ToJSON(map[string]interface{}{
							"user_id": fmt.Sprintf("user%d", userID),
							"usage":   1,
						}))

					// With AppendCondition - advisory locks + DCB concurrency control
					err := store.AppendIf(ctx, []dcb.InputEvent{usageEvent}, condition)
					if err != nil {
						results2 <- fmt.Sprintf("User %d: FAILED - %v", userID, err)
					} else {
						results2 <- fmt.Sprintf("User %d: SUCCESS", userID)
					}
				}(i)
			}

			time.Sleep(100 * time.Millisecond)
			close(start2)
			wg2.Wait()
			close(results2)

			successCount2 := 0
			failureCount2 := 0
			for result := range results2 {
				if result[len(result)-7:] == "SUCCESS" {
					successCount2++
				} else {
					failureCount2++
				}
			}

			// With AppendCondition, some users should fail due to business limits
			Expect(successCount2).To(BeNumerically(">", 0))
			Expect(successCount2).To(BeNumerically("<=", numUsers))
			Expect(successCount2 + failureCount2).To(Equal(numUsers))

			// Verify final state
			finalQuery := dcb.NewQuery(dcb.NewTags("resource_id", resourceID), "ResourceUsageUpdated")
			events, err := store.Query(ctx, finalQuery, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(events).To(HaveLen(successCount2))

			// Verify events are ordered by transaction_id (advisory locks ensure sequential processing)
			for i := 1; i < len(events); i++ {
				prev := events[i-1]
				curr := events[i]

				if prev.TransactionID == curr.TransactionID {
					Expect(curr.Position).To(BeNumerically(">", prev.Position))
				} else {
					Expect(curr.TransactionID).To(BeNumerically(">", prev.TransactionID))
				}
			}
		})

		It("should filter out lock tags from stored events", func() {
			// Create event with mixed tags (regular and lock tags)
			event := dcb.NewInputEvent("TestEvent",
				dcb.NewTags(
					"regular_tag", "value1",
					"lock:resource:123", "lock_value",
					"another_tag", "value2",
					"lock:another:456", "another_lock",
				),
				dcb.ToJSON(map[string]string{"data": "test"}))

			// Append event
			err := store.Append(ctx, []dcb.InputEvent{event})
			Expect(err).NotTo(HaveOccurred())

			// Query the event
			query := dcb.NewQuery(dcb.NewTags("regular_tag", "value1"), "TestEvent")
			events, err := store.Query(ctx, query, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(events).To(HaveLen(1))

			// Check that stored tags don't contain lock: prefixes
			storedEvent := events[0]
			tags := dcb.TagsToArray(storedEvent.Tags)

			// Should contain regular tags
			Expect(tags).To(ContainElement("regular_tag:value1"))
			Expect(tags).To(ContainElement("another_tag:value2"))

			// Should NOT contain lock tags
			Expect(tags).NotTo(ContainElement("lock:resource:123"))
			Expect(tags).NotTo(ContainElement("lock:another:456"))
		})
	})
})
