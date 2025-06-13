package dcb

import (
	"context"
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Core API Tests", func() {
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()
		err := truncateEventsTable(ctx, pool)
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("EventStore Core Operations", func() {
		It("should handle basic operations", func() {
			tags := NewTags("test_id", "test-001")
			events := []InputEvent{
				NewInputEvent("TestEvent", tags, []byte(`{"data":"value"}`)),
			}

			q := NewQuery(tags, "TestEvent")
			_, err := store.Append(ctx, events, &AppendCondition{
				FailIfEventsMatch: &q,
				After:             nil,
			})
			Expect(err).NotTo(HaveOccurred())

			// Test basic read
			sequencedEvents, err := store.Read(ctx, q, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(sequencedEvents.Events).To(HaveLen(1))
			Expect(sequencedEvents.Events[0].Type).To(Equal("TestEvent"))
		})

		It("should handle decision model projection", func() {
			// Setup test data
			tags := NewTags("test_id", "test-002")
			events := []InputEvent{
				NewInputEvent("Event1", tags, []byte(`{"count":1}`)),
				NewInputEvent("Event2", tags, []byte(`{"count":2}`)),
			}

			q := NewQuery(tags, "Event1", "Event2")
			_, err := store.Append(ctx, events, &AppendCondition{
				FailIfEventsMatch: &q,
				After:             nil,
			})
			Expect(err).NotTo(HaveOccurred())

			// Test decision model projection (DCB pattern)
			projector := StateProjector{
				Query:        q,
				InitialState: 0,
				TransitionFn: func(state any, e Event) any {
					return state.(int) + 1
				},
			}

			states, appendCondition, err := store.ProjectDecisionModel(ctx, q, nil, []BatchProjector{
				{ID: "test", StateProjector: projector},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(states["test"]).To(Equal(2))
			Expect(appendCondition.FailIfEventsMatch).NotTo(BeNil())
			Expect(appendCondition.After).NotTo(BeNil())
		})

		It("should handle optimistic locking", func() {
			tags := NewTags("test_id", "test-003")

			// First append
			events1 := []InputEvent{
				NewInputEvent("Event1", tags, []byte(`{"data":"first"}`)),
			}
			q := NewQuery(tags, "Event1")
			_, err := store.Append(ctx, events1, &AppendCondition{
				FailIfEventsMatch: &q,
				After:             nil,
			})
			Expect(err).NotTo(HaveOccurred())

			// Get current position using Read
			sequencedEvents, err := store.Read(ctx, q, nil)
			Expect(err).NotTo(HaveOccurred())
			position := sequencedEvents.Position

			// Second append with correct position
			events2 := []InputEvent{
				NewInputEvent("Event2", tags, []byte(`{"data":"second"}`)),
			}
			q = NewQuery(tags, "Event2")
			_, err = store.Append(ctx, events2, &AppendCondition{
				FailIfEventsMatch: &q,
				After:             &position,
			})
			Expect(err).NotTo(HaveOccurred())

			// Verify both events exist
			q = NewQuery(tags, "Event1", "Event2")
			sequencedEvents, err = store.Read(ctx, q, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(sequencedEvents.Events).To(HaveLen(2))
		})

		It("should handle tag-based queries", func() {
			// Create events with different tag combinations
			courseTags := NewTags("course_id", "course-001")
			userTags := NewTags("user_id", "user-001")
			mixedTags := NewTags("course_id", "course-001", "user_id", "user-001")

			events := []InputEvent{
				NewInputEvent("CourseCreated", courseTags, []byte(`{"title":"Math"}`)),
				NewInputEvent("UserRegistered", userTags, []byte(`{"name":"Alice"}`)),
				NewInputEvent("Enrollment", mixedTags, []byte(`{"status":"active"}`)),
			}

			q := NewQuery(NewTags(), "CourseCreated", "UserRegistered", "Enrollment")
			_, err := store.Append(ctx, events, &AppendCondition{
				FailIfEventsMatch: &q,
				After:             nil,
			})
			Expect(err).NotTo(HaveOccurred())

			// Query by course_id
			courseQuery := NewQuery(courseTags, "CourseCreated", "Enrollment")
			courseEvents, err := store.Read(ctx, courseQuery, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(courseEvents.Events).To(HaveLen(2))

			// Query by user_id
			userQuery := NewQuery(userTags, "UserRegistered", "Enrollment")
			userEvents, err := store.Read(ctx, userQuery, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(userEvents.Events).To(HaveLen(2))
		})

		It("should return the position of the last event added", func() {
			tags := NewTags("test_id", "test-007")
			events := []InputEvent{
				NewInputEvent("Event1", tags, []byte(`{"data":"first"}`)),
				NewInputEvent("Event2", tags, []byte(`{"data":"second"}`)),
				NewInputEvent("Event3", tags, []byte(`{"data":"third"}`)),
			}

			q := NewQuery(tags, "Event1", "Event2", "Event3")
			position, err := store.Append(ctx, events, &AppendCondition{
				FailIfEventsMatch: &q,
				After:             nil,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(position).To(Equal(int64(3))) // Should return position 3 (last event)
		})
	})

	Describe("Helper Functions", func() {
		It("should create valid queries", func() {
			tags := NewTags("test_id", "test-004")
			query := NewQuery(tags, "Event1", "Event2")

			Expect(query.Items).To(HaveLen(1))
			Expect(query.Items[0].Tags).To(Equal(tags))
			Expect(query.Items[0].EventTypes).To(ContainElements("Event1", "Event2"))
		})

		It("should create valid events", func() {
			tags := NewTags("test_id", "test-005")
			data := map[string]string{"key": "value"}
			jsonData, _ := json.Marshal(data)

			event := NewInputEvent("TestEvent", tags, jsonData)
			Expect(event.Type).To(Equal("TestEvent"))
			Expect(event.Tags).To(Equal(tags))
			Expect(event.Data).To(Equal(jsonData))
		})

		It("should generate consistent TypeIDs", func() {
			tags := []Tag{
				{Key: "course_id", Value: "course-001"},
				{Key: "user_id", Value: "user-001"},
			}

			typeID1 := generateTagBasedTypeID(tags)
			typeID2 := generateTagBasedTypeID(tags)

			// UUIDs should be different but prefixes should be same
			Expect(typeID1).NotTo(Equal(typeID2))

			// Extract prefixes (remove UUID part)
			prefix1 := typeID1[:len(typeID1)-27]
			prefix2 := typeID2[:len(typeID2)-27]
			Expect(prefix1).To(Equal(prefix2))
		})
	})

	Describe("Error Handling", func() {
		It("should handle validation errors", func() {
			// Test empty events slice
			q := NewQuery(NewTags())
			_, err := store.Append(ctx, []InputEvent{}, &AppendCondition{
				FailIfEventsMatch: &q,
				After:             nil,
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("empty"))
		})

		It("should handle concurrency errors", func() {
			tags := NewTags("test_id", "test-006")

			// First append
			events1 := []InputEvent{
				NewInputEvent("Event1", tags, []byte(`{"data":"first"}`)),
			}
			q := NewQuery(tags, "Event1")
			_, err := store.Append(ctx, events1, &AppendCondition{
				FailIfEventsMatch: &q,
				After:             nil,
			})
			Expect(err).NotTo(HaveOccurred())

			// Try to append with conflicting condition (should fail)
			events2 := []InputEvent{
				NewInputEvent("Event2", tags, []byte(`{"data":"second"}`)),
			}
			// Use the same query that matches existing events
			conflictingQuery := NewQuery(tags, "Event1")
			_, err = store.Append(ctx, events2, &AppendCondition{
				FailIfEventsMatch: &conflictingQuery,
				After:             nil,
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("append condition violated"))
		})
	})
})
