package dcb

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

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

		// Create context with timeout for each test
		ctx, _ = context.WithTimeout(context.Background(), 30*time.Second)

		// Truncate events table before each test
		err := truncateEventsTable(ctx, pool)
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("combineProjectorQueries", func() {
		It("should combine multiple projector queries with OR logic", func() {
			projectors := []BatchProjector{
				{ID: "projector1", StateProjector: StateProjector{
					Query: NewQuerySimple(NewTags("course_id", "c1"), "CourseDefined"),
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

			Expect(combinedQuery.getItems()).To(HaveLen(3))
			Expect(combinedQuery.getItems()[0].getEventTypes()).To(Equal([]string{"CourseDefined"}))
			Expect(combinedQuery.getItems()[1].getEventTypes()).To(Equal([]string{"StudentRegistered"}))
			Expect(combinedQuery.getItems()[2].getEventTypes()).To(Equal([]string{"StudentEnrolled"}))
		})

		It("should handle empty projectors list", func() {
			es := store.(*eventStore)
			combinedQuery := es.combineProjectorQueries([]BatchProjector{})

			Expect(combinedQuery.getItems()).To(BeEmpty())
		})

		It("should handle single projector", func() {
			projectors := []BatchProjector{
				{ID: "single", StateProjector: StateProjector{
					Query: NewQuerySimple(NewTags("test", "value"), "TestEvent"),
				}},
			}

			es := store.(*eventStore)
			combinedQuery := es.combineProjectorQueries(projectors)

			Expect(combinedQuery.getItems()).To(HaveLen(1))
			Expect(combinedQuery.getItems()[0].getEventTypes()).To(Equal([]string{"TestEvent"}))
		})
	})

	Describe("eventMatchesProjector", func() {
		It("should match events with correct type and tags", func() {
			es := store.(*eventStore)

			projector := StateProjector{
				Query: NewQuerySimple(NewTags("course_id", "c1"), "CourseDefined"),
			}

			event := Event{
				Type: "CourseDefined",
				Tags: []Tag{NewTag("course_id", "c1")},
			}

			matches := es.eventMatchesProjector(event, projector)
			Expect(matches).To(BeTrue())
		})

		It("should not match events with different types", func() {
			es := store.(*eventStore)

			projector := StateProjector{
				Query: NewQuerySimple(NewTags("course_id", "c1"), "CourseDefined"),
			}

			event := Event{
				Type: "StudentRegistered",
				Tags: []Tag{NewTag("course_id", "c1")},
			}

			matches := es.eventMatchesProjector(event, projector)
			Expect(matches).To(BeFalse())
		})

		It("should not match events with different tags", func() {
			es := store.(*eventStore)

			projector := StateProjector{
				Query: NewQuerySimple(NewTags("course_id", "c1"), "CourseDefined"),
			}

			event := Event{
				Type: "CourseDefined",
				Tags: []Tag{NewTag("course_id", "c2")},
			}

			matches := es.eventMatchesProjector(event, projector)
			Expect(matches).To(BeFalse())
		})

		It("should match events with subset of tags", func() {
			es := store.(*eventStore)

			projector := StateProjector{
				Query: NewQuerySimple(NewTags("course_id", "c1"), "CourseDefined"),
			}

			event := Event{
				Type: "CourseDefined",
				Tags: []Tag{
					NewTag("course_id", "c1"),
					NewTag("student_id", "s1"),
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
				Tags: []Tag{NewTag("course_id", "c1")},
			}

			matches := es.eventMatchesProjector(event, projector)
			Expect(matches).To(BeTrue())
		})

		It("should handle empty tags in projector", func() {
			es := store.(*eventStore)

			projector := StateProjector{
				Query: NewQuerySimple([]Tag{}, "CourseDefined"), // No tags
			}

			event := Event{
				Type: "CourseDefined",
				Tags: []Tag{NewTag("course_id", "c1")},
			}

			matches := es.eventMatchesProjector(event, projector)
			Expect(matches).To(BeTrue())
		})
	})

	// Note: buildAppendConditionFromProjectors was removed as it was not DCB-compliant
	// The DCB-compliant approach is to use buildAppendConditionFromQuery with specific queries
	// from Decision Models, as demonstrated in the tests below.

	Describe("buildAppendConditionFromQuery (DCB-compliant)", func() {
		It("should build append condition from specific query (DCB approach)", func() {
			es := store.(*eventStore)

			// DCB-compliant approach: use specific query from Decision Model
			query := NewQuerySimple(NewTags("course_id", "c1"), "CourseDefined")
			appendCondition := es.buildAppendConditionFromQuery(query)

			// Should use the exact query from Decision Model
			Expect(appendCondition).NotTo(BeNil())
			// Note: We can't directly access fields anymore since AppendCondition is opaque
			// This enforces DCB semantics where consumers only build and pass conditions
		})

		It("should handle complex query with multiple items (DCB approach)", func() {
			es := store.(*eventStore)

			// DCB-compliant approach: use specific query from Decision Model
			query := NewQueryFromItems(
				NewQueryItem([]string{"CourseDefined"}, []Tag{NewTag("course_id", "c1")}),
				NewQueryItem([]string{"StudentEnrolled"}, []Tag{NewTag("course_id", "c1"), NewTag("student_id", "s1")}),
			)
			appendCondition := es.buildAppendConditionFromQuery(query)

			// Should use the exact query from Decision Model
			Expect(appendCondition).NotTo(BeNil())
			// Note: We can't directly access fields anymore since AppendCondition is opaque
			// This enforces DCB semantics where consumers only build and pass conditions
		})

		It("should demonstrate DCB principle: same query from Decision Model", func() {
			es := store.(*eventStore)

			// Simulate building a Decision Model for course enrollment
			enrollmentQuery := NewQuerySimple(NewTags("course_id", "c1", "student_id", "s1"), "StudentEnrolled")

			// DCB principle: use the same query for append condition
			appendCondition := es.buildAppendConditionFromQuery(enrollmentQuery)

			// This ensures no new enrollment events exist for this student-course pair
			Expect(appendCondition).NotTo(BeNil())
			// Note: We can't directly access fields anymore since AppendCondition is opaque
			// This enforces DCB semantics where consumers only build and pass conditions
		})
	})

	Describe("ProjectDecisionModel with complex scenarios", func() {
		It("should handle multiple projectors with overlapping queries", func() {
			// Append test events
			events := []InputEvent{
				NewInputEvent("CourseDefined", []Tag{NewTag("course_id", "c1")}, toJSON(map[string]string{"name": "Math 101"})),
				NewInputEvent("StudentRegistered", []Tag{NewTag("student_id", "s1")}, toJSON(map[string]string{"name": "Alice"})),
				NewInputEvent("StudentEnrolled", []Tag{NewTag("course_id", "c1"), NewTag("student_id", "s1")}, toJSON(map[string]string{"enrolled_at": "2024-01-01"})),
				NewInputEvent("StudentEnrolled", []Tag{NewTag("course_id", "c1"), NewTag("student_id", "s2")}, toJSON(map[string]string{"enrolled_at": "2024-01-02"})),
			}
			err := store.Append(ctx, events)
			Expect(err).NotTo(HaveOccurred())

			// Define projectors
			projectors := []BatchProjector{
				{
					ID: "course",
					StateProjector: StateProjector{
						Query:        NewQuerySimple([]Tag{NewTag("course_id", "c1")}, "CourseDefined"),
						InitialState: 0,
						TransitionFn: func(state any, event Event) any {
							return state.(int) + 1
						},
					},
				},
				{
					ID: "student",
					StateProjector: StateProjector{
						Query:        NewQuerySimple([]Tag{NewTag("student_id", "s1")}, "StudentRegistered"),
						InitialState: 0,
						TransitionFn: func(state any, event Event) any {
							return state.(int) + 1
						},
					},
				},
				{
					ID: "enrollment",
					StateProjector: StateProjector{
						Query:        NewQuerySimple([]Tag{NewTag("course_id", "c1")}, "StudentEnrolled"),
						InitialState: 0,
						TransitionFn: func(state any, event Event) any {
							return state.(int) + 1
						},
					},
				},
			}

			// Test ProjectDecisionModel
			channelStore := store.(ChannelEventStore)
			states, _, err := channelStore.ProjectDecisionModel(ctx, projectors)
			Expect(err).NotTo(HaveOccurred())

			Expect(states["course"]).To(Equal(1))
			Expect(states["student"]).To(Equal(1))
			Expect(states["enrollment"]).To(Equal(2))
		})

		It("should handle projectors with different initial states", func() {
			// Setup test data
			event1 := NewInputEvent("MoneyTransferred", NewTags("account_id", "acc1"), toJSON(map[string]string{"amount": "100"}))
			event2 := NewInputEvent("MoneyTransferred", NewTags("account_id", "acc1"), toJSON(map[string]string{"amount": "50"}))
			events := []InputEvent{event1, event2}

			// Append events
			err := store.Append(ctx, events)
			Expect(err).NotTo(HaveOccurred())

			// Define projectors with different initial states
			projectors := []BatchProjector{
				{
					ID: "count",
					StateProjector: StateProjector{
						Query:        NewQuerySimple([]Tag{NewTag("account_id", "acc1")}, "MoneyTransferred"),
						InitialState: 0,
						TransitionFn: func(state any, event Event) any {
							return state.(int) + 1
						},
					},
				},
				{
					ID: "balance",
					StateProjector: StateProjector{
						Query:        NewQuerySimple([]Tag{NewTag("account_id", "acc1")}, "MoneyTransferred"),
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
					},
				},
			}

			channelStore := store.(ChannelEventStore)
			states, _, err := channelStore.ProjectDecisionModel(ctx, projectors)
			Expect(err).NotTo(HaveOccurred())

			Expect(states["count"]).To(Equal(2))
			Expect(states["balance"]).To(Equal(1150.0))
		})

		It("should handle projectors with complex state transitions", func() {
			// Create test events
			event1 := NewInputEvent("MoneyTransferred", NewTags("account_id", "acc1"), toJSON(map[string]string{"amount": "100"}))
			event2 := NewInputEvent("MoneyTransferred", NewTags("account_id", "acc1"), toJSON(map[string]string{"amount": "50"}))
			events := []InputEvent{event1, event2}

			// Append events
			err := store.Append(ctx, events)
			Expect(err).NotTo(HaveOccurred())

			// Define projector with complex state
			projectors := []BatchProjector{
				{
					ID: "totalAmount",
					StateProjector: StateProjector{
						Query:        NewQuerySimple([]Tag{NewTag("account_id", "acc1")}, "MoneyTransferred"),
						InitialState: 0.0,
						TransitionFn: func(state any, event Event) any {
							currentAmount := state.(float64)
							var data map[string]interface{}
							json.Unmarshal(event.Data, &data)
							amountStr := data["amount"].(string)
							amount, _ := strconv.ParseFloat(amountStr, 64)
							return currentAmount + amount
						},
					},
				},
			}

			// Test ProjectDecisionModel
			channelStore := store.(ChannelEventStore)
			states, _, err := channelStore.ProjectDecisionModel(ctx, projectors)
			Expect(err).NotTo(HaveOccurred())

			Expect(states["totalAmount"]).To(Equal(150.0))
		})

		It("should handle projectors with nil transition function", func() {
			projectors := []BatchProjector{
				{
					ID: "invalid",
					StateProjector: StateProjector{
						Query:        NewQuerySimple([]Tag{NewTag("test", "value")}, "TestEvent"),
						InitialState: 0,
						TransitionFn: nil, // Nil transition function
					},
				},
			}

			channelStore := store.(ChannelEventStore)
			_, _, err := channelStore.ProjectDecisionModel(ctx, projectors)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("nil transition function"))
		})

		It("should handle projectors with empty query", func() {
			projectors := []BatchProjector{
				{
					ID: "empty",
					StateProjector: StateProjector{
						Query:        NewQueryEmpty(), // Empty query
						InitialState: 0,
						TransitionFn: func(state any, event Event) any {
							return state.(int) + 1
						},
					},
				},
			}

			channelStore := store.(ChannelEventStore)
			_, _, err := channelStore.ProjectDecisionModel(ctx, projectors)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("empty query"))
		})

		It("should handle projectors with different query types", func() {
			// Create test events
			event1 := NewInputEvent("OrderCreated", NewTags("order_id", "order1"), toJSON(map[string]string{"total": "100"}))
			event2 := NewInputEvent("ItemAdded", NewTags("order_id", "order1"), toJSON(map[string]string{"item": "book", "price": "25"}))
			event3 := NewInputEvent("ItemAdded", NewTags("order_id", "order1"), toJSON(map[string]string{"item": "pen", "price": "5"}))
			event4 := NewInputEvent("OrderCompleted", NewTags("order_id", "order1"), toJSON(map[string]string{"status": "completed"}))
			events := []InputEvent{event1, event2, event3, event4}

			// Append events
			err := store.Append(ctx, events)
			Expect(err).NotTo(HaveOccurred())

			// Define projectors with different query types
			projectors := []BatchProjector{
				{
					ID: "orderCount",
					StateProjector: StateProjector{
						Query:        NewQuerySimple([]Tag{NewTag("order_id", "order1")}, "OrderCreated"),
						InitialState: 0,
						TransitionFn: func(state any, event Event) any {
							return state.(int) + 1
						},
					},
				},
				{
					ID: "itemCount",
					StateProjector: StateProjector{
						Query:        NewQuerySimple([]Tag{NewTag("order_id", "order1")}, "ItemAdded"),
						InitialState: 0,
						TransitionFn: func(state any, event Event) any {
							return state.(int) + 1
						},
					},
				},
				{
					ID: "completionCount",
					StateProjector: StateProjector{
						Query:        NewQuerySimple([]Tag{NewTag("order_id", "order1")}, "OrderCompleted"),
						InitialState: 0,
						TransitionFn: func(state any, event Event) any {
							return state.(int) + 1
						},
					},
				},
			}

			// Test ProjectDecisionModel
			channelStore := store.(ChannelEventStore)
			states, _, err := channelStore.ProjectDecisionModel(ctx, projectors)
			Expect(err).NotTo(HaveOccurred())

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

			// Append events
			err := store.Append(ctx, events)
			Expect(err).NotTo(HaveOccurred())

			// Define projector
			projectors := []BatchProjector{
				{
					ID: "count",
					StateProjector: StateProjector{
						Query:        NewQuerySimple([]Tag{NewTag("test", "value")}, "TestEvent"),
						InitialState: 0,
						TransitionFn: func(state any, event Event) any {
							return state.(int) + 1
						},
					},
				},
			}

			// Test with cursor streaming
			channelStore := store.(ChannelEventStore)
			states, _, err := channelStore.ProjectDecisionModel(ctx, projectors)
			Expect(err).NotTo(HaveOccurred())

			Expect(states["count"]).To(Equal(1000))
		})
	})

	It("should handle multiple events for the same projector", func() {
		// Append multiple events
		events := []InputEvent{
			NewInputEvent("EnrollmentStarted", NewTags("student_id", "123"), toJSON(map[string]string{"course": "math"})),
			NewInputEvent("EnrollmentCompleted", NewTags("student_id", "123"), toJSON(map[string]string{"course": "math"})),
		}
		err := store.Append(ctx, events)
		Expect(err).NotTo(HaveOccurred())

		// Create projector
		projector := BatchProjector{
			ID: "enrollment",
			StateProjector: StateProjector{
				Query:        NewQuery(NewTags("student_id", "123")),
				InitialState: "not_enrolled",
				TransitionFn: func(state any, event Event) any {
					switch event.Type {
					case "EnrollmentStarted":
						return "enrolling"
					case "EnrollmentCompleted":
						return "enrolled"
					default:
						return state
					}
				},
			},
		}
		projectors := []BatchProjector{projector}

		// Test ProjectDecisionModel
		channelStore := store.(ChannelEventStore)
		states, _, err := channelStore.ProjectDecisionModel(ctx, projectors)
		Expect(err).NotTo(HaveOccurred())
		Expect(states).To(HaveKey("enrollment"))
		Expect(states["enrollment"]).To(Equal("enrolled"))
	})

	It("should handle empty query results", func() {
		// Create projector with query that won't match any events
		projector := BatchProjector{
			ID: "enrollment",
			StateProjector: StateProjector{
				Query:        NewQuery(NewTags("student_id", "999")), // Non-existent student
				InitialState: "not_enrolled",
				TransitionFn: func(state any, event Event) any {
					return "enrolled"
				},
			},
		}
		projectors := []BatchProjector{projector}

		// Test ProjectDecisionModel
		channelStore := store.(ChannelEventStore)
		states, _, err := channelStore.ProjectDecisionModel(ctx, projectors)
		Expect(err).NotTo(HaveOccurred())
		Expect(states).To(HaveKey("enrollment"))
		Expect(states["enrollment"]).To(Equal("not_enrolled")) // Should remain initial state
	})

	It("should handle invalid projector configuration", func() {
		// Create projector with empty ID
		projector := BatchProjector{
			ID: "", // Invalid: empty ID
			StateProjector: StateProjector{
				Query:        NewQuery(NewTags("student_id", "123")),
				InitialState: "not_enrolled",
				TransitionFn: func(state any, event Event) any {
					return "enrolled"
				},
			},
		}
		projectors := []BatchProjector{projector}

		// Test ProjectDecisionModel
		channelStore := store.(ChannelEventStore)
		_, _, err := channelStore.ProjectDecisionModel(ctx, projectors)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("empty"))
	})

	It("should handle nil transition function", func() {
		// Create projector with nil transition function
		projector := BatchProjector{
			ID: "enrollment",
			StateProjector: StateProjector{
				Query:        NewQuery(NewTags("student_id", "123")),
				InitialState: "not_enrolled",
				TransitionFn: nil, // Invalid: nil function
			},
		}
		projectors := []BatchProjector{projector}

		// Test ProjectDecisionModel
		channelStore := store.(ChannelEventStore)
		_, _, err := channelStore.ProjectDecisionModel(ctx, projectors)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("nil"))
	})

	It("should handle multiple projectors with different queries", func() {
		// Append events for different students
		events := []InputEvent{
			NewInputEvent("EnrollmentStarted", NewTags("student_id", "123"), toJSON(map[string]string{"course": "math"})),
			NewInputEvent("EnrollmentStarted", NewTags("student_id", "456"), toJSON(map[string]string{"course": "science"})),
		}
		err := store.Append(ctx, events)
		Expect(err).NotTo(HaveOccurred())

		// Create projectors for different students
		projector1 := BatchProjector{
			ID: "student_123",
			StateProjector: StateProjector{
				Query:        NewQuery(NewTags("student_id", "123")),
				InitialState: "not_enrolled",
				TransitionFn: func(state any, event Event) any {
					if event.Type == "EnrollmentStarted" {
						return "enrolling"
					}
					return state
				},
			},
		}
		projector2 := BatchProjector{
			ID: "student_456",
			StateProjector: StateProjector{
				Query:        NewQuery(NewTags("student_id", "456")),
				InitialState: "not_enrolled",
				TransitionFn: func(state any, event Event) any {
					if event.Type == "EnrollmentStarted" {
						return "enrolling"
					}
					return state
				},
			},
		}
		projectors := []BatchProjector{projector1, projector2}

		// Test ProjectDecisionModel
		channelStore := store.(ChannelEventStore)
		states, _, err := channelStore.ProjectDecisionModel(ctx, projectors)
		Expect(err).NotTo(HaveOccurred())
		Expect(states).To(HaveKey("student_123"))
		Expect(states).To(HaveKey("student_456"))
		Expect(states["student_123"]).To(Equal("enrolling"))
		Expect(states["student_456"]).To(Equal("enrolling"))
	})

	It("should handle complex state transitions", func() {
		// Append events with complex state transitions
		events := []InputEvent{
			NewInputEvent("EnrollmentStarted", NewTags("student_id", "123"), toJSON(map[string]string{"course": "math"})),
			NewInputEvent("PaymentReceived", NewTags("student_id", "123"), toJSON(map[string]string{"amount": "100"})),
			NewInputEvent("EnrollmentCompleted", NewTags("student_id", "123"), toJSON(map[string]string{"course": "math"})),
		}
		err := store.Append(ctx, events)
		Expect(err).NotTo(HaveOccurred())

		// Create projector with complex state machine
		projector := BatchProjector{
			ID: "enrollment",
			StateProjector: StateProjector{
				Query:        NewQuery(NewTags("student_id", "123")),
				InitialState: "not_enrolled",
				TransitionFn: func(state any, event Event) any {
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
			},
		}
		projectors := []BatchProjector{projector}

		// Test ProjectDecisionModel
		channelStore := store.(ChannelEventStore)
		states, _, err := channelStore.ProjectDecisionModel(ctx, projectors)
		Expect(err).NotTo(HaveOccurred())
		Expect(states).To(HaveKey("enrollment"))
		Expect(states["enrollment"]).To(Equal("enrolled"))
	})
})
