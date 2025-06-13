# Implementation Details

## DCB Pattern Implementation

go-crablet implements the Dynamic Consistency Boundary (DCB) pattern using:

1. **Batch Projection**: Project multiple states in a single streamlined PostgreSQL query
2. **Combined Append Conditions**: Use the same query scope for projection and optimistic locking
3. **Streaming**: Process events row-by-row using PostgreSQL's native streaming via pgx

## Event Store Interface

The core interface for DCB-compliant event management:

```go
type EventStore interface {
    // Read reads events matching a query, optionally starting from a specified sequence position
    Read(ctx context.Context, query Query, options *ReadOptions) (SequencedEvents, error)

    // ReadStream returns a streaming iterator for events matching a query
    ReadStream(ctx context.Context, query Query, options *ReadOptions) (EventIterator, error)

    // Append atomically persists one or more events, optionally with an append condition
    Append(ctx context.Context, events []InputEvent, condition *AppendCondition) (int64, error)

    // ProjectDecisionModel projects multiple states using projectors and returns final states and append condition
    // This is the primary DCB API for building decision models in command handlers
    ProjectDecisionModel(ctx context.Context, query Query, options *ReadOptions, projectors []BatchProjector) (map[string]any, AppendCondition, error)
}
```

## DCB Decision Model Pattern

The core DCB pattern for command handlers:

```go
// 1. Define projectors for the decision model
projectors := []dcb.BatchProjector{
    {ID: "courseExists", StateProjector: dcb.StateProjector{
        Query: dcb.NewQuery(dcb.NewTags("course_id", courseID), "CourseDefined"),
        InitialState: false,
        TransitionFn: func(state any, e dcb.Event) any { return true },
    }},
    {ID: "numSubscriptions", StateProjector: dcb.StateProjector{
        Query: dcb.NewQuery(dcb.NewTags("course_id", courseID), "StudentSubscribed"),
        InitialState: 0,
        TransitionFn: func(state any, e dcb.Event) any { return state.(int) + 1 },
    }},
}

// 2. Build combined query for all projectors
query := dcb.NewQueryFromItems(
    dcb.NewQueryItem([]string{"CourseDefined"}, dcb.NewTags("course_id", courseID)),
    dcb.NewQueryItem([]string{"StudentSubscribed"}, dcb.NewTags("course_id", courseID)),
)

// 3. Project all states in single query
states, appendCondition, err := store.ProjectDecisionModel(ctx, query, nil, projectors)

// 4. Apply business rules using projected states
if !states["courseExists"].(bool) {
    // Append CourseDefined event
}

// 5. Append new events with optimistic locking
store.Append(ctx, []dcb.InputEvent{newEvent}, &appendCondition)
```

## Streaming Event Reading

Memory-efficient event processing using PostgreSQL's native streaming:

```go
// Read events using streaming interface
iterator, err := store.ReadStream(ctx, query, nil)
if err != nil {
    return err
}
defer iterator.Close()

// Process events row-by-row
for iterator.Next() {
    event := iterator.Event()
    // Process event without loading all events into memory
    fmt.Printf("Event: %s at position %d\n", event.Type, event.Position)
}
```

## Key Types

```go
// Event represents a persisted event in the system
type Event struct {
    ID            string `json:"id"`
    Type          string `json:"type"`
    Tags          []Tag  `json:"tags"`
    Data          []byte `json:"data"`
    Position      int64  `json:"position"`
    CausationID   string `json:"causation_id"`
    CorrelationID string `json:"correlation_id"`
}

// InputEvent represents an event to be appended to the store
type InputEvent struct {
    Type string `json:"type"`
    Tags []Tag  `json:"tags"`
    Data []byte `json:"data"`
}

// StateProjector defines how to project a state from events
type StateProjector struct {
    Query        Query                            `json:"query"`
    InitialState any                              `json:"initial_state"`
    TransitionFn func(state any, event Event) any `json:"-"`
}

// BatchProjector combines a state projector with an identifier
type BatchProjector struct {
    ID             string         `json:"id"`
    StateProjector StateProjector `json:"state_projector"`
}

// AppendCondition represents conditions for optimistic locking during append operations
type AppendCondition struct {
    FailIfEventsMatch *Query `json:"fail_if_events_match"`
    After             *int64 `json:"after"`
}
```

## Single Streamlined Query

The batch projection uses a single optimized PostgreSQL query:

```sql
SELECT id, type, tags, data, position, causation_id, correlation_id 
FROM events 
WHERE (tags @> '{"course_id": "c1"}' AND type IN ('CourseDefined'))
   OR (tags @> '{"course_id": "c1"}' AND type IN ('StudentSubscribed'))
ORDER BY position ASC
```

This query:
- Combines all projector queries with OR logic
- Uses PostgreSQL's native streaming via `pgx.Rows.Next()`
- Routes each event to appropriate projectors as it's streamed
- Maintains constant memory usage regardless of event count 