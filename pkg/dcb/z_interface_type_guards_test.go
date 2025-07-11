package dcb

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Interface Type Guards and Edge Cases", func() {
	Describe("AppendCondition edge cases", func() {
		It("should handle nil FailIfEventsMatch in getFailIfEventsMatch", func() {
			// Create an append condition with nil FailIfEventsMatch
			condition := &appendCondition{
				FailIfEventsMatch: nil,
				AfterCursor:       &Cursor{TransactionID: 1, Position: 1},
			}

			result := condition.getFailIfEventsMatch()
			Expect(result).To(BeNil())
		})

		It("should handle non-nil FailIfEventsMatch in getFailIfEventsMatch", func() {
			// Create an append condition with non-nil FailIfEventsMatch
			query := NewQuery(NewTags("user_id", "123"), "UserCreated")
			condition := NewAppendCondition(query)

			result := condition.getFailIfEventsMatch()
			Expect(result).NotTo(BeNil())
		})

		It("should handle nil AfterCursor in getAfterCursor", func() {
			// Create an append condition with nil AfterCursor
			condition := &appendCondition{
				FailIfEventsMatch: nil,
				AfterCursor:       nil,
			}

			result := condition.getAfterCursor()
			Expect(result).To(BeNil())
		})

		It("should handle non-nil AfterCursor in getAfterCursor", func() {
			// Create an append condition with non-nil AfterCursor
			cursor := &Cursor{TransactionID: 1, Position: 1}
			condition := &appendCondition{
				FailIfEventsMatch: nil,
				AfterCursor:       cursor,
			}

			result := condition.getAfterCursor()
			Expect(result).To(Equal(cursor))
		})

		It("should set AfterCursor correctly", func() {
			// Create an append condition
			condition := &appendCondition{
				FailIfEventsMatch: nil,
				AfterCursor:       nil,
			}

			cursor := &Cursor{TransactionID: 2, Position: 2}
			condition.setAfterCursor(cursor)

			Expect(condition.AfterCursor).To(Equal(cursor))
		})
	})

	Describe("IsolationLevel parsing", func() {
		It("should handle invalid isolation level string", func() {
			level, err := ParseIsolationLevel("INVALID_LEVEL")

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid isolation level: INVALID_LEVEL"))
			Expect(level).To(Equal(IsolationLevelReadCommitted)) // Default fallback
		})

		It("should handle valid isolation level strings", func() {
			testCases := []struct {
				input    string
				expected IsolationLevel
			}{
				{"READ_COMMITTED", IsolationLevelReadCommitted},
				{"REPEATABLE_READ", IsolationLevelRepeatableRead},
				{"SERIALIZABLE", IsolationLevelSerializable},
			}

			for _, tc := range testCases {
				level, err := ParseIsolationLevel(tc.input)
				Expect(err).NotTo(HaveOccurred())
				Expect(level).To(Equal(tc.expected))
			}
		})

		It("should handle unknown isolation level in String()", func() {
			// Create an invalid isolation level
			invalidLevel := IsolationLevel(999)
			result := invalidLevel.String()
			Expect(result).To(Equal("UNKNOWN"))
		})
	})

	Describe("Query helper functions", func() {
		It("should handle NewQueryAll", func() {
			query := NewQueryAll()
			Expect(query).NotTo(BeNil())

			items := query.getItems()
			Expect(items).To(HaveLen(1))
			Expect(items[0].getEventTypes()).To(BeEmpty())
			Expect(items[0].getTags()).To(BeEmpty())
		})

		It("should handle NewQueryFromItems", func() {
			item1 := NewQueryItem([]string{"Event1"}, NewTags("key1", "value1"))
			item2 := NewQueryItem([]string{"Event2"}, NewTags("key2", "value2"))

			query := NewQueryFromItems(item1, item2)
			Expect(query).NotTo(BeNil())

			items := query.getItems()
			Expect(items).To(HaveLen(2))
			Expect(items[0].getEventTypes()).To(Equal([]string{"Event1"}))
			Expect(items[1].getEventTypes()).To(Equal([]string{"Event2"}))
		})

		It("should handle NewQueryFromItems with empty items", func() {
			query := NewQueryFromItems()
			Expect(query).NotTo(BeNil())

			items := query.getItems()
			Expect(items).To(BeEmpty())
		})
	})

	Describe("EventStoreConfig", func() {
		It("should return correct config from GetConfig", func() {
			// Use the global store from support_test.go
			config := store.GetConfig()
			Expect(config).NotTo(BeNil())
			Expect(config.MaxBatchSize).To(BeNumerically(">", 0))
			Expect(config.QueryTimeout).To(BeNumerically(">", 0))
		})
	})

	Describe("Tag MarshalJSON", func() {
		It("should marshal tag to correct JSON format", func() {
			t := &tag{key: "test_key", value: "test_value"}
			data, err := t.MarshalJSON()

			Expect(err).NotTo(HaveOccurred())
			Expect(string(data)).To(Equal(`{"key":"test_key","value":"test_value"}`))
		})
	})

	Describe("Query item operations", func() {
		It("should handle query item with multiple event types", func() {
			eventTypes := []string{"Event1", "Event2", "Event3"}
			tags := NewTags("key1", "value1", "key2", "value2")

			item := NewQueryItem(eventTypes, tags)
			Expect(item.getEventTypes()).To(Equal(eventTypes))
			Expect(item.getTags()).To(Equal(tags))
		})

		It("should handle query item with empty event types", func() {
			eventTypes := []string{}
			tags := NewTags("key1", "value1")

			item := NewQueryItem(eventTypes, tags)
			Expect(item.getEventTypes()).To(BeEmpty())
			Expect(item.getTags()).To(Equal(tags))
		})

		It("should handle query item with empty tags", func() {
			eventTypes := []string{"Event1"}
			tags := []Tag{}

			item := NewQueryItem(eventTypes, tags)
			Expect(item.getEventTypes()).To(Equal(eventTypes))
			Expect(item.getTags()).To(BeEmpty())
		})
	})

	Describe("Query operations", func() {
		It("should handle query with multiple items", func() {
			item1 := NewQueryItem([]string{"Event1"}, NewTags("key1", "value1"))
			item2 := NewQueryItem([]string{"Event2"}, NewTags("key2", "value2"))

			query := NewQueryFromItems(item1, item2)
			items := query.getItems()

			Expect(items).To(HaveLen(2))
			Expect(items[0].getEventTypes()).To(Equal([]string{"Event1"}))
			Expect(items[1].getEventTypes()).To(Equal([]string{"Event2"}))
		})

		It("should handle empty query", func() {
			query := NewQueryEmpty()
			items := query.getItems()

			Expect(items).To(BeEmpty())
		})
	})
})
