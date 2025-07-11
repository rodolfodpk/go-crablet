package dcb_test

import (
	"context"
	"encoding/json"

	"go-crablet/pkg/dcb"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Constructor Tests", func() {
	Describe("EventStore Constructors", func() {
		It("should fail NewEventStore with nil pool", func() {
			ctx := context.Background()
			// This will panic, so we need to recover
			defer func() {
				if r := recover(); r != nil {
					// Expected panic
				}
			}()
			_, err := dcb.NewEventStore(ctx, nil)
			Expect(err).To(HaveOccurred())
		})

		It("should fail NewEventStoreWithConfig with nil pool", func() {
			ctx := context.Background()
			config := dcb.EventStoreConfig{
				MaxBatchSize:           1000,
				LockTimeout:            5000,
				StreamBuffer:           1000,
				DefaultAppendIsolation: dcb.IsolationLevelReadCommitted,
			}

			// This will panic, so we need to recover
			defer func() {
				if r := recover(); r != nil {
					// Expected panic
				}
			}()
			_, err := dcb.NewEventStoreWithConfig(ctx, nil, config)
			Expect(err).To(HaveOccurred())
		})

		It("should create NewEventStoreFromPool with nil pool", func() {
			store := dcb.NewEventStoreFromPool(nil)
			Expect(store).NotTo(BeNil())

			// Test GetConfig method
			config := store.GetConfig()
			Expect(config.MaxBatchSize).To(Equal(1000))
			Expect(config.LockTimeout).To(Equal(5000))
			Expect(config.StreamBuffer).To(Equal(1000))
			Expect(config.DefaultAppendIsolation).To(Equal(dcb.IsolationLevelReadCommitted))
		})

		It("should create NewEventStoreFromPoolWithConfig with custom config", func() {
			config := dcb.EventStoreConfig{
				MaxBatchSize:           500,
				LockTimeout:            3000,
				StreamBuffer:           500,
				DefaultAppendIsolation: dcb.IsolationLevelSerializable,
			}

			store := dcb.NewEventStoreFromPoolWithConfig(nil, config)
			Expect(store).NotTo(BeNil())

			// Test GetConfig method
			actualConfig := store.GetConfig()
			Expect(actualConfig.MaxBatchSize).To(Equal(500))
			Expect(actualConfig.LockTimeout).To(Equal(3000))
			Expect(actualConfig.StreamBuffer).To(Equal(500))
			Expect(actualConfig.DefaultAppendIsolation).To(Equal(dcb.IsolationLevelSerializable))
		})
	})

	Describe("Event Constructors", func() {
		It("should create NewInputEvent with valid data", func() {
			tags := []dcb.Tag{dcb.NewTag("test", "value")}
			data := []byte(`{"key": "value"}`)

			event := dcb.NewInputEvent("TestEvent", tags, data)
			Expect(event).NotTo(BeNil())
			Expect(event.GetType()).To(Equal("TestEvent"))
			Expect(event.GetTags()).To(Equal(tags))
			Expect(event.GetData()).To(Equal(data))
		})

		It("should create NewInputEventUnsafe with valid data", func() {
			tags := []dcb.Tag{dcb.NewTag("test", "value")}
			data := []byte(`{"key": "value"}`)

			event := dcb.NewInputEventUnsafe("TestEvent", tags, data)
			Expect(event).NotTo(BeNil())
			Expect(event.GetType()).To(Equal("TestEvent"))
			Expect(event.GetTags()).To(Equal(tags))
			Expect(event.GetData()).To(Equal(data))
		})

		It("should create NewEventBatch with multiple events", func() {
			event1 := dcb.NewInputEvent("Event1", []dcb.Tag{dcb.NewTag("key1", "value1")}, []byte(`{"data": "1"}`))
			event2 := dcb.NewInputEvent("Event2", []dcb.Tag{dcb.NewTag("key2", "value2")}, []byte(`{"data": "2"}`))

			batch := dcb.NewEventBatch(event1, event2)
			Expect(batch).To(HaveLen(2))
			Expect(batch[0]).To(Equal(event1))
			Expect(batch[1]).To(Equal(event2))
		})

		It("should create empty NewEventBatch", func() {
			emptyBatch := dcb.NewEventBatch()
			Expect(emptyBatch).To(BeEmpty())
		})
	})

	Describe("Tag Constructors", func() {
		It("should create NewTag with valid key-value", func() {
			tag := dcb.NewTag("test", "value")
			Expect(tag).NotTo(BeNil())
			Expect(tag.GetKey()).To(Equal("test"))
			Expect(tag.GetValue()).To(Equal("value"))
		})

		It("should create NewTags with valid key-value pairs", func() {
			tags := dcb.NewTags("key1", "value1", "key2", "value2")
			Expect(tags).To(HaveLen(2))
			Expect(tags[0].GetKey()).To(Equal("key1"))
			Expect(tags[0].GetValue()).To(Equal("value1"))
			Expect(tags[1].GetKey()).To(Equal("key2"))
			Expect(tags[1].GetValue()).To(Equal("value2"))
		})

		It("should return empty slice for odd number of arguments in NewTags", func() {
			oddTags := dcb.NewTags("key1", "value1", "key2")
			Expect(oddTags).To(BeEmpty())
		})

		It("should return empty slice for no arguments in NewTags", func() {
			emptyTags := dcb.NewTags()
			Expect(emptyTags).To(BeEmpty())
		})
	})

	Describe("Query Constructors", func() {
		It("should create NewQuery with valid data", func() {
			tags := []dcb.Tag{dcb.NewTag("test", "value")}
			query := dcb.NewQuery(tags, "Event1")
			Expect(query).NotTo(BeNil())
		})

		It("should create NewQueryEmpty", func() {
			query := dcb.NewQueryEmpty()
			Expect(query).NotTo(BeNil())
		})

		It("should create NewQueryFromItems with multiple items", func() {
			item1 := dcb.NewQueryItem([]string{"Event1"}, []dcb.Tag{dcb.NewTag("key1", "value1")})
			item2 := dcb.NewQueryItem([]string{"Event2"}, []dcb.Tag{dcb.NewTag("key2", "value2")})

			query := dcb.NewQueryFromItems(item1, item2)
			Expect(query).NotTo(BeNil())
		})

		It("should create NewQueryAll", func() {
			query := dcb.NewQueryAll()
			Expect(query).NotTo(BeNil())
		})

		It("should create NewQueryItem with valid data", func() {
			types := []string{"Event1", "Event2"}
			tags := []dcb.Tag{dcb.NewTag("test", "value")}

			item := dcb.NewQueryItem(types, tags)
			Expect(item).NotTo(BeNil())
		})
	})

	Describe("AppendCondition Constructors", func() {
		It("should create NewAppendCondition with valid query", func() {
			query := dcb.NewQuery([]dcb.Tag{dcb.NewTag("test", "value")}, "TestEvent")
			condition := dcb.NewAppendCondition(query)

			// Verify the condition is not nil
			Expect(condition).NotTo(BeNil())
		})

		It("should create NewAppendCondition with nil query", func() {
			condition := dcb.NewAppendCondition(nil)
			Expect(condition).NotTo(BeNil())
		})
	})

	Describe("Interface Implementations", func() {
		It("should implement InputEvent interface", func() {
			event := dcb.NewInputEvent("Test", []dcb.Tag{dcb.NewTag("key", "value")}, []byte(`{}`))
			Expect(event).NotTo(BeNil())
			Expect(event.GetType()).To(Equal("Test"))
		})

		It("should implement Tag interface", func() {
			tag := dcb.NewTag("key", "value")
			Expect(tag).NotTo(BeNil())
			Expect(tag.GetKey()).To(Equal("key"))
		})

		It("should implement Query interface", func() {
			query := dcb.NewQuery([]dcb.Tag{dcb.NewTag("key", "value")}, "Test")
			Expect(query).NotTo(BeNil())
		})

		It("should implement QueryItem interface", func() {
			item := dcb.NewQueryItem([]string{"Test"}, []dcb.Tag{dcb.NewTag("key", "value")})
			Expect(item).NotTo(BeNil())
		})

		It("should implement AppendCondition interface", func() {
			query := dcb.NewQuery([]dcb.Tag{dcb.NewTag("key", "value")}, "Test")
			condition := dcb.NewAppendCondition(query)
			Expect(condition).NotTo(BeNil())
		})
	})

	Describe("IsolationLevel String()", func() {
		It("should return correct string for ReadCommitted", func() {
			Expect(dcb.IsolationLevelReadCommitted.String()).To(Equal("READ_COMMITTED"))
		})

		It("should return correct string for RepeatableRead", func() {
			Expect(dcb.IsolationLevelRepeatableRead.String()).To(Equal("REPEATABLE_READ"))
		})

		It("should return correct string for Serializable", func() {
			Expect(dcb.IsolationLevelSerializable.String()).To(Equal("SERIALIZABLE"))
		})

		It("should return UNKNOWN for invalid level", func() {
			invalidLevel := dcb.IsolationLevel(999)
			Expect(invalidLevel.String()).To(Equal("UNKNOWN"))
		})
	})

	Describe("ParseIsolationLevel()", func() {
		It("should parse READ_COMMITTED correctly", func() {
			level, err := dcb.ParseIsolationLevel("READ_COMMITTED")
			Expect(err).To(BeNil())
			Expect(level).To(Equal(dcb.IsolationLevelReadCommitted))
		})

		It("should parse REPEATABLE_READ correctly", func() {
			level, err := dcb.ParseIsolationLevel("REPEATABLE_READ")
			Expect(err).To(BeNil())
			Expect(level).To(Equal(dcb.IsolationLevelRepeatableRead))
		})

		It("should parse SERIALIZABLE correctly", func() {
			level, err := dcb.ParseIsolationLevel("SERIALIZABLE")
			Expect(err).To(BeNil())
			Expect(level).To(Equal(dcb.IsolationLevelSerializable))
		})

		It("should return error for invalid level", func() {
			level, err := dcb.ParseIsolationLevel("INVALID_LEVEL")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid isolation level"))
			Expect(level).To(Equal(dcb.IsolationLevelReadCommitted))
		})
	})

	Describe("Interface Methods", func() {
		Describe("InputEvent interface", func() {
			It("should implement all required methods", func() {
				event := dcb.NewInputEvent("TestEvent", dcb.NewTags("key", "value"), []byte(`{"data": "test"}`))

				// Test that it implements the interface
				var _ dcb.InputEvent = event

				// Test getter methods
				Expect(event.GetType()).To(Equal("TestEvent"))
				Expect(event.GetTags()).To(HaveLen(1))
				Expect(event.GetTags()[0].GetKey()).To(Equal("key"))
				Expect(event.GetTags()[0].GetValue()).To(Equal("value"))
				Expect(event.GetData()).To(Equal([]byte(`{"data": "test"}`)))
			})
		})

		Describe("Tag interface", func() {
			It("should implement all required methods", func() {
				tag := dcb.NewTag("key", "value")

				// Test that it implements the interface
				var _ dcb.Tag = tag

				// Test getter methods
				Expect(tag.GetKey()).To(Equal("key"))
				Expect(tag.GetValue()).To(Equal("value"))
			})

			It("should marshal to JSON correctly", func() {
				tag := dcb.NewTag("key", "value")
				jsonData, err := json.Marshal(tag)
				Expect(err).To(BeNil())
				Expect(string(jsonData)).To(Equal(`{"key":"key","value":"value"}`))
			})
		})

		Describe("Query interface", func() {
			It("should implement all required methods", func() {
				query := dcb.NewQuery(dcb.NewTags("key", "value"), "TestEvent")

				// Test that it implements the interface
				var _ dcb.Query = query

				// Test that it's not nil
				Expect(query).NotTo(BeNil())
			})
		})

		Describe("AppendCondition interface", func() {
			It("should implement all required methods", func() {
				query := dcb.NewQuery(dcb.NewTags("key", "value"), "TestEvent")
				condition := dcb.NewAppendCondition(query)

				// Test that it implements the interface
				var _ dcb.AppendCondition = condition

				// Test that it's not nil
				Expect(condition).NotTo(BeNil())
			})
		})
	})
})
