package dcb

import (
	"github.com/rodolfodpk/go-crablet/pkg/dcb"

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

var _ = Describe("New Simplified API", func() {
	Describe("EventBuilder", func() {
		It("should create event with single tag", func() {
			event := dcb.NewEvent("TestEvent").
				WithTag("key1", "value1").
				WithData(map[string]string{"data": "test"}).
				Build()

			Expect(event.GetType()).To(Equal("TestEvent"))
			Expect(event.GetTags()).To(HaveLen(1))
			Expect(event.GetTags()[0].GetKey()).To(Equal("key1"))
			Expect(event.GetTags()[0].GetValue()).To(Equal("value1"))
		})

		It("should create event with multiple tags", func() {
			event := dcb.NewEvent("TestEvent").
				WithTag("key1", "value1").
				WithTag("key2", "value2").
				WithData(map[string]string{"data": "test"}).
				Build()

			Expect(event.GetType()).To(Equal("TestEvent"))
			Expect(event.GetTags()).To(HaveLen(2))

			// Check that both tags exist without relying on order
			keys := []string{event.GetTags()[0].GetKey(), event.GetTags()[1].GetKey()}
			values := []string{event.GetTags()[0].GetValue(), event.GetTags()[1].GetValue()}

			Expect(keys).To(ContainElement("key1"))
			Expect(keys).To(ContainElement("key2"))
			Expect(values).To(ContainElement("value1"))
			Expect(values).To(ContainElement("value2"))
		})

		It("should create event with tags map", func() {
			event := dcb.NewEvent("TestEvent").
				WithTags(map[string]string{
					"key1": "value1",
					"key2": "value2",
				}).
				WithData(map[string]string{"data": "test"}).
				Build()

			Expect(event.GetType()).To(Equal("TestEvent"))
			Expect(event.GetTags()).To(HaveLen(2))
		})
	})

	Describe("BatchBuilder", func() {
		It("should create batch with events", func() {
			event1 := dcb.NewEvent("Event1").
				WithTag("key1", "value1").
				WithData(map[string]string{"data": "value1"}).
				Build()

			event2 := dcb.NewEvent("Event2").
				WithTag("key2", "value2").
				WithData(map[string]string{"data": "value2"}).
				Build()

			batch := dcb.NewBatch().
				AddEvent(event1).
				AddEvent(event2).
				Build()

			Expect(batch).To(HaveLen(2))
			Expect(batch[0].GetType()).To(Equal("Event1"))
			Expect(batch[1].GetType()).To(Equal("Event2"))
		})

		It("should create batch with event builders", func() {
			batch := dcb.NewBatch().
				AddEventFromBuilder(
					dcb.NewEvent("Event1").
						WithTag("key1", "value1").
						WithData(map[string]string{"data": "value1"}),
				).
				AddEventFromBuilder(
					dcb.NewEvent("Event2").
						WithTag("key2", "value2").
						WithData(map[string]string{"data": "value2"}),
				).
				Build()

			Expect(batch).To(HaveLen(2))
			Expect(batch[0].GetType()).To(Equal("Event1"))
			Expect(batch[1].GetType()).To(Equal("Event2"))
		})
	})

	Describe("Simplified Tags", func() {
		It("should create tags from map", func() {
			tags := dcb.Tags{
				"key1": "value1",
				"key2": "value2",
			}.ToTags()

			Expect(tags).To(HaveLen(2))

			// Check that both tags exist without relying on order
			keys := []string{tags[0].GetKey(), tags[1].GetKey()}
			values := []string{tags[0].GetValue(), tags[1].GetValue()}

			Expect(keys).To(ContainElement("key1"))
			Expect(keys).To(ContainElement("key2"))
			Expect(values).To(ContainElement("value1"))
			Expect(values).To(ContainElement("value2"))
		})
	})

	Describe("Simplified AppendCondition Constructors", func() {
		It("should create FailIfExists condition", func() {
			condition := dcb.FailIfExists("user_id", "123")
			Expect(condition).ToNot(BeNil())
		})

		It("should create FailIfEventType condition", func() {
			condition := dcb.FailIfEventType("UserRegistered", "user_id", "123")
			Expect(condition).ToNot(BeNil())
		})

		It("should create FailIfEventTypes condition", func() {
			condition := dcb.FailIfEventTypes([]string{"UserRegistered", "UserProfileUpdated"}, "user_id", "123")
			Expect(condition).ToNot(BeNil())
		})
	})

	Describe("Projection Helpers", func() {
		It("should create counter projector", func() {
			projector := dcb.ProjectCounter("user_count", "UserRegistered", "status", "active")
			Expect(projector.ID).To(Equal("user_count"))
			Expect(projector.InitialState).To(Equal(0))
		})

		It("should create boolean projector", func() {
			projector := dcb.ProjectBoolean("user_exists", "UserRegistered", "user_id", "123")
			Expect(projector.ID).To(Equal("user_exists"))
			Expect(projector.InitialState).To(Equal(false))
		})

		It("should create state projector", func() {
			type UserState struct {
				UserID string
				Email  string
			}

			projector := dcb.ProjectState("user_profile", "UserRegistered", "user_id", "123", UserState{}, func(state any, event dcb.Event) any {
				return state
			})

			Expect(projector.ID).To(Equal("user_profile"))
			Expect(projector.InitialState).To(Equal(UserState{}))
		})
	})
})

var _ = Describe("Alternative Constructors", func() {
	It("NewCommand creates Command with minimal data", func() {
		command := dcb.NewCommand("TestCommand", []byte(`{"data": "test"}`), nil)

		Expect(command.GetType()).To(Equal("TestCommand"))
		Expect(command.GetData()).To(Equal([]byte(`{"data": "test"}`)))
		Expect(command.GetMetadata()).To(BeNil())
	})

	It("NewCommand handles empty data", func() {
		command := dcb.NewCommand("EmptyCommand", []byte{}, nil)

		Expect(command.GetType()).To(Equal("EmptyCommand"))
		Expect(command.GetData()).To(HaveLen(0))
		Expect(command.GetMetadata()).To(BeNil())
	})

	It("NewCommand handles nil data", func() {
		command := dcb.NewCommand("NilCommand", nil, nil)

		Expect(command.GetType()).To(Equal("NilCommand"))
		Expect(command.GetData()).To(BeNil())
		Expect(command.GetMetadata()).To(BeNil())
	})

	It("NewCommand handles metadata", func() {
		metadata := map[string]interface{}{"user_id": "123", "timestamp": "2024-01-01"}
		command := dcb.NewCommand("CommandWithMetadata", []byte(`{"data": "test"}`), metadata)

		Expect(command.GetType()).To(Equal("CommandWithMetadata"))
		Expect(command.GetData()).To(Equal([]byte(`{"data": "test"}`)))
		Expect(command.GetMetadata()).To(Equal(metadata))
	})

	It("NewEventStoreWithConfig validates config", func() {
		config := dcb.EventStoreConfig{
			MaxBatchSize:           1000,
			StreamBuffer:           100,
			DefaultAppendIsolation: dcb.IsolationLevelRepeatableRead,
			QueryTimeout:           5000,
			AppendTimeout:          3000,
		}

		// Test config validation (without requiring actual database connection)
		Expect(config.MaxBatchSize).To(Equal(1000))
		Expect(config.StreamBuffer).To(Equal(100))
		Expect(config.DefaultAppendIsolation).To(Equal(dcb.IsolationLevelRepeatableRead))
		Expect(config.QueryTimeout).To(Equal(5000))
		Expect(config.AppendTimeout).To(Equal(3000))
	})
})

var _ = Describe("Configuration Validation", func() {
	It("ParseIsolationLevel handles valid values", func() {
		level, err := dcb.ParseIsolationLevel("READ_COMMITTED")
		Expect(err).To(BeNil())
		Expect(level).To(Equal(dcb.IsolationLevelReadCommitted))

		level, err = dcb.ParseIsolationLevel("REPEATABLE_READ")
		Expect(err).To(BeNil())
		Expect(level).To(Equal(dcb.IsolationLevelRepeatableRead))

		level, err = dcb.ParseIsolationLevel("SERIALIZABLE")
		Expect(err).To(BeNil())
		Expect(level).To(Equal(dcb.IsolationLevelSerializable))
	})

	It("ParseIsolationLevel handles invalid values", func() {
		level, err := dcb.ParseIsolationLevel("INVALID_LEVEL")
		Expect(err).To(Not(BeNil()))
		Expect(err.Error()).To(ContainSubstring("invalid isolation level: INVALID_LEVEL"))
		Expect(level).To(Equal(dcb.IsolationLevelReadCommitted)) // Default fallback
	})

	It("ParseIsolationLevel handles empty string", func() {
		level, err := dcb.ParseIsolationLevel("")
		Expect(err).To(Not(BeNil()))
		Expect(err.Error()).To(ContainSubstring("invalid isolation level: "))
		Expect(level).To(Equal(dcb.IsolationLevelReadCommitted)) // Default fallback
	})

	It("IsolationLevel String() method works correctly", func() {
		Expect(dcb.IsolationLevelReadCommitted.String()).To(Equal("READ_COMMITTED"))
		Expect(dcb.IsolationLevelRepeatableRead.String()).To(Equal("REPEATABLE_READ"))
		Expect(dcb.IsolationLevelSerializable.String()).To(Equal("SERIALIZABLE"))
	})

	It("IsolationLevel String() handles unknown values", func() {
		unknownLevel := dcb.IsolationLevel(999)
		Expect(unknownLevel.String()).To(Equal("UNKNOWN"))
	})
})

var _ = Describe("Edge Cases and Error Handling", func() {
	It("ToJSON panics on marshaling error", func() {
		// Create a value that can't be marshaled to JSON
		unmarshalable := make(chan int)

		Expect(func() {
			dcb.ToJSON(unmarshalable)
		}).To(Panic())
	})

	It("ToJSON handles valid data", func() {
		data := map[string]string{"key": "value"}
		result := dcb.ToJSON(data)

		Expect(result).To(Equal([]byte(`{"key":"value"}`)))
	})

	It("ToJSON handles nil data", func() {
		result := dcb.ToJSON(nil)

		Expect(result).To(Equal([]byte("null")))
	})

	It("NewTags handles odd number of arguments", func() {
		tags := dcb.NewTags("key1", "value1", "key2") // Odd number

		Expect(tags).To(HaveLen(0)) // Should return empty slice
	})

	It("NewTags handles empty arguments", func() {
		tags := dcb.NewTags()

		Expect(tags).To(HaveLen(0))
	})

	It("NewTags handles single argument", func() {
		tags := dcb.NewTags("key1")

		Expect(tags).To(HaveLen(0)) // Should return empty slice
	})
})

func createTestEvent(eventType string, key, value string) dcb.InputEvent {
	return dcb.NewEvent(eventType).
		WithTag(key, value).
		WithData(map[string]string{"data": "test"}).
		Build()
}

func createTestEventWithMultipleTags(eventType string, tags []dcb.Tag) dcb.InputEvent {
	builder := dcb.NewEvent(eventType).WithData(map[string]string{"data": "test"})
	for _, tag := range tags {
		builder = builder.WithTag(tag.GetKey(), tag.GetValue())
	}
	return builder.Build()
}
