# DCB-Inspired Example: Course Subscription with Invariants

This example demonstrates our exploration of the Dynamic Consistency Boundary (DCB) pattern using go-crablet. It shows how we're experimenting with:
- Project multiple states (decision model) in a single query
- Enforce business invariants (course exists, not full, student not already subscribed)
- Use a combined append condition for optimistic concurrency
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

    // Project all states in single query (cursor-based approach)
    states, appendCond, err := store.Project(ctx, projectors, nil)

    if !states["courseExists"].(bool) {
        // Append CourseDefined event
        data, _ := json.Marshal(CourseDefined{courseID, 2})
        event := dcb.NewInputEvent("CourseDefined", dcb.NewTags("course_id", courseID), data)
        store.Append(ctx, []dcb.InputEvent{event}, nil)
    }
    if states["alreadySubscribed"].(bool) {
        panic("student already subscribed")
    }
    if states["numSubscriptions"].(int) >= 2 {
        panic("course is full")
    }
    // Subscribe student
    data, _ := json.Marshal(StudentSubscribed{studentID, courseID})
    event := dcb.NewInputEvent("StudentSubscribed", dcb.NewTags("student_id", studentID, "course_id", courseID), data)
    store.Append(ctx, []dcb.InputEvent{event}, &appendCond)
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

## Example: Money Transfer with Optimistic Locking

This example demonstrates concurrent money transfers with optimistic locking to prevent double-spending:

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
    // Use appendCond for optimistic concurrency
    // ...
    return nil
}
```

**Key features:**
- **Business logic validation**: Checks sufficient funds before transfer
- **Optimistic locking**: Uses `Append` with conditions and configurable isolation level (primary mechanism)
- **Advisory locks**: Optional additional locking via `lock:` prefixed tags (e.g., `"lock:account-123"`)
- **Concurrent safety**: Only one transfer can succeed when multiple try to spend the same balance
- **Event sourcing**: All transfers are recorded as events for audit trail

## Key Points We're Exploring

- **All invariants are checked in a single query** (multiple state projection)
- **The append condition is the OR-combination of all projector queries**
- **Only one database round trip is needed for all business rules**
- **No aggregates or legacy event sourcing patterns required**
- **Channel-based streaming provides immediate processing feedback**
- **Choose the right streaming approach for your dataset size**
- **Optimistic locking prevents double-spending** in concurrent scenarios

## Transaction Isolation Levels and Locking

### Primary: Optimistic Locking
go-crablet primarily uses optimistic locking via transaction IDs and append conditions:

```go
// Simple append (no conditions) - uses default isolation level
store.Append(ctx, events, nil)

// Conditional append - uses default isolation level with optimistic locking
store.Append(ctx, events, &condition)
```

### Optional: Advisory Locks
For additional concurrency control, you can use advisory locks via `lock:` prefixed tags:

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

**Note**: Advisory locks are currently available in the database functions but not actively used by the Go implementation.

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
- **Append (with condition)**: Good for conditional appends, prevents phantom reads with optimistic locking

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
- **Optimized implementation**: Cursor-based optimistic locking and SQL functions are highly efficient

## Available Examples

- **`internal/examples/cursor_streaming/`** - Cursor-based streaming for large datasets
- **`internal/examples/channel_streaming/`** - Channel-based streaming for small-medium datasets
- **`internal/examples/channel_projection/`** - Channel-based projection with immediate feedback
- **`internal/examples/extension_interface/`** - Extension interface pattern demonstration
- **`internal/examples/transfer/`** - Event sourcing with semantic event names
- **`internal/examples/enrollment/`** - Course enrollment with business rules

All `Query` and `QueryItem` usage must go through the provided helper functions. Direct struct access is not possible. This enforces DCB compliance and improves type safety.
