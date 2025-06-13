package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"go-crablet/pkg/dcb"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	// Connect to PostgreSQL
	pool, err := pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	// Create event store
	store, err := dcb.NewEventStore(context.Background(), pool)
	if err != nil {
		log.Fatalf("Failed to create event store: %v", err)
	}

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
				Query: dcb.Query{
					Items: []dcb.QueryItem{
						{
							EventTypes: []string{"OrderCreated"},
							Tags:       []dcb.Tag{{Key: "customer_id", Value: "customer-1"}},
						},
					},
				},
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
				Query: dcb.Query{
					Items: []dcb.QueryItem{
						{
							EventTypes: []string{"OrderCreated"},
							Tags:       []dcb.Tag{{Key: "customer_id", Value: "customer-1"}},
						},
					},
				},
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
	query := dcb.Query{
		Items: []dcb.QueryItem{
			{
				EventTypes: []string{"OrderCreated"},
				Tags:       []dcb.Tag{{Key: "customer_id", Value: "customer-1"}},
			},
		},
	}

	fmt.Println("\n=== Using cursor-based streaming (BatchSize: 100) ===")
	batchSize := 100
	options := &dcb.ReadOptions{BatchSize: &batchSize}

	// Test ProjectDecisionModel with cursor streaming
	states, appendCondition, err := store.ProjectDecisionModel(ctx, query, options, projectors)
	if err != nil {
		log.Fatalf("Failed to project decision model: %v", err)
	}

	fmt.Printf("Order count: %d\n", states["order_count"])
	fmt.Printf("Total amount: $%.2f\n", states["total_amount"])
	fmt.Printf("Last position: %d\n", *appendCondition.After)

	fmt.Println("\n=== Using ReadStream with cursor-based streaming ===")
	iterator, err := store.ReadStream(ctx, query, options)
	if err != nil {
		log.Fatalf("Failed to create stream: %v", err)
	}
	defer iterator.Close()

	count := 0
	for iterator.Next() {
		event := iterator.Event()
		count++
		if count <= 5 || count%100 == 0 {
			fmt.Printf("Event %d: %s at position %d\n", count, event.Type, event.Position)
		}
	}

	if err := iterator.Err(); err != nil {
		log.Fatalf("Error during iteration: %v", err)
	}

	fmt.Printf("Processed %d events via streaming\n", count)
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

		events[i] = dcb.InputEvent{
			Type: "OrderCreated",
			Tags: []dcb.Tag{
				{Key: "customer_id", Value: customerID},
				{Key: "order_id", Value: fmt.Sprintf("order-%d", i+1)},
			},
			Data: data,
		}
	}
	return events
}
