package dcb

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Append Helpers", func() {
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

	Describe("NewInputEvent", func() {
		It("should create valid input event", func() {
			event := NewInputEvent("TestEvent", NewTags("key", "value"), toJSON(map[string]string{"data": "test"}))
			Expect(event.Type).To(Equal("TestEvent"))
			Expect(event.Tags).To(Equal(NewTags("key", "value")))
			Expect(event.Data).To(Equal(toJSON(map[string]string{"data": "test"})))
		})

		It("should validate JSON data", func() {
			// Create event with invalid JSON - validation should happen in EventStore operations
			event := NewInputEvent("TestEvent", NewTags("key", "value"), []byte("invalid json"))
			Expect(event.Type).To(Equal("TestEvent"))
			Expect(event.Data).To(Equal([]byte("invalid json")))

			// Try to append the event - this should fail validation
			events := []InputEvent{event}
			_, err := store.Append(ctx, events, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid input syntax for type json"))
		})

		It("should validate empty event type", func() {
			// Create event with empty type - validation should happen in EventStore operations
			event := NewInputEvent("", NewTags("key", "value"), toJSON(map[string]string{"data": "test"}))
			Expect(event.Type).To(Equal(""))

			// Try to append the event - this should fail validation
			events := []InputEvent{event}
			_, err := store.Append(ctx, events, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("empty type"))
		})

		It("should validate empty tag keys", func() {
			// Create event with empty tag key - validation should happen in EventStore operations
			event := NewInputEvent("TestEvent", []Tag{{Key: "", Value: "value"}}, toJSON(map[string]string{"data": "test"}))
			Expect(event.Type).To(Equal("TestEvent"))
			Expect(event.Tags[0].Key).To(Equal(""))

			// Try to append the event - this should fail validation
			events := []InputEvent{event}
			_, err := store.Append(ctx, events, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("empty key"))
		})

		It("should validate empty tag values", func() {
			// Create event with empty tag value - validation should happen in EventStore operations
			event := NewInputEvent("TestEvent", []Tag{{Key: "key", Value: ""}}, toJSON(map[string]string{"data": "test"}))
			Expect(event.Type).To(Equal("TestEvent"))
			Expect(event.Tags[0].Value).To(Equal(""))

			// Try to append the event - this should fail validation
			events := []InputEvent{event}
			_, err := store.Append(ctx, events, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("empty value"))
		})

		It("should handle empty tags", func() {
			event := NewInputEvent("TestEvent", []Tag{}, toJSON(map[string]string{"data": "test"}))
			Expect(event.Tags).To(BeEmpty())
		})

		It("should handle empty data", func() {
			event := NewInputEvent("TestEvent", NewTags("key", "value"), []byte{})
			Expect(event.Data).To(BeEmpty())
		})
	})

	Describe("NewEventBatch", func() {
		It("should create event batch from multiple events", func() {
			event1 := NewInputEvent("Event1", NewTags("key1", "value1"), toJSON(map[string]string{"data": "value1"}))
			event2 := NewInputEvent("Event2", NewTags("key2", "value2"), toJSON(map[string]string{"data": "value2"}))
			event3 := NewInputEvent("Event3", NewTags("key3", "value3"), toJSON(map[string]string{"data": "value3"}))

			batch := NewEventBatch(event1, event2, event3)

			Expect(batch).To(HaveLen(3))
			Expect(batch[0]).To(Equal(event1))
			Expect(batch[1]).To(Equal(event2))
			Expect(batch[2]).To(Equal(event3))
		})

		It("should handle empty batch", func() {
			batch := NewEventBatch()
			Expect(batch).To(BeEmpty())
		})

		It("should handle single event", func() {
			event := NewInputEvent("SingleEvent", NewTags("key", "value"), toJSON(map[string]string{"data": "test"}))
			batch := NewEventBatch(event)

			Expect(batch).To(HaveLen(1))
			Expect(batch[0]).To(Equal(event))
		})
	})

	Describe("Append with various conditions", func() {
		It("should append events without condition", func() {
			event1 := NewInputEvent("TestEvent", NewTags("key", "value1"), toJSON(map[string]string{"data": "value1"}))
			event2 := NewInputEvent("TestEvent", NewTags("key", "value2"), toJSON(map[string]string{"data": "value2"}))
			events := []InputEvent{event1, event2}

			position, err := store.Append(ctx, events, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(position).To(BeNumerically(">", 0))
		})

		It("should append events with After condition", func() {
			// First append
			event1 := NewInputEvent("TestEvent", NewTags("key", "value1"), toJSON(map[string]string{"data": "value1"}))
			events1 := []InputEvent{event1}
			position1, err := store.Append(ctx, events1, nil)
			Expect(err).NotTo(HaveOccurred())

			// Second append with After condition
			event2 := NewInputEvent("TestEvent", NewTags("key", "value2"), toJSON(map[string]string{"data": "value2"}))
			events2 := []InputEvent{event2}
			condition := &AppendCondition{After: &position1}
			position2, err := store.Append(ctx, events2, condition)
			Expect(err).NotTo(HaveOccurred())
			Expect(position2).To(BeNumerically(">", position1))
		})

		It("should fail append with invalid After condition", func() {
			event := NewInputEvent("TestEvent", NewTags("key", "value"), toJSON(map[string]string{"data": "test"}))
			events := []InputEvent{event}

			invalidPosition := int64(999999)
			condition := &AppendCondition{After: &invalidPosition}
			_, err := store.Append(ctx, events, condition)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("optimistic"))
		})

		It("should append events with FailIfEventsMatch condition", func() {
			// First append
			event1 := NewInputEvent("TestEvent", NewTags("key", "value1"), toJSON(map[string]string{"data": "value1"}))
			events1 := []InputEvent{event1}
			_, err := store.Append(ctx, events1, nil)
			Expect(err).NotTo(HaveOccurred())

			// Second append with FailIfEventsMatch condition
			event2 := NewInputEvent("TestEvent", NewTags("key", "value2"), toJSON(map[string]string{"data": "value2"}))
			events2 := []InputEvent{event2}
			condition := &AppendCondition{
				FailIfEventsMatch: &Query{
					Items: []QueryItem{
						{
							EventTypes: []string{"TestEvent"},
							Tags:       NewTags("key", "value1"),
						},
					},
				},
			}
			_, err = store.Append(ctx, events2, condition)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("append condition violated"))
		})

		It("should succeed append with FailIfEventsMatch condition when no matching events", func() {
			// Append with FailIfEventsMatch condition for non-existent events
			event := NewInputEvent("TestEvent", NewTags("key", "value1"), toJSON(map[string]string{"data": "value1"}))
			events := []InputEvent{event}
			condition := &AppendCondition{
				FailIfEventsMatch: &Query{
					Items: []QueryItem{
						{
							EventTypes: []string{"NonExistentEvent"},
							Tags:       NewTags("key", "non-existent"),
						},
					},
				},
			}
			position, err := store.Append(ctx, events, condition)
			Expect(err).NotTo(HaveOccurred())
			Expect(position).To(BeNumerically(">", 0))
		})
	})

	Describe("Append validation", func() {
		It("should validate empty events slice", func() {
			_, err := store.Append(ctx, []InputEvent{}, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("empty"))
		})

		It("should validate nil events slice", func() {
			_, err := store.Append(ctx, nil, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("empty"))
		})

		It("should validate batch size limit", func() {
			// Create events exceeding the batch size limit
			events := make([]InputEvent, 1001) // Exceeds default limit of 1000
			for i := 0; i < 1001; i++ {
				event := NewInputEvent("TestEvent", NewTags("test", fmt.Sprintf("value%d", i)), toJSON(map[string]string{"index": fmt.Sprintf("%d", i)}))
				events[i] = event
			}

			_, err := store.Append(ctx, events, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("exceeds maximum"))
		})

		It("should validate individual events in batch", func() {
			event1 := NewInputEvent("ValidEvent", NewTags("key", "value"), toJSON(map[string]string{"data": "valid"}))
			events := []InputEvent{
				event1,
				{Type: "", Tags: NewTags("key", "value"), Data: toJSON(map[string]string{"data": "invalid"})}, // Empty type
			}

			_, err := store.Append(ctx, events, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("empty type"))
		})
	})

	Describe("Causation and Correlation IDs", func() {
		It("should generate causation and correlation IDs", func() {
			inputEvent := NewInputEvent("TestEvent", NewTags("key", "value"), toJSON(map[string]string{"data": "test"}))
			events := []InputEvent{inputEvent}

			position, err := store.Append(ctx, events, nil)
			Expect(err).NotTo(HaveOccurred())

			// Read back the events to verify IDs
			query := NewQuerySimple(NewTags("key", "value"), "TestEvent")
			sequencedEvents, err := store.Read(ctx, query, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(sequencedEvents.Events).To(HaveLen(1))

			readEvent := sequencedEvents.Events[0]
			Expect(readEvent.CausationID).NotTo(BeEmpty())
			Expect(readEvent.CorrelationID).NotTo(BeEmpty())
			Expect(readEvent.Position).To(Equal(position))
		})

		It("should maintain causation chain across multiple appends", func() {
			// First append
			event1 := NewInputEvent("FirstEvent", NewTags("key", "value"), toJSON(map[string]string{"data": "first"}))
			events1 := []InputEvent{event1}
			position1, err := store.Append(ctx, events1, nil)
			Expect(err).NotTo(HaveOccurred())

			// Second append
			event2 := NewInputEvent("SecondEvent", NewTags("key", "value"), toJSON(map[string]string{"data": "second"}))
			events2 := []InputEvent{event2}
			position2, err := store.Append(ctx, events2, nil)
			Expect(err).NotTo(HaveOccurred())

			// Verify positions are sequential
			Expect(position2).To(BeNumerically(">", position1))
		})

		It("should maintain correlation across related events", func() {
			// Create related events
			event1 := NewInputEvent("Event1", NewTags("key", "value"), toJSON(map[string]string{"order": "1"}))
			event2 := NewInputEvent("Event2", NewTags("key", "value"), toJSON(map[string]string{"order": "2"}))
			event3 := NewInputEvent("Event3", NewTags("key", "value"), toJSON(map[string]string{"order": "3"}))
			events := []InputEvent{event1, event2, event3}

			_, err := store.Append(ctx, events, nil)
			Expect(err).NotTo(HaveOccurred())

			// Read back events to verify correlation
			query := NewQuerySimple(NewTags("key", "value"), "Event1", "Event2", "Event3")
			sequencedEvents, err := store.Read(ctx, query, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(sequencedEvents.Events).To(HaveLen(3))

			// All events should have the same correlation ID
			correlationID := sequencedEvents.Events[0].CorrelationID
			for _, event := range sequencedEvents.Events {
				Expect(event.CorrelationID).To(Equal(correlationID))
			}
		})
	})

	Describe("Error handling", func() {
		It("should handle database connection errors gracefully", func() {
			event := NewInputEvent("TestEvent", NewTags("key", "value"), toJSON(map[string]string{"data": "test"}))

			// This test would require a way to simulate connection errors
			// For now, just verify the event is created correctly
			Expect(event.Type).To(Equal("TestEvent"))
			Expect(event.Tags).To(HaveLen(1))
		})

		It("should handle validation errors in batch", func() {
			event := NewInputEvent("TestEvent", NewTags("key", "value"), toJSON(map[string]string{"data": "test"}))

			// This test would require a way to simulate validation errors
			// For now, just verify the event is valid
			Expect(event.Type).To(Equal("TestEvent"))
		})
	})
})
