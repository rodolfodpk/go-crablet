# Channel-Based Streaming in go-crablet

go-crablet provides **channel-based streaming** as an alternative to the traditional cursor-based approach, offering a more Go-idiomatic interface for event processing.

## Overview

Channel-based streaming is optimized for **small to medium datasets** (< 500 events) and provides:

- **Immediate event delivery** - events are processed and delivered as soon as they're read from the result set
- **Go-idiomatic interface** - uses channels for streaming
- **Backward compatibility** - existing code continues to work
- **Extension interface pattern** - clean separation of concerns

## Interface Hierarchy

### Core EventStore Interface
```go
type EventStore interface {
    Read(ctx context.Context, query Query, options *ReadOptions) (SequencedEvents, error)
    Append(ctx context.Context, events []InputEvent, condition *AppendCondition) (int64, error)
    ProjectDecisionModel(ctx context.Context, projectors []BatchProjector) (map[string]any, AppendCondition, error)
}
```

### ChannelEventStore Extension Interface
```go
type ChannelEventStore interface {
    EventStore  // Inherits all core methods
    
    ReadStreamChannel(ctx context.Context, query Query) (<-chan Event, error)
    ProjectDecisionModelChannel(ctx context.Context, projectors []BatchProjector) (<-chan ProjectionResult, error)
}
```

## Usage Patterns

### Basic Channel-Based Streaming

```go
// Create event store
store, _ := dcb.NewEventStore(ctx, pool)

// Get channel-based store
channelStore := store.(dcb.ChannelEventStore)

// Create query
query := dcb.NewQuerySimple(dcb.NewTags("user_id", "user-1"), "UserCreated", "UserUpdated")

// Channel-based streaming
eventChan, err := channelStore.ReadStreamChannel(ctx, query)
if err != nil {
    log.Fatal(err)
}

// Process events with immediate delivery
for event := range eventChan {
    fmt.Printf("Processing event: %s at position %d\n", event.Type, event.Position)
    
    // Process event based on type
    switch event.Type {
    case "UserCreated":
        fmt.Println("User was created")
    case "UserUpdated":
        fmt.Println("User was updated")
    }
}
```

### Channel-Based Projection

```go
// Define projectors
projectors := []dcb.BatchProjector{
    {ID: "userCount", StateProjector: dcb.StateProjector{
        Query: dcb.NewQuerySimple(dcb.NewTags(), "UserCreated"),
        InitialState: 0,
        TransitionFn: func(state any, event dcb.Event) any {
            return state.(int) + 1
        },
    }},
    {ID: "activeUsers", StateProjector: dcb.StateProjector{
        Query: dcb.NewQuerySimple(dcb.NewTags(), "UserCreated", "UserDeactivated"),
        InitialState: 0,
        TransitionFn: func(state any, event dcb.Event) any {
            count := state.(int)
            switch event.Type {
            case "UserCreated":
                return count + 1
            case "UserDeactivated":
                return count - 1
            }
            return count
        },
    }},
}

// Channel-based projection
resultChan, err := channelStore.ProjectDecisionModelChannel(ctx, projectors)
if err != nil {
    log.Fatal(err)
}

// Process projection results with immediate feedback
for result := range resultChan {
    fmt.Printf("Projector %s: %v\n", result.ProjectorID, result.State)
}

// Use final states
fmt.Printf("Total users: %d\n", finalStates["userCount"])
fmt.Printf("Active users: %d\n", finalStates["activeUsers"])
```

### Controlled Streaming with EventStream

```go
// Create event stream with more control
stream, err := channelStore.NewEventStream(ctx, query, nil)
if err != nil {
    log.Fatal(err)
}
defer stream.Close()

// Process events with explicit control
eventCount := 0
for event := range stream.Events() {
    fmt.Printf("Event %d: %s at position %d\n", 
        eventCount+1, event.Type, event.Position)
    eventCount++
    
    // Can break early if needed
    if eventCount >= 10 {
        break
    }
}

// Check for stream errors
if stream.err != nil {
    fmt.Printf("Stream error: %v\n", stream.err)
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

## When to Use Channel-Based Streaming

### ✅ Use Channel-Based When:
- **Small to medium datasets** (< 500 events)
- **Immediate feedback** is needed during processing
- **Go-idiomatic code** is preferred
- **Interactive processing** is required
- **Debugging and monitoring** projection progress

### ❌ Use Cursor-Based When:
- **Large datasets** (> 1000 events)
- **Memory efficiency** is critical
- **Batch processing** is sufficient
- **Maximum scalability** is needed

## Processing Differences

### Cursor-Based (Batch Processing)
- **Batch-oriented**: Fetches events in chunks (default 1000 events per batch)
- **Blocking**: Waits for entire batch to be processed before moving to next
- **Memory-efficient**: Optimized for large datasets with server-side cursors
- **Processing pattern**: `FETCH 1000 → process all 1000 → FETCH 1000 → repeat`

### Channel-Based (Streaming Processing)
- **Event-by-event**: Processes and delivers each event immediately as it's read
- **Non-blocking**: Events flow through channels as they become available
- **Immediate delivery**: Consumers receive events as soon as they're processed
- **Processing pattern**: `row.Next() → process 1 event → send to channel → row.Next() → repeat`

**Note**: Both approaches read from the same database query results. The difference is in how quickly events are delivered to the consumer, not when the events actually occurred in the system.

## Type Definitions

### ProjectionResult
```go
type ProjectionResult struct {
    ProjectorID string      // Which projector produced this result
    State       interface{} // Current state after projection
    Event       Event       // Event that was processed
    Position    int64       // Sequence position
    Error       error       // Any error that occurred
}
```

### EventStream
```go
type EventStream struct {
    rows pgx.Rows
    ch   chan Event
    err  error
    ctx  context.Context
}

func (s *EventStream) Events() <-chan Event
func (s *EventStream) Close() error
```

## Error Handling

Channel-based streaming provides comprehensive error handling:

```go
// Handle stream errors
for result := range resultChan {
    if result.Error != nil {
        // Handle individual projection errors
        fmt.Printf("Projection error: %v\n", result.Error)
        continue
    }
    
    // Process successful result
    fmt.Printf("Success: %s processed %s\n", 
        result.ProjectorID, result.Event.Type)
}

// Check for stream completion errors
if stream.err != nil {
    fmt.Printf("Stream completion error: %v\n", stream.err)
}
```

## Examples

See the following examples for complete implementations:

- **`internal/examples/channel_streaming/`** - Basic channel-based event streaming
- **`internal/examples/channel_projection/`** - Channel-based state projection
- **`internal/examples/extension_interface/`** - Extension interface pattern demonstration

## Migration from Traditional Approach

### Before (Cursor-Based)
```go
// Traditional approach
states, appendCondition, err := store.ProjectDecisionModel(ctx, projectors)
if err != nil {
    return err
}

// Use final states
courseExists := states["courseExists"].(bool)
```

### After (Channel-Based)
```go
// Channel-based approach
channelStore := store.(dcb.ChannelEventStore)
resultChan, err := channelStore.ProjectDecisionModelChannel(ctx, projectors)
if err != nil {
    return err
}

// Process results with immediate feedback
for result := range resultChan {
    fmt.Printf("Projector %s: %v\n", result.ProjectorID, result.State)
}

// Use final states
courseExists := finalStates["courseExists"].(bool)
```

The channel-based approach provides the same functionality with immediate feedback and a more Go-idiomatic interface. 