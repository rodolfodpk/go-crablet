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
    // ReadEvents reads events matching the query with optional configuration.
    // Returns an EventIterator for streaming events efficiently.
    ReadEvents(ctx context.Context, query Query, options *ReadOptions) (EventIterator, error)
    
    // AppendEvents adds multiple events to the stream and returns the latest position.
    // It ensures that no conflicting events have been appended since latestKnownPosition
    // for the given query, maintaining consistency boundaries.
    // Returns the new latest position or an error if the append fails.
    AppendEvents(ctx context.Context, events []InputEvent, query Query, latestKnownPosition int64) (int64, error)
    
    // AppendEventsIf appends events only if no events match the append condition.
    // It uses the condition to enforce consistency by failing if any events match the query
    // after the specified position (if any).
    AppendEventsIf(ctx context.Context, events []InputEvent, condition AppendCondition) (int64, error)
    
    // GetCurrentPosition returns the current position for the given query.
    // This is a convenience method, not required by DCB spec.
    GetCurrentPosition(ctx context.Context, query Query) (int64, error)
    
    // ProjectState projects the current state using the provided projector.
    // It streams events from PostgreSQL that match the projector's query,
    // applying the transition function to build the current state.
    // Returns the latest position processed, the final state, and any error.
    ProjectState(ctx context.Context, projector StateProjector) (int64, any, error)
    
    // ProjectStateUpTo computes a state by streaming events matching the projector's query, up to maxPosition.
    ProjectStateUpTo(ctx context.Context, projector StateProjector, maxPosition int64) (int64, any, error)
    
    // ProjectBatch projects multiple states using multiple projectors in a single database query.
    // This is more efficient than calling ProjectState multiple times as it uses one combined query
    // and streams events once, routing them to the appropriate projectors.
    // Returns the latest position processed and a map of projector results keyed by projector ID.
    ProjectBatch(ctx context.Context, projectors []BatchProjector) (BatchProjectionResult, error)
    
    // ProjectBatchUpTo projects multiple states up to a specific position using multiple projectors.
    // Similar to ProjectBatch but limits the events processed to those up to maxPosition.
    ProjectBatchUpTo(ctx context.Context, projectors []BatchProjector, maxPosition int64) (BatchProjectionResult, error)
}

// EventIterator provides a streaming interface for reading events
type EventIterator interface {
    // Next returns the next event in the stream
    // Returns nil when no more events are available
    Next() (*Event, error)
    
    // Close closes the iterator and releases resources
    Close() error
    
    // Position returns the position of the last event read
    Position() int64
}

// ReadOptions provides configuration for reading events
type ReadOptions struct {
    FromPosition int64  // Start reading from this position (inclusive)
    Limit        int    // Maximum number of events to return (0 = no limit)
    OrderBy      string // Ordering: "asc" (default) or "desc"
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

// QueryItem represents a single query constraint
type QueryItem struct {
    Types []string // Event types to match (empty means match any type)
    Tags  []Tag    // Tags that must all be present (empty means match any tags)
}

// Query defines criteria for selecting events.
// DCB spec allows multiple QueryItems combined with OR logic.
type Query struct {
    Items []QueryItem // Query items combined with OR logic
}

// Tag is a key-value pair for querying events
type Tag struct {
    Key   string
    Value string
}
```

## Streaming Event Reading

go-crablet implements memory-efficient event reading through streaming:

```go
// Create a query for account events
query := dcb.NewQuery(
	dcb.NewTags("account_id", "acc-123"),
	"AccountRegistered", "AccountDetailsChanged",
)

// Read events using streaming interface
iterator, err := store.ReadEvents(ctx, query, nil)
if err != nil {
    return err
}
defer iterator.Close()

// Process events one by one
for {
    event, err := iterator.Next()
    if err != nil {
        return err
    }
    if event == nil {
        break // No more events
    }
    
    // Process the event without loading all events into memory
    fmt.Printf("Event: %s at position %d\n", event.Type, event.Position)
}
```

## Complex Query Support

The new Query structure supports complex queries with multiple items:

```go
// Query for events that are either:
// 1. Account events for account "acc-123"
// 2. Transaction events for account "acc-123"
// 3. Any events tagged with "user_id" = "user-456"
query := dcb.NewQueryFromItems(
    dcb.NewQueryItem(
        []string{"AccountRegistered", "AccountDetailsChanged"},
        dcb.NewTags("account_id", "acc-123"),
    ),
    dcb.NewQueryItem(
        []string{"TransactionCompleted"},
        dcb.NewTags("account_id", "acc-123"),
    ),
    dcb.NewQueryItem(
        []string{}, // Any event type
        dcb.NewTags("user_id", "user-456"),
    ),
)
```

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