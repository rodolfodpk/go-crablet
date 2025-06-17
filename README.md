[![Go Report Card](https://goreportcard.com/badge/github.com/rodolfodpk/go-crablet)](https://goreportcard.com/report/github.com/rodolfodpk/go-crablet)
[![Code Coverage](https://img.shields.io/badge/code%20coverage-86.7%25-green?logo=go)](https://github.com/rodolfodpk/go-crablet/actions/workflows/coverage.yml)
[![GoDoc](https://godoc.org/github.com/rodolfodpk/go-crablet?status.svg)](https://godoc.org/github.com/rodolfodpk/go-crablet)
[![License](https://img.shields.io/github/license/rodolfodpk/go-crablet)](https://github.com/rodolfodpk/go-crablet/blob/main/LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/rodolfodpk/go-crablet)](https://github.com/rodolfodpk/go-crablet/blob/main/go.mod)

# Go-Crablet: Event Sourcing with Decision Models

Go-Crablet is a Go library for event sourcing that implements the **Decision Model** pattern. It provides a clean, type-safe way to build event-sourced applications with proper command handling, business rule validation, and optimistic locking.

## Key Features

- **Command Pattern**: Each command has its own business rules and invariants
- **Decision Models**: Project current state to make decisions before appending events
- **Optimistic Locking**: Built-in concurrency control with append conditions
- **Batch Operations**: Efficient handling of multiple commands atomically
- **Type Safety**: Strongly typed events and commands
- **PostgreSQL Backend**: Robust, production-ready storage

## Quick Start

### Installation

```bash
go get github.com/your-username/go-crablet
```

### Basic Usage

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "time"

    "go-crablet/pkg/dcb"
    "github.com/jackc/pgx/v5/pgxpool"
)

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
    ctx := context.Background()
    
    // Connect to PostgreSQL
    pool, err := pgxpool.New(ctx, "postgres://user:pass@localhost:5432/db")
    if err != nil {
        log.Fatal(err)
    }
    
    // Create event store
    store, err := dcb.NewEventStore(ctx, pool)
    if err != nil {
        log.Fatal(err)
    }

    // Command 1: Create Account
    createCmd := CreateAccountCommand{
        AccountID:      "acc1",
        Owner:          "Alice",
        InitialBalance: 1000,
    }
    err = handleCreateAccount(ctx, store, createCmd)
    if err != nil {
        log.Fatal(err)
    }

    // Command 2: Transfer Money
    transferCmd := TransferMoneyCommand{
        TransferID:    "transfer-123",
        FromAccountID: "acc1",
        ToAccountID:   "acc2",
        Amount:        150,
        Description:   "Payment for services",
    }
    err = handleTransferMoney(ctx, store, transferCmd)
    if err != nil {
        log.Fatal(err)
    }
}

// Command handlers with business rules
func handleCreateAccount(ctx context.Context, store dcb.EventStore, cmd CreateAccountCommand) error {
    // Command-specific projectors
    projectors := []dcb.BatchProjector{
        {ID: "accountExists", StateProjector: dcb.StateProjector{
            Query: dcb.NewQuery(
                dcb.NewTags("account_id", cmd.AccountID),
                "AccountOpened",
            ),
            InitialState: false,
            TransitionFn: func(state any, event dcb.Event) any {
                return true // If we see an AccountOpened event, account exists
            },
        }},
    }

    states, appendCondition, err := store.ProjectDecisionModel(ctx, projectors, nil)
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
    _, err = store.Append(ctx, events, &appendCondition)
    if err != nil {
        return fmt.Errorf("failed to create account: %w", err)
    }

    return nil

}

func handleTransferMoney(ctx context.Context, store dcb.EventStore, cmd TransferMoneyCommand) error {
    // Command-specific projectors for both accounts
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
                }
            case "MoneyTransferred":
                var data MoneyTransferred
                if err := json.Unmarshal(event.Data, &data); err == nil {
                    if data.FromAccountID == cmd.FromAccountID {
                        acc.Balance = data.FromBalance
                    } else if data.ToAccountID == cmd.FromAccountID {
                        acc.Balance = data.ToBalance
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
                }
            case "MoneyTransferred":
                var data MoneyTransferred
                if err := json.Unmarshal(event.Data, &data); err == nil {
                    if data.FromAccountID == cmd.ToAccountID {
                        acc.Balance = data.FromBalance
                    } else if data.ToAccountID == cmd.ToAccountID {
                        acc.Balance = data.ToBalance
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

    // Use the append condition from the decision model for optimistic locking
    // All events are appended atomically for this command
    _, err = store.Append(ctx, events, &appendCondition)
    if err != nil {
        return fmt.Errorf("append failed: %w", err)
    }

    return nil

```
}

## Examples

The library includes comprehensive examples demonstrating different patterns:

### 1. Transfer Example (`internal/examples/transfer/`)

Shows how to implement money transfers between accounts with proper business rules:

- Account creation with duplicate prevention
- Money transfers with balance validation
- Optimistic locking for concurrent access

### 2. Decision Model Example (`internal/examples/decision_model/`)

Demonstrates the core decision model pattern:

- State projection for decision making
- Optimistic locking with append conditions
- Efficient batch processing

### 3. Batch Example (`internal/examples/batch/`)

Shows how to handle multiple commands atomically:

- Batch user creation with validation
- Batch order processing
- Cross-command business rules

## REST API Implementation

A complete REST API implementation is available in `internal/web-app/` that provides HTTP endpoints for the DCB Bench specification:

### Features
- **OpenAPI 3.0.3 Compliance**: Implements the [DCB Bench specification](https://app.swaggerhub.com/apis/wwwision/dcb-bench/1.0.0)
- **HTTP Endpoints**: `/read` and `/append` endpoints with full feature support
- **Performance Testing**: Comprehensive k6 load testing with benchmarks
- **Docker Support**: Containerized deployment with PostgreSQL
- **Production Ready**: Includes health checks, monitoring, and error handling

### Quick Start
```bash
# Start the complete stack
cd internal/web-app
make setup-and-run

# Run performance tests
make test

# View API documentation
open http://localhost:8080
```

See [`internal/web-app/README.md`](internal/web-app/README.md) for complete documentation.

## Command Pattern

Each command in Go-Crablet follows a consistent pattern:

1. **Command Definition**: Define the command structure with all necessary data
2. **State Projection**: Use projectors to build the current state needed for decisions
3. **Business Rule Validation**: Apply command-specific business rules and invariants
4. **Event Creation**: Generate events that represent the command's effects
5. **Atomic Append**: Append all events atomically with optimistic locking

This pattern ensures:
- **Consistency**: Each command enforces its own business rules
- **Isolation**: Commands don't interfere with each other
- **Concurrency**: Optimistic locking prevents race conditions
- **Auditability**: All decisions are recorded as events

## Testing

All examples include comprehensive tests that demonstrate:
- Command success scenarios
- Business rule validation
- Error handling
- Optimistic locking behavior

Run the tests:

```bash
# Run all tests
go test ./...

# Run specific example tests
go test ./internal/examples/transfer/
go test ./internal/examples/decision_model/
go test ./internal/examples/batch/
```

## Database Setup

Go-Crablet requires PostgreSQL. Set up the database:

```sql
CREATE DATABASE dcb_app;
```

The library will automatically create the necessary tables on first use.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Ensure all tests pass
6. Submit a pull request

## License

MIT License - see LICENSE file for details.