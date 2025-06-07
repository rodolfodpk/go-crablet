package dcb

import (
	"fmt"

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
			invalidTags := NewTags("", "some-value")
			query := NewQuery(invalidTags, "EventType")
			events := []InputEvent{
				NewInputEvent("EventType", NewTags("valid-key", "valid-value"), []byte(`{}`)),
			}

			_, err := store.AppendEvents(ctx, events, query, 0)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("empty key in tag"))
			validationErr, ok := err.(*ValidationError)
			Expect(ok).To(BeTrue())
			Expect(validationErr.Field).To(Equal("tag.key"))
		})

		It("rejects query with empty tag value", func() {
			invalidTags := NewTags("some-key", "")
			query := NewQuery(invalidTags, "EventType")
			events := []InputEvent{
				NewInputEvent("EventType", NewTags("valid-key", "valid-value"), []byte(`{}`)),
			}

			_, err := store.AppendEvents(ctx, events, query, 0)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("empty value for key"))
			validationErr, ok := err.(*ValidationError)
			Expect(ok).To(BeTrue())
			Expect(validationErr.Field).To(ContainSubstring("value"))
		})

		It("rejects query with empty event type in the list", func() {
			query := NewQuery(NewTags("valid-key", "valid-value"), "ValidType", "")
			events := []InputEvent{
				NewInputEvent("ValidType", NewTags("valid-key", "valid-value"), []byte(`{}`)),
			}

			_, err := store.AppendEvents(ctx, events, query, 0)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("empty event type"))
		})
	})

	Describe("Invalid Events", func() {
		It("rejects event with empty type", func() {
			tags := NewTags("entity_id", "123")
			query := NewQuery(tags, "EventType")
			events := []InputEvent{
				NewInputEvent("", tags, []byte(`{"valid":"json"}`)),
			}

			_, err := store.AppendEvents(ctx, events, query, 0)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("empty type in event"))
		})

		It("rejects event with empty tags", func() {
			query := NewQuery(NewTags("valid-key", "valid-value"), "EventType")
			events := []InputEvent{
				NewInputEvent("EventType", NewTags(), []byte(`{"valid":"json"}`)),
			}

			_, err := store.AppendEvents(ctx, events, query, 0)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("empty tags in event"))
		})

		It("rejects event with tag having empty key", func() {
			query := NewQuery(NewTags("valid-key", "valid-value"), "EventType")
			events := []InputEvent{
				NewInputEvent("EventType", NewTags("", "value"), []byte(`{"valid":"json"}`)),
			}

			_, err := store.AppendEvents(ctx, events, query, 0)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("empty key in tag"))
		})

		It("rejects event with tag having empty value", func() {
			query := NewQuery(NewTags("valid-key", "valid-value"), "EventType")
			events := []InputEvent{
				NewInputEvent("EventType", NewTags("key", ""), []byte(`{"valid":"json"}`)),
			}

			_, err := store.AppendEvents(ctx, events, query, 0)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("empty value for key"))
		})

		It("rejects event with invalid JSON data", func() {
			tags := NewTags("entity_id", "123")
			query := NewQuery(tags, "EventType")
			events := []InputEvent{{
				Type: "EventType",
				Tags: tags,
				Data: []byte(`{"unclosed": "json"`),
			}}

			_, err := store.AppendEvents(ctx, events, query, 0)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid JSON data"))
			validationErr, ok := err.(*ValidationError)
			Expect(ok).To(BeTrue())
			Expect(validationErr.Field).To(Equal("data"))
		})

		It("rejects batch exceeding maxBatchSize", func() {
			tags := NewTags("batch_id", "large")
			query := NewQuery(tags, "BatchEvent")
			// Create a batch that exceeds the default maxBatchSize of 1000
			events := make([]InputEvent, 1001)
			for i := range events {
				events[i] = NewInputEvent("BatchEvent", tags, []byte(`{"index":`+fmt.Sprintf("%d", i)+`}`))
			}

			_, err := store.AppendEvents(ctx, events, query, 0)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("batch size 1001 exceeds maximum 1000"))
			validationErr, ok := err.(*ValidationError)
			Expect(ok).To(BeTrue())
			Expect(validationErr.Field).To(Equal("batchSize"))
			Expect(validationErr.Value).To(Equal("1001"))
		})
	})

	It("validates event tags", func() {
		tags := NewTags("course_id", "course1")
		query := NewQuery(tags, "Subscription")
		event := NewInputEvent("Subscription", tags, []byte(`{"foo":"bar"}`))
		events := []InputEvent{event}

		_, err := store.AppendEvents(ctx, events, query, 0)
		Expect(err).NotTo(HaveOccurred())

		// Try with empty tags (should be allowed)
		emptyTags := NewTags()
		invalidQuery := NewQuery(emptyTags, "Subscription")
		_, err = store.AppendEvents(ctx, events, invalidQuery, 0)
		Expect(err).NotTo(HaveOccurred())
	})

	It("validates event data", func() {
		tags := NewTags("course_id", "course2")
		query := NewQuery(tags, "Subscription")
		event := NewInputEvent("Subscription", tags, []byte(`invalid json`))
		events := []InputEvent{event}

		_, err := store.AppendEvents(ctx, events, query, 0)
		Expect(err).To(HaveOccurred())
		validationErr, ok := err.(*ValidationError)
		Expect(ok).To(BeTrue())
		Expect(validationErr.Field).To(Equal("data"))
		Expect(err.Error()).To(ContainSubstring("invalid JSON data"))
	})

	It("validates event type", func() {
		tags := NewTags("course_id", "course3")
		query := NewQuery(tags, "Subscription")
		event := NewInputEvent("", tags, []byte(`{"foo":"bar"}`)) // Empty event type
		events := []InputEvent{event}

		_, err := store.AppendEvents(ctx, events, query, 0)
		Expect(err).To(HaveOccurred())
		validationErr, ok := err.(*ValidationError)
		Expect(ok).To(BeTrue())
		Expect(validationErr.Field).To(Equal("type"))
		Expect(err.Error()).To(ContainSubstring("empty type in event"))
	})
})
