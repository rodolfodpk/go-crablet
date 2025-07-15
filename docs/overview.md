# Overview: Dynamic Consistency Boundary (DCB) in go-crablet

go-crablet is a Go library for event sourcing, exploring concepts inspired by the [Dynamic Consistency Boundary (DCB)](https://dcb.events/) pattern.

## Key Concepts

- **Batch Projection**: Project multiple states in one database query
- **Combined Append Condition**: Use OR-combined queries for DCB concurrency control
- **Tag-based Queries**: Flexible, cross-entity queries using tags
- **Streaming**: Process events efficiently for large datasets
- **Transaction-based Ordering**: Uses PostgreSQL transaction IDs for true event ordering
- **Atomic Command Execution**: Execute commands with handler-based event generation

## Core Interfaces

### EventStore Interface

```go
type EventStore interface {
    Query(ctx context.Context, query Query, cursor *Cursor) ([]Event, error)
    QueryStream(ctx context.Context, query Query, cursor *Cursor) (<-chan Event, error)
    Append(ctx context.Context, events []InputEvent, condition *AppendCondition) error
    Project(ctx context.Context, projectors []StateProjector, cursor *Cursor) (map[string]any, AppendCondition, error)
    ProjectStream(ctx context.Context, projectors []StateProjector, cursor *Cursor) (<-chan map[string]any, <-chan AppendCondition, error)
    GetConfig() EventStoreConfig
    GetPool() *pgxpool.Pool
}
```

### CommandExecutor Interface (Optional)

```go
type CommandExecutor interface {
    ExecuteCommand(ctx context.Context, command Command, handler CommandHandler, condition *AppendCondition) ([]InputEvent, error)
}

type CommandHandler interface {
    Handle(ctx context.Context, store EventStore, command Command) ([]InputEvent, error)
}
```

### Usage Pattern

```go
// 1. Create EventStore (primary interface)
store, err := dcb.NewEventStore(ctx, pool)

// 2. Optionally create CommandExecutor (not required)
commandExecutor := dcb.NewCommandExecutor(store)

// 3. Use either interface as needed
err = store.Append(ctx, events, condition)  // Direct usage
err = commandExecutor.ExecuteCommand(ctx, command, handler, condition)  // Command-driven
```

### Supporting Types

```go
type Cursor struct {
    TransactionID uint64 `json:"transaction_id"`
    Position      int64  `json:"position"`
}

type Command interface {
    GetType() string
    GetData() []byte
    GetMetadata() map[string]interface{}
}
```

## Concurrency Control

### Primary: DCB Concurrency Control
- Uses `AppendCondition` to check for existing events before appending
- Conflict detection: Only one append succeeds when conditions match
- No blocking: Failed appends return immediately with `ConcurrencyError`
- Event ordering: Transaction IDs ensure correct, gapless ordering

### Optional: Advisory Locks
- Tag-based locking: Add tags with `lock:` prefix (e.g., "lock:user-123")
- Automatic acquisition: Database functions acquire locks on these keys
- Deadlock prevention: Locks sorted and acquired in consistent order
- Transaction-scoped: Automatically released on commit/rollback

## Configuration

```go
type EventStoreConfig struct {
    MaxBatchSize           int            `json:"max_batch_size"`           // Default: 1000
    LockTimeout            int            `json:"lock_timeout"`             // Default: 5000ms
    StreamBuffer           int            `json:"stream_buffer"`            // Default: 1000
    DefaultAppendIsolation IsolationLevel `json:"default_append_isolation"` // Default: Read Committed
    QueryTimeout           int            `json:"query_timeout"`            // Default: 15000ms
    AppendTimeout          int            `json:"append_timeout"`           // Default: 10000ms
}
```

## Example: Course Subscription

```go
projectors := []dcb.StateProjector{
    {
        ID: "courseExists",
        Query: dcb.NewQuery(dcb.NewTags("course_id", courseID), "CourseDefined"),
        InitialState: false,
        TransitionFn: func(state any, event dcb.Event) any { return true },
    },
    {
        ID: "numSubscriptions",
        Query: dcb.NewQuery(dcb.NewTags("course_id", courseID), "StudentEnrolled"),
        InitialState: 0,
        TransitionFn: func(state any, event dcb.Event) any { return state.(int) + 1 },
    },
}

states, appendCond, _ := store.Project(ctx, projectors, nil)
if !states["courseExists"].(bool) {
    store.Append(ctx, []dcb.InputEvent{...}, nil)
}
if states["numSubscriptions"].(int) < 2 {
    store.Append(ctx, []dcb.InputEvent{...}, &appendCond)
}
```

## Performance

See [benchmarks documentation](benchmarks.md) for detailed performance analysis.

## Table Validation

The library validates that the `events` table exists and has the correct structure:
- Required columns: `type`, `tags`, `data`, `transaction_id`, `position`, `occurred_at`
- Returns `TableStructureError` with detailed validation failure information
