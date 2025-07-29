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

	Describe("ExecuteCommandWithLocks", func() {
		It("should acquire advisory locks and execute command successfully", func() {
			// Create a simple command handler that generates events without lock tags
			handler := dcb.CommandHandlerFunc(func(ctx context.Context, store dcb.EventStore, command dcb.Command) ([]dcb.InputEvent, error) {
				// Create events without lock tags (as required by ExecuteCommandWithLocks)
				events := []dcb.InputEvent{
					dcb.NewEvent("CourseDefined").
						WithTag("course_id", "CS101").
						WithData(map[string]string{"name": "Computer Science 101"}).
						Build(),
				}
				return events, nil
			})

			// Create command
			command := dcb.NewCommand("DefineCourse", dcb.ToJSON(map[string]string{"course_id": "CS101"}), nil)

			// Execute command with advisory locks
			locks := []string{"course:CS101"}
			events, err := commandExecutor.ExecuteCommandWithLocks(ctx, command, handler, locks, nil)

			Expect(err).NotTo(HaveOccurred())
			Expect(events).To(HaveLen(1))
			Expect(events[0].GetType()).To(Equal("CourseDefined"))
		})

		It("should validate that events do not contain lock tags", func() {
			// Create a command handler that generates events WITH lock tags (should fail)
			handler := dcb.CommandHandlerFunc(func(ctx context.Context, store dcb.EventStore, command dcb.Command) ([]dcb.InputEvent, error) {
				// Create events WITH lock tags (this should cause validation to fail)
				events := []dcb.InputEvent{
					dcb.NewEvent("CourseDefined").
						WithTag("course_id", "CS101").
						WithTag("lock:course", "CS101"). // This should cause validation to fail
						WithData(map[string]string{"name": "Computer Science 101"}).
						Build(),
				}
				return events, nil
			})

			// Create command
			command := dcb.NewCommand("DefineCourse", dcb.ToJSON(map[string]string{"course_id": "CS101"}), nil)

			// Execute command with advisory locks - should fail validation
			locks := []string{"course:CS101"}
			_, err := commandExecutor.ExecuteCommandWithLocks(ctx, command, handler, locks, nil)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("lock tags are not allowed when using ExecuteCommandWithLocks"))
		})

		It("should validate that locks slice is not empty", func() {
			// Create a simple command handler
			handler := dcb.CommandHandlerFunc(func(ctx context.Context, store dcb.EventStore, command dcb.Command) ([]dcb.InputEvent, error) {
				events := []dcb.InputEvent{
					dcb.NewEvent("CourseDefined").
						WithTag("course_id", "CS101").
						WithData(map[string]string{"name": "Computer Science 101"}).
						Build(),
				}
				return events, nil
			})

			// Create command
			command := dcb.NewCommand("DefineCourse", dcb.ToJSON(map[string]string{"course_id": "CS101"}), nil)

			// Execute command with empty locks slice - should fail validation
			_, err := commandExecutor.ExecuteCommandWithLocks(ctx, command, handler, []string{}, nil)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("locks slice cannot be empty"))
		})

		It("should acquire multiple advisory locks in sorted order", func() {
			// Create a simple command handler
			handler := dcb.CommandHandlerFunc(func(ctx context.Context, store dcb.EventStore, command dcb.Command) ([]dcb.InputEvent, error) {
				events := []dcb.InputEvent{
					dcb.NewEvent("TransferCompleted").
						WithTag("from_account", "123").
						WithTag("to_account", "456").
						WithData(map[string]string{"amount": "100"}).
						Build(),
				}
				return events, nil
			})

			// Create command
			command := dcb.NewCommand("TransferMoney", dcb.ToJSON(map[string]string{"amount": "100"}), nil)

			// Execute command with multiple locks in unsorted order
			// The method should sort them to prevent deadlocks
			locks := []string{"account:456", "account:123"} // Unsorted order
			events, err := commandExecutor.ExecuteCommandWithLocks(ctx, command, handler, locks, nil)

			Expect(err).NotTo(HaveOccurred())
			Expect(events).To(HaveLen(1))
			Expect(events[0].GetType()).To(Equal("TransferCompleted"))
		})

		It("should work with AppendCondition", func() {
			// Create a command handler that generates events
			handler := dcb.CommandHandlerFunc(func(ctx context.Context, store dcb.EventStore, command dcb.Command) ([]dcb.InputEvent, error) {
				events := []dcb.InputEvent{
					dcb.NewEvent("StudentEnrolled").
						WithTag("student_id", "student1").
						WithTag("course_id", "CS101").
						WithData(map[string]string{"enrolled_at": "2024-01-01"}).
						Build(),
				}
				return events, nil
			})

			// Create command
			command := dcb.NewCommand("EnrollStudent", dcb.ToJSON(map[string]string{"student_id": "student1"}), nil)

			// Create AppendCondition
			condition := dcb.FailIfExists("student_id", "student1")

			// Execute command with advisory locks and condition
			locks := []string{"course:CS101"}
			events, err := commandExecutor.ExecuteCommandWithLocks(ctx, command, handler, locks, &condition)

			Expect(err).NotTo(HaveOccurred())
			Expect(events).To(HaveLen(1))
			Expect(events[0].GetType()).To(Equal("StudentEnrolled"))
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
