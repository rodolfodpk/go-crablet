package dcb

import (
	"github.com/rodolfodpk/go-crablet/pkg/dcb"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Coverage Improvement Tests", func() {
	Describe("NewQuery", func() {
		It("should create query with validation", func() {
			tags := dcb.NewTags("user_id", "123")
			eventTypes := []string{"UserRegistered", "UserNameChanged"}

			query := dcb.NewQuery(tags, eventTypes...)

			Expect(query).NotTo(BeNil())
			Expect(query.GetItems()).To(HaveLen(1))
			Expect(query.GetItems()[0].GetEventTypes()).To(Equal(eventTypes))
			Expect(query.GetItems()[0].GetTags()).To(Equal(tags))
		})

		It("should create query with event types and tags", func() {
			eventTypes := []string{"Event1", "Event2"}
			tags := []dcb.Tag{dcb.NewTag("key1", "value1")}

			query := dcb.NewQuery(tags, eventTypes...)

			Expect(query).NotTo(BeNil())
			Expect(query.GetItems()).To(HaveLen(1))
			Expect(query.GetItems()[0].GetEventTypes()).To(Equal(eventTypes))
			Expect(query.GetItems()[0].GetTags()).To(Equal(tags))
		})
	})

	Describe("NewQueryItem", func() {
		It("should create query item with single event type and tags", func() {
			eventType := "UserRegistered"
			tags := dcb.NewTags("user_id", "123")

			item := dcb.NewQueryItem([]string{eventType}, tags)

			Expect(item).NotTo(BeNil())
			Expect(item.GetEventTypes()).To(Equal([]string{eventType}))
			Expect(item.GetTags()).To(Equal(tags))
		})

		It("should create query item with single event type and key-value tags", func() {
			eventType := "UserRegistered"
			kv := []string{"user_id", "123", "tenant", "test"}

			item := dcb.NewQueryItem([]string{eventType}, dcb.NewTags(kv...))

			Expect(item).NotTo(BeNil())
			Expect(item.GetEventTypes()).To(Equal([]string{eventType}))
			Expect(item.GetTags()).To(HaveLen(2))
			Expect(item.GetTags()[0].GetKey()).To(Equal("user_id"))
			Expect(item.GetTags()[0].GetValue()).To(Equal("123"))
			Expect(item.GetTags()[1].GetKey()).To(Equal("tenant"))
			Expect(item.GetTags()[1].GetValue()).To(Equal("test"))
		})
	})

	Describe("NewEventBatch", func() {
		It("should create batch from multiple events", func() {
			event1 := dcb.NewInputEvent("Event1", dcb.NewTags("key1", "value1"), []byte(`{"data": "test1"}`))
			event2 := dcb.NewInputEvent("Event2", dcb.NewTags("key2", "value2"), []byte(`{"data": "test2"}`))

			batch := dcb.NewEventBatch(event1, event2)

			Expect(batch).To(HaveLen(2))
			Expect(batch[0].GetType()).To(Equal("Event1"))
			Expect(batch[1].GetType()).To(Equal("Event2"))
		})

		It("should create empty batch", func() {
			batch := dcb.NewEventBatch()

			Expect(batch).To(BeEmpty())
		})

		It("should create batch from single event", func() {
			event := dcb.NewInputEvent("Event1", dcb.NewTags("key1", "value1"), []byte(`{"data": "test1"}`))

			batch := dcb.NewEventBatch(event)

			Expect(batch).To(HaveLen(1))
			Expect(batch[0].GetType()).To(Equal("Event1"))
		})
	})

	It("should cover NewQuery and NewQueryEmpty", func() {
		q := dcb.NewQuery(dcb.NewTags("foo", "bar"), "TypeA", "TypeB")
		Expect(q).NotTo(BeNil())
		q2 := dcb.NewQueryEmpty()
		Expect(q2).NotTo(BeNil())
	})

	It("should cover buildAppendConditionFromQuery", func() {
		q := dcb.NewQuery(dcb.NewTags("foo", "bar"), "TypeA")
		cond := dcb.BuildAppendConditionFromQuery(q)
		Expect(cond).NotTo(BeNil())
	})
})
