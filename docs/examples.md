# DCB-Inspired Example: Course Subscription with Invariants

This example demonstrates our exploration of the Dynamic Consistency Boundary (DCB) pattern using go-crablet. It shows how we're experimenting with:
- Project multiple states (decision model) in a single query
- Enforce business invariants (course exists, not full, student not already subscribed)
- Use a combined append condition for DCB concurrency control (uses transaction IDs, not classic optimistic locking)
- Use channel-based streaming for Go-idiomatic event processing

## Example: Course Subscription Command Handler

### Traditional Cursor-Based Approach

```go
package main

import (
    "context"
    "encoding/json"
    "github.com/rodolfodpk/go-crablet/pkg/dcb"
    "github.com/jackc/pgx/v5/pgxpool"
    "time"
)

type CourseDefined struct {
    CourseID string
    Capacity int
}

type StudentSubscribed struct {
    StudentID string
    CourseID  string
}

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    pool, _ := pgxpool.New(ctx, "postgres://user:pass@localhost/db")
    store, _ := dcb.NewEventStore(ctx, pool)

    courseID := "c1"
    studentID := "s1"

    // Define projectors for the decision model using fluent API
    projectors := []dcb.StateProjector{
        dcb.ProjectBoolean("courseExists", "CourseDefined", "course_id", courseID),
        dcb.ProjectCounter("numSubscriptions", "StudentSubscribed", "course_id", courseID),
        dcb.ProjectBoolean("alreadySubscribed", "StudentSubscribed", "student_id", studentID),
    }

    // Project all states in single query (cursor-based approach)
    states, appendCond, err := store.Project(ctx, projectors, nil)

    if !states["courseExists"].(bool) {
        // Append CourseDefined event using fluent API
        event := dcb.NewEvent("CourseDefined").
            WithTag("course_id", courseID).
            WithData(CourseDefined{courseID, 2}).
            Build()
        store.Append(ctx, []dcb.InputEvent{event}, nil)
    }
    if states["alreadySubscribed"].(bool) {
        panic("student already subscribed")
    }
    if states["numSubscriptions"].(int) >= 2 {
        panic("course is full")
    }
    // Subscribe student using fluent API
    event := dcb.NewEvent("StudentSubscribed").
        WithTag("student_id", studentID).
        WithTag("course_id", courseID).
        WithData(StudentSubscribed{studentID, courseID}).
        Build()
    store.Append(ctx, []dcb.InputEvent{event}, &appendCond)
}
```

### Command-Driven Approach with Function-Based Handler

```go
// Define command handler function
func handleSubscribeStudent(ctx context.Context, store dcb.EventStore, command dcb.Command) ([]dcb.InputEvent, error) {
    var cmdData SubscribeStudentCommand
    if err := json.Unmarshal(command.GetData(), &cmdData); err != nil {
        return nil, fmt.Errorf("failed to unmarshal command: %w", err)
    }

    courseID := cmdData.CourseID
    studentID := cmdData.StudentID

    // Define projectors for the decision model
    projectors := []dcb.StateProjector{
        {
            ID: "courseExists",
            Query: dcb.NewQuery(dcb.NewTags("course_id", courseID), "CourseDefined"),
            InitialState: false,
            TransitionFn: func(state any, e dcb.Event) any { return true },
        },
        {
            ID: "numSubscriptions",
            Query: dcb.NewQuery(dcb.NewTags("course_id", courseID), "StudentSubscribed"),
            InitialState: 0,
            TransitionFn: func(state any, e dcb.Event) any { return state.(int) + 1 },
        },
        {
            ID: "alreadySubscribed",
            Query: dcb.NewQuery(dcb.NewTags("student_id", studentID, "course_id", courseID), "StudentSubscribed"),
            InitialState: false,
            TransitionFn: func(state any, e dcb.Event) any { return true },
        },
    }

    // Project all states in single query
    states, appendCond, err := store.Project(ctx, projectors, nil)
    if err != nil {
        return nil, fmt.Errorf("failed to project state: %w", err)
    }

    // Check business rules
    if !states["courseExists"].(bool) {
        return nil, fmt.Errorf("course %s does not exist", courseID)
    }

    if states["alreadySubscribed"].(bool) {
        return nil, fmt.Errorf("student %s already subscribed to course %s", studentID, courseID)
    }

    if states["numSubscriptions"].(int) >= 2 {
        return nil, fmt.Errorf("course %s is full", courseID)
    }

    // Generate success event
    data, _ := json.Marshal(StudentSubscribed{studentID, courseID})
    return []dcb.InputEvent{
        dcb.NewInputEvent("StudentSubscribed",
            dcb.NewTags("student_id", studentID, "course_id", courseID),
            data),
    }, nil
}

// Usage with CommandExecutor
func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    pool, _ := pgxpool.New(ctx, "postgres://user:pass@localhost/db")
    store, _ := dcb.NewEventStore(ctx, pool)
    commandExecutor := dcb.NewCommandExecutor(store)

    // Create command
    cmdData := SubscribeStudentCommand{
        StudentID: "s1",
        CourseID:  "c1",
    }
    cmd := dcb.NewCommand("SubscribeStudent", dcb.ToJSON(cmdData), nil)

    // Execute command using function-based handler
    _, err := commandExecutor.ExecuteCommand(ctx, cmd, dcb.CommandHandlerFunc(handleSubscribeStudent), nil)
    if err != nil {
        panic(err)
    }
}
```

### Channel-Based Approach (New!)

```go
func channelBasedExample() {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    pool, _ := pgxpool.New(ctx, "postgres://user:pass@localhost/db")
    store, _ := dcb.NewEventStore(ctx, pool)

    courseID := "c1"
    studentID := "s1"

    // Define the same projectors
    projectors := []dcb.StateProjector{
        {
            ID: "courseExists",
            Query: dcb.NewQuery(dcb.NewTags("course_id", courseID), "CourseDefined"),
            InitialState: false,
            TransitionFn: func(state any, e dcb.Event) any { return true },
        },
        {
            ID: "numSubscriptions",
            Query: dcb.NewQuery(dcb.NewTags("course_id", courseID), "StudentSubscribed"),
            InitialState: 0,
            TransitionFn: func(state any, e dcb.Event) any { return state.(int) + 1 },
        },
        {
            ID: "alreadySubscribed",
            Query: dcb.NewQuery(dcb.NewTags("student_id", studentID, "course_id", courseID), "StudentSubscribed"),
            InitialState: false,
            TransitionFn: func(state any, e dcb.Event) any { return true },
        },
    }

    // Channel-based projection with immediate feedback
    stateChan, _, err := store.ProjectStream(ctx, projectors, nil)
    if err != nil {
        panic(err)
    }
    var finalStates map[string]any
    for states := range stateChan {
        finalStates = states
    }

    // Apply business rules using final states
    if !finalStates["courseExists"].(bool) {
        // Append CourseDefined event
    }
    if finalStates["alreadySubscribed"].(bool) {
        panic("student already subscribed")
    }
    if finalStates["numSubscriptions"].(int) >= 2 {
        panic("course is full")
    }
}
```

### Channel-Based Event Streaming

```go
func channelStreamingExample() {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    pool, _ := pgxpool.New(ctx, "postgres://user:pass@localhost/db")
    store, _ := dcb.NewEventStore(ctx, pool)

    // Create query for course events
    query := dcb.NewQuery(
        dcb.NewTags("course_id", "c1"), "CourseDefined",
        dcb.NewTags("student_id", "s1"), "StudentRegistered",
        dcb.NewTags("course_id", "c1", "student_id", "s1"), "StudentSubscribed",
    )

    // Channel-based streaming
    eventChan, err := store.QueryStream(ctx, query, nil)
    if err != nil {
        panic(err)
    }

    // Process events with immediate delivery
    for event := range eventChan {
        fmt.Printf("Event: %s at position %d\n", event.Type, event.Position)
        // Process event based on type
        switch event.Type {
        case "CourseDefined":
            fmt.Println("Course was defined")
        case "StudentSubscribed":
            fmt.Println("Student was subscribed")
        }
    }
}
```

## Example: Money Transfer with DCB Concurrency Control

This example demonstrates concurrent money transfers with DCB concurrency control to prevent double-spending. The transfer example uses a flat structure for simplicity and consistency with other examples.

### Project Structure
```
internal/examples/transfer/
└── main.go              # Complete example with types, handlers, and main function
```

```go
func handleTransferMoney(ctx context.Context, store dcb.EventStore, cmd TransferMoneyCommand) error {
    // Project state for the "from" account
    projectors := []dcb.StateProjector{
        {
            ID: "from",
            Query: dcb.NewQuery(dcb.NewTags("account_id", cmd.FromAccountID), "AccountOpened", "MoneyTransferred"),
            InitialState: &AccountState{AccountID: cmd.FromAccountID},
            TransitionFn: func(state any, event dcb.Event) any {
                acc := state.(*AccountState)
                switch event.Type {
                case "AccountOpened":
                    var data AccountOpened
                    if err := json.Unmarshal(event.Data, &data); err == nil {
                        acc.Owner = data.Owner
                        acc.Balance = data.InitialBalance
                        acc.OccurredAt = data.OpenedAt
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
        },
    }
    states, appendCond, err := store.Project(ctx, projectors, nil)
    if err != nil {
        return fmt.Errorf("projection failed: %w", err)
    }
    from := states["from"].(*AccountState)
    if from.Balance < cmd.Amount {
        return fmt.Errorf("insufficient funds: account %s has %d, needs %d", cmd.FromAccountID, from.Balance, cmd.Amount)
    }
    // ...
    // Use appendCond for DCB concurrency control
    // ...
    return nil
}
```

**Key features:**
- **Business logic validation**: Checks sufficient funds before transfer
- **DCB concurrency control**: Uses `Append` with conditions and configurable isolation level (primary mechanism, not classic optimistic locking; transaction IDs ensure correct event ordering, inspired by Oskar’s article)
- **Advisory locks**: Optional additional locking via `lock:` prefixed tags (e.g., "lock:account-123")
- **Concurrent safety**: Only one transfer can succeed when multiple try to spend the same balance
- **Event sourcing**: All transfers are recorded as events for audit trail

## Key Points We're Exploring

- **All invariants are checked in a single query** (multiple state projection)
- **The append condition is the OR-combination of all projector queries**
- **Only one database round trip is needed for all business rules**
- **No aggregates or legacy event sourcing patterns required**
- **Channel-based streaming provides immediate processing feedback**
- **Choose the right streaming approach for your dataset size**
- **DCB concurrency control prevents double-spending** in concurrent scenarios (not classic optimistic locking; transaction IDs ensure correct event ordering)

## Transaction Isolation Levels and Locking

### Primary: DCB Concurrency Control (Not Classic Optimistic Locking)
go-crablet primarily uses DCB concurrency control via transaction IDs and append conditions (not classic optimistic locking):

```go
// Simple append (no conditions) - uses default isolation level
store.Append(ctx, events, nil)

// Conditional append - uses default isolation level with DCB concurrency control
store.Append(ctx, events, &condition)
```

### Optional: Advisory Locks (Experimental)
For additional concurrency control, you can use advisory locks via `lock:` prefixed tags (experimental, not enabled by default):

```go
// Event with advisory lock on "account-123"
event := dcb.NewInputEvent("MoneyTransfer",
    dcb.NewTags("account_id", "123", "lock:account-123"),
    dcb.ToJSON(transferData))

// Multiple events with different locks
events := []dcb.InputEvent{
    dcb.NewInputEvent("DebitAccount",
        dcb.NewTags("account_id", "123", "lock:account-123"),
        dcb.ToJSON(debitData)),
    dcb.NewInputEvent("CreditAccount",
        dcb.NewTags("account_id", "456", "lock:account-456"),
        dcb.ToJSON(creditData)),
}
```

**Note**: Advisory locks are now fully implemented and available in the Go implementation via `lock:` prefixed tags. When `lock:` tags are present, advisory locks are acquired FIRST, then DCB concurrency checks are performed. Both mechanisms work together for comprehensive concurrency control.

### Isolation Levels
go-crablet uses configurable PostgreSQL transaction isolation levels for append operations:

```go
// Simple append (no conditions) - uses default isolation level
store.Append(ctx, events, nil)

// Conditional append - uses default isolation level
store.Append(ctx, events, &condition)
```

**When to use different methods:**
- **Append (nil condition)**: Fastest, safe for simple appends
- **Append (with condition)**: Good for conditional appends, prevents phantom reads with DCB concurrency control

The isolation level and other settings can be configured when creating the EventStore via `EventStoreConfig`:

```go
config := dcb.EventStoreConfig{
    MaxBatchSize:           1000, // Limits events per append call
    LockTimeout:            5000, // ms
    StreamBuffer:           1000,
    DefaultAppendIsolation: dcb.IsolationLevelReadCommitted,
    QueryTimeout:           15000, // ms
    AppendTimeout:          10000, // ms
}
store, err := dcb.NewEventStoreWithConfig(ctx, pool, config)
```

## Query Building with Helper Functions

go-crablet provides concise helper functions to simplify query building:

### Using QItem and QItemKV Helpers

**Before (verbose):**
```go
Query: dcb.NewQuerySimple(dcb.NewTags("course_id", courseID), "CourseDefined")
```

**After (concise):**
```go
Query: dcb.NewQueryFromItems(dcb.QItemKV("CourseDefined", "course_id", courseID))
```

**Complete example with helpers:**
```go
// Define projectors using the new helper functions
projectors := []dcb.StateProjector{
    {
        ID: "courseExists",
        Query: dcb.NewQueryFromItems(dcb.QItemKV("CourseDefined", "course_id", courseID)),
        InitialState: false,
        TransitionFn: func(state any, e dcb.Event) any { return true },
    },
    {
        ID: "numSubscriptions",
        Query: dcb.NewQueryFromItems(dcb.QItemKV("StudentSubscribed", "course_id", courseID)),
        InitialState: 0,
        TransitionFn: func(state any, e dcb.Event) any { return state.(int) + 1 },
    },
    {
        ID: "alreadySubscribed",
        Query: dcb.NewQueryFromItems(dcb.QItemKV("StudentSubscribed", "student_id", studentID, "course_id", courseID)),
        InitialState: false,
        TransitionFn: func(state any, e dcb.Event) any { return true },
    },
}
```

### Building Complex Queries

For more complex queries with multiple conditions:

```go
// Build a query with multiple event types and tags
query := dcb.NewQueryFromItems(
    dcb.QItemKV("CourseDefined", "course_id", "c1"),
    dcb.QItemKV("StudentRegistered", "student_id", "s1"),
    dcb.QItemKV("StudentSubscribed", "course_id", "c1", "student_id", "s1"),
)

// Read events with the combined query
events, err := store.Read(ctx, query)
```

## Performance Comparison

### Streaming Approaches

| Approach | Best For | Immediate Feedback | Memory Usage |
|----------|----------|-------------------|--------------|
| **Cursor-based** | Large datasets | ❌ No | Low |
| **Channel-based** | Small-medium datasets | ✅ Yes | Moderate |

### Isolation Level Performance

Benchmark results from web-app load testing (30-second tests):

| Method | Throughput | Avg Response Time | p95 Response Time | Use Case |
|--------|------------|------------------|------------------|----------|
| **Append** | 79.2 req/s | 24.87ms | 49.16ms | Simple appends |
| **Append (with condition)** | 61.7 req/s | 12.82ms | 21.86ms | Conditional appends |

**Key insights:**
- **Conditional appends are fastest**: Conditional appends with Repeatable Read perform better than simple appends
- **Excellent reliability**: Both methods achieve 100% success rate
- **Optimized implementation**: Cursor-based DCB concurrency control and SQL functions are highly efficient

## Available Examples

- **`internal/examples/cursor_streaming/`** - Cursor-based streaming for large datasets
- **`internal/examples/channel_streaming/`** - Channel-based streaming for small-medium datasets
- **`internal/examples/channel_projection/`** - Channel-based projection with immediate feedback
- **`internal/examples/extension_interface/`** - Extension interface pattern demonstration
- **`internal/examples/transfer/`** - Money transfer with DCB concurrency control (refactored with proper module structure)
- **`internal/examples/enrollment/`** - Course enrollment with business rules
- **`internal/examples/concurrency_comparison/`** - **NEW**: Performance comparison between DCB concurrency control and PostgreSQL advisory locks

## Concurrency Comparison Example

The `concurrency_comparison` example demonstrates the differences between DCB concurrency control and PostgreSQL advisory locks in a realistic concert ticket booking scenario:

### Key Features
- **N-user concurrency testing**: Simulates 50-100 concurrent users booking tickets
- **Performance comparison**: Benchmarks both approaches with timing and throughput metrics
- **Real-world scenario**: Concert ticket booking with limited seats
- **Comprehensive testing**: Shows both mechanisms working together

### Usage
```bash
# Run with default settings (100 users, 20 seats, 2 tickets per user)
go run internal/examples/concurrency_comparison/main.go

# Run with custom settings
go run internal/examples/concurrency_comparison/main.go -users 50 -seats 30 -tickets 1

# Test only advisory locks
go run internal/examples/concurrency_comparison/main.go -advisory-locks -users 50 -seats 30

# Test only DCB concurrency control
go run internal/examples/concurrency_comparison/main.go -users 50 -seats 30
```

### What It Demonstrates
1. **DCB Concurrency Control**: Uses `AppendCondition` to enforce business rules
2. **Advisory Locks**: Serialize access but don't enforce business limits without conditions
3. **Both Combined**: Serialize access AND enforce business rules
4. **Performance Metrics**: Throughput, success rates, and timing comparisons

### Test Results
The example shows:
- **DCB Concurrency Control**: Enforces business rules but may allow more than expected bookings
- **Advisory Locks**: Serialize access, ensuring sequential processing
- **Performance**: Both approaches have similar performance characteristics
- **Real-world Usage**: How to choose between the two approaches based on your needs

This example is particularly useful for understanding when to use each concurrency control mechanism and how they perform under realistic load.

All `Query` and `QueryItem` usage must go through the provided helper functions. Direct struct access is not possible. This enforces DCB compliance and improves type safety.
