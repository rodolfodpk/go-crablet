package dcb

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Validation", func() {
	Describe("validateEvent", func() {
		It("should validate valid event", func() {
			event := NewInputEvent("TestEvent", NewTags("key", "value"), toJSON(map[string]string{"data": "test"}))

			err := validateEvent(event, 0)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should validate event with empty type", func() {
			event := NewInputEvent("", NewTags("key", "value"), toJSON(map[string]string{"data": "test"}))

			err := validateEvent(event, 0)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("empty type"))
		})

		It("should validate event with empty tag key", func() {
			event := NewInputEvent("TestEvent", []Tag{NewTag("", "value")}, toJSON(map[string]string{"data": "test"}))

			err := validateEvent(event, 0)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("empty tag key"))
		})

		It("should validate event with empty tag value", func() {
			event := NewInputEvent("TestEvent", []Tag{NewTag("key", "")}, toJSON(map[string]string{"data": "test"}))

			err := validateEvent(event, 0)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("empty value for key"))
		})

		It("should validate event with invalid JSON data", func() {
			event := NewInputEvent("TestEvent", NewTags("key", "value"), []byte("invalid json"))

			err := validateEvent(event, 0)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid JSON data"))
		})

		It("should validate event with empty data", func() {
			event := NewInputEvent("TestEvent", NewTags("key", "value"), []byte{})

			err := validateEvent(event, 0)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid JSON data"))
		})

		It("should validate event with nil data", func() {
			event := NewInputEvent("TestEvent", NewTags("key", "value"), nil)

			err := validateEvent(event, 0)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid JSON data"))
		})

		It("should validate event with empty tags", func() {
			event := NewInputEvent("TestEvent", []Tag{}, toJSON(map[string]string{"data": "test"}))

			err := validateEvent(event, 0)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("empty tags"))
		})

		It("should validate event with multiple tags", func() {
			event := NewInputEvent("TestEvent", []Tag{
				NewTag("key1", "value1"),
				NewTag("key2", "value2"),
			}, toJSON(map[string]string{"data": "test"}))

			err := validateEvent(event, 0)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should include index in error message", func() {
			event := NewInputEvent("", NewTags("key", "value"), toJSON(map[string]string{"data": "test"}))

			err := validateEvent(event, 5)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("event 5"))
		})

		It("should validate event with valid data", func() {
			event := NewInputEvent("TestEvent", []Tag{NewTag("key", "value")}, toJSON(map[string]string{"data": "test"}))
			err := validateEvent(event, 0)
			Expect(err).To(BeNil())
		})

		It("should validate event with empty type", func() {
			event := NewInputEvent("", []Tag{NewTag("key", "value")}, toJSON(map[string]string{"data": "test"}))
			err := validateEvent(event, 0)
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("empty type"))
		})
	})

	Describe("validateEvents", func() {
		It("should validate valid events slice", func() {
			events := []InputEvent{
				NewInputEvent("Event1", NewTags("key1", "value1"), toJSON(map[string]string{"data": "value1"})),
				NewInputEvent("Event2", NewTags("key2", "value2"), toJSON(map[string]string{"data": "value2"})),
			}

			err := validateEvents(events)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should validate empty events slice", func() {
			err := validateEvents([]InputEvent{})
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return first validation error", func() {
			events := []InputEvent{
				NewInputEvent("Event1", NewTags("key1", "value1"), toJSON(map[string]string{"data": "value1"})),
				NewInputEvent("", NewTags("key2", "value2"), toJSON(map[string]string{"data": "value2"})),
				NewInputEvent("Event3", []Tag{NewTag("", "value3")}, toJSON(map[string]string{"data": "value3"})),
			}

			err := validateEvents(events)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("event 1"))
			Expect(err.Error()).To(ContainSubstring("empty type"))
		})

		It("should validate batch size limit", func() {
			events := make([]InputEvent, 1000) // Default limit
			for i := 0; i < 1000; i++ {
				events[i] = NewInputEvent("TestEvent", NewTags("test", fmt.Sprintf("value%d", i)), toJSON(map[string]string{"index": fmt.Sprintf("%d", i)}))
			}
			err := validateEvents(events)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should validate batch size limit (exceeds)", func() {
			events := make([]InputEvent, 1001)
			for i := 0; i < 1001; i++ {
				events[i] = NewInputEvent("TestEvent", NewTags("test", fmt.Sprintf("value%d", i)), toJSON(map[string]string{"index": fmt.Sprintf("%d", i)}))
			}
			err := validateEvents(events)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("exceeds maximum"))
		})

		It("should validate individual events in batch (with error)", func() {
			events := make([]InputEvent, 3)
			events[0] = NewInputEvent("Event1", NewTags("key1", "value1"), toJSON(map[string]string{"data": "value1"}))
			events[1] = NewInputEvent("", NewTags("key2", "value2"), toJSON(map[string]string{"data": "value2"}))         // Empty type
			events[2] = NewInputEvent("Event3", []Tag{NewTag("", "value3")}, toJSON(map[string]string{"data": "value3"})) // Empty key
			err := validateEvents(events)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("event 1"))
			Expect(err.Error()).To(ContainSubstring("empty type"))
		})
	})

	Describe("validateQueryTags", func() {
		It("should validate valid query", func() {
			query := NewQueryFromItems(
				NewQueryItem([]string{"Event1", "Event2"}, NewTags("key1", "value1")),
				NewQueryItem([]string{"Event3"}, NewTags("key2", "value2")),
			)

			err := validateQueryTags(query)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should validate empty query", func() {
			query := NewQueryEmpty()
			err := validateQueryTags(query)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should validate query with empty event types", func() {
			query := NewQueryFromItems(
				NewQueryItem([]string{""}, NewTags("key", "value")), // Empty event type
			)

			err := validateQueryTags(query)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("empty event type"))
		})

		It("should validate query with empty tag keys", func() {
			query := NewQueryFromItems(
				NewQueryItem([]string{"Event1"}, []Tag{NewTag("", "value")}), // Empty key
			)

			err := validateQueryTags(query)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("empty tag key"))
		})

		It("should validate query with empty tag values", func() {
			query := NewQueryFromItems(
				NewQueryItem([]string{"Event1"}, []Tag{NewTag("key", "")}), // Empty value
			)

			err := validateQueryTags(query)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("empty value"))
		})

		It("should validate query with empty event types and tags", func() {
			query := NewQueryFromItems(
				NewQueryItem([]string{}, []Tag{}),
			)

			err := validateQueryTags(query)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should validate query with multiple items", func() {
			query := NewQueryFromItems(
				NewQueryItem([]string{"Event1"}, NewTags("key1", "value1")),
				NewQueryItem([]string{"Event2"}, NewTags("key2", "value2")),
			)

			err := validateQueryTags(query)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("validateBatchSize", func() {
		BeforeEach(func() {
			// Use shared PostgreSQL container and truncate events between tests
			// Truncate events table before each test
			err := truncateEventsTable(ctx, pool)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should validate valid batch size", func() {
			es := store.(*eventStore)
			events := make([]InputEvent, 100)
			for i := 0; i < 100; i++ {
				events[i] = NewInputEvent("TestEvent", NewTags("key", fmt.Sprintf("value%d", i)), toJSON(map[string]string{"data": fmt.Sprintf("value%d", i)}))
			}
			err := es.validateBatchSize(events, "test")
			Expect(err).NotTo(HaveOccurred())
		})

		It("should validate batch size at limit", func() {
			es := store.(*eventStore)
			events := make([]InputEvent, 1000)
			for i := 0; i < 1000; i++ {
				events[i] = NewInputEvent("TestEvent", NewTags("key", fmt.Sprintf("value%d", i)), toJSON(map[string]string{"data": fmt.Sprintf("value%d", i)}))
			}
			err := es.validateBatchSize(events, "test")
			Expect(err).NotTo(HaveOccurred())
		})

		It("should reject batch size exceeding limit", func() {
			es := store.(*eventStore)
			events := make([]InputEvent, 1001)
			for i := 0; i < 1001; i++ {
				events[i] = NewInputEvent("TestEvent", NewTags("key", fmt.Sprintf("value%d", i)), toJSON(map[string]string{"data": fmt.Sprintf("value%d", i)}))
			}
			err := es.validateBatchSize(events, "test")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("exceeds maximum"))
		})

		It("should validate empty batch", func() {
			es := store.(*eventStore)
			err := es.validateBatchSize([]InputEvent{}, "test")
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("ValidationError", func() {
		It("should create validation error with proper fields", func() {
			err := &ValidationError{
				EventStoreError: EventStoreError{
					Op:  "test",
					Err: fmt.Errorf("test error"),
				},
				Field: "testField",
				Value: "testValue",
			}

			Expect(err.Error()).To(ContainSubstring("test error"))
			Expect(err.Field).To(Equal("testField"))
			Expect(err.Value).To(Equal("testValue"))
		})

		It("should allow type assertion", func() {
			err := &ValidationError{
				EventStoreError: EventStoreError{
					Op:  "test",
					Err: fmt.Errorf("test error"),
				},
				Field: "testField",
				Value: "testValue",
			}

			var eventStoreErr error = err
			_, ok := eventStoreErr.(*ValidationError)
			Expect(ok).To(BeTrue())
		})
	})
})
