# Overview: Exploring Dynamic Consistency Boundary (DCB) Concepts in go-crablet

go-crablet is a Go library for event sourcing, exploring concepts inspired by the [Dynamic Consistency Boundary (DCB)](https://dcb.events/) pattern. We're learning how DCB enables you to:

- Project multiple states and check business invariants in a single query
- Use tag-based, OR-combined queries for cross-entity consistency
- Enforce optimistic concurrency with combined append conditions
- Stream events efficiently for large datasets
- Use channel-based streaming for Go-idiomatic event processing

## Key Concepts We're Exploring

- **Batch Projection**: Project multiple states (decision model) in one database query
- **Combined Append Condition**: Use a single, OR-combined query for optimistic locking
- **Streaming**: Process events row-by-row, suitable for millions of events
- **Channel-based Streaming**: Go-idiomatic streaming using channels for small-medium datasets
- **Tag-based Queries**: Flexible, cross-entity queries using tags
- **Extension Interface Pattern**: Clean separation between core and extended functionality

## Interface Hierarchy

### Core EventStore Interface
```go
type EventStore interface {
    Read(ctx context.Context, query Query, options *ReadOptions) (SequencedEvents, error)
    ReadStream(ctx context.Context, query Query, options *ReadOptions) (EventIterator, error)
    Append(ctx context.Context, events []InputEvent, condition *AppendCondition) (int64, error)
    ProjectDecisionModel(ctx context.Context, projectors []BatchProjector, options *ReadOptions) (map[string]any, AppendCondition, error)
}
```

### ChannelEventStore Extension Interface
```go
type ChannelEventStore interface {
    EventStore  // Inherits all core methods
    
    ReadStreamChannel(ctx context.Context, query Query, options *ReadOptions) (<-chan Event, error)
    NewEventStream(ctx context.Context, query Query, options *ReadOptions) (*EventStream, error)
    ProjectDecisionModelChannel(ctx context.Context, projectors []BatchProjector, options *ReadOptions) (<-chan ProjectionResult, error)
}
```

## DCB Decision Model Pattern (Our Understanding)

A DCB decision model is built by:
1. Defining projectors for each business rule or invariant
2. Projecting all states in a single query
3. Building a combined append condition
4. Appending new events only if all invariants still hold

## Example: Course Subscription

### Traditional Cursor-Based Approach
```go
projectors := []dcb.BatchProjector{
    {ID: "courseExists", StateProjector: dcb.StateProjector{...}},
    {ID: "numSubscriptions", StateProjector: dcb.StateProjector{...}},
}
states, appendCond, _ := store.ProjectDecisionModel(ctx, projectors, nil)
if !states["courseExists"].(bool) { /* append CourseDefined */ }
if states["numSubscriptions"].(int) < 2 { /* append StudentSubscribed */ }
```

### Channel-Based Approach (New!)
```go
// Get channel-based store
channelStore := store.(dcb.ChannelEventStore)

// Immediate projection with feedback
resultChan, err := channelStore.ProjectDecisionModelChannel(ctx, projectors, nil)

// Process results with immediate feedback
for result := range resultChan {
    fmt.Printf("Projector %s: %v\n", result.ProjectorID, result.State)
}
```

## Streaming & Memory Efficiency

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
- **ReadStreamChannel**: Channel-based streaming for Go-idiomatic processing
- **ProjectDecisionModelChannel**: Immediate projection results via channels

## Why Explore DCB?
- **Single-query consistency**: All invariants checked atomically
- **No aggregates required**: Consistency boundaries are defined by your queries
- **Efficient**: One database round trip for all business rules
- **Go-idiomatic**: Channel-based streaming for immediate processing
- **Flexible**: Choose the right streaming approach for your dataset size

See the [README](../README.md) and [examples](examples.md) for more.