package dcb

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Channel-Based Streaming", func() {
	var (
		store        EventStore
		channelStore ChannelEventStore
		ctx          context.Context
	)

	BeforeEach(func() {
		// Use shared PostgreSQL container and truncate events between tests
		store = NewEventStoreFromPool(pool)
		var ok bool
		channelStore, ok = store.(ChannelEventStore)
		Expect(ok).To(BeTrue(), "Store should implement ChannelEventStore")

		// Create context with timeout for each test
		ctx, _ = context.WithTimeout(context.Background(), 30*time.Second)

		// Truncate events table before each test
		err := truncateEventsTable(ctx, pool)
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("ReadStreamChannel", func() {
		It("should stream events through channels", func() {
			// Setup test data
			event1 := NewInputEvent("TestEvent", NewTags("test", "value"), toJSON(map[string]string{"data": "value1"}))
			event2 := NewInputEvent("TestEvent", NewTags("test", "value"), toJSON(map[string]string{"data": "value2"}))
			event3 := NewInputEvent("TestEvent", NewTags("test", "value"), toJSON(map[string]string{"data": "value3"}))

			events := []InputEvent{event1, event2, event3}

			err := store.Append(ctx, events)
			Expect(err).NotTo(HaveOccurred())

			// Test channel-based streaming
			query := NewQuerySimple(NewTags("test", "value"), "TestEvent")
			eventChan, _, err := channelStore.ReadStreamChannel(ctx, query)
			Expect(err).NotTo(HaveOccurred())

			count := 0
			for event := range eventChan {
				Expect(event.Type).To(Equal("TestEvent"))
				count++
			}

			Expect(count).To(Equal(3))
		})

		It("should handle empty result sets", func() {
			query := NewQuerySimple(NewTags("non-existent", "value"), "TestEvent")
			eventChan, _, err := channelStore.ReadStreamChannel(ctx, query)
			Expect(err).NotTo(HaveOccurred())

			count := 0
			for range eventChan {
				count++
			}

			Expect(count).To(Equal(0))
		})

		It("should handle context cancellation", func() {
			// Setup test data
			event1 := NewInputEvent("TestEvent", NewTags("test", "value"), toJSON(map[string]string{"data": "value1"}))
			event2 := NewInputEvent("TestEvent", NewTags("test", "value"), toJSON(map[string]string{"data": "value2"}))

			events := []InputEvent{event1, event2}

			err := store.Append(ctx, events)
			Expect(err).NotTo(HaveOccurred())

			// Create cancellable context
			cancelCtx, cancel := context.WithCancel(ctx)
			defer cancel()

			query := NewQuerySimple(NewTags("test", "value"), "TestEvent")
			eventChan, _, err := channelStore.ReadStreamChannel(cancelCtx, query)
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
			events := make([]InputEvent, 10)
			for i := 0; i < 10; i++ {
				event := NewInputEvent("TestEvent", NewTags("test", "value"), toJSON(map[string]string{"index": fmt.Sprintf("%d", i)}))
				events[i] = event
			}

			err := store.Append(ctx, events)
			Expect(err).NotTo(HaveOccurred())

			// Test with small batch size
			query := NewQuerySimple(NewTags("test", "value"), "TestEvent")
			eventChan, _, err := channelStore.ReadStreamChannel(ctx, query)
			Expect(err).NotTo(HaveOccurred())

			count := 0
			for event := range eventChan {
				Expect(event.Type).To(Equal("TestEvent"))
				count++
			}

			Expect(count).To(Equal(10))
		})
	})

	Describe("ProjectDecisionModelChannel", func() {
		It("should project states using channels", func() {
			// Setup test data
			event1 := NewInputEvent("AccountOpened", NewTags("account_id", "acc1"), toJSON(map[string]string{"balance": "100"}))
			event2 := NewInputEvent("MoneyTransferred", NewTags("account_id", "acc1"), toJSON(map[string]string{"amount": "50"}))
			event3 := NewInputEvent("MoneyTransferred", NewTags("account_id", "acc1"), toJSON(map[string]string{"amount": "25"}))

			events := []InputEvent{event1, event2, event3}

			err := store.Append(ctx, events)
			Expect(err).NotTo(HaveOccurred())

			// Define projectors
			projectors := []BatchProjector{
				{ID: "accountCount", StateProjector: StateProjector{
					Query:        NewQuerySimple(NewTags("account_id", "acc1"), "AccountOpened"),
					InitialState: 0,
					TransitionFn: func(state any, event Event) any {
						return state.(int) + 1
					},
				}},
				{ID: "transferCount", StateProjector: StateProjector{
					Query:        NewQuerySimple(NewTags("account_id", "acc1"), "MoneyTransferred"),
					InitialState: 0,
					TransitionFn: func(state any, event Event) any {
						return state.(int) + 1
					},
				}},
			}

			// Use channel-based projection
			resultChan, _, err := channelStore.ProjectDecisionModelChannel(ctx, projectors)
			Expect(err).NotTo(HaveOccurred())

			// Process results
			projectionCount := 0
			finalStates := make(map[string]interface{})

			for result := range resultChan {
				if result.Error != nil {
					Fail(fmt.Sprintf("Unexpected error: %v", result.Error))
				}

				projectionCount++
				finalStates[result.ProjectorID] = result.State
			}

			Expect(projectionCount).To(Equal(3)) // 1 AccountOpened + 2 MoneyTransferred
			Expect(finalStates["accountCount"]).To(Equal(1))
			Expect(finalStates["transferCount"]).To(Equal(2))
		})

		It("should handle empty projectors list", func() {
			_, _, err := channelStore.ProjectDecisionModelChannel(ctx, []BatchProjector{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("at least one projector is required"))
		})

		It("should handle nil transition function", func() {
			projectors := []BatchProjector{
				{ID: "invalid", StateProjector: StateProjector{
					Query:        NewQuerySimple(NewTags("test", "value"), "TestEvent"),
					InitialState: 0,
					TransitionFn: nil, // Nil transition function
				}},
			}

			_, _, err := channelStore.ProjectDecisionModelChannel(ctx, projectors)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("nil transition function"))
		})

		It("should handle context cancellation during projection", func() {
			// Setup test data
			event1 := NewInputEvent("TestEvent", NewTags("test", "value"), toJSON(map[string]string{"data": "value1"}))
			event2 := NewInputEvent("TestEvent", NewTags("test", "value"), toJSON(map[string]string{"data": "value2"}))

			events := []InputEvent{event1, event2}

			err := store.Append(ctx, events)
			Expect(err).NotTo(HaveOccurred())

			// Create cancellable context
			cancelCtx, cancel := context.WithCancel(ctx)
			defer cancel()

			projectors := []BatchProjector{
				{ID: "test", StateProjector: StateProjector{
					Query:        NewQuerySimple(NewTags("test", "value"), "TestEvent"),
					InitialState: 0,
					TransitionFn: func(state any, event Event) any {
						return state.(int) + 1
					},
				}},
			}

			resultChan, _, err := channelStore.ProjectDecisionModelChannel(cancelCtx, projectors)
			Expect(err).NotTo(HaveOccurred())

			// Cancel context after first result
			count := 0
			for result := range resultChan {
				if result.Error != nil {
					Expect(result.Error.Error()).To(ContainSubstring("context canceled"))
					break
				}
				count++
				if count == 1 {
					cancel()
					break
				}
			}

			Expect(count).To(Equal(1))
		})

		It("should handle projection with no matching events", func() {
			projectors := []BatchProjector{
				{ID: "test", StateProjector: StateProjector{
					Query:        NewQuerySimple(NewTags("non-existent", "value"), "TestEvent"),
					InitialState: 0,
					TransitionFn: func(state any, event Event) any {
						return state.(int) + 1
					},
				}},
			}

			resultChan, _, err := channelStore.ProjectDecisionModelChannel(ctx, projectors)
			Expect(err).NotTo(HaveOccurred())

			count := 0
			for result := range resultChan {
				if result.Error != nil {
					Fail(fmt.Sprintf("Unexpected error: %v", result.Error))
				}
				count++
			}

			Expect(count).To(Equal(0))
		})

		It("should handle multiple projectors with different event types", func() {
			// Setup test data
			event1 := NewInputEvent("AccountOpened", NewTags("account_id", "acc1"), toJSON(map[string]string{"balance": "100"}))
			event2 := NewInputEvent("MoneyTransferred", NewTags("account_id", "acc1"), toJSON(map[string]string{"amount": "50"}))
			event3 := NewInputEvent("AccountClosed", NewTags("account_id", "acc1"), toJSON(map[string]string{"reason": "inactive"}))

			events := []InputEvent{event1, event2, event3}

			err := store.Append(ctx, events)
			Expect(err).NotTo(HaveOccurred())

			// Define projectors for different event types
			projectors := []BatchProjector{
				{ID: "accountCount", StateProjector: StateProjector{
					Query:        NewQuerySimple(NewTags("account_id", "acc1"), "AccountOpened"),
					InitialState: 0,
					TransitionFn: func(state any, event Event) any {
						return state.(int) + 1
					},
				}},
				{ID: "transferCount", StateProjector: StateProjector{
					Query:        NewQuerySimple(NewTags("account_id", "acc1"), "MoneyTransferred"),
					InitialState: 0,
					TransitionFn: func(state any, event Event) any {
						return state.(int) + 1
					},
				}},
				{ID: "closeCount", StateProjector: StateProjector{
					Query:        NewQuerySimple(NewTags("account_id", "acc1"), "AccountClosed"),
					InitialState: 0,
					TransitionFn: func(state any, event Event) any {
						return state.(int) + 1
					},
				}},
			}

			// Use channel-based projection
			resultChan, _, err := channelStore.ProjectDecisionModelChannel(ctx, projectors)
			Expect(err).NotTo(HaveOccurred())

			// Process results
			projectionCount := 0
			finalStates := make(map[string]interface{})

			for result := range resultChan {
				if result.Error != nil {
					Fail(fmt.Sprintf("Unexpected error: %v", result.Error))
				}

				projectionCount++
				finalStates[result.ProjectorID] = result.State
			}

			Expect(projectionCount).To(Equal(3)) // 1 of each event type
			Expect(finalStates["accountCount"]).To(Equal(1))
			Expect(finalStates["transferCount"]).To(Equal(1))
			Expect(finalStates["closeCount"]).To(Equal(1))
		})

		It("should handle large datasets with channel streaming", func() {
			// Create many events
			events := make([]InputEvent, 100)
			for i := 0; i < 100; i++ {
				event := NewInputEvent("TestEvent", NewTags("test", "value"), toJSON(map[string]string{"index": fmt.Sprintf("%d", i)}))
				events[i] = event
			}

			err := store.Append(ctx, events)
			Expect(err).NotTo(HaveOccurred())

			// Define projector
			projectors := []BatchProjector{
				{ID: "count", StateProjector: StateProjector{
					Query:        NewQuerySimple(NewTags("test", "value"), "TestEvent"),
					InitialState: 0,
					TransitionFn: func(state any, event Event) any {
						return state.(int) + 1
					},
				}},
			}

			// Use channel-based projection
			resultChan, _, err := channelStore.ProjectDecisionModelChannel(ctx, projectors)
			Expect(err).NotTo(HaveOccurred())

			// Process results
			projectionCount := 0
			for result := range resultChan {
				if result.Error != nil {
					Fail(fmt.Sprintf("Unexpected error: %v", result.Error))
				}
				projectionCount++
			}

			Expect(projectionCount).To(Equal(100))
		})
	})

	Describe("Extension Interface Pattern", func() {
		It("should properly implement ChannelEventStore interface", func() {
			// Test that the store implements the ChannelEventStore interface
			var channelEventStore ChannelEventStore = channelStore
			Expect(channelEventStore).NotTo(BeNil())

			// Test that our implementation does implement ChannelEventStore
			// (since our eventStore has the ReadStreamChannel method)
			_, ok := store.(ChannelEventStore)
			Expect(ok).To(BeTrue(), "Our EventStore implementation should implement ChannelEventStore")
		})
	})

	Describe("Performance Characteristics", func() {
		It("should handle moderate dataset sizes efficiently", func() {
			// Create moderate dataset (100 events)
			events := make([]InputEvent, 100)
			for i := 0; i < 100; i++ {
				event := NewInputEvent("TestEvent", NewTags("test", "value"), toJSON(map[string]string{"index": fmt.Sprintf("%d", i)}))
				events[i] = event
			}

			err := store.Append(ctx, events)
			Expect(err).NotTo(HaveOccurred())

			// Test channel-based streaming performance
			start := time.Now()
			query := NewQuerySimple(NewTags("test", "value"), "TestEvent")
			eventChan, _, err := channelStore.ReadStreamChannel(ctx, query)
			Expect(err).NotTo(HaveOccurred())

			count := 0
			for range eventChan {
				count++
			}

			duration := time.Since(start)
			Expect(count).To(Equal(100))
			Expect(duration).To(BeNumerically("<", 5*time.Second)) // Should complete quickly
		})
	})
})
