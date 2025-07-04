package dcb

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Coverage Improvement Tests", func() {
	Describe("NewQuerySimpleUnsafe", func() {
		It("should create query without validation", func() {
			tags := NewTags("user_id", "123")
			eventTypes := []string{"UserCreated", "UserUpdated"}

			query := NewQuerySimpleUnsafe(tags, eventTypes...)

			Expect(query.getItems()).To(HaveLen(1))
			Expect(query.getItems()[0].getEventTypes()).To(Equal(eventTypes))
			Expect(query.getItems()[0].getTags()).To(Equal(tags))
		})

		It("should create query with event types and tags using unsafe constructor", func() {
			eventTypes := []string{"Event1", "Event2"}
			tags := []Tag{NewTag("key1", "value1")}

			query := NewQuerySimpleUnsafe(tags, eventTypes...)

			Expect(query.getItems()).To(HaveLen(1))
			Expect(query.getItems()[0].getEventTypes()).To(Equal(eventTypes))
			Expect(query.getItems()[0].getTags()).To(Equal(tags))
		})
	})

	Describe("NewQItem", func() {
		It("should create query item with single event type and tags", func() {
			eventType := "UserCreated"
			tags := NewTags("user_id", "123")

			item := NewQItem(eventType, tags)

			Expect(item.getEventTypes()).To(Equal([]string{eventType}))
			Expect(item.getTags()).To(Equal(tags))
		})
	})

	Describe("NewQItemKV", func() {
		It("should create query item with single event type and key-value tags", func() {
			eventType := "UserCreated"
			kv := []string{"user_id", "123", "tenant", "test"}

			item := NewQItemKV(eventType, kv...)

			Expect(item.getEventTypes()).To(Equal([]string{eventType}))
			Expect(item.getTags()).To(HaveLen(2))
			Expect(item.getTags()[0].GetKey()).To(Equal("user_id"))
			Expect(item.getTags()[0].GetValue()).To(Equal("123"))
			Expect(item.getTags()[1].GetKey()).To(Equal("tenant"))
			Expect(item.getTags()[1].GetValue()).To(Equal("test"))
		})
	})

	Describe("NewEventBatch", func() {
		It("should create batch from multiple events", func() {
			event1 := NewInputEvent("Event1", NewTags("key1", "value1"), []byte(`{"data": "test1"}`))
			event2 := NewInputEvent("Event2", NewTags("key2", "value2"), []byte(`{"data": "test2"}`))

			batch := NewEventBatch(event1, event2)

			Expect(batch).To(HaveLen(2))
			Expect(batch[0].GetType()).To(Equal("Event1"))
			Expect(batch[1].GetType()).To(Equal("Event2"))
		})

		It("should create empty batch", func() {
			batch := NewEventBatch()

			Expect(batch).To(BeEmpty())
		})

		It("should create batch from single event", func() {
			event := NewInputEvent("Event1", NewTags("key1", "value1"), []byte(`{"data": "test1"}`))

			batch := NewEventBatch(event)

			Expect(batch).To(HaveLen(1))
			Expect(batch[0].GetType()).To(Equal("Event1"))
		})
	})

	Describe("QueryItem operations", func() {
		It("should create query item with event type and tags", func() {
			eventType := "UserCreated"
			tags := NewTags("user_id", "123", "tenant", "test")

			item := NewQItem(eventType, tags)

			Expect(item.getEventTypes()).To(Equal([]string{eventType}))
			Expect(item.getTags()).To(Equal(tags))
		})

		It("should create query item with key-value pairs", func() {
			eventType := "UserCreated"
			kv := []string{"user_id", "123", "tenant", "test"}

			item := NewQItemKV(eventType, kv...)

			Expect(item.getEventTypes()).To(Equal([]string{eventType}))
			Expect(item.getTags()).To(HaveLen(2))
			Expect(item.getTags()[0].GetKey()).To(Equal("user_id"))
			Expect(item.getTags()[0].GetValue()).To(Equal("123"))
			Expect(item.getTags()[1].GetKey()).To(Equal("tenant"))
			Expect(item.getTags()[1].GetValue()).To(Equal("test"))
		})
	})

	It("should cover NewQuerySimpleUnsafe and NewQueryEmpty", func() {
		q := NewQuerySimpleUnsafe(NewTags("foo", "bar"), "TypeA", "TypeB")
		Expect(q).NotTo(BeNil())
		q2 := NewQueryEmpty()
		Expect(q2).NotTo(BeNil())
	})

	It("should cover buildAppendConditionFromQuery", func() {
		q := NewQuery(NewTags("foo", "bar"), "TypeA")
		cond := BuildAppendConditionFromQuery(q)
		Expect(cond).NotTo(BeNil())
	})
})

func TestCoverageImprovement(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Coverage Improvement Tests")
}
