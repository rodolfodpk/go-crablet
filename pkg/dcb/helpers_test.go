package dcb

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Helper Functions", func() {
	Describe("NewQueryFromItems", func() {
		It("should create query from multiple items", func() {
			item1 := QueryItem{
				EventTypes: []string{"Event1", "Event2"},
				Tags:       []Tag{{Key: "key1", Value: "value1"}},
			}
			item2 := QueryItem{
				EventTypes: []string{"Event3"},
				Tags:       []Tag{{Key: "key2", Value: "value2"}},
			}

			query := NewQueryFromItems(item1, item2)

			Expect(query.Items).To(HaveLen(2))
			Expect(query.Items[0]).To(Equal(item1))
			Expect(query.Items[1]).To(Equal(item2))
		})

		It("should create empty query when no items provided", func() {
			query := NewQueryFromItems()

			Expect(query.Items).To(BeEmpty())
		})
	})

	Describe("NewQueryAll", func() {
		It("should create query that matches all events", func() {
			query := NewQueryAll()

			Expect(query.Items).To(HaveLen(1))
			Expect(query.Items[0].EventTypes).To(BeEmpty())
			Expect(query.Items[0].Tags).To(BeEmpty())
		})
	})

	Describe("NewQueryItem", func() {
		It("should create query item with types and tags", func() {
			types := []string{"Event1", "Event2"}
			tags := []Tag{{Key: "key1", Value: "value1"}}

			item := NewQueryItem(types, tags)

			Expect(item.EventTypes).To(Equal(types))
			Expect(item.Tags).To(Equal(tags))
		})

		It("should create query item with empty slices", func() {
			item := NewQueryItem([]string{}, []Tag{})

			Expect(item.EventTypes).To(BeEmpty())
			Expect(item.Tags).To(BeEmpty())
		})
	})

	Describe("NewEventBatch", func() {
		It("should create event batch", func() {
			event1 := NewInputEvent("Event1", []Tag{{Key: "key1", Value: "value1"}}, []byte("data1"))
			event2 := NewInputEvent("Event2", []Tag{{Key: "key2", Value: "value2"}}, []byte("data2"))

			batch := NewEventBatch(event1, event2)

			Expect(batch).To(HaveLen(2))
			Expect(batch[0]).To(Equal(event1))
			Expect(batch[1]).To(Equal(event2))
		})
	})
})
