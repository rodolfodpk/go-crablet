package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"go-crablet/pkg/dcb"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Example command types
const (
	CommandTypeCreateUser = "create_user"
	CommandTypeUpdateUser = "update_user"
)

// Example event types
const (
	EventTypeUserCreated = "user_created"
	EventTypeUserUpdated = "user_updated"
)

// User data structures
type CreateUserCommand struct {
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type UpdateUserCommand struct {
	UserID    string `json:"user_id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type UserCreatedEvent struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	CreatedAt string `json:"created_at"`
}

type UserUpdatedEvent struct {
	UserID    string `json:"user_id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	UpdatedAt string `json:"updated_at"`
}

// UserCommandHandler implements CommandHandler interface
type UserCommandHandler struct{}

func (h *UserCommandHandler) Handle(ctx context.Context, decisionModels map[string]any, command dcb.Command) []dcb.InputEvent {
	switch command.GetType() {
	case CommandTypeCreateUser:
		return h.handleCreateUser(command)
	case CommandTypeUpdateUser:
		return h.handleUpdateUser(command)
	default:
		log.Printf("Unknown command type: %s", command.GetType())
		return nil
	}
}

func (h *UserCommandHandler) handleCreateUser(command dcb.Command) []dcb.InputEvent {
	var cmd CreateUserCommand
	if err := json.Unmarshal(command.GetData(), &cmd); err != nil {
		log.Printf("Failed to unmarshal create user command: %v", err)
		return nil
	}

	// Generate user ID (in real app, this might come from a service)
	userID := fmt.Sprintf("user_%d", time.Now().UnixNano())

	// Create the event
	event := UserCreatedEvent{
		UserID:    userID,
		Email:     cmd.Email,
		FirstName: cmd.FirstName,
		LastName:  cmd.LastName,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	eventData, err := json.Marshal(event)
	if err != nil {
		log.Printf("Failed to marshal user created event: %v", err)
		return nil
	}

	return []dcb.InputEvent{
		dcb.NewInputEvent(EventTypeUserCreated, []dcb.Tag{
			dcb.NewTag("user_id", userID),
			dcb.NewTag("email", cmd.Email),
		}, eventData),
	}
}

func (h *UserCommandHandler) handleUpdateUser(command dcb.Command) []dcb.InputEvent {
	var cmd UpdateUserCommand
	if err := json.Unmarshal(command.GetData(), &cmd); err != nil {
		log.Printf("Failed to unmarshal update user command: %v", err)
		return nil
	}

	// Create the event
	event := UserUpdatedEvent{
		UserID:    cmd.UserID,
		FirstName: cmd.FirstName,
		LastName:  cmd.LastName,
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	eventData, err := json.Marshal(event)
	if err != nil {
		log.Printf("Failed to marshal user updated event: %v", err)
		return nil
	}

	return []dcb.InputEvent{
		dcb.NewInputEvent(EventTypeUserUpdated, []dcb.Tag{
			dcb.NewTag("user_id", cmd.UserID),
		}, eventData),
	}
}

func main() {
	ctx := context.Background()

	// Connect to database
	pool, err := pgxpool.New(ctx, "postgres://postgres:postgres@localhost:5432/dcb_app?sslmode=disable")
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	// Create event store
	eventStore, err := dcb.NewEventStoreWithConfig(ctx, pool, dcb.EventStoreConfig{
		MaxBatchSize:           1000,
		LockTimeout:            5000,
		StreamBuffer:           100,
		DefaultAppendIsolation: dcb.IsolationLevelReadCommitted,
		QueryTimeout:           30000,
		AppendTimeout:          30000,
		TargetEventsTable:      "events",
	})
	if err != nil {
		log.Fatalf("Failed to create event store: %v", err)
	}

	// Create command executor
	commandExecutor := dcb.NewCommandExecutor(eventStore)

	// Create command handler
	handler := &UserCommandHandler{}

	// Example 1: Create a user
	fmt.Println("=== Creating User ===")
	createCmd := CreateUserCommand{
		Email:     "john.doe@example.com",
		FirstName: "John",
		LastName:  "Doe",
	}

	createCmdData, err := json.Marshal(createCmd)
	if err != nil {
		log.Fatalf("Failed to marshal create command: %v", err)
	}

	command := dcb.NewCommand(CommandTypeCreateUser, createCmdData, map[string]interface{}{
		"user_id": "request_123",
		"source":  "web_api",
	})

	err = commandExecutor.ExecuteCommand(ctx, command, handler, nil)
	if err != nil {
		log.Fatalf("Failed to execute create user command: %v", err)
	}

	fmt.Println("✓ User created successfully")

	// Example 2: Update a user
	fmt.Println("\n=== Updating User ===")
	updateCmd := UpdateUserCommand{
		UserID:    "user_1234567890", // This would come from the previous command
		FirstName: "Jane",
		LastName:  "Smith",
	}

	updateCmdData, err := json.Marshal(updateCmd)
	if err != nil {
		log.Fatalf("Failed to marshal update command: %v", err)
	}

	updateCommand := dcb.NewCommand(CommandTypeUpdateUser, updateCmdData, map[string]interface{}{
		"user_id": "request_456",
		"source":  "web_api",
	})

	err = commandExecutor.ExecuteCommand(ctx, updateCommand, handler, nil)
	if err != nil {
		log.Fatalf("Failed to execute update user command: %v", err)
	}

	fmt.Println("✓ User updated successfully")

	// Example 3: Query events to see what was created
	fmt.Println("\n=== Querying Events ===")
	events, err := eventStore.Query(ctx, dcb.NewQueryAll(), nil)
	if err != nil {
		log.Fatalf("Failed to query events: %v", err)
	}

	fmt.Printf("Found %d events:\n", len(events))
	for i, event := range events {
		fmt.Printf("  %d. Type: %s, Tags: %v, Transaction: %d, Position: %d\n",
			i+1, event.Type, event.Tags, event.TransactionID, event.Position)
	}

	// Example 4: Query commands to see what was persisted
	fmt.Println("\n=== Querying Commands ===")
	rows, err := pool.Query(ctx, `
		SELECT transaction_id, type, data, metadata, target_events_table, occurred_at
		FROM commands
		ORDER BY occurred_at ASC
	`)
	if err != nil {
		log.Fatalf("Failed to query commands: %v", err)
	}
	defer rows.Close()

	commandCount := 0
	for rows.Next() {
		var (
			txID              uint64
			cmdType           string
			cmdData           []byte
			cmdMetadata       []byte
			targetEventsTable string
			occurredAt        time.Time
		)

		err := rows.Scan(&txID, &cmdType, &cmdData, &cmdMetadata, &targetEventsTable, &occurredAt)
		if err != nil {
			log.Printf("Failed to scan command row: %v", err)
			continue
		}

		commandCount++
		fmt.Printf("  %d. Type: %s, Transaction: %d, Target Table: %s, Occurred: %s\n",
			commandCount, cmdType, txID, targetEventsTable, occurredAt.Format(time.RFC3339))

		// Pretty print command data
		var prettyData interface{}
		if err := json.Unmarshal(cmdData, &prettyData); err == nil {
			if prettyJSON, err := json.MarshalIndent(prettyData, "    ", "  "); err == nil {
				fmt.Printf("    Data: %s\n", string(prettyJSON))
			}
		}

		// Pretty print metadata if present
		if len(cmdMetadata) > 0 {
			var prettyMetadata interface{}
			if err := json.Unmarshal(cmdMetadata, &prettyMetadata); err == nil {
				if prettyJSON, err := json.MarshalIndent(prettyMetadata, "    ", "  "); err == nil {
					fmt.Printf("    Metadata: %s\n", string(prettyJSON))
				}
			}
		}
	}

	if commandCount == 0 {
		fmt.Println("  No commands found in the database")
	} else {
		fmt.Printf("  Total commands found: %d\n", commandCount)
	}

	fmt.Println("\n=== Example completed successfully ===")
}
