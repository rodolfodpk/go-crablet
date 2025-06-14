package dcb

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Batch Projection", func() {
	var (
		store EventStore
		ctx   context.Context
	)

	BeforeEach(func() {
		// Use shared PostgreSQL container and truncate events between tests
		store = NewEventStoreFromPool(pool)
		ctx = context.Background()

		// Truncate events table before each test
		err := truncateEventsTable(ctx, pool)
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("combineProjectorQueries", func() {
		It("should combine multiple projector queries with OR logic", func() {
			projectors := []BatchProjector{
				{ID: "projector1", StateProjector: StateProjector{
					Query: NewQuerySimple(NewTags("course_id", "c1"), "CourseCreated"),
				}},
				{ID: "projector2", StateProjector: StateProjector{
					Query: NewQuerySimple(NewTags("student_id", "s1"), "StudentRegistered"),
				}},
				{ID: "projector3", StateProjector: StateProjector{
					Query: NewQuerySimple(NewTags("course_id", "c1", "student_id", "s1"), "StudentEnrolled"),
				}},
			}

			// Access the private method through the store implementation
			es := store.(*eventStore)
			combinedQuery := es.combineProjectorQueries(projectors)

			Expect(combinedQuery.Items).To(HaveLen(3))
			Expect(combinedQuery.Items[0].EventTypes).To(Equal([]string{"CourseCreated"}))
			Expect(combinedQuery.Items[1].EventTypes).To(Equal([]string{"StudentRegistered"}))
			Expect(combinedQuery.Items[2].EventTypes).To(Equal([]string{"StudentEnrolled"}))
		})

		It("should handle empty projectors list", func() {
			es := store.(*eventStore)
			combinedQuery := es.combineProjectorQueries([]BatchProjector{})

			Expect(combinedQuery.Items).To(BeEmpty())
		})

		It("should handle single projector", func() {
			projectors := []BatchProjector{
				{ID: "single", StateProjector: StateProjector{
					Query: NewQuerySimple(NewTags("test", "value"), "TestEvent"),
				}},
			}

			es := store.(*eventStore)
			combinedQuery := es.combineProjectorQueries(projectors)

			Expect(combinedQuery.Items).To(HaveLen(1))
			Expect(combinedQuery.Items[0].EventTypes).To(Equal([]string{"TestEvent"}))
		})
	})

	Describe("eventMatchesProjector", func() {
		It("should match events with exact tag and type match", func() {
			es := store.(*eventStore)

			projector := StateProjector{
				Query: NewQuerySimple(NewTags("course_id", "c1"), "CourseCreated"),
			}

			event := Event{
				Type: "CourseCreated",
				Tags: []Tag{{Key: "course_id", Value: "c1"}},
			}

			matches := es.eventMatchesProjector(event, projector)
			Expect(matches).To(BeTrue())
		})

		It("should not match events with different types", func() {
			es := store.(*eventStore)

			projector := StateProjector{
				Query: NewQuerySimple(NewTags("course_id", "c1"), "CourseCreated"),
			}

			event := Event{
				Type: "StudentRegistered",
				Tags: []Tag{{Key: "course_id", Value: "c1"}},
			}

			matches := es.eventMatchesProjector(event, projector)
			Expect(matches).To(BeFalse())
		})

		It("should not match events with different tags", func() {
			es := store.(*eventStore)

			projector := StateProjector{
				Query: NewQuerySimple(NewTags("course_id", "c1"), "CourseCreated"),
			}

			event := Event{
				Type: "CourseCreated",
				Tags: []Tag{{Key: "course_id", Value: "c2"}},
			}

			matches := es.eventMatchesProjector(event, projector)
			Expect(matches).To(BeFalse())
		})

		It("should match events with subset of tags", func() {
			es := store.(*eventStore)

			projector := StateProjector{
				Query: NewQuerySimple(NewTags("course_id", "c1"), "CourseCreated"),
			}

			event := Event{
				Type: "CourseCreated",
				Tags: []Tag{
					{Key: "course_id", Value: "c1"},
					{Key: "student_id", Value: "s1"},
				},
			}

			matches := es.eventMatchesProjector(event, projector)
			Expect(matches).To(BeTrue())
		})

		It("should handle empty event types in projector", func() {
			es := store.(*eventStore)

			projector := StateProjector{
				Query: NewQuerySimple(NewTags("course_id", "c1")), // No event types
			}

			event := Event{
				Type: "AnyEvent",
				Tags: []Tag{{Key: "course_id", Value: "c1"}},
			}

			matches := es.eventMatchesProjector(event, projector)
			Expect(matches).To(BeTrue())
		})

		It("should handle empty tags in projector", func() {
			es := store.(*eventStore)

			projector := StateProjector{
				Query: NewQuerySimple([]Tag{}, "CourseCreated"), // No tags
			}

			event := Event{
				Type: "CourseCreated",
				Tags: []Tag{{Key: "course_id", Value: "c1"}},
			}

			matches := es.eventMatchesProjector(event, projector)
			Expect(matches).To(BeTrue())
		})
	})

	Describe("buildAppendConditionFromProjectors", func() {
		It("should build append condition from projector queries", func() {
			es := store.(*eventStore)

			projectors := []BatchProjector{
				{ID: "projector1", StateProjector: StateProjector{
					Query: NewQuerySimple(NewTags("course_id", "c1"), "CourseCreated"),
				}},
				{ID: "projector2", StateProjector: StateProjector{
					Query: NewQuerySimple(NewTags("student_id", "s1"), "StudentRegistered"),
				}},
			}

			appendCondition := es.buildAppendConditionFromProjectors(projectors)

			Expect(appendCondition.FailIfEventsMatch).NotTo(BeNil())
			Expect(appendCondition.FailIfEventsMatch.Items).To(HaveLen(2))
			Expect(appendCondition.After).To(BeNil()) // Will be set during processing
		})

		It("should handle empty projectors list", func() {
			es := store.(*eventStore)

			appendCondition := es.buildAppendConditionFromProjectors([]BatchProjector{})

			Expect(appendCondition.FailIfEventsMatch).NotTo(BeNil())
			Expect(appendCondition.FailIfEventsMatch.Items).To(BeEmpty())
		})
	})

	Describe("ProjectDecisionModel with complex scenarios", func() {
		It("should handle multiple projectors with overlapping queries", func() {
			// Create test events
			event1 := NewInputEvent("CourseCreated", NewTags("course_id", "c1"), toJSON(map[string]string{"name": "Math 101"}))
			event2 := NewInputEvent("StudentRegistered", NewTags("student_id", "s1"), toJSON(map[string]string{"name": "Alice"}))
			event3 := NewInputEvent("StudentEnrolled", NewTags("course_id", "c1", "student_id", "s1"), toJSON(map[string]string{"enrolled_at": "2024-01-01"}))
			event4 := NewInputEvent("StudentEnrolled", NewTags("course_id", "c1", "student_id", "s2"), toJSON(map[string]string{"enrolled_at": "2024-01-02"}))
			events := []InputEvent{event1, event2, event3, event4}

			_, err := store.Append(ctx, events, nil)
			Expect(err).NotTo(HaveOccurred())

			// Define projectors with overlapping queries
			projectors := []BatchProjector{
				{ID: "courseCount", StateProjector: StateProjector{
					Query:        NewQuerySimple(NewTags("course_id", "c1"), "CourseCreated"),
					InitialState: 0,
					TransitionFn: func(state any, event Event) any {
						return state.(int) + 1
					},
				}},
				{ID: "studentCount", StateProjector: StateProjector{
					Query:        NewQuerySimple(NewTags("student_id", "s1"), "StudentRegistered"),
					InitialState: 0,
					TransitionFn: func(state any, event Event) any {
						return state.(int) + 1
					},
				}},
				{ID: "enrollmentCount", StateProjector: StateProjector{
					Query:        NewQuerySimple(NewTags("course_id", "c1"), "StudentEnrolled"),
					InitialState: 0,
					TransitionFn: func(state any, event Event) any {
						return state.(int) + 1
					},
				}},
				{ID: "totalEvents", StateProjector: StateProjector{
					Query:        NewQueryAll(),
					InitialState: 0,
					TransitionFn: func(state any, event Event) any {
						return state.(int) + 1
					},
				}},
			}

			// Test ProjectDecisionModel
			states, appendCondition, err := store.ProjectDecisionModel(ctx, projectors, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(appendCondition.After).NotTo(BeNil())

			Expect(states["courseCount"]).To(Equal(1))
			Expect(states["studentCount"]).To(Equal(1))
			Expect(states["enrollmentCount"]).To(Equal(2))
			Expect(states["totalEvents"]).To(Equal(4))
		})

		It("should handle projectors with different initial states", func() {
			// Setup test data
			event1 := NewInputEvent("MoneyTransferred", NewTags("account_id", "acc1"), toJSON(map[string]string{"amount": "100"}))
			event2 := NewInputEvent("MoneyTransferred", NewTags("account_id", "acc1"), toJSON(map[string]string{"amount": "50"}))
			events := []InputEvent{event1, event2}

			_, err := store.Append(ctx, events, nil)
			Expect(err).NotTo(HaveOccurred())

			// Define projectors with different initial states
			projectors := []BatchProjector{
				{ID: "count", StateProjector: StateProjector{
					Query:        NewQuerySimple(NewTags("account_id", "acc1"), "MoneyTransferred"),
					InitialState: 0,
					TransitionFn: func(state any, event Event) any {
						return state.(int) + 1
					},
				}},
				{ID: "balance", StateProjector: StateProjector{
					Query:        NewQuerySimple(NewTags("account_id", "acc1"), "MoneyTransferred"),
					InitialState: 1000.0, // Starting balance
					TransitionFn: func(state any, event Event) any {
						var data map[string]interface{}
						json.Unmarshal(event.Data, &data)
						amount := data["amount"].(string)
						// Parse amount (simplified for test)
						if amount == "100" {
							return state.(float64) + 100
						}
						return state.(float64) + 50
					},
				}},
			}

			states, _, err := store.ProjectDecisionModel(ctx, projectors, nil)
			Expect(err).NotTo(HaveOccurred())

			Expect(states["count"]).To(Equal(2))
			Expect(states["balance"]).To(Equal(1150.0))
		})

		It("should handle projectors with complex state transitions", func() {
			// Create test events
			event1 := NewInputEvent("MoneyTransferred", NewTags("account_id", "acc1"), toJSON(map[string]string{"amount": "100"}))
			event2 := NewInputEvent("MoneyTransferred", NewTags("account_id", "acc1"), toJSON(map[string]string{"amount": "50"}))
			events := []InputEvent{event1, event2}

			_, err := store.Append(ctx, events, nil)
			Expect(err).NotTo(HaveOccurred())

			// Define projector with complex state
			projectors := []BatchProjector{
				{ID: "totalAmount", StateProjector: StateProjector{
					Query:        NewQuerySimple(NewTags("account_id", "acc1"), "MoneyTransferred"),
					InitialState: 0.0,
					TransitionFn: func(state any, event Event) any {
						currentAmount := state.(float64)
						var data map[string]interface{}
						json.Unmarshal(event.Data, &data)
						amountStr := data["amount"].(string)
						amount, _ := strconv.ParseFloat(amountStr, 64)
						return currentAmount + amount
					},
				}},
			}

			// Test ProjectDecisionModel
			states, appendCondition, err := store.ProjectDecisionModel(ctx, projectors, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(appendCondition.After).NotTo(BeNil())

			Expect(states["totalAmount"]).To(Equal(150.0))
		})

		It("should handle projectors with nil transition function", func() {
			projectors := []BatchProjector{
				{ID: "invalid", StateProjector: StateProjector{
					Query:        NewQuerySimple(NewTags("test", "value"), "TestEvent"),
					InitialState: 0,
					TransitionFn: nil, // Nil transition function
				}},
			}

			_, _, err := store.ProjectDecisionModel(ctx, projectors, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("nil transition function"))
		})

		It("should handle projectors with empty query", func() {
			projectors := []BatchProjector{
				{ID: "empty", StateProjector: StateProjector{
					Query:        NewQueryEmpty(), // Empty query
					InitialState: 0,
					TransitionFn: func(state any, event Event) any {
						return state.(int) + 1
					},
				}},
			}

			_, _, err := store.ProjectDecisionModel(ctx, projectors, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("query must contain at least one item"))
		})

		It("should handle projectors with different query types", func() {
			// Create test events
			event1 := NewInputEvent("OrderCreated", NewTags("order_id", "order1"), toJSON(map[string]string{"total": "100"}))
			event2 := NewInputEvent("ItemAdded", NewTags("order_id", "order1"), toJSON(map[string]string{"item": "book", "price": "25"}))
			event3 := NewInputEvent("ItemAdded", NewTags("order_id", "order1"), toJSON(map[string]string{"item": "pen", "price": "5"}))
			event4 := NewInputEvent("OrderCompleted", NewTags("order_id", "order1"), toJSON(map[string]string{"status": "completed"}))
			events := []InputEvent{event1, event2, event3, event4}

			_, err := store.Append(ctx, events, nil)
			Expect(err).NotTo(HaveOccurred())

			// Define projectors with different query types
			projectors := []BatchProjector{
				{ID: "orderCount", StateProjector: StateProjector{
					Query:        NewQuerySimple(NewTags("order_id", "order1"), "OrderCreated"),
					InitialState: 0,
					TransitionFn: func(state any, event Event) any {
						return state.(int) + 1
					},
				}},
				{ID: "itemCount", StateProjector: StateProjector{
					Query:        NewQuerySimple(NewTags("order_id", "order1"), "ItemAdded"),
					InitialState: 0,
					TransitionFn: func(state any, event Event) any {
						return state.(int) + 1
					},
				}},
				{ID: "completionCount", StateProjector: StateProjector{
					Query:        NewQuerySimple(NewTags("order_id", "order1"), "OrderCompleted"),
					InitialState: 0,
					TransitionFn: func(state any, event Event) any {
						return state.(int) + 1
					},
				}},
			}

			// Test ProjectDecisionModel
			states, appendCondition, err := store.ProjectDecisionModel(ctx, projectors, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(appendCondition.After).NotTo(BeNil())

			Expect(states["orderCount"]).To(Equal(1))
			Expect(states["itemCount"]).To(Equal(2))
			Expect(states["completionCount"]).To(Equal(1))
		})
	})

	Describe("Performance with large datasets", func() {
		It("should handle large number of events efficiently", func() {
			// Create large dataset
			events := make([]InputEvent, 1000)
			for i := 0; i < 1000; i++ {
				event := NewInputEvent("TestEvent", NewTags("test", "value"), toJSON(map[string]string{"index": fmt.Sprintf("%d", i)}))
				events[i] = event
			}

			_, err := store.Append(ctx, events, nil)
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

			// Test with cursor streaming
			options := &ReadOptions{BatchSize: intPtr(100)}
			states, appendCondition, err := store.ProjectDecisionModel(ctx, projectors, options)
			Expect(err).NotTo(HaveOccurred())
			Expect(appendCondition.After).NotTo(BeNil())

			Expect(states["count"]).To(Equal(1000))
		})
	})
})
