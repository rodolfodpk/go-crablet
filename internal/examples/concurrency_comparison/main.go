// This example demonstrates and compares DCB concurrency control vs advisory locks for ticket booking.
// Run with: go run internal/examples/concurrency_comparison/main.go
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

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
	// Parse command-line arguments
	var numUsers = flag.Int("users", 100, "Number of concurrent users to simulate (default: 100)")
	var numSeats = flag.Int("seats", 20, "Number of seats available in the concert (default: 20)")
	var ticketsPerUser = flag.Int("tickets", 2, "Number of tickets each user wants to book (default: 2)")
	var useAdvisoryLocks = flag.Bool("advisory-locks", false, "Use advisory locks instead of DCB concurrency control (default: false)")
	flag.Parse()

	// Show configuration
	fmt.Printf("=== Concurrency Control Performance Comparison Demo ===\n")
	fmt.Printf("Configuration: %d users, %d seats, %d tickets per user\n", *numUsers, *numSeats, *ticketsPerUser)
	fmt.Printf("Expected successful bookings: %d (if seats >= users * tickets)\n", *numSeats / *ticketsPerUser)
	fmt.Printf("Expected failed bookings: %d\n\n", *numUsers-(*numSeats / *ticketsPerUser))

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, "postgres://crablet:crablet@localhost:5432/crablet?sslmode=disable")
	if err != nil {
		log.Fatalf("failed to connect to db: %v", err)
	}
	store, err := dcb.NewEventStore(ctx, pool)
	if err != nil {
		log.Fatalf("failed to create event store: %v", err)
	}

	// Determine test mode based on command line flags
	if *useAdvisoryLocks {
		runSingleApproach(ctx, store, *numUsers, *numSeats, *ticketsPerUser, *useAdvisoryLocks)
	} else {
		runPerformanceComparison(ctx, store, pool, *numUsers, *numSeats, *ticketsPerUser)
	}
}

// runSingleApproach runs a single concurrency control approach
func runSingleApproach(ctx context.Context, store dcb.EventStore, numUsers, numSeats, ticketsPerUser int, useAdvisoryLocks bool) {
	fmt.Printf("=== Testing %s ===\n", getConcurrencyMethod(useAdvisoryLocks))
	results := runConcurrencyTest(ctx, store, numUsers, numSeats, ticketsPerUser, useAdvisoryLocks)

	fmt.Printf("\n=== Results ===\n")
	fmt.Printf("Total Time: %v\n", results.totalTime)
	fmt.Printf("Average Time per Booking: %v\n", results.avgTime)
	fmt.Printf("Success Rate: %.1f%%\n", results.successRate)
	fmt.Printf("Throughput: %.0f ops/s\n", results.throughput)
}

// runPerformanceComparison runs both approaches and compares their performance
func runPerformanceComparison(ctx context.Context, store dcb.EventStore, pool *pgxpool.Pool, numUsers, numSeats, ticketsPerUser int) {
	// Test DCB Concurrency Control
	fmt.Println("=== Testing DCB Concurrency Control ===")
	dcbResults := runConcurrencyTest(ctx, store, numUsers, numSeats, ticketsPerUser, false)

	// Clear the database for the next test
	fmt.Println("\nClearing database for next test...")
	clearDatabase(ctx, pool)

	// Test Advisory Locks
	fmt.Println("\n=== Testing PostgreSQL Advisory Locks ===")
	advisoryResults := runConcurrencyTest(ctx, store, numUsers, numSeats, ticketsPerUser, true)

	// Compare results
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("=== PERFORMANCE COMPARISON ===")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("%-30s | %-15s | %-15s | %-15s | %-15s\n", "Method", "Total Time", "Avg Time/Booking", "Success Rate", "Throughput")
	fmt.Println(strings.Repeat("-", 80))
	fmt.Printf("%-30s | %-15s | %-15s | %-15s | %-15s\n",
		"DCB Concurrency Control",
		dcbResults.totalTime,
		dcbResults.avgTime,
		fmt.Sprintf("%.1f%%", dcbResults.successRate),
		fmt.Sprintf("%.0f ops/s", dcbResults.throughput))
	fmt.Printf("%-30s | %-15s | %-15s | %-15s | %-15s\n",
		"PostgreSQL Advisory Locks",
		advisoryResults.totalTime,
		advisoryResults.avgTime,
		fmt.Sprintf("%.1f%%", advisoryResults.successRate),
		fmt.Sprintf("%.0f ops/s", advisoryResults.throughput))
	fmt.Println(strings.Repeat("-", 80))

	// Performance analysis
	fmt.Println("\n=== PERFORMANCE ANALYSIS ===")
	if dcbResults.totalTime < advisoryResults.totalTime {
		improvement := float64(advisoryResults.totalTime) / float64(dcbResults.totalTime)
		fmt.Printf("✅ DCB Concurrency Control is %.1fx FASTER than Advisory Locks\n", improvement)
	} else {
		improvement := float64(dcbResults.totalTime) / float64(advisoryResults.totalTime)
		fmt.Printf("✅ Advisory Locks are %.1fx FASTER than DCB Concurrency Control\n", improvement)
	}

	if dcbResults.throughput > advisoryResults.throughput {
		improvement := dcbResults.throughput / advisoryResults.throughput
		fmt.Printf("✅ DCB Concurrency Control has %.1fx HIGHER throughput\n", improvement)
	} else {
		improvement := advisoryResults.throughput / dcbResults.throughput
		fmt.Printf("✅ Advisory Locks have %.1fx HIGHER throughput\n", improvement)
	}
}

type TestResults struct {
	totalTime    time.Duration
	avgTime      time.Duration
	successRate  float64
	throughput   float64
	successCount int
	failureCount int
}

func runConcurrencyTest(ctx context.Context, store dcb.EventStore, numUsers, numSeats, ticketsPerUser int, useAdvisoryLocks bool) TestResults {
	// Create a concert with limited seats
	concertID := fmt.Sprintf("concert_%d_%s", time.Now().Unix(), getConcurrencyMethod(useAdvisoryLocks))
	createConcertCmd := CreateConcertCommand{
		ConcertID:      concertID,
		Artist:         "The Event Sourcing Band",
		Venue:          "DCB Arena",
		TotalSeats:     numSeats,
		PricePerTicket: 50.0,
	}
	err := handleCreateConcert(ctx, store, createConcertCmd)
	if err != nil {
		log.Fatalf("Create concert failed: %v", err)
	}

	fmt.Printf("Concert %s has %d seats available. Attempting to book tickets for %d customers concurrently...\n",
		concertID, createConcertCmd.TotalSeats, numUsers)

	// Simulate concurrent ticket booking attempts with timing
	var wg sync.WaitGroup
	results := make(chan string, numUsers)
	startTime := time.Now()

	// Try to book tickets for customers concurrently
	for i := 1; i <= numUsers; i++ {
		wg.Add(1)
		go func(customerID int) {
			defer wg.Done()

			bookCmd := BookTicketsCommand{
				CustomerID: fmt.Sprintf("customer%d", customerID),
				ConcertID:  concertID,
				Quantity:   ticketsPerUser,
			}

			var err error
			if useAdvisoryLocks {
				err = handleBookTicketsWithAdvisoryLocks(ctx, store, bookCmd)
			} else {
				err = handleBookTicketsWithDCBControl(ctx, store, bookCmd)
			}

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

	// Calculate timing and statistics
	totalTime := time.Since(startTime)
	successCount := 0
	failureCount := 0

	// Count results and show errors for debugging
	fmt.Println("\n=== Detailed Results ===")
	for result := range results {
		fmt.Println(result)
		if strings.Contains(result, "SUCCESS") {
			successCount++
		} else {
			failureCount++
		}
	}

	avgTime := totalTime / time.Duration(numUsers)
	successRate := float64(successCount) / float64(numUsers) * 100
	throughput := float64(numUsers) / totalTime.Seconds()

	fmt.Printf("Total Time: %v\n", totalTime)
	fmt.Printf("Average Time per Booking: %v\n", avgTime)
	fmt.Printf("Successful Bookings: %d\n", successCount)
	fmt.Printf("Failed Bookings: %d\n", failureCount)
	fmt.Printf("Success Rate: %.2f%%\n", successRate)
	fmt.Printf("Throughput: %.0f ops/s\n", throughput)

	return TestResults{
		totalTime:    totalTime,
		avgTime:      avgTime,
		successRate:  successRate,
		throughput:   throughput,
		successCount: successCount,
		failureCount: failureCount,
	}
}

func clearDatabase(ctx context.Context, pool *pgxpool.Pool) {
	// Clear all events for a clean test
	_, err := pool.Exec(ctx, "DELETE FROM events")
	if err != nil {
		log.Printf("Warning: failed to clear database: %v", err)
	}
}

func getConcurrencyMethod(useAdvisoryLocks bool) string {
	if useAdvisoryLocks {
		return "PostgreSQL Advisory Locks"
	}
	return "DCB Concurrency Control"
}

func handleCreateConcert(ctx context.Context, store dcb.EventStore, cmd CreateConcertCommand) error {
	// Command-specific projectors
	projectors := []dcb.StateProjector{
		{
			ID: "concertExists",
			Query: dcb.NewQuery(
				dcb.NewTags("concert_id", cmd.ConcertID),
				"ConcertDefined",
			),
			InitialState: false,
			TransitionFn: func(state any, event dcb.Event) any {
				return true // If we see a ConcertDefined event, concert exists
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
			"ConcertDefined",
			dcb.NewTags("concert_id", cmd.ConcertID),
			mustJSON(map[string]any{
				"Artist":         cmd.Artist,
				"Venue":          cmd.Venue,
				"TotalSeats":     cmd.TotalSeats,
				"PricePerTicket": cmd.PricePerTicket,
			}),
		),
	}

	// Create AppendCondition to ensure concert doesn't exist since our projection
	// This prevents race conditions where multiple concert creations could succeed
	item := dcb.NewQueryItem([]string{"ConcertDefined"}, []dcb.Tag{dcb.NewTag("concert_id", cmd.ConcertID)})
	query := dcb.NewQueryFromItems(item)
	appendCondition := dcb.NewAppendCondition(query)

	// Append events atomically with DCB concurrency control
	err = store.AppendIf(ctx, events, appendCondition)
	if err != nil {
		return fmt.Errorf("failed to create concert: %w", err)
	}

	fmt.Printf("Created concert %s (%s at %s) with %d seats at $%.2f each\n",
		cmd.ConcertID, cmd.Artist, cmd.Venue, cmd.TotalSeats, cmd.PricePerTicket)
	return nil
}

func handleBookTicketsWithDCBControl(ctx context.Context, store dcb.EventStore, cmd BookTicketsCommand) error {
	// Use DCB concurrency control to prevent concurrent modifications to concert capacity
	// This ensures only one booking can check/update seat availability at a time
	// Note: This demonstrates DCB concurrency control, not advisory locks

	// Command-specific projectors to check current state
	projectors := []dcb.StateProjector{
		{
			ID: "concertState",
			Query: dcb.NewQuery(
				dcb.NewTags("concert_id", cmd.ConcertID),
				"ConcertDefined",
			),
			InitialState: ConcertState{},
			TransitionFn: func(state any, event dcb.Event) any {
				concert := state.(ConcertState)
				if event.Type == "ConcertDefined" {
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

	states, _, err := store.Project(ctx, projectors, nil)
	if err != nil {
		return fmt.Errorf("failed to check booking state: %w", err)
	}

	// Business rules with DCB concurrency control protection
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

	// Create AppendCondition to ensure concert state hasn't changed since our projection
	// This prevents race conditions where multiple bookings could overbook the concert
	item := dcb.NewQueryItem([]string{"ConcertDefined"}, []dcb.Tag{dcb.NewTag("concert_id", cmd.ConcertID)})
	query := dcb.NewQueryFromItems(item)
	appendCondition := dcb.NewAppendCondition(query)

	// Append events atomically with DCB concurrency control
	err = store.AppendIf(ctx, events, appendCondition)
	if err != nil {
		return fmt.Errorf("failed to book tickets: %w", err)
	}

	fmt.Printf("Successfully booked %d tickets for customer %s in concert %s\n",
		cmd.Quantity, cmd.CustomerID, cmd.ConcertID)
	return nil
}

func handleBookTicketsWithAdvisoryLocks(ctx context.Context, store dcb.EventStore, cmd BookTicketsCommand) error {
	// Use PostgreSQL advisory locks to prevent concurrent modifications to concert capacity
	// This ensures only one booking can check/update seat availability at a time
	// Note: This demonstrates advisory locks, which are experimental and optional

	// Command-specific projectors to check current state
	projectors := []dcb.StateProjector{
		{
			ID: "concertState",
			Query: dcb.NewQuery(
				dcb.NewTags("concert_id", cmd.ConcertID),
				"ConcertDefined",
			),
			InitialState: ConcertState{},
			TransitionFn: func(state any, event dcb.Event) any {
				concert := state.(ConcertState)
				if event.Type == "ConcertDefined" {
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

	states, _, err := store.Project(ctx, projectors, nil)
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

	// Create booking event with advisory lock tag
	events := []dcb.InputEvent{
		dcb.NewInputEvent(
			"TicketsBooked",
			dcb.NewTags("concert_id", cmd.ConcertID, "customer_id", cmd.CustomerID, "lock:concert", cmd.ConcertID),
			mustJSON(map[string]any{
				"quantity":   cmd.Quantity,
				"totalPrice": float64(cmd.Quantity) * concert.PricePerTicket,
				"bookedAt":   time.Now().UTC(),
			}),
		),
	}

	// Create AppendCondition to ensure concert state hasn't changed since our projection
	// This prevents race conditions where multiple bookings could overbook the concert
	// Note: This is in addition to advisory locks for comprehensive concurrency control
	item := dcb.NewQueryItem([]string{"ConcertDefined"}, []dcb.Tag{dcb.NewTag("concert_id", cmd.ConcertID)})
	query := dcb.NewQueryFromItems(item)
	appendCondition := dcb.NewAppendCondition(query)

	// Append events atomically using core API with both advisory locks and DCB control
	err = store.AppendIf(ctx, events, appendCondition)
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
				"ConcertDefined",
			),
			InitialState: ConcertState{},
			TransitionFn: func(state any, event dcb.Event) any {
				concert := state.(ConcertState)
				if event.Type == "ConcertDefined" {
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
