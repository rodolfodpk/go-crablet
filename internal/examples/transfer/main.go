// This example is standalone. Run with: go run examples/transfer/main.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
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
	// Create context with timeout for the entire application
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, "postgres://postgres:postgres@localhost:5432/dcb_app?sslmode=disable")
	if err != nil {
		log.Fatalf("failed to connect to db: %v", err)
	}
	// Truncate events table before running the example
	_, err = pool.Exec(ctx, "TRUNCATE TABLE events RESTART IDENTITY CASCADE")
	if err != nil {
		log.Fatalf("failed to truncate events table: %v", err)
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
		fmt.Printf("Transfer failed: %v\n", err)
		fmt.Println("\n=== Events in Database (after transfer failure) ===")
		utils.DumpEvents(ctx, pool)
	} else {
		fmt.Printf("Transfer successful! Transfer ID: %s\n", transferCmd.TransferID)
	}

	// Second transfer (should fail due to optimistic locking or insufficient funds)
	fmt.Println("\n--- Attempting second transfer (should fail if locking works) ---")
	secondTransferCmd := TransferMoneyCommand{
		TransferID:    fmt.Sprintf("tx-%d", time.Now().UnixNano()),
		FromAccountID: createAccount1Cmd.AccountID,
		ToAccountID:   createAccount2Cmd.AccountID,
		Amount:        300, // Try to transfer again
	}
	err = handleTransferMoney(ctx, store, secondTransferCmd)
	if err != nil {
		fmt.Printf("Second transfer failed (expected): %v\n", err)
		fmt.Println("\n=== Events in Database (after second transfer failure) ===")
		utils.DumpEvents(ctx, pool)
	} else {
		fmt.Printf("Second transfer succeeded (unexpected)! Transfer ID: %s\n", secondTransferCmd.TransferID)
	}

	// Third transfer (should succeed, balance will be 100)
	fmt.Println("\n--- Attempting third transfer (should succeed, balance will be 100) ---")
	thirdTransferCmd := TransferMoneyCommand{
		TransferID:    fmt.Sprintf("tx-%d", time.Now().UnixNano()),
		FromAccountID: createAccount1Cmd.AccountID,
		ToAccountID:   createAccount2Cmd.AccountID,
		Amount:        300, // Try to transfer again
	}
	err = handleTransferMoney(ctx, store, thirdTransferCmd)
	if err != nil {
		fmt.Printf("Third transfer failed (unexpected): %v\n", err)
		fmt.Println("\n=== Events in Database (after third transfer failure) ===")
		utils.DumpEvents(ctx, pool)
	} else {
		fmt.Printf("Third transfer succeeded (expected)! Transfer ID: %s\n", thirdTransferCmd.TransferID)
	}

	// Fourth transfer (should fail due to insufficient funds)
	fmt.Println("\n--- Attempting fourth transfer (should fail due to insufficient funds) ---")
	fourthTransferCmd := TransferMoneyCommand{
		TransferID:    fmt.Sprintf("tx-%d", time.Now().UnixNano()),
		FromAccountID: createAccount1Cmd.AccountID,
		ToAccountID:   createAccount2Cmd.AccountID,
		Amount:        300, // Try to transfer again
	}
	err = handleTransferMoney(ctx, store, fourthTransferCmd)
	if err != nil {
		fmt.Printf("Fourth transfer failed (expected): %v\n", err)
		fmt.Println("\n=== Events in Database (after fourth transfer failure) ===")
		utils.DumpEvents(ctx, pool)
	} else {
		fmt.Printf("Fourth transfer succeeded (unexpected)! Transfer ID: %s\n", fourthTransferCmd.TransferID)
	}

	// Dump all events to show what was created
	fmt.Println("\n=== Events in Database ===")
	utils.DumpEvents(ctx, pool)

	// TODO: This example must fail with a concurrency exception (optimistic locking) when working correctly.
	// Simulate concurrent/conflicting transfers (should trigger optimistic locking)
	fmt.Println("\n--- Simulating concurrent/conflicting transfers (should trigger optimistic locking) ---")

	// First, let's set up a scenario where account has exactly 100 balance
	// so concurrent transfers of 100 should cause conflicts
	fmt.Println("\n--- Setting up concurrent transfer scenario ---")
	setupTransferCmd := TransferMoneyCommand{
		TransferID:    fmt.Sprintf("tx-setup-%d", time.Now().UnixNano()),
		FromAccountID: createAccount1Cmd.AccountID,
		ToAccountID:   createAccount2Cmd.AccountID,
		Amount:        900, // This will leave exactly 100 balance
	}
	err = handleTransferMoney(ctx, store, setupTransferCmd)
	if err != nil {
		fmt.Printf("Setup transfer failed: %v\n", err)
	} else {
		fmt.Printf("Setup transfer successful! Balance should now be exactly 100\n")
	}

	simulateConcurrentTransfers(ctx, store, createAccount1Cmd.AccountID, createAccount2Cmd.AccountID, pool)
}

// Command handlers with their own business rules

func handleCreateAccount(ctx context.Context, store dcb.EventStore, cmd CreateAccountCommand) error {
	// Command-specific projectors
	projectors := []dcb.StateProjector{
		{
			ID: "accountExists",
			Query: dcb.NewQuery(
				dcb.NewTags("account_id", cmd.AccountID),
				"AccountOpened",
			),
			InitialState: false,
			TransitionFn: func(state any, event dcb.Event) any {
				return true // If we see an AccountOpened event, account exists
			},
		},
	}

	states, appendCondition, err := store.Project(ctx, projectors)
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
	err = store.AppendIf(ctx, events, appendCondition)
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

	// Project state and get append condition
	// Project only the 'from' account for the append condition
	states, appendCondition, err := store.Project(ctx, []dcb.StateProjector{
		{
			ID:           "from",
			Query:        fromProjector.Query,
			InitialState: fromProjector.InitialState,
			TransitionFn: fromProjector.TransitionFn,
		},
	})
	if err != nil {
		return fmt.Errorf("projection failed: %w", err)
	}
	from := states["from"].(*AccountState)

	// Project the 'to' account separately for business logic
	statesTo, _, err := store.Project(ctx, []dcb.StateProjector{
		{
			ID:           "to",
			Query:        toProjector.Query,
			InitialState: toProjector.InitialState,
			TransitionFn: toProjector.TransitionFn,
		},
	})
	if err != nil {
		return fmt.Errorf("projection failed for to account: %w", err)
	}
	to := statesTo["to"].(*AccountState)

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

	// Use the original append condition which has the correct AfterCursor
	// This ensures optimistic locking by checking for new events on the account after the cursor

	// Debug: print the exact JSON being sent to SQL function
	conditionJSON, _ := json.Marshal(appendCondition)
	fmt.Printf("[DEBUG Transfer %s] Condition JSON: %s\n", cmd.TransferID, string(conditionJSON))

	err = store.AppendIf(ctx, events, appendCondition)
	if err != nil {
		return fmt.Errorf("append failed: %w", err)
	}

	// Display the transfer results using the calculated new balances
	fmt.Printf("Account %s: %d -> %d\n", cmd.FromAccountID, from.Balance, newFromBalance)
	fmt.Printf("Account %s: %d -> %d\n", cmd.ToAccountID, to.Balance, newToBalance)

	return nil
}

// Debug: print append condition details before transfer append
func debugAppendCondition(cond any) {
	ac, ok := cond.(*struct {
		FailIfEventsMatch any
		AfterCursor       any
	})
	if !ok {
		fmt.Printf("[DEBUG] Could not type assert to appendCondition struct\n")
		return
	}
	fmt.Printf("[DEBUG] appendCondition: %+v\n", ac)
	if ac.FailIfEventsMatch != nil {
		fmt.Printf("[DEBUG] FailIfEventsMatch: %+v\n", ac.FailIfEventsMatch)
		if q, ok := ac.FailIfEventsMatch.(*struct{ Items []any }); ok {
			for i, item := range q.Items {
				fmt.Printf("[DEBUG] QueryItem %d: %+v\n", i, item)
			}
		}
	}
	if ac.AfterCursor != nil {
		fmt.Printf("[DEBUG] AfterCursor: %+v\n", ac.AfterCursor)
	}
}

func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

func simulateConcurrentTransfers(ctx context.Context, store dcb.EventStore, fromID, toID string, pool *pgxpool.Pool) {
	fmt.Println("\n--- Simulating concurrent/conflicting transfers (should trigger optimistic locking) ---")

	// First, get the current balance to transfer exactly that amount
	projectors := []dcb.StateProjector{
		{
			ID: "from",
			Query: dcb.NewQuery(
				dcb.NewTags("account_id", fromID),
				"AccountOpened", "MoneyTransferred",
			),
			InitialState: &AccountState{AccountID: fromID},
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
						if data.FromAccountID == fromID {
							acc.Balance = data.FromBalance
							acc.UpdatedAt = data.TransferredAt
						} else if data.ToAccountID == fromID {
							acc.Balance = data.ToBalance
							acc.UpdatedAt = data.TransferredAt
						}
					}
				}
				return acc
			},
		},
	}
	states, _, err := store.Project(ctx, projectors)
	if err != nil {
		fmt.Printf("Failed to get current balance: %v\n", err)
		return
	}
	from := states["from"].(*AccountState)
	exactAmount := from.Balance
	fmt.Printf("Current balance: %d, will attempt concurrent transfers of exactly %d\n", exactAmount, exactAmount)

	if exactAmount <= 0 {
		fmt.Println("No balance to transfer")
		return
	}

	var wg sync.WaitGroup
	start := make(chan struct{})
	results := make(chan string, 5) // Increased buffer for 5 goroutines

	transferFn := func(name string) {
		defer wg.Done()
		// Project state and get append condition
		projectors := []dcb.StateProjector{
			{
				ID: "from",
				Query: dcb.NewQuery(
					dcb.NewTags("account_id", fromID),
					"AccountOpened", "MoneyTransferred", // Include both for state projection
				),
				InitialState: &AccountState{AccountID: fromID},
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
							if data.FromAccountID == fromID {
								acc.Balance = data.FromBalance
								acc.UpdatedAt = data.TransferredAt
							} else if data.ToAccountID == fromID {
								acc.Balance = data.ToBalance
								acc.UpdatedAt = data.TransferredAt
							}
						}
					}
					return acc
				},
			},
		}
		states, appendCondition, err := store.Project(ctx, projectors)
		if err != nil {
			results <- fmt.Sprintf("%s: projection failed: %v", name, err)
			return
		}
		from := states["from"].(*AccountState)
		if from.Balance < exactAmount {
			results <- fmt.Sprintf("%s: insufficient funds: %d", name, from.Balance)
			return
		}

		// Debug: print the exact JSON being sent to SQL function
		conditionJSON, _ := json.Marshal(appendCondition)
		fmt.Printf("[DEBUG %s] Condition JSON: %s\n", name, string(conditionJSON))

		// Wait for all goroutines to be ready - BETTER SYNCHRONIZATION
		<-start

		// Attempt the transfer
		transferID := fmt.Sprintf("concurrent-tx-%d", time.Now().UnixNano())
		events := []dcb.InputEvent{
			dcb.NewInputEvent(
				"MoneyTransferred",
				dcb.NewTags(
					"transfer_id", transferID,
					"from_account_id", fromID,
					"to_account_id", toID,
					"account_id", fromID,
				),
				mustJSON(MoneyTransferred{
					TransferID:    transferID,
					FromAccountID: fromID,
					ToAccountID:   toID,
					Amount:        exactAmount,
					FromBalance:   from.Balance - exactAmount,
					ToBalance:     0, // Not used here
					TransferredAt: time.Now(),
				}),
			),
		}

		// Use the original append condition which has the correct AfterCursor
		// This ensures optimistic locking by checking for new events on the account after the cursor
		err = store.AppendIf(ctx, events, appendCondition)
		if err != nil {
			results <- fmt.Sprintf("%s: transfer failed (expected optimistic locking): %v", name, err)
		} else {
			results <- fmt.Sprintf("%s: transfer succeeded", name)
		}
	}

	wg.Add(5) // Increased to 5 goroutines
	go transferFn("Goroutine 1")
	go transferFn("Goroutine 2")
	go transferFn("Goroutine 3")
	go transferFn("Goroutine 4")
	go transferFn("Goroutine 5")

	// BETTER SYNCHRONIZATION: Let all goroutines reach the barrier and be ready
	time.Sleep(100 * time.Millisecond) // Give goroutines time to reach the barrier
	close(start)                       // Start all goroutines simultaneously

	wg.Wait()
	close(results)
	for res := range results {
		fmt.Println(res)
	}
	fmt.Println("\n=== Events in Database (after concurrent transfers) ===")
	utils.DumpEvents(ctx, pool)
}
