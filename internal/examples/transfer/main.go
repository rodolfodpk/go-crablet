// This example demonstrates command execution with the DCB approach for account transfers
//
// BEST PRACTICE: This example shows the recommended approach of using structs with WithData()
// instead of map[string]interface{} for better type safety, performance, and readability.
//
// ✅ RECOMMENDED: WithData(AccountOpened{...})
// ❌ AVOID: WithData(map[string]interface{}{...})
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/rodolfodpk/go-crablet/internal/examples/utils"
	"github.com/rodolfodpk/go-crablet/pkg/dcb"

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
const (
	CommandTypeOpenAccount   = "open_account"
	CommandTypeTransferMoney = "transfer_money"
)

// OpenAccountCommand represents a command to open an account
type OpenAccountCommand struct {
	AccountID      string `json:"account_id"`
	Owner          string `json:"owner"`
	InitialBalance int    `json:"initial_balance"`
}

// TransferMoneyCommand represents a command to transfer money between accounts
type TransferMoneyCommand struct {
	TransferID    string `json:"transfer_id"`
	FromAccountID string `json:"from_account_id"`
	ToAccountID   string `json:"to_account_id"`
	Amount        int    `json:"amount"`
	Description   string `json:"description,omitempty"`
}

// HandleOpenAccount handles the opening of an account
func HandleOpenAccount(ctx context.Context, store dcb.EventStore, cmd OpenAccountCommand) ([]dcb.InputEvent, error) {
	// Check if account already exists using simplified query
	query := dcb.NewQueryBuilder().WithTagAndType("account_id", cmd.AccountID, "AccountOpened").Build()

	events, err := store.Query(ctx, query, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to query existing account: %w", err)
	}

	if len(events) > 0 {
		return nil, fmt.Errorf("account %s already exists", cmd.AccountID)
	}

	// Create account event
	accountOpened := AccountOpened{
		AccountID:      cmd.AccountID,
		Owner:          cmd.Owner,
		InitialBalance: cmd.InitialBalance,
		OpenedAt:       time.Now(),
	}

	event := dcb.NewEvent("AccountOpened").
		WithTag("account_id", cmd.AccountID).
		WithData(accountOpened).
		Build()
	return []dcb.InputEvent{event}, nil
}

// HandleTransferMoney handles money transfers between accounts
func HandleTransferMoney(ctx context.Context, store dcb.EventStore, cmd TransferMoneyCommand) ([]dcb.InputEvent, *dcb.AppendCondition, error) {
	// Project source account state using simplified query
	fromAccountProjector := dcb.StateProjector{
		ID:           "fromAccount",
		Query:        dcb.NewQueryBuilder().WithTypes("AccountOpened", "MoneyTransferred").WithTag("account_id", cmd.FromAccountID).Build(),
		InitialState: AccountState{},
		TransitionFn: func(state any, event dcb.Event) any {
			accountState := state.(AccountState)

			if event.Type == "AccountOpened" {
				var accountOpened AccountOpened
				if err := json.Unmarshal(event.Data, &accountOpened); err != nil {
					return accountState
				}
				accountState.AccountID = accountOpened.AccountID
				accountState.Owner = accountOpened.Owner
				accountState.Balance = accountOpened.InitialBalance
				accountState.CreatedAt = accountOpened.OpenedAt
				accountState.UpdatedAt = accountOpened.OpenedAt
			} else if event.Type == "MoneyTransferred" {
				var transfer MoneyTransferred
				if err := json.Unmarshal(event.Data, &transfer); err != nil {
					return accountState
				}
				if transfer.FromAccountID == cmd.FromAccountID {
					accountState.Balance = transfer.FromBalance
					accountState.UpdatedAt = transfer.TransferredAt
				} else if transfer.ToAccountID == cmd.FromAccountID {
					accountState.Balance = transfer.ToBalance
					accountState.UpdatedAt = transfer.TransferredAt
				}
			}
			return accountState
		},
	}

	// Project destination account state using simplified query
	toAccountProjector := dcb.StateProjector{
		ID:           "toAccount",
		Query:        dcb.NewQueryBuilder().WithTypes("AccountOpened", "MoneyTransferred").WithTag("account_id", cmd.ToAccountID).Build(),
		InitialState: AccountState{},
		TransitionFn: func(state any, event dcb.Event) any {
			accountState := state.(AccountState)

			if event.Type == "AccountOpened" {
				var accountOpened AccountOpened
				if err := json.Unmarshal(event.Data, &accountOpened); err != nil {
					return accountState
				}
				accountState.AccountID = accountOpened.AccountID
				accountState.Owner = accountOpened.Owner
				accountState.Balance = accountOpened.InitialBalance
				accountState.CreatedAt = accountOpened.OpenedAt
				accountState.UpdatedAt = accountOpened.OpenedAt
			} else if event.Type == "MoneyTransferred" {
				var transfer MoneyTransferred
				if err := json.Unmarshal(event.Data, &transfer); err != nil {
					return accountState
				}
				if transfer.FromAccountID == cmd.ToAccountID {
					accountState.Balance = transfer.FromBalance
					accountState.UpdatedAt = transfer.TransferredAt
				} else if transfer.ToAccountID == cmd.ToAccountID {
					accountState.Balance = transfer.ToBalance
					accountState.UpdatedAt = transfer.TransferredAt
				}
			}
			return accountState
		},
	}

	// Project both accounts
	projectedStates, appendCondition, err := store.Project(ctx, []dcb.StateProjector{fromAccountProjector, toAccountProjector}, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to project account states: %w", err)
	}

	fromAccount := projectedStates["fromAccount"].(AccountState)
	toAccount := projectedStates["toAccount"].(AccountState)

	// Validate source account exists
	if fromAccount.Owner == "" {
		return nil, nil, fmt.Errorf("source account does not exist")
	}

	// Validate sufficient funds
	if fromAccount.Balance < cmd.Amount {
		return nil, nil, fmt.Errorf("insufficient funds: balance %d, requested %d", fromAccount.Balance, cmd.Amount)
	}

	// Create transfer event using simplified tags
	transfer := MoneyTransferred{
		TransferID:    cmd.TransferID,
		FromAccountID: cmd.FromAccountID,
		ToAccountID:   cmd.ToAccountID,
		Amount:        cmd.Amount,
		FromBalance:   fromAccount.Balance - cmd.Amount,
		ToBalance:     toAccount.Balance + cmd.Amount,
		TransferredAt: time.Now(),
		Description:   cmd.Description,
	}

	event := dcb.NewEvent("MoneyTransferred").
		WithTag("transfer_id", cmd.TransferID).
		WithTag("from_account_id", cmd.FromAccountID).
		WithTag("to_account_id", cmd.ToAccountID).
		WithData(transfer).
		Build()

	return []dcb.InputEvent{event}, &appendCondition, nil
}

// HandleCommand routes commands to appropriate handlers
func HandleCommand(ctx context.Context, store dcb.EventStore, command dcb.Command) ([]dcb.InputEvent, *dcb.AppendCondition, error) {
	switch command.GetType() {
	case CommandTypeOpenAccount:
		var cmd OpenAccountCommand
		if err := json.Unmarshal(command.GetData(), &cmd); err != nil {
			return nil, nil, fmt.Errorf("failed to unmarshal open account command: %w", err)
		}
		events, err := HandleOpenAccount(ctx, store, cmd)
		return events, nil, err
	case CommandTypeTransferMoney:
		var cmd TransferMoneyCommand
		if err := json.Unmarshal(command.GetData(), &cmd); err != nil {
			return nil, nil, fmt.Errorf("failed to unmarshal transfer command: %w", err)
		}
		events, appendCondition, err := HandleTransferMoney(ctx, store, cmd)
		return events, appendCondition, err
	default:
		return nil, nil, fmt.Errorf("unknown command type: %s", command.GetType())
	}
}

// Helper functions for flatter code structure

// executeOpenAccountCommand executes an open account command and returns success message
func executeOpenAccountCommand(ctx context.Context, commandExecutor dcb.CommandExecutor, handler dcb.CommandHandler, cmd OpenAccountCommand, requestID string) error {
	openCmdData, err := json.Marshal(cmd)
	if err != nil {
		return fmt.Errorf("failed to marshal open account command: %w", err)
	}

	command := dcb.NewCommand(CommandTypeOpenAccount, openCmdData, map[string]interface{}{
		"request_id": requestID,
		"source":     "web_api",
	})

	_, err = commandExecutor.ExecuteCommand(ctx, command, handler, nil)
	if err != nil {
		return fmt.Errorf("open account failed: %w", err)
	}

	fmt.Printf("✓ Opened account %s for %s with balance %d\n", cmd.AccountID, cmd.Owner, cmd.InitialBalance)
	return nil
}

// executeTransferCommand executes a transfer command and returns success message
func executeTransferCommand(ctx context.Context, commandExecutor dcb.CommandExecutor, handler dcb.CommandHandler, cmd TransferMoneyCommand, requestID string) error {
	transferCmdData, err := json.Marshal(cmd)
	if err != nil {
		return fmt.Errorf("failed to marshal transfer command: %w", err)
	}

	command := dcb.NewCommand(CommandTypeTransferMoney, transferCmdData, map[string]interface{}{
		"request_id": requestID,
		"source":     "web_api",
	})

	_, err = commandExecutor.ExecuteCommand(ctx, command, handler, nil)
	if err != nil {
		return fmt.Errorf("transfer failed: %w", err)
	}

	fmt.Printf("✓ Transferred %d from %s to %s (%s)\n", cmd.Amount, cmd.FromAccountID, cmd.ToAccountID, cmd.Description)
	return nil
}

// setupDatabase initializes the database connection and event store
func setupDatabase(ctx context.Context) (*pgxpool.Pool, dcb.EventStore, dcb.CommandExecutor, error) {
	// Get database URL from environment or use default
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://crablet:crablet@localhost:5432/crablet?sslmode=disable"
	}
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to connect to db: %w", err)
	}

	// Truncate events and commands tables before running the example
	_, err = pool.Exec(ctx, "TRUNCATE TABLE events, commands RESTART IDENTITY CASCADE")
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to truncate tables: %w", err)
	}

	store, err := dcb.NewEventStore(ctx, pool)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create event store: %w", err)
	}

	commandExecutor := dcb.NewCommandExecutor(store)
	return pool, store, commandExecutor, nil
}

// showFinalState displays the final events and commands
func showFinalState(ctx context.Context, pool *pgxpool.Pool) {
	fmt.Println("\n=== Final Account States ===")
	utils.DumpEvents(ctx, pool)

	fmt.Println("\n=== Commands Executed ===")
	rows, err := pool.Query(ctx, `
		SELECT transaction_id, type, data, metadata, occurred_at
		FROM commands
		ORDER BY occurred_at ASC
	`)
	if err != nil {
		log.Printf("Failed to query commands: %v", err)
		return
	}
	defer rows.Close()

	commandCount := 0
	for rows.Next() {
		var (
			txID        uint64
			cmdType     string
			cmdData     []byte
			cmdMetadata []byte
			occurredAt  time.Time
		)

		err := rows.Scan(&txID, &cmdType, &cmdData, &cmdMetadata, &occurredAt)
		if err != nil {
			log.Printf("Failed to scan command row: %v", err)
			continue
		}

		commandCount++
		fmt.Printf("  %d. Type: %s, Transaction: %d, At: %s\n",
			commandCount, cmdType, txID, occurredAt.Format("15:04:05.000"))
	}
	fmt.Printf("Total commands executed: %d\n", commandCount)
}

func main() {
	// Create context with timeout for the entire application
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Setup database and command executor
	pool, _, commandExecutor, err := setupDatabase(ctx)
	if err != nil {
		log.Fatalf("Setup failed: %v", err)
	}
	defer pool.Close()

	// Create command handler
	handler := dcb.CommandHandlerFunc(func(ctx context.Context, store dcb.EventStore, command dcb.Command) ([]dcb.InputEvent, error) {
		events, _, err := HandleCommand(ctx, store, command)
		return events, err
	})

	// Execute commands with early returns for failures
	fmt.Println("=== Opening First Account ===")
	openAccount1Cmd := OpenAccountCommand{
		AccountID:      "acc1",
		Owner:          "Alice",
		InitialBalance: 1000,
	}
	if err := executeOpenAccountCommand(ctx, commandExecutor, handler, openAccount1Cmd, "req_001"); err != nil {
		log.Fatalf("Open account 1 failed: %v", err)
	}

	fmt.Println("\n=== Opening Second Account ===")
	openAccount2Cmd := OpenAccountCommand{
		AccountID:      "acc456",
		Owner:          "Bob",
		InitialBalance: 500,
	}
	if err := executeOpenAccountCommand(ctx, commandExecutor, handler, openAccount2Cmd, "req_002"); err != nil {
		log.Fatalf("Open account 2 failed: %v", err)
	}

	fmt.Println("\n=== Transferring Money ===")
	transferCmd := TransferMoneyCommand{
		TransferID:    fmt.Sprintf("tx-%d", time.Now().UnixNano()),
		FromAccountID: "acc1",
		ToAccountID:   "acc456",
		Amount:        300,
		Description:   "First transfer",
	}
	if err := executeTransferCommand(ctx, commandExecutor, handler, transferCmd, "req_003"); err != nil {
		log.Fatalf("Transfer failed: %v", err)
	}

	fmt.Println("\n=== Transferring Money Back ===")
	transferBackCmd := TransferMoneyCommand{
		TransferID:    fmt.Sprintf("tx-%d", time.Now().UnixNano()),
		FromAccountID: "acc456",
		ToAccountID:   "acc1",
		Amount:        100,
		Description:   "Return transfer",
	}
	if err := executeTransferCommand(ctx, commandExecutor, handler, transferBackCmd, "req_004"); err != nil {
		log.Fatalf("Transfer back failed: %v", err)
	}

	// Test error cases
	fmt.Println("\n=== Attempting Invalid Transfer (Should Fail) ===")
	invalidTransferCmd := TransferMoneyCommand{
		TransferID:    fmt.Sprintf("tx-%d", time.Now().UnixNano()),
		FromAccountID: "acc456",
		ToAccountID:   "acc1",
		Amount:        1000, // More than available balance
		Description:   "Invalid transfer",
	}
	if err := executeTransferCommand(ctx, commandExecutor, handler, invalidTransferCmd, "req_005"); err != nil {
		fmt.Printf("✗ Expected failure: %v\n", err)
	} else {
		fmt.Println("✗ Unexpected success - transfer should have failed")
	}

	fmt.Println("\n=== Attempting Duplicate Account Opening (Should Fail) ===")
	duplicateAccountCmd := OpenAccountCommand{
		AccountID:      "acc1", // Same ID as first account
		Owner:          "Charlie",
		InitialBalance: 200,
	}
	if err := executeOpenAccountCommand(ctx, commandExecutor, handler, duplicateAccountCmd, "req_006"); err != nil {
		fmt.Printf("✗ Expected failure: %v\n", err)
	} else {
		fmt.Println("✗ Unexpected success - account opening should have failed")
	}

	showFinalState(ctx, pool)
	fmt.Println("\n=== Example Completed Successfully ===")
}
