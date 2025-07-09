package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// Example of how to use the new advisory lock function
func main() {
	// Connect to database
	db, err := sql.Open("postgres", "postgres://postgres:postgres@localhost:5432/dcb_app?sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	ctx := context.Background()

	// Test 1: Single event with lock tag
	fmt.Println("=== Test 1: Single event with lock tag ===")
	err = appendWithAdvisoryLocks(ctx, db, []Event{
		{
			Type: "UserCreated",
			Tags: []string{"user:123", "lock:user:123", "tenant:acme"},
			Data: map[string]interface{}{"name": "John Doe"},
		},
	}, nil)
	if err != nil {
		log.Printf("Error: %v", err)
	}

	// Test 2: Multiple events with same lock tag (should serialize)
	fmt.Println("\n=== Test 2: Multiple events with same lock tag ===")
	err = appendWithAdvisoryLocks(ctx, db, []Event{
		{
			Type: "OrderCreated",
			Tags: []string{"order:456", "lock:order:456", "customer:789"},
			Data: map[string]interface{}{"total": 100},
		},
		{
			Type: "ItemAdded",
			Tags: []string{"order:456", "lock:order:456", "item:123"},
			Data: map[string]interface{}{"item_id": "123", "quantity": 2},
		},
	}, nil)
	if err != nil {
		log.Printf("Error: %v", err)
	}

	// Test 3: Event with multiple lock tags
	fmt.Println("\n=== Test 3: Event with multiple lock tags ===")
	err = appendWithAdvisoryLocks(ctx, db, []Event{
		{
			Type: "TransactionProcessed",
			Tags: []string{"transaction:999", "lock:account:111", "lock:account:222", "amount:500"},
			Data: map[string]interface{}{
				"from_account": "111",
				"to_account":   "222",
				"amount":       500,
			},
		},
	}, nil)
	if err != nil {
		log.Printf("Error: %v", err)
	}

	// Test 4: Event with condition and lock tag
	fmt.Println("\n=== Test 4: Event with condition and lock tag ===")
	condition := map[string]interface{}{
		"fail_if_events_match": map[string]interface{}{
			"items": []map[string]interface{}{
				{
					"event_types": []string{"UserCreated"},
					"tags":        []string{"user:123"},
				},
			},
		},
	}
	err = appendWithAdvisoryLocks(ctx, db, []Event{
		{
			Type: "UserUpdated",
			Tags: []string{"user:123", "lock:user:123", "tenant:acme"},
			Data: map[string]interface{}{"name": "Jane Doe"},
		},
	}, condition)
	if err != nil {
		log.Printf("Error: %v", err)
	}

	// Test 5: Concurrent simulation (in real app, this would be different goroutines)
	fmt.Println("\n=== Test 5: Simulating concurrent access ===")
	fmt.Println("In a real application, these would be concurrent goroutines:")
	fmt.Println("- Goroutine 1: appendWithAdvisoryLocks(..., lock:user:123, ...)")
	fmt.Println("- Goroutine 2: appendWithAdvisoryLocks(..., lock:user:123, ...)")
	fmt.Println("The second would wait for the first to complete due to advisory lock")

	// Show stored events
	fmt.Println("\n=== Stored Events ===")
	showStoredEvents(ctx, db)
}

type Event struct {
	Type string                 `json:"type"`
	Tags []string               `json:"tags"`
	Data map[string]interface{} `json:"data"`
}

func appendWithAdvisoryLocks(ctx context.Context, db *sql.DB, events []Event, condition map[string]interface{}) error {
	// Prepare data for the function
	types := make([]string, len(events))
	tags := make([]string, len(events))
	data := make([][]byte, len(events))

	for i, event := range events {
		types[i] = event.Type
		tags[i] = encodeTagsArray(event.Tags)
		dataBytes, err := json.Marshal(event.Data)
		if err != nil {
			return fmt.Errorf("failed to marshal event data: %w", err)
		}
		data[i] = dataBytes
	}

	// Convert condition to JSON
	var conditionJSON []byte
	var err error
	if condition != nil {
		conditionJSON, err = json.Marshal(condition)
		if err != nil {
			return fmt.Errorf("failed to marshal condition: %w", err)
		}
	}

	// Call the advisory lock function
	_, err = db.ExecContext(ctx, `
		SELECT append_events_with_advisory_locks($1, $2, $3, $4)
	`, types, tags, data, conditionJSON)

	if err != nil {
		return fmt.Errorf("failed to append events with advisory locks: %w", err)
	}

	fmt.Printf("Successfully appended %d events with advisory locks\n", len(events))
	return nil
}

func encodeTagsArray(tags []string) string {
	// Convert to PostgreSQL array literal format
	result := "{"
	for i, tag := range tags {
		if i > 0 {
			result += ","
		}
		result += "\"" + tag + "\""
	}
	result += "}"
	return result
}

func showStoredEvents(ctx context.Context, db *sql.DB) {
	rows, err := db.QueryContext(ctx, `
		SELECT type, tags, data, position, transaction_id 
		FROM events 
		ORDER BY position DESC 
		LIMIT 10
	`)
	if err != nil {
		log.Printf("Error querying events: %v", err)
		return
	}
	defer rows.Close()

	fmt.Printf("%-20s %-30s %-50s %-10s %-20s\n", "Type", "Tags", "Data", "Position", "Transaction ID")
	fmt.Println(string(make([]byte, 130, 130)))
	for rows.Next() {
		var eventType string
		var tags []string
		var data []byte
		var position int64
		var transactionID string

		err := rows.Scan(&eventType, &tags, &data, &position, &transactionID)
		if err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}

		// Truncate long values for display
		tagsStr := fmt.Sprintf("%v", tags)
		if len(tagsStr) > 28 {
			tagsStr = tagsStr[:25] + "..."
		}

		dataStr := string(data)
		if len(dataStr) > 48 {
			dataStr = dataStr[:45] + "..."
		}

		fmt.Printf("%-20s %-30s %-50s %-10d %-20s\n",
			eventType, tagsStr, dataStr, position, transactionID)
	}
}
