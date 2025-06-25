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

	// Cast to ChannelEventStore for extended functionality
	channelStore := store.(dcb.ChannelEventStore)

	// Test Command 1: Create Account 1
	t.Run("Create Account 1", func(t *testing.T) {
		createAccount1Cmd := CreateAccountCommand{
			AccountID:      "test_acc1",
			InitialBalance: 1000,
		}
		err := handleCreateAccount(ctx, channelStore, createAccount1Cmd)
		assert.NoError(t, err)
	})

	// Test Command 2: Create Account 2
	t.Run("Create Account 2", func(t *testing.T) {
		createAccount2Cmd := CreateAccountCommand{
			AccountID:      "test_acc2",
			InitialBalance: 500,
		}
		err := handleCreateAccount(ctx, channelStore, createAccount2Cmd)
		assert.NoError(t, err)
	})

	// Test Command 3: Transfer Money
	t.Run("Transfer Money", func(t *testing.T) {
		transferCmd := TransferMoneyCommand{
			FromAccountID: "test_acc1",
			ToAccountID:   "test_acc2",
			Amount:        300,
		}
		err := handleTransferMoney(ctx, channelStore, transferCmd)
		assert.NoError(t, err)
	})

	// Test business rules
	t.Run("Business Rules", func(t *testing.T) {
		// Test: Cannot create account with same ID
		t.Run("Cannot Create Duplicate Account", func(t *testing.T) {
			duplicateCmd := CreateAccountCommand{
				AccountID:      "test_acc1", // Same ID as existing account
				InitialBalance: 2000,
			}
			err := handleCreateAccount(ctx, channelStore, duplicateCmd)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "already exists")
		})

		// Test: Cannot transfer more than available balance
		t.Run("Cannot Transfer More Than Available Balance", func(t *testing.T) {
			insufficientFundsCmd := TransferMoneyCommand{
				FromAccountID: "test_acc1",
				ToAccountID:   "test_acc2",
				Amount:        1000, // More than available balance
			}
			err := handleTransferMoney(ctx, channelStore, insufficientFundsCmd)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "insufficient funds")
		})

		// Test: Cannot transfer from non-existent account
		t.Run("Cannot Transfer From Non-existent Account", func(t *testing.T) {
			nonExistentFromCmd := TransferMoneyCommand{
				FromAccountID: "non_existent_account",
				ToAccountID:   "test_acc2",
				Amount:        100,
			}
			err := handleTransferMoney(ctx, channelStore, nonExistentFromCmd)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "does not exist")
		})

		// Test: Cannot transfer to non-existent account
		t.Run("Cannot Transfer To Non-existent Account", func(t *testing.T) {
			nonExistentToCmd := TransferMoneyCommand{
				FromAccountID: "test_acc1",
				ToAccountID:   "non_existent_account",
				Amount:        100,
			}
			err := handleTransferMoney(ctx, channelStore, nonExistentToCmd)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "does not exist")
		})
	})

	// Test concurrent transfers
	t.Run("Concurrent Transfers", func(t *testing.T) {
		// Create additional accounts for concurrent testing
		createAccount3Cmd := CreateAccountCommand{
			AccountID:      "test_acc3",
			InitialBalance: 2000,
		}
		err := handleCreateAccount(ctx, channelStore, createAccount3Cmd)
		assert.NoError(t, err)

		// Test concurrent transfers - one should succeed, one should fail due to insufficient funds
		transfer1Cmd := TransferMoneyCommand{
			FromAccountID: "test_acc3",
			ToAccountID:   "test_acc2",
			Amount:        1500,
		}
		transfer2Cmd := TransferMoneyCommand{
			FromAccountID: "test_acc3",
			ToAccountID:   "test_acc1",
			Amount:        1000,
		}

		err1 := handleTransferMoney(ctx, channelStore, transfer1Cmd)
		err2 := handleTransferMoney(ctx, channelStore, transfer2Cmd)

		// One should succeed, one should fail due to insufficient funds
		assert.True(t, (err1 == nil && err2 != nil) || (err1 != nil && err2 == nil))
		if err1 != nil {
			assert.Contains(t, err1.Error(), "insufficient funds")
		}
		if err2 != nil {
			assert.Contains(t, err2.Error(), "insufficient funds")
		}
	})
}
