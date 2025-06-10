# Reading Events

go-crablet provides a streaming interface for reading events that is both memory-efficient and DCB-compliant. Instead of loading all events into memory at once, events are streamed directly from PostgreSQL and processed one at a time.

## EventIterator Interface

The core of the streaming approach is the `EventIterator` interface:

```go
type EventIterator interface {
    // Next returns the next event in the stream
    // Returns nil when no more events are available
    Next() (*Event, error)
    
    // Close closes the iterator and releases resources
    Close() error
    
    // Position returns the position of the last event read
    Position() int64
}
```

## Basic Event Reading

```go
// Create a query for account events
query := dcb.NewLegacyQuery(
    dcb.NewTags("account_id", "acc-123"),
    []string{"AccountCreated", "AccountUpdated"},
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
    
    // Process the event
    fmt.Printf("Event: %s at position %d\n", event.Type, event.Position)
}
```

## ReadOptions Configuration

You can configure how events are read using `ReadOptions`:

```go
options := &dcb.ReadOptions{
    FromPosition: 100,    // Start reading from position 100
    Limit:        50,     // Read at most 50 events
    OrderBy:      "desc", // Read in descending order
}

iterator, err := store.ReadEvents(ctx, query, options)
```

### ReadOptions Fields

- **FromPosition**: Start reading from this position (inclusive). Default: 0
- **Limit**: Maximum number of events to return. Default: 0 (no limit)
- **OrderBy**: Ordering direction. Options: "asc" (default) or "desc"

## Complex Queries

The new Query structure supports complex queries with multiple items combined with OR logic:

```go
// Query for events that are either:
// 1. Account events for account "acc-123"
// 2. Transaction events for account "acc-123"
// 3. Any events tagged with "user_id" = "user-456"
query := dcb.NewQueryFromItems(
    dcb.NewQueryItem(
        []string{"AccountCreated", "AccountUpdated"},
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

iterator, err := store.ReadEvents(ctx, query, nil)
```

## Backward Compatibility

For existing code, you can use the `NewLegacyQuery` helper function:

```go
// Old way (still works)
query := dcb.Query{
    Tags:       dcb.NewTags("account_id", "acc-123"),
    EventTypes: []string{"AccountCreated"},
}

// New way (recommended)
query := dcb.NewLegacyQuery(
    dcb.NewTags("account_id", "acc-123"),
    []string{"AccountCreated"},
)
```

## Error Handling

The streaming interface provides comprehensive error handling:

```go
iterator, err := store.ReadEvents(ctx, query, options)
if err != nil {
    // Handle validation errors, database errors, etc.
    return err
}
defer iterator.Close()

for {
    event, err := iterator.Next()
    if err != nil {
        // Handle scanning errors, database errors, etc.
        return err
    }
    if event == nil {
        break
    }
    
    // Process event
}
```

## Memory Efficiency

The streaming approach is particularly beneficial for large event streams:

```go
// This approach processes events one at a time
// No matter how many events match the query
iterator, err := store.ReadEvents(ctx, query, nil)
if err != nil {
    return err
}
defer iterator.Close()

var count int
for {
    event, err := iterator.Next()
    if err != nil {
        return err
    }
    if event == nil {
        break
    }
    
    count++
    // Process event without loading all events into memory
}

fmt.Printf("Processed %d events\n", count)
```

## Integration with State Projection

The streaming approach is consistent with the existing state projection functionality:

```go
// Both use the same streaming approach internally
projector := dcb.StateProjector{
    Query:        query,
    InitialState: &AccountState{},
    TransitionFn: func(state any, event dcb.Event) any {
        // Process event and update state
        return state
    },
}

position, state, err := store.ProjectState(ctx, projector)
```

## Performance Considerations

- **Use limits** when you only need a subset of events
- **Use FromPosition** to resume reading from where you left off
- **Close iterators** to release database resources
- **Handle errors** appropriately to avoid resource leaks
- **Consider ordering** - "desc" can be faster for recent events

## DCB Compliance

The streaming interface fully complies with the DCB specification:

- ✅ **Streaming events**: Events are returned as a stream, not a complete list
- ✅ **Query-based filtering**: Support for complex queries with multiple items
- ✅ **Position tracking**: Each event includes its sequence position
- ✅ **Efficient processing**: Memory-efficient for large event streams 