package dcb

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Event Store: Input Validation", func() {
	BeforeEach(func() {
		_, err := pool.Exec(ctx, "TRUNCATE TABLE events RESTART IDENTITY CASCADE")
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("Invalid Query Tags", func() {
		It("rejects query with empty tag key", func() {
			invalidTags := []Tag{{Key: "", Value: "some-value"}}
			query := Query{Tags: invalidTags}
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
			invalidTags := []Tag{{Key: "some-key", Value: ""}}
			query := Query{Tags: invalidTags}
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

		It("rejects query with empty event type", func() {
			query := Query{
				Tags:       NewTags("valid-key", "valid-value"),
				EventTypes: []string{"ValidType", ""},
			}
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
			query := NewQuery(tags)
			events := []InputEvent{{
				Type: "",
				Tags: tags,
				Data: []byte(`{"valid":"json"}`),
			}}

			_, err := store.AppendEvents(ctx, events, query, 0)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("empty type in event"))
		})

		It("rejects event with empty tags", func() {
			query := NewQuery(NewTags("valid-key", "valid-value"))
			events := []InputEvent{{
				Type: "EventType",
				Tags: []Tag{},
				Data: []byte(`{"valid":"json"}`),
			}}

			_, err := store.AppendEvents(ctx, events, query, 0)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("empty tags in event"))
		})

		It("rejects event with tag having empty key", func() {
			query := NewQuery(NewTags("valid-key", "valid-value"))
			events := []InputEvent{{
				Type: "EventType",
				Tags: []Tag{{Key: "", Value: "value"}},
				Data: []byte(`{"valid":"json"}`),
			}}

			_, err := store.AppendEvents(ctx, events, query, 0)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("empty key in tag"))
		})

		It("rejects event with tag having empty value", func() {
			query := NewQuery(NewTags("valid-key", "valid-value"))
			events := []InputEvent{{
				Type: "EventType",
				Tags: []Tag{{Key: "key", Value: ""}},
				Data: []byte(`{"valid":"json"}`),
			}}

			_, err := store.AppendEvents(ctx, events, query, 0)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("empty value for key"))
		})

		It("rejects event with invalid JSON data", func() {
			tags := NewTags("entity_id", "123")
			query := NewQuery(tags)
			events := []InputEvent{{
				Type: "EventType",
				Tags: tags,
				Data: []byte(`{"unclosed": "json"`),
			}}

			_, err := store.AppendEvents(ctx, events, query, 0)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid JSON data"))
		})
	})
})
