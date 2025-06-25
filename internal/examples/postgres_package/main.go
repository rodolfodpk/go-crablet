// This example demonstrates how the postgres package would work as an alternative
// to the core package, showing the separation of concerns.
package main

import (
	"context"
	"fmt"
	"log"

	"go-crablet/pkg/dcb"
	postgres "go-crablet/pkg/dcb/postgres"

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

	fmt.Println("=== PostgreSQL Package Example ===")

	// Method 1: Using the core package (current approach)
	fmt.Println("\n1. Core Package Approach:")
	demonstrateCorePackage(ctx, pool)

	// Method 2: Using the postgres package (proposed approach)
	fmt.Println("\n2. PostgreSQL Package Approach:")
	demonstratePostgresPackage(ctx, pool)
}

// demonstrateCorePackage shows usage of the current core package approach
func demonstrateCorePackage(ctx context.Context, pool *pgxpool.Pool) {
	fmt.Println("   Using core package (current):")

	// Create event store using core package
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

	fmt.Printf("   - Core package: Loaded %d events\n", len(sequencedEvents.Events))
	fmt.Println("   - Note: Core package includes PostgreSQL dependencies")
}

// demonstratePostgresPackage shows usage of the proposed postgres package approach
func demonstratePostgresPackage(ctx context.Context, pool *pgxpool.Pool) {
	fmt.Println("   Using postgres package (proposed):")

	// Create event store using postgres package
	store, err := postgres.NewEventStore(ctx, pool)
	if err != nil {
		log.Printf("Failed to create postgres event store: %v", err)
		return
	}

	// Create some test events
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

	// Read events
	query := dcb.NewQuerySimple(dcb.NewTags(), "UserCreated")
	sequencedEvents, err := store.Read(ctx, query, nil)
	if err != nil {
		log.Printf("Failed to read events: %v", err)
		return
	}

	fmt.Printf("   - Postgres package: Loaded %d events\n", len(sequencedEvents.Events))
	fmt.Println("   - Note: Postgres package separates PostgreSQL dependencies")

	// Demonstrate channel streaming
	if channelStore, ok := store.(dcb.ChannelEventStore); ok {
		fmt.Println("   - Channel streaming available")
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

// demonstrateDependencySeparation shows the key benefit of the postgres package
func demonstrateDependencySeparation() {
	fmt.Println("\n3. Dependency Separation Benefits:")
	fmt.Println("   - Core package (pkg/dcb): Only interfaces and types")
	fmt.Println("   - Postgres package (pkg/dcb/postgres): PostgreSQL-specific implementation")
	fmt.Println("   - SQLite package (pkg/dcb/sqlite): SQLite-specific implementation (future)")
	fmt.Println("   - Consumers only import what they need")
	fmt.Println("   - No unused dependencies pulled in")
}
