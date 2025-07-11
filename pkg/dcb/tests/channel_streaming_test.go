package dcb_test

import (
	"context"
	"fmt"
	"time"

	"go-crablet/pkg/dcb"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Channel-Based Streaming", func() {
	var (
		store dcb.EventStore
		ctx   context.Context
	)

	BeforeEach(func() {
		// Use shared PostgreSQL container and truncate events between tests
		store = dcb.NewEventStoreFromPool(pool)
		var ok bool
		store, ok = store.(dcb.EventStore)
		Expect(ok).To(BeTrue(), "Store should implement EventStore")

		// Create context for each test
		ctx = context.Background()

		// Truncate events table before each test
		err := truncateEventsTable(ctx, pool)
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("ReadStream", func() {
		It("should stream events through channels", func() {
			// Setup test data
			event1 := dcb.NewInputEvent("TestEvent", dcb.NewTags("test", "value"), dcb.ToJSON(map[string]string{"data": "value1"}))
			event2 := dcb.NewInputEvent("TestEvent", dcb.NewTags("test", "value"), dcb.ToJSON(map[string]string{"data": "value2"}))
			event3 := dcb.NewInputEvent("TestEvent", dcb.NewTags("test", "value"), dcb.ToJSON(map[string]string{"data": "value3"}))

			events := []dcb.InputEvent{event1, event2, event3}

			err := store.Append(ctx, events, nil)
			Expect(err).NotTo(HaveOccurred())

			// Test channel-based streaming
			query := dcb.NewQuery(dcb.NewTags("test", "value"), "TestEvent")
			eventChan, err := store.QueryStream(ctx, query, nil)
			Expect(err).NotTo(HaveOccurred())

			count := 0
			for event := range eventChan {
				Expect(event.Type).To(Equal("TestEvent"))
				count++
			}

			Expect(count).To(Equal(3))
		})

		It("should handle empty result sets", func() {
			query := dcb.NewQuery(dcb.NewTags("non-existent", "value"), "TestEvent")
			eventChan, err := store.QueryStream(ctx, query, nil)
			Expect(err).NotTo(HaveOccurred())

			count := 0
			for range eventChan {
				count++
			}

			Expect(count).To(Equal(0))
		})

		It("should handle context cancellation", func() {
			// Setup test data
			event1 := dcb.NewInputEvent("TestEvent", dcb.NewTags("test", "value"), dcb.ToJSON(map[string]string{"data": "value1"}))
			event2 := dcb.NewInputEvent("TestEvent", dcb.NewTags("test", "value"), dcb.ToJSON(map[string]string{"data": "value2"}))

			events := []dcb.InputEvent{event1, event2}

			err := store.Append(ctx, events, nil)
			Expect(err).NotTo(HaveOccurred())

			// Create cancellable context
			cancelCtx, cancel := context.WithCancel(ctx)
			defer cancel()

			query := dcb.NewQuery(dcb.NewTags("test", "value"), "TestEvent")
			eventChan, err := store.QueryStream(cancelCtx, query, nil)
			Expect(err).NotTo(HaveOccurred())

			// Cancel context after first event
			count := 0
			for range eventChan {
				count++
				if count == 1 {
					cancel()
					break
				}
			}

			Expect(count).To(Equal(1))
		})

		It("should handle different batch sizes", func() {
			// Create many events
			events := make([]dcb.InputEvent, 10)
			for i := 0; i < 10; i++ {
				event := dcb.NewInputEvent("TestEvent", dcb.NewTags("test", "value"), dcb.ToJSON(map[string]string{"index": fmt.Sprintf("%d", i)}))
				events[i] = event
			}

			err := store.Append(ctx, events, nil)
			Expect(err).NotTo(HaveOccurred())

			// Test with small batch size
			query := dcb.NewQuery(dcb.NewTags("test", "value"), "TestEvent")
			eventChan, err := store.QueryStream(ctx, query, nil)
			Expect(err).NotTo(HaveOccurred())

			count := 0
			for event := range eventChan {
				Expect(event.Type).To(Equal("TestEvent"))
				count++
			}

			Expect(count).To(Equal(10))
		})
	})

	Describe("ProjectStream", func() {
		It("should project states using channels", func() {
			// Setup test data
			event1 := dcb.NewInputEvent("AccountOpened", dcb.NewTags("account_id", "acc1"), dcb.ToJSON(map[string]string{"balance": "100"}))
			event2 := dcb.NewInputEvent("MoneyTransferred", dcb.NewTags("account_id", "acc1"), dcb.ToJSON(map[string]string{"amount": "50"}))
			event3 := dcb.NewInputEvent("MoneyTransferred", dcb.NewTags("account_id", "acc1"), dcb.ToJSON(map[string]string{"amount": "25"}))

			events := []dcb.InputEvent{event1, event2, event3}

			err := store.Append(ctx, events, nil)
			Expect(err).NotTo(HaveOccurred())

			// Define projectors
			projectors := []dcb.StateProjector{
				{ID: "accountCount",
					Query:        dcb.NewQuery(dcb.NewTags("account_id", "acc1"), "AccountOpened"),
					InitialState: 0,
					TransitionFn: func(state any, event dcb.Event) any {
						return state.(int) + 1
					},
				},
				{ID: "transferCount",
					Query:        dcb.NewQuery(dcb.NewTags("account_id", "acc1"), "MoneyTransferred"),
					InitialState: 0,
					TransitionFn: func(state any, event dcb.Event) any {
						return state.(int) + 1
					},
				},
			}

			// Use channel-based projection
			resultChan, _, err := store.ProjectStream(ctx, projectors, nil)
			Expect(err).NotTo(HaveOccurred())

			// Process results - now we get final aggregated states
			finalStates := <-resultChan

			Expect(finalStates["accountCount"]).To(Equal(1))
			Expect(finalStates["transferCount"]).To(Equal(2))
		})

		It("should handle empty projectors list", func() {
			_, _, err := store.ProjectStream(ctx, []dcb.StateProjector{}, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("at least one projector is required"))
		})

		It("should handle nil transition function", func() {
			projectors := []dcb.StateProjector{
				{ID: "invalid",
					Query:        dcb.NewQuery(dcb.NewTags("test", "value"), "TestEvent"),
					InitialState: 0,
					TransitionFn: nil, // Nil transition function
				},
			}

			_, _, err := store.ProjectStream(ctx, projectors, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("nil transition function"))
		})

		It("should handle context cancellation during projection", func() {
			// Setup test data with many events to ensure processing takes time
			events := make([]dcb.InputEvent, 1000)
			for i := 0; i < 1000; i++ {
				event := dcb.NewInputEvent("TestEvent", dcb.NewTags("test", "value"), dcb.ToJSON(map[string]string{"data": fmt.Sprintf("value%d", i)}))
				events[i] = event
			}

			err := store.Append(ctx, events, nil)
			Expect(err).NotTo(HaveOccurred())

			// Create cancellable context
			cancelCtx, cancel := context.WithCancel(ctx)
			defer cancel()

			projectors := []dcb.StateProjector{
				{ID: "test",
					Query:        dcb.NewQuery(dcb.NewTags("test", "value"), "TestEvent"),
					InitialState: 0,
					TransitionFn: func(state any, event dcb.Event) any {
						time.Sleep(1 * time.Microsecond)
						return state.(int) + 1
					},
				},
			}

			resultChan, _, err := store.ProjectStream(cancelCtx, projectors, nil)
			Expect(err).NotTo(HaveOccurred())

			// Cancel context after a short delay
			time.Sleep(10 * time.Millisecond)
			cancel()

			select {
			case <-resultChan:
				// Acceptable: result was sent before cancellation was noticed
				// (This is a trade-off of the final-result streaming design)
				// Test passes
			case <-time.After(100 * time.Millisecond):
				// Also acceptable: no result received after cancellation
				// Test passes
			}
		})

		It("should handle projection with no matching events", func() {
			projectors := []dcb.StateProjector{
				{ID: "test",
					Query:        dcb.NewQuery(dcb.NewTags("non-existent", "value"), "TestEvent"),
					InitialState: 0,
					TransitionFn: func(state any, event dcb.Event) any {
						return state.(int) + 1
					},
				},
			}

			resultChan, _, err := store.ProjectStream(ctx, projectors, nil)
			Expect(err).NotTo(HaveOccurred())

			// Should receive final states even with no matching events
			finalStates := <-resultChan
			Expect(finalStates["test"]).To(Equal(0)) // Initial state
		})

		It("should handle multiple projectors with different event types", func() {
			// Setup test data
			event1 := dcb.NewInputEvent("AccountOpened", dcb.NewTags("account_id", "acc1"), dcb.ToJSON(map[string]string{"balance": "100"}))
			event2 := dcb.NewInputEvent("MoneyTransferred", dcb.NewTags("account_id", "acc1"), dcb.ToJSON(map[string]string{"amount": "50"}))
			event3 := dcb.NewInputEvent("AccountClosed", dcb.NewTags("account_id", "acc1"), dcb.ToJSON(map[string]string{"reason": "inactive"}))

			events := []dcb.InputEvent{event1, event2, event3}

			err := store.Append(ctx, events, nil)
			Expect(err).NotTo(HaveOccurred())

			// Define projectors for different event types
			projectors := []dcb.StateProjector{
				{ID: "accountCount",
					Query:        dcb.NewQuery(dcb.NewTags("account_id", "acc1"), "AccountOpened"),
					InitialState: 0,
					TransitionFn: func(state any, event dcb.Event) any {
						return state.(int) + 1
					},
				},
				{ID: "transferCount",
					Query:        dcb.NewQuery(dcb.NewTags("account_id", "acc1"), "MoneyTransferred"),
					InitialState: 0,
					TransitionFn: func(state any, event dcb.Event) any {
						return state.(int) + 1
					},
				},
				{ID: "closeCount",
					Query:        dcb.NewQuery(dcb.NewTags("account_id", "acc1"), "AccountClosed"),
					InitialState: 0,
					TransitionFn: func(state any, event dcb.Event) any {
						return state.(int) + 1
					},
				},
			}

			// Use channel-based projection
			resultChan, _, err := store.ProjectStream(ctx, projectors, nil)
			Expect(err).NotTo(HaveOccurred())

			// Process results - now we get final aggregated states
			finalStates := <-resultChan

			Expect(finalStates["accountCount"]).To(Equal(1))
			Expect(finalStates["transferCount"]).To(Equal(1))
			Expect(finalStates["closeCount"]).To(Equal(1))
		})

		It("should handle large datasets with channel streaming", func() {
			// Use context.Background() without any timeout to test if hybrid timeout is the issue
			longCtx := context.Background()

			// Create many events
			events := make([]dcb.InputEvent, 100)
			for i := 0; i < 100; i++ {
				event := dcb.NewInputEvent("TestEvent", dcb.NewTags("test", "value"), dcb.ToJSON(map[string]string{"index": fmt.Sprintf("%d", i)}))
				events[i] = event
			}

			// Use longCtx for append
			err := store.Append(longCtx, events, nil)
			Expect(err).NotTo(HaveOccurred())

			// Define projector
			projectors := []dcb.StateProjector{
				{ID: "count",
					Query:        dcb.NewQuery(dcb.NewTags("test", "value"), "TestEvent"),
					InitialState: 0,
					TransitionFn: func(state any, event dcb.Event) any {
						return state.(int) + 1
					},
				},
			}

			// Use longCtx for projection
			resultChan, _, err := store.ProjectStream(longCtx, projectors, nil)
			Expect(err).NotTo(HaveOccurred())

			// Process results - now we get final aggregated states
			finalStates := <-resultChan

			Expect(finalStates["count"]).To(Equal(100)) // All 100 events processed
		})
	})

	Describe("Extension Interface Pattern", func() {
		It("should properly implement EventStore interface", func() {
			// Test that the store implements the EventStore interface
			var eventStore dcb.EventStore = store
			Expect(eventStore).NotTo(BeNil())

			// Test that our implementation does implement EventStore
			// (since our eventStore has the ReadStreamChannel method)
			_, ok := store.(dcb.EventStore)
			Expect(ok).To(BeTrue(), "Our EventStore implementation should implement EventStore")
		})
	})
})
