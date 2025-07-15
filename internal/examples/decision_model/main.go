package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"

	"github.com/rodolfodpk/go-crablet/internal/examples/utils"
	"github.com/rodolfodpk/go-crablet/pkg/dcb"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Command types
type OpenAccountCommand struct {
	AccountID      string
	InitialBalance int
}

type ProcessTransactionCommand struct {
	AccountID string
	Amount    int
}

func main() {
	ctx := context.Background()

	// Connect to PostgreSQL
	pool, err := pgxpool.New(ctx, "postgres://crablet:crablet@localhost:5432/crablet?sslmode=disable")
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	// Create event store
	store, err := dcb.NewEventStore(ctx, pool)
	if err != nil {
		log.Fatalf("Failed to create event store: %v", err)
	}

	// Use the event store directly

	runID := fmt.Sprintf("run_id_%d", rand.Int63())
	accountID := "acc_decision_" + runID

	// Command 1: Open Account
	openAccountCmd := OpenAccountCommand{
		AccountID:      accountID,
		InitialBalance: 1000,
	}
	err = handleOpenAccount(ctx, store, openAccountCmd)
	if err != nil {
		log.Fatalf("Open account failed: %v", err)
	}

	// Command 2: Process Transaction
	processTransactionCmd := ProcessTransactionCommand{
		AccountID: accountID,
		Amount:    500,
	}
	err = handleProcessTransaction(ctx, store, processTransactionCmd)
	if err != nil {
		log.Fatalf("Process transaction failed: %v", err)
	}

	// Re-project state and get a fresh append condition for optimistic locking
	projectors := []dcb.StateProjector{
		{
			ID: "account",
			Query: dcb.NewQuery(
				dcb.NewTags("account_id", accountID),
				"AccountOpened", "AccountBalanceChanged",
			),
			InitialState: &AccountState{ID: accountID, Balance: 0},
			TransitionFn: func(state any, event dcb.Event) any {
				account := state.(*AccountState)
				switch event.Type {
				case "AccountOpened":
					var data AccountOpenedData
					json.Unmarshal(event.Data, &data)
					account.Balance = data.InitialBalance
				case "AccountBalanceChanged":
					var data AccountBalanceChangedData
					json.Unmarshal(event.Data, &data)
					account.Balance = data.NewBalance
				}
				return account
			},
		},
	}
	_, appendCondition, err := store.Project(ctx, projectors, nil)
	if err != nil {
		log.Fatalf("Failed to project state for optimistic locking: %v", err)
	}

	// Command 3: Process another transaction with optimistic locking
	processTransaction2Cmd := ProcessTransactionCommand{
		AccountID: accountID,
		Amount:    200,
	}
	err = handleProcessTransactionWithCondition(ctx, store, processTransaction2Cmd, appendCondition)
	if err != nil {
		log.Fatalf("Process transaction 2 failed: %v", err)
	}

	// Use Project to build decision model
	fmt.Println("\n=== Using Project API ===")

	// Define projectors for decision model
	accountProjector := dcb.StateProjector{
		Query: dcb.NewQuery(
			dcb.NewTags("account_id", accountID),
			"AccountOpened", "AccountBalanceChanged",
		),
		InitialState: &AccountState{ID: accountID, Balance: 0},
		TransitionFn: func(state any, event dcb.Event) any {
			account := state.(*AccountState)
			switch event.Type {
			case "AccountOpened":
				var data AccountOpenedData
				json.Unmarshal(event.Data, &data)
				account.Balance = data.InitialBalance
			case "AccountBalanceChanged":
				var data AccountBalanceChangedData
				json.Unmarshal(event.Data, &data)
				account.Balance = data.NewBalance
			}
			return account
		},
	}

	transactionProjector := dcb.StateProjector{
		Query: dcb.NewQuery(
			dcb.NewTags("account_id", accountID),
			"TransactionProcessed",
		),
		InitialState: &TransactionState{Count: 0, TotalAmount: 0},
		TransitionFn: func(state any, event dcb.Event) any {
			transactions := state.(*TransactionState)
			if event.Type == "TransactionProcessed" {
				var data TransactionProcessedData
				json.Unmarshal(event.Data, &data)
				transactions.Count++
				transactions.TotalAmount += data.Amount
			}
			return transactions
		},
	}

	// Create batch projectors
	projectors = []dcb.StateProjector{
		{
			ID:           "account",
			Query:        accountProjector.Query,
			InitialState: accountProjector.InitialState,
			TransitionFn: accountProjector.TransitionFn,
		},
		{
			ID:           "transactions",
			Query:        transactionProjector.Query,
			InitialState: transactionProjector.InitialState,
			TransitionFn: transactionProjector.TransitionFn,
		},
	}

	states, appendCondition, err := store.Project(ctx, projectors, nil)
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
	fmt.Printf("Using append condition for optimistic locking\n")

	// Dump all events to show what was created
	fmt.Println("\n=== Events in Database ===")
	utils.DumpEvents(ctx, pool)
}

// Command handlers with their own business rules

func handleOpenAccount(ctx context.Context, store dcb.EventStore, cmd OpenAccountCommand) error {
	projectors := []dcb.StateProjector{
		{
			ID: "accountExists",
			Query: dcb.NewQuery(
				dcb.NewTags("account_id", cmd.AccountID),
				"AccountOpened",
			),
			InitialState: false,
			TransitionFn: func(state any, event dcb.Event) any { return true },
		},
	}
	states, appendCondition, err := store.Project(ctx, projectors, nil)
	if err != nil {
		return fmt.Errorf("failed to check account existence: %w", err)
	}
	if states["accountExists"].(bool) {
		return fmt.Errorf("account %s already exists", cmd.AccountID)
	}
	events := []dcb.InputEvent{
		dcb.NewInputEvent(
			"AccountOpened",
			dcb.NewTags("account_id", cmd.AccountID),
			toJSON(AccountOpenedData{InitialBalance: cmd.InitialBalance}),
		),
	}
	err = store.Append(ctx, events, &appendCondition)
	if err != nil {
		return fmt.Errorf("failed to open account: %w", err)
	}
	fmt.Printf("Opened account %s with balance %d\n", cmd.AccountID, cmd.InitialBalance)
	return nil
}

func handleProcessTransaction(ctx context.Context, store dcb.EventStore, cmd ProcessTransactionCommand) error {
	accountProjector := dcb.StateProjector{
		Query: dcb.NewQuery(
			dcb.NewTags("account_id", cmd.AccountID),
			"AccountOpened", "AccountBalanceChanged",
		),
		InitialState: &AccountState{ID: cmd.AccountID, Balance: 0},
		TransitionFn: func(state any, event dcb.Event) any {
			account := state.(*AccountState)
			switch event.Type {
			case "AccountOpened":
				var data AccountOpenedData
				json.Unmarshal(event.Data, &data)
				account.Balance = data.InitialBalance
			case "AccountBalanceChanged":
				var data AccountBalanceChangedData
				json.Unmarshal(event.Data, &data)
				account.Balance = data.NewBalance
			}
			return account
		},
	}
	states, appendCondition, err := store.Project(ctx, []dcb.StateProjector{{
		ID:           "account",
		Query:        accountProjector.Query,
		InitialState: accountProjector.InitialState,
		TransitionFn: accountProjector.TransitionFn,
	}}, nil)
	if err != nil {
		return fmt.Errorf("failed to project account state: %w", err)
	}
	account := states["account"].(*AccountState)
	if account.Balance == 0 {
		return fmt.Errorf("account %s does not exist", cmd.AccountID)
	}
	events := []dcb.InputEvent{
		dcb.NewInputEvent(
			"TransactionProcessed",
			dcb.NewTags("account_id", cmd.AccountID),
			toJSON(TransactionProcessedData{Amount: cmd.Amount}),
		),
	}
	err = store.Append(ctx, events, &appendCondition)
	if err != nil {
		return fmt.Errorf("failed to process transaction: %w", err)
	}
	fmt.Printf("Processed transaction of %d for account %s\n", cmd.Amount, cmd.AccountID)
	return nil
}

func handleProcessTransactionWithCondition(ctx context.Context, store dcb.EventStore, cmd ProcessTransactionCommand, condition dcb.AppendCondition) error {
	accountProjector := dcb.StateProjector{
		Query: dcb.NewQuery(
			dcb.NewTags("account_id", cmd.AccountID),
			"AccountOpened", "AccountBalanceChanged",
		),
		InitialState: &AccountState{ID: cmd.AccountID, Balance: 0},
		TransitionFn: func(state any, event dcb.Event) any {
			account := state.(*AccountState)
			switch event.Type {
			case "AccountOpened":
				var data AccountOpenedData
				json.Unmarshal(event.Data, &data)
				account.Balance = data.InitialBalance
			case "AccountBalanceChanged":
				var data AccountBalanceChangedData
				json.Unmarshal(event.Data, &data)
				account.Balance = data.NewBalance
			}
			return account
		},
	}
	states, _, err := store.Project(ctx, []dcb.StateProjector{{
		ID:           "account",
		Query:        accountProjector.Query,
		InitialState: accountProjector.InitialState,
		TransitionFn: accountProjector.TransitionFn,
	}}, nil)
	if err != nil {
		return fmt.Errorf("failed to project account state: %w", err)
	}
	account := states["account"].(*AccountState)
	if account.Balance == 0 {
		return fmt.Errorf("account %s does not exist", cmd.AccountID)
	}
	events := []dcb.InputEvent{
		dcb.NewInputEvent(
			"TransactionProcessed",
			dcb.NewTags("account_id", cmd.AccountID),
			toJSON(TransactionProcessedData{Amount: cmd.Amount}),
		),
	}
	err = store.Append(ctx, events, &condition)
	if err != nil {
		return fmt.Errorf("failed to process transaction with optimistic locking: %w", err)
	}
	fmt.Printf("Successfully processed transaction of %d for account %s\n", cmd.Amount, cmd.AccountID)
	return nil
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

type AccountOpenedData struct {
	InitialBalance int `json:"initial_balance"`
}

type AccountBalanceChangedData struct {
	NewBalance int `json:"new_balance"`
}

type TransactionProcessedData struct {
	Amount int `json:"amount"`
}

func toJSON(v any) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("Failed to marshal: %v", err))
	}
	return data
}
