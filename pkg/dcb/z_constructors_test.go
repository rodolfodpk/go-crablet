package dcb

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5"
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
			_, err := NewEventStore(ctx, nil)
			Expect(err).To(HaveOccurred())
		})

		It("should fail NewEventStoreWithConfig with nil pool", func() {
			ctx := context.Background()
			config := EventStoreConfig{
				MaxBatchSize:           1000,
				LockTimeout:            5000,
				StreamBuffer:           1000,
				DefaultAppendIsolation: IsolationLevelReadCommitted,
			}

			// This will panic, so we need to recover
			defer func() {
				if r := recover(); r != nil {
					// Expected panic
				}
			}()
			_, err := NewEventStoreWithConfig(ctx, nil, config)
			Expect(err).To(HaveOccurred())
		})

		It("should create NewEventStoreFromPool with nil pool", func() {
			store := NewEventStoreFromPool(nil)
			Expect(store).NotTo(BeNil())

			// Test GetConfig method
			config := store.GetConfig()
			Expect(config.MaxBatchSize).To(Equal(1000))
			Expect(config.LockTimeout).To(Equal(5000))
			Expect(config.StreamBuffer).To(Equal(1000))
			Expect(config.DefaultAppendIsolation).To(Equal(IsolationLevelReadCommitted))
		})

		It("should create NewEventStoreFromPoolWithConfig with custom config", func() {
			config := EventStoreConfig{
				MaxBatchSize:           500,
				LockTimeout:            3000,
				StreamBuffer:           500,
				DefaultAppendIsolation: IsolationLevelSerializable,
			}

			store := NewEventStoreFromPoolWithConfig(nil, config)
			Expect(store).NotTo(BeNil())

			// Test GetConfig method
			actualConfig := store.GetConfig()
			Expect(actualConfig.MaxBatchSize).To(Equal(500))
			Expect(actualConfig.LockTimeout).To(Equal(3000))
			Expect(actualConfig.StreamBuffer).To(Equal(500))
			Expect(actualConfig.DefaultAppendIsolation).To(Equal(IsolationLevelSerializable))
		})
	})

	Describe("Event Constructors", func() {
		It("should create NewInputEvent with valid data", func() {
			tags := []Tag{NewTag("test", "value")}
			data := []byte(`{"key": "value"}`)

			event := NewInputEvent("TestEvent", tags, data)
			Expect(event).NotTo(BeNil())
			Expect(event.GetType()).To(Equal("TestEvent"))
			Expect(event.GetTags()).To(Equal(tags))
			Expect(event.GetData()).To(Equal(data))
		})

		It("should create NewInputEventUnsafe with valid data", func() {
			tags := []Tag{NewTag("test", "value")}
			data := []byte(`{"key": "value"}`)

			event := NewInputEventUnsafe("TestEvent", tags, data)
			Expect(event).NotTo(BeNil())
			Expect(event.GetType()).To(Equal("TestEvent"))
			Expect(event.GetTags()).To(Equal(tags))
			Expect(event.GetData()).To(Equal(data))
		})

		It("should create NewEventBatch with multiple events", func() {
			event1 := NewInputEvent("Event1", []Tag{NewTag("key1", "value1")}, []byte(`{"data": "1"}`))
			event2 := NewInputEvent("Event2", []Tag{NewTag("key2", "value2")}, []byte(`{"data": "2"}`))

			batch := NewEventBatch(event1, event2)
			Expect(batch).To(HaveLen(2))
			Expect(batch[0]).To(Equal(event1))
			Expect(batch[1]).To(Equal(event2))
		})

		It("should create empty NewEventBatch", func() {
			emptyBatch := NewEventBatch()
			Expect(emptyBatch).To(BeEmpty())
		})
	})

	Describe("Tag Constructors", func() {
		It("should create NewTag with valid key-value", func() {
			tag := NewTag("test", "value")
			Expect(tag).NotTo(BeNil())
			Expect(tag.GetKey()).To(Equal("test"))
			Expect(tag.GetValue()).To(Equal("value"))
		})

		It("should create NewTags with valid key-value pairs", func() {
			tags := NewTags("key1", "value1", "key2", "value2")
			Expect(tags).To(HaveLen(2))
			Expect(tags[0].GetKey()).To(Equal("key1"))
			Expect(tags[0].GetValue()).To(Equal("value1"))
			Expect(tags[1].GetKey()).To(Equal("key2"))
			Expect(tags[1].GetValue()).To(Equal("value2"))
		})

		It("should return empty slice for odd number of arguments in NewTags", func() {
			oddTags := NewTags("key1", "value1", "key2")
			Expect(oddTags).To(BeEmpty())
		})

		It("should return empty slice for no arguments in NewTags", func() {
			emptyTags := NewTags()
			Expect(emptyTags).To(BeEmpty())
		})
	})

	Describe("Query Constructors", func() {
		It("should create NewQuery with valid data", func() {
			tags := []Tag{NewTag("test", "value")}
			query := NewQuery(tags, "Event1")
			Expect(query).NotTo(BeNil())

			items := query.getItems()
			Expect(items).To(HaveLen(1))
			Expect(items[0].getEventTypes()).To(Equal([]string{"Event1"}))
			Expect(items[0].getTags()).To(Equal(tags))
		})

		It("should create NewQuery with valid data", func() {
			tags := []Tag{NewTag("test", "value")}
			query := NewQuery(tags, "Event1")
			Expect(query).NotTo(BeNil())

			items := query.getItems()
			Expect(items).To(HaveLen(1))
			Expect(items[0].getEventTypes()).To(Equal([]string{"Event1"}))
			Expect(items[0].getTags()).To(Equal(tags))
		})

		It("should create NewQueryEmpty", func() {
			query := NewQueryEmpty()
			Expect(query).NotTo(BeNil())

			items := query.getItems()
			Expect(items).To(BeEmpty())
		})

		It("should create NewQueryFromItems with multiple items", func() {
			item1 := NewQueryItem([]string{"Event1"}, []Tag{NewTag("key1", "value1")})
			item2 := NewQueryItem([]string{"Event2"}, []Tag{NewTag("key2", "value2")})

			query := NewQueryFromItems(item1, item2)
			Expect(query).NotTo(BeNil())

			items := query.getItems()
			Expect(items).To(HaveLen(2))
			Expect(items[0]).To(Equal(item1))
			Expect(items[1]).To(Equal(item2))
		})

		It("should create NewQueryAll", func() {
			query := NewQueryAll()
			Expect(query).NotTo(BeNil())

			items := query.getItems()
			Expect(items).To(HaveLen(1))
			Expect(items[0].getEventTypes()).To(BeEmpty())
			Expect(items[0].getTags()).To(BeEmpty())
		})

		It("should create NewQueryItem with valid data", func() {
			types := []string{"Event1", "Event2"}
			tags := []Tag{NewTag("test", "value")}

			item := NewQueryItem(types, tags)
			Expect(item).NotTo(BeNil())
			Expect(item.getEventTypes()).To(Equal(types))
			Expect(item.getTags()).To(Equal(tags))
		})

		It("should create NewQueryItem with single event type", func() {
			tags := []Tag{NewTag("test", "value")}

			item := NewQueryItem([]string{"Event1"}, tags)
			Expect(item).NotTo(BeNil())
			Expect(item.getEventTypes()).To(Equal([]string{"Event1"}))
			Expect(item.getTags()).To(Equal(tags))
		})

		It("should create NewQueryItem with key-value tags", func() {
			item := NewQueryItem([]string{"Event1"}, NewTags("key1", "value1", "key2", "value2"))
			Expect(item).NotTo(BeNil())
			Expect(item.getEventTypes()).To(Equal([]string{"Event1"}))

			tags := item.getTags()
			Expect(tags).To(HaveLen(2))
			Expect(tags[0].GetKey()).To(Equal("key1"))
			Expect(tags[0].GetValue()).To(Equal("value1"))
			Expect(tags[1].GetKey()).To(Equal("key2"))
			Expect(tags[1].GetValue()).To(Equal("value2"))
		})
	})

	Describe("AppendCondition Constructors", func() {
		It("should create NewAppendCondition with valid query", func() {
			query := NewQuery([]Tag{NewTag("test", "value")}, "TestEvent")
			condition := NewAppendCondition(query)

			// Verify the condition is not nil
			Expect(condition).NotTo(BeNil())
		})

		It("should create NewAppendCondition with nil query", func() {
			condition := NewAppendCondition(nil)
			Expect(condition).NotTo(BeNil())
		})
	})

	Describe("Interface Implementations", func() {
		It("should implement InputEvent interface", func() {
			event := NewInputEvent("Test", []Tag{NewTag("key", "value")}, []byte(`{}`))
			Expect(event).NotTo(BeNil())
			Expect(event.GetType()).To(Equal("Test"))
		})

		It("should implement Tag interface", func() {
			tag := NewTag("key", "value")
			Expect(tag).NotTo(BeNil())
			Expect(tag.GetKey()).To(Equal("key"))
		})

		It("should implement Query interface", func() {
			query := NewQuery([]Tag{NewTag("key", "value")}, "Test")
			Expect(query).NotTo(BeNil())
			Expect(query.getItems()).To(HaveLen(1))
		})

		It("should implement QueryItem interface", func() {
			item := NewQueryItem([]string{"Test"}, []Tag{NewTag("key", "value")})
			Expect(item).NotTo(BeNil())
			Expect(item.getEventTypes()).To(Equal([]string{"Test"}))
		})

		It("should implement AppendCondition interface", func() {
			query := NewQuery([]Tag{NewTag("key", "value")}, "Test")
			condition := NewAppendCondition(query)
			Expect(condition).NotTo(BeNil())
		})
	})
})

var _ = Describe("IsolationLevel", func() {
	Describe("String()", func() {
		It("should return correct string for ReadCommitted", func() {
			Expect(IsolationLevelReadCommitted.String()).To(Equal("READ_COMMITTED"))
		})

		It("should return correct string for RepeatableRead", func() {
			Expect(IsolationLevelRepeatableRead.String()).To(Equal("REPEATABLE_READ"))
		})

		It("should return correct string for Serializable", func() {
			Expect(IsolationLevelSerializable.String()).To(Equal("SERIALIZABLE"))
		})

		It("should return UNKNOWN for invalid level", func() {
			invalidLevel := IsolationLevel(999)
			Expect(invalidLevel.String()).To(Equal("UNKNOWN"))
		})
	})

	Describe("ParseIsolationLevel()", func() {
		It("should parse READ_COMMITTED correctly", func() {
			level, err := ParseIsolationLevel("READ_COMMITTED")
			Expect(err).To(BeNil())
			Expect(level).To(Equal(IsolationLevelReadCommitted))
		})

		It("should parse REPEATABLE_READ correctly", func() {
			level, err := ParseIsolationLevel("REPEATABLE_READ")
			Expect(err).To(BeNil())
			Expect(level).To(Equal(IsolationLevelRepeatableRead))
		})

		It("should parse SERIALIZABLE correctly", func() {
			level, err := ParseIsolationLevel("SERIALIZABLE")
			Expect(err).To(BeNil())
			Expect(level).To(Equal(IsolationLevelSerializable))
		})

		It("should return error for invalid level", func() {
			level, err := ParseIsolationLevel("INVALID_LEVEL")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid isolation level"))
			Expect(level).To(Equal(IsolationLevelReadCommitted))
		})
	})
})

var _ = Describe("Interface Methods", func() {
	Describe("InputEvent interface", func() {
		It("should implement all required methods", func() {
			event := NewInputEvent("TestEvent", NewTags("key", "value"), []byte(`{"data": "test"}`))

			// Test that it implements the interface
			var _ InputEvent = event

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
			tag := NewTag("key", "value")

			// Test that it implements the interface
			var _ Tag = tag

			// Test getter methods
			Expect(tag.GetKey()).To(Equal("key"))
			Expect(tag.GetValue()).To(Equal("value"))
		})

		It("should marshal to JSON correctly", func() {
			tag := NewTag("key", "value")
			jsonData, err := json.Marshal(tag)
			Expect(err).To(BeNil())
			Expect(string(jsonData)).To(Equal(`{"key":"key","value":"value"}`))
		})
	})

	Describe("Query interface", func() {
		It("should implement all required methods", func() {
			query := NewQuery(NewTags("key", "value"), "TestEvent")

			// Test that it implements the interface
			var _ Query = query

			// Test getter methods
			items := query.getItems()
			Expect(items).To(HaveLen(1))
			Expect(items[0].getEventTypes()).To(Equal([]string{"TestEvent"}))
			Expect(items[0].getTags()).To(HaveLen(1))
			Expect(items[0].getTags()[0].GetKey()).To(Equal("key"))
			Expect(items[0].getTags()[0].GetValue()).To(Equal("value"))
		})
	})

	Describe("AppendCondition interface", func() {
		It("should implement all required methods", func() {
			query := NewQuery(NewTags("key", "value"), "TestEvent")
			condition := NewAppendCondition(query)

			// Test that it implements the interface
			var _ AppendCondition = condition

			// Test getter methods
			failQuery := condition.getFailIfEventsMatch()
			Expect(failQuery).ToNot(BeNil())
			Expect((*failQuery).getItems()).To(HaveLen(1))

			// Test cursor methods
			cursor := &Cursor{TransactionID: 123, Position: 456}
			condition.setAfterCursor(cursor)
			Expect(condition.getAfterCursor()).To(Equal(cursor))

			// Test nil cursor
			condition.setAfterCursor(nil)
			Expect(condition.getAfterCursor()).To(BeNil())
		})
	})
})

var _ = Describe("toPgxIsoLevel", func() {
	It("should convert ReadCommitted correctly", func() {
		level := toPgxIsoLevel(IsolationLevelReadCommitted)
		Expect(level).To(Equal(pgx.ReadCommitted))
	})

	It("should convert RepeatableRead correctly", func() {
		level := toPgxIsoLevel(IsolationLevelRepeatableRead)
		Expect(level).To(Equal(pgx.RepeatableRead))
	})

	It("should convert Serializable correctly", func() {
		level := toPgxIsoLevel(IsolationLevelSerializable)
		Expect(level).To(Equal(pgx.Serializable))
	})

	It("should default to ReadCommitted for invalid level", func() {
		invalidLevel := IsolationLevel(999)
		level := toPgxIsoLevel(invalidLevel)
		Expect(level).To(Equal(pgx.ReadCommitted))
	})
})
