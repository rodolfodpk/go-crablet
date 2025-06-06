package dcb

import (
	"encoding/json"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Event Store: Appending Events", func() {
	BeforeEach(func() {
		err := truncateEventsTable(ctx, pool)
		Expect(err).NotTo(HaveOccurred())
	})

	It("appends events successfully", func() {
		tags := NewTags("course_id", "course1")
		query := NewQuery(tags, "Subscription")
		event := NewInputEvent("Subscription", tags, []byte(`{"foo":"bar"}`))
		events := []InputEvent{event}
		pos, err := store.AppendEvents(ctx, events, query, 0)
		Expect(err).NotTo(HaveOccurred())
		Expect(pos).To(Equal(int64(1)))

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

	It("appends events with multiple tags", func() {
		tags := NewTags("course_id", "course1", "user_id", "user123", "action", "enroll")
		query := NewQuery(NewTags("course_id", "course1"), "Enrollment")
		events := []InputEvent{
			NewInputEvent("Enrollment", tags, []byte(`{"action":"enrolled"}`)),
		}

		pos, err := store.AppendEvents(ctx, events, query, 0)
		Expect(err).NotTo(HaveOccurred())
		Expect(pos).To(Equal(int64(1)))
		// Query by different tag combinations
		projector := StateProjector{
			Query:        NewQuery(NewTags("user_id", "user123"), "Enrollment"),
			InitialState: 0,
			TransitionFn: func(state any, e Event) any { return state.(int) + 1 },
		}

		_, state, err := store.ProjectState(ctx, projector)
		Expect(err).NotTo(HaveOccurred())
		Expect(state).To(Equal(1))

		dumpEvents(pool)
	})

	It("appends multiple events in a batch", func() {
		tags := NewTags("course_id", "course2")
		query := NewQuery(tags, "CourseLaunched", "LessonAdded")
		events := []InputEvent{
			NewInputEvent("CourseLaunched", tags, []byte(`{"title":"Go Programming"}`)),
			NewInputEvent("LessonAdded", tags, []byte(`{"lesson_id":"L1"}`)),
			NewInputEvent("LessonAdded", tags, []byte(`{"lesson_id":"L2"}`)),
		}

		pos, err := store.AppendEvents(ctx, events, query, 0)
		Expect(err).NotTo(HaveOccurred())
		Expect(pos).To(Equal(int64(3)))

		projector := StateProjector{
			Query:        query,
			InitialState: 0,
			TransitionFn: func(state any, e Event) any { return state.(int) + 1 },
		}
		_, state, err := store.ProjectState(ctx, projector)
		Expect(err).NotTo(HaveOccurred())
		Expect(state).To(Equal(3))

		dumpEvents(pool)
	})

	It("fails with concurrency error when position is outdated", func() {
		tags := NewTags("course_id", "course3")
		query := NewQuery(tags, "Initial", "Second")

		// First append - will succeed
		_, err := store.AppendEvents(ctx, []InputEvent{
			NewInputEvent("Initial", tags, []byte(`{"status":"first"}`)),
		}, query, 0)
		Expect(err).NotTo(HaveOccurred())

		// Second append with outdated position - should fail
		_, err = store.AppendEvents(ctx, []InputEvent{
			NewInputEvent("Second", tags, []byte(`{"status":"second"}`)),
		}, query, 0) // Using 0 again when it should be 1

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Consistency"))

		dumpEvents(pool)
	})

	It("properly sets causation and correlation IDs", func() {
		tags := NewTags("entity_id", "E1")
		query := NewQuery(tags, "EntityRegistered", "EntityAttributeChanged")
		events := []InputEvent{
			NewInputEvent("EntityRegistered", tags, []byte(`{"initial":true}`)),
			NewInputEvent("EntityAttributeChanged", tags, []byte(`{"step":1}`)),
			NewInputEvent("EntityAttributeChanged", tags, []byte(`{"step":2}`)),
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

		relationshipprojector := StateProjector{
			Query:        query,
			InitialState: EventRelationships{Count: 0},
			TransitionFn: func(state any, e Event) any {
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

		_, state, err := store.ProjectState(ctx, relationshipprojector)
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
			tags := NewTags() // Empty tags
			query := NewQuery(NewTags("course_id", "C1"))
			events := []InputEvent{
				NewInputEvent("Subscription", tags, []byte(`{"foo":"bar"}`)),
			}
			_, err := store.AppendEvents(ctx, events, query, 0)
			Expect(err).To(HaveOccurred())
		})

		It("returns error when appending invalid JSON data", func() {
			tags := NewTags("course_id", "C1")
			query := NewQuery(tags)
			events := []InputEvent{
				NewInputEvent("Subscription", tags, []byte(`not-json`)),
			}
			_, err := store.AppendEvents(ctx, events, query, 0)
			Expect(err).To(HaveOccurred())
		})
	})

	It("verifies events are retrieved in correct order", func() {
		tags := NewTags("order_id", "order1")
		query := NewQuery(tags, "OrderCreated", "ItemAdded")
		events := []InputEvent{
			NewInputEvent("OrderCreated", tags, []byte(`{"id":"order1"}`)),
			NewInputEvent("ItemAdded", tags, []byte(`{"item":"product1"}`)),
			NewInputEvent("ItemAdded", tags, []byte(`{"item":"product2"}`)),
		}

		_, err := store.AppendEvents(ctx, events, query, 0)
		Expect(err).NotTo(HaveOccurred())

		type OrderState struct {
			Items []string
		}

		orderProjector := StateProjector{
			Query:        query,
			InitialState: OrderState{Items: []string{}},
			TransitionFn: func(state any, e Event) any {
				s := state.(OrderState)
				if e.Type == "ItemAdded" {
					var data map[string]string
					_ = json.Unmarshal(e.Data, &data)
					s.Items = append(s.Items, data["item"])
				}
				return s
			},
		}

		_, state, err := store.ProjectState(ctx, orderProjector)
		Expect(err).NotTo(HaveOccurred())
		Expect(state.(OrderState).Items).To(Equal([]string{"product1", "product2"}))
	})

	It("appends events with expected position correctly", func() {
		tags := NewTags("sequence_id", "seq1")
		query := NewQuery(tags, "First", "Second")

		// First event
		_, err := store.AppendEvents(ctx, []InputEvent{
			NewInputEvent("First", tags, []byte(`{"value":1}`)),
		}, query, 0)
		Expect(err).NotTo(HaveOccurred())

		// Second event with correct position
		pos, err := store.AppendEvents(ctx, []InputEvent{
			NewInputEvent("Second", tags, []byte(`{"value":2}`)),
		}, query, 1)

		Expect(err).NotTo(HaveOccurred())
		Expect(pos).To(Equal(int64(2)))
	})

	It("handles complex JSON payloads", func() {
		tags := NewTags("doc_id", "doc1")
		query := NewQuery(tags, "ComplexData")

		complexJSON := []byte(`{
			"nested": {
				"array": [1, 2, 3],
				"object": {"key": "value"}
			},
			"special": "quotes \"inside\" and emoji 🚀",
			"numbers": [3.14159, 42, -1]
		}`)

		pos, err := store.AppendEvents(ctx, []InputEvent{
			NewInputEvent("ComplexData", tags, complexJSON),
		}, query, 0)

		Expect(err).NotTo(HaveOccurred())
		Expect(pos).To(Equal(int64(1)))

		// Verify the data can be retrieved correctly
		type Result struct {
			Event Event
		}

		verifyProjector := StateProjector{
			Query:        query,
			InitialState: nil,
			TransitionFn: func(state any, e Event) any {
				return Result{Event: e}
			},
		}

		_, state, err := store.ProjectState(ctx, verifyProjector)
		Expect(err).NotTo(HaveOccurred())

		// Parse the JSON to verify it's valid
		var parsedData map[string]interface{}
		err = json.Unmarshal(state.(Result).Event.Data, &parsedData)
		Expect(err).NotTo(HaveOccurred())
		Expect(parsedData["special"]).To(ContainSubstring("emoji"))
	})

	It("handles boundary condition with many events in a batch", func() {
		tags := NewTags("batch_id", "large")
		query := NewQuery(tags, "BatchItem")

		// Create 50 events in a batch
		events := make([]InputEvent, 50)
		for i := 0; i < 50; i++ {
			events[i] = NewInputEvent("BatchItem", tags,
				[]byte(fmt.Sprintf(`{"index":%d}`, i)))
		}

		pos, err := store.AppendEvents(ctx, events, query, 0)
		Expect(err).NotTo(HaveOccurred())
		Expect(pos).To(Equal(int64(50)))

		// Verify count
		countProjector := StateProjector{
			Query:        query,
			InitialState: 0,
			TransitionFn: func(state any, e Event) any {
				return state.(int) + 1
			},
		}

		_, state, err := store.ProjectState(ctx, countProjector)
		Expect(err).NotTo(HaveOccurred())
		Expect(state).To(Equal(50))
	})

	// --- Additional critical test scenarios ---
	It("projects state with only event types", func() {
		tags := NewTags("type_id", "T1")
		query := NewQuery(tags, "TypeA", "TypeB")
		events := []InputEvent{
			NewInputEvent("TypeA", tags, []byte(`{"val":1}`)),
			NewInputEvent("TypeB", tags, []byte(`{"val":2}`)),
			NewInputEvent("TypeA", tags, []byte(`{"val":3}`)),
		}
		_, err := store.AppendEvents(ctx, events, query, 0)
		Expect(err).NotTo(HaveOccurred())

		projector := StateProjector{
			Query:        NewQuery(NewTags(), "TypeA"),
			InitialState: 0,
			TransitionFn: func(state any, e Event) any { return state.(int) + 1 },
		}
		_, state, err := store.ProjectState(ctx, projector)
		Expect(err).NotTo(HaveOccurred())
		Expect(state).To(Equal(2)) // Only TypeA events
	})

	It("projects state with only tags", func() {
		tags1 := NewTags("tag", "A")
		tags2 := NewTags("tag", "B")
		query := NewQuery(NewTags(), "E1", "E2")
		events := []InputEvent{
			NewInputEvent("E1", tags1, []byte(`{"v":1}`)),
			NewInputEvent("E2", tags2, []byte(`{"v":2}`)),
		}
		_, err := store.AppendEvents(ctx, events, query, 0)
		Expect(err).NotTo(HaveOccurred())

		projector := StateProjector{
			Query:        NewQuery(NewTags("tag", "A"), "E1"),
			InitialState: 0,
			TransitionFn: func(state any, e Event) any { return state.(int) + 1 },
		}
		_, state, err := store.ProjectState(ctx, projector)
		Expect(err).NotTo(HaveOccurred())
		Expect(state).To(Equal(1))
	})

	It("projects state with both tags and event types", func() {
		tags := NewTags("combo", "yes")
		events := []InputEvent{
			NewInputEvent("A", tags, []byte(`{"x":1}`)),
			NewInputEvent("B", tags, []byte(`{"x":2}`)),
			NewInputEvent("A", NewTags("combo", "no"), []byte(`{"x":3}`)),
		}
		_, err := store.AppendEvents(ctx, events, NewQuery(NewTags(), "A", "B"), 0)
		Expect(err).NotTo(HaveOccurred())

		projector := StateProjector{
			Query:        NewQuery(NewTags("combo", "yes"), "A"),
			InitialState: 0,
			TransitionFn: func(state any, e Event) any { return state.(int) + 1 },
		}
		_, state, err := store.ProjectState(ctx, projector)
		Expect(err).NotTo(HaveOccurred())
		Expect(state).To(Equal(1))
	})

	It("returns initial state if projector query matches no events", func() {
		tags := NewTags("none", "match")
		query := NewQuery(tags, "NonExistentEvent")
		// No events appended
		projector := StateProjector{
			Query:        query,
			InitialState: 42,
			TransitionFn: func(state any, e Event) any { return 0 },
		}
		_, state, err := store.ProjectState(ctx, projector)
		Expect(err).NotTo(HaveOccurred())
		Expect(state).To(Equal(42))
	})

	It("ProjectStateUpTo with maxPosition = 0 returns initial state", func() {
		tags := NewTags("seq", "zero")
		query := NewQuery(tags)
		events := []InputEvent{
			NewInputEvent("E", tags, []byte(`{"v":1}`)),
		}
		_, err := store.AppendEvents(ctx, events, query, 0)
		Expect(err).NotTo(HaveOccurred())
		projector := StateProjector{
			Query:        query,
			InitialState: 99,
			TransitionFn: func(state any, e Event) any { return 0 },
		}
		pos, state, err := store.ProjectStateUpTo(ctx, projector, 0)
		Expect(err).NotTo(HaveOccurred())
		Expect(pos).To(Equal(int64(0)))
		Expect(state).To(Equal(99))
	})

	It("ProjectStateUpTo with maxPosition in the middle", func() {
		tags := NewTags("seq", "mid")
		query := NewQuery(tags)
		events := []InputEvent{
			NewInputEvent("E", tags, []byte(`{"v":1}`)),
			NewInputEvent("E", tags, []byte(`{"v":2}`)),
			NewInputEvent("E", tags, []byte(`{"v":3}`)),
		}
		_, err := store.AppendEvents(ctx, events, query, 0)
		Expect(err).NotTo(HaveOccurred())
		projector := StateProjector{
			Query:        query,
			InitialState: 0,
			TransitionFn: func(state any, e Event) any { return state.(int) + 1 },
		}
		pos, state, err := store.ProjectStateUpTo(ctx, projector, 2)
		Expect(err).NotTo(HaveOccurred())
		Expect(pos).To(Equal(int64(2)))
		Expect(state).To(Equal(2))
	})

	It("returns error if projector TransitionFn is nil", func() {
		projector := StateProjector{
			Query:        NewQuery(NewTags()),
			InitialState: 0,
			TransitionFn: nil,
		}
		_, _, err := store.ProjectState(ctx, projector)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("projector function cannot be nil"))
	})

	It("returns error if projector TransitionFn panics", func() {
		tags := NewTags("panic", "yes")
		query := NewQuery(tags)
		events := []InputEvent{
			NewInputEvent("E", tags, []byte(`{"v":1}`)),
		}
		_, err := store.AppendEvents(ctx, events, query, 0)
		Expect(err).NotTo(HaveOccurred())
		projector := StateProjector{
			Query:        query,
			InitialState: 0,
			TransitionFn: func(state any, e Event) any { panic("fail!") },
		}
		_, _, err = store.ProjectState(ctx, projector)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("panic in projector"))
	})

	It("handles mutable pointer state in projector", func() {
		tags := NewTags("mut", "ptr")
		query := NewQuery(tags)
		events := []InputEvent{
			NewInputEvent("E", tags, []byte(`{"v":1}`)),
			NewInputEvent("E", tags, []byte(`{"v":2}`)),
		}
		_, err := store.AppendEvents(ctx, events, query, 0)
		Expect(err).NotTo(HaveOccurred())
		type Counter struct{ N int }
		projector := StateProjector{
			Query:        query,
			InitialState: &Counter{N: 0},
			TransitionFn: func(state any, e Event) any {
				c := state.(*Counter)
				c.N++
				return c
			},
		}
		_, state, err := store.ProjectState(ctx, projector)
		Expect(err).NotTo(HaveOccurred())
		c := state.(*Counter)
		Expect(c.N).To(Equal(2))
	})
})
