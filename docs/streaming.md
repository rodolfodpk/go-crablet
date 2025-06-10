# Streaming & Memory Efficiency

This document explains how the event store handles large datasets efficiently through streaming and memory-optimized approaches.

## Overview

The event store provides two primary methods for reading events, both designed for **memory-efficient processing** of large event datasets:

- **`ReadEvents`**: Application-level streaming with batching
- **`ProjectState`**: Database-level streaming for state reconstruction

## ReadEvents - Application-Level Streaming

`ReadEvents` provides streaming events with application-level batching inspired by the DCB pattern.

### How It Works

```go
// Create a query
query := dcb.NewQuery(
    dcb.NewTags("account_id", "acc-123"),
    "AccountRegistered", "AccountDetailsChanged",
)

// Get iterator (one call is enough)
iterator, err := store.ReadEvents(ctx, query, nil)
if err != nil {
    return err
}
defer iterator.Close()

// Process events one by one (iterator handles pagination)
for {
    event, err := iterator.Next()
    if err != nil {
        return err
    }
    if event == nil {
        break // No more events
    }
    
    // Process single event
    processEvent(event)
}
```

### Pagination: What the Caller Sees vs. What Happens Internally

#### **Caller's Perspective** (Simple):
```go
// 1. Get iterator (one call)
iterator, err := store.ReadEvents(ctx, query, nil)

// 2. Process events one by one
for {
    event, err := iterator.Next()
    if event == nil {
        break // Done!
    }
    // Process event
}
```

#### **What Happens Internally** (Automatic):
1. **First call to `Next()`**: Fetches first batch of 1000 events
2. **Subsequent calls**: Returns events from current batch
3. **When batch exhausted**: Automatically fetches next batch
4. **When no more events**: Returns `nil` to signal completion

### Key Points

✅ **One call to `ReadEvents`** is enough to get started  
✅ **Iterator handles pagination** automatically  
✅ **Caller processes events one by one**  
✅ **Memory efficient** - only batch size events in memory  
✅ **No manual pagination management** required  

### Memory Management

The iterator maintains only the current batch in memory:

```go
type eventIterator struct {
    store         *eventStore
    query         Query
    options       *ReadOptions
    lastPosition  int64
    batchSize     int           // Configurable batch size (default: 1000)
    currentBatch  []Event       // Only current batch in memory
    currentIndex  int
    closed        bool
    hasMore       bool
    initialized   bool
    ctx           context.Context
    totalFetched  int
}
```

### Memory Efficiency Features

1. **Batch Processing**: Fetches events in configurable batches (default: 1000)
2. **Keyset Pagination**: Uses `position > lastPosition` for efficient queries
3. **Lazy Loading**: Only fetches next batch when current batch is exhausted
4. **Constant Memory**: Memory usage remains constant regardless of total events
5. **Configurable Batch Size**: Can be adjusted via `ReadOptions.BatchSize`

### Use Cases

- **Event Processing**: Custom event processing logic
- **Event Replay**: Replaying events for debugging or migration
- **Event Export**: Exporting events to external systems
- **Analytics**: Processing events for analytics or reporting

## ProjectState - Database-Level Streaming

`ProjectState` uses PostgreSQL's native streaming capabilities for real-time state reconstruction.

### How It Works

```go
// Define a projector
projector := dcb.StateProjector{
    Query: dcb.NewQuery(
        dcb.NewTags("account_id", "acc-123"),
        "AccountRegistered", "AccountDetailsChanged",
    ),
    InitialState: &AccountState{},
    TransitionFn: func(state any, event dcb.Event) any {
        account := state.(*AccountState)
        switch event.Type {
        case "AccountRegistered":
            // Update account state
        case "AccountDetailsChanged":
            // Update account details
        }
        return account
    },
}

// Stream events and build state
state, position, err := store.ProjectState(ctx, projector)
if err != nil {
    return err
}
```

### Memory Efficiency Features

1. **Native PostgreSQL Streaming**: Uses `rows.Next()` for row-by-row processing
2. **Zero Accumulation**: No intermediate storage of events
3. **Incremental Updates**: State is updated as each event is processed
4. **Constant Memory**: Memory usage is O(1) regardless of event count

### Implementation Details

```go
// Process events with proper error handling
for rows.Next() {
    var row rowEvent
    if err := rows.Scan(&row.ID, &row.Type, &row.Tags, &row.Data, &row.Position, &row.CausationID, &row.CorrelationID); err != nil {
        return err
    }
    
    // Convert row to Event
    event := convertRowToEvent(row)
    
    // Apply projector (processes ONE event at a time)
    state = projector.TransitionFn(state, event)
    position = row.Position
}
```

### Use Cases

- **State Reconstruction**: Building current state from event history
- **Real-time Projections**: Updating read models in real-time
- **Decision Models**: Building decision models for business logic
- **Audit Trails**: Reconstructing state at any point in time

## Memory Usage Comparison

| Method | Memory Pattern | Memory Complexity | Use Case |
|--------|---------------|------------------|----------|
| `ReadEvents` | 1000 events per batch | O(1) | Event iteration, custom processing |
| `ProjectState` | 1 event at a time | O(1) | State reconstruction, projections |

## Performance Characteristics

### ReadEvents
- **Initial Response**: Fast (first batch loaded quickly)
- **Memory Usage**: Constant (1000 events max in memory)
- **Network Efficiency**: Batched queries reduce round trips
- **Scalability**: Handles millions of events efficiently

### ProjectState
- **Initial Response**: Fast (streaming starts immediately)
- **Memory Usage**: Minimal (one event at a time)
- **Network Efficiency**: Native PostgreSQL streaming
- **Scalability**: Excellent for large event streams

## Best Practices

### When to Use ReadEvents
- Custom event processing logic
- Event replay or migration scenarios
- Exporting events to external systems
- Analytics or reporting workflows
- When you need fine-grained control over event processing

### When to Use ProjectState
- State reconstruction for read models
- Real-time projections
- Decision model building
- Audit trail reconstruction
- When you need to build state incrementally

### Memory Management
- Always call `iterator.Close()` when using `ReadEvents`
- Use `defer iterator.Close()` for automatic cleanup
- Consider using `ProjectStateUpTo` for partial state reconstruction
- Monitor memory usage in production environments
- Configure batch size based on your use case:
  - **Small batch size (100-500)**: For memory-constrained environments
  - **Default batch size (1000)**: Good balance for most use cases
  - **Large batch size (2000-5000)**: For high-throughput scenarios with sufficient memory

## Example: Processing Large Event Streams

```go
// Efficiently process millions of events
func ProcessLargeEventStream(ctx context.Context, store dcb.EventStore) error {
    query := dcb.NewQuery(dcb.NewTags("tenant_id", "tenant-123"))
    
    // Configure batch size for memory-constrained environment
    options := &dcb.ReadOptions{
        BatchSize: 500, // Smaller batches for lower memory usage
    }
    
    iterator, err := store.ReadEvents(ctx, query, options)
    if err != nil {
        return err
    }
    defer iterator.Close()
    
    processed := 0
    for {
        event, err := iterator.Next()
        if err != nil {
            return err
        }
        if event == nil {
            break
        }
        
        // Process single event
        if err := processEvent(event); err != nil {
            return err
        }
        
        processed++
        if processed%10000 == 0 {
            log.Printf("Processed %d events", processed)
        }
    }
    
    log.Printf("Total processed: %d events", processed)
    return nil
}
```

## Example: High-Throughput Processing

```go
// High-throughput processing with larger batches
func ProcessHighThroughputStream(ctx context.Context, store dcb.EventStore) error {
    query := dcb.NewQuery(dcb.NewTags("tenant_id", "tenant-123"))
    
    // Configure larger batch size for high-throughput scenarios
    options := &dcb.ReadOptions{
        BatchSize: 2000, // Larger batches for better throughput
    }
    
    iterator, err := store.ReadEvents(ctx, query, options)
    if err != nil {
        return err
    }
    defer iterator.Close()
    
    // Process events in larger batches for better performance
    // ... processing logic ...
    
    return nil
}
```

This approach ensures **constant memory usage** even when processing millions of events, making it suitable for production environments with large event stores. 