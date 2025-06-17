package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"go-crablet/internal/examples/utils"
	"go-crablet/pkg/dcb"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Command types
type CreateUserCommand struct {
	UserID   string
	Username string
	Email    string
}

type CreateOrderCommand struct {
	OrderID string
	UserID  string
	Items   []OrderItem
}

type OrderItem struct {
	ProductID string
	Quantity  int
	Price     float64
}

func main() {
	ctx := context.Background()

	// Connect to PostgreSQL
	pool, err := pgxpool.New(ctx, "postgres://postgres:postgres@localhost:5432/dcb_app?sslmode=disable")
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	// Create event store
	store, err := dcb.NewEventStore(ctx, pool)
	if err != nil {
		log.Fatalf("Failed to create event store: %v", err)
	}

	// Command 1: Create User
	createUserCmd := CreateUserCommand{
		UserID:   "user123",
		Username: "john_doe",
		Email:    "john@example.com",
	}
	err = handleCreateUser(ctx, store, createUserCmd)
	if err != nil {
		log.Fatalf("Create user failed: %v", err)
	}

	// Command 2: Create Order
	createOrderCmd := CreateOrderCommand{
		OrderID: "order456",
		UserID:  "user123",
		Items: []OrderItem{
			{ProductID: "prod1", Quantity: 2, Price: 29.99},
			{ProductID: "prod2", Quantity: 1, Price: 49.99},
		},
	}
	err = handleCreateOrder(ctx, store, createOrderCmd)
	if err != nil {
		log.Fatalf("Create order failed: %v", err)
	}

	// Demonstrate batch operations with multiple commands
	fmt.Println("\n=== Batch Operations ===")

	// Command 3: Create multiple users in a batch
	users := []CreateUserCommand{
		{UserID: "user456", Username: "jane_smith", Email: "jane@example.com"},
		{UserID: "user789", Username: "bob_wilson", Email: "bob@example.com"},
	}

	err = handleBatchCreateUsers(ctx, store, users)
	if err != nil {
		log.Fatalf("Batch create users failed: %v", err)
	}

	// Command 4: Create multiple orders in a batch
	orders := []CreateOrderCommand{
		{
			OrderID: "order789",
			UserID:  "user456",
			Items: []OrderItem{
				{ProductID: "prod3", Quantity: 1, Price: 19.99},
			},
		},
		{
			OrderID: "order101",
			UserID:  "user789",
			Items: []OrderItem{
				{ProductID: "prod1", Quantity: 3, Price: 29.99},
				{ProductID: "prod4", Quantity: 1, Price: 99.99},
			},
		},
	}

	err = handleBatchCreateOrders(ctx, store, orders)
	if err != nil {
		log.Fatalf("Batch create orders failed: %v", err)
	}

	// Dump all events to show what was created
	fmt.Println("\n=== Events in Database ===")
	utils.DumpEvents(ctx, pool)
}

// Command handlers with their own business rules

func handleCreateUser(ctx context.Context, store dcb.EventStore, cmd CreateUserCommand) error {
	// Command-specific projectors
	projectors := []dcb.BatchProjector{
		{ID: "userExists", StateProjector: dcb.StateProjector{
			Query: dcb.NewQuery(
				dcb.NewTags("user_id", cmd.UserID),
				"UserCreated",
			),
			InitialState: false,
			TransitionFn: func(state any, event dcb.Event) any {
				return true // If we see a UserCreated event, user exists
			},
		}},
		{ID: "emailExists", StateProjector: dcb.StateProjector{
			Query: dcb.NewQuery(
				dcb.NewTags("email", cmd.Email),
				"UserCreated",
			),
			InitialState: false,
			TransitionFn: func(state any, event dcb.Event) any {
				return true // If we see a UserCreated event with this email, email exists
			},
		}},
	}

	states, appendCondition, err := store.ProjectDecisionModel(ctx, projectors, nil)
	if err != nil {
		return fmt.Errorf("failed to check user existence: %w", err)
	}

	// Command-specific business rules
	if states["userExists"].(bool) {
		return fmt.Errorf("user %s already exists", cmd.UserID)
	}
	if states["emailExists"].(bool) {
		return fmt.Errorf("email %s already exists", cmd.Email)
	}

	// Create events for this command
	events := []dcb.InputEvent{
		dcb.NewInputEvent(
			"UserCreated",
			dcb.NewTags("user_id", cmd.UserID, "email", cmd.Email),
			toJSON(UserCreatedData{
				UserID:   cmd.UserID,
				Username: cmd.Username,
				Email:    cmd.Email,
			}),
		),
	}

	// Append events atomically for this command
	position, err := store.Append(ctx, events, &appendCondition)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	fmt.Printf("Created user %s (%s) at position %d\n", cmd.Username, cmd.Email, position)
	return nil
}

func handleCreateOrder(ctx context.Context, store dcb.EventStore, cmd CreateOrderCommand) error {
	// Command-specific projectors
	projectors := []dcb.BatchProjector{
		{ID: "orderExists", StateProjector: dcb.StateProjector{
			Query: dcb.NewQuery(
				dcb.NewTags("order_id", cmd.OrderID),
				"OrderCreated",
			),
			InitialState: false,
			TransitionFn: func(state any, event dcb.Event) any {
				return true // If we see an OrderCreated event, order exists
			},
		}},
		{ID: "userExists", StateProjector: dcb.StateProjector{
			Query: dcb.NewQuery(
				dcb.NewTags("user_id", cmd.UserID),
				"UserCreated",
			),
			InitialState: false,
			TransitionFn: func(state any, event dcb.Event) any {
				return true // If we see a UserCreated event, user exists
			},
		}},
	}

	states, appendCondition, err := store.ProjectDecisionModel(ctx, projectors, nil)
	if err != nil {
		return fmt.Errorf("failed to check order and user existence: %w", err)
	}

	// Command-specific business rules
	if states["orderExists"].(bool) {
		return fmt.Errorf("order %s already exists", cmd.OrderID)
	}
	if !states["userExists"].(bool) {
		return fmt.Errorf("user %s does not exist", cmd.UserID)
	}

	// Calculate total
	total := 0.0
	for _, item := range cmd.Items {
		total += float64(item.Quantity) * item.Price
	}

	// Create events for this command
	events := []dcb.InputEvent{
		dcb.NewInputEvent(
			"OrderCreated",
			dcb.NewTags("order_id", cmd.OrderID, "user_id", cmd.UserID),
			toJSON(OrderCreatedData{
				OrderID: cmd.OrderID,
				UserID:  cmd.UserID,
				Items:   cmd.Items,
				Total:   total,
			}),
		),
	}

	// Append events atomically for this command
	position, err := store.Append(ctx, events, &appendCondition)
	if err != nil {
		return fmt.Errorf("failed to create order: %w", err)
	}

	fmt.Printf("Created order %s for user %s with total %.2f at position %d\n", cmd.OrderID, cmd.UserID, total, position)
	return nil
}

func handleBatchCreateUsers(ctx context.Context, store dcb.EventStore, commands []CreateUserCommand) error {
	// Batch-specific projectors to check all users and emails at once
	projectors := []dcb.BatchProjector{}

	// Add projectors for each user and email
	for _, cmd := range commands {
		projectors = append(projectors, dcb.BatchProjector{
			ID: fmt.Sprintf("userExists_%s", cmd.UserID),
			StateProjector: dcb.StateProjector{
				Query: dcb.NewQuery(
					dcb.NewTags("user_id", cmd.UserID),
					"UserCreated",
				),
				InitialState: false,
				TransitionFn: func(state any, event dcb.Event) any {
					return true
				},
			},
		})

		projectors = append(projectors, dcb.BatchProjector{
			ID: fmt.Sprintf("emailExists_%s", cmd.Email),
			StateProjector: dcb.StateProjector{
				Query: dcb.NewQuery(
					dcb.NewTags("email", cmd.Email),
					"UserCreated",
				),
				InitialState: false,
				TransitionFn: func(state any, event dcb.Event) any {
					return true
				},
			},
		})
	}

	states, appendCondition, err := store.ProjectDecisionModel(ctx, projectors, nil)
	if err != nil {
		return fmt.Errorf("failed to check batch user existence: %w", err)
	}

	// Batch-specific business rules
	for _, cmd := range commands {
		if states[fmt.Sprintf("userExists_%s", cmd.UserID)].(bool) {
			return fmt.Errorf("user %s already exists", cmd.UserID)
		}
		if states[fmt.Sprintf("emailExists_%s", cmd.Email)].(bool) {
			return fmt.Errorf("email %s already exists", cmd.Email)
		}
	}

	// Create events for all commands in the batch
	events := []dcb.InputEvent{}
	for _, cmd := range commands {
		events = append(events, dcb.NewInputEvent(
			"UserCreated",
			dcb.NewTags("user_id", cmd.UserID, "email", cmd.Email),
			toJSON(UserCreatedData{
				UserID:   cmd.UserID,
				Username: cmd.Username,
				Email:    cmd.Email,
			}),
		))
	}

	// Append all events atomically for this batch
	position, err := store.Append(ctx, events, &appendCondition)
	if err != nil {
		return fmt.Errorf("failed to batch create users: %w", err)
	}

	fmt.Printf("Batch created %d users at position %d\n", len(commands), position)
	return nil
}

func handleBatchCreateOrders(ctx context.Context, store dcb.EventStore, commands []CreateOrderCommand) error {
	// Batch-specific projectors to check all orders and users at once
	projectors := []dcb.BatchProjector{}

	// Add projectors for each order and user
	for _, cmd := range commands {
		projectors = append(projectors, dcb.BatchProjector{
			ID: fmt.Sprintf("orderExists_%s", cmd.OrderID),
			StateProjector: dcb.StateProjector{
				Query: dcb.NewQuery(
					dcb.NewTags("order_id", cmd.OrderID),
					"OrderCreated",
				),
				InitialState: false,
				TransitionFn: func(state any, event dcb.Event) any {
					return true
				},
			},
		})

		projectors = append(projectors, dcb.BatchProjector{
			ID: fmt.Sprintf("userExists_%s", cmd.UserID),
			StateProjector: dcb.StateProjector{
				Query: dcb.NewQuery(
					dcb.NewTags("user_id", cmd.UserID),
					"UserCreated",
				),
				InitialState: false,
				TransitionFn: func(state any, event dcb.Event) any {
					return true
				},
			},
		})
	}

	states, appendCondition, err := store.ProjectDecisionModel(ctx, projectors, nil)
	if err != nil {
		return fmt.Errorf("failed to check batch order existence: %w", err)
	}

	// Batch-specific business rules
	for _, cmd := range commands {
		if states[fmt.Sprintf("orderExists_%s", cmd.OrderID)].(bool) {
			return fmt.Errorf("order %s already exists", cmd.OrderID)
		}
		if !states[fmt.Sprintf("userExists_%s", cmd.UserID)].(bool) {
			return fmt.Errorf("user %s does not exist", cmd.UserID)
		}
	}

	// Create events for all commands in the batch
	events := []dcb.InputEvent{}
	for _, cmd := range commands {
		// Calculate total for this order
		total := 0.0
		for _, item := range cmd.Items {
			total += float64(item.Quantity) * item.Price
		}

		events = append(events, dcb.NewInputEvent(
			"OrderCreated",
			dcb.NewTags("order_id", cmd.OrderID, "user_id", cmd.UserID),
			toJSON(OrderCreatedData{
				OrderID: cmd.OrderID,
				UserID:  cmd.UserID,
				Items:   cmd.Items,
				Total:   total,
			}),
		))
	}

	// Append all events atomically for this batch
	position, err := store.Append(ctx, events, &appendCondition)
	if err != nil {
		return fmt.Errorf("failed to batch create orders: %w", err)
	}

	fmt.Printf("Batch created %d orders at position %d\n", len(commands), position)
	return nil
}

// Helper types
type UserCreatedData struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

type OrderCreatedData struct {
	OrderID string      `json:"order_id"`
	UserID  string      `json:"user_id"`
	Items   []OrderItem `json:"items"`
	Total   float64     `json:"total"`
}

func toJSON(v any) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("Failed to marshal: %v", err))
	}
	return data
}
