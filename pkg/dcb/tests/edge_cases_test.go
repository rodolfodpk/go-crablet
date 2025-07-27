package dcb

import (
	"github.com/rodolfodpk/go-crablet/pkg/dcb"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Interface Type Guards and Edge Cases", func() {
	Describe("IsolationLevel parsing", func() {
		It("should handle invalid isolation level string", func() {
			level, err := dcb.ParseIsolationLevel("INVALID_LEVEL")

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid isolation level: INVALID_LEVEL"))
			Expect(level).To(Equal(dcb.IsolationLevelReadCommitted)) // Default fallback
		})

		It("should handle valid isolation level strings", func() {
			testCases := []struct {
				input    string
				expected dcb.IsolationLevel
			}{
				{"READ_COMMITTED", dcb.IsolationLevelReadCommitted},
				{"REPEATABLE_READ", dcb.IsolationLevelRepeatableRead},
				{"SERIALIZABLE", dcb.IsolationLevelSerializable},
			}

			for _, tc := range testCases {
				level, err := dcb.ParseIsolationLevel(tc.input)
				Expect(err).NotTo(HaveOccurred())
				Expect(level).To(Equal(tc.expected))
			}
		})

		It("should handle unknown isolation level in String()", func() {
			// Create an invalid isolation level
			invalidLevel := dcb.IsolationLevel(999)
			result := invalidLevel.String()
			Expect(result).To(Equal("UNKNOWN"))
		})

		It("should handle edge case isolation level values", func() {
			// Test boundary values
			level0 := dcb.IsolationLevel(0)
			Expect(level0.String()).To(Equal("READ_COMMITTED"))

			level2 := dcb.IsolationLevel(2)
			Expect(level2.String()).To(Equal("SERIALIZABLE"))

			level3 := dcb.IsolationLevel(3)
			Expect(level3.String()).To(Equal("UNKNOWN"))
		})
	})

	Describe("Input Validation Edge Cases", func() {
		It("should handle empty event type", func() {
			event := dcb.NewInputEvent("", dcb.NewTags("key", "value"), dcb.ToJSON(map[string]string{"data": "test"}))
			Expect(event.GetType()).To(Equal(""))
			Expect(event.GetTags()).To(Equal(dcb.NewTags("key", "value")))
		})

		It("should handle whitespace-only event type", func() {
			event := dcb.NewInputEvent("   ", dcb.NewTags("key", "value"), dcb.ToJSON(map[string]string{"data": "test"}))
			Expect(event.GetType()).To(Equal("   "))
		})

		It("should handle very long event type", func() {
			longType := "A" + string(make([]byte, 63)) // 64 characters total
			event := dcb.NewInputEvent(longType, dcb.NewTags("key", "value"), dcb.ToJSON(map[string]string{"data": "test"}))
			Expect(event.GetType()).To(Equal(longType))
		})

		It("should handle empty tag key", func() {
			tags := []dcb.Tag{dcb.NewTag("", "value")}
			event := dcb.NewInputEvent("TestEvent", tags, dcb.ToJSON(map[string]string{"data": "test"}))
			Expect(event.GetTags()[0].GetKey()).To(Equal(""))
		})

		It("should handle empty tag value", func() {
			tags := []dcb.Tag{dcb.NewTag("key", "")}
			event := dcb.NewInputEvent("TestEvent", tags, dcb.ToJSON(map[string]string{"data": "test"}))
			Expect(event.GetTags()[0].GetValue()).To(Equal(""))
		})

		It("should handle special characters in tag keys and values", func() {
			specialKey := "key-with-special-chars:!@#$%^&*()"
			specialValue := "value-with-special-chars:!@#$%^&*()"
			tags := []dcb.Tag{dcb.NewTag(specialKey, specialValue)}
			event := dcb.NewInputEvent("TestEvent", tags, dcb.ToJSON(map[string]string{"data": "test"}))
			Expect(event.GetTags()[0].GetKey()).To(Equal(specialKey))
			Expect(event.GetTags()[0].GetValue()).To(Equal(specialValue))
		})

		It("should handle nil data", func() {
			event := dcb.NewInputEvent("TestEvent", dcb.NewTags("key", "value")
			Expect(event.GetData()).To(BeNil())
		})

		It("should handle empty data", func() {
			event := dcb.NewInputEvent("TestEvent", dcb.NewTags("key", "value"), []byte{})
			Expect(event.GetData()).To(Equal([]byte{}))
		})
	})

	Describe("Query Item Edge Cases", func() {
		It("should handle query item with nil event types slice", func() {
			var eventTypes []string
			tags := dcb.NewTags("key1", "value1")
			item := dcb.NewQueryItem(eventTypes, tags)
			Expect(item.GetEventTypes()).To(BeNil())
		})

		It("should handle query item with nil tags slice", func() {
			eventTypes := []string{"Event1"}
			var tags []dcb.Tag
			item := dcb.NewQueryItem(eventTypes, tags)
			Expect(item.GetTags()).To(BeNil())
		})

		It("should handle query item with empty event types", func() {
			eventTypes := []string{}
			tags := dcb.NewTags("key1", "value1")
			item := dcb.NewQueryItem(eventTypes, tags)
			Expect(item.GetEventTypes()).To(BeEmpty())
		})

		It("should handle query item with empty tags", func() {
			eventTypes := []string{"Event1"}
			tags := []dcb.Tag{}
			item := dcb.NewQueryItem(eventTypes, tags)
			Expect(item.GetTags()).To(BeEmpty())
		})

		It("should handle query item with duplicate event types", func() {
			eventTypes := []string{"Event1", "Event1", "Event2"}
			tags := dcb.NewTags("key1", "value1")
			item := dcb.NewQueryItem(eventTypes, tags)
			Expect(item.GetEventTypes()).To(Equal([]string{"Event1", "Event1", "Event2"}))
		})
	})

	Describe("EventStoreConfig Edge Cases", func() {
		It("should return correct config from GetConfig", func() {
			// Use the global store from support_test.go
			config := store.GetConfig()
			Expect(config).NotTo(BeNil())
			Expect(config.MaxBatchSize).To(BeNumerically(">", 0))
			Expect(config.QueryTimeout).To(BeNumerically(">", 0))
		})

		It("should handle zero values in config", func() {
			config := dcb.EventStoreConfig{
				MaxBatchSize:           0,
				LockTimeout:            0,
				StreamBuffer:           0,
				DefaultAppendIsolation: dcb.IsolationLevelReadCommitted,
				QueryTimeout:           0,
			}
			Expect(config.MaxBatchSize).To(Equal(0))
			Expect(config.LockTimeout).To(Equal(0))
			Expect(config.StreamBuffer).To(Equal(0))
			Expect(config.QueryTimeout).To(Equal(0))
		})

		It("should handle extreme values in config", func() {
			config := dcb.EventStoreConfig{
				MaxBatchSize:           999999,
				LockTimeout:            999999,
				StreamBuffer:           999999,
				DefaultAppendIsolation: dcb.IsolationLevelSerializable,
				QueryTimeout:           999999,
			}
			Expect(config.MaxBatchSize).To(Equal(999999))
			Expect(config.LockTimeout).To(Equal(999999))
			Expect(config.StreamBuffer).To(Equal(999999))
			Expect(config.QueryTimeout).To(Equal(999999))
		})

		It("should create EventStore with custom config using NewEventStoreFromPoolWithConfig", func() {
			customConfig := dcb.EventStoreConfig{
				MaxBatchSize:           500,
				LockTimeout:            3000,
				StreamBuffer:           500,
				DefaultAppendIsolation: dcb.IsolationLevelRepeatableRead,
				QueryTimeout:           8000,
				AppendTimeout:          6000,
			}

			// Create EventStore with custom config
			customStore := dcb.NewEventStoreFromPoolWithConfig(pool, customConfig)
			Expect(customStore).NotTo(BeNil())

			// Verify the config was applied correctly
			retrievedConfig := customStore.GetConfig()
			Expect(retrievedConfig.MaxBatchSize).To(Equal(500))
			Expect(retrievedConfig.LockTimeout).To(Equal(3000))
			Expect(retrievedConfig.StreamBuffer).To(Equal(500))
			Expect(retrievedConfig.DefaultAppendIsolation).To(Equal(dcb.IsolationLevelRepeatableRead))
			Expect(retrievedConfig.QueryTimeout).To(Equal(8000))
			Expect(retrievedConfig.AppendTimeout).To(Equal(6000))

			// Verify it can perform basic operations
			event := dcb.NewInputEvent("TestEvent", dcb.NewTags("key", "value"), dcb.ToJSON(map[string]string{"data": "test"}))
			err := customStore.Append(ctx, []dcb.InputEvent{event})
			Expect(err).To(BeNil())

			// Query the event back
			query := dcb.NewQuery(dcb.NewTags("key", "value"), "TestEvent")
			events, err := customStore.Query(ctx, query
			Expect(err).To(BeNil())
			Expect(events).To(HaveLen(1))
			Expect(events[0].Type).To(Equal("TestEvent"))
		})
	})

	Describe("Tag Constructor Edge Cases", func() {
		It("should handle NewTags with odd number of arguments", func() {
			// This should return empty tags instead of panicking
			tags := dcb.NewTags("key1", "value1", "key2") // Odd number
			Expect(tags).To(BeEmpty())
		})

		It("should handle NewTags with empty arguments", func() {
			tags := dcb.NewTags()
			Expect(tags).To(BeEmpty())
		})

		It("should handle NewTags with empty key-value pairs", func() {
			tags := dcb.NewTags("", "", "", "")
			Expect(tags).To(HaveLen(2))
			Expect(tags[0].GetKey()).To(Equal(""))
			Expect(tags[0].GetValue()).To(Equal(""))
			Expect(tags[1].GetKey()).To(Equal(""))
			Expect(tags[1].GetValue()).To(Equal(""))
		})

		It("should handle NewTag with empty strings", func() {
			tag := dcb.NewTag("", "")
			Expect(tag.GetKey()).To(Equal(""))
			Expect(tag.GetValue()).To(Equal(""))
		})
	})

	Describe("Query Constructor Edge Cases", func() {
		It("should handle NewQuery with nil tags", func() {
			var tags []dcb.Tag
			query := dcb.NewQuery(tags, "Event1")
			Expect(query).NotTo(BeNil())
			items := query.GetItems()
			Expect(items).To(HaveLen(1))
			Expect(items[0].GetTags()).To(BeNil())
		})

		It("should handle NewQuery with nil event types", func() {
			tags := dcb.NewTags("key1", "value1")
			query := dcb.NewQuery(tags)
			Expect(query).NotTo(BeNil())
			items := query.GetItems()
			Expect(items).To(HaveLen(1))
			Expect(items[0].GetEventTypes()).To(BeEmpty())
		})

		It("should handle NewQuery with empty event types", func() {
			tags := dcb.NewTags("key1", "value1")
			query := dcb.NewQuery(tags, "")
			Expect(query).NotTo(BeNil())
			items := query.GetItems()
			Expect(items).To(HaveLen(1))
			Expect(items[0].GetEventTypes()).To(Equal([]string{""}))
		})
	})

	Describe("Interface Contract Edge Cases", func() {
		It("should handle empty query items", func() {
			query := dcb.NewQueryEmpty()
			items := query.GetItems()
			Expect(items).To(BeEmpty())
		})

		It("should handle query with multiple items", func() {
			item1 := dcb.NewQueryItem([]string{"Event1"}, dcb.NewTags("key1", "value1"))
			item2 := dcb.NewQueryItem([]string{"Event2"}, dcb.NewTags("key2", "value2"))

			query := dcb.NewQueryFromItems(item1, item2)
			items := query.GetItems()

			Expect(items).To(HaveLen(2))
			Expect(items[0].GetEventTypes()).To(Equal([]string{"Event1"}))
			Expect(items[1].GetEventTypes()).To(Equal([]string{"Event2"}))
		})

		It("should handle NewQueryAll", func() {
			query := dcb.NewQueryAll()
			Expect(query).NotTo(BeNil())

			items := query.GetItems()
			Expect(items).To(HaveLen(1))
			Expect(items[0].GetEventTypes()).To(BeEmpty())
			Expect(items[0].GetTags()).To(BeEmpty())
		})
	})
})
