package dcb

import (
	"github.com/rodolfodpk/go-crablet/pkg/dcb"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("QueryBuilder", func() {
	Describe("Build() method", func() {
		It("should build a simple query correctly", func() {
			// Create a QueryBuilder and build a query
			builder := dcb.NewQueryBuilder().
				WithTag("user_id", "123").
				WithType("UserRegistered")

			query := builder.Build()
			Expect(query).ToNot(BeNil())

			items := query.GetItems()
			Expect(items).To(HaveLen(1))

			// Verify the query has the expected content
			Expect(items[0].GetEventTypes()).To(ContainElement("UserRegistered"))
			Expect(items[0].GetTags()).To(HaveLen(1))
			Expect(items[0].GetTags()[0].GetKey()).To(Equal("user_id"))
			Expect(items[0].GetTags()[0].GetValue()).To(Equal("123"))
		})

		It("should build empty query correctly", func() {
			builder := dcb.NewQueryBuilder()

			query := builder.Build()
			Expect(query).ToNot(BeNil())
			Expect(query.GetItems()).To(HaveLen(0))
		})

		It("should build complex queries with multiple items", func() {
			builder := dcb.NewQueryBuilder()

			// Build a complex query with multiple items
			query := builder.
				WithTag("user_id", "123").
				WithType("UserRegistered").
				AddItem().
				WithTag("user_id", "456").
				WithType("UserProfileUpdated").
				Build()

			Expect(query).ToNot(BeNil())
			items := query.GetItems()
			Expect(items).To(HaveLen(2))

			// Verify first item
			Expect(items[0].GetEventTypes()).To(ContainElement("UserRegistered"))
			Expect(items[0].GetTags()).To(HaveLen(1))
			Expect(items[0].GetTags()[0].GetKey()).To(Equal("user_id"))
			Expect(items[0].GetTags()[0].GetValue()).To(Equal("123"))

			// Verify second item
			Expect(items[1].GetEventTypes()).To(ContainElement("UserProfileUpdated"))
			Expect(items[1].GetTags()).To(HaveLen(1))
			Expect(items[1].GetTags()[0].GetKey()).To(Equal("user_id"))
			Expect(items[1].GetTags()[0].GetValue()).To(Equal("456"))
		})

		It("should handle multiple queries with separate builders", func() {
			// First query
			builder1 := dcb.NewQueryBuilder().
				WithTag("user_id", "123").
				WithType("UserRegistered")

			query1 := builder1.Build()
			Expect(query1).ToNot(BeNil())
			items1 := query1.GetItems()
			Expect(items1).To(HaveLen(1))

			// Second query with new builder
			builder2 := dcb.NewQueryBuilder().
				WithTag("user_id", "456").
				WithType("UserProfileUpdated")

			query2 := builder2.Build()
			Expect(query2).ToNot(BeNil())
			items2 := query2.GetItems()
			Expect(items2).To(HaveLen(1))

			// Verify queries are independent
			Expect(items1[0].GetTags()[0].GetValue()).To(Equal("123"))
			Expect(items2[0].GetTags()[0].GetValue()).To(Equal("456"))
		})
	})

	Describe("Builder pattern usage", func() {
		It("should follow standard builder pattern - one-shot usage", func() {
			// Standard builder pattern: create, configure, build, discard
			query := dcb.NewQueryBuilder().
				WithTag("user_id", "123").
				WithType("UserRegistered").
				Build()

			Expect(query).ToNot(BeNil())
			Expect(query.GetItems()).To(HaveLen(1))
		})

		It("should not accumulate state between separate builder instances", func() {
			// Each builder should be independent
			builder1 := dcb.NewQueryBuilder().WithTag("key1", "value1")
			builder2 := dcb.NewQueryBuilder().WithTag("key2", "value2")

			query1 := builder1.Build()
			query2 := builder2.Build()

			Expect(query1.GetItems()).To(HaveLen(1))
			Expect(query2.GetItems()).To(HaveLen(1))
			Expect(query1.GetItems()[0].GetTags()[0].GetKey()).To(Equal("key1"))
			Expect(query2.GetItems()[0].GetTags()[0].GetKey()).To(Equal("key2"))
		})
	})
})
