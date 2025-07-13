// This example demonstrates different streaming approaches in github.com/rodolfodpk/go-crablet
// Run with: go run internal/examples/streaming/main.go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/rodolfodpk/go-crablet/pkg/dcb"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	// Create context with timeout for the entire application
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Connect to database
	pool, err := pgxpool.New(ctx, "postgres://postgres:postgres@localhost:5432/dcb_app?sslmode=disable")
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	// Truncate events table before running the example
	_, err = pool.Exec(ctx, "TRUNCATE TABLE events RESTART IDENTITY CASCADE")
	if err != nil {
		log.Fatalf("Failed to truncate events table: %v", err)
	}
	defer pool.Close()

	// Create event store
	store, err := dcb.NewEventStore(ctx, pool)
	if err != nil {
		log.Fatalf("Failed to create event store: %v", err)
	}

	// Create test events
	events := []dcb.InputEvent{
		dcb.NewInputEvent("UserCreated", dcb.NewTags("user_id", "user-1"), []byte(`{"name": "Alice"}`)),
		dcb.NewInputEvent("UserCreated", dcb.NewTags("user_id", "user-2"), []byte(`{"name": "Bob"}`)),
		dcb.NewInputEvent("UserCreated", dcb.NewTags("user_id", "user-3"), []byte(`{"name": "Charlie"}`)),
		dcb.NewInputEvent("UserUpdated", dcb.NewTags("user_id", "user-1"), []byte(`{"name": "Alice Smith"}`)),
		dcb.NewInputEvent("UserUpdated", dcb.NewTags("user_id", "user-2"), []byte(`{"name": "Bob Johnson"}`)),
	}

	// Append events
	err = store.Append(ctx, events, nil)
	if err != nil {
		log.Fatalf("Failed to append events: %v", err)
	}

	fmt.Println("=== Streaming Examples ===")

	// 1. Core EventStore - Read into memory
	fmt.Println("\n1. Core EventStore - Read into memory:")
	demonstrateCoreRead(ctx, store)

	// 2. EventStore - Channel-based streaming
	fmt.Println("\n2. EventStore - Channel-based streaming:")
	demonstrateChannelStreaming(ctx, store)

	// 3. EventStore - Channel-based projection
	fmt.Println("\n3. EventStore - Channel-based projection:")
	demonstrateChannelProjection(ctx, store)
}

// demonstrateCoreRead shows traditional Read into memory
func demonstrateCoreRead(ctx context.Context, store dcb.EventStore) {
	query := dcb.NewQuery(dcb.NewTags(), "UserCreated", "UserUpdated")

	// Read all events into memory
	events, err := store.Query(ctx, query, nil)
	if err != nil {
		log.Printf("Read failed: %v", err)
		return
	}

	fmt.Printf("   - Read(): Loaded %d events into memory\n", len(events))
	for i, event := range events {
		fmt.Printf("     Event %d: Position=%d, Type=%s\n", i+1, event.Position, event.Type)
	}
}

// demonstrateChannelStreaming shows channel-based event streaming
func demonstrateChannelStreaming(ctx context.Context, store dcb.EventStore) {
	// Check if store implements EventStore
	store, ok := store.(dcb.EventStore)
	if !ok {
		fmt.Println("   - Store does not implement EventStore interface")
		return
	}

	query := dcb.NewQuery(dcb.NewTags(), "UserCreated", "UserUpdated")

	// Stream events through channel
	eventChan, err := store.QueryStream(ctx, query, nil)
	if err != nil {
		log.Printf("ReadStream failed: %v", err)
		return
	}

	count := 0
	for event := range eventChan {
		fmt.Printf("   - ReadStream(): Event %d: Position=%d, Type=%s\n",
			count+1, event.Position, event.Type)
		count++
	}
	fmt.Printf("   - ReadStream(): Processed %d events using channels\n", count)
}

// demonstrateChannelProjection shows channel-based state projection
func demonstrateChannelProjection(ctx context.Context, store dcb.EventStore) {
	// Check if store implements EventStore
	store, ok := store.(dcb.EventStore)
	if !ok {
		fmt.Println("   - Store does not implement EventStore interface")
		return
	}

	// Define projectors for state projection
	projectors := []dcb.StateProjector{
		{
			ID:           "userCount",
			Query:        dcb.NewQuery(dcb.NewTags(), "UserCreated"),
			InitialState: 0,
			TransitionFn: func(state any, event dcb.Event) any {
				return state.(int) + 1
			},
		},
		{
			ID:           "updateCount",
			Query:        dcb.NewQuery(dcb.NewTags(), "UserUpdated"),
			InitialState: 0,
			TransitionFn: func(state any, event dcb.Event) any {
				return state.(int) + 1
			},
		},
	}

	// Project states (non-streaming)
	finalStates, _, err := store.Project(ctx, projectors, nil)
	if err != nil {
		log.Printf("Project failed: %v", err)
		return
	}

	fmt.Println("   - Project(): Final aggregated states:")
	for projectorID, state := range finalStates {
		fmt.Printf("     Projector %s: State=%v\n", projectorID, state)
	}
}
