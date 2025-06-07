# Implementation Details

## Stream Position Handling

When appending events to the store, it's crucial to use the current stream position rather than a fixed position (like 0). This ensures:

1. **Event Ordering**: Events are processed in the correct sequence
2. **Race Condition Prevention**: Concurrent updates are handled safely
3. **Consistency**: The final state reflects the most recent update

Example of proper stream position handling:

```go
// Get current stream position
query := dcb.NewQuery(dcb.NewTags("account_id", "acc123"))
position, err := store.GetCurrentPosition(ctx, query)
if err != nil {
    return err
}

// Append events using the current position
events := []dcb.InputEvent{
    {
        Type: "AccountBalanceUpdated",
        Tags: dcb.NewTags("account_id", "acc123"),
        Data: []byte(`{"balance": 1000}`),
    },
}
newPosition, err := store.AppendEvents(ctx, events, query, position)
```

## Event Store Interface

The core interface for event management:

```go
// EventStore provides methods to append and read events in a PostgreSQL database.
// It implements the Dynamic Consistency Boundary pattern, ensuring that events
// within the same boundary are processed atomically and maintaining consistency
// through optimistic locking.
type EventStore interface {
    // AppendEvents adds multiple events to the stream and returns the latest position.
    // It ensures that no conflicting events have been appended since latestKnownPosition
    // for the given query, maintaining consistency boundaries.
    // Returns the new latest position or an error if the append fails.
    AppendEvents(ctx context.Context, events []InputEvent, query Query, latestKnownPosition int64) (int64, error)
    
    // ProjectState projects the current state using the provided projector.
    // It streams events from PostgreSQL that match the projector's query,
    // applying the transition function to build the current state.
    // Returns the latest position processed, the final state, and any error.
    ProjectState(ctx context.Context, projector StateProjector) (int64, any, error)
}

// Event represents a persisted event in the system
type Event struct {
    ID            string // Unique event identifier (UUID)
    Type          string // Event type (e.g., "Subscription")
    Tags          []Tag  // Tags for querying (e.g., {"course_id": "C1"})
    Data          []byte // Event payload
    Position      int64  // Position in the event stream
    CausationID   string // UUID of the event that caused this event
    CorrelationID string // UUID linking to the root event or process
}

// InputEvent represents an event to be appended to the store
type InputEvent struct {
    Type string // Event type (e.g., "Subscription")
    Tags []Tag  // Tags for querying (e.g., {"course_id": "C1"})
    Data []byte // JSON-encoded event payload
}

// StateProjector defines how to project state from events
type StateProjector struct {
    // Query defines criteria for selecting events at the database level
    Query Query
    
    // InitialState is the starting state for the projection
    InitialState any
    
    // TransitionFn defines how to update state for each event
    TransitionFn func(state any, event Event) any
}

// Query defines criteria for selecting events
type Query struct {
    // Tags must match all specified tags (empty means match any tag)
    Tags []Tag
    
    // EventTypes must match one of these types (empty means match any type)
    EventTypes []string
}

// Tag is a key-value pair for querying events
type Tag struct {
    Key   string
    Value string
}
```

For a practical example of using go-crablet to implement a course subscription system, see [Course Subscription Example](course-subscription.md).

## State Projection

go-crablet implements efficient state projection by leveraging PostgreSQL's streaming capabilities. Instead of loading all events into memory, events are streamed directly from the database and processed one at a time. This approach provides several benefits:

1. **Memory Efficiency**: Events are processed in a streaming fashion, making it suitable for large event streams
2. **Database Efficiency**: Uses PostgreSQL's native JSONB indexing and querying capabilities
3. **Consistent Views**: The same query used for consistency checks is used for state projection

Example of state projection:

```go
// Create a projector for account balances
projector := dcb.StateProjector{
    Query: dcb.NewQuery(dcb.NewTags("account_id", "acc123")),
    InitialState: &AccountState{},
    TransitionFn: func(state any, event dcb.Event) any {
        // Handle events and update state
        return state
    },
}

// Project the current state
position, state, err := store.ProjectState(ctx, projector)
```

## Appending Events

go-crablet provides a robust mechanism for appending events with optimistic concurrency control. This ensures:

1. **Event Ordering**: Events are processed in the correct sequence
2. **Race Condition Prevention**: Concurrent updates are handled safely
3. **Consistency**: The final state reflects the most recent update

Example of appending events:

```go
// Get current stream position
query := dcb.NewQuery(dcb.NewTags("account_id", "acc123"))
position, err := store.GetCurrentPosition(ctx, query)
if err != nil {
    return err
}

// Create and append events
events := []dcb.InputEvent{
    {
        Type: "AccountBalanceUpdated",
        Tags: dcb.NewTags("account_id", "acc123"),
        Data: []byte(`{"balance": 1000}`),
    },
}

// Append events using the current position
newPosition, err := store.AppendEvents(ctx, events, query, position)
if err != nil {
    // Handle error - might be due to concurrent modification
    return err
}
```

The event store automatically handles optimistic concurrency control by:
1. Checking if the provided position matches the current stream position
2. Rejecting the append if there are concurrent modifications
3. Updating the stream position atomically with the event append 