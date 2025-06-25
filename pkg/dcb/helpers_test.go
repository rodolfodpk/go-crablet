package dcb

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Helper Functions", func() {
	Describe("NewQueryFromItems", func() {
		It("should create query from multiple items", func() {
			item1 := NewQueryItem([]string{"Event1"}, []Tag{{Key: "key1", Value: "value1"}})
			item2 := NewQueryItem([]string{"Event2"}, []Tag{{Key: "key2", Value: "value2"}})
			query := NewQueryFromItems(item1, item2)

			Expect(query.getItems()).To(HaveLen(2))
			Expect(query.getItems()[0]).To(Equal(item1))
			Expect(query.getItems()[1]).To(Equal(item2))
		})

		It("should create empty query", func() {
			query := NewQueryEmpty()
			Expect(query.getItems()).To(BeEmpty())
		})

		It("should create query that matches all events", func() {
			query := NewQueryAll()
			Expect(query.getItems()).To(HaveLen(1))
			Expect(query.getItems()[0].getEventTypes()).To(BeEmpty())
			Expect(query.getItems()[0].getTags()).To(BeEmpty())
		})
	})

	Describe("NewQueryItem", func() {
		It("should create query item with types and tags", func() {
			types := []string{"Event1", "Event2"}
			tags := []Tag{{Key: "key1", Value: "value1"}}
			item := NewQueryItem(types, tags)

			Expect(item.getEventTypes()).To(Equal(types))
			Expect(item.getTags()).To(Equal(tags))
		})

		It("should create query item with empty types and tags", func() {
			item := NewQueryItem([]string{}, []Tag{})
			Expect(item.getEventTypes()).To(BeEmpty())
			Expect(item.getTags()).To(BeEmpty())
		})
	})

	Describe("NewEventBatch", func() {
		It("should create event batch", func() {
			event1 := NewInputEvent("Event1", []Tag{{Key: "key1", Value: "value1"}}, []byte(`{"data": "value1"}`))
			event2 := NewInputEvent("Event2", []Tag{{Key: "key2", Value: "value2"}}, []byte(`{"data": "value2"}`))

			batch := NewEventBatch(event1, event2)

			Expect(batch).To(HaveLen(2))
			Expect(batch[0]).To(Equal(event1))
			Expect(batch[1]).To(Equal(event2))
		})
	})
})
