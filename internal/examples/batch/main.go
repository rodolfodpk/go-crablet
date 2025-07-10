package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

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

	// Connect to database
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

	// Cast to EventStore for extended functionality
	// Use store directly

	fmt.Println("=== Batch Processing Example ===")
	fmt.Println("This example demonstrates batch processing with DCB-inspired event sourcing.")
	fmt.Println()

	// Single command examples
	fmt.Println("1. Single Command Processing:")
	if err := handleCreateUser(ctx, store, CreateUserCommand{
		UserID:   "user1",
		Username: "john_doe",
		Email:    "john@example.com",
	}); err != nil {
		log.Printf("Failed to create user: %v", err)
	}

	if err := handleCreateOrder(ctx, store, CreateOrderCommand{
		OrderID: "order1",
		UserID:  "user1",
		Items: []OrderItem{
			{ProductID: "prod1", Quantity: 2, Price: 10.0},
			{ProductID: "prod2", Quantity: 1, Price: 15.0},
		},
	}); err != nil {
		log.Printf("Failed to create order: %v", err)
	}

	fmt.Println()

	// Batch command examples
	fmt.Println("2. Batch Command Processing:")
	batchUsers := []CreateUserCommand{
		{UserID: "user2", Username: "jane_smith", Email: "jane@example.com"},
		{UserID: "user3", Username: "bob_wilson", Email: "bob@example.com"},
		{UserID: "user4", Username: "alice_brown", Email: "alice@example.com"},
	}

	if err := handleBatchCreateUsers(ctx, store, batchUsers); err != nil {
		log.Printf("Failed to batch create users: %v", err)
	}

	batchOrders := []CreateOrderCommand{
		{
			OrderID: "order2",
			UserID:  "user2",
			Items:   []OrderItem{{ProductID: "prod3", Quantity: 1, Price: 25.0}},
		},
		{
			OrderID: "order3",
			UserID:  "user3",
			Items:   []OrderItem{{ProductID: "prod4", Quantity: 3, Price: 8.0}},
		},
	}

	if err := handleBatchCreateOrders(ctx, store, batchOrders); err != nil {
		log.Printf("Failed to batch create orders: %v", err)
	}

	fmt.Println()
	fmt.Println("=== Example Complete ===")
}

// Command handlers with their own business rules

func handleCreateUser(ctx context.Context, store dcb.EventStore, cmd CreateUserCommand) error {
	// Command-specific projectors
	projectors := []dcb.StateProjector{
		{
			ID: "userExists",
			Query: dcb.NewQuery(
				dcb.NewTags("user_id", cmd.UserID),
				"UserCreated",
			),
			InitialState: false,
			TransitionFn: func(state any, event dcb.Event) any {
				return true // If we see a UserCreated event, user exists
			},
		},
		{
			ID: "emailExists",
			Query: dcb.NewQuery(
				dcb.NewTags("email", cmd.Email),
				"UserCreated",
			),
			InitialState: false,
			TransitionFn: func(state any, event dcb.Event) any {
				return true // If we see a UserCreated event with this email, email exists
			},
		},
	}

	states, _, err := store.Project(ctx, projectors, nil)
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
	err = store.Append(ctx, events, nil)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	fmt.Printf("Created user %s (%s)\n", cmd.Username, cmd.Email)
	return nil
}

func handleCreateOrder(ctx context.Context, store dcb.EventStore, cmd CreateOrderCommand) error {
	// Command-specific projectors
	projectors := []dcb.StateProjector{
		{
			ID: "orderExists",
			Query: dcb.NewQuery(
				dcb.NewTags("order_id", cmd.OrderID),
				"OrderCreated",
			),
			InitialState: false,
			TransitionFn: func(state any, event dcb.Event) any {
				return true // If we see an OrderCreated event, order exists
			},
		},
		{
			ID: "userExists",
			Query: dcb.NewQuery(
				dcb.NewTags("user_id", cmd.UserID),
				"UserCreated",
			),
			InitialState: false,
			TransitionFn: func(state any, event dcb.Event) any {
				return true // If we see a UserCreated event, user exists
			},
		},
	}

	states, _, err := store.Project(ctx, projectors, nil)
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
	err = store.Append(ctx, events, nil)
	if err != nil {
		return fmt.Errorf("failed to create order: %w", err)
	}

	fmt.Printf("Created order %s for user %s with total %.2f\n", cmd.OrderID, cmd.UserID, total)
	return nil
}

func handleBatchCreateUsers(ctx context.Context, store dcb.EventStore, commands []CreateUserCommand) error {
	// Batch-specific projectors to check all users and emails at once
	projectors := []dcb.StateProjector{}

	// Add projectors for each user and email
	for _, cmd := range commands {
		projectors = append(projectors, dcb.StateProjector{
			ID: fmt.Sprintf("userExists_%s", cmd.UserID),
			Query: dcb.NewQuery(
				dcb.NewTags("user_id", cmd.UserID),
				"UserCreated",
			),
			InitialState: false,
			TransitionFn: func(state any, event dcb.Event) any {
				return true
			},
		})

		projectors = append(projectors, dcb.StateProjector{
			ID: fmt.Sprintf("emailExists_%s", cmd.Email),
			Query: dcb.NewQuery(
				dcb.NewTags("email", cmd.Email),
				"UserCreated",
			),
			InitialState: false,
			TransitionFn: func(state any, event dcb.Event) any {
				return true
			},
		})
	}

	states, _, err := store.Project(ctx, projectors, nil)
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

	// Append events atomically for this batch
	err = store.Append(ctx, events, nil)
	if err != nil {
		return fmt.Errorf("failed to batch create users: %w", err)
	}

	fmt.Printf("Created %d users in batch\n", len(commands))
	return nil
}

func handleBatchCreateOrders(ctx context.Context, store dcb.EventStore, commands []CreateOrderCommand) error {
	// Batch-specific projectors to check all orders and users at once
	projectors := []dcb.StateProjector{}

	// Add projectors for each order and user
	for _, cmd := range commands {
		projectors = append(projectors, dcb.StateProjector{
			ID: fmt.Sprintf("orderExists_%s", cmd.OrderID),
			Query: dcb.NewQuery(
				dcb.NewTags("order_id", cmd.OrderID),
				"OrderCreated",
			),
			InitialState: false,
			TransitionFn: func(state any, event dcb.Event) any {
				return true
			},
		})

		projectors = append(projectors, dcb.StateProjector{
			ID: fmt.Sprintf("userExists_%s", cmd.UserID),
			Query: dcb.NewQuery(
				dcb.NewTags("user_id", cmd.UserID),
				"UserCreated",
			),
			InitialState: false,
			TransitionFn: func(state any, event dcb.Event) any {
				return true
			},
		})
	}

	states, _, err := store.Project(ctx, projectors, nil)
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

	// Append events atomically for this batch
	err = store.Append(ctx, events, nil)
	if err != nil {
		return fmt.Errorf("failed to batch create orders: %w", err)
	}

	fmt.Printf("Created %d orders in batch\n", len(commands))
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
