package dcb

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Event Store: Idempotent Event Appending", func() {
	BeforeEach(func() {
		err := truncateEventsTable(ctx, pool)
		Expect(err).NotTo(HaveOccurred())
	})

	It("appends events with idempotency check", func() {
		tags := NewTags("course_id", "course1")
		query := NewQuery(tags, "Subscription")
		event := NewInputEvent("Subscription", tags, []byte(`{"foo":"bar"}`))
		events := []InputEvent{event}

		// First append
		pos, err := store.AppendEvents(ctx, events, query, 0)
		Expect(err).NotTo(HaveOccurred())
		Expect(pos).To(Equal(int64(1)))

		// Try to append the same events again with the current position
		// This should fail with a concurrency error since we're using the wrong position
		_, err = store.AppendEvents(ctx, events, query, 0)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Consistency"))

		// Verify only one event exists
		projector := StateProjector{
			Query:        query,
			InitialState: 0,
			TransitionFn: func(state any, e Event) any {
				return state.(int) + 1
			},
		}
		readPos, state, err := store.ProjectState(ctx, projector)
		Expect(err).NotTo(HaveOccurred())
		Expect(readPos).To(Equal(int64(1)))
		Expect(state).To(Equal(1))

		dumpEvents(pool)
	})

	It("handles multiple events in a batch with idempotency", func() {
		tags := NewTags("course_id", "course2")
		query := NewQuery(tags, "CourseLaunched", "LessonAdded")
		events := []InputEvent{
			NewInputEvent("CourseLaunched", tags, []byte(`{"title":"Go Programming"}`)),
			NewInputEvent("LessonAdded", tags, []byte(`{"lesson_id":"L1"}`)),
			NewInputEvent("LessonAdded", tags, []byte(`{"lesson_id":"L2"}`)),
		}

		// First append
		pos, err := store.AppendEvents(ctx, events, query, 0)
		Expect(err).NotTo(HaveOccurred())
		Expect(pos).To(Equal(int64(3)))

		// Try to append the same events again with the current position
		// This should fail with a concurrency error
		_, err = store.AppendEvents(ctx, events, query, 0)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Consistency"))

		// Verify only the original events exist
		projector := StateProjector{
			Query:        query,
			InitialState: 0,
			TransitionFn: func(state any, e Event) any {
				return state.(int) + 1
			},
		}
		readPos, state, err := store.ProjectState(ctx, projector)
		Expect(err).NotTo(HaveOccurred())
		Expect(readPos).To(Equal(int64(3)))
		Expect(state).To(Equal(3))

		dumpEvents(pool)
	})

	It("handles partial event appending with idempotency", func() {
		tags := NewTags("course_id", "course3")
		query := NewQuery(tags, "CourseLaunched", "LessonAdded")
		initialEvents := []InputEvent{
			NewInputEvent("CourseLaunched", tags, []byte(`{"title":"Go Programming"}`)),
			NewInputEvent("LessonAdded", tags, []byte(`{"lesson_id":"L1"}`)),
		}

		// First append
		pos, err := store.AppendEvents(ctx, initialEvents, query, 0)
		Expect(err).NotTo(HaveOccurred())
		Expect(pos).To(Equal(int64(2)))

		// Try to append a mix of existing and new events
		newEvents := []InputEvent{
			NewInputEvent("CourseLaunched", tags, []byte(`{"title":"Go Programming"}`)), // Existing
			NewInputEvent("LessonAdded", tags, []byte(`{"lesson_id":"L2"}`)),            // New
		}
		// This should fail with a concurrency error since we're using the wrong position
		_, err = store.AppendEvents(ctx, newEvents, query, 0)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Consistency"))

		// Append only the new event with the correct position
		_, err = store.AppendEvents(ctx, []InputEvent{newEvents[1]}, query, pos)
		Expect(err).NotTo(HaveOccurred())

		// Verify we have all three events
		projector := StateProjector{
			Query:        query,
			InitialState: 0,
			TransitionFn: func(state any, e Event) any {
				return state.(int) + 1
			},
		}
		readPos, state, err := store.ProjectState(ctx, projector)
		Expect(err).NotTo(HaveOccurred())
		Expect(readPos).To(Equal(int64(3)))
		Expect(state).To(Equal(3))

		dumpEvents(pool)
	})

	It("handles empty event list", func() {
		tags := NewTags("course_id", "course4")
		query := NewQuery(tags)
		events := []InputEvent{}

		pos, err := store.AppendEvents(ctx, events, query, 0)
		Expect(err).NotTo(HaveOccurred())
		Expect(pos).To(Equal(int64(0))) // No events appended

		projector := StateProjector{
			Query:        query,
			InitialState: 0,
			TransitionFn: func(state any, e Event) any {
				return state.(int) + 1
			},
		}
		readPos, state, err := store.ProjectState(ctx, projector)
		Expect(err).NotTo(HaveOccurred())
		Expect(readPos).To(Equal(int64(0)))
		Expect(state).To(Equal(0)) // No events processed

		dumpEvents(pool)
	})
})
