// This example demonstrates the Extension Interface pattern with ChannelEventStore
package main

import (
	"context"
	"fmt"
	"log"

	"go-crablet/pkg/dcb"

	"github.com/jackc/pgx/v5/pgxpool"
)

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

	// Create some test events
	events := []dcb.InputEvent{
		func() dcb.InputEvent {
			event := dcb.NewInputEvent("UserCreated", dcb.NewTags("user_id", "user-1"), []byte(`{"name": "Alice"}`))
			return event
		}(),
		func() dcb.InputEvent {
			event := dcb.NewInputEvent("UserCreated", dcb.NewTags("user_id", "user-2"), []byte(`{"name": "Bob"}`))
			return event
		}(),
		func() dcb.InputEvent {
			event := dcb.NewInputEvent("UserCreated", dcb.NewTags("user_id", "user-3"), []byte(`{"name": "Charlie"}`))
			return event
		}(),
	}

	// Append events
	err = store.Append(ctx, events, nil)
	if err != nil {
		log.Fatalf("Failed to append events: %v", err)
	}

	fmt.Println("=== Extension Interface Example ===")

	// Method 1: Using the core EventStore interface
	fmt.Println("\n1. Core EventStore Interface:")
	demonstrateCoreInterface(ctx, store)

	// Method 2: Using the ChannelEventStore extension interface
	fmt.Println("\n2. ChannelEventStore Extension Interface:")
	demonstrateChannelInterface(ctx, store)
}

// demonstrateCoreInterface shows usage of the core EventStore interface
func demonstrateCoreInterface(ctx context.Context, store dcb.EventStore) {
	fmt.Println("   Using core EventStore methods:")

	// Traditional Read
	query := dcb.NewQuerySimple(dcb.NewTags(), "UserCreated")
	sequencedEvents, err := store.Read(ctx, query, nil)
	if err != nil {
		log.Printf("Read failed: %v", err)
		return
	}
	fmt.Printf("   - Read(): Loaded %d events into memory\n", len(sequencedEvents.Events))

	// Note: ReadStream method has been removed from the EventStore interface
	// Streaming is now handled through the ChannelEventStore extension interface
	fmt.Println("   - Note: ReadStream() method has been removed from the core interface")
	fmt.Println("   - Streaming is now handled through ChannelEventStore extension")
}

// demonstrateChannelInterface shows usage of the ChannelEventStore extension interface
func demonstrateChannelInterface(ctx context.Context, store dcb.EventStore) {
	fmt.Println("   Using ChannelEventStore extension methods:")

	// Check if store implements ChannelEventStore
	channelStore, ok := store.(dcb.ChannelEventStore)
	if !ok {
		fmt.Println("   - Store does not implement ChannelEventStore interface")
		return
	}

	query := dcb.NewQuerySimple(dcb.NewTags(), "UserCreated")

	// Channel-based ReadStreamChannel
	eventChan, err := channelStore.ReadStreamChannel(ctx, query)
	if err != nil {
		log.Printf("ReadStreamChannel failed: %v", err)
		return
	}

	count := 0
	for event := range eventChan {
		fmt.Printf("   - ReadStreamChannel(): Event %d: Position=%d, Type=%s\n",
			count+1, event.Position, event.Type)
		count++
	}
	fmt.Printf("   - ReadStreamChannel(): Processed %d events using channels\n", count)

	// Note: NewEventStream method has been removed from the ChannelEventStore interface
	fmt.Println("   - Note: NewEventStream() method has been removed from the interface")
	fmt.Println("   - ChannelEventStore extends EventStore with channel methods")

	// This works for both EventStore and ChannelEventStore
	if channelStore, ok := store.(dcb.ChannelEventStore); ok {
		query := dcb.NewQuerySimple(dcb.NewTags(), "UserCreated")
		_, err := channelStore.ReadStreamChannel(ctx, query)
		if err != nil {
			log.Printf("Extension interface failed: %v", err)
		} else {
			fmt.Println("   ✅ Extension interface works")
		}
	} else {
		fmt.Println("   - Store does not implement ChannelEventStore")
	}
}

// demonstrateInterfaceCompatibility shows how both interfaces work together
func demonstrateInterfaceCompatibility(ctx context.Context, store dcb.EventStore) {
	fmt.Println("\n3. Interface Compatibility:")
	fmt.Println("   - Core EventStore methods work on all implementations")
	fmt.Println("   - ChannelEventStore extends EventStore with channel methods")
	fmt.Println("   - Type assertion allows access to extension methods")
	fmt.Println("   - Graceful fallback when extension not available")

	// This works for both EventStore and ChannelEventStore
	query := dcb.NewQuerySimple(dcb.NewTags(), "UserCreated")
	_, err := store.Read(ctx, query, nil)
	if err != nil {
		log.Printf("Core interface failed: %v", err)
	} else {
		fmt.Println("   ✅ Core interface works")
	}

	// Extension methods require type assertion
	if channelStore, ok := store.(dcb.ChannelEventStore); ok {
		_, err := channelStore.ReadStreamChannel(ctx, query)
		if err != nil {
			log.Printf("Extension interface failed: %v", err)
		} else {
			fmt.Println("   ✅ Extension interface works")
		}
	} else {
		fmt.Println("   ⚠️  Extension interface not available")
	}
}
