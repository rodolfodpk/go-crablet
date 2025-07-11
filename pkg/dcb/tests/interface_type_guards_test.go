package dcb_test

import (
	"go-crablet/pkg/dcb"

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
	})

	Describe("Query helper functions", func() {
		It("should handle NewQueryAll", func() {
			query := dcb.NewQueryAll()
			Expect(query).NotTo(BeNil())

			items := query.GetItems()
			Expect(items).To(HaveLen(1))
			Expect(items[0].GetEventTypes()).To(BeEmpty())
			Expect(items[0].GetTags()).To(BeEmpty())
		})

		It("should handle NewQueryFromItems", func() {
			item1 := dcb.NewQueryItem([]string{"Event1"}, dcb.NewTags("key1", "value1"))
			item2 := dcb.NewQueryItem([]string{"Event2"}, dcb.NewTags("key2", "value2"))

			query := dcb.NewQueryFromItems(item1, item2)
			Expect(query).NotTo(BeNil())

			items := query.GetItems()
			Expect(items).To(HaveLen(2))
			Expect(items[0].GetEventTypes()).To(Equal([]string{"Event1"}))
			Expect(items[1].GetEventTypes()).To(Equal([]string{"Event2"}))
		})

		It("should handle NewQueryFromItems with empty items", func() {
			query := dcb.NewQueryFromItems()
			Expect(query).NotTo(BeNil())

			items := query.GetItems()
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

	Describe("Query item operations", func() {
		It("should handle query item with multiple event types", func() {
			eventTypes := []string{"Event1", "Event2", "Event3"}
			tags := dcb.NewTags("key1", "value1", "key2", "value2")

			item := dcb.NewQueryItem(eventTypes, tags)
			Expect(item.GetEventTypes()).To(Equal(eventTypes))
			Expect(item.GetTags()).To(Equal(tags))
		})

		It("should handle query item with empty event types", func() {
			eventTypes := []string{}
			tags := dcb.NewTags("key1", "value1")

			item := dcb.NewQueryItem(eventTypes, tags)
			Expect(item.GetEventTypes()).To(BeEmpty())
			Expect(item.GetTags()).To(Equal(tags))
		})

		It("should handle query item with empty tags", func() {
			eventTypes := []string{"Event1"}
			tags := []dcb.Tag{}

			item := dcb.NewQueryItem(eventTypes, tags)
			Expect(item.GetEventTypes()).To(Equal(eventTypes))
			Expect(item.GetTags()).To(BeEmpty())
		})
	})

	Describe("Query operations", func() {
		It("should handle query with multiple items", func() {
			item1 := dcb.NewQueryItem([]string{"Event1"}, dcb.NewTags("key1", "value1"))
			item2 := dcb.NewQueryItem([]string{"Event2"}, dcb.NewTags("key2", "value2"))

			query := dcb.NewQueryFromItems(item1, item2)
			items := query.GetItems()

			Expect(items).To(HaveLen(2))
			Expect(items[0].GetEventTypes()).To(Equal([]string{"Event1"}))
			Expect(items[1].GetEventTypes()).To(Equal([]string{"Event2"}))
		})

		It("should handle empty query", func() {
			query := dcb.NewQueryEmpty()
			items := query.GetItems()

			Expect(items).To(BeEmpty())
		})
	})
})
