package dcb

import (
	"encoding/json"
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Event Store: Appending Events", func() {
	BeforeEach(func() {
		// Truncate the events table and reset sequences before each test
		_, err := pool.Exec(ctx, "TRUNCATE TABLE events RESTART IDENTITY CASCADE")
		Expect(err).NotTo(HaveOccurred())
	})

	It("appends events successfully", func() {
		tags := NewTags("course_id", "course1")
		query := NewQuery(tags)
		event := NewInputEvent("Subscription", tags, []byte(`{"foo":"bar"}`))
		events := []InputEvent{event}
		pos, err := store.AppendEvents(ctx, events, query, 0)
		Expect(err).NotTo(HaveOccurred())
		Expect(pos).To(Equal(int64(1)))

		projector := StateProjector{
			InitialState: 0,
			TransitionFn: func(state any, e Event) any {
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
		tags := NewTags("course_id", "course1", "user_id", "user123", "action", "enroll")
		query := NewQuery(NewTags("course_id", "course1"))
		events := []InputEvent{
			NewInputEvent("Enrollment", tags, []byte(`{"action":"enrolled"}`)),
		}

		pos, err := store.AppendEvents(ctx, events, query, 0)
		Expect(err).NotTo(HaveOccurred())
		Expect(pos).To(Equal(int64(1)))
		// Query by different tag combinations
		queryByUser := NewQuery(NewTags("user_id", "user123"))
		projector := StateProjector{
			InitialState: 0,
			TransitionFn: func(state any, e Event) any { return state.(int) + 1 },
		}

		_, state, err := store.ProjectState(ctx, queryByUser, projector)
		Expect(err).NotTo(HaveOccurred())
		Expect(state).To(Equal(1))

		dumpEvents(pool)
	})

	It("appends multiple events in a batch", func() {
		tags := NewTags("course_id", "course2")
		query := NewQuery(tags)
		events := []InputEvent{
			NewInputEvent("CourseLaunched", tags, []byte(`{"title":"Go Programming"}`)),
			NewInputEvent("LessonAdded", tags, []byte(`{"lesson_id":"L1"}`)),
			NewInputEvent("LessonAdded", tags, []byte(`{"lesson_id":"L2"}`)),
		}

		pos, err := store.AppendEvents(ctx, events, query, 0)
		Expect(err).NotTo(HaveOccurred())
		Expect(pos).To(Equal(int64(3)))

		projector := StateProjector{
			InitialState: 0,
			TransitionFn: func(state any, e Event) any { return state.(int) + 1 },
		}
		_, state, err := store.ProjectState(ctx, query, projector)
		Expect(err).NotTo(HaveOccurred())
		Expect(state).To(Equal(3))

		dumpEvents(pool)
	})

	It("fails with concurrency error when position is outdated", func() {
		tags := NewTags("course_id", "course3")
		query := NewQuery(tags)

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
		query := NewQuery(tags)
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
		query := NewQuery(tags)
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

		_, state, err := store.ProjectState(ctx, query, orderProjector)
		Expect(err).NotTo(HaveOccurred())
		Expect(state.(OrderState).Items).To(Equal([]string{"product1", "product2"}))
	})

	It("appends events with expected position correctly", func() {
		tags := NewTags("sequence_id", "seq1")
		query := NewQuery(tags)

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
		query := NewQuery(tags)

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
			InitialState: nil,
			TransitionFn: func(state any, e Event) any {
				return Result{Event: e}
			},
		}

		_, state, err := store.ProjectState(ctx, query, verifyProjector)
		Expect(err).NotTo(HaveOccurred())

		// Parse the JSON to verify it's valid
		var parsedData map[string]interface{}
		err = json.Unmarshal(state.(Result).Event.Data, &parsedData)
		Expect(err).NotTo(HaveOccurred())
		Expect(parsedData["special"]).To(ContainSubstring("emoji"))
	})

	It("handles boundary condition with many events in a batch", func() {
		tags := NewTags("batch_id", "large")
		query := NewQuery(tags)

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
			InitialState: 0,
			TransitionFn: func(state any, e Event) any {
				return state.(int) + 1
			},
		}

		_, state, err := store.ProjectState(ctx, query, countProjector)
		Expect(err).NotTo(HaveOccurred())
		Expect(state).To(Equal(50))
	})
})
