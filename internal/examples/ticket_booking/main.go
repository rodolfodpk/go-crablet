// This example demonstrates advisory locking for ticket booking.
// Run with: go run internal/examples/ticket_booking/main.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/rodolfodpk/go-crablet/internal/examples/utils"
	"github.com/rodolfodpk/go-crablet/pkg/dcb"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Command types
type CreateConcertCommand struct {
	ConcertID      string
	Artist         string
	Venue          string
	TotalSeats     int
	PricePerTicket float64
}

type BookTicketsCommand struct {
	CustomerID string
	ConcertID  string
	Quantity   int
}

// Ticket booking state
type ConcertState struct {
	Artist         string
	Venue          string
	TotalSeats     int
	BookedSeats    int
	PricePerTicket float64
}

func main() {
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, "postgres://postgres:postgres@localhost:5432/dcb_app?sslmode=disable")
	if err != nil {
		log.Fatalf("failed to connect to db: %v", err)
	}
	store, err := dcb.NewEventStore(ctx, pool)
	if err != nil {
		log.Fatalf("failed to create event store: %v", err)
	}

	// Create a concert with limited seats
	concertID := fmt.Sprintf("concert_%d", time.Now().Unix())
	createConcertCmd := CreateConcertCommand{
		ConcertID:      concertID,
		Artist:         "The Event Sourcing Band",
		Venue:          "DCB Arena",
		TotalSeats:     20, // 20 seats available
		PricePerTicket: 50.0,
	}
	err = handleCreateConcert(ctx, store, createConcertCmd)
	if err != nil {
		log.Fatalf("Create concert failed: %v", err)
	}

	fmt.Println("=== Testing Concurrent Ticket Booking with Advisory Locks ===")
	fmt.Printf("Concert %s has %d seats available. Attempting to book tickets for 100 customers concurrently...\n",
		concertID, createConcertCmd.TotalSeats)

	// Simulate concurrent ticket booking attempts
	var wg sync.WaitGroup
	results := make(chan string, 100)

	// Try to book tickets for 100 customers concurrently (but only some should succeed)
	for i := 1; i <= 100; i++ {
		wg.Add(1)
		go func(customerID int) {
			defer wg.Done()

			// Each customer wants 2 tickets
			bookCmd := BookTicketsCommand{
				CustomerID: fmt.Sprintf("customer%d", customerID),
				ConcertID:  concertID,
				Quantity:   2, // Each customer wants 2 tickets
			}

			err := handleBookTicketsWithAdvisoryLock(ctx, store, bookCmd)
			if err != nil {
				results <- fmt.Sprintf("Customer %d: FAILED - %v", customerID, err)
			} else {
				results <- fmt.Sprintf("Customer %d: SUCCESS - booked %d tickets", customerID, bookCmd.Quantity)
			}
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(results)

	// Print results
	fmt.Println("\n=== Booking Results ===")
	for result := range results {
		fmt.Println(result)
	}

	// Show final state
	fmt.Println("\n=== Final Concert State ===")
	showConcertState(ctx, store, concertID)

	// Dump all events to show what was created
	fmt.Println("\n=== Events in Database ===")
	utils.DumpEvents(ctx, pool)
}

func handleCreateConcert(ctx context.Context, store dcb.EventStore, cmd CreateConcertCommand) error {
	// Command-specific projectors
	projectors := []dcb.StateProjector{
		{
			ID: "concertExists",
			Query: dcb.NewQuery(
				dcb.NewTags("concert_id", cmd.ConcertID),
				"ConcertCreated",
			),
			InitialState: false,
			TransitionFn: func(state any, event dcb.Event) any {
				return true // If we see a ConcertCreated event, concert exists
			},
		},
	}

	states, _, err := store.Project(ctx, projectors, nil)
	if err != nil {
		return fmt.Errorf("failed to check concert existence: %w", err)
	}

	// Command-specific business rule: concert must not already exist
	if states["concertExists"].(bool) {
		return fmt.Errorf("concert %s already exists", cmd.ConcertID)
	}

	// Create events for this command
	events := []dcb.InputEvent{
		dcb.NewInputEvent(
			"ConcertCreated",
			dcb.NewTags("concert_id", cmd.ConcertID),
			mustJSON(map[string]any{
				"Artist":         cmd.Artist,
				"Venue":          cmd.Venue,
				"TotalSeats":     cmd.TotalSeats,
				"PricePerTicket": cmd.PricePerTicket,
			}),
		),
	}

	// Append events atomically for this command
	err = store.Append(ctx, events, nil)
	if err != nil {
		return fmt.Errorf("failed to create concert: %w", err)
	}

	fmt.Printf("Created concert %s (%s at %s) with %d seats at $%.2f each\n",
		cmd.ConcertID, cmd.Artist, cmd.Venue, cmd.TotalSeats, cmd.PricePerTicket)
	return nil
}

func handleBookTicketsWithAdvisoryLock(ctx context.Context, store dcb.EventStore, cmd BookTicketsCommand) error {
	// Use advisory lock to prevent concurrent modifications to concert capacity
	// This ensures only one booking can check/update seat availability at a time
	// Note: Advisory locking is currently experimental and not fully implemented
	config := dcb.EventStoreConfig{
		LockTimeout: 5000, // 5 second timeout in milliseconds
	}

	storeWithLocks := dcb.NewEventStoreFromPoolWithConfig(store.GetPool(), config)

	// Command-specific projectors to check current state
	projectors := []dcb.StateProjector{
		{
			ID: "concertState",
			Query: dcb.NewQuery(
				dcb.NewTags("concert_id", cmd.ConcertID),
				"ConcertCreated",
			),
			InitialState: ConcertState{},
			TransitionFn: func(state any, event dcb.Event) any {
				concert := state.(ConcertState)
				if event.Type == "ConcertCreated" {
					var data map[string]any
					json.Unmarshal(event.Data, &data)
					concert.Artist = data["Artist"].(string)
					concert.Venue = data["Venue"].(string)
					concert.TotalSeats = int(data["TotalSeats"].(float64))
					concert.PricePerTicket = data["PricePerTicket"].(float64)
				} else if event.Type == "TicketsBooked" {
					var data map[string]any
					json.Unmarshal(event.Data, &data)
					concert.BookedSeats += int(data["quantity"].(float64))
				} else if event.Type == "BookingCancelled" {
					var data map[string]any
					json.Unmarshal(event.Data, &data)
					concert.BookedSeats -= int(data["quantity"].(float64))
				}
				return concert
			},
		},
		{
			ID: "customerBookings",
			Query: dcb.NewQuery(
				dcb.NewTags("concert_id", cmd.ConcertID, "customer_id", cmd.CustomerID),
				"TicketsBooked",
			),
			InitialState: 0,
			TransitionFn: func(state any, event dcb.Event) any {
				var data map[string]any
				json.Unmarshal(event.Data, &data)
				return state.(int) + int(data["quantity"].(float64))
			},
		},
	}

	states, _, err := storeWithLocks.Project(ctx, projectors, nil)
	if err != nil {
		return fmt.Errorf("failed to check booking state: %w", err)
	}

	// Business rules with advisory lock protection
	concert := states["concertState"].(ConcertState)
	customerBookings := states["customerBookings"].(int)

	// Check if customer has already booked tickets
	if customerBookings > 0 {
		return fmt.Errorf("customer %s has already booked %d tickets for concert %s",
			cmd.CustomerID, customerBookings, cmd.ConcertID)
	}

	// Check if enough seats are available
	availableSeats := concert.TotalSeats - concert.BookedSeats
	if cmd.Quantity > availableSeats {
		return fmt.Errorf("not enough seats available for concert %s (requested: %d, available: %d)",
			cmd.ConcertID, cmd.Quantity, availableSeats)
	}

	// Create booking event
	events := []dcb.InputEvent{
		dcb.NewInputEvent(
			"TicketsBooked",
			dcb.NewTags("concert_id", cmd.ConcertID, "customer_id", cmd.CustomerID),
			mustJSON(map[string]any{
				"quantity":   cmd.Quantity,
				"totalPrice": float64(cmd.Quantity) * concert.PricePerTicket,
				"bookedAt":   time.Now().UTC(),
			}),
		),
	}

	// Append events atomically (advisory lock ensures no concurrent modifications)
	err = storeWithLocks.Append(ctx, events, nil)
	if err != nil {
		return fmt.Errorf("failed to book tickets: %w", err)
	}

	fmt.Printf("Successfully booked %d tickets for customer %s in concert %s\n",
		cmd.Quantity, cmd.CustomerID, cmd.ConcertID)
	return nil
}

func showConcertState(ctx context.Context, store dcb.EventStore, concertID string) {
	projectors := []dcb.StateProjector{
		{
			ID: "concertState",
			Query: dcb.NewQuery(
				dcb.NewTags("concert_id", concertID),
				"ConcertCreated",
			),
			InitialState: ConcertState{},
			TransitionFn: func(state any, event dcb.Event) any {
				concert := state.(ConcertState)
				if event.Type == "ConcertCreated" {
					var data map[string]any
					json.Unmarshal(event.Data, &data)
					concert.Artist = data["Artist"].(string)
					concert.Venue = data["Venue"].(string)
					concert.TotalSeats = int(data["TotalSeats"].(float64))
					concert.PricePerTicket = data["PricePerTicket"].(float64)
				} else if event.Type == "TicketsBooked" {
					var data map[string]any
					json.Unmarshal(event.Data, &data)
					concert.BookedSeats += int(data["quantity"].(float64))
				} else if event.Type == "BookingCancelled" {
					var data map[string]any
					json.Unmarshal(event.Data, &data)
					concert.BookedSeats -= int(data["quantity"].(float64))
				}
				return concert
			},
		},
	}

	states, _, err := store.Project(ctx, projectors, nil)
	if err != nil {
		fmt.Printf("Error getting concert state: %v\n", err)
		return
	}

	concert := states["concertState"].(ConcertState)
	fmt.Printf("Concert: %s\n", concert.Artist)
	fmt.Printf("Venue: %s\n", concert.Venue)
	fmt.Printf("Total Seats: %d\n", concert.TotalSeats)
	fmt.Printf("Booked Seats: %d\n", concert.BookedSeats)
	fmt.Printf("Available Seats: %d\n", concert.TotalSeats-concert.BookedSeats)
	fmt.Printf("Price per Ticket: $%.2f\n", concert.PricePerTicket)
	fmt.Printf("Total Revenue: $%.2f\n", float64(concert.BookedSeats)*concert.PricePerTicket)
}

func mustJSON(v any) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal JSON: %v", err))
	}
	return data
}
