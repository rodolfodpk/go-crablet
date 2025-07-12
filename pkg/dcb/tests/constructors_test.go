package dcb

import (
	"go-crablet/pkg/dcb"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Helper Functions", func() {
	Describe("NewQueryFromItems", func() {
		It("should create query from multiple items", func() {
			item1 := dcb.NewQueryItem([]string{"Event1"}, []dcb.Tag{dcb.NewTag("key1", "value1")})
			item2 := dcb.NewQueryItem([]string{"Event2"}, []dcb.Tag{dcb.NewTag("key2", "value2")})
			query := dcb.NewQueryFromItems(item1, item2)

			Expect(query.GetItems()).To(HaveLen(2))
			Expect(query.GetItems()[0]).To(Equal(item1))
			Expect(query.GetItems()[1]).To(Equal(item2))
		})

		It("should create empty query", func() {
			query := dcb.NewQueryEmpty()
			Expect(query.GetItems()).To(BeEmpty())
		})

		It("should create query that matches all events", func() {
			query := dcb.NewQueryAll()
			Expect(query.GetItems()).To(HaveLen(1))
			Expect(query.GetItems()[0].GetEventTypes()).To(BeEmpty())
			Expect(query.GetItems()[0].GetTags()).To(BeEmpty())
		})
	})

	Describe("NewQueryItem", func() {
		It("should create query item with types and tags", func() {
			types := []string{"Event1", "Event2"}
			tags := []dcb.Tag{dcb.NewTag("key1", "value1")}
			item := dcb.NewQueryItem(types, tags)

			Expect(item.GetEventTypes()).To(Equal(types))
			Expect(item.GetTags()).To(Equal(tags))
		})

		It("should create query item with empty types and tags", func() {
			item := dcb.NewQueryItem([]string{}, []dcb.Tag{})
			Expect(item.GetEventTypes()).To(BeEmpty())
			Expect(item.GetTags()).To(BeEmpty())
		})
	})

	Describe("NewEventBatch", func() {
		It("should create event batch", func() {
			event1 := dcb.NewInputEvent("Event1", []dcb.Tag{dcb.NewTag("key1", "value1")}, []byte(`{"data": "value1"}`))
			event2 := dcb.NewInputEvent("Event2", []dcb.Tag{dcb.NewTag("key2", "value2")}, []byte(`{"data": "value2"}`))

			batch := dcb.NewEventBatch(event1, event2)

			Expect(batch).To(HaveLen(2))
			Expect(batch[0]).To(Equal(event1))
			Expect(batch[1]).To(Equal(event2))
		})
	})

	It("should cover TagsToString helper", func() {
		tags := dcb.NewTags("foo", "bar")
		strs := dcb.TagsToString(tags)
		Expect(strs).To(ContainElement("foo:bar"))
	})
})

func createTestEvent(eventType string, key, value string) dcb.InputEvent {
	return dcb.NewInputEvent(eventType, []dcb.Tag{dcb.NewTag(key, value)}, dcb.ToJSON(map[string]string{"data": "test"}))
}

func createTestEventWithMultipleTags(eventType string, tags []dcb.Tag) dcb.InputEvent {
	return dcb.NewInputEvent(eventType, tags, dcb.ToJSON(map[string]string{"data": "test"}))
}
