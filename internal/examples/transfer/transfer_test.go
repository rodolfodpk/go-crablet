package main

import (
	"context"
	"testing"

	"go-crablet/pkg/dcb"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransferExample(t *testing.T) {
	ctx := context.Background()

	// Connect to test database
	pool, err := pgxpool.New(ctx, "postgres://postgres:postgres@localhost:5432/dcb_app?sslmode=disable")
	require.NoError(t, err)
	defer pool.Close()

	// Create event store
	store, err := dcb.NewEventStore(ctx, pool)
	require.NoError(t, err)

	// Test Command 1: Create first account
	t.Run("Create Account 1", func(t *testing.T) {
		createAccount1Cmd := CreateAccountCommand{
			AccountID:      "test_acc1",
			Owner:          "Alice",
			InitialBalance: 1000,
		}
		err := handleCreateAccount(ctx, store, createAccount1Cmd)
		assert.NoError(t, err)
	})

	// Test Command 2: Create second account
	t.Run("Create Account 2", func(t *testing.T) {
		createAccount2Cmd := CreateAccountCommand{
			AccountID:      "test_acc2",
			Owner:          "Bob",
			InitialBalance: 500,
		}
		err := handleCreateAccount(ctx, store, createAccount2Cmd)
		assert.NoError(t, err)
	})

	// Test Command 3: Transfer money between accounts
	t.Run("Transfer Money", func(t *testing.T) {
		transferCmd := TransferMoneyCommand{
			TransferID:    "test_transfer_123",
			FromAccountID: "test_acc1",
			ToAccountID:   "test_acc2",
			Amount:        150,
			Description:   "Test payment",
		}
		err := handleTransferMoney(ctx, store, transferCmd)
		assert.NoError(t, err)
	})

	// Test business rules
	t.Run("Business Rules", func(t *testing.T) {
		// Test: Cannot create account with same ID
		t.Run("Cannot Create Duplicate Account", func(t *testing.T) {
			duplicateCmd := CreateAccountCommand{
				AccountID:      "test_acc1", // Same ID as existing account
				Owner:          "Charlie",
				InitialBalance: 2000,
			}
			err := handleCreateAccount(ctx, store, duplicateCmd)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "already exists")
		})

		// Test: Cannot transfer more than available balance
		t.Run("Cannot Transfer Insufficient Funds", func(t *testing.T) {
			insufficientCmd := TransferMoneyCommand{
				TransferID:    "test_transfer_456",
				FromAccountID: "test_acc1",
				ToAccountID:   "test_acc2",
				Amount:        2000, // More than available balance
				Description:   "Should fail",
			}
			err := handleTransferMoney(ctx, store, insufficientCmd)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "insufficient funds")
		})

		// Test: Cannot transfer to same account
		t.Run("Cannot Transfer to Same Account", func(t *testing.T) {
			sameAccountCmd := TransferMoneyCommand{
				TransferID:    "test_transfer_789",
				FromAccountID: "test_acc1",
				ToAccountID:   "test_acc1", // Same account
				Amount:        100,
				Description:   "Should fail",
			}
			err := handleTransferMoney(ctx, store, sameAccountCmd)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "same account")
		})

		// Test: Cannot transfer to non-existent account
		t.Run("Cannot Transfer to Non-existent Account", func(t *testing.T) {
			nonExistentCmd := TransferMoneyCommand{
				TransferID:    "test_transfer_999",
				FromAccountID: "test_acc1",
				ToAccountID:   "non_existent_account",
				Amount:        100,
				Description:   "Should fail",
			}
			err := handleTransferMoney(ctx, store, nonExistentCmd)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "does not exist")
		})
	})

	// Test optimistic locking
	t.Run("Optimistic Locking", func(t *testing.T) {
		// Create a new account for concurrent access testing
		concurrentAccountCmd := CreateAccountCommand{
			AccountID:      "concurrent_acc",
			Owner:          "David",
			InitialBalance: 1000,
		}
		err := handleCreateAccount(ctx, store, concurrentAccountCmd)
		require.NoError(t, err)

		// Simulate concurrent transfers
		transfer1Cmd := TransferMoneyCommand{
			TransferID:    "concurrent_transfer_1",
			FromAccountID: "concurrent_acc",
			ToAccountID:   "test_acc2",
			Amount:        100,
			Description:   "Concurrent transfer 1",
		}

		transfer2Cmd := TransferMoneyCommand{
			TransferID:    "concurrent_transfer_2",
			FromAccountID: "concurrent_acc",
			ToAccountID:   "test_acc2",
			Amount:        200,
			Description:   "Concurrent transfer 2",
		}

		// Both should succeed due to optimistic locking
		err1 := handleTransferMoney(ctx, store, transfer1Cmd)
		err2 := handleTransferMoney(ctx, store, transfer2Cmd)

		assert.NoError(t, err1)
		assert.NoError(t, err2)
	})
}
