// This example demonstrates command execution with the DCB pattern for account transfers
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
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
	CommandTypeCreateAccount = "create_account"
	CommandTypeTransferMoney = "transfer_money"
)

// CreateAccountCommand represents a command to create an account
type CreateAccountCommand struct {
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

// HandleCreateAccount handles the creation of an account
func HandleCreateAccount(ctx context.Context, store dcb.EventStore, command dcb.Command) ([]dcb.InputEvent, *dcb.AppendCondition, error) {
	var cmd CreateAccountCommand
	if err := json.Unmarshal(command.GetData(), &cmd); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal create account command: %w", err)
	}

	// Check for duplicate account
	query := dcb.NewQuery(dcb.NewTags("account_id", cmd.AccountID), "AccountOpened")
	events, err := store.Query(ctx, query, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query for existing account: %w", err)
	}
	if len(events) > 0 {
		return nil, nil, fmt.Errorf("account %s already exists", cmd.AccountID)
	}

	// Create the event
	event := AccountOpened{
		AccountID:      cmd.AccountID,
		Owner:          cmd.Owner,
		InitialBalance: cmd.InitialBalance,
		OpenedAt:       time.Now(),
	}
	eventData, err := json.Marshal(event)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal account opened event: %w", err)
	}

	return []dcb.InputEvent{
		dcb.NewInputEvent("AccountOpened", []dcb.Tag{
			dcb.NewTag("account_id", cmd.AccountID),
		}, eventData),
	}, nil, nil
}

// HandleTransferMoney handles money transfers between accounts
func HandleTransferMoney(ctx context.Context, store dcb.EventStore, command dcb.Command) ([]dcb.InputEvent, *dcb.AppendCondition, error) {
	var cmd TransferMoneyCommand
	if err := json.Unmarshal(command.GetData(), &cmd); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal transfer command: %w", err)
	}

	// Validate transfer amount
	if cmd.Amount <= 0 {
		return nil, nil, fmt.Errorf("transfer amount must be positive")
	}

	// Project current state for both accounts - the projection system automatically handles balance calculation
	projectors := []dcb.StateProjector{
		{
			ID: "fromAccount",
			Query: dcb.NewQuery(dcb.NewTags("account_id", cmd.FromAccountID), "AccountOpened", "MoneyTransferred"),
			InitialState: AccountState{},
			TransitionFn: func(state any, event dcb.Event) any {
				if event.Type == "AccountOpened" {
					var accountOpened AccountOpened
					if err := json.Unmarshal(event.Data, &accountOpened); err != nil {
						return state
					}
					return AccountState{
						AccountID: accountOpened.AccountID,
						Owner:     accountOpened.Owner,
						Balance:   accountOpened.InitialBalance,
						CreatedAt: accountOpened.OpenedAt,
						UpdatedAt: accountOpened.OpenedAt,
					}
				}
				if event.Type == "MoneyTransferred" {
					var transfer MoneyTransferred
					if err := json.Unmarshal(event.Data, &transfer); err != nil {
						return state
					}
					if current, ok := state.(AccountState); ok {
						if transfer.FromAccountID == current.AccountID {
							current.Balance = transfer.FromBalance
							current.UpdatedAt = transfer.TransferredAt
						} else if transfer.ToAccountID == current.AccountID {
							current.Balance = transfer.ToBalance
							current.UpdatedAt = transfer.TransferredAt
						}
						return current
					}
				}
				return state
			},
		},
		{
			ID: "toAccount",
			Query: dcb.NewQuery(dcb.NewTags("account_id", cmd.ToAccountID), "AccountOpened", "MoneyTransferred"),
			InitialState: AccountState{},
			TransitionFn: func(state any, event dcb.Event) any {
				if event.Type == "AccountOpened" {
					var accountOpened AccountOpened
					if err := json.Unmarshal(event.Data, &accountOpened); err != nil {
						return state
					}
					return AccountState{
						AccountID: accountOpened.AccountID,
						Owner:     accountOpened.Owner,
						Balance:   accountOpened.InitialBalance,
						CreatedAt: accountOpened.OpenedAt,
						UpdatedAt: accountOpened.OpenedAt,
					}
				}
				if event.Type == "MoneyTransferred" {
					var transfer MoneyTransferred
					if err := json.Unmarshal(event.Data, &transfer); err != nil {
						return state
					}
					if current, ok := state.(AccountState); ok {
						if transfer.FromAccountID == current.AccountID {
							current.Balance = transfer.FromBalance
							current.UpdatedAt = transfer.TransferredAt
						} else if transfer.ToAccountID == current.AccountID {
							current.Balance = transfer.ToBalance
							current.UpdatedAt = transfer.TransferredAt
						}
						return current
					}
				}
				return state
			},
		},
	}

	states, _, err := store.Project(ctx, projectors, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to project account states: %w", err)
	}

	fromAccount := states["fromAccount"].(AccountState)
	toAccount := states["toAccount"].(AccountState)

	// Validate source account exists and has sufficient funds
	if fromAccount.Owner == "" {
		return nil, nil, fmt.Errorf("source account %s does not exist", cmd.FromAccountID)
	}
	if fromAccount.Balance < cmd.Amount {
		return nil, nil, fmt.Errorf("insufficient funds in account %s (balance: %d, required: %d)", cmd.FromAccountID, fromAccount.Balance, cmd.Amount)
	}

	// Calculate new balances
	newFromBalance := fromAccount.Balance - cmd.Amount
	newToBalance := toAccount.Balance + cmd.Amount

	// Create the transfer event
	transferEvent := MoneyTransferred{
		TransferID:    cmd.TransferID,
		FromAccountID: cmd.FromAccountID,
		ToAccountID:   cmd.ToAccountID,
		Amount:        cmd.Amount,
		FromBalance:   newFromBalance,
		ToBalance:     newToBalance,
		TransferredAt: time.Now(),
		Description:   cmd.Description,
	}
	eventData, err := json.Marshal(transferEvent)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal money transferred event: %w", err)
	}

	// Create AppendCondition to ensure source account hasn't changed since our projection
	// This prevents race conditions where multiple transfers could succeed
	item := dcb.NewQueryItem([]string{"AccountOpened", "MoneyTransferred"}, []dcb.Tag{dcb.NewTag("account_id", cmd.FromAccountID)})
	query := dcb.NewQueryFromItems(item)
	appendCondition := dcb.NewAppendCondition(query)

	return []dcb.InputEvent{
		dcb.NewInputEvent("MoneyTransferred", []dcb.Tag{
			dcb.NewTag("transfer_id", cmd.TransferID),
			dcb.NewTag("from_account_id", cmd.FromAccountID),
			dcb.NewTag("to_account_id", cmd.ToAccountID),
		}, eventData),
	}, &appendCondition, nil
}

// HandleCommand routes commands to appropriate handlers
func HandleCommand(ctx context.Context, store dcb.EventStore, command dcb.Command) ([]dcb.InputEvent, *dcb.AppendCondition, error) {
	switch command.GetType() {
	case CommandTypeCreateAccount:
		return HandleCreateAccount(ctx, store, command)
	case CommandTypeTransferMoney:
		return HandleTransferMoney(ctx, store, command)
	default:
		return nil, nil, fmt.Errorf("unknown command type: %s", command.GetType())
	}
}

// Helper functions for flatter code structure

// executeCreateAccountCommand executes a create account command and returns success message
func executeCreateAccountCommand(ctx context.Context, commandExecutor dcb.CommandExecutor, handler dcb.CommandHandler, cmd CreateAccountCommand, requestID string) error {
	createCmdData, err := json.Marshal(cmd)
	if err != nil {
		return fmt.Errorf("failed to marshal create account command: %w", err)
	}

	command := dcb.NewCommand(CommandTypeCreateAccount, createCmdData, map[string]interface{}{
		"request_id": requestID,
		"source":     "web_api",
	})

	_, err = commandExecutor.ExecuteCommand(ctx, command, handler, nil)
	if err != nil {
		return fmt.Errorf("create account failed: %w", err)
	}

	fmt.Printf("✓ Created account %s for %s with balance %d\n", cmd.AccountID, cmd.Owner, cmd.InitialBalance)
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
	pool, err := pgxpool.New(ctx, "postgres://crablet:crablet@localhost:5432/crablet?sslmode=disable")
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
	fmt.Println("=== Creating First Account ===")
	createAccount1Cmd := CreateAccountCommand{
		AccountID:      "acc1",
		Owner:          "Alice",
		InitialBalance: 1000,
	}
	if err := executeCreateAccountCommand(ctx, commandExecutor, handler, createAccount1Cmd, "req_001"); err != nil {
		log.Fatalf("Create account 1 failed: %v", err)
	}

	fmt.Println("\n=== Creating Second Account ===")
	createAccount2Cmd := CreateAccountCommand{
		AccountID:      "acc456",
		Owner:          "Bob",
		InitialBalance: 500,
	}
	if err := executeCreateAccountCommand(ctx, commandExecutor, handler, createAccount2Cmd, "req_002"); err != nil {
		log.Fatalf("Create account 2 failed: %v", err)
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

	fmt.Println("\n=== Attempting Duplicate Account Creation (Should Fail) ===")
	duplicateAccountCmd := CreateAccountCommand{
		AccountID:      "acc1", // Same ID as first account
		Owner:          "Charlie",
		InitialBalance: 200,
	}
	if err := executeCreateAccountCommand(ctx, commandExecutor, handler, duplicateAccountCmd, "req_006"); err != nil {
		fmt.Printf("✗ Expected failure: %v\n", err)
	} else {
		fmt.Println("✗ Unexpected success - account creation should have failed")
	}

	showFinalState(ctx, pool)
	fmt.Println("\n=== Example Completed Successfully ===")
}
