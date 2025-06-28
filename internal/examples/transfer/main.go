// This example is standalone. Run with: go run examples/transfer/main.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"go-crablet/internal/examples/utils"
	"go-crablet/pkg/dcb"

	"github.com/jackc/pgx/v5/pgxpool"
)

// AccountState holds the state for an account
type AccountState struct {
	AccountID string
	Owner     string
	Balance   int
	CreatedAt time.Time
	UpdatedAt time.Time
}

// AccountOpened represents when an account is opened
type AccountOpened struct {
	AccountID      string    `json:"account_id"`
	Owner          string    `json:"owner"`
	InitialBalance int       `json:"initial_balance"`
	OpenedAt       time.Time `json:"opened_at"`
}

// MoneyTransferred represents a money transfer between accounts
type MoneyTransferred struct {
	TransferID    string    `json:"transfer_id"`
	FromAccountID string    `json:"from_account_id"`
	ToAccountID   string    `json:"to_account_id"`
	Amount        int       `json:"amount"`
	FromBalance   int       `json:"from_balance"` // Balance after transfer
	ToBalance     int       `json:"to_balance"`   // Balance after transfer
	TransferredAt time.Time `json:"transferred_at"`
	Description   string    `json:"description,omitempty"`
}

// Command types
type CreateAccountCommand struct {
	AccountID      string
	Owner          string
	InitialBalance int
}

type TransferMoneyCommand struct {
	TransferID    string
	FromAccountID string
	ToAccountID   string
	Amount        int
	Description   string
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

	// Command 1: Create first account
	createAccount1Cmd := CreateAccountCommand{
		AccountID:      "acc1",
		Owner:          "Alice",
		InitialBalance: 1000,
	}
	err = handleCreateAccount(ctx, store, createAccount1Cmd)
	if err != nil {
		log.Fatalf("Create account 1 failed: %v", err)
	}

	// Command 2: Create second account
	createAccount2Cmd := CreateAccountCommand{
		AccountID:      "acc456",
		InitialBalance: 500,
	}
	err = handleCreateAccount(ctx, store, createAccount2Cmd)
	if err != nil {
		log.Fatalf("Create account 2 failed: %v", err)
	}

	// Command 3: Transfer money
	transferCmd := TransferMoneyCommand{
		TransferID:    fmt.Sprintf("tx-%d", time.Now().UnixNano()),
		FromAccountID: "acc1",
		ToAccountID:   "acc456",
		Amount:        300,
	}
	err = handleTransferMoney(ctx, store, transferCmd)
	if err != nil {
		log.Fatalf("Transfer failed: %v", err)
	}

	fmt.Printf("Transfer successful! Transfer ID: %s\n", transferCmd.TransferID)

	// Dump all events to show what was created
	fmt.Println("\n=== Events in Database ===")
	utils.DumpEvents(ctx, pool)
}

// Command handlers with their own business rules

func handleCreateAccount(ctx context.Context, store dcb.EventStore, cmd CreateAccountCommand) error {
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
			mustJSON(AccountOpened{
				AccountID:      cmd.AccountID,
				Owner:          cmd.Owner,
				InitialBalance: cmd.InitialBalance,
				OpenedAt:       time.Now(),
			}),
		),
	}

	// Append events atomically for this command
	err = store.AppendIfSerializable(ctx, events, appendCondition)
	if err != nil {
		return fmt.Errorf("failed to create account: %w", err)
	}

	fmt.Printf("Created account %s for %s with balance %d\n", cmd.AccountID, cmd.Owner, cmd.InitialBalance)
	return nil
}

func handleTransferMoney(ctx context.Context, store dcb.EventStore, cmd TransferMoneyCommand) error {
	// Command-specific projectors
	fromProjector := dcb.StateProjector{
		Query: dcb.NewQuery(
			dcb.NewTags("account_id", cmd.FromAccountID),
			"AccountOpened", "MoneyTransferred",
		),
		InitialState: &AccountState{AccountID: cmd.FromAccountID},
		TransitionFn: func(state any, event dcb.Event) any {
			acc := state.(*AccountState)
			switch event.Type {
			case "AccountOpened":
				var data AccountOpened
				if err := json.Unmarshal(event.Data, &data); err == nil {
					acc.Owner = data.Owner
					acc.Balance = data.InitialBalance
					acc.CreatedAt = data.OpenedAt
					acc.UpdatedAt = data.OpenedAt
				}
			case "MoneyTransferred":
				var data MoneyTransferred
				if err := json.Unmarshal(event.Data, &data); err == nil {
					// Check if this event affects the from account
					if data.FromAccountID == cmd.FromAccountID {
						acc.Balance = data.FromBalance
						acc.UpdatedAt = data.TransferredAt
					} else if data.ToAccountID == cmd.FromAccountID {
						acc.Balance = data.ToBalance
						acc.UpdatedAt = data.TransferredAt
					}
				}
			}
			return acc
		},
	}

	toProjector := dcb.StateProjector{
		Query: dcb.NewQuery(
			dcb.NewTags("account_id", cmd.ToAccountID),
			"AccountOpened", "MoneyTransferred",
		),
		InitialState: &AccountState{AccountID: cmd.ToAccountID},
		TransitionFn: func(state any, event dcb.Event) any {
			acc := state.(*AccountState)
			switch event.Type {
			case "AccountOpened":
				var data AccountOpened
				if err := json.Unmarshal(event.Data, &data); err == nil {
					acc.Owner = data.Owner
					acc.Balance = data.InitialBalance
					acc.CreatedAt = data.OpenedAt
					acc.UpdatedAt = data.OpenedAt
				}
			case "MoneyTransferred":
				var data MoneyTransferred
				if err := json.Unmarshal(event.Data, &data); err == nil {
					// Check if this event affects the to account
					if data.FromAccountID == cmd.ToAccountID {
						acc.Balance = data.FromBalance
						acc.UpdatedAt = data.TransferredAt
					} else if data.ToAccountID == cmd.ToAccountID {
						acc.Balance = data.ToBalance
						acc.UpdatedAt = data.TransferredAt
					}
				}
			}
			return acc
		},
	}

	// Project both accounts using the DCB decision model pattern
	states, appendCondition, err := store.ProjectDecisionModel(ctx, []dcb.BatchProjector{
		{ID: "from", StateProjector: fromProjector},
		{ID: "to", StateProjector: toProjector},
	})
	if err != nil {
		return fmt.Errorf("projection failed: %w", err)
	}

	from := states["from"].(*AccountState)
	to := states["to"].(*AccountState)

	// Command-specific business rules
	if from.Balance < cmd.Amount {
		return fmt.Errorf("insufficient funds: account %s has %d, needs %d", cmd.FromAccountID, from.Balance, cmd.Amount)
	}
	if cmd.Amount <= 0 {
		return fmt.Errorf("invalid transfer amount: %d", cmd.Amount)
	}
	if cmd.FromAccountID == cmd.ToAccountID {
		return fmt.Errorf("cannot transfer to the same account")
	}

	// Calculate new balances
	newFromBalance := from.Balance - cmd.Amount
	newToBalance := to.Balance + cmd.Amount

	// Create events for this command
	events := []dcb.InputEvent{
		dcb.NewInputEvent(
			"MoneyTransferred",
			dcb.NewTags(
				"transfer_id", cmd.TransferID,
				"from_account_id", cmd.FromAccountID,
				"to_account_id", cmd.ToAccountID,
				"account_id", cmd.FromAccountID, // Tag for from account
			),
			mustJSON(MoneyTransferred{
				TransferID:    cmd.TransferID,
				FromAccountID: cmd.FromAccountID,
				ToAccountID:   cmd.ToAccountID,
				Amount:        cmd.Amount,
				FromBalance:   newFromBalance,
				ToBalance:     newToBalance,
				TransferredAt: time.Now(),
				Description:   cmd.Description,
			}),
		),
		dcb.NewInputEvent(
			"MoneyTransferred",
			dcb.NewTags(
				"transfer_id", cmd.TransferID,
				"from_account_id", cmd.FromAccountID,
				"to_account_id", cmd.ToAccountID,
				"account_id", cmd.ToAccountID, // Tag for to account
			),
			mustJSON(MoneyTransferred{
				TransferID:    cmd.TransferID,
				FromAccountID: cmd.FromAccountID,
				ToAccountID:   cmd.ToAccountID,
				Amount:        cmd.Amount,
				FromBalance:   newFromBalance,
				ToBalance:     newToBalance,
				TransferredAt: time.Now(),
				Description:   cmd.Description,
			}),
		),
	}

	// Use the append condition from the decision model for optimistic locking
	// All events are appended atomically for this command with serializable isolation
	err = store.AppendIfSerializable(ctx, events, appendCondition)
	if err != nil {
		return fmt.Errorf("append failed: %w", err)
	}

	// Display the transfer results using the calculated new balances
	fmt.Printf("Account %s: %d -> %d\n", cmd.FromAccountID, from.Balance, newFromBalance)
	fmt.Printf("Account %s: %d -> %d\n", cmd.ToAccountID, to.Balance, newToBalance)

	return nil
}

func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}
