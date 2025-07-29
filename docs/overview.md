# go-crablet Overview

A Go library for event sourcing, exploring concepts inspired by Sara Pellegrini's Dynamic Consistency Boundary (DCB) pattern. This library provides both low-level event store operations and high-level command execution patterns.

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
    Project(ctx context.Context, projectors []StateProjector, after *Cursor) (map[string]any, *Cursor, error)
    ProjectStream(ctx context.Context, projectors []StateProjector, after *Cursor) (<-chan map[string]any, <-chan *Cursor, error)
}
```

#### 2. CommandExecutor (High-Level API)
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

### CommandExecutor Flow
```
Client → CommandExecutor → CommandHandler → EventStore → PostgreSQL
                                    ↓
                                Events + Commands Tables
```

## DCB Concurrency Control

DCB (Dynamic Consistency Boundary) provides event-level concurrency control:

```go
// Define condition to prevent conflicts using QueryBuilder
condition := dcb.NewAppendCondition(
    dcb.NewQueryBuilder().
        WithTag("account_id", "123").
        WithType("AccountCreated").
        Build(),
)

// Append with condition - fails if account doesn't exist
err := store.AppendIf(ctx, events, condition)
```

**What DCB Provides:**
- **Conflict Detection**: Identifies when business rules are violated during event appends
- **Domain Constraints**: Allows you to define conditions that must be met before events can be stored
- **Non-blocking**: Doesn't wait for locks or other resources
- **Multi-instance Support**: Can work across different application instances

## Usage Examples

### Simple Event Store Usage

```go
// Create EventStore
store, err := dcb.NewEventStore(ctx, "postgres://user:pass@localhost:5432/db")
if err != nil {
    log.Fatal(err)
}
defer store.Close()

// Create events
events := []dcb.InputEvent{
    dcb.NewEvent("UserRegistered").
        WithTag("user_id", "123").
        WithData(map[string]any{"name": "John Doe"}).
        Build(),
}

// Append events
err = store.Append(ctx, events)
```

### Command Execution

```go
// Create command
command := dcb.NewCommand("EnrollStudent", dcb.ToJSON(map[string]any{
    "student_id": "student123",
    "course_id": "CS101",
}), nil)

// Define command handler
handler := dcb.CommandHandlerFunc(func(ctx context.Context, store dcb.EventStore, cmd dcb.Command) ([]dcb.InputEvent, error) {
    var data map[string]any
    json.Unmarshal(cmd.GetData(), &data)
    
    // Business logic
    events := []dcb.InputEvent{
        dcb.NewEvent("StudentEnrolled").
            WithTag("student_id", data["student_id"].(string)).
            WithTag("course_id", data["course_id"].(string)).
            WithData(map[string]any{"enrolled_at": time.Now()}).
            Build(),
    }
    
    return events, nil
})

// Execute command
events, err := commandExecutor.ExecuteCommand(ctx, command, handler, nil)
```

### Concurrency Control

```go
// Create condition to prevent duplicate enrollment using QueryBuilder
enrollmentCondition := dcb.NewAppendCondition(
    dcb.NewQueryBuilder().
        WithTag("student_id", "student123").
        WithTag("course_id", "CS101").
        WithType("StudentEnrolled").
        Build(),
)

// Execute with condition
events, err := commandExecutor.ExecuteCommand(ctx, enrollmentCommand, handleEnrollStudent, &enrollmentCondition)
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

### Database Schema

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

## Performance Characteristics

### EventStore Performance
- **Append**: ~1,000 ops/s (simple append)
- **AppendIf**: ~800 ops/s (with DCB conditions)
- **Query**: ~2,000 ops/s (tag-based filtering)
- **Project**: ~500 ops/s (state reconstruction)

### Memory Usage
- **Low overhead**: ~6KB per operation
- **Efficient batching**: Up to 1,000 events per batch
- **Streaming support**: Memory-efficient for large datasets

## Best Practices

### 1. Event Design
- Use descriptive event types
- Include relevant tags for querying
- Keep data JSON-serializable
- Avoid large event payloads

### 2. Concurrency Control
- Use DCB conditions for business rules
- Design idempotent operations
- Handle concurrency errors gracefully

### 3. Performance
- Batch events when possible
- Use streaming for large datasets
- Optimize tag-based queries
- Monitor transaction sizes

### 4. Error Handling
- Check for concurrency errors
- Validate events before appending
- Handle database connection issues
- Implement retry logic for transient failures

## Getting Started

1. **Install**: `go get github.com/rodolfodpk/go-crablet`
2. **Setup Database**: Use provided Docker Compose or PostgreSQL
3. **Create EventStore**: Use `dcb.NewEventStore()` or `dcb.NewEventStoreWithConfig()`
4. **Start Appending**: Use `store.Append()` or `store.AppendIf()`

## Documentation

- [Quick Start](docs/quick-start.md): Get started in minutes
- [EventStore Flow](docs/eventstore-flow.md): Direct event operations
- [Command Execution Flow](docs/command-execution-flow.md): High-level command pattern
- [Low-Level Implementation](docs/low-level-implementation.md): Database schema and internals
- [Testing](docs/testing.md): Comprehensive testing guide
- [Benchmarks](docs/benchmarks.md): Performance analysis
- [Examples](docs/examples.md): Complete usage examples

This library provides a solid foundation for event sourcing with DCB concurrency control, suitable for both simple event logging and complex business applications.
