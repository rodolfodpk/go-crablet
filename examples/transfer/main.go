// This example is standalone. Run with: go run examples/account_transfer_example.go
package transfer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"go-crablet/pkg/dcb"

	"github.com/jackc/pgx/v5/pgxpool"
)

// AccountState holds the state for an account
type AccountState struct {
	Balance int
	Owner   string
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

	cmd := struct {
		FromAccount string
		ToAccount   string
		Amount      int
	}{FromAccount: "acc1", ToAccount: "acc2", Amount: 50}

	// Define projectors
	fromProjector := dcb.StateProjector{
		Query: dcb.NewQuery(
			dcb.NewTags("account_id", cmd.FromAccount),
			"AccountCreated", "MoneyDeposited", "MoneyWithdrawn",
		),
		InitialState: &AccountState{},
		TransitionFn: func(state any, e dcb.Event) any {
			acc := state.(*AccountState)
			switch e.Type {
			case "AccountCreated":
				var data struct{ Owner string }
				if err := json.Unmarshal(e.Data, &data); err == nil {
					acc.Owner = data.Owner
				}
			case "MoneyDeposited":
				var data struct{ Amount int }
				if err := json.Unmarshal(e.Data, &data); err == nil {
					acc.Balance += data.Amount
				}
			case "MoneyWithdrawn":
				var data struct{ Amount int }
				if err := json.Unmarshal(e.Data, &data); err == nil {
					acc.Balance -= data.Amount
				}
			}
			return acc
		},
	}
	toProjector := dcb.StateProjector{
		Query: dcb.NewQuery(
			dcb.NewTags("account_id", cmd.ToAccount),
			"AccountCreated", "MoneyDeposited", "MoneyWithdrawn",
		),
		InitialState: &AccountState{},
		TransitionFn: func(state any, e dcb.Event) any {
			acc := state.(*AccountState)
			switch e.Type {
			case "AccountCreated":
				var data struct{ Owner string }
				if err := json.Unmarshal(e.Data, &data); err == nil {
					acc.Owner = data.Owner
				}
			case "MoneyDeposited":
				var data struct{ Amount int }
				if err := json.Unmarshal(e.Data, &data); err == nil {
					acc.Balance += data.Amount
				}
			case "MoneyWithdrawn":
				var data struct{ Amount int }
				if err := json.Unmarshal(e.Data, &data); err == nil {
					acc.Balance -= data.Amount
				}
			}
			return acc
		},
	}

	// Project both accounts using the DCB decision model pattern
	query := dcb.NewQuery(dcb.NewTags("account_id", cmd.FromAccount, "account_id", cmd.ToAccount))
	states, appendCondition, err := store.ProjectDecisionModel(ctx, query, nil, []dcb.BatchProjector{
		{ID: "from", StateProjector: fromProjector},
		{ID: "to", StateProjector: toProjector},
	})
	if err != nil {
		log.Fatalf("projection failed: %v", err)
	}
	from := states["from"].(*AccountState)

	// Business rules
	if from.Balance < cmd.Amount {
		log.Fatalf("insufficient funds")
	}
	if cmd.Amount <= 0 {
		log.Fatalf("invalid transfer amount")
	}

	// Prepare events
	withdrawEvent := dcb.InputEvent{
		Type: "MoneyWithdrawn",
		Tags: dcb.NewTags("account_id", cmd.FromAccount),
		Data: mustJSON(map[string]any{"Amount": cmd.Amount}),
	}
	depositEvent := dcb.InputEvent{
		Type: "MoneyDeposited",
		Tags: dcb.NewTags("account_id", cmd.ToAccount),
		Data: mustJSON(map[string]any{"Amount": cmd.Amount}),
	}

	// Use the append condition from the decision model for optimistic locking
	_, err = store.Append(ctx, []dcb.InputEvent{withdrawEvent, depositEvent}, &appendCondition)
	if err != nil {
		log.Fatalf("append failed: %v", err)
	}

	fmt.Println("Transfer successful!")
}

func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}
