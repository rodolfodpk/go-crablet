package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"go-crablet/pkg/dcb"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	// Connect to PostgreSQL
	pool, err := pgxpool.New(context.Background(), "postgres://postgres:postgres@localhost:5432/dcb_app?sslmode=disable")
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	// Create event store
	store, err := dcb.NewEventStore(context.Background(), pool)
	if err != nil {
		log.Fatalf("Failed to create event store: %v", err)
	}

	// Cast to CrabletEventStore to access extended methods
	channelStore := store.(dcb.CrabletEventStore)

	ctx := context.Background()

	// Create a large number of events to demonstrate streaming
	fmt.Println("Creating test events...")
	events := createTestEvents(1000) // Create 1000 events

	// Append events
	position, err := store.Append(ctx, events, nil)
	if err != nil {
		log.Fatalf("Failed to append events: %v", err)
	}
	fmt.Printf("Appended %d events, last position: %d\n", len(events), position)

	// Define projectors
	projectors := []dcb.BatchProjector{
		{
			ID: "order_count",
			StateProjector: dcb.StateProjector{
				Query: dcb.NewQuery(
					dcb.NewTags("customer_id", "customer-1"),
					"OrderCreated",
				),
				InitialState: 0,
				TransitionFn: func(state any, event dcb.Event) any {
					count := state.(int)
					return count + 1
				},
			},
		},
		{
			ID: "total_amount",
			StateProjector: dcb.StateProjector{
				Query: dcb.NewQuery(
					dcb.NewTags("customer_id", "customer-1"),
					"OrderCreated",
				),
				InitialState: 0.0,
				TransitionFn: func(state any, event dcb.Event) any {
					total := state.(float64)
					var data map[string]interface{}
					if err := json.Unmarshal(event.Data, &data); err == nil {
						if amount, ok := data["amount"].(float64); ok {
							return total + amount
						}
					}
					return total
				},
			},
		},
	}

	// Query for events
	query := dcb.NewQuery(
		dcb.NewTags("customer_id", "customer-1"),
		"OrderCreated",
	)

	fmt.Println("\n=== Using cursor-based streaming (BatchSize: 100) ===")
	batchSize := 100
	options := &dcb.ReadOptions{BatchSize: &batchSize}

	// Test ProjectDecisionModel with cursor streaming
	states, appendCondition, err := channelStore.ProjectDecisionModel(ctx, projectors)
	if err != nil {
		log.Fatalf("Failed to project decision model: %v", err)
	}

	fmt.Printf("Order count: %d\n", states["order_count"])
	fmt.Printf("Total amount: $%.2f\n", states["total_amount"])
	fmt.Printf("Last position: %d\n", *appendCondition.After)

	fmt.Println("\n=== ReadStream has been removed - use Read for batch operations ===")
	// ReadStream method has been removed from the interface
	// Use Read for batch operations or ReadStreamChannel for streaming

	// Example using Read for batch operations
	result, err := store.Read(ctx, query, options)
	if err != nil {
		log.Fatalf("Failed to read events: %v", err)
	}

	count := 0
	for _, event := range result.Events {
		count++
		if count <= 5 || count%100 == 0 {
			fmt.Printf("Event %d: %s at position %d\n", count, event.Type, event.Position)
		}
	}

	fmt.Printf("Processed %d events via batch read\n", count)
}

func createTestEvents(count int) []dcb.InputEvent {
	events := make([]dcb.InputEvent, count)
	for i := 0; i < count; i++ {
		customerID := fmt.Sprintf("customer-%d", (i%10)+1) // 10 different customers
		amount := float64((i%1000)+1) * 10.0               // Random amounts

		data, _ := json.Marshal(map[string]interface{}{
			"order_id": fmt.Sprintf("order-%d", i+1),
			"amount":   amount,
		})

		event := dcb.NewInputEvent(
			"OrderCreated",
			dcb.NewTags(
				"customer_id", customerID,
				"order_id", fmt.Sprintf("order-%d", i+1),
			),
			data,
		)

		events[i] = event
	}
	return events
}
