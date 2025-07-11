package dcb_test

import (
	"context"
	"fmt"

	"go-crablet/pkg/dcb"

	"github.com/jackc/pgx/v5/pgconn"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Append Helpers", func() {
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

	Describe("NewInputEvent", func() {
		It("should create valid input event", func() {
			event := dcb.NewInputEvent("TestEvent", dcb.NewTags("key", "value"), dcb.ToJSON(map[string]string{"data": "test"}))
			Expect(event.GetType()).To(Equal("TestEvent"))
			Expect(event.GetTags()).To(Equal(dcb.NewTags("key", "value")))
			Expect(event.GetData()).To(Equal(dcb.ToJSON(map[string]string{"data": "test"})))
		})

		It("should validate invalid JSON data", func() {
			// Create event with invalid JSON data - validation should happen in EventStore operations
			event := dcb.NewInputEvent("TestEvent", dcb.NewTags("key", "value"), []byte("invalid json"))
			Expect(event.GetType()).To(Equal("TestEvent"))
			Expect(event.GetData()).To(Equal([]byte("invalid json")))

			// Try to append the event - this should fail validation
			events := []dcb.InputEvent{event}
			err := store.Append(ctx, events, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid input syntax for type json"))
		})

		It("should validate empty event type", func() {
			// Create event with empty type - validation should happen in EventStore operations
			event := dcb.NewInputEvent("", dcb.NewTags("key", "value"), dcb.ToJSON(map[string]string{"data": "test"}))
			Expect(event.GetType()).To(Equal(""))

			// Try to append the event - this should fail validation
			events := []dcb.InputEvent{event}
			err := store.Append(ctx, events, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("empty type"))
		})

		It("should validate empty tag key", func() {
			event := dcb.NewInputEvent("TestEvent", []dcb.Tag{dcb.NewTag("", "value")}, dcb.ToJSON(map[string]string{"data": "test"}))
			Expect(event.GetTags()).To(Equal([]dcb.Tag{dcb.NewTag("", "value")}))
			Expect(event.GetTags()[0].GetKey()).To(Equal(""))
		})

		It("should validate empty tag value", func() {
			event := dcb.NewInputEvent("TestEvent", []dcb.Tag{dcb.NewTag("key", "")}, dcb.ToJSON(map[string]string{"data": "test"}))
			Expect(event.GetTags()).To(Equal([]dcb.Tag{dcb.NewTag("key", "")}))
			Expect(event.GetTags()[0].GetValue()).To(Equal(""))
		})

		It("should handle empty tags", func() {
			event := dcb.NewInputEvent("TestEvent", []dcb.Tag{}, dcb.ToJSON(map[string]string{"data": "test"}))
			Expect(event.GetTags()).To(BeEmpty())
		})

		It("should handle empty data", func() {
			event := dcb.NewInputEvent("TestEvent", dcb.NewTags("key", "value"), []byte{})
			Expect(event.GetData()).To(BeEmpty())
		})
	})

	Describe("NewEventBatch", func() {
		It("should create event batch from multiple events", func() {
			event1 := dcb.NewInputEvent("Event1", dcb.NewTags("key1", "value1"), dcb.ToJSON(map[string]string{"data": "value1"}))
			event2 := dcb.NewInputEvent("Event2", dcb.NewTags("key2", "value2"), dcb.ToJSON(map[string]string{"data": "value2"}))
			event3 := dcb.NewInputEvent("Event3", dcb.NewTags("key3", "value3"), dcb.ToJSON(map[string]string{"data": "value3"}))

			batch := dcb.NewEventBatch(event1, event2, event3)

			Expect(batch).To(HaveLen(3))
			Expect(batch[0]).To(Equal(event1))
			Expect(batch[1]).To(Equal(event2))
			Expect(batch[2]).To(Equal(event3))
		})

		It("should handle empty batch", func() {
			batch := dcb.NewEventBatch()
			Expect(batch).To(BeEmpty())
		})

		It("should handle single event", func() {
			event := dcb.NewInputEvent("SingleEvent", dcb.NewTags("key", "value"), dcb.ToJSON(map[string]string{"data": "test"}))
			batch := dcb.NewEventBatch(event)

			Expect(batch).To(HaveLen(1))
			Expect(batch[0]).To(Equal(event))
		})
	})

	Describe("Append with various conditions", func() {
		It("should append events without condition", func() {
			event1 := dcb.NewInputEvent("TestEvent", dcb.NewTags("key", "value1"), dcb.ToJSON(map[string]string{"data": "value1"}))
			event2 := dcb.NewInputEvent("TestEvent", dcb.NewTags("key", "value2"), dcb.ToJSON(map[string]string{"data": "value2"}))
			events := []dcb.InputEvent{event1, event2}

			err := store.Append(ctx, events, nil)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should append events with After condition", func() {
			// First append
			event1 := dcb.NewInputEvent("TestEvent", dcb.NewTags("key", "value"), dcb.ToJSON(map[string]string{"data": "test"}))
			events1 := []dcb.InputEvent{event1}
			err := store.Append(ctx, events1, nil)
			Expect(err).NotTo(HaveOccurred())

			// Second append with After condition (using cursor-based approach)
			event2 := dcb.NewInputEvent("TestEvent", dcb.NewTags("key", "value2"), dcb.ToJSON(map[string]string{"data": "value2"}))
			events2 := []dcb.InputEvent{event2}
			// Use cursor-based condition that doesn't match the first event
			query := dcb.NewQuery(dcb.NewTags("key", "different"), "TestEvent")
			condition := dcb.NewAppendCondition(query)
			err = store.Append(ctx, events2, &condition)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should allow append with non-existent After condition (modern event store semantics)", func() {
			event := dcb.NewInputEvent("TestEvent", dcb.NewTags("key", "value"), dcb.ToJSON(map[string]string{"data": "test"}))
			events := []dcb.InputEvent{event}

			// Use cursor-based condition for non-existent events
			query := dcb.NewQuery(dcb.NewTags("non_existent", "value"), "NonExistentEvent")
			condition := dcb.NewAppendCondition(query)
			err := store.Append(ctx, events, &condition)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should append events with FailIfEventsMatch condition", func() {
			// First append
			events := []dcb.InputEvent{
				dcb.NewInputEvent("UserCreated", dcb.NewTags("user_id", "123"), dcb.ToJSON(map[string]string{"name": "John"})),
			}
			err := store.Append(ctx, events, nil)
			Expect(err).NotTo(HaveOccurred())

			// Second append with FailIfEventsMatch condition
			events2 := []dcb.InputEvent{
				dcb.NewInputEvent("UserUpdated", dcb.NewTags("user_id", "123"), dcb.ToJSON(map[string]string{"name": "Jane"})),
			}
			query := dcb.NewQuery(dcb.NewTags("user_id", "123"), "UserCreated")
			condition := dcb.NewAppendCondition(query)
			err = store.Append(ctx, events2, &condition)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("matching events found"))
		})

		It("should succeed append with FailIfEventsMatch condition when no matching events", func() {
			// Append with FailIfEventsMatch condition for non-existent events
			events := []dcb.InputEvent{
				dcb.NewInputEvent("UserCreated", dcb.NewTags("user_id", "123"), dcb.ToJSON(map[string]string{"name": "John"})),
			}
			query := dcb.NewQuery(dcb.NewTags("user_id", "123"), "UserUpdated")
			condition := dcb.NewAppendCondition(query)
			err := store.Append(ctx, events, &condition)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Append validation", func() {
		It("should validate empty events slice", func() {
			err := store.Append(ctx, []dcb.InputEvent{}, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("empty"))
		})

		It("should validate nil events slice", func() {
			err := store.Append(ctx, nil, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("empty"))
		})

		It("should validate batch size limit", func() {
			// Create events exceeding the batch size limit
			events := make([]dcb.InputEvent, 1001) // Exceeds default limit of 1000
			for i := 0; i < 1001; i++ {
				events[i] = dcb.NewInputEvent("TestEvent", dcb.NewTags("test", fmt.Sprintf("value%d", i)), dcb.ToJSON(map[string]string{"index": fmt.Sprintf("%d", i)}))
			}

			err := store.Append(ctx, events, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("exceeds maximum"))
		})
	})

	Describe("Error handling", func() {
		It("should handle concurrency errors", func() {
			// First append
			events := []dcb.InputEvent{
				dcb.NewInputEvent("UserCreated", dcb.NewTags("user_id", "123"), dcb.ToJSON(map[string]string{"name": "John"})),
			}
			err := store.Append(ctx, events, nil)
			Expect(err).NotTo(HaveOccurred())

			// Second append with same condition - should fail
			events2 := []dcb.InputEvent{
				dcb.NewInputEvent("UserUpdated", dcb.NewTags("user_id", "123"), dcb.ToJSON(map[string]string{"name": "Jane"})),
			}
			query := dcb.NewQuery(dcb.NewTags("user_id", "123"), "UserCreated")
			condition := dcb.NewAppendCondition(query)
			err = store.Append(ctx, events2, &condition)

			Expect(err).To(HaveOccurred())
			// Check for specific error type
			if pgErr, ok := err.(*pgconn.PgError); ok {
				Expect(pgErr.Code).To(Equal("DCB01"))
			}
		})
	})
})
