package dcb

import (
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
			Query:        NewQuery(NewTags()),
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
			Query:        NewQuery(NewTags("course_id", "course101")),
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
			Query:        NewQuery(NewTags()),
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
})
