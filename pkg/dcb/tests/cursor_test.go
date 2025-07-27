package dcb

import (
	"context"
	"fmt"
	"time"

	"github.com/rodolfodpk/go-crablet/pkg/dcb"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Cursor-based operations", func() {
	Context("Read with cursor", func() {
		It("should read only new events after the cursor", func() {
			ctx := context.Background()

			// Create some test events with unique tags
			events := []dcb.InputEvent{
				dcb.NewInputEvent("TestEvent", dcb.NewTags("cursor_test", "1"), []byte(`{"value": 1}`)),
				dcb.NewInputEvent("TestEvent", dcb.NewTags("cursor_test", "2"), []byte(`{"value": 2}`)),
				dcb.NewInputEvent("TestEvent", dcb.NewTags("cursor_test", "3"), []byte(`{"value": 3}`)),
			}

			// Append events
			err := store.Append(ctx, events)
			Expect(err).NotTo(HaveOccurred())

			// Read first event to get cursor
			query := dcb.NewQuery(dcb.NewTags("cursor_test", "1"), "TestEvent")
			firstEvents, err := store.Query(ctx, query, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(firstEvents).To(HaveLen(1))

			// Create cursor from first event
			cursor := &dcb.Cursor{
				TransactionID: firstEvents[0].TransactionID,
				Position:      firstEvents[0].Position,
			}

			// Read from cursor - should get events after the cursor with same tag pattern
			query2 := dcb.NewQuery(dcb.NewTags("cursor_test"), "TestEvent") // Query all cursor_test events
			eventsFromCursor, err := store.Query(ctx, query2, cursor)
			Expect(err).NotTo(HaveOccurred())

			// Should get the remaining events (2 and 3)
			Expect(eventsFromCursor).To(HaveLen(2))
			Expect(eventsFromCursor[0].Position).To(BeNumerically(">", cursor.Position))
			Expect(eventsFromCursor[1].Position).To(BeNumerically(">", cursor.Position))
		})

		It("should handle nil cursor gracefully", func() {
			ctx := context.Background()

			// Create some test events
			events := []dcb.InputEvent{
				dcb.NewInputEvent("TestEvent", dcb.NewTags("test", "nil"), []byte(`{"value": 1}`)),
			}

			// Append events
			err := store.Append(ctx, events)
			Expect(err).NotTo(HaveOccurred())

			// Read with nil cursor should work like regular Read
			query := dcb.NewQuery(dcb.NewTags("test", "nil"), "TestEvent")
			eventsFromCursor, err := store.Query(ctx, query, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(eventsFromCursor).To(HaveLen(1))
		})
	})

	Context("Project with cursor", func() {
		It("should project only new events after the cursor", func() {
			ctx := context.Background()

			// Use unique tags to avoid interference from other tests
			uniqueID := fmt.Sprintf("cursor_test_%d", time.Now().UnixNano())

			// Create some test events
			events := []dcb.InputEvent{
				dcb.NewInputEvent("UserRegistered", dcb.NewTags("user_id", uniqueID), []byte(`{"name": "John"}`)),
				dcb.NewInputEvent("UserNameChanged", dcb.NewTags("user_id", uniqueID), []byte(`{"name": "John Doe"}`)),
				dcb.NewInputEvent("UserNameChanged", dcb.NewTags("user_id", uniqueID), []byte(`{"name": "John Doe Smith"}`)),
			}

			// Append events
			err := store.Append(ctx, events)
			Expect(err).NotTo(HaveOccurred())

			// Read first event to get cursor
			query := dcb.NewQuery(dcb.NewTags("user_id", uniqueID), "UserRegistered")
			firstEvents, err := store.Query(ctx, query, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(firstEvents).To(HaveLen(1))

			// Create cursor from first event
			cursor := &dcb.Cursor{
				TransactionID: firstEvents[0].TransactionID,
				Position:      firstEvents[0].Position,
			}

			// Create projector
			projector := dcb.StateProjector{
				ID:    "user",
				Query: dcb.NewQuery(dcb.NewTags("user_id", uniqueID)),
				InitialState: map[string]interface{}{
					"name": "",
				},
				TransitionFn: func(state any, event dcb.Event) any {
					userState := state.(map[string]interface{})
					if event.Type == "UserRegistered" || event.Type == "UserNameChanged" {
						// Parse event data to get name
						userState["name"] = "updated" // Simplified for test
					}
					return userState
				},
			}

			// Project from cursor
			states, appendCondition, err := store.Project(ctx, []dcb.StateProjector{projector}, cursor)
			Expect(err).NotTo(HaveOccurred())
			Expect(states).To(HaveKey("user"))
			Expect(appendCondition).NotTo(BeNil())

			// Should have processed events after the cursor (2 UserNameChanged events)
			userState := states["user"].(map[string]interface{})
			Expect(userState["name"]).To(Equal("updated"))
		})
	})

	Context("ProjectStreamFromCursor", func() {
		It("should stream projection from a specific cursor", func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// Use unique tags to avoid interference from other tests
			uniqueID := fmt.Sprintf("stream_cursor_test_%d", time.Now().UnixNano())

			// Create some test events
			events := []dcb.InputEvent{
				dcb.NewInputEvent("UserRegistered", dcb.NewTags("user_id", uniqueID), []byte(`{"name": "Jane"}`)),
				dcb.NewInputEvent("UserNameChanged", dcb.NewTags("user_id", uniqueID), []byte(`{"name": "Jane Doe"}`)),
			}

			// Append events
			err := store.Append(ctx, events)
			Expect(err).NotTo(HaveOccurred())

			// Read first event to get cursor
			query := dcb.NewQuery(dcb.NewTags("user_id", uniqueID), "UserRegistered")
			firstEvents, err := store.Query(ctx, query, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(firstEvents).To(HaveLen(1))

			// Create cursor from first event
			cursor := &dcb.Cursor{
				TransactionID: firstEvents[0].TransactionID,
				Position:      firstEvents[0].Position,
			}

			// Create projector
			projector := dcb.StateProjector{
				ID:    "user",
				Query: dcb.NewQuery(dcb.NewTags("user_id", uniqueID)),
				InitialState: map[string]interface{}{
					"name": "",
				},
				TransitionFn: func(state any, event dcb.Event) any {
					userState := state.(map[string]interface{})
					if event.Type == "UserCreated" || event.Type == "UserNameChanged" {
						userState["name"] = "streamed"
					}
					return userState
				},
			}

			// Stream projection from cursor
			statesChan, appendConditionChan, err := store.ProjectStream(ctx, []dcb.StateProjector{projector}, cursor)
			Expect(err).NotTo(HaveOccurred())

			// Collect results
			var finalStates map[string]any
			var finalAppendCondition dcb.AppendCondition

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
