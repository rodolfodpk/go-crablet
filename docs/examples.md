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
    projectors := []dcb.BatchProjector{
        {ID: "courseExists", StateProjector: dcb.StateProjector{
            Query: dcb.NewQuerySimple(dcb.NewTags("course_id", courseID), "CourseDefined"),
            InitialState: false,
            TransitionFn: func(state any, e dcb.Event) any { return true },
        }},
        {ID: "numSubscriptions", StateProjector: dcb.StateProjector{
            Query: dcb.NewQuerySimple(dcb.NewTags("course_id", courseID), "StudentSubscribed"),
            InitialState: 0,
            TransitionFn: func(state any, e dcb.Event) any { return state.(int) + 1 },
        }},
        {ID: "alreadySubscribed", StateProjector: dcb.StateProjector{
            Query: dcb.NewQuerySimple(dcb.NewTags("student_id", studentID, "course_id", courseID), "StudentSubscribed"),
            InitialState: false,
            TransitionFn: func(state any, e dcb.Event) any { return true },
        }},
    }

    // Project all states in single query (cursor-based approach)
    states, appendCond, err := store.ProjectDecisionModel(ctx, projectors)

    if !states["courseExists"].(bool) {
        // Append CourseDefined event
        data, _ := json.Marshal(CourseDefined{courseID, 2})
        event := dcb.NewInputEvent("CourseDefined", dcb.NewTags("course_id", courseID), data)
        store.Append(ctx, []dcb.InputEvent{event})
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
    store.AppendIf(ctx, []dcb.InputEvent{event}, appendCond)
}
```

### Channel-Based Approach (New!)

```go
func channelBasedExample() {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    pool, _ := pgxpool.New(ctx, "postgres://user:pass@localhost/db")
    store, _ := dcb.NewEventStore(ctx, pool)
    
    // Get channel-based store
    channelStore := store.(dcb.ChannelEventStore)

    courseID := "c1"
    studentID := "s1"

    // Define the same projectors
    projectors := []dcb.BatchProjector{
        {ID: "courseExists", StateProjector: dcb.StateProjector{
            Query: dcb.NewQuerySimple(dcb.NewTags("course_id", courseID), "CourseDefined"),
            InitialState: false,
            TransitionFn: func(state any, e dcb.Event) any { return true },
        }},
        {ID: "numSubscriptions", StateProjector: dcb.StateProjector{
            Query: dcb.NewQuerySimple(dcb.NewTags("course_id", courseID), "StudentSubscribed"),
            InitialState: 0,
            TransitionFn: func(state any, e dcb.Event) any { return state.(int) + 1 },
        }},
        {ID: "alreadySubscribed", StateProjector: dcb.StateProjector{
            Query: dcb.NewQuerySimple(dcb.NewTags("student_id", studentID, "course_id", courseID), "StudentSubscribed"),
            InitialState: false,
            TransitionFn: func(state any, e dcb.Event) any { return true },
        }},
    }

    // Channel-based projection with immediate feedback
    resultChan, _, err := channelStore.ProjectDecisionModelChannel(ctx, projectors)
    
    // Process results as they come in
    finalStates := make(map[string]interface{})
    for result := range resultChan {
        if result.Error != nil {
            fmt.Printf("Error: %v\n", result.Error)
            continue
        }
        
        finalStates[result.ProjectorID] = result.State
        
        fmt.Printf("Projector %s processed event %s (position %d)\n", 
            result.ProjectorID, result.Event.Type, result.Position)
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
    
    // Get channel-based store
    channelStore := store.(dcb.ChannelEventStore)

    // Create query for course events
    query := dcb.NewQueryFromItems(
        dcb.QItemKV("CourseDefined", "course_id", "c1"),
        dcb.QItemKV("StudentRegistered", "student_id", "s1"),
        dcb.QItemKV("StudentSubscribed", "course_id", "c1", "student_id", "s1"),
    )

    // Channel-based streaming
    eventChan, err := channelStore.ReadStreamChannel(ctx, query)
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
    projectors := []dcb.BatchProjector{
        {ID: "from", StateProjector: dcb.StateProjector{
            Query: dcb.NewQuery(dcb.NewTags("account_id", cmd.FromAccountID), "AccountOpened", "MoneyTransferred"),
            InitialState: &AccountState{AccountID: cmd.FromAccountID},
            TransitionFn: func(state any, event dcb.Event) any {
                acc := state.(*AccountState)
                switch event.Type {
                case "AccountOpened":
                    // Initialize account
                case "MoneyTransferred":
                    // Update balance
                }
                return acc
            },
        }},
    }
    
    states, appendCondition, err := store.ProjectDecisionModel(ctx, projectors)
    if err != nil {
        return err
    }
    
    from := states["from"].(*AccountState)
    if from.Balance < cmd.Amount {
        return fmt.Errorf("insufficient funds")
    }
    
    // Create transfer events
    events := []dcb.InputEvent{
        dcb.NewInputEvent("MoneyTransferred", dcb.NewTags("account_id", cmd.FromAccountID), data),
    }
    
    // Use optimistic locking to prevent double-spending
    return store.AppendIfIsolated(ctx, events, appendCondition)
}
```

**Key features:**
- **Business logic validation**: Checks sufficient funds before transfer
- **Optimistic locking**: Uses `AppendIfIsolated` with SERIALIZABLE isolation
- **Concurrent safety**: Only one transfer can succeed when multiple try to spend the same balance
- **Event sourcing**: All transfers are recorded as events for audit trail

## Key Points We're Exploring

- **All invariants are checked in a single query** (batch projection)
- **The append condition is the OR-combination of all projector queries**
- **Only one database round trip is needed for all business rules**
- **No aggregates or legacy event sourcing patterns required**
- **Channel-based streaming provides immediate processing feedback**
- **Choose the right streaming approach for your dataset size**
- **Optimistic locking prevents double-spending** in concurrent scenarios

## Transaction Isolation Levels

go-crablet uses the following PostgreSQL transaction isolation levels for append operations:

```go
// Simple append (no conditions) - Read Committed
store.Append(ctx, events)

// Conditional append - Repeatable Read
store.AppendIf(ctx, events, condition)

// Conditional append with strongest consistency - Serializable
store.AppendIfIsolated(ctx, events, condition)
```

**When to use different isolation levels:**
- **Read Committed** (`Append`): Fastest, safe for simple appends
- **Repeatable Read** (`AppendIf`): Good for most conditional appends, prevents phantom reads
- **Serializable** (`AppendIfIsolated`): Use for the strongest consistency guarantees

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
projectors := []dcb.BatchProjector{
    {ID: "courseExists", StateProjector: dcb.StateProjector{
        Query: dcb.NewQueryFromItems(dcb.QItemKV("CourseDefined", "course_id", courseID)),
        InitialState: false,
        TransitionFn: func(state any, e dcb.Event) any { return true },
    }},
    {ID: "numSubscriptions", StateProjector: dcb.StateProjector{
        Query: dcb.NewQueryFromItems(dcb.QItemKV("StudentSubscribed", "course_id", courseID)),
        InitialState: 0,
        TransitionFn: func(state any, e dcb.Event) any { return state.(int) + 1 },
    }},
    {ID: "alreadySubscribed", StateProjector: dcb.StateProjector{
        Query: dcb.NewQueryFromItems(dcb.QItemKV("StudentSubscribed", "student_id", studentID, "course_id", courseID)),
        InitialState: false,
        TransitionFn: func(state any, e dcb.Event) any { return true },
    }},
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

| Approach | Best For | Immediate Feedback | Memory Usage |
|----------|----------|-------------------|--------------|
| **Cursor-based** | Large datasets | ❌ No | Low |
| **Channel-based** | Small-medium datasets | ✅ Yes | Moderate |

## Available Examples

- **`internal/examples/cursor_streaming/`** - Cursor-based streaming for large datasets
- **`internal/examples/channel_streaming/`** - Channel-based streaming for small-medium datasets
- **`internal/examples/channel_projection/`** - Channel-based projection with immediate feedback
- **`internal/examples/extension_interface/`** - Extension interface pattern demonstration
- **`internal/examples/transfer/`** - Event sourcing with semantic event names
- **`internal/examples/enrollment/`** - Course enrollment with business rules

All `Query` and `QueryItem` usage must go through the provided helper functions. Direct struct access is not possible. This enforces DCB compliance and improves type safety.