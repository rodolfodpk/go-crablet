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

// Process events with immediate delivery
for event := range eventChan {
    fmt.Printf("Event: %s at position %d\n", event.Type, event.Position)
}
```

### Channel-Based Projection (New!)

Immediate projection results via channels:

```go
// Channel-based projection
resultChan, err := channelStore.ProjectDecisionModelChannel(ctx, projectors, nil)
if err != nil {
    return err
}

// Process projection results with immediate feedback
for result := range resultChan {
    fmt.Printf("Projector %s: %v\n", result.ProjectorID, result.State)
}
```

## Performance Characteristics

| Method | Best For | Memory Usage | Immediate Feedback | Scalability |
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

// Query represents a composite query with multiple conditions combined with OR logic
type Query struct {
    Items []QueryItem `json:"items"`
}

// QueryItem represents a single atomic query condition
type QueryItem struct {
    EventTypes []string `json:"event_types"`
    Tags       []Tag    `json:"tags"`
}

// ReadOptions provides options for reading events (position, limits, batch size)
type ReadOptions struct {
    FromPosition *int64 `json:"from_position"`
    Limit        *int   `json:"limit"`
    BatchSize    *int   `json:"batch_size"`
}
```

## Query Computation

Understanding how the EventStore methods compute the queries they need to fetch events is crucial for the DCB pattern.

### Read / ReadStream Methods

**How the query is computed:**
- The **caller** (your application code) is responsible for building the `Query` object
- This `Query` is typically constructed using helper functions like `dcb.NewQuery`, `dcb.NewQueryFromItems`, or the new `QItem`/`QItemKV` helpers
- The `Query` contains all the filtering logic: event types and tags (OR-combined)
- The `ReadOptions` only controls things like position, limit, and batch size—not the query itself

**Example:**
```go
// Using the new QItemKV helper for concise syntax
query := dcb.NewQueryFromItems(
    dcb.QItemKV("CourseDefined", "course_id", "c1"),
    dcb.QItemKV("StudentRegistered", "student_id", "s1"),
    dcb.QItemKV("StudentSubscribed", "course_id", "c1", "student_id", "s1"),
)

// Or using the traditional approach
query := dcb.NewQuery(dcb.NewTags("course_id", "c1"), "CourseDefined")

events, err := store.Read(ctx, query, &dcb.ReadOptions{Limit: &limit})
```

### ProjectDecisionModel / ProjectDecisionModelChannel

**How the query is computed:**
- Each `BatchProjector` contains a `StateProjector`, which has its own `Query`
- The method **combines all the queries** from each projector into a single `Query` object
- This is done by collecting all `QueryItem`s from all projectors and OR-combining them into one `Query`
- This combined query is then used to fetch all relevant events from the event store

**Example (pseudo-code of internal implementation):**
```go
func (es *eventStore) combineProjectorQueries(projectors []BatchProjector) Query {
    var combinedItems []QueryItem
    for _, bp := range projectors {
        for _, item := range bp.StateProjector.Query.Items {
            combinedItems = append(combinedItems, item)
        }
    }
    return Query{Items: combinedItems}
}
```

**Why this matters for DCB:**
- The combined query ensures that **all relevant events** for all projectors are fetched in a single database operation
- This is the foundation of the DCB pattern: using the same query scope for both projection and optimistic locking
- The `AppendCondition` returned by `ProjectDecisionModel` uses this same combined query to ensure consistency

### Query Computation Summary

| Method                        | How Query is Computed                | DCB Alignment |
|-------------------------------|--------------------------------------|---------------|
| `Read` / `ReadStream`         | Passed directly by caller            | Manual query building |
| `ProjectDecisionModel`        | OR-combination of all projectors' queries | Automatic query combination |

This approach is fully aligned with the [DCB specification](https://dcb.events/specification/#concepts), where queries are always explicit and based on event type/tags, and projectors define their own queries that get combined for batch projection.

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

## ReadOptions Usage

**Important:** `ReadOptions` is used for controlling **how** events are read, not **which** events are read. The query logic (event types and tags) is handled by the `Query` parameter, not by `ReadOptions`.

### Common Usage Patterns

**Most common case (no options):**
```go
// Use nil for default behavior
states, appendCondition, err := store.ProjectDecisionModel(ctx, projectors, nil)
```

**With position limits:**
```go
from := int64(1000)
readOptions := &dcb.ReadOptions{
    FromPosition: &from,  // Start reading from position 1000
}
states, appendCondition, err := store.ProjectDecisionModel(ctx, projectors, readOptions)
```

**With result limits:**
```go
limit := 1000
readOptions := &dcb.ReadOptions{
    Limit: &limit,  // Process at most 1000 events
}
states, appendCondition, err := store.ProjectDecisionModel(ctx, projectors, readOptions)
```

**With streaming batch size:**
```go
batch := 100
readOptions := &dcb.ReadOptions{
    BatchSize: &batch,  // Process events in batches of 100
}
states, appendCondition, err := store.ProjectDecisionModel(ctx, projectors, readOptions)
```

### When to Use ReadOptions

- **`FromPosition`**: Resume processing from a specific event position
- **`Limit`**: Prevent processing too many events (safety mechanism)
- **`BatchSize`**: Control memory usage in streaming operations

**Note:** For most DCB use cases, using `nil` for `ReadOptions` is sufficient and recommended. 