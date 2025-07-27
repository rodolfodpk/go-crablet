# Overview: Dynamic Consistency Boundary (DCB) in go-crablet

go-crablet is a Go library for event sourcing, exploring concepts inspired by the [Dynamic Consistency Boundary (DCB)](https://dcb.events/) pattern.

## Key Concepts

- **Batch Projection**: Project multiple states in one database query
- **Combined Append Condition**: Use OR-combined queries for DCB concurrency control
- **Tag-based Queries**: Flexible, cross-entity queries using tags
- **Streaming**: Process events efficiently for large datasets
- **Transaction-based Ordering**: Uses PostgreSQL transaction IDs for true event ordering
- **Atomic Command Execution**: Execute commands with handler-based event generation
- **Fluent API**: Intuitive interfaces for events, queries, and projections with 50% less boilerplate

## Core Interfaces

### EventStore Interface

```go
type EventStore interface {
    Query(ctx context.Context, query Query, cursor *Cursor) ([]Event, error)
    QueryStream(ctx context.Context, query Query, cursor *Cursor) (<-chan Event, error)
    Append(ctx context.Context, events []InputEvent, condition *AppendCondition) error
    AppendIf(ctx context.Context, events []InputEvent, condition AppendCondition) error
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

// 3. Use fluent API for better developer experience
event := dcb.NewEvent("UserCreated").
    WithTag("user_id", "123").
    WithData(userData).
    Build()

query := dcb.NewQueryBuilder().
    WithTag("user_id", "123").
    WithType("UserCreated").
    Build()

condition := dcb.FailIfExists("user_id", "123")

// 4. Use either interface as needed
err = store.AppendIf(ctx, []dcb.InputEvent{event}, condition)  // Fluent conditional append
err = store.Append(ctx, events, &condition)  // Direct usage with pointer
err = commandExecutor.ExecuteCommand(ctx, command, handler, &condition)  // Command-driven
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
- Automatic acquisition: Database functions acquire locks on these keys before DCB condition checks
- Deadlock prevention: Locks sorted and acquired in consistent order
- Transaction-scoped: Automatically released on commit/rollback
- Performance: 1 I/O operation when used alone, 2 I/O operations when combined with DCB conditions
- Use case: Resource serialization without complex business logic validation

## Fluent API

The library provides a fluent API for common operations, reducing boilerplate by 50%:

### EventBuilder
```go
event := dcb.NewEvent("UserCreated").
    WithTag("user_id", "123").
    WithTags(map[string]string{
        "tenant": "acme",
        "version": "1.0",
    }).
    WithData(userData).
    Build()
```

### QueryBuilder
```go
query := dcb.NewQueryBuilder().
    WithTag("user_id", "123").
    WithType("UserCreated").
    AddItem().
    WithTag("user_id", "456").
    WithType("UserProfileUpdated").
    Build()
```

### Simplified AppendConditions
```go
condition := dcb.FailIfExists("user_id", "123")
condition := dcb.FailIfEventType("UserRegistered", "user_id", "123")
```

### Projection Helpers
```go
projector := dcb.ProjectCounter("user_count", "UserRegistered", "status", "active")
projector := dcb.ProjectBoolean("user_exists", "UserRegistered", "user_id", "123")
```

### BatchBuilder
```go
batch := dcb.NewBatch().
    AddEvent(event1).
    AddEvent(event2).
    AddEventFromBuilder(eventBuilder).
    Build()
```

### Convenience Functions
```go
// Append single event with tags
err := dcb.AppendSingleEvent(ctx, store, "UserLogin", map[string]string{
    "user_id": "123",
    "ip": "192.168.1.1",
}, loginData)

// Append single event with condition
err := dcb.AppendSingleEventIf(ctx, store, "UserProfileUpdated", 
    map[string]string{"user_id": "123"}, 
    userData, 
    dcb.FailIfExists("user_id", "123"))
```

See the [Quick Start](quick-start.md) and [Examples](../internal/examples/) for complete usage examples.

## Migration from Legacy API

If you're familiar with the legacy API, here's how to migrate to the new fluent API:

### Event Creation
```go
// Legacy way
event := dcb.NewInputEvent("UserCreated", dcb.NewTags("user_id", "123"), dcb.ToJSON(userData))

// New way
event := dcb.NewEvent("UserCreated").
    WithTag("user_id", "123").
    WithData(userData).
    Build()
```

### Query Building
```go
// Legacy way
query := dcb.NewQuery(dcb.NewTags("user_id", "123"), "UserCreated")

// New way
query := dcb.NewQueryBuilder().
    WithTag("user_id", "123").
    WithType("UserCreated").
    Build()
```

### Append Conditions
```go
// Legacy way
condition := dcb.NewAppendCondition(dcb.NewQuery(dcb.NewTags("user_id", "123"), "UserCreated"))

// New way
condition := dcb.FailIfExists("user_id", "123")
```

### Projections
```go
// Legacy way
projector := dcb.StateProjector{
    ID: "user_count",
    Query: dcb.NewQuery(dcb.NewTags("status", "active"), "UserRegistered"),
    InitialState: 0,
    TransitionFn: func(state any, event dcb.Event) any { return state.(int) + 1 },
}

// New way
projector := dcb.ProjectCounter("user_count", "UserRegistered", "status", "active")
```

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
// Define projectors using fluent API
projectors := []dcb.StateProjector{
    dcb.ProjectBoolean("courseExists", "CourseDefined", "course_id", courseID),
    dcb.ProjectCounter("numSubscriptions", "StudentEnrolled", "course_id", courseID),
}

states, appendCond, _ := store.Project(ctx, projectors, nil)

if !states["courseExists"].(bool) {
    // Create course using fluent API
    courseEvent := dcb.NewEvent("CourseDefined").
        WithTag("course_id", courseID).
        WithData(CourseDefined{courseID, 2}).
        Build()
    store.Append(ctx, []dcb.InputEvent{courseEvent}, nil)
}

if states["numSubscriptions"].(int) < 2 {
    // Enroll student using fluent API
    enrollmentEvent := dcb.NewEvent("StudentEnrolled").
        WithTag("student_id", studentID).
        WithTag("course_id", courseID).
        WithData(StudentEnrolled{studentID, courseID}).
        Build()
    store.AppendIf(ctx, []dcb.InputEvent{enrollmentEvent}, appendCond)
}
```

## Performance

See [benchmarks documentation](benchmarks.md) for detailed performance analysis.

## Table Validation

The library validates that the `events` table exists and has the correct structure:
- Required columns: `type`, `tags`, `data`, `transaction_id`, `position`, `occurred_at`
- Returns `TableStructureError` with detailed validation failure information
