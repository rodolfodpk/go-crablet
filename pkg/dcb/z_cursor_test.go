package dcb

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Cursor-based operations", func() {
	Context("Read with cursor", func() {
		It("should read only new events after the cursor", func() {
			ctx := context.Background()

			// Create some test events
			events := []InputEvent{
				NewInputEvent("TestEvent", NewTags("test", "1"), []byte(`{"value": 1}`)),
				NewInputEvent("TestEvent", NewTags("test", "2"), []byte(`{"value": 2}`)),
				NewInputEvent("TestEvent", NewTags("test", "3"), []byte(`{"value": 3}`)),
			}

			// Append events
			err := store.Append(ctx, events, nil)
			Expect(err).NotTo(HaveOccurred())

			// Read all events first to get cursor
			query := NewQuery(NewTags("test", "1"), "TestEvent")
			firstEvents, err := store.Read(ctx, query, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(firstEvents).To(HaveLen(1))

			// Create cursor from first event
			cursor := &Cursor{
				TransactionID: firstEvents[0].TransactionID,
				Position:      firstEvents[0].Position,
			}

			// Read from cursor - should get events after the cursor
			query2 := NewQuery(nil, "TestEvent") // Query all TestEvents
			eventsFromCursor, err := store.Read(ctx, query2, cursor)
			Expect(err).NotTo(HaveOccurred())

			// Should get the remaining events (2 and 3)
			Expect(eventsFromCursor).To(HaveLen(2))
			Expect(eventsFromCursor[0].Position).To(BeNumerically(">", cursor.Position))
			Expect(eventsFromCursor[1].Position).To(BeNumerically(">", cursor.Position))
		})

		It("should handle nil cursor gracefully", func() {
			ctx := context.Background()

			// Create some test events
			events := []InputEvent{
				NewInputEvent("TestEvent", NewTags("test", "nil"), []byte(`{"value": 1}`)),
			}

			// Append events
			err := store.Append(ctx, events, nil)
			Expect(err).NotTo(HaveOccurred())

			// Read with nil cursor should work like regular Read
			query := NewQuery(NewTags("test", "nil"), "TestEvent")
			eventsFromCursor, err := store.Read(ctx, query, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(eventsFromCursor).To(HaveLen(1))
		})
	})

	Context("Project with cursor", func() {
		It("should project only new events after the cursor", func() {
			ctx := context.Background()

			// Create some test events
			events := []InputEvent{
				NewInputEvent("UserCreated", NewTags("user_id", "123"), []byte(`{"name": "John"}`)),
				NewInputEvent("UserUpdated", NewTags("user_id", "123"), []byte(`{"name": "John Doe"}`)),
				NewInputEvent("UserUpdated", NewTags("user_id", "123"), []byte(`{"name": "John Doe Smith"}`)),
			}

			// Append events
			err := store.Append(ctx, events, nil)
			Expect(err).NotTo(HaveOccurred())

			// Read first event to get cursor
			query := NewQuery(NewTags("user_id", "123"), "UserCreated")
			firstEvents, err := store.Read(ctx, query, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(firstEvents).To(HaveLen(1))

			// Create cursor from first event
			cursor := &Cursor{
				TransactionID: firstEvents[0].TransactionID,
				Position:      firstEvents[0].Position,
			}

			// Create projector
			projector := StateProjector{
				ID:    "user",
				Query: NewQuery(NewTags("user_id", "123")),
				InitialState: map[string]interface{}{
					"name": "",
				},
				TransitionFn: func(state any, event Event) any {
					userState := state.(map[string]interface{})
					if event.Type == "UserCreated" || event.Type == "UserUpdated" {
						// Parse event data to get name
						userState["name"] = "updated" // Simplified for test
					}
					return userState
				},
			}

			// Project from cursor
			states, appendCondition, err := store.Project(ctx, []StateProjector{projector}, cursor)
			Expect(err).NotTo(HaveOccurred())
			Expect(states).To(HaveKey("user"))
			Expect(appendCondition).NotTo(BeNil())

			// Should have processed events after the cursor
			userState := states["user"].(map[string]interface{})
			Expect(userState["name"]).To(Equal("updated"))
		})
	})

	Context("ProjectStreamFromCursor", func() {
		It("should stream projection from a specific cursor", func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// Create some test events
			events := []InputEvent{
				NewInputEvent("UserCreated", NewTags("user_id", "456"), []byte(`{"name": "Jane"}`)),
				NewInputEvent("UserUpdated", NewTags("user_id", "456"), []byte(`{"name": "Jane Doe"}`)),
			}

			// Append events
			err := store.Append(ctx, events, nil)
			Expect(err).NotTo(HaveOccurred())

			// Read first event to get cursor
			query := NewQuery(NewTags("user_id", "456"), "UserCreated")
			firstEvents, err := store.Read(ctx, query, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(firstEvents).To(HaveLen(1))

			// Create cursor from first event
			cursor := &Cursor{
				TransactionID: firstEvents[0].TransactionID,
				Position:      firstEvents[0].Position,
			}

			// Create projector
			projector := StateProjector{
				ID:    "user",
				Query: NewQuery(NewTags("user_id", "456")),
				InitialState: map[string]interface{}{
					"name": "",
				},
				TransitionFn: func(state any, event Event) any {
					userState := state.(map[string]interface{})
					if event.Type == "UserCreated" || event.Type == "UserUpdated" {
						userState["name"] = "streamed"
					}
					return userState
				},
			}

			// Stream projection from cursor
			statesChan, appendConditionChan, err := store.ProjectStream(ctx, []StateProjector{projector}, cursor)
			Expect(err).NotTo(HaveOccurred())

			// Collect results
			var finalStates map[string]any
			var finalAppendCondition AppendCondition

			select {
			case states := <-statesChan:
				finalStates = states
			case <-ctx.Done():
				Fail("timeout waiting for states")
			}

			select {
			case appendCondition := <-appendConditionChan:
				finalAppendCondition = appendCondition
			case <-ctx.Done():
				Fail("timeout waiting for append condition")
			}

			// Verify results
			Expect(finalStates).To(HaveKey("user"))
			Expect(finalAppendCondition).NotTo(BeNil())

			userState := finalStates["user"].(map[string]interface{})
			Expect(userState["name"]).To(Equal("streamed"))
		})
	})
})
