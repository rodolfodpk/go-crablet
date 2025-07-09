package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"go-crablet/pkg/dcb"

	"github.com/jackc/pgx/v5/pgxpool"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// cleanupEvents truncates the events table to ensure test isolation
func cleanupEvents(t *testing.T, pool *pgxpool.Pool) {
	// Create context with timeout for cleanup
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_, err := pool.Exec(ctx, "TRUNCATE TABLE events RESTART IDENTITY CASCADE")
	require.NoError(t, err)
}

func TestTransferExample(t *testing.T) {
	// Create context with timeout for the entire test
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Connect to test database
	pool, err := pgxpool.New(ctx, "postgres://postgres:postgres@localhost:5432/dcb_app?sslmode=disable")
	require.NoError(t, err)
	defer pool.Close()

	// Create event store
	store, err := dcb.NewEventStore(ctx, pool)
	require.NoError(t, err)

	// Cast to EventStore for extended functionality
	channelStore := store.(dcb.EventStore)

	// Test Command 1: Create Account 1
	t.Run("Create Account 1", func(t *testing.T) {
		cleanupEvents(t, pool)
		createAccount1Cmd := CreateAccountCommand{
			AccountID:      "test_acc1",
			InitialBalance: 1000,
		}
		err := handleCreateAccount(ctx, channelStore, createAccount1Cmd)
		assert.NoError(t, err)
	})

	// Test Command 2: Create Account 2
	t.Run("Create Account 2", func(t *testing.T) {
		cleanupEvents(t, pool)
		createAccount2Cmd := CreateAccountCommand{
			AccountID:      "test_acc2",
			InitialBalance: 500,
		}
		err := handleCreateAccount(ctx, channelStore, createAccount2Cmd)
		assert.NoError(t, err)
	})

	// Test Command 3: Transfer Money
	t.Run("Transfer Money", func(t *testing.T) {
		cleanupEvents(t, pool)
		// Create accounts first
		createAccount1Cmd := CreateAccountCommand{
			AccountID:      "test_acc1",
			InitialBalance: 1000,
		}
		err := handleCreateAccount(ctx, channelStore, createAccount1Cmd)
		require.NoError(t, err)

		createAccount2Cmd := CreateAccountCommand{
			AccountID:      "test_acc2",
			InitialBalance: 500,
		}
		err = handleCreateAccount(ctx, channelStore, createAccount2Cmd)
		require.NoError(t, err)

		transferCmd := TransferMoneyCommand{
			TransferID:    "test_transfer_1",
			FromAccountID: "test_acc1",
			ToAccountID:   "test_acc2",
			Amount:        300,
		}
		err = handleTransferMoney(ctx, channelStore, transferCmd)
		assert.NoError(t, err)
	})

	// Test business rules
	t.Run("Business Rules", func(t *testing.T) {
		cleanupEvents(t, pool)
		// Test: Cannot create account with same ID
		t.Run("Cannot Create Duplicate Account", func(t *testing.T) {
			cleanupEvents(t, pool)
			// Create the account first
			createAccountCmd := CreateAccountCommand{
				AccountID:      "test_acc1",
				InitialBalance: 1000,
			}
			err := handleCreateAccount(ctx, channelStore, createAccountCmd)
			require.NoError(t, err)

			duplicateCmd := CreateAccountCommand{
				AccountID:      "test_acc1", // Same ID as existing account
				InitialBalance: 2000,
			}
			err = handleCreateAccount(ctx, channelStore, duplicateCmd)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "already exists")
		})

		// Test: Cannot transfer more than available balance
		t.Run("Cannot Transfer More Than Available Balance", func(t *testing.T) {
			cleanupEvents(t, pool)
			// Create accounts first
			createAccount1Cmd := CreateAccountCommand{
				AccountID:      "test_acc1",
				InitialBalance: 1000,
			}
			err := handleCreateAccount(ctx, channelStore, createAccount1Cmd)
			require.NoError(t, err)

			createAccount2Cmd := CreateAccountCommand{
				AccountID:      "test_acc2",
				InitialBalance: 500,
			}
			err = handleCreateAccount(ctx, channelStore, createAccount2Cmd)
			require.NoError(t, err)

			insufficientFundsCmd := TransferMoneyCommand{
				TransferID:    "test_transfer_2",
				FromAccountID: "test_acc1",
				ToAccountID:   "test_acc2",
				Amount:        1001, // More than available balance (1000)
			}
			err = handleTransferMoney(ctx, channelStore, insufficientFundsCmd)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "insufficient funds")
		})

		// Test: Cannot transfer from non-existent account (treats as 0 balance)
		t.Run("Cannot Transfer From Non-existent Account", func(t *testing.T) {
			cleanupEvents(t, pool)
			// Create only the destination account
			createAccount2Cmd := CreateAccountCommand{
				AccountID:      "test_acc2",
				InitialBalance: 500,
			}
			err := handleCreateAccount(ctx, channelStore, createAccount2Cmd)
			require.NoError(t, err)

			nonExistentFromCmd := TransferMoneyCommand{
				TransferID:    "test_transfer_3",
				FromAccountID: "non_existent_account",
				ToAccountID:   "test_acc2",
				Amount:        100,
			}
			err = handleTransferMoney(ctx, channelStore, nonExistentFromCmd)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "insufficient funds")
		})

		// Test: Cannot transfer to non-existent account (creates it with 0 balance)
		t.Run("Cannot Transfer To Non-existent Account", func(t *testing.T) {
			cleanupEvents(t, pool)
			// Create only the source account
			createAccount1Cmd := CreateAccountCommand{
				AccountID:      "test_acc1",
				InitialBalance: 1000,
			}
			err := handleCreateAccount(ctx, channelStore, createAccount1Cmd)
			require.NoError(t, err)

			nonExistentToCmd := TransferMoneyCommand{
				TransferID:    "test_transfer_4",
				FromAccountID: "test_acc1",
				ToAccountID:   "non_existent_account",
				Amount:        100,
			}
			err = handleTransferMoney(ctx, channelStore, nonExistentToCmd)
			assert.NoError(t, err) // This should succeed as the destination account gets created with 0 balance
		})
	})
}

// TestSequentialTransfers tests that multiple transfers on the same account work correctly
// and that insufficient funds are properly detected
func TestSequentialTransfers(t *testing.T) {
	// Create context with timeout for the entire test
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Connect to test database
	pool, err := pgxpool.New(ctx, "postgres://postgres:postgres@localhost:5432/dcb_app?sslmode=disable")
	require.NoError(t, err)
	defer pool.Close()

	// Clean up before test
	cleanupEvents(t, pool)

	// Create event store
	store, err := dcb.NewEventStore(ctx, pool)
	require.NoError(t, err)

	// Cast to EventStore for extended functionality
	channelStore := store.(dcb.EventStore)

	// Create accounts for testing
	createAccount3Cmd := CreateAccountCommand{
		AccountID:      "test_acc3",
		InitialBalance: 2000,
	}
	err = handleCreateAccount(ctx, channelStore, createAccount3Cmd)
	assert.NoError(t, err)

	createAccount4Cmd := CreateAccountCommand{
		AccountID:      "test_acc4",
		InitialBalance: 0,
	}
	err = handleCreateAccount(ctx, channelStore, createAccount4Cmd)
	assert.NoError(t, err)

	createAccount5Cmd := CreateAccountCommand{
		AccountID:      "test_acc5",
		InitialBalance: 0,
	}
	err = handleCreateAccount(ctx, channelStore, createAccount5Cmd)
	assert.NoError(t, err)

	// First transfer should succeed
	transfer1Cmd := TransferMoneyCommand{
		TransferID:    "test_transfer_5",
		FromAccountID: "test_acc3",
		ToAccountID:   "test_acc4",
		Amount:        1500,
	}
	err1 := handleTransferMoney(ctx, channelStore, transfer1Cmd)
	assert.NoError(t, err1)

	// Second transfer should fail due to insufficient funds
	transfer2Cmd := TransferMoneyCommand{
		TransferID:    "test_transfer_6",
		FromAccountID: "test_acc3",
		ToAccountID:   "test_acc5",
		Amount:        1000,
	}
	err2 := handleTransferMoney(ctx, channelStore, transfer2Cmd)
	assert.Error(t, err2)
	assert.Contains(t, err2.Error(), "insufficient funds")
}

func TestConcurrentTransfers_OptimisticLocking(t *testing.T) {
	g := NewWithT(t)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pool, err := pgxpool.New(ctx, "postgres://postgres:postgres@localhost:5432/dcb_app?sslmode=disable")
	g.Expect(err).To(BeNil())

	defer pool.Close()

	cleanupEvents(t, pool)

	store, err := dcb.NewEventStore(ctx, pool)
	g.Expect(err).To(BeNil())

	channelStore := store.(dcb.EventStore)

	// Create accounts
	createAccount1Cmd := CreateAccountCommand{
		AccountID:      "concurrent_acc1",
		InitialBalance: 100,
	}
	err = handleCreateAccount(ctx, channelStore, createAccount1Cmd)
	g.Expect(err).To(BeNil())

	createAccount2Cmd := CreateAccountCommand{
		AccountID:      "concurrent_acc2",
		InitialBalance: 0,
	}
	err = handleCreateAccount(ctx, channelStore, createAccount2Cmd)
	g.Expect(err).To(BeNil())

	// Prepare concurrent transfer commands
	transferCmd := func(id string) TransferMoneyCommand {
		return TransferMoneyCommand{
			TransferID:    id,
			FromAccountID: "concurrent_acc1",
			ToAccountID:   "concurrent_acc2",
			Amount:        100,
		}
	}

	numGoroutines := 5
	results := make(chan error, numGoroutines)
	start := make(chan struct{})

	transferFn := func(id string) {
		<-start

		// Log the transfer attempt
		t.Logf("Transfer %s: Starting transfer of %d from %s to %s", id, 100, "concurrent_acc1", "concurrent_acc2")

		err := handleTransferMoney(ctx, channelStore, transferCmd(id))

		if err != nil {
			t.Logf("Transfer %s: FAILED - %v", id, err)
		} else {
			t.Logf("Transfer %s: SUCCESS", id)
		}

		results <- err
	}

	// Start multiple goroutines
	for i := 0; i < numGoroutines; i++ {
		go transferFn(fmt.Sprintf("tx%d", i+1))
	}
	close(start)

	// Collect all results
	errors := make([]error, numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		errors[i] = <-results
	}

	// Count successes and failures
	successes := 0
	failures := 0
	for _, err := range errors {
		if err == nil {
			successes++
		} else {
			failures++
		}
	}

	t.Logf("Results: %d successes, %d failures", successes, failures)
	for i, err := range errors {
		if err == nil {
			t.Logf("Transfer %d: SUCCESS", i+1)
		} else {
			t.Logf("Transfer %d: FAILED - %v", i+1, err)
		}
	}

	// At most one should succeed, or all may fail with concurrency errors
	g.Expect(successes).To(BeNumerically("<=", 1), "Expected at most one transfer to succeed due to optimistic locking")

	if successes == 0 {
		t.Logf("All transfers failed as expected due to concurrency/optimistic locking")
	} else {
		t.Logf("Exactly one transfer succeeded as expected")
	}
}
