package dcb

import (
	"fmt"
	"go-crablet/pkg/dcb"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Event Store: Input Validation", func() {
	BeforeEach(func() {
		err := truncateEventsTable(ctx, pool)
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("Invalid Query Tags", func() {
		It("rejects query with empty tag key", func() {
			invalidTags := []dcb.Tag{{Key: "", Value: "some-value"}}
			query := dcb.Query{
				Tags:       invalidTags,
				EventTypes: []string{"EventType"}, // Event types are optional
			}
			events := []dcb.InputEvent{
				dcb.NewInputEvent("EventType", dcb.NewTags("valid-key", "valid-value"), []byte(`{}`)),
			}

			_, err := store.AppendEvents(ctx, events, query, 0)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("empty key in tag"))
			validationErr, ok := err.(*dcb.ValidationError)
			Expect(ok).To(BeTrue())
			Expect(validationErr.Field).To(Equal("tag.key"))
		})

		It("rejects query with empty tag value", func() {
			invalidTags := []dcb.Tag{{Key: "some-key", Value: ""}}
			query := dcb.Query{
				Tags:       invalidTags,
				EventTypes: []string{"EventType"}, // Event types are optional
			}
			events := []dcb.InputEvent{
				dcb.NewInputEvent("EventType", dcb.NewTags("valid-key", "valid-value"), []byte(`{}`)),
			}

			_, err := store.AppendEvents(ctx, events, query, 0)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("empty value for key"))
			validationErr, ok := err.(*dcb.ValidationError)
			Expect(ok).To(BeTrue())
			Expect(validationErr.Field).To(ContainSubstring("value"))
		})

		It("rejects query with empty event type in the list", func() {
			query := dcb.Query{
				Tags:       dcb.NewTags("valid-key", "valid-value"),
				EventTypes: []string{"ValidType", ""}, // Empty event type in the list
			}
			events := []dcb.InputEvent{
				dcb.NewInputEvent("ValidType", dcb.NewTags("valid-key", "valid-value"), []byte(`{}`)),
			}

			_, err := store.AppendEvents(ctx, events, query, 0)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("empty event type"))
		})
	})

	Describe("Invalid Events", func() {
		It("rejects event with empty type", func() {
			tags := dcb.NewTags("entity_id", "123")
			query := dcb.NewQuery(tags, "EventType")
			events := []dcb.InputEvent{{
				Type: "",
				Tags: tags,
				Data: []byte(`{"valid":"json"}`),
			}}

			_, err := store.AppendEvents(ctx, events, query, 0)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("empty type in event"))
		})

		It("rejects event with empty tags", func() {
			query := dcb.NewQuery(dcb.NewTags("valid-key", "valid-value"), "EventType")
			events := []dcb.InputEvent{{
				Type: "EventType",
				Tags: []dcb.Tag{},
				Data: []byte(`{"valid":"json"}`),
			}}

			_, err := store.AppendEvents(ctx, events, query, 0)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("empty tags in event"))
		})

		It("rejects event with tag having empty key", func() {
			query := dcb.NewQuery(dcb.NewTags("valid-key", "valid-value"), "EventType")
			events := []dcb.InputEvent{{
				Type: "EventType",
				Tags: []dcb.Tag{{Key: "", Value: "value"}},
				Data: []byte(`{"valid":"json"}`),
			}}

			_, err := store.AppendEvents(ctx, events, query, 0)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("empty key in tag"))
		})

		It("rejects event with tag having empty value", func() {
			query := dcb.NewQuery(dcb.NewTags("valid-key", "valid-value"), "EventType")
			events := []dcb.InputEvent{{
				Type: "EventType",
				Tags: []dcb.Tag{{Key: "key", Value: ""}},
				Data: []byte(`{"valid":"json"}`),
			}}

			_, err := store.AppendEvents(ctx, events, query, 0)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("empty value for key"))
		})

		It("rejects event with invalid JSON data", func() {
			tags := dcb.NewTags("entity_id", "123")
			query := dcb.NewQuery(tags, "EventType")
			events := []dcb.InputEvent{{
				Type: "EventType",
				Tags: tags,
				Data: []byte(`{"unclosed": "json"`),
			}}

			_, err := store.AppendEvents(ctx, events, query, 0)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid JSON data"))
		})

		It("rejects batch exceeding maxBatchSize", func() {
			tags := dcb.NewTags("batch_id", "large")
			query := dcb.NewQuery(tags, "BatchEvent")
			// Create a batch that exceeds the default maxBatchSize of 1000
			events := make([]dcb.InputEvent, 1001)
			for i := range events {
				events[i] = dcb.NewInputEvent("BatchEvent", tags, []byte(`{"index":`+fmt.Sprintf("%d", i)+`}`))
			}

			_, err := store.AppendEvents(ctx, events, query, 0)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("batch size 1001 exceeds maximum 1000"))
			validationErr, ok := err.(*dcb.ValidationError)
			Expect(ok).To(BeTrue())
			Expect(validationErr.Field).To(Equal("batchSize"))
			Expect(validationErr.Value).To(Equal("1001"))
		})
	})

	It("validates event tags", func() {
		tags := dcb.NewTags("course_id", "course1")
		query := dcb.NewQuery(tags, "Subscription")
		event := dcb.NewInputEvent("Subscription", tags, []byte(`{"foo":"bar"}`))
		events := []dcb.InputEvent{event}

		_, err := store.AppendEvents(ctx, events, query, 0)
		Expect(err).NotTo(HaveOccurred())

		// Try with empty tags (should be allowed)
		emptyTags := dcb.NewTags()
		invalidQuery := dcb.NewQuery(emptyTags, "Subscription")
		_, err = store.AppendEvents(ctx, events, invalidQuery, 0)
		Expect(err).NotTo(HaveOccurred())
	})

	It("validates event data", func() {
		tags := dcb.NewTags("course_id", "course2")
		query := dcb.NewQuery(tags, "Subscription")
		event := dcb.NewInputEvent("Subscription", tags, []byte(`invalid json`))
		events := []dcb.InputEvent{event}

		_, err := store.AppendEvents(ctx, events, query, 0)
		Expect(err).To(HaveOccurred())
		validationErr, ok := err.(*dcb.ValidationError)
		Expect(ok).To(BeTrue())
		Expect(validationErr.Field).To(Equal("data"))
		Expect(err.Error()).To(ContainSubstring("invalid JSON data"))
	})

	It("validates event type", func() {
		tags := dcb.NewTags("course_id", "course3")
		query := dcb.NewQuery(tags, "Subscription")
		event := dcb.NewInputEvent("", tags, []byte(`{"foo":"bar"}`)) // Empty event type
		events := []dcb.InputEvent{event}

		_, err := store.AppendEvents(ctx, events, query, 0)
		Expect(err).To(HaveOccurred())
		validationErr, ok := err.(*dcb.ValidationError)
		Expect(ok).To(BeTrue())
		Expect(validationErr.Field).To(Equal("type"))
		Expect(err.Error()).To(ContainSubstring("empty type in event"))
	})
})
