package dcb

import (
	"context"
	"encoding/json"

	"go-crablet/pkg/dcb"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("CommandExecutor", func() {
	var commandExecutor dcb.CommandExecutor

	BeforeEach(func() {
		// Clean up events table before each test
		err := truncateEventsTable(ctx, pool)
		Expect(err).NotTo(HaveOccurred())

		// Use the shared event store from setup_test.go
		commandExecutor = dcb.NewCommandExecutor(store)
	})

	Describe("ExecuteCommand", func() {
		Context("with valid command and handler", func() {
			It("should execute command and generate events", func() {
				// Create command
				cmdData := map[string]interface{}{
					"message": "Hello, World!",
				}
				cmdBytes, err := json.Marshal(cmdData)
				Expect(err).NotTo(HaveOccurred())

				command := dcb.NewCommand("test_command", cmdBytes, map[string]interface{}{
					"test_id": "123",
					"source":  "test",
				})

				// Execute command
				err = commandExecutor.ExecuteCommand(ctx, command, &testCommandHandler{}, nil)
				Expect(err).NotTo(HaveOccurred())

				// Verify events were created
				events, err := store.Query(ctx, dcb.NewQueryAll(), nil)
				Expect(err).NotTo(HaveOccurred())
				Expect(events).To(HaveLen(1))

				event := events[0]
				Expect(event.Type).To(Equal("test_event"))
				Expect(event.Tags).To(HaveLen(1))
				Expect(event.Tags[0].GetKey()).To(Equal("test_tag"))
				Expect(event.Tags[0].GetValue()).To(Equal("test_value"))

				// Verify event data
				var eventData map[string]interface{}
				err = json.Unmarshal(event.Data, &eventData)
				Expect(err).NotTo(HaveOccurred())
				Expect(eventData["message"]).To(Equal("Hello, World!"))
				Expect(eventData["echoed"]).To(BeTrue())
			})
		})

		Context("with nil command", func() {
			It("should return validation error", func() {
				err := commandExecutor.ExecuteCommand(ctx, nil, &testCommandHandler{}, nil)
				Expect(err).To(HaveOccurred())
				Expect(err).To(BeAssignableToTypeOf(&dcb.ValidationError{}))
			})
		})

		Context("with nil handler", func() {
			It("should return validation error", func() {
				command := dcb.NewCommand("test", []byte("{}"), nil)
				err := commandExecutor.ExecuteCommand(ctx, command, nil, nil)
				Expect(err).To(HaveOccurred())
				Expect(err).To(BeAssignableToTypeOf(&dcb.ValidationError{}))
			})
		})

		Context("with handler that returns no events", func() {
			It("should return validation error", func() {
				command := dcb.NewCommand("empty_command", []byte("{}"), nil)
				err := commandExecutor.ExecuteCommand(ctx, command, &emptyCommandHandler{}, nil)
				Expect(err).To(HaveOccurred())
				Expect(err).To(BeAssignableToTypeOf(&dcb.ValidationError{}))
			})
		})
	})
})

// Test command handlers
type testCommandHandler struct{}

func (h *testCommandHandler) Handle(ctx context.Context, eventStore dcb.EventStore, command dcb.Command) []dcb.InputEvent {
	var cmdData map[string]interface{}
	json.Unmarshal(command.GetData(), &cmdData)

	eventData := map[string]interface{}{
		"message": cmdData["message"],
		"echoed":  true,
	}

	eventBytes, _ := json.Marshal(eventData)

	return []dcb.InputEvent{
		dcb.NewInputEvent("test_event", []dcb.Tag{
			dcb.NewTag("test_tag", "test_value"),
		}, eventBytes),
	}
}

type emptyCommandHandler struct{}

func (h *emptyCommandHandler) Handle(ctx context.Context, eventStore dcb.EventStore, command dcb.Command) []dcb.InputEvent {
	return nil // Return empty events to test validation
}
