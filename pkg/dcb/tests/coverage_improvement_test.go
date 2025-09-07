package dcb

import (
	"fmt"

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

	Describe("New Simplified API Coverage", func() {
		It("should test EventBuilder", func() {
			event := dcb.NewEvent("TestEvent").
				WithTag("key1", "value1").
				WithTag("key2", "value2").
				WithData(map[string]string{"data": "test"}).
				Build()

			Expect(event.GetType()).To(Equal("TestEvent"))
			Expect(event.GetTags()).To(HaveLen(2))
		})

		It("should test BatchBuilder", func() {
			batch := dcb.NewBatch().
				AddEventFromBuilder(
					dcb.NewEvent("Event1").
						WithTag("key1", "value1").
						WithData(map[string]string{"data": "test1"}),
				).
				AddEventFromBuilder(
					dcb.NewEvent("Event2").
						WithTag("key2", "value2").
						WithData(map[string]string{"data": "test2"}),
				).
				Build()

			Expect(batch).To(HaveLen(2))
			Expect(batch[0].GetType()).To(Equal("Event1"))
			Expect(batch[1].GetType()).To(Equal("Event2"))
		})

		It("should test QueryBuilder", func() {
			query := dcb.NewQueryBuilder().
				WithTag("user_id", "123").
				WithType("UserRegistered").
				AddItem().
				WithTag("user_id", "456").
				WithType("UserProfileUpdated").
				Build()

			Expect(query.GetItems()).To(HaveLen(2))
		})

		It("should test simplified append condition constructors", func() {
			condition1 := dcb.FailIfExists("user_id", "123")
			condition2 := dcb.FailIfEventType("UserRegistered", "user_id", "123")
			condition3 := dcb.FailIfEventTypes([]string{"UserRegistered", "UserProfileUpdated"}, "user_id", "123")

			Expect(condition1).ToNot(BeNil())
			Expect(condition2).ToNot(BeNil())
			Expect(condition3).ToNot(BeNil())
		})

		It("should test projection helpers", func() {
			counterProjector := dcb.ProjectCounter("user_count", "UserRegistered", "status", "active")
			booleanProjector := dcb.ProjectBoolean("user_exists", "UserRegistered", "user_id", "123")

			type UserState struct {
				UserID string
				Email  string
			}

			stateProjector := dcb.ProjectState("user_profile", "UserRegistered", "user_id", "123", UserState{}, func(state any, event dcb.Event) any {
				return state
			})

			Expect(counterProjector.ID).To(Equal("user_count"))
			Expect(booleanProjector.ID).To(Equal("user_exists"))
			Expect(stateProjector.ID).To(Equal("user_profile"))
		})

		It("should test simplified tags", func() {
			tags := dcb.Tags{
				"user_id": "123",
				"tenant":  "acme",
			}.ToTags()

			Expect(tags).To(HaveLen(2))

			// Check that both tags exist without relying on order
			keys := []string{tags[0].GetKey(), tags[1].GetKey()}
			values := []string{tags[0].GetValue(), tags[1].GetValue()}

			Expect(keys).To(ContainElement("user_id"))
			Expect(keys).To(ContainElement("tenant"))
			Expect(values).To(ContainElement("123"))
			Expect(values).To(ContainElement("acme"))
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

	Describe("Error Handling Coverage", func() {
		It("should test IsTableStructureError", func() {
			// Test with actual table structure error
			tableErr := &dcb.TableStructureError{
				EventStoreError: dcb.EventStoreError{
					Op:  "test",
					Err: fmt.Errorf("table structure issue"),
				},
				TableName:    "events",
				ColumnName:   "id",
				ExpectedType: "bigint",
				ActualType:   "integer",
				Issue:        "column type mismatch",
			}

			Expect(dcb.IsTableStructureError(tableErr)).To(BeTrue())

			// Test with non-table structure error
			otherErr := &dcb.ValidationError{
				EventStoreError: dcb.EventStoreError{
					Op:  "test",
					Err: fmt.Errorf("validation issue"),
				},
				Field: "type",
				Value: "invalid",
			}

			Expect(dcb.IsTableStructureError(otherErr)).To(BeFalse())

			// Test with nil error
			Expect(dcb.IsTableStructureError(nil)).To(BeFalse())

			// Test with regular error
			regularErr := fmt.Errorf("regular error")
			Expect(dcb.IsTableStructureError(regularErr)).To(BeFalse())
		})

		It("should test GetTableStructureError", func() {
			// Test error extraction
			tableErr := &dcb.TableStructureError{
				EventStoreError: dcb.EventStoreError{
					Op:  "test",
					Err: fmt.Errorf("table structure issue"),
				},
				TableName:    "events",
				ColumnName:   "id",
				ExpectedType: "bigint",
				ActualType:   "integer",
				Issue:        "column type mismatch",
			}

			extractedErr, found := dcb.GetTableStructureError(tableErr)
			Expect(found).To(BeTrue())
			Expect(extractedErr).To(Equal(tableErr))
			Expect(extractedErr.TableName).To(Equal("events"))
			Expect(extractedErr.ColumnName).To(Equal("id"))

			// Test with non-table structure error
			otherErr := &dcb.ValidationError{
				EventStoreError: dcb.EventStoreError{
					Op:  "test",
					Err: fmt.Errorf("validation issue"),
				},
				Field: "type",
				Value: "invalid",
			}

			extractedErr, found = dcb.GetTableStructureError(otherErr)
			Expect(found).To(BeFalse())
			Expect(extractedErr).To(BeNil())

			// Test with nil error
			extractedErr, found = dcb.GetTableStructureError(nil)
			Expect(found).To(BeFalse())
			Expect(extractedErr).To(BeNil())
		})

		It("should test AsConcurrencyError alias", func() {
			concurrencyErr := &dcb.ConcurrencyError{
				EventStoreError: dcb.EventStoreError{
					Op:  "append",
					Err: fmt.Errorf("concurrency conflict"),
				},
				ExpectedPosition: 100,
				ActualPosition:   200,
			}

			extractedErr, found := dcb.AsConcurrencyError(concurrencyErr)
			Expect(found).To(BeTrue())
			Expect(extractedErr).To(Equal(concurrencyErr))
			Expect(extractedErr.ExpectedPosition).To(Equal(int64(100)))
			Expect(extractedErr.ActualPosition).To(Equal(int64(200)))

			// Test with non-concurrency error
			otherErr := &dcb.ValidationError{
				EventStoreError: dcb.EventStoreError{
					Op:  "test",
					Err: fmt.Errorf("validation issue"),
				},
				Field: "type",
				Value: "invalid",
			}

			extractedErr, found = dcb.AsConcurrencyError(otherErr)
			Expect(found).To(BeFalse())
			Expect(extractedErr).To(BeNil())
		})

		It("should test AsResourceError alias", func() {
			resourceErr := &dcb.ResourceError{
				EventStoreError: dcb.EventStoreError{
					Op:  "connect",
					Err: fmt.Errorf("connection failed"),
				},
				Resource: "database",
			}

			extractedErr, found := dcb.AsResourceError(resourceErr)
			Expect(found).To(BeTrue())
			Expect(extractedErr).To(Equal(resourceErr))
			Expect(extractedErr.Resource).To(Equal("database"))

			// Test with non-resource error
			otherErr := &dcb.ValidationError{
				EventStoreError: dcb.EventStoreError{
					Op:  "test",
					Err: fmt.Errorf("validation issue"),
				},
				Field: "type",
				Value: "invalid",
			}

			extractedErr, found = dcb.AsResourceError(otherErr)
			Expect(found).To(BeFalse())
			Expect(extractedErr).To(BeNil())
		})

		It("should test AsTableStructureError alias", func() {
			tableErr := &dcb.TableStructureError{
				EventStoreError: dcb.EventStoreError{
					Op:  "test",
					Err: fmt.Errorf("table structure issue"),
				},
				TableName:    "events",
				ColumnName:   "id",
				ExpectedType: "bigint",
				ActualType:   "integer",
				Issue:        "column type mismatch",
			}

			extractedErr, found := dcb.AsTableStructureError(tableErr)
			Expect(found).To(BeTrue())
			Expect(extractedErr).To(Equal(tableErr))
			Expect(extractedErr.TableName).To(Equal("events"))

			// Test with non-table structure error
			otherErr := &dcb.ValidationError{
				EventStoreError: dcb.EventStoreError{
					Op:  "test",
					Err: fmt.Errorf("validation issue"),
				},
				Field: "type",
				Value: "invalid",
			}

			extractedErr, found = dcb.AsTableStructureError(otherErr)
			Expect(found).To(BeFalse())
			Expect(extractedErr).To(BeNil())
		})
	})
})
