# Overview: Dynamic Consistency Boundary (DCB) in go-crablet

go-crablet is a Go library for event sourcing, exploring concepts inspired by the [Dynamic Consistency Boundary (DCB)](https://dcb.events/) pattern. We're exploring how DCB might enable you to:

- Project multiple states and check business invariants in a single query
- Use tag-based, OR-combined queries for cross-entity consistency
- Enforce optimistic concurrency with combined append conditions

## Key Concepts

- **Batch Projection**: Project multiple states (decision model) in one database query
- **Combined Append Condition**: Use a single, OR-combined query for optimistic locking
- **Tag-based Queries**: Flexible, cross-entity queries using tags
- **Streaming**: Process events efficiently for large datasets
- **Transaction-based Ordering**: Uses PostgreSQL transaction IDs for true event ordering

## Core Interface

```go
type EventStore interface {
    // Query reads events matching the query with optional cursor
    // cursor == nil: query from beginning of stream
    // cursor != nil: query from specified cursor position (EXCLUSIVE - events after cursor, not including cursor)
    Query(ctx context.Context, query Query, cursor *Cursor) ([]Event, error)

    // QueryStream creates a channel-based stream of events matching a query with optional cursor
    // cursor == nil: stream from beginning of stream
    // cursor != nil: stream from specified cursor position (EXCLUSIVE - events after cursor, not including cursor)
    // This is optimized for large datasets and provides backpressure through channels
    // for efficient memory usage and Go-idiomatic streaming
    QueryStream(ctx context.Context, query Query, cursor *Cursor) (<-chan Event, error)

    // Append appends events to the store with optional condition
    // condition == nil: unconditional append
    // condition != nil: conditional append (optimistic locking)
    Append(ctx context.Context, events []InputEvent, condition *AppendCondition) error

    // Project projects multiple states using projectors with optional cursor
    // cursor == nil: project from beginning of stream
    // cursor != nil: project from specified cursor position (EXCLUSIVE - events after cursor, not including cursor)
    // Returns final aggregated states and append condition for optimistic locking
    Project(ctx context.Context, projectors []StateProjector, cursor *Cursor) (map[string]any, AppendCondition, error)

    // ProjectStream projects multiple states using channel-based streaming with optional cursor
    // cursor == nil: stream from beginning of stream
    // cursor != nil: stream from specified cursor position (EXCLUSIVE - events after cursor, not including cursor)
    // This is optimized for large datasets and provides backpressure through channels
    // for efficient memory usage and Go-idiomatic streaming
    ProjectStream(ctx context.Context, projectors []StateProjector, cursor *Cursor) (<-chan map[string]any, <-chan AppendCondition, error)

    // GetConfig returns the current EventStore configuration
    GetConfig() EventStoreConfig
}

type Cursor struct {
    TransactionID uint64 `json:"transaction_id"`
    Position      int64  `json:"position"`
}
```

## Transaction ID Ordering

go-crablet uses PostgreSQL's `xid8` transaction IDs for event ordering and optimistic locking:

- **True ordering**: No gaps or out-of-order events
- **Optimistic locking**: Uses transaction IDs for conflict detection
- **Cursor-based**: Combines `(transaction_id, position)` for precise positioning

## DCB Decision Model Pattern

We're exploring how a Dynamic Consistency Boundary decision model might work:

1. Define projectors for each business rule or invariant
2. Project all states in a single query
3. Build a combined append condition
4. Append new events only if all invariants still hold

## Example: Course Subscription

```go
projectors := []dcb.StateProjector{
    {
        ID: "courseExists",
        Query: dcb.NewQuery(dcb.NewTags("course_id", courseID), "CourseDefined"),
        InitialState: false,
        TransitionFn: func(state any, event dcb.Event) any {
            return true // If we see a CourseDefined event, course exists
        },
    },
    {
        ID: "numSubscriptions",
        Query: dcb.NewQuery(dcb.NewTags("course_id", courseID), "StudentEnrolled"),
        InitialState: 0,
        TransitionFn: func(state any, event dcb.Event) any {
            return state.(int) + 1
        },
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

## Transaction Isolation Levels

go-crablet uses configurable PostgreSQL transaction isolation levels:

- **Append (unconditional)**: Uses the default isolation level configured in `EventStoreConfig` (typically Read Committed)
- **Append (conditional)**: Uses the default isolation level configured in `EventStoreConfig` (typically Repeatable Read)

The isolation level can be configured when creating the EventStore via `EventStoreConfig.DefaultAppendIsolation`.

## Configuration

The EventStore can be configured with various settings:

```go
type EventStoreConfig struct {
    MaxBatchSize           int            `json:"max_batch_size"`           // Maximum events per batch
    LockTimeout            int            `json:"lock_timeout"`             // Lock timeout in milliseconds for advisory locks
    StreamBuffer           int            `json:"stream_buffer"`            // Channel buffer size for streaming operations
    DefaultAppendIsolation IsolationLevel `json:"default_append_isolation"` // Default isolation level for Append operations
    QueryTimeout           int            `json:"query_timeout"`            // Query timeout in milliseconds (defensive against hanging queries)
    AppendTimeout          int            `json:"append_timeout"`           // Append timeout in milliseconds (defensive against hanging appends)
    TargetEventsTable      string         `json:"target_events_table"`      // Target events table name (default: "events")
}
```

### Default Values
- `MaxBatchSize`: 1000 events
- `LockTimeout`: 5000ms (5 seconds)
- `StreamBuffer`: 1000 events
- `DefaultAppendIsolation`: Read Committed
- `QueryTimeout`: 15000ms (15 seconds)
- `AppendTimeout`: 10000ms (10 seconds)
- `TargetEventsTable`: "events"

## Performance Comparison Across Isolation Levels

Benchmark results from web-app load testing (30-second tests, multiple VUs):

| Metric | Append (unconditional) | Append (conditional) |
|--------|------------------------|---------------------------|
| **Throughput** | 79.2 req/s | 61.7 req/s |
| **Avg Response Time** | 24.87ms | 12.82ms |
| **p95 Response Time** | 49.16ms | 21.86ms |
| **Success Rate** | 100% | 100% |
| **VUs** | 10 | 10 |
| **Use Case** | Simple appends | Conditional appends |

### Key Performance Insights

- **Conditional append is fastest**: Conditional appends with Repeatable Read isolation actually perform better than simple appends
- **Excellent reliability**: Both isolation levels achieve 100% success rate
- **Optimized implementation**: Cursor-based optimistic locking and SQL functions are highly efficient

### When to Use Each Method

- **Append (nil condition)**: Use for simple event appends where no conditions are needed
- **Append (with condition)**: Use for conditional appends requiring optimistic locking

## Table Validation

When creating an EventStore with a custom `TargetEventsTable`, the library validates that the table exists and has the correct structure:

- **Required columns**: `type`, `tags`, `data`, `transaction_id`, `position`, `occurred_at`
- **Data types**: Validates column types and nullable constraints
- **Error handling**: Returns `TableStructureError` with detailed information about validation failures

Example validation errors:
- `table nonexistent_events does not exist`
- `missing required column 'occurred_at'`
- `column 'tags' should be ARRAY type, got TEXT`

## Command Execution

go-crablet supports command execution with automatic event generation based on current state:

```go
type Command interface {
    GetType() string
    GetData() []byte
    GetMetadata() map[string]interface{}
}

type CommandHandler interface {
    Handle(ctx context.Context, decisionModels map[string]any, command Command) []InputEvent
}

// ExecuteCommand executes a command and generates events based on current state
ExecuteCommand(ctx context.Context, command Command, handler CommandHandler, condition *AppendCondition) error
```

### Command Execution Flow

1. **Store command** in the `commands` table with transaction ID
2. **Get current state** using existing projectors (if condition provided)
3. **Generate events** using the command handler
4. **Append events** atomically within the same transaction

### Basic Usage

```go
// Create command
cmd := NewCommand("CreateUser", ToJSON(userData), map[string]interface{}{
    "correlation_id": "corr_789",
    "source": "web_api",
})

// Define command handler
type CreateUserHandler struct{}

func (h *CreateUserHandler) Handle(ctx context.Context, decisionModels map[string]any, command Command) []InputEvent {
    // Extract command data
    var cmdData CreateUserCommand
    json.Unmarshal(command.GetData(), &cmdData)
    
    // Check current state
    if userState, ok := decisionModels["user_state"].(UserState); ok {
        if userState.UserExists(cmdData.Email) {
            return []InputEvent{
                NewInputEvent("UserCreationFailed", 
                    NewTags("email", cmdData.Email, "reason", "user_exists"), 
                    ToJSON(map[string]string{"error": "User already exists"})),
            }
        }
    }
    
    // Generate success events
    return []InputEvent{
        NewInputEvent("UserCreated", 
            NewTags("email", cmdData.Email), 
            ToJSON(userCreatedData)),
    }
}

// Execute command
handler := &CreateUserHandler{}
err := eventStore.ExecuteCommand(ctx, cmd, handler, &condition)
```

### Type Safety for Decision Models

The `decisionModels` parameter is `map[string]any` for flexibility, but you can add type safety:

#### Option 1: Type Assertion in Handler
```go
func (h *MyHandler) Handle(ctx context.Context, decisionModels map[string]any, command Command) []InputEvent {
    // Type-safe access with assertion
    if userState, ok := decisionModels["user_state"].(UserState); ok {
        if userState.UserExists(email) {
            // Type-safe access
        }
    }
    return events
}
```

#### Option 2: Typed Wrapper
```go
// Define your typed decision models
type MyDecisionModels struct {
    UserState   UserState
    CourseState CourseState
}

// Create typed wrapper
func (h *MyHandler) getTypedModels(raw map[string]any) MyDecisionModels {
    return MyDecisionModels{
        UserState:   raw["user_state"].(UserState),
        CourseState: raw["course_state"].(CourseState),
    }
}

// Use typed models in handler
func (h *MyHandler) Handle(ctx context.Context, decisionModels map[string]any, command Command) []InputEvent {
    typed := h.getTypedModels(decisionModels)
    return h.handleTyped(typed, command)
}

func (h *MyHandler) handleTyped(models MyDecisionModels, command Command) []InputEvent {
    // Full type safety with IDE support
    if models.UserState.UserExists(email) {
        // IDE autocomplete works!
    }
    return events
}
```

#### Option 3: Generic Helper
```go
// Generic helper for type-safe access
func getState[T any](decisionModels map[string]any, key string) (T, bool) {
    if value, exists := decisionModels[key]; exists {
        if typedValue, ok := value.(T); ok {
            return typedValue, true
        }
    }
    var zero T
    return zero, false
}

// Usage in handler
func (h *MyHandler) Handle(ctx context.Context, decisionModels map[string]any, command Command) []InputEvent {
    if userState, ok := getState[UserState](decisionModels, "user_state"); ok {
        if userState.UserExists(email) {
            // Type-safe access
        }
    }
    return events
}
```

## Implementation Details

- **Database**: PostgreSQL with events table, commands table, and append functions
- **Streaming**: Multiple approaches for different dataset sizes
- **Extensions**: Channel-based streaming for Go-idiomatic processing
- **Commands**: Atomic command execution with event generation

See [examples](examples.md) for complete working examples including course subscriptions and money transfers, and [getting-started](getting-started.md) for setup instructions.