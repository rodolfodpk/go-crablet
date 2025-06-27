# Overview: Exploring Dynamic Consistency Boundary (DCB) Concepts in go-crablet

go-crablet is a Go library for event sourcing, exploring and learning about concepts inspired by the [Dynamic Consistency Boundary (DCB)](https://dcb.events/) pattern. We're learning how DCB enables you to:

- Project multiple states and check business invariants in a single query
- Use tag-based, OR-combined queries for cross-entity consistency
- Enforce optimistic concurrency with combined append conditions

Our implementation also provides:
- Stream events efficiently for large datasets
- Channel-based streaming for Go-idiomatic processing

## Event Store Structure

Our implementation stores events in PostgreSQL with this structure:
```sql
CREATE TABLE events (
    type VARCHAR(64) NOT NULL,
    tags TEXT[] NOT NULL,
    data JSON NOT NULL,
    position BIGSERIAL NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
```

See the complete table definition in [schema.sql](../docker-entrypoint-initdb.d/schema.sql#L8-L13).

## Append Procedure

We use PostgreSQL functions for atomic append operations:
- [`append_events_with_condition()`](../docker-entrypoint-initdb.d/schema.sql#L75-L95): Checks conditions and appends events atomically
- [`check_append_condition()`](../docker-entrypoint-initdb.d/schema.sql#L25-L65): Validates that no conflicting events exist
- [`append_events_batch()`](../docker-entrypoint-initdb.d/schema.sql#L67-L73): Efficiently inserts multiple events using UNNEST

The append procedure ensures optimistic concurrency by checking that no events matching the append condition exist before inserting new events.

## Key Concepts We're Learning About

- **Batch Projection**: Project multiple states (decision model) in one database query
- **Combined Append Condition**: Use a single, OR-combined query for optimistic locking
- **Streaming**: Process events row-by-row, suitable for millions of events
- **Tag-based Queries**: Flexible, cross-entity queries using tags

## Interface Hierarchy

### Core EventStore Interface (DCB-inspired)
```go
type EventStore interface {
    Read(ctx context.Context, query Query, options *ReadOptions) (SequencedEvents, error)
    Append(ctx context.Context, events []InputEvent, condition *AppendCondition) (int64, error)
}
```

### ChannelEventStore Extension Interface (Our Implementation)
```go
type ChannelEventStore interface {
    EventStore  // Inherits all core methods
    
    ReadStreamChannel(ctx context.Context, query Query) (<-chan Event, error)
    ProjectDecisionModel(ctx context.Context, projectors []BatchProjector) (map[string]any, AppendCondition, error)
    ProjectDecisionModelChannel(ctx context.Context, projectors []BatchProjector) (<-chan ProjectionResult, error)
}
```

## Our Understanding of DCB Decision Model Pattern

We're exploring how a Dynamic Consistency Boundary decision model might be built by:
1. Defining projectors for each business rule or invariant
2. Projecting all states in a single query
3. Building a combined append condition
4. Appending new events only if all invariants still hold

## Example: Course Subscription

### DCB Decision Model Approach
```go
projectors := []dcb.BatchProjector{
    {ID: "courseExists", StateProjector: dcb.StateProjector{...}},
    {ID: "numSubscriptions", StateProjector: dcb.StateProjector{...}},
}
states, appendCond, _ := store.ProjectDecisionModel(ctx, projectors)
if !states["courseExists"].(bool) { /* append CourseDefined */ }
if states["numSubscriptions"].(int) < 2 { /* append StudentSubscribed */ }
```

### Our Channel-Based Extension
```go
// Get channel-based store (our implementation extension)
channelStore := store.(dcb.ChannelEventStore)

// Immediate projection with feedback
resultChan, err := channelStore.ProjectDecisionModelChannel(ctx, projectors)

// Process results with immediate feedback
for result := range resultChan {
    fmt.Printf("Projector %s: %v\n", result.ProjectorID, result.State)
}
```

## Streaming Options

### Performance Characteristics
| Approach | Best For | Memory Usage | Immediate Feedback | Scalability |
|----------|----------|--------------|-------------------|-------------|
| **Read()** | < 100 events | High | ❌ No | Limited |
| **ReadStream()** | > 1000 events | Low | ❌ No | Excellent |
| **ReadStreamChannel()** | 100-500 events | Moderate | ✅ Yes | Good |
| **ProjectDecisionModel()** | > 1000 events | Low | ❌ No | Excellent |
| **ProjectDecisionModelChannel()** | 100-500 events | Moderate | ✅ Yes | Good |

### Streaming Options
- **ProjectDecisionModel**: Projects all states in one query, streams events row-by-row (cursor-based)
- **ReadStream**: Streams events for custom processing (cursor-based)
- **ReadStreamChannel**: Channel-based streaming for Go-idiomatic processing (our extension)
- **ProjectDecisionModelChannel**: Immediate projection results via channels (our extension)

## Why Explore Dynamic Consistency Boundaries?
- **Single-query consistency**: All invariants checked atomically
- **No aggregates required**: Consistency boundaries are defined by your queries
- **Efficient**: One database round trip for all business rules
- **Flexible**: Choose the right streaming approach for your dataset size

See the [README](../README.md) and [examples](examples.md) for more.

The `Query` and `QueryItem` types are opaque. They can only be constructed using the provided helper functions (e.g., `NewQuery`, `NewQueryItem`, `NewQueryFromItems`, etc.). Direct struct access or field manipulation is not possible. This enforces DCB compliance and improves type safety for all consumers of the library.