// This example demonstrates command execution with the DCB pattern for account transfers
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
const (
	CommandTypeCreateAccount = "create_account"
	CommandTypeTransferMoney = "transfer_money"
)

type CreateAccountCommand struct {
	AccountID      string `json:"account_id"`
	Owner          string `json:"owner"`
	InitialBalance int    `json:"initial_balance"`
}

type TransferMoneyCommand struct {
	TransferID    string `json:"transfer_id"`
	FromAccountID string `json:"from_account_id"`
	ToAccountID   string `json:"to_account_id"`
	Amount        int    `json:"amount"`
	Description   string `json:"description,omitempty"`
}

// TransferCommandHandler implements CommandHandler interface
type TransferCommandHandler struct{}

func (h *TransferCommandHandler) Handle(ctx context.Context, store dcb.EventStore, command dcb.Command) []dcb.InputEvent {
	switch command.GetType() {
	case CommandTypeCreateAccount:
		return h.handleCreateAccount(command)
	case CommandTypeTransferMoney:
		return h.handleTransferMoney(ctx, store, command)
	default:
		log.Printf("Unknown command type: %s", command.GetType())
		return nil
	}
}

func (h *TransferCommandHandler) handleCreateAccount(command dcb.Command) []dcb.InputEvent {
	var cmd CreateAccountCommand
	if err := json.Unmarshal(command.GetData(), &cmd); err != nil {
		log.Printf("Failed to unmarshal create account command: %v", err)
		return nil
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
		log.Printf("Failed to marshal account opened event: %v", err)
		return nil
	}

	return []dcb.InputEvent{
		dcb.NewInputEvent("AccountOpened", []dcb.Tag{
			dcb.NewTag("account_id", cmd.AccountID),
		}, eventData),
	}
}

func (h *TransferCommandHandler) handleTransferMoney(ctx context.Context, store dcb.EventStore, command dcb.Command) []dcb.InputEvent {
	var cmd TransferMoneyCommand
	if err := json.Unmarshal(command.GetData(), &cmd); err != nil {
		log.Printf("Failed to unmarshal transfer money command: %v", err)
		return nil
	}

	// Define projectors for account states (DCB pattern)
	projectors := []dcb.StateProjector{
		{
			ID:           "fromAccount",
			Query:        dcb.NewQuery(dcb.NewTags("account_id", cmd.FromAccountID), "AccountOpened"),
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
				}
				return acc
			},
		},
		{
			ID:           "toAccount",
			Query:        dcb.NewQuery(dcb.NewTags("account_id", cmd.ToAccountID), "AccountOpened"),
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
				}
				return acc
			},
		},
		{
			ID:           "allTransfers",
			Query:        dcb.NewQuery(nil, "MoneyTransferred"),
			InitialState: []MoneyTransferred{},
			TransitionFn: func(state any, event dcb.Event) any {
				transfers := state.([]MoneyTransferred)
				if event.Type == "MoneyTransferred" {
					var data MoneyTransferred
					if err := json.Unmarshal(event.Data, &data); err == nil {
						transfers = append(transfers, data)
					}
				}
				return transfers
			},
		},
	}

	// Project the account states
	states, _, err := store.Project(ctx, projectors, nil)
	if err != nil {
		log.Printf("Failed to project account states: %v", err)
		return nil
	}

	fromAccount, fromOk := states["fromAccount"].(*AccountState)
	toAccount, toOk := states["toAccount"].(*AccountState)
	allTransfers, transfersOk := states["allTransfers"].([]MoneyTransferred)

	if !fromOk || !toOk || !transfersOk {
		log.Printf("Failed to get account states from projection")
		return nil
	}

	// Apply transfer history to calculate current balances
	for _, transfer := range allTransfers {
		if transfer.FromAccountID == cmd.FromAccountID {
			fromAccount.Balance = transfer.FromBalance
			fromAccount.UpdatedAt = transfer.TransferredAt
		} else if transfer.ToAccountID == cmd.FromAccountID {
			fromAccount.Balance = transfer.ToBalance
			fromAccount.UpdatedAt = transfer.TransferredAt
		}

		if transfer.FromAccountID == cmd.ToAccountID {
			toAccount.Balance = transfer.FromBalance
			toAccount.UpdatedAt = transfer.TransferredAt
		} else if transfer.ToAccountID == cmd.ToAccountID {
			toAccount.Balance = transfer.ToBalance
			toAccount.UpdatedAt = transfer.TransferredAt
		}
	}

	// Debug logging
	log.Printf("DEBUG: From account %s balance: %d, To account %s balance: %d, Transfer amount: %d",
		cmd.FromAccountID, fromAccount.Balance, cmd.ToAccountID, toAccount.Balance, cmd.Amount)

	// Business rule: check sufficient funds
	if fromAccount.Balance < cmd.Amount {
		log.Printf("Insufficient funds: account %s has %d, needs %d", cmd.FromAccountID, fromAccount.Balance, cmd.Amount)
		return nil
	}

	// Calculate new balances
	newFromBalance := fromAccount.Balance - cmd.Amount
	newToBalance := toAccount.Balance + cmd.Amount

	// Create the event
	event := MoneyTransferred{
		TransferID:    cmd.TransferID,
		FromAccountID: cmd.FromAccountID,
		ToAccountID:   cmd.ToAccountID,
		Amount:        cmd.Amount,
		FromBalance:   newFromBalance,
		ToBalance:     newToBalance,
		TransferredAt: time.Now(),
		Description:   cmd.Description,
	}

	eventData, err := json.Marshal(event)
	if err != nil {
		log.Printf("Failed to marshal money transferred event: %v", err)
		return nil
	}

	return []dcb.InputEvent{
		dcb.NewInputEvent("MoneyTransferred", []dcb.Tag{
			dcb.NewTag("transfer_id", cmd.TransferID),
			dcb.NewTag("from_account_id", cmd.FromAccountID),
			dcb.NewTag("to_account_id", cmd.ToAccountID),
		}, eventData),
	}
}

func main() {
	// Create context with timeout for the entire application
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, "postgres://postgres:postgres@localhost:5432/dcb_app?sslmode=disable")
	if err != nil {
		log.Fatalf("failed to connect to db: %v", err)
	}

	// Truncate events and commands tables before running the example
	_, err = pool.Exec(ctx, "TRUNCATE TABLE events, commands RESTART IDENTITY CASCADE")
	if err != nil {
		log.Fatalf("failed to truncate tables: %v", err)
	}

	store, err := dcb.NewEventStore(ctx, pool)
	if err != nil {
		log.Fatalf("failed to create event store: %v", err)
	}

	// Create command executor
	commandExecutor := dcb.NewCommandExecutor(store)

	// Create command handler
	handler := &TransferCommandHandler{}

	// Command 1: Create first account
	fmt.Println("=== Creating First Account ===")
	createAccount1Cmd := CreateAccountCommand{
		AccountID:      "acc1",
		Owner:          "Alice",
		InitialBalance: 1000,
	}

	createCmdData1, err := json.Marshal(createAccount1Cmd)
	if err != nil {
		log.Fatalf("Failed to marshal create account command: %v", err)
	}

	command1 := dcb.NewCommand(CommandTypeCreateAccount, createCmdData1, map[string]interface{}{
		"request_id": "req_001",
		"source":     "web_api",
	})

	err = commandExecutor.ExecuteCommand(ctx, command1, handler, nil)
	if err != nil {
		log.Fatalf("Create account 1 failed: %v", err)
	}
	fmt.Printf("✓ Created account %s for %s with balance %d\n", createAccount1Cmd.AccountID, createAccount1Cmd.Owner, createAccount1Cmd.InitialBalance)

	// Command 2: Create second account
	fmt.Println("\n=== Creating Second Account ===")
	createAccount2Cmd := CreateAccountCommand{
		AccountID:      "acc456",
		Owner:          "Bob",
		InitialBalance: 500,
	}

	createCmdData2, err := json.Marshal(createAccount2Cmd)
	if err != nil {
		log.Fatalf("Failed to marshal create account command: %v", err)
	}

	command2 := dcb.NewCommand(CommandTypeCreateAccount, createCmdData2, map[string]interface{}{
		"request_id": "req_002",
		"source":     "web_api",
	})

	err = commandExecutor.ExecuteCommand(ctx, command2, handler, nil)
	if err != nil {
		log.Fatalf("Create account 2 failed: %v", err)
	}
	fmt.Printf("✓ Created account %s for %s with balance %d\n", createAccount2Cmd.AccountID, createAccount2Cmd.Owner, createAccount2Cmd.InitialBalance)

	// Command 3: Transfer money
	fmt.Println("\n=== Transferring Money ===")
	transferCmd := TransferMoneyCommand{
		TransferID:    fmt.Sprintf("tx-%d", time.Now().UnixNano()),
		FromAccountID: "acc1",
		ToAccountID:   "acc456",
		Amount:        300,
		Description:   "First transfer",
	}

	transferCmdData, err := json.Marshal(transferCmd)
	if err != nil {
		log.Fatalf("Failed to marshal transfer command: %v", err)
	}

	command3 := dcb.NewCommand(CommandTypeTransferMoney, transferCmdData, map[string]interface{}{
		"request_id": "req_003",
		"source":     "web_api",
	})

	err = commandExecutor.ExecuteCommand(ctx, command3, handler, nil)
	if err != nil {
		fmt.Printf("❌ Transfer failed: %v\n", err)
	} else {
		fmt.Printf("✓ Transfer successful! Transfer ID: %s\n", transferCmd.TransferID)
	}

	// Second transfer (should fail due to insufficient funds)
	fmt.Println("\n=== Attempting Second Transfer (should fail due to insufficient funds) ===")
	secondTransferCmd := TransferMoneyCommand{
		TransferID:    fmt.Sprintf("tx-%d", time.Now().UnixNano()),
		FromAccountID: "acc1",
		ToAccountID:   "acc456",
		Amount:        800, // This should fail - only 700 left
		Description:   "Second transfer - should fail",
	}

	secondTransferCmdData, err := json.Marshal(secondTransferCmd)
	if err != nil {
		log.Fatalf("Failed to marshal second transfer command: %v", err)
	}

	command4 := dcb.NewCommand(CommandTypeTransferMoney, secondTransferCmdData, map[string]interface{}{
		"request_id": "req_004",
		"source":     "web_api",
	})

	err = commandExecutor.ExecuteCommand(ctx, command4, handler, nil)
	if err != nil {
		fmt.Printf("❌ Second transfer failed (expected): %v\n", err)
	} else {
		fmt.Printf("✓ Second transfer succeeded (unexpected)! Transfer ID: %s\n", secondTransferCmd.TransferID)
	}

	// Show final state
	fmt.Println("\n=== Final Account States ===")
	utils.DumpEvents(ctx, pool)

	// Show commands that were executed
	fmt.Println("\n=== Commands Executed ===")
	rows, err := pool.Query(ctx, `
		SELECT transaction_id, type, data, metadata, occurred_at
		FROM commands
		ORDER BY occurred_at ASC
	`)
	if err != nil {
		log.Printf("Failed to query commands: %v", err)
	} else {
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
}
