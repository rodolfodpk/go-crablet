// This example demonstrates channel-based streaming as an alternative to iterator-based streaming
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
	pool, err := pgxpool.New(ctx, "postgres://postgres:password@localhost:5432/events?sslmode=disable")
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
		dcb.NewInputEventUnsafe("UserCreated", dcb.NewTags("user_id", "user-1"), []byte(`{"name": "Alice"}`)),
		dcb.NewInputEventUnsafe("UserCreated", dcb.NewTags("user_id", "user-2"), []byte(`{"name": "Bob"}`)),
		dcb.NewInputEventUnsafe("UserCreated", dcb.NewTags("user_id", "user-3"), []byte(`{"name": "Charlie"}`)),
	}

	// Append events
	_, err = store.Append(ctx, events, nil)
	if err != nil {
		log.Fatalf("Failed to append events: %v", err)
	}

	fmt.Println("=== Channel-Based Streaming Example ===")

	// Create a query
	query := dcb.NewQuerySimple(dcb.NewTags(), "UserCreated")

	// Method 1: Using the channel-based interface (if implemented)
	// This would be the equivalent of your pgx channel approach
	fmt.Println("\n1. Channel-based streaming (conceptual):")
	fmt.Println("   - More Go-idiomatic with channels")
	fmt.Println("   - Buffered processing")
	fmt.Println("   - Context cancellation support")
	fmt.Println("   - Error handling via Close()")

	// Method 2: Current iterator-based approach
	fmt.Println("\n2. Current iterator-based streaming:")
	iterator, err := store.ReadStream(ctx, query, nil)
	if err != nil {
		log.Fatalf("Failed to create stream: %v", err)
	}
	defer iterator.Close()

	count := 0
	for iterator.Next() {
		event := iterator.Event()
		fmt.Printf("   Event %d: ID=%s, Type=%s, Position=%d\n",
			count+1, event.ID, event.Type, event.Position)
		count++
	}

	if err := iterator.Err(); err != nil {
		log.Printf("Iterator error: %v", err)
	}

	fmt.Printf("\nProcessed %d events using iterator\n", count)

	// Method 3: Traditional Read (loads all into memory)
	fmt.Println("\n3. Traditional Read (loads all into memory):")
	sequencedEvents, err := store.Read(ctx, query, nil)
	if err != nil {
		log.Fatalf("Failed to read events: %v", err)
	}

	fmt.Printf("   Loaded %d events into memory\n", len(sequencedEvents.Events))
	for i, event := range sequencedEvents.Events {
		fmt.Printf("   Event %d: ID=%s, Type=%s, Position=%d\n",
			i+1, event.ID, event.Type, event.Position)
	}

	fmt.Println("\n=== Comparison Summary ===")
	fmt.Println("Channel-based streaming:")
	fmt.Println("  ✅ Go-idiomatic with channels")
	fmt.Println("  ✅ Buffered processing")
	fmt.Println("  ✅ Context cancellation")
	fmt.Println("  ❌ Potential memory overhead from buffering")
	fmt.Println("  ❌ More complex error handling")

	fmt.Println("\nIterator-based streaming (current):")
	fmt.Println("  ✅ Memory efficient")
	fmt.Println("  ✅ Simple error handling")
	fmt.Println("  ✅ Cursor-based for large datasets")
	fmt.Println("  ✅ Automatic resource cleanup")
	fmt.Println("  ❌ Less Go-idiomatic")

	fmt.Println("\nTraditional Read:")
	fmt.Println("  ✅ Simple to use")
	fmt.Println("  ✅ All data available immediately")
	fmt.Println("  ❌ Loads everything into memory")
	fmt.Println("  ❌ Not suitable for large datasets")
}
