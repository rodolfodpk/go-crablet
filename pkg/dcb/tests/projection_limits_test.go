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

var _ = Describe("Projection Limits", func() {
	var (
		store dcb.EventStore
		ctx   context.Context
	)

	BeforeEach(func() {
		var err error
		ctx = context.Background()
		err = truncateEventsTable(ctx, pool)
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("MaxConcurrentProjections Limit", func() {
		It("should enforce MaxConcurrentProjections limit with low limit", func() {
			// Create EventStore with very low limit for testing
			config := dcb.EventStoreConfig{
				MaxConcurrentProjections: 3, // Very low limit for testing
				MaxProjectionGoroutines:  10,
				StreamBuffer:             100,
				QueryTimeout:             5000,
				AppendTimeout:            3000,
			}

			var err error
			store, err = dcb.NewEventStoreWithConfig(ctx, pool, config)
			Expect(err).NotTo(HaveOccurred())

			// Create test data
			testEvents := make([]dcb.InputEvent, 50)
			for i := 0; i < 50; i++ {
				testEvents[i] = dcb.NewInputEvent("TestEvent",
					dcb.NewTags("test", "limits", "id", fmt.Sprintf("event_%d", i)),
					dcb.ToJSON(map[string]string{"value": fmt.Sprintf("test_%d", i)}))
			}

			err = store.Append(ctx, testEvents)
			Expect(err).NotTo(HaveOccurred())

			// Create projector with longer processing time to ensure semaphore overlap
			projector := dcb.StateProjector{
				ID:           "test_limit_projection",
				Query:        dcb.NewQuery(dcb.NewTags("test", "limits"), "TestEvent"),
				InitialState: 0,
				TransitionFn: func(state any, event dcb.Event) any {
					// Longer delay to ensure semaphore overlap
					time.Sleep(100 * time.Millisecond)
					return state.(int) + 1
				},
			}

			// Use context with timeout to prevent hanging
			ctxWithTimeout, cancel := context.WithTimeout(ctx, 1*time.Second)
			defer cancel()

			// Launch 8 concurrent projections (exceeds limit of 3)
			var wg sync.WaitGroup
			results := make(chan error, 8)

			for i := 0; i < 8; i++ {
				wg.Go(func() {
					_, _, err := store.Project(ctxWithTimeout, []dcb.StateProjector{projector}, nil)
					results <- err
				})
			}

			wg.Wait()
			close(results)

			// Count successes vs limit exceeded errors
			successCount := 0
			limitExceededCount := 0
			otherErrors := 0

			for err := range results {
				if err == nil {
					successCount++
				} else if tooManyErr, ok := err.(*dcb.TooManyProjectionsError); ok {
					limitExceededCount++
					// Verify error details
					Expect(tooManyErr.MaxConcurrent).To(Equal(3))
					Expect(tooManyErr.CurrentCount).To(BeNumerically("<=", 3))
				} else {
					otherErrors++
					fmt.Printf("Unexpected error: %v\n", err)
				}
			}

			// With fail-fast behavior, we should have some successes and some immediate failures
			Expect(successCount).To(BeNumerically(">=", 1), "Expected at least 1 successful projection")
			Expect(successCount).To(BeNumerically("<=", 3), "Expected at most 3 successful projections (limit)")
			Expect(successCount+limitExceededCount+otherErrors).To(Equal(8), "All operations should be accounted for")

			// The semaphore should fail fast instead of blocking
			Expect(limitExceededCount).To(BeNumerically(">=", 1), "Expected some operations to fail fast due to semaphore limit")
		})

		It("should enforce ProjectStream limits with low limit", func() {
			// Create EventStore with very low limit for testing
			config := dcb.EventStoreConfig{
				MaxConcurrentProjections: 2, // Very low limit for testing
				MaxProjectionGoroutines:  10,
				StreamBuffer:             100,
				QueryTimeout:             5000,
				AppendTimeout:            3000,
			}

			var err error
			store, err = dcb.NewEventStoreWithConfig(ctx, pool, config)
			Expect(err).NotTo(HaveOccurred())

			// Create test data
			testEvents := make([]dcb.InputEvent, 30)
			for i := 0; i < 30; i++ {
				testEvents[i] = dcb.NewInputEvent("TestEvent",
					dcb.NewTags("test", "stream_limits", "id", fmt.Sprintf("event_%d", i)),
					dcb.ToJSON(map[string]string{"value": fmt.Sprintf("test_%d", i)}))
			}

			err = store.Append(ctx, testEvents)
			Expect(err).NotTo(HaveOccurred())

			// Create projector with longer processing time to ensure semaphore overlap
			projector := dcb.StateProjector{
				ID:           "test_stream_limit_projection",
				Query:        dcb.NewQuery(dcb.NewTags("test", "stream_limits"), "TestEvent"),
				InitialState: 0,
				TransitionFn: func(state any, event dcb.Event) any {
					// Longer delay to ensure semaphore overlap
					time.Sleep(100 * time.Millisecond)
					return state.(int) + 1
				},
			}

			// Use context with timeout to prevent hanging
			ctxWithTimeout, cancel := context.WithTimeout(ctx, 1*time.Second)
			defer cancel()

			// Launch 5 concurrent ProjectStream operations (exceeds limit of 2)
			var wg sync.WaitGroup
			results := make(chan error, 5)

			for i := 0; i < 5; i++ {
				wg.Go(func() {
					stateChan, conditionChan, err := store.ProjectStream(ctxWithTimeout, []dcb.StateProjector{projector}, nil)
					if err != nil {
						results <- err
						return
					}

					// Consume from channels
					select {
					case state := <-stateChan:
						_ = state // Use state to prevent optimization
					case <-time.After(500 * time.Millisecond):
						results <- fmt.Errorf("ProjectStream timeout")
						return
					}

					// Wait for condition
					select {
					case condition := <-conditionChan:
						_ = condition // Use condition to prevent optimization
					case <-time.After(200 * time.Millisecond):
						results <- fmt.Errorf("ProjectStream condition timeout")
						return
					}

					results <- nil
				})
			}

			wg.Wait()
			close(results)

			// Count successes vs limit exceeded errors
			successCount := 0
			limitExceededCount := 0
			otherErrors := 0

			for err := range results {
				if err == nil {
					successCount++
				} else if tooManyErr, ok := err.(*dcb.TooManyProjectionsError); ok {
					limitExceededCount++
					// Verify error details
					Expect(tooManyErr.MaxConcurrent).To(Equal(2))
					Expect(tooManyErr.CurrentCount).To(BeNumerically("<=", 2))
				} else {
					otherErrors++
					fmt.Printf("Unexpected error: %v\n", err)
				}
			}

			// With fail-fast behavior, we should have some successes and some immediate failures
			Expect(successCount).To(BeNumerically(">=", 0), "Expected at least 0 successful ProjectStream operations")
			Expect(successCount).To(BeNumerically("<=", 2), "Expected at most 2 successful ProjectStream operations (limit)")
			Expect(successCount+limitExceededCount+otherErrors).To(Equal(5), "All operations should be accounted for")

			// The semaphore should fail fast instead of blocking
			Expect(limitExceededCount).To(BeNumerically(">=", 1), "Expected some operations to fail fast due to semaphore limit")
		})

		It("should fail fast when semaphore limit is exceeded", func() {
			// Create EventStore with very low limit for testing
			config := dcb.EventStoreConfig{
				MaxConcurrentProjections: 1, // Very low limit for testing
				MaxProjectionGoroutines:  10,
				StreamBuffer:             100,
				QueryTimeout:             5000,
				AppendTimeout:            3000,
			}

			var err error
			store, err = dcb.NewEventStoreWithConfig(ctx, pool, config)
			Expect(err).NotTo(HaveOccurred())

			// Create test data
			testEvents := make([]dcb.InputEvent, 5)
			for i := 0; i < 5; i++ {
				testEvents[i] = dcb.NewInputEvent("TestEvent",
					dcb.NewTags("test", "fail_fast", "id", fmt.Sprintf("event_%d", i)),
					dcb.ToJSON(map[string]string{"value": fmt.Sprintf("test_%d", i)}))
			}

			err = store.Append(ctx, testEvents)
			Expect(err).NotTo(HaveOccurred())

			// Create projector with moderate processing time
			projector := dcb.StateProjector{
				ID:           "test_fail_fast_projection",
				Query:        dcb.NewQuery(dcb.NewTags("test", "fail_fast"), "TestEvent"),
				InitialState: 0,
				TransitionFn: func(state any, event dcb.Event) any {
					// Moderate processing time
					time.Sleep(50 * time.Millisecond)
					return state.(int) + 1
				},
			}

			// Launch 3 concurrent projections (exceeds limit of 1)
			var wg sync.WaitGroup
			results := make(chan error, 3)

			for i := 0; i < 3; i++ {
				wg.Go(func() {
					_, _, err := store.Project(ctx, []dcb.StateProjector{projector}, nil)
					results <- err
				})
			}

			wg.Wait()
			close(results)

			// Count results
			successCount := 0
			limitExceededCount := 0
			otherErrors := 0

			for err := range results {
				if err == nil {
					successCount++
				} else if tooManyErr, ok := err.(*dcb.TooManyProjectionsError); ok {
					limitExceededCount++
					// Verify error details
					Expect(tooManyErr.MaxConcurrent).To(Equal(1))
					Expect(tooManyErr.CurrentCount).To(Equal(1))
					Expect(tooManyErr.Error()).To(ContainSubstring("too many concurrent projections"))
				} else {
					otherErrors++
					fmt.Printf("Unexpected error: %v\n", err)
				}
			}

			// Should have 1 success and 2 fail-fast errors
			Expect(successCount).To(Equal(1), "Expected exactly 1 successful projection")
			Expect(limitExceededCount).To(Equal(2), "Expected exactly 2 fail-fast errors")
			Expect(otherErrors).To(Equal(0), "Expected no other errors")
		})
	})

	Describe("MaxProjectionGoroutines Limit", func() {
		It("should handle MaxProjectionGoroutines limit", func() {
			// Create EventStore with low goroutine limit for testing
			config := dcb.EventStoreConfig{
				MaxConcurrentProjections: 5,
				MaxProjectionGoroutines:  2, // Very low limit for testing
				StreamBuffer:             100,
				QueryTimeout:             5000,
				AppendTimeout:            3000,
			}

			var err error
			store, err = dcb.NewEventStoreWithConfig(ctx, pool, config)
			Expect(err).NotTo(HaveOccurred())

			// Create test data
			testEvents := make([]dcb.InputEvent, 20)
			for i := 0; i < 20; i++ {
				testEvents[i] = dcb.NewInputEvent("TestEvent",
					dcb.NewTags("test", "goroutine_limits", "id", fmt.Sprintf("event_%d", i)),
					dcb.ToJSON(map[string]string{"value": fmt.Sprintf("test_%d", i)}))
			}

			err = store.Append(ctx, testEvents)
			Expect(err).NotTo(HaveOccurred())

			// Create projector
			projector := dcb.StateProjector{
				ID:           "test_goroutine_limit_projection",
				Query:        dcb.NewQuery(dcb.NewTags("test", "goroutine_limits"), "TestEvent"),
				InitialState: 0,
				TransitionFn: func(state any, event dcb.Event) any {
					return state.(int) + 1
				},
			}

			// Test ProjectStream with low goroutine limit
			stateChan, conditionChan, err := store.ProjectStream(ctx, []dcb.StateProjector{projector}, nil)
			Expect(err).NotTo(HaveOccurred())

			// Consume from channels
			var finalState map[string]any
			select {
			case state := <-stateChan:
				finalState = state
			case <-time.After(5 * time.Second):
				Fail("ProjectStream timeout")
			}

			// Wait for condition
			select {
			case condition := <-conditionChan:
				Expect(condition).NotTo(BeNil())
			case <-time.After(2 * time.Second):
				Fail("ProjectStream condition timeout")
			}

			// Verify result
			Expect(finalState).NotTo(BeNil())
			Expect(finalState["test_goroutine_limit_projection"]).To(Equal(20))
		})
	})

	Describe("Error Handling", func() {
		It("should return TooManyProjectionsError with correct details", func() {
			// Create EventStore with very low limit for testing
			config := dcb.EventStoreConfig{
				MaxConcurrentProjections: 1, // Very low limit for testing
				MaxProjectionGoroutines:  10,
				StreamBuffer:             100,
				QueryTimeout:             5000,
				AppendTimeout:            3000,
			}

			var err error
			store, err = dcb.NewEventStoreWithConfig(ctx, pool, config)
			Expect(err).NotTo(HaveOccurred())

			// Create test data
			testEvents := make([]dcb.InputEvent, 5)
			for i := 0; i < 5; i++ {
				testEvents[i] = dcb.NewInputEvent("TestEvent",
					dcb.NewTags("test", "error_details", "id", fmt.Sprintf("event_%d", i)),
					dcb.ToJSON(map[string]string{"value": fmt.Sprintf("test_%d", i)}))
			}

			err = store.Append(ctx, testEvents)
			Expect(err).NotTo(HaveOccurred())

			// Create projector with long processing time to hold the semaphore
			projector := dcb.StateProjector{
				ID:           "test_error_details_projection",
				Query:        dcb.NewQuery(dcb.NewTags("test", "error_details"), "TestEvent"),
				InitialState: 0,
				TransitionFn: func(state any, event dcb.Event) any {
					// Long processing time to hold the semaphore
					time.Sleep(200 * time.Millisecond)
					return state.(int) + 1
				},
			}

			// Start first projection (will succeed and hold semaphore)
			var wg sync.WaitGroup
			results := make(chan error, 2)
			start := make(chan struct{})

			// First projection (should succeed and hold semaphore)
			wg.Go(func() {
				_, _, err := store.Project(ctx, []dcb.StateProjector{projector}, nil)
				results <- err
			})

			// Second projection (should fail with TooManyProjectionsError)
			wg.Go(func() {
				<-start // Wait for synchronization
				_, _, err := store.Project(ctx, []dcb.StateProjector{projector}, nil)
				results <- err
			})

			// Give first projection time to acquire semaphore
			time.Sleep(10 * time.Millisecond)
			close(start) // Start second projection

			wg.Wait()
			close(results)

			// Count results
			successCount := 0
			limitExceededCount := 0
			otherErrors := 0

			for err := range results {
				if err == nil {
					successCount++
				} else if tooManyErr, ok := err.(*dcb.TooManyProjectionsError); ok {
					limitExceededCount++
					// Verify error details
					Expect(tooManyErr.MaxConcurrent).To(Equal(1))
					Expect(tooManyErr.CurrentCount).To(Equal(1))
					Expect(tooManyErr.Error()).To(ContainSubstring("too many concurrent projections"))
				} else {
					otherErrors++
					fmt.Printf("Unexpected error: %v\n", err)
				}
			}

			// Should have 1 success and 1 limit exceeded error
			Expect(successCount).To(Equal(1), "Expected exactly 1 successful projection")
			Expect(limitExceededCount).To(Equal(1), "Expected exactly 1 limit exceeded error")
			Expect(otherErrors).To(Equal(0), "Expected no other errors")
		})
	})
})
