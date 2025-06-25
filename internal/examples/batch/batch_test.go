package main

import (
	"context"
	"testing"

	"go-crablet/pkg/dcb"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBatchExample(t *testing.T) {
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

	// Test Command 1: Create User
	t.Run("Create User", func(t *testing.T) {
		createUserCmd := CreateUserCommand{
			UserID:   "test_user123",
			Username: "john_doe",
			Email:    "john@example.com",
		}
		err := handleCreateUser(ctx, channelStore, createUserCmd)
		assert.NoError(t, err)
	})

	// Test Command 2: Create Order
	t.Run("Create Order", func(t *testing.T) {
		createOrderCmd := CreateOrderCommand{
			OrderID: "test_order456",
			UserID:  "test_user123",
			Items: []OrderItem{
				{ProductID: "prod1", Quantity: 2, Price: 29.99},
				{ProductID: "prod2", Quantity: 1, Price: 49.99},
			},
		}
		err := handleCreateOrder(ctx, channelStore, createOrderCmd)
		assert.NoError(t, err)
	})

	// Test business rules
	t.Run("Business Rules", func(t *testing.T) {
		// Test: Cannot create user with same ID
		t.Run("Cannot Create Duplicate User", func(t *testing.T) {
			duplicateCmd := CreateUserCommand{
				UserID:   "test_user123", // Same ID as existing user
				Username: "jane_doe",
				Email:    "jane@example.com",
			}
			err := handleCreateUser(ctx, channelStore, duplicateCmd)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "already exists")
		})

		// Test: Cannot create user with same email
		t.Run("Cannot Create User with Duplicate Email", func(t *testing.T) {
			duplicateEmailCmd := CreateUserCommand{
				UserID:   "test_user456",
				Username: "jane_doe",
				Email:    "john@example.com", // Same email as existing user
			}
			err := handleCreateUser(ctx, channelStore, duplicateEmailCmd)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "already exists")
		})

		// Test: Cannot create order with same ID
		t.Run("Cannot Create Duplicate Order", func(t *testing.T) {
			duplicateOrderCmd := CreateOrderCommand{
				OrderID: "test_order456", // Same ID as existing order
				UserID:  "test_user123",
				Items: []OrderItem{
					{ProductID: "prod3", Quantity: 1, Price: 19.99},
				},
			}
			err := handleCreateOrder(ctx, channelStore, duplicateOrderCmd)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "already exists")
		})

		// Test: Cannot create order for non-existent user
		t.Run("Cannot Create Order for Non-existent User", func(t *testing.T) {
			nonExistentUserCmd := CreateOrderCommand{
				OrderID: "test_order789",
				UserID:  "non_existent_user",
				Items: []OrderItem{
					{ProductID: "prod1", Quantity: 1, Price: 29.99},
				},
			}
			err := handleCreateOrder(ctx, channelStore, nonExistentUserCmd)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "does not exist")
		})
	})

	// Test batch operations
	t.Run("Batch Operations", func(t *testing.T) {
		// Test batch create users
		t.Run("Batch Create Users", func(t *testing.T) {
			users := []CreateUserCommand{
				{UserID: "batch_user1", Username: "batch_user1", Email: "batch1@example.com"},
				{UserID: "batch_user2", Username: "batch_user2", Email: "batch2@example.com"},
			}
			err := handleBatchCreateUsers(ctx, channelStore, users)
			assert.NoError(t, err)
		})

		// Test batch create orders
		t.Run("Batch Create Orders", func(t *testing.T) {
			orders := []CreateOrderCommand{
				{
					OrderID: "batch_order1",
					UserID:  "batch_user1",
					Items: []OrderItem{
						{ProductID: "prod1", Quantity: 1, Price: 29.99},
					},
				},
				{
					OrderID: "batch_order2",
					UserID:  "batch_user2",
					Items: []OrderItem{
						{ProductID: "prod2", Quantity: 2, Price: 49.99},
					},
				},
			}
			err := handleBatchCreateOrders(ctx, channelStore, orders)
			assert.NoError(t, err)
		})

		// Test batch validation - one user already exists
		t.Run("Batch Validation - Duplicate User", func(t *testing.T) {
			users := []CreateUserCommand{
				{UserID: "batch_user3", Username: "batch_user3", Email: "batch3@example.com"},
				{UserID: "batch_user1", Username: "batch_user1_duplicate", Email: "batch1_duplicate@example.com"}, // Already exists
			}
			err := handleBatchCreateUsers(ctx, channelStore, users)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "already exists")
		})

		// Test batch validation - one order already exists
		t.Run("Batch Validation - Duplicate Order", func(t *testing.T) {
			orders := []CreateOrderCommand{
				{
					OrderID: "batch_order3",
					UserID:  "batch_user3",
					Items: []OrderItem{
						{ProductID: "prod3", Quantity: 1, Price: 19.99},
					},
				},
				{
					OrderID: "batch_order1", // Already exists
					UserID:  "batch_user3",
					Items: []OrderItem{
						{ProductID: "prod4", Quantity: 1, Price: 39.99},
					},
				},
			}
			err := handleBatchCreateOrders(ctx, channelStore, orders)
			assert.NoError(t, err)
		})
	})
}
