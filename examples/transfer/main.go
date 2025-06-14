// This example is standalone. Run with: go run examples/transfer/main.go
package transfer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

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

// TransferCommand represents a transfer command
type TransferCommand struct {
	TransferID    string
	FromAccountID string
	ToAccountID   string
	Amount        int
	Description   string
}

func main() {
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, "postgres://user:password@localhost:5432/db")
	if err != nil {
		log.Fatalf("failed to connect to db: %v", err)
	}
	store, err := dcb.NewEventStore(ctx, pool)
	if err != nil {
		log.Fatalf("failed to create event store: %v", err)
	}

	// Create some initial accounts if they don't exist
	err = createAccountIfNotExists(ctx, store, "acc1", "Alice", 1000)
	if err != nil {
		log.Fatalf("failed to create account acc1: %v", err)
	}

	err = createAccountIfNotExists(ctx, store, "acc2", "Bob", 500)
	if err != nil {
		log.Fatalf("failed to create account acc2: %v", err)
	}

	// Execute transfer command
	cmd := TransferCommand{
		TransferID:    "transfer-123",
		FromAccountID: "acc1",
		ToAccountID:   "acc2",
		Amount:        150,
		Description:   "Payment for services",
	}

	err = executeTransfer(ctx, store, cmd)
	if err != nil {
		log.Fatalf("transfer failed: %v", err)
	}

	fmt.Printf("Transfer successful! Transfer ID: %s\n", cmd.TransferID)
}

// createAccountIfNotExists creates an account if it doesn't already exist
func createAccountIfNotExists(ctx context.Context, store dcb.EventStore, accountID, owner string, initialBalance int) error {
	// Check if account exists
	accountExistsProjector := dcb.StateProjector{
		Query: dcb.NewQuery(
			dcb.NewTags("account_id", accountID),
			"AccountOpened",
		),
		InitialState: false,
		TransitionFn: func(state any, event dcb.Event) any {
			return true // If we see an AccountOpened event, account exists
		},
	}

	states, appendCondition, err := store.ProjectDecisionModel(ctx, []dcb.BatchProjector{
		{ID: "accountExists", StateProjector: accountExistsProjector},
	}, nil)
	if err != nil {
		return fmt.Errorf("failed to check account existence: %w", err)
	}

	if states["accountExists"].(bool) {
		fmt.Printf("Account %s already exists\n", accountID)
		return nil
	}

	// Create account
	accountOpenedEvent := dcb.NewInputEvent(
		"AccountOpened",
		dcb.NewTags("account_id", accountID),
		mustJSON(AccountOpened{
			AccountID:      accountID,
			Owner:          owner,
			InitialBalance: initialBalance,
			OpenedAt:       time.Now(),
		}),
	)

	_, err = store.Append(ctx, dcb.NewEventBatch(accountOpenedEvent), &appendCondition)
	if err != nil {
		return fmt.Errorf("failed to append account opened event: %w", err)
	}

	fmt.Printf("Created account %s for %s with balance %d\n", accountID, owner, initialBalance)
	return nil
}

// executeTransfer executes a money transfer between accounts
func executeTransfer(ctx context.Context, store dcb.EventStore, cmd TransferCommand) error {
	// Define projectors for both accounts
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

	// Project both accounts using the DCB decision model pattern
	states, appendCondition, err := store.ProjectDecisionModel(ctx, []dcb.BatchProjector{
		{ID: "from", StateProjector: fromProjector},
		{ID: "to", StateProjector: toProjector},
	}, nil)
	if err != nil {
		return fmt.Errorf("projection failed: %w", err)
	}

	from := states["from"].(*AccountState)
	to := states["to"].(*AccountState)

	// Business rule validations
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

	// Create the MoneyTransferred event with final balances
	transferEvent := dcb.NewInputEvent(
		"MoneyTransferred",
		dcb.NewTags(
			"transfer_id", cmd.TransferID,
			"from_account_id", cmd.FromAccountID,
			"to_account_id", cmd.ToAccountID,
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
	)

	// Use the append condition from the decision model for optimistic locking
	_, err = store.Append(ctx, dcb.NewEventBatch(transferEvent), &appendCondition)
	if err != nil {
		return fmt.Errorf("append failed: %w", err)
	}

	fmt.Printf("Account %s: %d -> %d\n", cmd.FromAccountID, from.Balance, newFromBalance)
	fmt.Printf("Account %s: %d -> %d\n", cmd.ToAccountID, to.Balance, newToBalance)

	return nil
}

func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}
