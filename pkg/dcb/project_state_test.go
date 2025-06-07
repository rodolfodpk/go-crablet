package dcb

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ProjectState", func() {
	BeforeEach(func() {

		// Truncate the events table before each test
		err := truncateEventsTable(ctx, pool)
		Expect(err).NotTo(HaveOccurred())

		// Set up some test events for all ProjectState tests
		courseTags := NewTags("course_id", "course101")
		userTags := NewTags("user_id", "user101")
		mixedTags := NewTags("course_id", "course101", "user_id", "user101")

		query := NewQuery(courseTags)

		// Insert different event types with different tag combinations
		events := []InputEvent{
			NewInputEvent("CourseLaunched", courseTags, []byte(`{"title":"Test Course"}`)),
			NewInputEvent("UserRegistered", userTags, []byte(`{"name":"Test User"}`)),
			NewInputEvent("Enrollment", mixedTags, []byte(`{"status":"active"}`)),
			NewInputEvent("CourseUpdated", courseTags, []byte(`{"title":"Updated Course"}`)),
		}

		pos, err := store.AppendEvents(ctx, events, query, 0)
		Expect(err).NotTo(HaveOccurred())
		Expect(pos).To(Equal(int64(4)))
	})

	It("reads state with empty tags in query", func() {
		projector := StateProjector{
			Query:        NewQuery(NewTags(), "CourseLaunched", "UserRegistered", "Enrollment", "CourseUpdated"),
			InitialState: 0,
			TransitionFn: func(state any, e Event) any { return state.(int) + 1 },
		}

		_, state, err := store.ProjectState(ctx, projector)
		// Should return all events since no tag filtering is applied
		Expect(err).NotTo(HaveOccurred())
		Expect(state).To(Equal(4)) // All 4 events should be read
	})

	It("reads state with specific tags but empty eventTypes", func() {
		projector := StateProjector{
			Query:        NewQuery(NewTags("course_id", "course101"), "CourseLaunched", "Enrollment", "CourseUpdated"),
			InitialState: 0,
			TransitionFn: func(state any, e Event) any { return state.(int) + 1 },
		}

		_, state, err := store.ProjectState(ctx, projector)
		Expect(err).NotTo(HaveOccurred())
		Expect(state).To(Equal(3)) // Should match CourseLaunched, Enrollment, and CourseUpdated
	})

	It("reads state with empty tags but specific eventTypes", func() {
		projector := StateProjector{
			Query:        NewQuery(NewTags(), "CourseLaunched", "CourseUpdated"),
			InitialState: 0,
			TransitionFn: func(state any, e Event) any { return state.(int) + 1 },
		}

		_, state, err := store.ProjectState(ctx, projector)
		Expect(err).NotTo(HaveOccurred())
		Expect(state).To(Equal(2)) // Should match only CourseLaunched and CourseUpdated
	})

	It("reads state with both empty tags and empty eventTypes", func() {
		projector := StateProjector{
			Query:        NewQuery(NewTags(), "CourseLaunched", "UserRegistered", "Enrollment", "CourseUpdated"),
			InitialState: 0,
			TransitionFn: func(state any, e Event) any { return state.(int) + 1 },
		}

		_, state, err := store.ProjectState(ctx, projector)
		Expect(err).NotTo(HaveOccurred())
		Expect(state).To(Equal(4)) // Should match all events
	})

	It("reads state with both specific tags and specific eventTypes", func() {
		projector := StateProjector{
			Query:        NewQuery(NewTags("course_id", "course101"), "CourseLaunched"),
			InitialState: 0,
			TransitionFn: func(state any, e Event) any { return state.(int) + 1 },
		}

		_, state, err := store.ProjectState(ctx, projector)
		Expect(err).NotTo(HaveOccurred())
		Expect(state).To(Equal(1)) // Should match only CourseLaunched event
	})

	It("reads state with tags that don't match any events", func() {
		projector := StateProjector{
			Query:        NewQuery(NewTags("nonexistent_tag", "value")),
			InitialState: 0,
			TransitionFn: func(state any, e Event) any { return state.(int) + 1 },
		}

		_, state, err := store.ProjectState(ctx, projector)
		Expect(err).NotTo(HaveOccurred())
		Expect(state).To(Equal(0)) // Should not match any events
	})

	It("reads state with event types that don't match any events", func() {
		projector := StateProjector{
			Query:        NewQuery(NewTags(), "NonExistentEventType"),
			InitialState: 0,
			TransitionFn: func(state any, e Event) any { return state.(int) + 1 },
		}

		_, state, err := store.ProjectState(ctx, projector)
		Expect(err).NotTo(HaveOccurred())
		Expect(state).To(Equal(0)) // Should not match any events
	})

	It("uses projector's query when available", func() {
		// Create a projector with its own query
		projector := StateProjector{
			Query:        NewQuery(NewTags("course_id", "course101"), "CourseLaunched"),
			InitialState: 0,
			TransitionFn: func(state any, e Event) any { return state.(int) + 1 },
		}

		_, state, err := store.ProjectState(ctx, projector)
		Expect(err).NotTo(HaveOccurred())
		Expect(state).To(Equal(1)) // Should only match CourseLaunched event
	})

	It("falls back to provided query when projector's query is empty", func() {
		// Create a projector with empty query
		projector := StateProjector{
			Query:        NewQuery(NewTags()), // Empty query
			InitialState: 0,
			TransitionFn: func(state any, e Event) any { return state.(int) + 1 },
		}

		_, state, err := store.ProjectState(ctx, projector)
		Expect(err).NotTo(HaveOccurred())
		Expect(state).To(Equal(4)) // Should match all events
	})

	var _ = Describe("ProjectStateUpTo", func() {
		BeforeEach(func() {
			// Truncate the events table before each test
			_, err := pool.Exec(ctx, "TRUNCATE TABLE events RESTART IDENTITY CASCADE")
			Expect(err).NotTo(HaveOccurred())

			// Set up sequential events for position testing
			tags := NewTags("sequence_id", "seq1")
			query := NewQuery(tags)
			events := []InputEvent{
				NewInputEvent("Event1", tags, []byte(`{"order":1}`)),
				NewInputEvent("Event2", tags, []byte(`{"order":2}`)),
				NewInputEvent("Event3", tags, []byte(`{"order":3}`)),
				NewInputEvent("Event4", tags, []byte(`{"order":4}`)),
				NewInputEvent("Event5", tags, []byte(`{"order":5}`)),
			}

			pos, err := store.AppendEvents(ctx, events, query, 0)
			Expect(err).NotTo(HaveOccurred())
			Expect(pos).To(Equal(int64(5)))
		})

		It("reads state up to a specific position limit", func() {
			projector := StateProjector{
				Query:        NewQuery(NewTags("sequence_id", "seq1")),
				InitialState: 0,
				TransitionFn: func(state any, e Event) any {
					return state.(int) + 1
				},
			}

			// Read up to position 3 (should include events at positions 1, 2, and 3)
			pos, state, err := store.ProjectStateUpTo(ctx, projector, 3)
			Expect(err).NotTo(HaveOccurred())
			Expect(pos).To(Equal(int64(3)))
			Expect(state).To(Equal(3))

			// Read all events (maxPosition = -1)
			pos, state, err = store.ProjectStateUpTo(ctx, projector, -1)
			Expect(err).NotTo(HaveOccurred())
			Expect(pos).To(Equal(int64(5)))
			Expect(state).To(Equal(5))

			// Read up to position 0 (should find no events)
			pos, state, err = store.ProjectStateUpTo(ctx, projector, 0)
			Expect(err).NotTo(HaveOccurred())
			Expect(pos).To(Equal(int64(0)))
			Expect(state).To(Equal(0))
		})

		It("reads state with position beyond available maximum", func() {
			projector := StateProjector{
				Query:        NewQuery(NewTags("sequence_id", "seq1")),
				InitialState: 0,
				TransitionFn: func(state any, e Event) any {
					return state.(int) + 1
				},
			}

			// Request position 100, which is beyond our max of 5
			pos, state, err := store.ProjectStateUpTo(ctx, projector, 100)
			Expect(err).NotTo(HaveOccurred())
			Expect(pos).To(Equal(int64(5))) // Should return the actual max position
			Expect(state).To(Equal(5))      // All 5 events should be counted
		})

		It("combines position limits with event type filtering", func() {
			projector := StateProjector{
				Query:        NewQuery(NewTags("sequence_id", "seq1"), "Event2", "Event4"),
				InitialState: 0,
				TransitionFn: func(state any, e Event) any {
					return state.(int) + 1
				},
			}

			// Read up to position 4 with event type filtering
			pos, state, err := store.ProjectStateUpTo(ctx, projector, 4)
			Expect(err).NotTo(HaveOccurred())
			Expect(pos).To(Equal(int64(4)))
			Expect(state).To(Equal(2)) // Should only count Event2 and Event4
		})

		It("reads with tags that partially match events", func() {
			// Setup additional events with mixed tags
			partialTags1 := NewTags("sequence_id", "seq2", "extra", "value1")
			partialTags2 := NewTags("sequence_id", "seq2", "extra", "value2")

			extraEvents := []InputEvent{
				NewInputEvent("ExtraEvent1", partialTags1, []byte(`{"extra":1}`)),
				NewInputEvent("ExtraEvent2", partialTags2, []byte(`{"extra":2}`)),
			}

			query := NewQuery(NewTags("sequence_id", "seq2"))
			_, err := store.AppendEvents(ctx, extraEvents, query, 0)
			Expect(err).NotTo(HaveOccurred())

			projector := StateProjector{
				Query:        NewQuery(NewTags("extra", "value1")),
				InitialState: 0,
				TransitionFn: func(state any, e Event) any {
					return state.(int) + 1
				},
			}

			// Read all events with the partial tag match
			_, state, err := store.ProjectStateUpTo(ctx, projector, -1)
			Expect(err).NotTo(HaveOccurred())
			Expect(state).To(Equal(1)) // Should only match ExtraEvent1
		})

		It("combines projector's query with position limits", func() {
			// Create a projector with its own query
			projector := StateProjector{
				Query:        NewQuery(NewTags("sequence_id", "seq1")),
				InitialState: 0,
				TransitionFn: func(state any, e Event) any { return state.(int) + 1 },
			}

			// Read up to position 3 using projector's query
			pos, state, err := store.ProjectStateUpTo(ctx, projector, 3)
			Expect(err).NotTo(HaveOccurred())
			Expect(pos).To(Equal(int64(3)))
			Expect(state).To(Equal(3)) // Should count events up to position 3
		})
	})

	It("handles complex projector with multiple event types and tags", func() {
		// Set up test events with different combinations of tags and types
		courseTags := NewTags("course_id", "course_test_1", "category", "programming")
		userTags := NewTags("user_id", "user_test_1", "role", "student", "course_id", "course_test_1")
		mixedTags := NewTags("course_id", "course_test_1", "user_id", "user_test_1", "category", "programming", "role", "student")

		query := NewQuery(NewTags("test_id", "test_1"), "CourseCreated", "UserRegistered", "EnrollmentStarted", "EnrollmentCompleted", "CourseUpdated", "UserProfileUpdated")

		events := []InputEvent{
			NewInputEvent("CourseCreated", courseTags, []byte(`{"title":"Go Programming"}`)),
			NewInputEvent("UserRegistered", userTags, []byte(`{"name":"Alice"}`)),
			NewInputEvent("EnrollmentStarted", mixedTags, []byte(`{"status":"pending"}`)),
			NewInputEvent("EnrollmentCompleted", mixedTags, []byte(`{"status":"active"}`)),
			NewInputEvent("CourseUpdated", courseTags, []byte(`{"title":"Advanced Go"}`)),
			NewInputEvent("UserProfileUpdated", userTags, []byte(`{"level":"intermediate"}`)),
		}

		// Use a unique query to avoid conflicts with existing events
		_, err := store.AppendEvents(ctx, events, query, 0)
		Expect(err).NotTo(HaveOccurred())

		// Define a complex state type to track course and user interactions
		type CourseUserState struct {
			CourseTitle      string
			UserName         string
			EnrollmentStatus string
			UserLevel        string
			EventCount       int
		}

		// Create a projector that tracks both course and user events
		projector := StateProjector{
			Query: NewQuery(
				NewTags("course_id", "course_test_1"),
				"CourseCreated", "CourseUpdated", "UserRegistered", "UserProfileUpdated",
				"EnrollmentStarted", "EnrollmentCompleted",
			),
			InitialState: &CourseUserState{},
			TransitionFn: func(state any, e Event) any {
				s := state.(*CourseUserState)
				s.EventCount++

				var data map[string]string
				_ = json.Unmarshal(e.Data, &data)

				switch e.Type {
				case "CourseCreated", "CourseUpdated":
					s.CourseTitle = data["title"]
				case "UserRegistered":
					s.UserName = data["name"]
				case "UserProfileUpdated":
					s.UserLevel = data["level"]
				case "EnrollmentStarted", "EnrollmentCompleted":
					s.EnrollmentStatus = data["status"]
				}
				return s
			},
		}

		_, state, err := store.ProjectState(ctx, projector)
		Expect(err).NotTo(HaveOccurred())
		finalState := state.(*CourseUserState)

		// Verify the final state reflects all relevant events
		Expect(finalState.CourseTitle).To(Equal("Advanced Go"))
		Expect(finalState.UserName).To(Equal("Alice"))
		Expect(finalState.UserLevel).To(Equal("intermediate"))
		Expect(finalState.EnrollmentStatus).To(Equal("active"))
		Expect(finalState.EventCount).To(Equal(6)) // All events matching course_id
	})

	It("handles projector with partial tag matches", func() {
		// Set up events with overlapping but different tag combinations
		baseTags := NewTags("tenant_id", "tenant_test_1")
		userTags := NewTags("tenant_id", "tenant_test_1", "user_id", "user_test_1")
		orderTags := NewTags("tenant_id", "tenant_test_1", "order_id", "order_test_1")
		mixedTags := NewTags("tenant_id", "tenant_test_1", "user_id", "user_test_1", "order_id", "order_test_1")

		query := NewQuery(NewTags("test_id", "test_2"), "TenantCreated", "UserRegistered", "OrderCreated", "OrderAssigned")

		events := []InputEvent{
			NewInputEvent("TenantCreated", baseTags, []byte(`{"name":"Tenant 1"}`)),
			NewInputEvent("UserRegistered", userTags, []byte(`{"name":"John"}`)),
			NewInputEvent("OrderCreated", orderTags, []byte(`{"amount":100}`)),
			NewInputEvent("OrderAssigned", mixedTags, []byte(`{"status":"assigned"}`)),
		}

		// Use a unique query to avoid conflicts with existing events
		_, err := store.AppendEvents(ctx, events, query, 0)
		Expect(err).NotTo(HaveOccurred())

		// Create a projector that tracks all events for a tenant
		type TenantState struct {
			UserCount      int
			OrderCount     int
			AssignedOrders int
			EventTypes     []string
		}

		projector := StateProjector{
			Query:        NewQuery(NewTags("tenant_id", "tenant_test_1")),
			InitialState: &TenantState{},
			TransitionFn: func(state any, e Event) any {
				s := state.(*TenantState)
				s.EventTypes = append(s.EventTypes, e.Type)

				// Check for user_id tag
				for _, tag := range e.Tags {
					if tag.Key == "user_id" {
						s.UserCount++
					}
					if tag.Key == "order_id" {
						s.OrderCount++
					}
				}

				// Check for assigned orders
				if e.Type == "OrderAssigned" {
					s.AssignedOrders++
				}

				return s
			},
		}

		_, state, err := store.ProjectState(ctx, projector)
		Expect(err).NotTo(HaveOccurred())
		finalState := state.(*TenantState)

		// Verify the state reflects all tenant events
		Expect(finalState.UserCount).To(Equal(2))  // UserRegistered and OrderAssigned
		Expect(finalState.OrderCount).To(Equal(2)) // OrderCreated and OrderAssigned
		Expect(finalState.AssignedOrders).To(Equal(1))
		Expect(finalState.EventTypes).To(HaveLen(4))
		Expect(finalState.EventTypes).To(ContainElements(
			"TenantCreated",
			"UserRegistered",
			"OrderCreated",
			"OrderAssigned",
		))
	})

	It("handles projector with event type filtering and complex state transitions", func() {
		// Set up a sequence of events representing a workflow
		workflowTags := NewTags("workflow_id", "workflow_test_1", "status", "active")
		query := NewQuery(NewTags("test_id", "test_3"), "WorkflowStarted", "TaskAssigned", "TaskCompleted", "TaskFailed", "TaskRetried", "WorkflowCompleted")

		events := []InputEvent{
			NewInputEvent("WorkflowStarted", workflowTags, []byte(`{"step":1}`)),
			NewInputEvent("TaskAssigned", workflowTags, []byte(`{"task":"A"}`)),
			NewInputEvent("TaskCompleted", workflowTags, []byte(`{"task":"A"}`)),
			NewInputEvent("TaskAssigned", workflowTags, []byte(`{"task":"B"}`)),
			NewInputEvent("TaskFailed", workflowTags, []byte(`{"task":"B","error":"timeout"}`)),
			NewInputEvent("TaskRetried", workflowTags, []byte(`{"task":"B"}`)),
			NewInputEvent("TaskCompleted", workflowTags, []byte(`{"task":"B"}`)),
			NewInputEvent("WorkflowCompleted", workflowTags, []byte(`{"step":2}`)),
		}

		// Use a unique query to avoid conflicts with existing events
		_, err := store.AppendEvents(ctx, events, query, 0)
		Expect(err).NotTo(HaveOccurred())

		// Define a state type to track workflow progress
		type WorkflowState struct {
			CurrentStep    int
			CompletedTasks []string
			FailedTasks    map[string]string
			RetryCount     map[string]int
			IsComplete     bool
		}

		// Create a projector that only processes task-related events
		projector := StateProjector{
			Query: NewQuery(
				NewTags("workflow_id", "workflow_test_1"),
				"TaskAssigned", "TaskCompleted", "TaskFailed", "TaskRetried",
			),
			InitialState: &WorkflowState{
				FailedTasks: make(map[string]string),
				RetryCount:  make(map[string]int),
			},
			TransitionFn: func(state any, e Event) any {
				s := state.(*WorkflowState)
				var data map[string]string
				_ = json.Unmarshal(e.Data, &data)
				taskID := data["task"]

				switch e.Type {
				case "TaskAssigned":
					// No state changes needed
				case "TaskCompleted":
					s.CompletedTasks = append(s.CompletedTasks, taskID)
					delete(s.FailedTasks, taskID)
				case "TaskFailed":
					s.FailedTasks[taskID] = data["error"]
				case "TaskRetried":
					s.RetryCount[taskID]++
					delete(s.FailedTasks, taskID)
				}
				return s
			},
		}

		_, state, err := store.ProjectState(ctx, projector)
		Expect(err).NotTo(HaveOccurred())
		finalState := state.(*WorkflowState)

		// Verify the final state reflects the workflow progress
		Expect(finalState.CompletedTasks).To(HaveLen(2))
		Expect(finalState.CompletedTasks).To(ContainElements("A", "B"))
		Expect(finalState.FailedTasks).To(BeEmpty())
		Expect(finalState.RetryCount).To(HaveKeyWithValue("B", 1))
	})
})
