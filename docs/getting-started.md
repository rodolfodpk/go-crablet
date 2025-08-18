# Getting Started with go-crablet

This guide will help you get started with go-crablet, a Go library for event sourcing with Dynamic Consistency Boundary (DCB) concurrency control.

## Quick Start

### 1. Installation

```bash
go get github.com/rodolfodpk/go-crablet/pkg/dcb
```

### 2. Basic Usage

```go
package main

import (
    "context"
    "log"
    "time"
    
    "github.com/rodolfodpk/go-crablet/pkg/dcb"
)

// BEST PRACTICE: Define event data as structs for type safety and performance
type UserRegisteredData struct {
    Name         string    `json:"name"`
    Email        string    `json:"email"`
    RegisteredAt time.Time `json:"registered_at"`
}

func main() {
    ctx := context.Background()
    
    // Create EventStore
    store, err := dcb.NewEventStore(ctx, "postgres://user:pass@localhost:5432/db")
    if err != nil {
        log.Fatal(err)
    }
    
    // Create events with struct-based data (RECOMMENDED)
    events := []dcb.InputEvent{
        dcb.NewEvent("UserRegistered").
            WithTag("user_id", "123").
            WithData(UserRegisteredData{
                Name:         "John Doe",
                Email:        "john@example.com",
                RegisteredAt: time.Now(),
            }).
            Build(),
    }
    
    // Append events
    err = store.Append(ctx, events)
    if err != nil {
        log.Fatal(err)
    }
    
    log.Println("User registered successfully")
}
```

### 3. DCB Concurrency Control

```go
// Create condition to prevent conflicts using QueryBuilder
condition := dcb.NewAppendCondition(
    dcb.NewQueryBuilder().
        WithTag("user_id", "123").
        WithType("UserRegistered").
        Build(),
)

// Append with condition (fails if user already exists)
err = store.AppendIf(ctx, events, condition)
if err != nil {
    if dcb.IsConcurrencyError(err) {
        log.Println("User already exists")
    } else {
        log.Fatal(err)
    }
}
```

### 4. Query Events

```go
// Query events by tags using QueryBuilder
query := dcb.NewQueryBuilder().
    WithTag("user_id", "123").
    Build()

events, err := store.Query(ctx, query, nil)
if err != nil {
    log.Fatal(err)
}

log.Printf("Found %d events for user 123", len(events))
```

### 5. Project State

```go
// Define state projector using QueryBuilder
projector := dcb.StateProjector{
    ID: "UserState",
    Query: dcb.NewQueryBuilder().
        WithTag("user_id", "123").
        Build(),
    InitialState: map[string]any{
        "name": "",
        "email": "",
        "registered": false,
    },
    TransitionFn: func(state any, event dcb.Event) any {
        currentState := state.(map[string]any)
        switch event.GetType() {
        case "UserRegistered":
            var data map[string]any
            json.Unmarshal(event.GetData(), &data)
            currentState["name"] = data["name"]
            currentState["email"] = data["email"]
            currentState["registered"] = true
        }
        return currentState
    },
}

// Execute projection
finalState, cursor, err := store.Project(ctx, []dcb.StateProjector{projector}, nil)
if err != nil {
    log.Fatal(err)
}

userState := finalState["UserState"].(map[string]any)
log.Printf("User: %s (%s)", userState["name"], userState["email"])
```

## Command Execution

### 1. Create CommandExecutor

```go
commandExecutor := dcb.NewCommandExecutor(store)
```

### 2. Define Command Handler

```go
func handleRegisterUser(ctx context.Context, store dcb.EventStore, cmd dcb.Command) ([]dcb.InputEvent, error) {
    var data map[string]any
    json.Unmarshal(cmd.GetData(), &data)
    
    // Business logic validation
    if data["email"] == "" {
        return nil, errors.New("email required")
    }
    
    // Create event
    event := dcb.NewEvent("UserRegistered").
        WithTag("user_id", data["user_id"].(string)).
        WithData(data).
        Build()
    
    return []dcb.InputEvent{event}, nil
}
```

### 3. Execute Command

```go
// Create command
command := dcb.NewCommand("RegisterUser", dcb.ToJSON(map[string]any{
    "user_id": "123",
    "name": "John Doe",
    "email": "john@example.com",
}), nil)

// Execute command
events, err := commandExecutor.ExecuteCommand(ctx, command, handleRegisterUser, nil)
if err != nil {
    log.Fatal(err)
}
```

## Configuration

### EventStore Configuration

```go
config := dcb.EventStoreConfig{
    MaxBatchSize:           1000,
    StreamBuffer:           1000,
    DefaultAppendIsolation: dcb.IsolationLevelReadCommitted,
    QueryTimeout:           15000, // 15 seconds
    AppendTimeout:          10000, // 10 seconds
}

store, err := dcb.NewEventStoreWithConfig(ctx, pool, config)
```

### Connection Pool Configuration

```go
pool, err := pgxpool.New(ctx, "postgres://user:pass@localhost:5432/crablet")
if err != nil {
    log.Fatal(err)
}
defer pool.Close()

// Configure pool
pool.Config().MaxConns = 20
pool.Config().MinConns = 5
```

## Examples

The `internal/examples/` directory contains complete, runnable examples:

- **`internal/examples/transfer/`** - Money transfer system with DCB concurrency control
- **`internal/examples/concurrency_comparison/`** - Concert ticket booking comparing DCB concurrency control
- **`internal/examples/decision_model/`** - Complex decision model with multiple projectors
- **`internal/examples/batch/`** - Batch event processing examples

Run any example with: `go run internal/examples/[example-name]/main.go`

## Testing

### Run Tests

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run specific test package
go test -v ./pkg/dcb/tests/...
```

### Run Benchmarks

```bash
# Run Go library benchmarks
make benchmark-go

# Run web app benchmarks
make benchmark-web-app

# Run all benchmarks
make benchmark-all
```

## Next Steps

1. **Read the Documentation**:
   - [Overview](docs/overview.md): Core concepts and architecture
   - [EventStore Flow](docs/eventstore-flow.md): Direct event operations
   - [Command Execution Flow](docs/command-execution-flow.md): High-level command pattern
   - [Examples](docs/examples.md): Complete usage examples

2. **Explore Examples**:
   - Start with `internal/examples/transfer/` for basic usage
   - Try `internal/examples/concurrency_comparison/` for DCB concurrency control
   - Check `internal/examples/decision_model/` for complex scenarios

3. **Run Benchmarks**:
   - Use `make benchmark-go` to test performance
   - Check `docs/benchmarks.md` for detailed results

4. **Production Setup**:
   - Configure connection pooling
   - Set up monitoring and alerting
   - Implement proper error handling
   - Consider backup and recovery strategies

## Troubleshooting

### Common Issues

1. **Database Connection**:
   ```bash
   # Check if PostgreSQL is running
   docker-compose ps
   
   # Check connection
   psql -h localhost -p 5432 -U postgres -d crablet
   ```

2. **Schema Issues**:
   ```bash
   # Recreate database
   docker-compose down
   docker-compose up -d
   ```

3. **Test Failures**:
   ```bash
   # Clean and rebuild
   go clean -cache
   go test ./...
   ```

### Getting Help

- **Issues**: Create an issue on GitHub
- **Discussions**: Use GitHub Discussions
- **Documentation**: Check the docs/ directory

This getting started guide provides the foundation for using go-crablet. Explore the examples and documentation to learn more about advanced features and best practices.
