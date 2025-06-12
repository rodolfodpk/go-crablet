package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

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

	// Define projectors for different business concerns
	accountProjector := dcb.StateProjector{
		Query: dcb.Query{
			Items: []dcb.QueryItem{
				{
					EventTypes: []string{"AccountCreated", "AccountUpdated"},
					Tags:       []dcb.Tag{{Key: "account_id", Value: "acc123"}},
				},
			},
		},
		InitialState: &AccountState{ID: "acc123", Balance: 0, CreatedAt: time.Now()},
		TransitionFn: func(state any, event dcb.Event) any {
			account := state.(*AccountState)
			switch event.Type {
			case "AccountCreated":
				var data AccountCreatedData
				json.Unmarshal(event.Data, &data)
				account.Balance = data.InitialBalance
				account.CreatedAt = time.Now()
			case "AccountUpdated":
				var data AccountUpdatedData
				json.Unmarshal(event.Data, &data)
				account.Balance = data.NewBalance
				account.UpdatedAt = time.Now()
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
		InitialState: &TransactionState{Count: 0, TotalAmount: 0, LastTransaction: time.Time{}},
		TransitionFn: func(state any, event dcb.Event) any {
			transactions := state.(*TransactionState)
			if event.Type == "TransactionCompleted" {
				var data TransactionCompletedData
				json.Unmarshal(event.Data, &data)
				transactions.Count++
				transactions.TotalAmount += data.Amount
				transactions.LastTransaction = time.Now()
			}
			return transactions
		},
	}

	balanceProjector := dcb.StateProjector{
		Query: dcb.Query{
			Items: []dcb.QueryItem{
				{
					EventTypes: []string{"AccountCreated", "AccountUpdated", "TransactionCompleted"},
					Tags:       []dcb.Tag{{Key: "account_id", Value: "acc123"}},
				},
			},
		},
		InitialState: &BalanceState{CurrentBalance: 0, LastUpdated: time.Time{}, ChangeCount: 0},
		TransitionFn: func(state any, event dcb.Event) any {
			balance := state.(*BalanceState)
			switch event.Type {
			case "AccountCreated":
				var data AccountCreatedData
				json.Unmarshal(event.Data, &data)
				balance.CurrentBalance = data.InitialBalance
				balance.LastUpdated = time.Now()
				balance.ChangeCount++
			case "AccountUpdated":
				var data AccountUpdatedData
				json.Unmarshal(event.Data, &data)
				balance.CurrentBalance = data.NewBalance
				balance.LastUpdated = time.Now()
				balance.ChangeCount++
			case "TransactionCompleted":
				var data TransactionCompletedData
				json.Unmarshal(event.Data, &data)
				balance.CurrentBalance += data.Amount
				balance.LastUpdated = time.Now()
				balance.ChangeCount++
			}
			return balance
		},
	}

	// Create batch projectors
	projectors := []dcb.BatchProjector{
		{ID: "account", StateProjector: accountProjector},
		{ID: "transactions", StateProjector: transactionProjector},
		{ID: "balance", StateProjector: balanceProjector},
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

	// Create test events
	events := []dcb.InputEvent{
		{
			Type: "AccountCreated",
			Tags: []dcb.Tag{{Key: "account_id", Value: "acc123"}},
			Data: mustMarshal(AccountCreatedData{InitialBalance: 1000}),
		},
		{
			Type: "TransactionCompleted",
			Tags: []dcb.Tag{{Key: "account_id", Value: "acc123"}},
			Data: mustMarshal(TransactionCompletedData{Amount: 500, Description: "Deposit"}),
		},
		{
			Type: "AccountUpdated",
			Tags: []dcb.Tag{{Key: "account_id", Value: "acc123"}},
			Data: mustMarshal(AccountUpdatedData{NewBalance: 2000, Reason: "Manual adjustment"}),
		},
		{
			Type: "TransactionCompleted",
			Tags: []dcb.Tag{{Key: "account_id", Value: "acc123"}},
			Data: mustMarshal(TransactionCompletedData{Amount: -300, Description: "Withdrawal"}),
		},
		{
			Type: "TransactionCompleted",
			Tags: []dcb.Tag{{Key: "account_id", Value: "acc123"}},
			Data: mustMarshal(TransactionCompletedData{Amount: 100, Description: "Interest"}),
		},
	}

	// Append events
	fmt.Println("Appending events...")
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
	ID        string    `json:"id"`
	Balance   int       `json:"balance"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type TransactionState struct {
	Count           int       `json:"count"`
	TotalAmount     int       `json:"total_amount"`
	LastTransaction time.Time `json:"last_transaction"`
}

type BalanceState struct {
	CurrentBalance int       `json:"current_balance"`
	LastUpdated    time.Time `json:"last_updated"`
	ChangeCount    int       `json:"change_count"`
}

type AccountCreatedData struct {
	InitialBalance int `json:"initial_balance"`
}

type AccountUpdatedData struct {
	NewBalance int    `json:"new_balance"`
	Reason     string `json:"reason"`
}

type TransactionCompletedData struct {
	Amount      int    `json:"amount"`
	Description string `json:"description"`
}

func mustMarshal(v any) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("Failed to marshal: %v", err))
	}
	return data
}
