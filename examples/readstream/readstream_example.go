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
	ctx := context.Background()

	// Connect to PostgreSQL
	pool, err := pgxpool.New(ctx, "postgres://postgres:postgres@localhost:5432/crablet?sslmode=disable")
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	// Create event store
	store, err := dcb.NewEventStore(ctx, pool)
	if err != nil {
		log.Fatalf("Failed to create event store: %v", err)
	}

	// Define projectors
	accountProjector := dcb.StateProjector{
		Query: dcb.Query{
			Items: []dcb.QueryItem{
				{
					EventTypes: []string{"AccountCreated", "AccountUpdated"},
					Tags:       []dcb.Tag{{Key: "account_id", Value: "acc123"}},
				},
			},
		},
		InitialState: &AccountState{ID: "acc123", Balance: 0},
		TransitionFn: func(state any, event dcb.Event) any {
			account := state.(*AccountState)
			switch event.Type {
			case "AccountCreated":
				var data AccountCreatedData
				json.Unmarshal(event.Data, &data)
				account.Balance = data.InitialBalance
			case "AccountUpdated":
				var data AccountUpdatedData
				json.Unmarshal(event.Data, &data)
				account.Balance = data.NewBalance
			}
			return account
		},
	}

	transactionProjector := dcb.StateProjector{
		Query: dcb.Query{
			Items: []dcb.QueryItem{
				{
					EventTypes: []string{"TransactionCompleted"},
					Tags:       []dcb.Tag{{Key: "account_id", Value: "acc123"}},
				},
			},
		},
		InitialState: &TransactionState{Count: 0, TotalAmount: 0},
		TransitionFn: func(state any, event dcb.Event) any {
			transactions := state.(*TransactionState)
			if event.Type == "TransactionCompleted" {
				var data TransactionCompletedData
				json.Unmarshal(event.Data, &data)
				transactions.Count++
				transactions.TotalAmount += data.Amount
			}
			return transactions
		},
	}

	// Create batch projectors
	projectors := []dcb.BatchProjector{
		{ID: "account", StateProjector: accountProjector},
		{ID: "transactions", StateProjector: transactionProjector},
	}

	// Create query for events
	query := dcb.Query{
		Items: []dcb.QueryItem{
			{
				EventTypes: []string{"AccountCreated", "AccountUpdated", "TransactionCompleted"},
				Tags:       []dcb.Tag{{Key: "account_id", Value: "acc123"}},
			},
		},
	}

	// Create some test events
	events := []dcb.InputEvent{
		{
			Type: "AccountCreated",
			Tags: []dcb.Tag{{Key: "account_id", Value: "acc123"}},
			Data: mustMarshal(AccountCreatedData{InitialBalance: 1000}),
		},
		{
			Type: "TransactionCompleted",
			Tags: []dcb.Tag{{Key: "account_id", Value: "acc123"}},
			Data: mustMarshal(TransactionCompletedData{Amount: 500}),
		},
	}

	// Append events
	position, err := store.Append(ctx, events, nil)
	if err != nil {
		log.Fatalf("Failed to append events: %v", err)
	}
	fmt.Printf("Appended events up to position: %d\n", position)

	// Use ProjectDecisionModel to build decision model
	fmt.Println("\n=== Using ProjectDecisionModel API ===")
	states, appendCondition, err := store.ProjectDecisionModel(ctx, query, nil, projectors)
	if err != nil {
		log.Fatalf("Failed to read stream: %v", err)
	}

	fmt.Printf("\n=== Decision Model Results ===\n")

	// Display final states
	if account, ok := states["account"].(*AccountState); ok {
		fmt.Printf("Account State: Balance=%d\n", account.Balance)
	}

	if transactions, ok := states["transactions"].(*TransactionState); ok {
		fmt.Printf("Transaction State: Count=%d, Total=%d\n", transactions.Count, transactions.TotalAmount)
	}

	// The AppendCondition can be used for optimistic locking
	fmt.Printf("\n=== Append Condition for Optimistic Locking ===\n")
	fmt.Printf("FailIfEventsMatch: %+v\n", appendCondition.FailIfEventsMatch)
	fmt.Printf("After position: %d\n", *appendCondition.After)

	// Example: Use the AppendCondition to append new events
	newEvents := []dcb.InputEvent{
		{
			Type: "TransactionCompleted",
			Tags: []dcb.Tag{{Key: "account_id", Value: "acc123"}},
			Data: mustMarshal(TransactionCompletedData{Amount: 200}),
		},
	}

	fmt.Println("\n=== Appending New Events with Optimistic Locking ===")
	newPosition, err := store.Append(ctx, newEvents, &appendCondition)
	if err != nil {
		log.Fatalf("Failed to append new events: %v", err)
	}
	fmt.Printf("Successfully appended new events up to position: %d\n", newPosition)
}

// Helper types
type AccountState struct {
	ID      string
	Balance int
}

type TransactionState struct {
	Count       int
	TotalAmount int
}

type AccountCreatedData struct {
	InitialBalance int `json:"initial_balance"`
}

type AccountUpdatedData struct {
	NewBalance int `json:"new_balance"`
}

type TransactionCompletedData struct {
	Amount int `json:"amount"`
}

func mustMarshal(v any) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("Failed to marshal: %v", err))
	}
	return data
}
