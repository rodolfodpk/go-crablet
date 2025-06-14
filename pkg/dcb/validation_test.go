package dcb

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Validation", func() {
	Describe("validateEvent", func() {
		It("should validate valid event", func() {
			event := InputEvent{
				Type: "TestEvent",
				Tags: NewTags("key", "value"),
				Data: toJSON(map[string]string{"data": "test"}),
			}

			err := validateEvent(event, 0)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should validate event with empty type", func() {
			event := InputEvent{
				Type: "",
				Tags: NewTags("key", "value"),
				Data: toJSON(map[string]string{"data": "test"}),
			}

			err := validateEvent(event, 0)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("empty type"))
		})

		It("should validate event with empty tag key", func() {
			event := InputEvent{
				Type: "TestEvent",
				Tags: []Tag{{Key: "", Value: "value"}},
				Data: toJSON(map[string]string{"data": "test"}),
			}

			err := validateEvent(event, 0)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("empty key"))
		})

		It("should validate event with empty tag value", func() {
			event := InputEvent{
				Type: "TestEvent",
				Tags: []Tag{{Key: "key", Value: ""}},
				Data: toJSON(map[string]string{"data": "test"}),
			}

			err := validateEvent(event, 0)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("empty value"))
		})

		It("should validate event with invalid JSON data", func() {
			event := InputEvent{
				Type: "TestEvent",
				Tags: NewTags("key", "value"),
				Data: []byte("invalid json"),
			}

			err := validateEvent(event, 0)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid JSON data"))
		})

		It("should validate event with empty data", func() {
			event := InputEvent{
				Type: "TestEvent",
				Tags: NewTags("key", "value"),
				Data: []byte{},
			}

			err := validateEvent(event, 0)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid JSON data"))
		})

		It("should validate event with nil data", func() {
			event := InputEvent{
				Type: "TestEvent",
				Tags: NewTags("key", "value"),
				Data: nil,
			}

			err := validateEvent(event, 0)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid JSON data"))
		})

		It("should validate event with empty tags", func() {
			event := InputEvent{
				Type: "TestEvent",
				Tags: []Tag{},
				Data: toJSON(map[string]string{"data": "test"}),
			}

			err := validateEvent(event, 0)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("empty tags"))
		})

		It("should validate event with multiple tags", func() {
			event := InputEvent{
				Type: "TestEvent",
				Tags: []Tag{
					{Key: "key1", Value: "value1"},
					{Key: "key2", Value: "value2"},
				},
				Data: toJSON(map[string]string{"data": "test"}),
			}

			err := validateEvent(event, 0)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should include index in error message", func() {
			event := InputEvent{
				Type: "",
				Tags: NewTags("key", "value"),
				Data: toJSON(map[string]string{"data": "test"}),
			}

			err := validateEvent(event, 5)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("event 5"))
		})
	})

	Describe("validateEvents", func() {
		It("should validate valid events slice", func() {
			events := []InputEvent{
				{
					Type: "Event1",
					Tags: NewTags("key1", "value1"),
					Data: toJSON(map[string]string{"data": "value1"}),
				},
				{
					Type: "Event2",
					Tags: NewTags("key2", "value2"),
					Data: toJSON(map[string]string{"data": "value2"}),
				},
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
				{
					Type: "Event1",
					Tags: NewTags("key1", "value1"),
					Data: toJSON(map[string]string{"data": "value1"}),
				},
				{
					Type: "", // Invalid: empty type
					Tags: NewTags("key2", "value2"),
					Data: toJSON(map[string]string{"data": "value2"}),
				},
				{
					Type: "Event3",
					Tags: []Tag{{Key: "", Value: "value3"}}, // Invalid: empty key
					Data: toJSON(map[string]string{"data": "value3"}),
				},
			}

			err := validateEvents(events)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("event 1"))
			Expect(err.Error()).To(ContainSubstring("empty type"))
		})
	})

	Describe("validateQueryTags", func() {
		It("should validate valid query", func() {
			query := Query{
				Items: []QueryItem{
					{
						EventTypes: []string{"Event1", "Event2"},
						Tags:       NewTags("key1", "value1"),
					},
					{
						EventTypes: []string{"Event3"},
						Tags:       NewTags("key2", "value2"),
					},
				},
			}

			err := validateQueryTags(query)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should validate empty query", func() {
			query := NewQueryEmpty()
			err := validateQueryTags(query)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should validate query with empty event types", func() {
			query := Query{
				Items: []QueryItem{
					{
						EventTypes: []string{""}, // Empty event type
						Tags:       NewTags("key", "value"),
					},
				},
			}

			err := validateQueryTags(query)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("empty event type"))
		})

		It("should validate query with empty tag keys", func() {
			query := Query{
				Items: []QueryItem{
					{
						EventTypes: []string{"Event1"},
						Tags:       []Tag{{Key: "", Value: "value"}}, // Empty key
					},
				},
			}

			err := validateQueryTags(query)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("empty key"))
		})

		It("should validate query with empty tag values", func() {
			query := Query{
				Items: []QueryItem{
					{
						EventTypes: []string{"Event1"},
						Tags:       []Tag{{Key: "key", Value: ""}}, // Empty value
					},
				},
			}

			err := validateQueryTags(query)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("empty value"))
		})

		It("should validate query with empty event types and tags", func() {
			query := Query{
				Items: []QueryItem{
					{
						EventTypes: []string{},
						Tags:       []Tag{},
					},
				},
			}

			err := validateQueryTags(query)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should validate query with multiple items", func() {
			query := Query{
				Items: []QueryItem{
					{
						EventTypes: []string{"Event1"},
						Tags:       NewTags("key1", "value1"),
					},
					{
						EventTypes: []string{"Event2"},
						Tags:       NewTags("key2", "value2"),
					},
				},
			}

			err := validateQueryTags(query)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("validateBatchSize", func() {
		var store EventStore
		var ctx context.Context

		BeforeEach(func() {
			// Use shared PostgreSQL container and truncate events between tests
			store = NewEventStoreFromPool(pool)
			ctx = context.Background()

			// Truncate events table before each test
			err := truncateEventsTable(ctx, pool)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should validate valid batch size", func() {
			es := store.(*eventStore)
			events := make([]InputEvent, 100)
			for i := 0; i < 100; i++ {
				events[i] = InputEvent{
					Type: "TestEvent",
					Tags: NewTags("key", fmt.Sprintf("value%d", i)),
					Data: toJSON(map[string]string{"data": fmt.Sprintf("value%d", i)}),
				}
			}

			err := es.validateBatchSize(events, "test")
			Expect(err).NotTo(HaveOccurred())
		})

		It("should validate batch size at limit", func() {
			es := store.(*eventStore)
			events := make([]InputEvent, 1000) // Default limit
			for i := 0; i < 1000; i++ {
				events[i] = InputEvent{
					Type: "TestEvent",
					Tags: NewTags("key", fmt.Sprintf("value%d", i)),
					Data: toJSON(map[string]string{"data": fmt.Sprintf("value%d", i)}),
				}
			}

			err := es.validateBatchSize(events, "test")
			Expect(err).NotTo(HaveOccurred())
		})

		It("should reject batch size exceeding limit", func() {
			es := store.(*eventStore)
			events := make([]InputEvent, 1001) // Exceeds default limit
			for i := 0; i < 1001; i++ {
				events[i] = InputEvent{
					Type: "TestEvent",
					Tags: NewTags("key", fmt.Sprintf("value%d", i)),
					Data: toJSON(map[string]string{"data": fmt.Sprintf("value%d", i)}),
				}
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
