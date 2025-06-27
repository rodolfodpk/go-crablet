// This example demonstrates how the consolidated dcb package works,
// showing the unified approach with PostgreSQL implementation.
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

	fmt.Println("=== Consolidated DCB Package Example ===")

	// Demonstrate the consolidated package approach
	fmt.Println("\n1. Consolidated Package Approach:")
	demonstrateConsolidatedPackage(ctx, pool)

	// Demonstrate channel streaming
	fmt.Println("\n2. Channel Streaming:")
	demonstrateChannelStreaming(ctx, pool)
}

// demonstrateConsolidatedPackage shows usage of the consolidated dcb package
func demonstrateConsolidatedPackage(ctx context.Context, pool *pgxpool.Pool) {
	fmt.Println("   Using consolidated dcb package:")

	// Create event store using consolidated package
	store, err := dcb.NewEventStore(ctx, pool)
	if err != nil {
		log.Printf("Failed to create event store: %v", err)
		return
	}

	// Create some test events
	events := []dcb.InputEvent{
		dcb.NewInputEvent("UserCreated", dcb.NewTags("user_id", "user-1"), []byte(`{"name": "Alice"}`)),
		dcb.NewInputEvent("UserCreated", dcb.NewTags("user_id", "user-2"), []byte(`{"name": "Bob"}`)),
	}

	// Append events
	err = store.Append(ctx, events, nil)
	if err != nil {
		log.Printf("Failed to append events: %v", err)
		return
	}

	// Read events
	query := dcb.NewQuerySimple(dcb.NewTags(), "UserCreated")
	sequencedEvents, err := store.Read(ctx, query, nil)
	if err != nil {
		log.Printf("Failed to read events: %v", err)
		return
	}

	fmt.Printf("   - Consolidated package: Loaded %d events\n", len(sequencedEvents.Events))
	fmt.Println("   - Note: Consolidated package includes PostgreSQL implementation")
}

// demonstrateChannelStreaming shows channel streaming capabilities
func demonstrateChannelStreaming(ctx context.Context, pool *pgxpool.Pool) {
	fmt.Println("   Using channel streaming:")

	// Create event store using consolidated package
	store, err := dcb.NewEventStore(ctx, pool)
	if err != nil {
		log.Printf("Failed to create event store: %v", err)
		return
	}

	// Create some test events for streaming
	events := []dcb.InputEvent{
		dcb.NewInputEvent("UserCreated", dcb.NewTags("user_id", "user-3"), []byte(`{"name": "Charlie"}`)),
		dcb.NewInputEvent("UserCreated", dcb.NewTags("user_id", "user-4"), []byte(`{"name": "Diana"}`)),
	}

	// Append events
	err = store.Append(ctx, events, nil)
	if err != nil {
		log.Printf("Failed to append events: %v", err)
		return
	}

	// Demonstrate channel streaming
	if channelStore, ok := store.(dcb.ChannelEventStore); ok {
		fmt.Println("   - Channel streaming available")
		query := dcb.NewQuerySimple(dcb.NewTags(), "UserCreated")
		eventChan, err := channelStore.ReadStreamChannel(ctx, query)
		if err != nil {
			log.Printf("Failed to create event stream: %v", err)
			return
		}

		count := 0
		for event := range eventChan {
			fmt.Printf("   - Streamed event %d: Position=%d, Type=%s\n",
				count+1, event.Position, event.Type)
			count++
		}
		fmt.Printf("   - Streamed %d events using channels\n", count)
	} else {
		fmt.Println("   - Channel streaming not available")
	}
}

// demonstrateConsolidationBenefits shows the benefits of the consolidated approach
func demonstrateConsolidationBenefits() {
	fmt.Println("\n3. Consolidation Benefits:")
	fmt.Println("   - Single package (pkg/dcb): Interfaces, types, and PostgreSQL implementation")
	fmt.Println("   - Simplified import structure")
	fmt.Println("   - All functionality in one place")
	fmt.Println("   - Easier maintenance and testing")
	fmt.Println("   - No package fragmentation")
}
