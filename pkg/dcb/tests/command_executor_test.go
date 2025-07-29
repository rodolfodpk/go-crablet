package dcb

import (
	"context"
	"encoding/json"

	"github.com/rodolfodpk/go-crablet/pkg/dcb"

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

				// Execute command using function-based handler
				_, err = commandExecutor.ExecuteCommand(ctx, command, dcb.CommandHandlerFunc(handleTestCommand), nil)
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
				_, err := commandExecutor.ExecuteCommand(ctx, nil, dcb.CommandHandlerFunc(handleTestCommand), nil)
				Expect(err).To(HaveOccurred())
				Expect(err).To(BeAssignableToTypeOf(&dcb.ValidationError{}))
			})
		})

		Context("with nil handler", func() {
			It("should return validation error", func() {
				command := dcb.NewCommand("test", []byte("{}"), nil)
				_, err := commandExecutor.ExecuteCommand(ctx, command, nil, nil)
				Expect(err).To(HaveOccurred())
				Expect(err).To(BeAssignableToTypeOf(&dcb.ValidationError{}))
			})
		})

		Context("with handler that returns no events", func() {
			It("should return validation error", func() {
				command := dcb.NewCommand("empty_command", []byte("{}"), nil)
				_, err := commandExecutor.ExecuteCommand(ctx, command, dcb.CommandHandlerFunc(handleEmptyCommand), nil)
				Expect(err).To(HaveOccurred())
				Expect(err).To(BeAssignableToTypeOf(&dcb.ValidationError{}))
			})
		})
	})

})

// Test command handler functions
func handleTestCommand(ctx context.Context, eventStore dcb.EventStore, command dcb.Command) ([]dcb.InputEvent, error) {
	var cmdData map[string]interface{}
	if err := json.Unmarshal(command.GetData(), &cmdData); err != nil {
		return nil, err
	}

	eventData := map[string]interface{}{
		"message": cmdData["message"],
		"echoed":  true,
	}

	eventBytes, err := json.Marshal(eventData)
	if err != nil {
		return nil, err
	}

	return []dcb.InputEvent{
		dcb.NewInputEvent("test_event", []dcb.Tag{
			dcb.NewTag("test_tag", "test_value"),
		}, eventBytes),
	}, nil
}

func handleEmptyCommand(ctx context.Context, eventStore dcb.EventStore, command dcb.Command) ([]dcb.InputEvent, error) {
	return nil, nil // Return empty events to test validation
}
