# go-crablet Overview

go-crablet is a Go library **exploring** event sourcing concepts with Dynamic Consistency Boundary (DCB) patterns. This project focuses on learning and experimenting with DCB concepts rather than providing a production-ready solution.

**Note: This is an exploration project for learning and experimenting with DCB concepts, not a production-ready solution.**

## 🚀 Quick Start

### 1. Start Database
```bash
docker-compose up -d
docker-compose ps  # Wait for ready
```

### 2. Run Examples
```bash
# Try the transfer example
go run internal/examples/transfer/main.go

# Or use Makefile
make example-transfer
```

### 3. Cleanup
```bash
docker-compose down
```

## Core Concepts

### Event Sourcing
- **Events**: Immutable records of what happened
- **Event Store**: Append-only storage for events
- **Projections**: State reconstruction from events
- **DCB**: Dynamic Consistency Boundary for concurrency control

### Key Components

#### 1. EventStore (Core API)
```go
type EventStore interface {
    Append(ctx context.Context, events []InputEvent) error
    AppendIf(ctx context.Context, events []InputEvent, condition AppendCondition) error
    Query(ctx context.Context, query Query, after *Cursor) ([]Event, error)
    QueryStream(ctx context.Context, query Query, after *Cursor) (<-chan Event, error)
    Project(ctx context.Context, projectors []StateProjector, after *Cursor) (map[string]any, AppendCondition, error)
    ProjectStream(ctx context.Context, projectors []StateProjector, after *Cursor) (<-chan map[string]any, <-chan AppendCondition, error)
}
```

#### 2. StateProjector (State Reconstruction)
```go
type StateProjector struct {
    ID           string
    InitialState any
    EventTypes   []string
    Tags         []Tag
    Project      func(state any, event Event) any
}
```

#### 3. CommandExecutor (High-Level API)
```go
type CommandExecutor interface {
    ExecuteCommand(ctx context.Context, command Command, handler CommandHandler, condition *AppendCondition) ([]InputEvent, error)
}
```

## Architecture

### EventStore Flow
```
Client → EventStore → PostgreSQL
                ↓
            Events Table
            - type, tags, data
            - transaction_id, position
            - occurred_at
```

### State Projection Flow
```
Events → StateProjector → Aggregated State
   ↓
Project() function processes each event
   ↓
Returns final state + append condition
```

### CommandExecutor Flow
```
Client → CommandExecutor → CommandHandler → EventStore → PostgreSQL
                                    ↓
                                Events + Commands Tables
```

## DCB Concurrency Control

DCB (Dynamic Consistency Boundary) provides event-level concurrency control:

```go
// Define condition to prevent duplicate account creation
condition := dcb.NewAppendCondition(
    dcb.NewQueryBuilder().
        WithTag("account_id", "123").
        WithType("AccountCreated").
        Build(),
)

// Create the account creation event
accountEvent := dcb.NewEvent("AccountCreated").
    WithTag("account_id", "123").
    WithData(map[string]any{
        "owner": "John Doe",
        "balance": 0,
    }).
    Build()

// Append with condition - only succeeds if account doesn't exist
// This prevents duplicate account creation (race condition protection)
err := store.AppendIf(ctx, []dcb.InputEvent{accountEvent}, condition)
```

**What DCB Provides:**
- **Conflict Detection**: Identifies when business rules are violated during event appends
- **Domain Constraints**: Allows you to define conditions that must be met before events can be stored
- **Non-blocking**: Doesn't wait for locks or other resources
- **Multi-instance Support**: Can work across different application instances

**How It Works**: The condition checks if any `AccountCreated` events with `account_id: "123"` already exist. If they do, the append fails (preventing duplicates). If none exist, the append succeeds (first-time creation).

## Usage Examples

### Simple Event Store Usage

```go
// Create EventStore
store, err := dcb.NewEventStore(ctx, pool)
if err != nil {
    log.Fatal(err)
}

// Create events
events := []dcb.InputEvent{
    dcb.NewEvent("UserRegistered").
        WithTag("user_id", "123").
        WithData(map[string]any{
            "name": "John Doe",
            "email": "john@example.com",
        }).
        Build(),
}

// Append events
err = store.Append(ctx, events)
```

### State Projection

```go
// Define projector for user state
userProjector := dcb.ProjectState("user_state", "UserRegistered", "user_id", "123", 
    map[string]any{}, 
    func(state any, event dcb.Event) any {
        userState := state.(map[string]any)
        // Update state based on event
        return userState
    })

// Project state from events
state, condition, err := store.Project(ctx, []dcb.StateProjector{userProjector}, nil)
```

### Command Execution

```go
// Define command handler
handler := dcb.CommandHandlerFunc(func(ctx context.Context, store dcb.EventStore, cmd dcb.Command) ([]dcb.InputEvent, error) {
    // Business logic to generate events
    return events, nil
})

// Execute command
events, err := commandExecutor.ExecuteCommand(ctx, command, handler, nil)
```

## Configuration

### EventStore Configuration

The EventStore can be configured with logical grouping for append and query operations:

```go
config := dcb.EventStoreConfig{
    // =============================================================================
    // APPEND OPERATIONS CONFIGURATION
    // =============================================================================
    
    // MaxBatchSize controls the maximum number of events per batch
    MaxBatchSize: 1000,
    
    // DefaultAppendIsolation sets PostgreSQL transaction isolation level
    DefaultAppendIsolation: dcb.IsolationLevelReadCommitted,
    
    // AppendTimeout sets maximum time for append operations (milliseconds)
    AppendTimeout: 10000, // 10 seconds
    
    // =============================================================================
    // QUERY OPERATIONS CONFIGURATION  
    // =============================================================================
    
    // QueryTimeout sets maximum time for query operations (milliseconds)
    QueryTimeout: 15000, // 15 seconds
    
    // StreamBuffer sets channel buffer size for streaming operations
    StreamBuffer: 1000,
}

store, err := dcb.NewEventStoreWithConfig(ctx, pool, config)
```

**Configuration Benefits:**
- **Logical Grouping**: Clear separation of append vs query settings
- **Performance Tuning**: Batch sizes, timeouts, and buffer settings
- **Database Control**: Transaction isolation levels and timeouts
- **Streaming Support**: Buffer configuration for high-throughput operations

## Performance Characteristics

- **Append**: ~1,000 ops/s (simple append)
- **AppendIf**: ~800 ops/s (with DCB conditions)
- **Query**: ~2,000 ops/s (tag-based filtering)
- **Project**: ~500 ops/s (state reconstruction)

## Best Practices

1. **Use descriptive event types** and relevant tags for querying
2. **Implement idempotent operations** in command handlers
3. **Use DCB conditions** for business rule enforcement
4. **Batch events** when possible for better performance
5. **Handle concurrency errors** gracefully with retry logic

This library explores event sourcing concepts with DCB concurrency control. It's a learning project that experiments with DCB patterns using PostgreSQL, suitable for understanding event sourcing principles, testing DCB concepts, and benchmarking performance characteristics.
