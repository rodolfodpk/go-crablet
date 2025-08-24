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

#### 3. CommandExecutor (Optional High-Level API)
```go
type CommandExecutor interface {
    ExecuteCommand(ctx context.Context, command Command, handler CommandHandler, condition *AppendCondition) ([]InputEvent, error)
}

type Command interface {
    GetType() string
    GetData() []byte
    GetMetadata() map[string]interface{}
}

type CommandHandler interface {
    Handle(ctx context.Context, store EventStore, command Command) ([]InputEvent, error)
}
```

**Note**: CommandExecutor is an optional convenience layer. You can use the core EventStore API directly for full control, or use CommandExecutor for simplified command handling patterns.



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

**How It Works**: The condition checks if any `AccountCreated` events with `account_id: "123"` already exist. If they do, the append fails (preventing duplicates). If none exist, the append succeeds (first-time creation).

## Code Examples (Progressive Complexity)

### 1. Basic Event Storage
```go
// Create EventStore
store, err := dcb.NewEventStore(ctx, pool)
if err != nil {
    log.Fatal(err)
}

// Create and append a simple event
event := dcb.NewEvent("UserRegistered").
    WithTag("user_id", "123").
    WithData(map[string]any{
        "name": "John Doe",
        "email": "john@example.com",
    }).
    Build()

err = store.Append(ctx, []dcb.InputEvent{event})
```

### 2. Event Querying
```go
// Query events by tags
query := dcb.NewQueryBuilder().
    WithTag("user_id", "123").
    WithType("UserRegistered").
    Build()

events, err := store.Query(ctx, query, nil)
```

### 3. State Projection
```go
// Project user state from events
userProjector := dcb.ProjectState("user_state", "UserRegistered", "user_id", "123", 
    map[string]any{}, 
    func(state any, event dcb.Event) any {
        userState := state.(map[string]any)
        // Update state based on event
        return userState
    })

state, condition, err := store.Project(ctx, []dcb.StateProjector{userProjector}, nil)
```

### 4. DCB Concurrency Control
```go
// Prevent duplicate account creation
condition := dcb.NewAppendCondition(
    dcb.NewQueryBuilder().
        WithTag("account_id", "123").
        WithType("AccountCreated").
        Build(),
)

accountEvent := dcb.NewEvent("AccountCreated").
    WithTag("account_id", "123").
    WithData(map[string]any{
        "owner": "John Doe",
        "balance": 0,
    }).
    Build()

// Only succeeds if account doesn't exist
err := store.AppendIf(ctx, []dcb.InputEvent{accountEvent}, condition)
```

### 5. Command Pattern (Optional)
```go
// Define command handler
handler := dcb.CommandHandlerFunc(func(ctx context.Context, store dcb.EventStore, cmd dcb.Command) ([]dcb.InputEvent, error) {
    // Business logic to generate events
    return events, nil
})

// Execute command with concurrency control
events, err := commandExecutor.ExecuteCommand(ctx, command, handler, &condition)
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



## Use Cases

### Event Logging
- **Simple event storage** for audit trails and activity logs
- **Tag-based querying** for filtering events by context
- **Streaming operations** for real-time event processing

### Business Rule Enforcement
- **DCB conditions** prevent invalid state transitions
- **Race condition protection** without database locks
- **Domain constraint validation** through event queries

### State Reconstruction
- **Event-driven projections** rebuild current state
- **Aggregate state** from multiple event types
- **Historical state** at any point in time

## Database Schema

The library uses PostgreSQL with an optimized schema:

```sql
-- Events table (primary data)
CREATE TABLE events (
    type VARCHAR(64) NOT NULL,
    tags TEXT[] NOT NULL,
    data JSON NOT NULL,
    transaction_id xid8 NOT NULL,
    position BIGSERIAL NOT NULL PRIMARY KEY,
    occurred_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Commands table (metadata for CommandExecutor)
CREATE TABLE commands (
    transaction_id xid8 NOT NULL PRIMARY KEY,
    type VARCHAR(64) NOT NULL,
    data JSONB NOT NULL,
    metadata JSONB,
    occurred_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

**Schema Features:**
- **Optimized indexing** on tags and event types
- **JSON data storage** for flexible event payloads
- **Transaction tracking** for consistency and debugging
- **Position-based ordering** for reliable event sequencing

## Architecture

The library follows a layered architecture with clear separation of concerns:

### Core Layer (EventStore)
```go
// Direct database operations
store.Append(ctx, events)                    // Simple append
store.AppendIf(ctx, events, condition)       // Conditional append with DCB
store.Query(ctx, query, cursor)              // Event querying
store.Project(ctx, projectors, cursor)       // State reconstruction
```

### Optional Layer (CommandExecutor)
```go
// High-level command handling
commandExecutor.ExecuteCommand(ctx, command, handler, condition)
// Internally uses EventStore for all operations
```

### Data Flow
```go
// 1. Events flow directly to PostgreSQL
Client → EventStore → PostgreSQL (events table)

// 2. Commands flow through optional CommandExecutor
Client → CommandExecutor → CommandHandler → EventStore → PostgreSQL (commands + events tables)
```

**Key Design Principles:**
- **EventStore is the foundation** - always available and fully functional
- **CommandExecutor is optional** - convenience layer for command patterns
- **DCB at event level** - concurrency control through business rules
- **PostgreSQL as storage** - reliable, ACID-compliant event persistence

This library explores event sourcing concepts with DCB concurrency control. It's a learning project that experiments with DCB patterns using PostgreSQL, suitable for understanding event sourcing principles, testing DCB concepts, and benchmarking performance characteristics.
