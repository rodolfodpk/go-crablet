# Implementation Details

## DCB Pattern Exploration

go-crablet explores the Dynamic Consistency Boundary (DCB) pattern using:

1. **Batch Projection**: Project multiple states in a single streamlined PostgreSQL query
2. **Combined Append Conditions**: Use the same query scope for projection and optimistic locking
3. **Streaming**: Process events row-by-row using PostgreSQL's native streaming via pgx
4. **Channel-based Streaming**: Go-idiomatic streaming using channels for small-medium datasets
5. **Extension Interface Pattern**: Clean separation between core and extended functionality

## Event Store Interface Hierarchy

### Core EventStore Interface

The core interface for our DCB-inspired event management:

```go
type EventStore interface {
    // Read reads events matching a query, optionally starting from a specified sequence position
    Read(ctx context.Context, query Query, options *ReadOptions) (SequencedEvents, error)

    // ReadStream creates a stream of events matching a query (cursor-based for large datasets)
    ReadStream(ctx context.Context, query Query, options *ReadOptions) (EventIterator, error)

    // Append atomically persists one or more events, optionally with an append condition
    Append(ctx context.Context, events []InputEvent, condition *AppendCondition) (int64, error)

    // ProjectDecisionModel projects multiple states using multiple projectors and returns final states with append condition
    // This is the primary DCB API for building decision models in command handlers
    ProjectDecisionModel(ctx context.Context, projectors []BatchProjector, options *ReadOptions) (map[string]any, AppendCondition, error)
}
```

### ChannelEventStore Extension Interface

The extension interface provides Go-idiomatic channel-based streaming:

```go
type ChannelEventStore interface {
    EventStore  // Inherits all core methods

    // ReadStreamChannel creates a channel-based stream of events matching a query
    // This is optimized for small to medium datasets (< 500 events) and provides
    // a more Go-idiomatic interface using channels
    ReadStreamChannel(ctx context.Context, query Query, options *ReadOptions) (<-chan Event, error)

    // NewEventStream creates a new EventStream for the given query
    // This provides more control over the streaming process
    NewEventStream(ctx context.Context, query Query, options *ReadOptions) (*EventStream, error)

    // ProjectDecisionModelChannel projects multiple states using channel-based streaming
    // This is optimized for small to medium datasets (< 500 events) and provides
    // a more Go-idiomatic interface using channels for state projection
    ProjectDecisionModelChannel(ctx context.Context, projectors []BatchProjector, options *ReadOptions) (<-chan ProjectionResult, error)
}
```

## DCB Decision Model Pattern (Our Approach)

Our understanding of the DCB pattern for command handlers:

```go
// 1. Define projectors for the decision model
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
}

// 2. Project all states in single query (traditional cursor-based approach)
states, appendCondition, err := store.ProjectDecisionModel(ctx, projectors, nil)

// 3. Apply business rules using projected states
if !states["courseExists"].(bool) {
    // Append CourseDefined event
}

// 4. Append new events with optimistic locking
store.Append(ctx, []dcb.InputEvent{newEvent}, &appendCondition)
```

## Streaming Event Reading

### Cursor-Based Streaming (Traditional)

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

### Channel-Based Streaming (New!)

Go-idiomatic streaming using channels for small-medium datasets:

```go
// Get channel-based store
channelStore := store.(dcb.ChannelEventStore)

// Channel-based streaming
eventChan, err := channelStore.ReadStreamChannel(ctx, query, nil)
if err != nil {
    return err
}

// Process events using Go channels
for event := range eventChan {
    // Process event in real-time
    fmt.Printf("Event: %s at position %d\n", event.Type, event.Position)
}
```

### Channel-Based Projection (New!)

Real-time projection results via channels:

```go
// Channel-based projection
resultChan, err := channelStore.ProjectDecisionModelChannel(ctx, projectors, nil)
if err != nil {
    return err
}

// Process projection results in real-time
for result := range resultChan {
    if result.Error != nil {
        // Handle error
        continue
    }
    
    fmt.Printf("Projector %s processed event %s (position %d)\n", 
        result.ProjectorID, result.Event.Type, result.Position)
    
    // Access current state
    currentState := result.State
}
```

## Performance Characteristics

| Method | Best For | Memory Usage | Real-time Feedback | Scalability |
|--------|----------|--------------|-------------------|-------------|
| `Read()` | < 100 events | High | ❌ No | Limited |
| `ReadStream()` | > 1000 events | Low | ❌ No | Excellent |
| `ReadStreamChannel()` | 100-500 events | Moderate | ✅ Yes | Good |
| `ProjectDecisionModel()` | > 1000 events | Low | ❌ No | Excellent |
| `ProjectDecisionModelChannel()` | 100-500 events | Moderate | ✅ Yes | Good |

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

// ProjectionResult represents a single projection result from channel-based projection
type ProjectionResult struct {
    ProjectorID string      // Which projector produced this result
    State       interface{} // Current state after projection
    Event       Event       // Event that was processed
    Position    int64       // Sequence position
    Error       error       // Any error that occurred
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
- Uses PostgreSQL's native streaming via `pgx.Rows.Next()` (cursor-based)
- Uses channel-based streaming for small-medium datasets
- Routes each event to appropriate projectors as it's streamed
- Maintains constant memory usage regardless of event count 