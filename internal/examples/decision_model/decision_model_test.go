package main

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"go-crablet/pkg/dcb"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecisionModelExample(t *testing.T) {
	ctx := context.Background()

	// Connect to test database
	pool, err := pgxpool.New(ctx, "postgres://postgres:postgres@localhost:5432/dcb_app?sslmode=disable")
	require.NoError(t, err)
	defer pool.Close()

	// Create event store
	store, err := dcb.NewEventStore(ctx, pool)
	require.NoError(t, err)

	// Cast to EventStore for extended functionality
	// Use store directly

	// Generate unique test ID to avoid conflicts
	testID := fmt.Sprintf("test_%d", time.Now().UnixNano())

	// Test Command 1: Open Account
	t.Run("Open Account", func(t *testing.T) {
		openAccountCmd := OpenAccountCommand{
			AccountID:      fmt.Sprintf("test_acc_decision_%s", testID),
			InitialBalance: 1000,
		}
		err := handleOpenAccount(ctx, store, openAccountCmd)
		assert.NoError(t, err)
	})

	// Test Command 2: Process Transaction
	t.Run("Process Transaction", func(t *testing.T) {
		processTransactionCmd := ProcessTransactionCommand{
			AccountID: fmt.Sprintf("test_acc_decision_%s", testID),
			Amount:    500,
		}
		err := handleProcessTransaction(ctx, store, processTransactionCmd)
		assert.NoError(t, err)
	})

	// Test business rules
	t.Run("Business Rules", func(t *testing.T) {
		// Test: Cannot open account with same ID
		t.Run("Cannot Open Duplicate Account", func(t *testing.T) {
			duplicateCmd := OpenAccountCommand{
				AccountID:      fmt.Sprintf("test_acc_decision_%s", testID), // Same ID as existing account
				InitialBalance: 2000,
			}
			err := handleOpenAccount(ctx, store, duplicateCmd)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "already exists")
		})

		// Test: Cannot process transaction for non-existent account
		t.Run("Cannot Process Transaction for Non-existent Account", func(t *testing.T) {
			nonExistentAccountCmd := ProcessTransactionCommand{
				AccountID: fmt.Sprintf("non_existent_account_%s", testID),
				Amount:    100,
			}
			err := handleProcessTransaction(ctx, store, nonExistentAccountCmd)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "does not exist")
		})
	})

	// Test decision model projection
	t.Run("Decision Model Projection", func(t *testing.T) {
		accountID := fmt.Sprintf("test_acc_decision_%s", testID)

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
		projectors := []dcb.StateProjector{
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

		states, _, err := store.Project(ctx, projectors, nil)
		require.NoError(t, err)

		// Verify account state
		if account, ok := states["account"].(*AccountState); ok {
			assert.Equal(t, accountID, account.ID)
			assert.Equal(t, 1000, account.Balance) // Initial balance
		}

		// Verify transaction state
		if transactions, ok := states["transactions"].(*TransactionState); ok {
			assert.Equal(t, 1, transactions.Count) // One transaction processed
			assert.Equal(t, 500, transactions.TotalAmount)
		}

		// Test optimistic locking
		t.Run("Optimistic Locking", func(t *testing.T) {
			// Get current append condition for optimistic locking
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

			_, appendCondition, err := store.Project(ctx, []dcb.StateProjector{{
				ID:           "account",
				Query:        accountProjector.Query,
				InitialState: accountProjector.InitialState,
				TransitionFn: accountProjector.TransitionFn,
			}}, nil)
			require.NoError(t, err)

			// Test optimistic locking with append condition
			optimisticCmd := ProcessTransactionCommand{
				AccountID: accountID,
				Amount:    200,
			}
			err = handleProcessTransactionWithCondition(ctx, store, optimisticCmd, appendCondition)
			assert.NoError(t, err)
		})
	})
}
