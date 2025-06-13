package dcb

import (
	"encoding/json"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Streaming Projection", func() {
	BeforeEach(func() {
		err := truncateEventsTable(ctx, pool)
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("ProjectDecisionModel", func() {
		It("should project multiple states and return append condition", func() {
			// Setup test data
			events := []InputEvent{
				{Type: "AccountCreated", Tags: []Tag{{Key: "account_id", Value: "acc123"}}, Data: []byte(`{"balance": 1000}`)},
				{Type: "TransactionCompleted", Tags: []Tag{{Key: "account_id", Value: "acc123"}}, Data: []byte(`{"amount": 500}`)},
				{Type: "TransactionCompleted", Tags: []Tag{{Key: "account_id", Value: "acc123"}}, Data: []byte(`{"amount": 300}`)},
			}

			// Append events
			_, err := store.Append(ctx, events, nil)
			Expect(err).NotTo(HaveOccurred())

			// Define projectors
			projectors := []BatchProjector{
				{
					ID: "account",
					StateProjector: StateProjector{
						Query:        Query{Items: []QueryItem{{EventTypes: []string{"AccountCreated", "TransactionCompleted"}}}},
						InitialState: &AccountState{Balance: 0},
						TransitionFn: func(state any, event Event) any {
							account := state.(*AccountState)
							switch event.Type {
							case "AccountCreated":
								var data AccountCreatedData
								json.Unmarshal(event.Data, &data)
								account.Balance = int64(data.Balance)
							case "TransactionCompleted":
								var data TransactionCompletedData
								json.Unmarshal(event.Data, &data)
								account.Balance += int64(data.Amount)
							}
							return account
						},
					},
				},
			}

			// Test ProjectDecisionModel
			query := Query{Items: []QueryItem{{EventTypes: []string{"AccountCreated", "TransactionCompleted"}}}}
			states, appendCondition, err := store.ProjectDecisionModel(ctx, query, nil, projectors)
			Expect(err).NotTo(HaveOccurred())

			// Verify results
			Expect(states).NotTo(BeNil())
			Expect(appendCondition.After).NotTo(BeNil())
			Expect(*appendCondition.After).To(Equal(int64(3)))

			account, ok := states["account"].(*AccountState)
			Expect(ok).To(BeTrue())
			Expect(account.Balance).To(Equal(int64(1800))) // 1000 + 500 + 300
		})
	})

	Describe("ReadStream", func() {
		It("should return a pure event iterator", func() {
			// Setup test data
			events := []InputEvent{
				{Type: "AccountCreated", Tags: []Tag{{Key: "account_id", Value: "acc123"}}, Data: []byte(`{"balance": 1000}`)},
				{Type: "TransactionCompleted", Tags: []Tag{{Key: "account_id", Value: "acc123"}}, Data: []byte(`{"amount": 500}`)},
			}

			// Append events
			_, err := store.Append(ctx, events, nil)
			Expect(err).NotTo(HaveOccurred())

			// Test ReadStream (pure event iterator)
			query := Query{Items: []QueryItem{{EventTypes: []string{"AccountCreated", "TransactionCompleted"}}}}
			iterator, err := store.ReadStream(ctx, query, nil)
			Expect(err).NotTo(HaveOccurred())
			defer iterator.Close()

			// Process events
			var processedEvents []Event
			for iterator.Next() {
				event := iterator.Event()
				processedEvents = append(processedEvents, event)
			}

			// Verify results
			Expect(iterator.Err()).NotTo(HaveOccurred())
			Expect(processedEvents).To(HaveLen(2))
			Expect(processedEvents[0].Type).To(Equal("AccountCreated"))
			Expect(processedEvents[1].Type).To(Equal("TransactionCompleted"))
		})
	})

	Describe("ProjectDecisionModel with cursor streaming", func() {
		It("should use cursor-based streaming when BatchSize is specified", func() {
			// Create test events
			events := []InputEvent{
				{Type: "CourseCreated", Tags: []Tag{{Key: "course_id", Value: "course-1"}}, Data: []byte(`{"name":"Math 101"}`)},
				{Type: "StudentEnrolled", Tags: []Tag{{Key: "course_id", Value: "course-1"}, {Key: "student_id", Value: "student-1"}}, Data: []byte(`{"student_name":"Alice"}`)},
				{Type: "StudentEnrolled", Tags: []Tag{{Key: "course_id", Value: "course-1"}, {Key: "student_id", Value: "student-2"}}, Data: []byte(`{"student_name":"Bob"}`)},
			}

			// Append events
			_, err := store.Append(ctx, events, nil)
			Expect(err).To(BeNil())

			// Define projectors
			projectors := []BatchProjector{
				{
					ID: "course_enrollment",
					StateProjector: StateProjector{
						Query: Query{
							Items: []QueryItem{
								{
									EventTypes: []string{"CourseCreated", "StudentEnrolled"},
									Tags:       []Tag{{Key: "course_id", Value: "course-1"}},
								},
							},
						},
						InitialState: 0,
						TransitionFn: func(state any, event Event) any {
							count := state.(int)
							if event.Type == "StudentEnrolled" {
								return count + 1
							}
							return count
						},
					},
				},
			}

			// Query for events
			query := Query{
				Items: []QueryItem{
					{
						EventTypes: []string{"CourseCreated", "StudentEnrolled"},
						Tags:       []Tag{{Key: "course_id", Value: "course-1"}},
					},
				},
			}

			// Use cursor-based streaming with small batch size
			batchSize := 2
			options := &ReadOptions{BatchSize: &batchSize}

			// Test ProjectDecisionModel with cursor streaming
			states, appendCondition, err := store.ProjectDecisionModel(ctx, query, options, projectors)
			Expect(err).To(BeNil())
			Expect(states).To(HaveKey("course_enrollment"))
			Expect(states["course_enrollment"]).To(Equal(2)) // 2 StudentEnrolled events
			Expect(appendCondition.After).To(Not(BeNil()))
			Expect(*appendCondition.After).To(Equal(int64(3))) // Last event position
		})
	})

	Describe("Cursor Streaming Edge Cases", func() {
		It("should handle empty result set and batch size > event count", func() {
			batchSize := 10
			options := &ReadOptions{BatchSize: &batchSize}

			// 1. Empty result set
			emptyQuery := Query{Items: []QueryItem{{EventTypes: []string{"OrderCreated"}, Tags: []Tag{{Key: "customer_id", Value: "doesnotexist"}}}}}
			iterator, err := store.ReadStream(ctx, emptyQuery, options)
			Expect(err).To(BeNil())
			defer iterator.Close()
			Expect(iterator.Next()).To(BeFalse())
			Expect(iterator.Err()).To(BeNil())

			// 2. Batch size > number of events
			events := []InputEvent{
				{Type: "OrderCreated", Tags: []Tag{{Key: "customer_id", Value: "c1"}}, Data: []byte(`{"amount": 10}`)},
				{Type: "OrderCreated", Tags: []Tag{{Key: "customer_id", Value: "c1"}}, Data: []byte(`{"amount": 20}`)},
				{Type: "OrderCreated", Tags: []Tag{{Key: "customer_id", Value: "c1"}}, Data: []byte(`{"amount": 30}`)},
			}
			_, err = store.Append(ctx, events, nil)
			Expect(err).To(BeNil())
			query := Query{Items: []QueryItem{{EventTypes: []string{"OrderCreated"}, Tags: []Tag{{Key: "customer_id", Value: "c1"}}}}}
			iterator, err = store.ReadStream(ctx, query, options)
			Expect(err).To(BeNil())
			defer iterator.Close()
			count := 0
			for iterator.Next() {
				count++
			}
			Expect(count).To(Equal(3))
			Expect(iterator.Err()).To(BeNil())

			// 3. Iterator Close and resource cleanup (should not panic)
			iterator, err = store.ReadStream(ctx, query, options)
			Expect(err).To(BeNil())
			Expect(iterator.Close()).To(BeNil())
			Expect(iterator.Close()).To(BeNil()) // Double close should be safe
		})

		It("should handle boundary batch sizes", func() {
			// Create test events
			events := []InputEvent{
				{Type: "OrderCreated", Tags: []Tag{{Key: "customer_id", Value: "c2"}}, Data: []byte(`{"amount": 10}`)},
				{Type: "OrderCreated", Tags: []Tag{{Key: "customer_id", Value: "c2"}}, Data: []byte(`{"amount": 20}`)},
				{Type: "OrderCreated", Tags: []Tag{{Key: "customer_id", Value: "c2"}}, Data: []byte(`{"amount": 30}`)},
				{Type: "OrderCreated", Tags: []Tag{{Key: "customer_id", Value: "c2"}}, Data: []byte(`{"amount": 40}`)},
				{Type: "OrderCreated", Tags: []Tag{{Key: "customer_id", Value: "c2"}}, Data: []byte(`{"amount": 50}`)},
			}
			_, err := store.Append(ctx, events, nil)
			Expect(err).To(BeNil())

			query := Query{Items: []QueryItem{{EventTypes: []string{"OrderCreated"}, Tags: []Tag{{Key: "customer_id", Value: "c2"}}}}}

			// Test batch size of 1 (smallest meaningful batch)
			batchSize1 := 1
			options1 := &ReadOptions{BatchSize: &batchSize1}
			iterator, err := store.ReadStream(ctx, query, options1)
			Expect(err).To(BeNil())
			defer iterator.Close()
			count := 0
			for iterator.Next() {
				count++
			}
			Expect(count).To(Equal(5))
			Expect(iterator.Err()).To(BeNil())

			// Test batch size exactly equal to number of events
			batchSize5 := 5
			options5 := &ReadOptions{BatchSize: &batchSize5}
			iterator, err = store.ReadStream(ctx, query, options5)
			Expect(err).To(BeNil())
			defer iterator.Close()
			count = 0
			for iterator.Next() {
				count++
			}
			Expect(count).To(Equal(5))
			Expect(iterator.Err()).To(BeNil())
		})

		It("should handle ProjectDecisionModel with cursor streaming edge cases", func() {
			// Create events for testing
			events := []InputEvent{
				{Type: "OrderCreated", Tags: []Tag{{Key: "customer_id", Value: "c3"}}, Data: []byte(`{"amount": 100}`)},
				{Type: "OrderCreated", Tags: []Tag{{Key: "customer_id", Value: "c3"}}, Data: []byte(`{"amount": 200}`)},
				{Type: "OrderCreated", Tags: []Tag{{Key: "customer_id", Value: "c3"}}, Data: []byte(`{"amount": 300}`)},
			}
			_, err := store.Append(ctx, events, nil)
			Expect(err).To(BeNil())

			// Define projectors
			projectors := []BatchProjector{
				{
					ID: "order_count",
					StateProjector: StateProjector{
						Query: Query{
							Items: []QueryItem{
								{
									EventTypes: []string{"OrderCreated"},
									Tags:       []Tag{{Key: "customer_id", Value: "c3"}},
								},
							},
						},
						InitialState: 0,
						TransitionFn: func(state any, event Event) any {
							count := state.(int)
							return count + 1
						},
					},
				},
			}

			query := Query{
				Items: []QueryItem{
					{
						EventTypes: []string{"OrderCreated"},
						Tags:       []Tag{{Key: "customer_id", Value: "c3"}},
					},
				},
			}

			// Test with batch size 1 (smallest meaningful batch)
			batchSize := 1
			options := &ReadOptions{BatchSize: &batchSize}

			states, appendCondition, err := store.ProjectDecisionModel(ctx, query, options, projectors)
			Expect(err).To(BeNil())
			Expect(states).To(HaveKey("order_count"))
			Expect(states["order_count"]).To(Equal(3))
			Expect(appendCondition.After).To(Not(BeNil()))
			Expect(*appendCondition.After).To(Equal(int64(3)))
		})

		It("should handle nil ReadOptions gracefully", func() {
			// Test that ReadStream works with nil options (should use default batch size)
			query := Query{Items: []QueryItem{{EventTypes: []string{"OrderCreated"}, Tags: []Tag{{Key: "customer_id", Value: "nonexistent"}}}}}
			iterator, err := store.ReadStream(ctx, query, nil)
			Expect(err).To(BeNil())
			defer iterator.Close()
			Expect(iterator.Next()).To(BeFalse())
			Expect(iterator.Err()).To(BeNil())
		})

		It("should handle zero batch size gracefully", func() {
			// Test that zero batch size is handled (should use default)
			batchSize := 0
			options := &ReadOptions{BatchSize: &batchSize}
			query := Query{Items: []QueryItem{{EventTypes: []string{"OrderCreated"}, Tags: []Tag{{Key: "customer_id", Value: "nonexistent"}}}}}
			iterator, err := store.ReadStream(ctx, query, options)
			Expect(err).To(BeNil())
			defer iterator.Close()
			Expect(iterator.Next()).To(BeFalse())
			Expect(iterator.Err()).To(BeNil())
		})

		It("should handle negative batch size gracefully", func() {
			// Test that negative batch size is handled (should use default)
			batchSize := -1
			options := &ReadOptions{BatchSize: &batchSize}
			query := Query{Items: []QueryItem{{EventTypes: []string{"OrderCreated"}, Tags: []Tag{{Key: "customer_id", Value: "nonexistent"}}}}}
			iterator, err := store.ReadStream(ctx, query, options)
			Expect(err).To(BeNil())
			defer iterator.Close()
			Expect(iterator.Next()).To(BeFalse())
			Expect(iterator.Err()).To(BeNil())
		})

		It("should handle very large batch sizes", func() {
			// Create a few events
			events := []InputEvent{
				{Type: "OrderCreated", Tags: []Tag{{Key: "customer_id", Value: "c4"}}, Data: []byte(`{"amount": 100}`)},
				{Type: "OrderCreated", Tags: []Tag{{Key: "customer_id", Value: "c4"}}, Data: []byte(`{"amount": 200}`)},
			}
			_, err := store.Append(ctx, events, nil)
			Expect(err).To(BeNil())

			// Test with very large batch size
			batchSize := 1000000
			options := &ReadOptions{BatchSize: &batchSize}
			query := Query{Items: []QueryItem{{EventTypes: []string{"OrderCreated"}, Tags: []Tag{{Key: "customer_id", Value: "c4"}}}}}
			iterator, err := store.ReadStream(ctx, query, options)
			Expect(err).To(BeNil())
			defer iterator.Close()
			count := 0
			for iterator.Next() {
				count++
			}
			Expect(count).To(Equal(2))
			Expect(iterator.Err()).To(BeNil())
		})

		It("should handle cursor streaming with multiple event types", func() {
			// Create events with different types
			events := []InputEvent{
				{Type: "OrderCreated", Tags: []Tag{{Key: "customer_id", Value: "c5"}}, Data: []byte(`{"amount": 100}`)},
				{Type: "OrderCancelled", Tags: []Tag{{Key: "customer_id", Value: "c5"}}, Data: []byte(`{"reason": "test"}`)},
				{Type: "OrderCreated", Tags: []Tag{{Key: "customer_id", Value: "c5"}}, Data: []byte(`{"amount": 200}`)},
			}
			_, err := store.Append(ctx, events, nil)
			Expect(err).To(BeNil())

			// Query for multiple event types
			query := Query{
				Items: []QueryItem{
					{
						EventTypes: []string{"OrderCreated", "OrderCancelled"},
						Tags:       []Tag{{Key: "customer_id", Value: "c5"}},
					},
				},
			}

			batchSize := 2
			options := &ReadOptions{BatchSize: &batchSize}
			iterator, err := store.ReadStream(ctx, query, options)
			Expect(err).To(BeNil())
			defer iterator.Close()

			count := 0
			eventTypes := make([]string, 0)
			for iterator.Next() {
				event := iterator.Event()
				eventTypes = append(eventTypes, event.Type)
				count++
			}
			Expect(count).To(Equal(3))
			Expect(eventTypes).To(ContainElements("OrderCreated", "OrderCancelled", "OrderCreated"))
			Expect(iterator.Err()).To(BeNil())
		})

		It("should handle validation errors in ReadStream", func() {
			// Test empty query (should return validation error)
			emptyQuery := Query{Items: []QueryItem{}}
			iterator, err := store.ReadStream(ctx, emptyQuery, nil)
			Expect(err).To(Not(BeNil()))
			Expect(err.Error()).To(ContainSubstring("query must contain at least one item"))
			Expect(iterator).To(BeNil())
		})

		It("should handle validation errors in ProjectDecisionModel", func() {
			// Test empty query
			emptyQuery := Query{Items: []QueryItem{}}
			projectors := []BatchProjector{
				{
					ID: "test",
					StateProjector: StateProjector{
						Query:        Query{Items: []QueryItem{}},
						InitialState: 0,
						TransitionFn: func(state any, event Event) any { return state },
					},
				},
			}
			states, appendCondition, err := store.ProjectDecisionModel(ctx, emptyQuery, nil, projectors)
			Expect(err).To(Not(BeNil()))
			Expect(err.Error()).To(ContainSubstring("query must contain at least one item"))
			Expect(states).To(BeNil())
			Expect(appendCondition).To(Equal(AppendCondition{}))

			// Test nil transition function
			validQuery := Query{Items: []QueryItem{{EventTypes: []string{"Test"}}}}
			invalidProjectors := []BatchProjector{
				{
					ID: "test",
					StateProjector: StateProjector{
						Query:        Query{Items: []QueryItem{}},
						InitialState: 0,
						TransitionFn: nil, // This should cause validation error
					},
				},
			}
			states, appendCondition, err = store.ProjectDecisionModel(ctx, validQuery, nil, invalidProjectors)
			Expect(err).To(Not(BeNil()))
			Expect(err.Error()).To(ContainSubstring("has nil transition function"))
			Expect(states).To(BeNil())
			Expect(appendCondition).To(Equal(AppendCondition{}))
		})

		It("should handle complex projection scenarios with multiple projectors", func() {
			// Create events for complex scenario
			events := []InputEvent{
				{Type: "OrderCreated", Tags: []Tag{{Key: "customer_id", Value: "c6"}, {Key: "region", Value: "us"}}, Data: []byte(`{"amount": 100}`)},
				{Type: "OrderCreated", Tags: []Tag{{Key: "customer_id", Value: "c6"}, {Key: "region", Value: "eu"}}, Data: []byte(`{"amount": 200}`)},
				{Type: "OrderCancelled", Tags: []Tag{{Key: "customer_id", Value: "c6"}, {Key: "region", Value: "us"}}, Data: []byte(`{"reason": "test"}`)},
				{Type: "OrderCreated", Tags: []Tag{{Key: "customer_id", Value: "c7"}, {Key: "region", Value: "us"}}, Data: []byte(`{"amount": 300}`)},
			}
			_, err := store.Append(ctx, events, nil)
			Expect(err).To(BeNil())

			// Define multiple projectors with different queries
			projectors := []BatchProjector{
				{
					ID: "us_orders",
					StateProjector: StateProjector{
						Query: Query{
							Items: []QueryItem{
								{
									EventTypes: []string{"OrderCreated", "OrderCancelled"},
									Tags:       []Tag{{Key: "region", Value: "us"}},
								},
							},
						},
						InitialState: map[string]int{"count": 0, "total": 0},
						TransitionFn: func(state any, event Event) any {
							s := state.(map[string]int)
							if event.Type == "OrderCreated" {
								s["count"]++
								s["total"] += 100 // Simplified for test
							} else if event.Type == "OrderCancelled" {
								s["count"]--
							}
							return s
						},
					},
				},
				{
					ID: "eu_orders",
					StateProjector: StateProjector{
						Query: Query{
							Items: []QueryItem{
								{
									EventTypes: []string{"OrderCreated"},
									Tags:       []Tag{{Key: "region", Value: "eu"}},
								},
							},
						},
						InitialState: map[string]int{"count": 0, "total": 0},
						TransitionFn: func(state any, event Event) any {
							s := state.(map[string]int)
							if event.Type == "OrderCreated" {
								s["count"]++
								s["total"] += 200 // Simplified for test
							}
							return s
						},
					},
				},
			}

			// Query for all events
			query := Query{
				Items: []QueryItem{
					{
						EventTypes: []string{"OrderCreated", "OrderCancelled"},
						Tags:       []Tag{{Key: "customer_id", Value: "c6"}},
					},
					{
						EventTypes: []string{"OrderCreated", "OrderCancelled"},
						Tags:       []Tag{{Key: "customer_id", Value: "c7"}},
					},
				},
			}

			// Test with cursor streaming
			batchSize := 2
			options := &ReadOptions{BatchSize: &batchSize}

			states, appendCondition, err := store.ProjectDecisionModel(ctx, query, options, projectors)
			Expect(err).To(BeNil())
			Expect(states).To(HaveKey("us_orders"))
			Expect(states).To(HaveKey("eu_orders"))

			usOrders := states["us_orders"].(map[string]int)
			euOrders := states["eu_orders"].(map[string]int)

			Expect(usOrders["count"]).To(Equal(1))   // 2 created - 1 cancelled
			Expect(usOrders["total"]).To(Equal(200)) // 2 * 100
			Expect(euOrders["count"]).To(Equal(1))   // 1 created
			Expect(euOrders["total"]).To(Equal(200)) // 1 * 200

			Expect(appendCondition.After).To(Not(BeNil()))
			Expect(*appendCondition.After).To(Equal(int64(4)))
		})

		It("should handle ReadOptions with FromPosition and Limit", func() {
			// Create events
			events := []InputEvent{
				{Type: "OrderCreated", Tags: []Tag{{Key: "customer_id", Value: "c8"}}, Data: []byte(`{"amount": 100}`)},
				{Type: "OrderCreated", Tags: []Tag{{Key: "customer_id", Value: "c8"}}, Data: []byte(`{"amount": 200}`)},
				{Type: "OrderCreated", Tags: []Tag{{Key: "customer_id", Value: "c8"}}, Data: []byte(`{"amount": 300}`)},
				{Type: "OrderCreated", Tags: []Tag{{Key: "customer_id", Value: "c8"}}, Data: []byte(`{"amount": 400}`)},
			}
			_, err := store.Append(ctx, events, nil)
			Expect(err).To(BeNil())

			query := Query{Items: []QueryItem{{EventTypes: []string{"OrderCreated"}, Tags: []Tag{{Key: "customer_id", Value: "c8"}}}}}

			// Test with FromPosition
			fromPos := int64(2) // Start from position 2
			limit := 2
			options := &ReadOptions{
				FromPosition: &fromPos,
				Limit:        &limit,
				BatchSize:    intPtr(1), // Use cursor streaming
			}

			iterator, err := store.ReadStream(ctx, query, options)
			Expect(err).To(BeNil())
			defer iterator.Close()

			count := 0
			for iterator.Next() {
				event := iterator.Event()
				Expect(event.Position).To(BeNumerically(">=", fromPos))
				count++
			}
			Expect(count).To(Equal(2)) // Limited to 2 events
			Expect(iterator.Err()).To(BeNil())
		})

		It("should handle iterator error scenarios", func() {
			// Create events
			events := []InputEvent{
				{Type: "OrderCreated", Tags: []Tag{{Key: "customer_id", Value: "c9"}}, Data: []byte(`{"amount": 100}`)},
			}
			_, err := store.Append(ctx, events, nil)
			Expect(err).To(BeNil())

			query := Query{Items: []QueryItem{{EventTypes: []string{"OrderCreated"}, Tags: []Tag{{Key: "customer_id", Value: "c9"}}}}}
			batchSize := 1
			options := &ReadOptions{BatchSize: &batchSize}

			iterator, err := store.ReadStream(ctx, query, options)
			Expect(err).To(BeNil())

			// Test Event() before Next()
			event := iterator.Event()
			Expect(event).To(Equal(Event{})) // Should return empty event

			// Test Next() and Event()
			Expect(iterator.Next()).To(BeTrue())
			event = iterator.Event()
			Expect(event.Type).To(Equal("OrderCreated"))
			Expect(event.Position).To(Equal(int64(1)))

			// Test Next() after end
			Expect(iterator.Next()).To(BeFalse())
			Expect(iterator.Err()).To(BeNil())

			iterator.Close()
		})

		It("should handle concurrent cursor streaming", func() {
			// Create events
			events := []InputEvent{
				{Type: "OrderCreated", Tags: []Tag{{Key: "customer_id", Value: "c10"}}, Data: []byte(`{"amount": 100}`)},
				{Type: "OrderCreated", Tags: []Tag{{Key: "customer_id", Value: "c10"}}, Data: []byte(`{"amount": 200}`)},
			}
			_, err := store.Append(ctx, events, nil)
			Expect(err).To(BeNil())

			query := Query{Items: []QueryItem{{EventTypes: []string{"OrderCreated"}, Tags: []Tag{{Key: "customer_id", Value: "c10"}}}}}
			batchSize := 1
			options := &ReadOptions{BatchSize: &batchSize}

			// Create two iterators concurrently
			iterator1, err := store.ReadStream(ctx, query, options)
			Expect(err).To(BeNil())
			defer iterator1.Close()

			iterator2, err := store.ReadStream(ctx, query, options)
			Expect(err).To(BeNil())
			defer iterator2.Close()

			// Both should work independently
			count1 := 0
			for iterator1.Next() {
				count1++
			}
			Expect(count1).To(Equal(2))
			Expect(iterator1.Err()).To(BeNil())

			count2 := 0
			for iterator2.Next() {
				count2++
			}
			Expect(count2).To(Equal(2))
			Expect(iterator2.Err()).To(BeNil())
		})

		It("should test Read API with various options", func() {
			// Debug: Check initial state
			GinkgoWriter.Println("=== BEFORE TEST: Read API with various options ===")
			dumpEvents(pool)

			// Create events
			events := []InputEvent{
				{Type: "OrderCreated", Tags: []Tag{{Key: "customer_id", Value: "c11"}}, Data: []byte(`{"amount": 100}`)},
				{Type: "OrderCreated", Tags: []Tag{{Key: "customer_id", Value: "c11"}}, Data: []byte(`{"amount": 200}`)},
				{Type: "OrderCreated", Tags: []Tag{{Key: "customer_id", Value: "c11"}}, Data: []byte(`{"amount": 300}`)},
			}
			_, err := store.Append(ctx, events, nil)
			Expect(err).To(BeNil())

			// Debug: Check after append
			GinkgoWriter.Println("=== AFTER APPEND ===")
			dumpEvents(pool)

			query := Query{Items: []QueryItem{{EventTypes: []string{"OrderCreated"}, Tags: []Tag{{Key: "customer_id", Value: "c11"}}}}}

			// Test Read with nil options
			sequencedEvents, err := store.Read(ctx, query, nil)
			Expect(err).To(BeNil())
			Expect(sequencedEvents.Events).To(HaveLen(3))
			Expect(sequencedEvents.Position).To(Equal(int64(3)))

			// Test Read with limit
			limit := 2
			options := &ReadOptions{Limit: &limit}
			sequencedEvents, err = store.Read(ctx, query, options)
			Expect(err).To(BeNil())
			Expect(sequencedEvents.Events).To(HaveLen(2))
			Expect(sequencedEvents.Position).To(Equal(int64(2)))

			// Test Read with FromPosition
			fromPos := int64(2)
			options = &ReadOptions{FromPosition: &fromPos}
			sequencedEvents, err = store.Read(ctx, query, options)
			Expect(err).To(BeNil())
			Expect(sequencedEvents.Events).To(HaveLen(1))
			Expect(sequencedEvents.Events[0].Position).To(Equal(int64(3)))
			Expect(sequencedEvents.Position).To(Equal(int64(3)))

			// Test Read with empty result
			emptyQuery := Query{Items: []QueryItem{{EventTypes: []string{"OrderCreated"}, Tags: []Tag{{Key: "customer_id", Value: "nonexistent"}}}}}
			sequencedEvents, err = store.Read(ctx, emptyQuery, nil)
			Expect(err).To(BeNil())
			Expect(sequencedEvents.Events).To(HaveLen(0))
			Expect(sequencedEvents.Position).To(Equal(int64(0)))

			// Test Read with truly empty query (no items)
			trulyEmptyQuery := Query{Items: []QueryItem{}}
			sequencedEvents, err = store.Read(ctx, trulyEmptyQuery, nil)
			Expect(err).To(Not(BeNil()))
			Expect(err.Error()).To(ContainSubstring("query must contain at least one item"))
			Expect(sequencedEvents).To(Equal(SequencedEvents{}))
		})

		It("should test Append API with various conditions", func() {
			// Test Append with nil condition
			events := []InputEvent{
				{Type: "OrderCreated", Tags: []Tag{{Key: "customer_id", Value: "c12"}}, Data: []byte(`{"amount": 100}`)},
			}
			position, err := store.Append(ctx, events, nil)
			Expect(err).To(BeNil())
			Expect(position).To(Equal(int64(1)))

			// Test Append with empty events slice
			emptyEvents := []InputEvent{}
			position, err = store.Append(ctx, emptyEvents, nil)
			Expect(err).To(Not(BeNil()))
			Expect(err.Error()).To(ContainSubstring("events must not be empty"))

			// Test Append with optimistic locking
			query := Query{Items: []QueryItem{{EventTypes: []string{"OrderCreated"}, Tags: []Tag{{Key: "customer_id", Value: "c12"}}}}}
			sequencedEvents, err := store.Read(ctx, query, nil)
			Expect(err).To(BeNil())

			appendCondition := AppendCondition{
				After: &sequencedEvents.Position,
			}

			events = []InputEvent{
				{Type: "OrderCreated", Tags: []Tag{{Key: "customer_id", Value: "c12"}}, Data: []byte(`{"amount": 200}`)},
			}
			position, err = store.Append(ctx, events, &appendCondition)
			Expect(err).To(BeNil())
			Expect(position).To(Equal(int64(2)))

			// Test Append with conflicting condition
			conflictingCondition := AppendCondition{
				After: int64Ptr(int64(0)), // Should conflict with current position
			}
			events = []InputEvent{
				{Type: "OrderCreated", Tags: []Tag{{Key: "customer_id", Value: "c12"}}, Data: []byte(`{"amount": 300}`)},
			}
			position, err = store.Append(ctx, events, &conflictingCondition)
			Expect(err).To(Not(BeNil()))
			Expect(err.Error()).To(ContainSubstring("optimistic concurrency conflict"))
		})

		It("should test ProjectDecisionModel API with various scenarios", func() {
			// Create events
			events := []InputEvent{
				{Type: "OrderCreated", Tags: []Tag{{Key: "customer_id", Value: "c13"}}, Data: []byte(`{"amount": 100}`)},
				{Type: "OrderCreated", Tags: []Tag{{Key: "customer_id", Value: "c13"}}, Data: []byte(`{"amount": 200}`)},
			}
			_, err := store.Append(ctx, events, nil)
			Expect(err).To(BeNil())

			// Define projectors
			projectors := []BatchProjector{
				{
					ID: "order_count",
					StateProjector: StateProjector{
						Query: Query{
							Items: []QueryItem{
								{
									EventTypes: []string{"OrderCreated"},
									Tags:       []Tag{{Key: "customer_id", Value: "c13"}},
								},
							},
						},
						InitialState: 0,
						TransitionFn: func(state any, event Event) any {
							count := state.(int)
							return count + 1
						},
					},
				},
			}

			query := Query{
				Items: []QueryItem{
					{
						EventTypes: []string{"OrderCreated"},
						Tags:       []Tag{{Key: "customer_id", Value: "c13"}},
					},
				},
			}

			// Test ProjectDecisionModel with nil options (uses query-based approach)
			states, appendCondition, err := store.ProjectDecisionModel(ctx, query, nil, projectors)
			Expect(err).To(BeNil())
			Expect(states).To(HaveKey("order_count"))
			Expect(states["order_count"]).To(Equal(2))
			Expect(appendCondition.After).To(Not(BeNil()))
			Expect(*appendCondition.After).To(Equal(int64(2)))

			// Test ProjectDecisionModel with cursor streaming
			batchSize := 1
			options := &ReadOptions{BatchSize: &batchSize}
			states, appendCondition, err = store.ProjectDecisionModel(ctx, query, options, projectors)
			Expect(err).To(BeNil())
			Expect(states).To(HaveKey("order_count"))
			Expect(states["order_count"]).To(Equal(2))
			Expect(appendCondition.After).To(Not(BeNil()))
			Expect(*appendCondition.After).To(Equal(int64(2)))

			// Test ProjectDecisionModel with empty result
			emptyQuery := Query{
				Items: []QueryItem{
					{
						EventTypes: []string{"OrderCreated"},
						Tags:       []Tag{{Key: "customer_id", Value: "nonexistent"}},
					},
				},
			}
			states, appendCondition, err = store.ProjectDecisionModel(ctx, emptyQuery, nil, projectors)
			Expect(err).To(BeNil())
			Expect(states).To(HaveKey("order_count"))
			Expect(states["order_count"]).To(Equal(0))
			Expect(appendCondition.After).To(BeNil())

			// Test ProjectDecisionModel with truly empty query (no items)
			trulyEmptyQuery := Query{Items: []QueryItem{}}
			states, appendCondition, err = store.ProjectDecisionModel(ctx, trulyEmptyQuery, nil, projectors)
			Expect(err).To(Not(BeNil()))
			Expect(err.Error()).To(ContainSubstring("query must contain at least one item"))
			Expect(states).To(BeNil())
			Expect(appendCondition).To(Equal(AppendCondition{}))
		})

		It("should test EventIterator API thoroughly", func() {
			// Create events
			events := []InputEvent{
				{Type: "OrderCreated", Tags: []Tag{{Key: "customer_id", Value: "c14"}}, Data: []byte(`{"amount": 100}`)},
				{Type: "OrderCreated", Tags: []Tag{{Key: "customer_id", Value: "c14"}}, Data: []byte(`{"amount": 200}`)},
			}
			_, err := store.Append(ctx, events, nil)
			Expect(err).To(BeNil())

			query := Query{Items: []QueryItem{{EventTypes: []string{"OrderCreated"}, Tags: []Tag{{Key: "customer_id", Value: "c14"}}}}}
			batchSize := 1
			options := &ReadOptions{BatchSize: &batchSize}

			iterator, err := store.ReadStream(ctx, query, options)
			Expect(err).To(BeNil())

			// Test Event() before Next()
			event := iterator.Event()
			Expect(event).To(Equal(Event{}))

			// Test Next() and Event() sequence
			Expect(iterator.Next()).To(BeTrue())
			event = iterator.Event()
			Expect(event.Type).To(Equal("OrderCreated"))
			Expect(event.Position).To(Equal(int64(1)))

			Expect(iterator.Next()).To(BeTrue())
			event = iterator.Event()
			Expect(event.Type).To(Equal("OrderCreated"))
			Expect(event.Position).To(Equal(int64(2)))

			// Test Next() after end
			Expect(iterator.Next()).To(BeFalse())
			Expect(iterator.Err()).To(BeNil())

			// Test Close()
			Expect(iterator.Close()).To(BeNil())

			// Test Err() after close
			Expect(iterator.Err()).To(BeNil())
		})

		It("should test public API error handling", func() {
			// Test Read with invalid query
			invalidQuery := Query{Items: []QueryItem{}}
			sequencedEvents, err := store.Read(ctx, invalidQuery, nil)
			Expect(err).To(Not(BeNil()))
			Expect(err.Error()).To(ContainSubstring("query must contain at least one item"))
			Expect(sequencedEvents).To(Equal(SequencedEvents{}))

			// Test ReadStream with invalid query
			iterator, err := store.ReadStream(ctx, invalidQuery, nil)
			Expect(err).To(Not(BeNil()))
			Expect(err.Error()).To(ContainSubstring("query must contain at least one item"))
			Expect(iterator).To(BeNil())

			// Test Append with invalid events
			invalidEvents := []InputEvent{
				{Type: "", Tags: []Tag{}, Data: []byte{}}, // Empty type
			}
			position, err := store.Append(ctx, invalidEvents, nil)
			Expect(err).To(Not(BeNil()))
			Expect(position).To(Equal(int64(0)))

			// Test ProjectDecisionModel with invalid projectors
			validQuery := Query{Items: []QueryItem{{EventTypes: []string{"Test"}}}}
			invalidProjectors := []BatchProjector{
				{
					ID: "test",
					StateProjector: StateProjector{
						Query:        Query{Items: []QueryItem{}},
						InitialState: 0,
						TransitionFn: nil, // Invalid: nil transition function
					},
				},
			}
			states, appendCondition, err := store.ProjectDecisionModel(ctx, validQuery, nil, invalidProjectors)
			Expect(err).To(Not(BeNil()))
			Expect(err.Error()).To(ContainSubstring("has nil transition function"))
			Expect(states).To(BeNil())
			Expect(appendCondition).To(Equal(AppendCondition{}))
		})
	})
})

// Helper types for testing
type AccountState struct {
	ID      string
	Balance int64
}

type TransactionState struct {
	Count       int
	TotalAmount int
}

type BalanceState struct {
	CurrentBalance int
	LastUpdated    time.Time
}

type AccountCreatedData struct {
	Balance int `json:"balance"`
}

type AccountUpdatedData struct {
	NewBalance int `json:"new_balance"`
}

type TransactionCompletedData struct {
	Amount int `json:"amount"`
}

func mustMarshal(v any) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}

func intPtr(i int) *int {
	return &i
}

func int64Ptr(i int64) *int64 {
	return &i
}
