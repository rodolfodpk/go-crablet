# Reading Events

> **Note**: For detailed information about streaming and memory efficiency, see [Streaming & Memory Efficiency](streaming.md).

This document explains how to read events from the event store using the streaming interface inspired by the DCB pattern.

go-crablet provides a streaming interface for reading events that is both memory-efficient and inspired by the DCB pattern. Instead of loading all events into memory at once, events are streamed directly from PostgreSQL and processed one at a time.

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
    
    // Process the event
    fmt.Printf("Event: %s at position %d\n", event.Type, event.Position)
}
```

## How Pagination Works

**One call to `ReadEvents` is enough** - the iterator handles all pagination complexity internally. You don't need to manage pagination manually.

### What Happens When You Call ReadEvents

```go
// This ONE call returns an iterator, not all events
iterator, err := store.ReadEvents(ctx, query, nil)
if err != nil {
    return err
}
defer iterator.Close()

// The iterator handles pagination automatically
for {
    event, err := iterator.Next()
    if err != nil {
        return err
    }
    if event == nil {
        break // No more events
    }
    
    // Process this single event
    processEvent(event)
}
```

### What the Caller Sees vs. What Happens Internally

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

### Example: Processing All Events

```go
// This will process ALL events matching the query, regardless of how many
// The iterator handles fetching them in batches of 1000 internally
func ProcessAllEvents(ctx context.Context, store dcb.EventStore) error {
    query := dcb.NewQuery(dcb.NewTags("tenant_id", "tenant-123"))
    
    // ONE call to get iterator
    iterator, err := store.ReadEvents(ctx, query, nil)
    if err != nil {
        return err
    }
    defer iterator.Close()
    
    // Process ALL events (iterator handles pagination)
    for {
        event, err := iterator.Next()
        if err != nil {
            return err
        }
        if event == nil {
            break // All events processed
        }
        
        // Process this event
        fmt.Printf("Processing event %d: %s\n", event.Position, event.Type)
    }
    
    return nil
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

iterator, err := store.ReadEvents(ctx, query, nil)
```

## Backward Compatibility

For existing code, you can use the `NewQuery` helper function:

```go
// Old way (still works)
query := dcb.Query{
	Tags:       dcb.NewTags("account_id", "acc-123"),
	EventTypes: []string{"AccountRegistered"},
}

// New way (recommended)
query := dcb.NewQuery(
	dcb.NewTags("account_id", "acc-123"),
	"AccountRegistered",
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