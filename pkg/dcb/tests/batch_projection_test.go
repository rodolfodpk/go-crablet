package dcb

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"go-crablet/pkg/dcb"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Batch Projection", func() {
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

	Describe("CombineProjectorQueries", func() {
		It("should combine multiple projector queries with OR logic", func() {
			projectors := []dcb.StateProjector{
				{ID: "projector1",
					Query: dcb.NewQuery(dcb.NewTags("course_id", "c1"), "CourseDefined"),
				},
				{ID: "projector2",
					Query: dcb.NewQuery(dcb.NewTags("student_id", "s1"), "StudentRegistered"),
				},
				{ID: "projector3",
					Query: dcb.NewQuery(dcb.NewTags("course_id", "c1", "student_id", "s1"), "StudentEnrolled"),
				},
			}

			combinedQuery := dcb.CombineProjectorQueries(projectors)

			// The optimization may merge items with same tags, so we check the total count
			// and that all expected event types are present
			items := combinedQuery.GetItems()
			Expect(items).To(HaveLen(3))

			// Collect all event types from all items
			var allEventTypes []string
			for _, item := range items {
				allEventTypes = append(allEventTypes, item.GetEventTypes()...)
			}

			// Check that all expected event types are present (order doesn't matter)
			Expect(allEventTypes).To(ContainElements("CourseDefined", "StudentRegistered", "StudentEnrolled"))
		})

		It("should handle empty projectors list", func() {
			combinedQuery := dcb.CombineProjectorQueries([]dcb.StateProjector{})

			Expect(combinedQuery.GetItems()).To(BeEmpty())
		})

		It("should handle single projector", func() {
			projectors := []dcb.StateProjector{
				{ID: "single",
					Query: dcb.NewQuery(dcb.NewTags("test", "value"), "TestEvent"),
				},
			}

			combinedQuery := dcb.CombineProjectorQueries(projectors)

			Expect(combinedQuery.GetItems()).To(HaveLen(1))
			Expect(combinedQuery.GetItems()[0].GetEventTypes()).To(Equal([]string{"TestEvent"}))
		})
	})

	Describe("EventMatchesProjector", func() {
		It("should match events with correct type and tags", func() {
			projector := dcb.StateProjector{
				Query: dcb.NewQuery(dcb.NewTags("course_id", "c1"), "CourseDefined"),
			}

			event := dcb.Event{
				Type: "CourseDefined",
				Tags: []dcb.Tag{dcb.NewTag("course_id", "c1")},
			}

			matches := dcb.EventMatchesProjector(event, projector)
			Expect(matches).To(BeTrue())
		})

		It("should not match events with different types", func() {
			projector := dcb.StateProjector{
				Query: dcb.NewQuery(dcb.NewTags("course_id", "c1"), "CourseDefined"),
			}

			event := dcb.Event{
				Type: "StudentRegistered",
				Tags: []dcb.Tag{dcb.NewTag("course_id", "c1")},
			}

			matches := dcb.EventMatchesProjector(event, projector)
			Expect(matches).To(BeFalse())
		})

		It("should not match events with different tags", func() {
			projector := dcb.StateProjector{
				Query: dcb.NewQuery(dcb.NewTags("course_id", "c1"), "CourseDefined"),
			}

			event := dcb.Event{
				Type: "CourseDefined",
				Tags: []dcb.Tag{dcb.NewTag("course_id", "c2")},
			}

			matches := dcb.EventMatchesProjector(event, projector)
			Expect(matches).To(BeFalse())
		})

		It("should match events with subset of tags", func() {
			projector := dcb.StateProjector{
				Query: dcb.NewQuery(dcb.NewTags("course_id", "c1"), "CourseDefined"),
			}

			event := dcb.Event{
				Type: "CourseDefined",
				Tags: []dcb.Tag{
					dcb.NewTag("course_id", "c1"),
					dcb.NewTag("student_id", "s1"),
				},
			}

			matches := dcb.EventMatchesProjector(event, projector)
			Expect(matches).To(BeTrue())
		})

		It("should handle empty event types in projector", func() {
			projector := dcb.StateProjector{
				Query: dcb.NewQuery(dcb.NewTags("course_id", "c1")), // No event types
			}

			event := dcb.Event{
				Type: "AnyEvent",
				Tags: []dcb.Tag{dcb.NewTag("course_id", "c1")},
			}

			matches := dcb.EventMatchesProjector(event, projector)
			Expect(matches).To(BeTrue())
		})

		It("should handle empty tags in projector", func() {
			projector := dcb.StateProjector{
				Query: dcb.NewQuery([]dcb.Tag{}, "CourseDefined"), // No tags
			}

			event := dcb.Event{
				Type: "CourseDefined",
				Tags: []dcb.Tag{dcb.NewTag("course_id", "c1")},
			}

			matches := dcb.EventMatchesProjector(event, projector)
			Expect(matches).To(BeTrue())
		})
	})

	Describe("BuildAppendConditionFromQuery (DCB-compliant)", func() {
		It("should build append condition from specific query (DCB approach)", func() {
			// DCB-compliant approach: use specific query from Decision Model
			query := dcb.NewQuery(dcb.NewTags("course_id", "c1"), "CourseDefined")
			appendCondition := dcb.BuildAppendConditionFromQuery(query)

			// Should use the exact query from Decision Model
			Expect(appendCondition).NotTo(BeNil())
		})

		It("should build append condition from enrollment query", func() {
			// DCB-compliant approach: use specific query from Decision Model
			query := dcb.NewQuery(dcb.NewTags("student_id", "s1"), "StudentRegistered")
			appendCondition := dcb.BuildAppendConditionFromQuery(query)

			Expect(appendCondition).NotTo(BeNil())
		})

		It("should build append condition from complex query", func() {
			// DCB-compliant approach: use specific query from Decision Model
			enrollmentQuery := dcb.NewQuery(dcb.NewTags("course_id", "c1", "student_id", "s1"), "StudentEnrolled")
			appendCondition := dcb.BuildAppendConditionFromQuery(enrollmentQuery)

			Expect(appendCondition).NotTo(BeNil())
		})
	})

	Describe("Project with complex scenarios", func() {
		It("should handle multiple projectors with overlapping queries", func() {
			// Append test events
			events := []dcb.InputEvent{
				dcb.NewInputEvent("CourseDefined", []dcb.Tag{dcb.NewTag("course_id", "c1")}, dcb.ToJSON(map[string]string{"name": "Math 101"})),
				dcb.NewInputEvent("StudentRegistered", []dcb.Tag{dcb.NewTag("student_id", "s1")}, dcb.ToJSON(map[string]string{"name": "Alice"})),
				dcb.NewInputEvent("StudentEnrolled", []dcb.Tag{dcb.NewTag("course_id", "c1"), dcb.NewTag("student_id", "s1")}, dcb.ToJSON(map[string]string{"enrolled_at": "2024-01-01"})),
				dcb.NewInputEvent("StudentEnrolled", []dcb.Tag{dcb.NewTag("course_id", "c1"), dcb.NewTag("student_id", "s2")}, dcb.ToJSON(map[string]string{"enrolled_at": "2024-01-02"})),
			}
			err := store.Append(ctx, events, nil)
			Expect(err).To(BeNil())

			// Define projectors
			projectors := []dcb.StateProjector{
				{
					ID:           "course",
					Query:        dcb.NewQuery([]dcb.Tag{dcb.NewTag("course_id", "c1")}, "CourseDefined"),
					InitialState: 0,
					TransitionFn: func(state any, event dcb.Event) any {
						return state.(int) + 1
					},
				},
				{
					ID:           "student",
					Query:        dcb.NewQuery([]dcb.Tag{dcb.NewTag("student_id", "s1")}, "StudentRegistered"),
					InitialState: 0,
					TransitionFn: func(state any, event dcb.Event) any {
						return state.(int) + 1
					},
				},
				{
					ID:           "enrollment",
					Query:        dcb.NewQuery([]dcb.Tag{dcb.NewTag("course_id", "c1")}, "StudentEnrolled"),
					InitialState: 0,
					TransitionFn: func(state any, event dcb.Event) any {
						return state.(int) + 1
					},
				},
			}

			// Test Project
			states, _, err := store.Project(ctx, projectors, nil)
			Expect(err).NotTo(HaveOccurred())

			Expect(states["course"]).To(Equal(1))
			Expect(states["student"]).To(Equal(1))
			Expect(states["enrollment"]).To(Equal(2))
		})

		It("should handle projectors with different initial states", func() {
			// Setup test data
			event1 := dcb.NewInputEvent("MoneyTransferred", dcb.NewTags("account_id", "acc1"), dcb.ToJSON(map[string]string{"amount": "100"}))
			event2 := dcb.NewInputEvent("MoneyTransferred", dcb.NewTags("account_id", "acc1"), dcb.ToJSON(map[string]string{"amount": "50"}))
			events := []dcb.InputEvent{event1, event2}

			// Append events
			err := store.Append(ctx, events, nil)
			Expect(err).To(BeNil())

			// Define projectors with different initial states
			projectors := []dcb.StateProjector{
				{
					ID:           "count",
					Query:        dcb.NewQuery([]dcb.Tag{dcb.NewTag("account_id", "acc1")}, "MoneyTransferred"),
					InitialState: 0,
					TransitionFn: func(state any, event dcb.Event) any {
						return state.(int) + 1
					},
				},
				{
					ID:           "balance",
					Query:        dcb.NewQuery([]dcb.Tag{dcb.NewTag("account_id", "acc1")}, "MoneyTransferred"),
					InitialState: 1000.0, // Starting balance
					TransitionFn: func(state any, event dcb.Event) any {
						var data map[string]interface{}
						json.Unmarshal(event.Data, &data)
						amount := data["amount"].(string)
						// Parse amount (simplified for test)
						if amount == "100" {
							return state.(float64) + 100
						}
						return state.(float64) + 50
					},
				},
			}

			states, _, err := store.Project(ctx, projectors, nil)
			Expect(err).NotTo(HaveOccurred())

			Expect(states["count"]).To(Equal(2))
			Expect(states["balance"]).To(Equal(1150.0))
		})

		It("should handle projectors with complex state transitions", func() {
			// Create test events
			event1 := dcb.NewInputEvent("MoneyTransferred", dcb.NewTags("account_id", "acc1"), dcb.ToJSON(map[string]string{"amount": "100"}))
			event2 := dcb.NewInputEvent("MoneyTransferred", dcb.NewTags("account_id", "acc1"), dcb.ToJSON(map[string]string{"amount": "50"}))
			events := []dcb.InputEvent{event1, event2}

			// Append events
			err := store.Append(ctx, events, nil)
			Expect(err).To(BeNil())

			// Define projector with complex state
			projectors := []dcb.StateProjector{
				{
					ID:           "totalAmount",
					Query:        dcb.NewQuery([]dcb.Tag{dcb.NewTag("account_id", "acc1")}, "MoneyTransferred"),
					InitialState: 0.0,
					TransitionFn: func(state any, event dcb.Event) any {
						currentAmount := state.(float64)
						var data map[string]interface{}
						json.Unmarshal(event.Data, &data)
						amountStr := data["amount"].(string)
						amount, _ := strconv.ParseFloat(amountStr, 64)
						return currentAmount + amount
					},
				},
			}

			// Test Project
			states, _, err := store.Project(ctx, projectors, nil)
			Expect(err).NotTo(HaveOccurred())

			Expect(states["totalAmount"]).To(Equal(150.0))
		})

		It("should handle projectors with nil transition function", func() {
			projectors := []dcb.StateProjector{
				{
					ID:           "invalid",
					Query:        dcb.NewQuery([]dcb.Tag{dcb.NewTag("test", "value")}, "TestEvent"),
					InitialState: 0,
					TransitionFn: nil, // Nil transition function
				},
			}

			_, _, err := store.Project(ctx, projectors, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("nil transition function"))
		})

		It("should handle projectors with empty query", func() {
			projectors := []dcb.StateProjector{
				{
					ID:           "empty",
					Query:        dcb.NewQueryEmpty(), // Empty query
					InitialState: 0,
					TransitionFn: func(state any, event dcb.Event) any {
						return state.(int) + 1
					},
				},
			}

			_, _, err := store.Project(ctx, projectors, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("empty query"))
		})

		It("should handle projectors with different query types", func() {
			// Create test events
			event1 := dcb.NewInputEvent("OrderCreated", dcb.NewTags("order_id", "order1"), dcb.ToJSON(map[string]string{"total": "100"}))
			event2 := dcb.NewInputEvent("ItemAdded", dcb.NewTags("order_id", "order1"), dcb.ToJSON(map[string]string{"item": "book", "price": "25"}))
			event3 := dcb.NewInputEvent("ItemAdded", dcb.NewTags("order_id", "order1"), dcb.ToJSON(map[string]string{"item": "pen", "price": "5"}))
			event4 := dcb.NewInputEvent("OrderCompleted", dcb.NewTags("order_id", "order1"), dcb.ToJSON(map[string]string{"status": "completed"}))
			events := []dcb.InputEvent{event1, event2, event3, event4}

			// Append events
			err := store.Append(ctx, events, nil)
			Expect(err).To(BeNil())

			// Define projectors with different query types
			projectors := []dcb.StateProjector{
				{
					ID:           "orderCount",
					Query:        dcb.NewQuery([]dcb.Tag{dcb.NewTag("order_id", "order1")}, "OrderCreated"),
					InitialState: 0,
					TransitionFn: func(state any, event dcb.Event) any {
						return state.(int) + 1
					},
				},
				{
					ID:           "itemCount",
					Query:        dcb.NewQuery([]dcb.Tag{dcb.NewTag("order_id", "order1")}, "ItemAdded"),
					InitialState: 0,
					TransitionFn: func(state any, event dcb.Event) any {
						return state.(int) + 1
					},
				},
				{
					ID:           "completionCount",
					Query:        dcb.NewQuery([]dcb.Tag{dcb.NewTag("order_id", "order1")}, "OrderCompleted"),
					InitialState: 0,
					TransitionFn: func(state any, event dcb.Event) any {
						return state.(int) + 1
					},
				},
			}

			// Test Project
			states, _, err := store.Project(ctx, projectors, nil)
			Expect(err).NotTo(HaveOccurred())

			Expect(states["orderCount"]).To(Equal(1))
			Expect(states["itemCount"]).To(Equal(2))
			Expect(states["completionCount"]).To(Equal(1))
		})
	})

	Describe("Performance with large datasets", func() {
		It("should handle large number of events efficiently", func() {
			// Create large dataset
			events := make([]dcb.InputEvent, 1000)
			for i := 0; i < 1000; i++ {
				event := dcb.NewInputEvent("TestEvent", dcb.NewTags("test", "value"), dcb.ToJSON(map[string]string{"index": fmt.Sprintf("%d", i)}))
				events[i] = event
			}

			// Append events
			err := store.Append(ctx, events, nil)
			Expect(err).To(BeNil())

			// Define projector
			projectors := []dcb.StateProjector{
				{
					ID:           "count",
					Query:        dcb.NewQuery([]dcb.Tag{dcb.NewTag("test", "value")}, "TestEvent"),
					InitialState: 0,
					TransitionFn: func(state any, event dcb.Event) any {
						return state.(int) + 1
					},
				},
			}

			// Test with cursor streaming
			states, _, err := store.Project(ctx, projectors, nil)
			Expect(err).NotTo(HaveOccurred())

			Expect(states["count"]).To(Equal(1000))
		})
	})

	It("should handle multiple events for the same projector", func() {
		// Append multiple events
		events := []dcb.InputEvent{
			dcb.NewInputEvent("EnrollmentStarted", dcb.NewTags("student_id", "123"), dcb.ToJSON(map[string]string{"course": "math"})),
			dcb.NewInputEvent("EnrollmentCompleted", dcb.NewTags("student_id", "123"), dcb.ToJSON(map[string]string{"course": "math"})),
		}
		err := store.Append(ctx, events, nil)
		Expect(err).To(BeNil())

		// Create projector
		projector := dcb.StateProjector{
			ID:           "enrollment",
			Query:        dcb.NewQuery(dcb.NewTags("student_id", "123")),
			InitialState: "not_enrolled",
			TransitionFn: func(state any, event dcb.Event) any {
				switch event.Type {
				case "EnrollmentStarted":
					return "enrolling"
				case "EnrollmentCompleted":
					return "enrolled"
				default:
					return state
				}
			},
		}
		projectors := []dcb.StateProjector{projector}

		// Test Project
		states, _, err := store.Project(ctx, projectors, nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(states).To(HaveKey("enrollment"))
		Expect(states["enrollment"]).To(Equal("enrolled"))
	})

	It("should handle empty query results", func() {
		// Create projector with query that won't match any events
		projector := dcb.StateProjector{
			ID:           "enrollment",
			Query:        dcb.NewQuery(dcb.NewTags("student_id", "999")), // Non-existent student
			InitialState: "not_enrolled",
			TransitionFn: func(state any, event dcb.Event) any {
				return "enrolled"
			},
		}
		projectors := []dcb.StateProjector{projector}

		// Test Project
		states, _, err := store.Project(ctx, projectors, nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(states).To(HaveKey("enrollment"))
		Expect(states["enrollment"]).To(Equal("not_enrolled")) // Should remain initial state
	})

	It("should handle invalid projector configuration", func() {
		// Create projector with empty ID
		projector := dcb.StateProjector{
			ID:           "", // Invalid: empty ID
			Query:        dcb.NewQuery(dcb.NewTags("student_id", "123")),
			InitialState: "not_enrolled",
			TransitionFn: func(state any, event dcb.Event) any {
				return "enrolled"
			},
		}
		projectors := []dcb.StateProjector{projector}

		// Test Project
		_, _, err := store.Project(ctx, projectors, nil)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("empty"))
	})

	It("should handle nil transition function", func() {
		// Create projector with nil transition function
		projector := dcb.StateProjector{
			ID:           "enrollment",
			Query:        dcb.NewQuery(dcb.NewTags("student_id", "123")),
			InitialState: "not_enrolled",
			TransitionFn: nil, // Invalid: nil function
		}
		projectors := []dcb.StateProjector{projector}

		// Test Project
		_, _, err := store.Project(ctx, projectors, nil)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("nil"))
	})

	It("should handle multiple projectors with different queries", func() {
		// Append events for different students
		events := []dcb.InputEvent{
			dcb.NewInputEvent("EnrollmentStarted", dcb.NewTags("student_id", "123"), dcb.ToJSON(map[string]string{"course": "math"})),
			dcb.NewInputEvent("EnrollmentStarted", dcb.NewTags("student_id", "456"), dcb.ToJSON(map[string]string{"course": "science"})),
		}
		err := store.Append(ctx, events, nil)
		Expect(err).To(BeNil())

		// Create projectors for different students
		projector1 := dcb.StateProjector{
			ID:           "student_123",
			Query:        dcb.NewQuery(dcb.NewTags("student_id", "123")),
			InitialState: "not_enrolled",
			TransitionFn: func(state any, event dcb.Event) any {
				if event.Type == "EnrollmentStarted" {
					return "enrolling"
				}
				return state
			},
		}
		projector2 := dcb.StateProjector{
			ID:           "student_456",
			Query:        dcb.NewQuery(dcb.NewTags("student_id", "456")),
			InitialState: "not_enrolled",
			TransitionFn: func(state any, event dcb.Event) any {
				if event.Type == "EnrollmentStarted" {
					return "enrolling"
				}
				return state
			},
		}
		projectors := []dcb.StateProjector{projector1, projector2}

		// Test Project
		states, _, err := store.Project(ctx, projectors, nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(states).To(HaveKey("student_123"))
		Expect(states).To(HaveKey("student_456"))
		Expect(states["student_123"]).To(Equal("enrolling"))
		Expect(states["student_456"]).To(Equal("enrolling"))
	})

	It("should handle complex state transitions", func() {
		// Append events with complex state transitions
		events := []dcb.InputEvent{
			dcb.NewInputEvent("EnrollmentStarted", dcb.NewTags("student_id", "123"), dcb.ToJSON(map[string]string{"course": "math"})),
			dcb.NewInputEvent("PaymentReceived", dcb.NewTags("student_id", "123"), dcb.ToJSON(map[string]string{"amount": "100"})),
			dcb.NewInputEvent("EnrollmentCompleted", dcb.NewTags("student_id", "123"), dcb.ToJSON(map[string]string{"course": "math"})),
		}
		err := store.Append(ctx, events, nil)
		Expect(err).To(BeNil())

		// Create projector with complex state machine
		projector := dcb.StateProjector{
			ID:           "enrollment",
			Query:        dcb.NewQuery(dcb.NewTags("student_id", "123")),
			InitialState: "not_enrolled",
			TransitionFn: func(state any, event dcb.Event) any {
				switch state {
				case "not_enrolled":
					if event.Type == "EnrollmentStarted" {
						return "enrolling"
					}
				case "enrolling":
					if event.Type == "PaymentReceived" {
						return "paid"
					}
				case "paid":
					if event.Type == "EnrollmentCompleted" {
						return "enrolled"
					}
				}
				return state
			},
		}
		projectors := []dcb.StateProjector{projector}

		// Test Project
		states, _, err := store.Project(ctx, projectors, nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(states).To(HaveKey("enrollment"))
		Expect(states["enrollment"]).To(Equal("enrolled"))
	})
})
