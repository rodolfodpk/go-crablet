package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"go-crablet/internal/examples/utils"
	"go-crablet/pkg/dcb"

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

	// Cast to ChannelEventStore for extended functionality
	channelStore := store.(dcb.ChannelEventStore)

	// Command 1: Open Account
	openAccountCmd := OpenAccountCommand{
		AccountID:      "acc123",
		InitialBalance: 1000,
	}
	err = handleOpenAccount(ctx, channelStore, openAccountCmd)
	if err != nil {
		log.Fatalf("Open account failed: %v", err)
	}

	// Command 2: Process Transaction
	processTransactionCmd := ProcessTransactionCommand{
		AccountID: "acc123",
		Amount:    500,
	}
	err = handleProcessTransaction(ctx, channelStore, processTransactionCmd)
	if err != nil {
		log.Fatalf("Process transaction failed: %v", err)
	}

	// Use ProjectDecisionModel to build decision model
	fmt.Println("\n=== Using ProjectDecisionModel API ===")

	// Define projectors for decision model
	accountProjector := dcb.StateProjector{
		Query: dcb.NewQuery(
			dcb.NewTags("account_id", "acc123"),
			"AccountOpened", "AccountBalanceChanged",
		),
		InitialState: &AccountState{ID: "acc123", Balance: 0},
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
			dcb.NewTags("account_id", "acc123"),
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
	projectors := []dcb.BatchProjector{
		{ID: "account", StateProjector: accountProjector},
		{ID: "transactions", StateProjector: transactionProjector},
	}

	states, appendCondition, err := channelStore.ProjectDecisionModel(ctx, projectors)
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

	// Command 3: Process another transaction with optimistic locking
	processTransaction2Cmd := ProcessTransactionCommand{
		AccountID: "acc123",
		Amount:    200,
	}
	err = handleProcessTransactionWithCondition(ctx, channelStore, processTransaction2Cmd, appendCondition)
	if err != nil {
		log.Fatalf("Process transaction 2 failed: %v", err)
	}

	// Dump all events to show what was created
	fmt.Println("\n=== Events in Database ===")
	utils.DumpEvents(ctx, pool)
}

// Command handlers with their own business rules

func handleOpenAccount(ctx context.Context, store dcb.ChannelEventStore, cmd OpenAccountCommand) error {
	// Command-specific projectors
	projectors := []dcb.BatchProjector{
		{ID: "accountExists", StateProjector: dcb.StateProjector{
			Query: dcb.NewQuery(
				dcb.NewTags("account_id", cmd.AccountID),
				"AccountOpened",
			),
			InitialState: false,
			TransitionFn: func(state any, event dcb.Event) any {
				return true // If we see an AccountOpened event, account exists
			},
		}},
	}

	states, appendCondition, err := store.ProjectDecisionModel(ctx, projectors)
	if err != nil {
		return fmt.Errorf("failed to check account existence: %w", err)
	}

	// Command-specific business rule: account must not already exist
	if states["accountExists"].(bool) {
		return fmt.Errorf("account %s already exists", cmd.AccountID)
	}

	// Create events for this command
	events := []dcb.InputEvent{
		dcb.NewInputEvent(
			"AccountOpened",
			dcb.NewTags("account_id", cmd.AccountID),
			toJSON(AccountOpenedData{InitialBalance: cmd.InitialBalance}),
		),
	}

	// Append events atomically for this command
	err = store.Append(ctx, events, appendCondition)
	if err != nil {
		return fmt.Errorf("failed to open account: %w", err)
	}

	fmt.Printf("Opened account %s with balance %d\n", cmd.AccountID, cmd.InitialBalance)
	return nil
}

func handleProcessTransaction(ctx context.Context, store dcb.ChannelEventStore, cmd ProcessTransactionCommand) error {
	// Command-specific projectors
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

	states, appendCondition, err := store.ProjectDecisionModel(ctx, []dcb.BatchProjector{
		{ID: "account", StateProjector: accountProjector},
	})
	if err != nil {
		return fmt.Errorf("failed to project account state: %w", err)
	}

	account := states["account"].(*AccountState)

	// Command-specific business rule: account must exist
	if account.Balance == 0 {
		return fmt.Errorf("account %s does not exist", cmd.AccountID)
	}

	// Create events for this command
	events := []dcb.InputEvent{
		dcb.NewInputEvent(
			"TransactionProcessed",
			dcb.NewTags("account_id", cmd.AccountID),
			toJSON(TransactionProcessedData{Amount: cmd.Amount}),
		),
	}

	// Append events atomically for this command
	err = store.Append(ctx, events, appendCondition)
	if err != nil {
		return fmt.Errorf("failed to process transaction: %w", err)
	}

	fmt.Printf("Processed transaction of %d for account %s\n", cmd.Amount, cmd.AccountID)
	return nil
}

func handleProcessTransactionWithCondition(ctx context.Context, store dcb.ChannelEventStore, cmd ProcessTransactionCommand, condition dcb.AppendCondition) error {
	// Command-specific projectors
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

	states, _, err := store.ProjectDecisionModel(ctx, []dcb.BatchProjector{
		{ID: "account", StateProjector: accountProjector},
	})
	if err != nil {
		return fmt.Errorf("failed to project account state: %w", err)
	}

	account := states["account"].(*AccountState)

	// Command-specific business rule: account must exist
	if account.Balance == 0 {
		return fmt.Errorf("account %s does not exist", cmd.AccountID)
	}

	// Create events for this command
	events := []dcb.InputEvent{
		dcb.NewInputEvent(
			"TransactionProcessed",
			dcb.NewTags("account_id", cmd.AccountID),
			toJSON(TransactionProcessedData{Amount: cmd.Amount}),
		),
	}

	// Append events atomically for this command with optimistic locking
	err = store.Append(ctx, events, condition)
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
