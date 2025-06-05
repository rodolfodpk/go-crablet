package dcb_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go-crablet/internal/dcb"
)

var _ = Describe("AppendEvents", func() {
	BeforeEach(func() {
		// Truncate the events table and reset sequences before each test
		_, err := pool.Exec(ctx, "TRUNCATE TABLE events RESTART IDENTITY CASCADE")
		Expect(err).NotTo(HaveOccurred())
	})

	It("appends events successfully", func() {
		tags := dcb.NewTags("course_id", "course1")
		query := dcb.NewQuery(tags)
		event := dcb.NewInputEvent("Subscription", tags, []byte(`{"foo":"bar"}`))
		events := []dcb.InputEvent{event}
		pos, err := store.AppendEvents(ctx, events, query, 0)
		Expect(err).NotTo(HaveOccurred())
		Expect(pos).To(Equal(int64(1)))

		projector := dcb.StateProjector{
			InitialState: 0,
			TransitionFn: func(state any, e dcb.Event) any {
				return state.(int) + 1
			},
		}
		readPos, state, err := store.ProjectState(ctx, query, projector)
		Expect(err).NotTo(HaveOccurred())
		Expect(readPos).To(Equal(int64(1)))
		Expect(state).To(Equal(1))

		dumpEvents(pool)
	})

	It("appends events with multiple tags", func() {
		tags := dcb.NewTags("course_id", "course1", "user_id", "user123", "action", "enroll")
		query := dcb.NewQuery(dcb.NewTags("course_id", "course1"))
		events := []dcb.InputEvent{
			dcb.NewInputEvent("Enrollment", tags, []byte(`{"action":"enrolled"}`)),
		}

		pos, err := store.AppendEvents(ctx, events, query, 0)
		Expect(err).NotTo(HaveOccurred())
		Expect(pos).To(Equal(int64(1)))
		// Query by different tag combinations
		queryByUser := dcb.NewQuery(dcb.NewTags("user_id", "user123"))
		projector := dcb.StateProjector{
			InitialState: 0,
			TransitionFn: func(state any, e dcb.Event) any { return state.(int) + 1 },
		}

		_, state, err := store.ProjectState(ctx, queryByUser, projector)
		Expect(err).NotTo(HaveOccurred())
		Expect(state).To(Equal(1))

		dumpEvents(pool)
	})

	It("appends multiple events in a batch", func() {
		tags := dcb.NewTags("course_id", "course2")
		query := dcb.NewQuery(tags)
		events := []dcb.InputEvent{
			dcb.NewInputEvent("CourseLaunched", tags, []byte(`{"title":"Go Programming"}`)),
			dcb.NewInputEvent("LessonAdded", tags, []byte(`{"lesson_id":"L1"}`)),
			dcb.NewInputEvent("LessonAdded", tags, []byte(`{"lesson_id":"L2"}`)),
		}

		pos, err := store.AppendEvents(ctx, events, query, 0)
		Expect(err).NotTo(HaveOccurred())
		Expect(pos).To(Equal(int64(3)))

		projector := dcb.StateProjector{
			InitialState: 0,
			TransitionFn: func(state any, e dcb.Event) any { return state.(int) + 1 },
		}
		_, state, err := store.ProjectState(ctx, query, projector)
		Expect(err).NotTo(HaveOccurred())
		Expect(state).To(Equal(3))

		dumpEvents(pool)
	})

	It("fails with concurrency error when position is outdated", func() {
		tags := dcb.NewTags("course_id", "course3")
		query := dcb.NewQuery(tags)

		// First append - will succeed
		_, err := store.AppendEvents(ctx, []dcb.InputEvent{
			dcb.NewInputEvent("Initial", tags, []byte(`{"status":"first"}`)),
		}, query, 0)
		Expect(err).NotTo(HaveOccurred())

		// Second append with outdated position - should fail
		_, err = store.AppendEvents(ctx, []dcb.InputEvent{
			dcb.NewInputEvent("Second", tags, []byte(`{"status":"second"}`)),
		}, query, 0) // Using 0 again when it should be 1

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Consistency"))

		dumpEvents(pool)
	})

	It("properly sets causation and correlation IDs", func() {
		tags := dcb.NewTags("entity_id", "E1")
		query := dcb.NewQuery(tags)
		events := []dcb.InputEvent{
			dcb.NewInputEvent("EntityRegistered", tags, []byte(`{"initial":true}`)),
			dcb.NewInputEvent("EntityAttributeChanged", tags, []byte(`{"step":1}`)),
			dcb.NewInputEvent("EntityAttributeChanged", tags, []byte(`{"step":2}`)),
		}

		pos, err := store.AppendEvents(ctx, events, query, 0)
		Expect(err).NotTo(HaveOccurred())
		Expect(pos).To(Equal(int64(3)))

		// Check that causation and correlation IDs were set correctly
		dumpEvents(pool)

		// Custom projector to check causation and correlation IDs
		type EventRelationships struct {
			Count          int
			FirstID        string
			CausationIDs   []string
			CorrelationIDs []string
		}

		relationshipprojector := dcb.StateProjector{
			InitialState: EventRelationships{Count: 0},
			TransitionFn: func(state any, e dcb.Event) any {
				s := state.(EventRelationships)
				s.Count++
				if s.Count == 1 {
					s.FirstID = e.ID
				}
				s.CausationIDs = append(s.CausationIDs, e.CausationID)
				s.CorrelationIDs = append(s.CorrelationIDs, e.CorrelationID)
				return s
			},
		}

		_, state, err := store.ProjectState(ctx, query, relationshipprojector)
		Expect(err).NotTo(HaveOccurred())
		relationships := state.(EventRelationships)

		// First event is self-caused
		Expect(relationships.CausationIDs[0]).To(Equal(relationships.FirstID))

		// All events have same correlation ID (the first event's ID)
		for _, cid := range relationships.CorrelationIDs {
			Expect(cid).To(Equal(relationships.FirstID))
		}

		// Later events are caused by their predecessors
		Expect(relationships.CausationIDs[1]).To(Equal(relationships.FirstID))
		Expect(relationships.CausationIDs[2]).NotTo(Equal(relationships.FirstID))
	})

	Describe("Error scenarios", func() {
		It("returns error when appending events with empty tags", func() {
			tags := dcb.NewTags() // Empty tags
			query := dcb.NewQuery(dcb.NewTags("course_id", "C1"))
			events := []dcb.InputEvent{
				dcb.NewInputEvent("Subscription", tags, []byte(`{"foo":"bar"}`)),
			}
			_, err := store.AppendEvents(ctx, events, query, 0)
			Expect(err).To(HaveOccurred())
		})

		It("returns error when appending invalid JSON data", func() {
			tags := dcb.NewTags("course_id", "C1")
			query := dcb.NewQuery(tags)
			events := []dcb.InputEvent{
				dcb.NewInputEvent("Subscription", tags, []byte(`not-json`)),
			}
			_, err := store.AppendEvents(ctx, events, query, 0)
			Expect(err).To(HaveOccurred())
		})
	})
})
