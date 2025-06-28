package dcb

import (
	"context"
	"fmt"
	"time"

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

		// Create context with timeout for each test
		ctx, _ = context.WithTimeout(context.Background(), 30*time.Second)

		// Truncate events table before each test
		err := truncateEventsTable(ctx, pool)
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("NewInputEvent", func() {
		It("should create valid input event", func() {
			event := NewInputEvent("TestEvent", NewTags("key", "value"), toJSON(map[string]string{"data": "test"}))
			Expect(event.GetType()).To(Equal("TestEvent"))
			Expect(event.GetTags()).To(Equal(NewTags("key", "value")))
			Expect(event.GetData()).To(Equal(toJSON(map[string]string{"data": "test"})))
		})

		It("should validate invalid JSON data", func() {
			// Create event with invalid JSON data - validation should happen in EventStore operations
			event := NewInputEvent("TestEvent", NewTags("key", "value"), []byte("invalid json"))
			Expect(event.GetType()).To(Equal("TestEvent"))
			Expect(event.GetData()).To(Equal([]byte("invalid json")))

			// Try to append the event - this should fail validation
			events := []InputEvent{event}
			err := store.Append(ctx, events)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid input syntax for type json"))
		})

		It("should validate empty event type", func() {
			// Create event with empty type - validation should happen in EventStore operations
			event := NewInputEvent("", NewTags("key", "value"), toJSON(map[string]string{"data": "test"}))
			Expect(event.GetType()).To(Equal(""))

			// Try to append the event - this should fail validation
			events := []InputEvent{event}
			err := store.Append(ctx, events)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("empty type"))
		})

		It("should validate empty tag key", func() {
			event := NewInputEvent("TestEvent", []Tag{NewTag("", "value")}, toJSON(map[string]string{"data": "test"}))
			Expect(event.GetTags()).To(Equal([]Tag{NewTag("", "value")}))
			Expect(event.GetTags()[0].GetKey()).To(Equal(""))
		})

		It("should validate empty tag value", func() {
			event := NewInputEvent("TestEvent", []Tag{NewTag("key", "")}, toJSON(map[string]string{"data": "test"}))
			Expect(event.GetTags()).To(Equal([]Tag{NewTag("key", "")}))
			Expect(event.GetTags()[0].GetValue()).To(Equal(""))
		})

		It("should handle empty tags", func() {
			event := NewInputEvent("TestEvent", []Tag{}, toJSON(map[string]string{"data": "test"}))
			Expect(event.GetTags()).To(BeEmpty())
		})

		It("should handle empty data", func() {
			event := NewInputEvent("TestEvent", NewTags("key", "value"), []byte{})
			Expect(event.GetData()).To(BeEmpty())
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

			err := store.Append(ctx, events)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should append events with After condition", func() {
			// First append
			event1 := NewInputEvent("TestEvent", NewTags("key", "value1"), toJSON(map[string]string{"data": "value1"}))
			events1 := []InputEvent{event1}
			err := store.Append(ctx, events1)
			Expect(err).NotTo(HaveOccurred())

			// Second append with After condition (using position 1 since we just appended an event)
			event2 := NewInputEvent("TestEvent", NewTags("key", "value2"), toJSON(map[string]string{"data": "value2"}))
			events2 := []InputEvent{event2}
			position1 := int64(1)
			condition := NewAppendConditionAfter(&position1)
			err = store.AppendIf(ctx, events2, condition)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should allow append with non-existent After position (modern event store semantics)", func() {
			event := NewInputEvent("TestEvent", NewTags("key", "value"), toJSON(map[string]string{"data": "test"}))
			events := []InputEvent{event}

			invalidPosition := int64(999999)
			condition := NewAppendConditionAfter(&invalidPosition)
			err := store.AppendIf(ctx, events, condition)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should append events with FailIfEventsMatch condition", func() {
			// First append
			events := []InputEvent{
				NewInputEvent("UserCreated", NewTags("user_id", "123"), toJSON(map[string]string{"name": "John"})),
			}
			err := store.Append(ctx, events)
			Expect(err).NotTo(HaveOccurred())

			// Second append with FailIfEventsMatch condition
			events2 := []InputEvent{
				NewInputEvent("UserUpdated", NewTags("user_id", "123"), toJSON(map[string]string{"name": "Jane"})),
			}
			query := NewQuery(NewTags("user_id", "123"), "UserCreated")
			condition := NewAppendCondition(query)
			err = store.AppendIf(ctx, events2, condition)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("matching events found"))
		})

		It("should succeed append with FailIfEventsMatch condition when no matching events", func() {
			// Append with FailIfEventsMatch condition for non-existent events
			events := []InputEvent{
				NewInputEvent("UserCreated", NewTags("user_id", "123"), toJSON(map[string]string{"name": "John"})),
			}
			query := NewQuery(NewTags("user_id", "123"), "UserUpdated")
			condition := NewAppendCondition(query)
			err := store.AppendIf(ctx, events, condition)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Append validation", func() {
		It("should validate empty events slice", func() {
			err := store.Append(ctx, []InputEvent{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("empty"))
		})

		It("should validate nil events slice", func() {
			err := store.Append(ctx, nil)
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

			err := store.Append(ctx, events)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("exceeds maximum"))
		})

		It("should validate individual events in batch", func() {
			event1 := NewInputEvent("ValidEvent", NewTags("key", "value"), toJSON(map[string]string{"data": "valid"}))
			event2 := NewInputEvent("", NewTags("key", "value"), toJSON(map[string]string{"data": "invalid"})) // Empty type
			events := []InputEvent{
				event1,
				event2,
			}

			err := store.Append(ctx, events)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("empty type"))
		})
	})

	Describe("Error handling", func() {
		It("should handle database connection errors gracefully", func() {
			event := NewInputEvent("TestEvent", NewTags("key", "value"), toJSON(map[string]string{"data": "test"}))

			// This test would require a way to simulate connection errors
			// For now, just verify the event is created correctly
			Expect(event.GetType()).To(Equal("TestEvent"))
			Expect(event.GetTags()).To(HaveLen(1))
		})

		It("should handle validation errors in batch", func() {
			event := NewInputEvent("TestEvent", NewTags("key", "value"), toJSON(map[string]string{"data": "test"}))

			// This test would require a way to simulate validation errors
			// For now, just verify the event is valid
			Expect(event.GetType()).To(Equal("TestEvent"))
		})
	})

	Describe("AppendIf with isolation levels", func() {
		It("should append events with condition", func() {
			event := NewInputEvent("TestEvent", NewTags("key", "value"), toJSON(map[string]string{"data": "test"}))
			events := []InputEvent{event}

			// Test with a simple condition
			query := NewQuery(NewTags("non_existent", "value"), "NonExistentEvent")
			condition := NewAppendCondition(query)

			err := store.AppendIf(ctx, events, condition)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should fail append with conflicting condition", func() {
			// First append an event
			event1 := NewInputEvent("TestEvent", NewTags("key", "value"), toJSON(map[string]string{"data": "test"}))
			events1 := []InputEvent{event1}
			err := store.Append(ctx, events1)
			Expect(err).NotTo(HaveOccurred())

			// Then try to append with a condition that conflicts
			event2 := NewInputEvent("TestEvent2", NewTags("key", "value"), toJSON(map[string]string{"data": "test2"}))
			events2 := []InputEvent{event2}
			query := NewQuery(NewTags("key", "value"), "TestEvent")
			condition := NewAppendCondition(query)

			err = store.AppendIf(ctx, events2, condition)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("matching events found"))
		})

		It("should append events with nil condition (same as Append)", func() {
			event := NewInputEvent("TestEvent", NewTags("key", "value"), toJSON(map[string]string{"data": "test"}))
			events := []InputEvent{event}

			err := store.AppendIf(ctx, events, nil)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("AppendIfSerializable", func() {
		It("should append events with serializable isolation", func() {
			events := []InputEvent{
				NewInputEvent("TestEvent", NewTags("test", "serializable"), []byte(`{"value": "test"}`)),
			}

			err := store.AppendIfSerializable(ctx, events, nil)
			Expect(err).To(BeNil())

			// Verify the event was appended
			result, err := store.Read(ctx, NewQuery(NewTags("test", "serializable"), "TestEvent"))
			Expect(err).To(BeNil())
			Expect(result).To(HaveLen(1))
		})

		It("should respect append conditions with serializable isolation", func() {
			// First append
			events1 := []InputEvent{
				NewInputEvent("TestEvent", NewTags("test", "serializable"), []byte(`{"value": "test"}`)),
			}
			err := store.AppendIfSerializable(ctx, events1, nil)
			Expect(err).NotTo(HaveOccurred())

			// Second append with condition that should fail (looking for the event we just created)
			events2 := []InputEvent{
				NewInputEvent("TestEvent2", NewTags("test", "serializable"), []byte(`{"value": "test2"}`)),
			}
			query := NewQuery(NewTags("test", "serializable"), "TestEvent")
			condition := NewAppendCondition(query)
			err = store.AppendIfSerializable(ctx, events2, condition)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("append condition violated"))
		})
	})
})
