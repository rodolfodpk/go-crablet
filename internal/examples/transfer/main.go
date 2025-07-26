// This example demonstrates command execution with the DCB pattern for account transfers
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	transferpkg "github.com/rodolfodpk/go-crablet/internal/examples/transfer/pkg"
	"github.com/rodolfodpk/go-crablet/internal/examples/utils"
	"github.com/rodolfodpk/go-crablet/pkg/dcb"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	// Create context with timeout for the entire application
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, "postgres://crablet:crablet@localhost:5432/crablet?sslmode=disable")
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
	handler := dcb.CommandHandlerFunc(func(ctx context.Context, store dcb.EventStore, command dcb.Command) ([]dcb.InputEvent, error) {
		events, _, err := transferpkg.HandleCommand(ctx, store, command)
		return events, err
	})

	// Command 1: Create first account
	fmt.Println("=== Creating First Account ===")
	createAccount1Cmd := transferpkg.CreateAccountCommand{
		AccountID:      "acc1",
		Owner:          "Alice",
		InitialBalance: 1000,
	}

	createCmdData1, err := json.Marshal(createAccount1Cmd)
	if err != nil {
		log.Fatalf("Failed to marshal create account command: %v", err)
	}

	command1 := dcb.NewCommand(transferpkg.CommandTypeCreateAccount, createCmdData1, map[string]interface{}{
		"request_id": "req_001",
		"source":     "web_api",
	})

	_, err = commandExecutor.ExecuteCommand(ctx, command1, handler, nil)
	if err != nil {
		log.Fatalf("Create account 1 failed: %v", err)
	}
	fmt.Printf("✓ Created account %s for %s with balance %d\n", createAccount1Cmd.AccountID, createAccount1Cmd.Owner, createAccount1Cmd.InitialBalance)

	// Command 2: Create second account
	fmt.Println("\n=== Creating Second Account ===")
	createAccount2Cmd := transferpkg.CreateAccountCommand{
		AccountID:      "acc456",
		Owner:          "Bob",
		InitialBalance: 500,
	}

	createCmdData2, err := json.Marshal(createAccount2Cmd)
	if err != nil {
		log.Fatalf("Failed to marshal create account command: %v", err)
	}

	command2 := dcb.NewCommand(transferpkg.CommandTypeCreateAccount, createCmdData2, map[string]interface{}{
		"request_id": "req_002",
		"source":     "web_api",
	})

	_, err = commandExecutor.ExecuteCommand(ctx, command2, handler, nil)
	if err != nil {
		log.Fatalf("Create account 2 failed: %v", err)
	}
	fmt.Printf("✓ Created account %s for %s with balance %d\n", createAccount2Cmd.AccountID, createAccount2Cmd.Owner, createAccount2Cmd.InitialBalance)

	// Command 3: Transfer money
	fmt.Println("\n=== Transferring Money ===")
	transferCmd := transferpkg.TransferMoneyCommand{
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

	command3 := dcb.NewCommand(transferpkg.CommandTypeTransferMoney, transferCmdData, map[string]interface{}{
		"request_id": "req_003",
		"source":     "web_api",
	})

	// Use the DCB-compliant handler to get events and append condition
	_, appendCondition, err := transferpkg.HandleCommand(ctx, store, command3)
	if err != nil {
		fmt.Printf("❌ Transfer failed: %v\n", err)
	} else {
		_, err = commandExecutor.ExecuteCommand(ctx, command3, handler, appendCondition)
		if err != nil {
			fmt.Printf("❌ Transfer failed: %v\n", err)
		} else {
			fmt.Printf("✓ Transfer successful! Transfer ID: %s\n", transferCmd.TransferID)
		}
	}

	// Second transfer (should fail due to insufficient funds)
	fmt.Println("\n=== Attempting Second Transfer (should fail due to insufficient funds) ===")
	secondTransferCmd := transferpkg.TransferMoneyCommand{
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

	command4 := dcb.NewCommand(transferpkg.CommandTypeTransferMoney, secondTransferCmdData, map[string]interface{}{
		"request_id": "req_004",
		"source":     "web_api",
	})

	_, err = commandExecutor.ExecuteCommand(ctx, command4, handler, nil)
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
