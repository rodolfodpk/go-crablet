# Overview: Dynamic Consistency Boundary (DCB) in go-crablet

go-crablet is a Go library for event sourcing, exploring concepts inspired by the [Dynamic Consistency Boundary (DCB)](https://dcb.events/) pattern. We're exploring how DCB might enable you to:

- Project multiple states and check business invariants in a single query
- Use tag-based, OR-combined queries for cross-entity consistency
- Enforce optimistic concurrency with combined append conditions

## Key Concepts

- **Batch Projection**: Project multiple states (decision model) in one database query
- **Combined Append Condition**: Use a single, OR-combined query for optimistic locking
- **Tag-based Queries**: Flexible, cross-entity queries using tags
- **Streaming**: Process events efficiently for large datasets
- **Transaction-based Ordering**: Uses PostgreSQL transaction IDs for true event ordering

## Core Interface

```go
type EventStore interface {
    // Query reads events matching the query with optional cursor
    // cursor == nil: query from beginning of stream
    // cursor != nil: query from specified cursor position (EXCLUSIVE - events after cursor, not including cursor)
    Query(ctx context.Context, query Query, cursor *Cursor) ([]Event, error)

    // QueryStream creates a channel-based stream of events matching a query with optional cursor
    // cursor == nil: stream from beginning of stream
    // cursor != nil: stream from specified cursor position (EXCLUSIVE - events after cursor, not including cursor)
    // This is optimized for large datasets and provides backpressure through channels
    // for efficient memory usage and Go-idiomatic streaming
    QueryStream(ctx context.Context, query Query, cursor *Cursor) (<-chan Event, error)

    // Append appends events to the store with optional condition
    // condition == nil: unconditional append
    // condition != nil: conditional append (optimistic locking)
    Append(ctx context.Context, events []InputEvent, condition *AppendCondition) error

    // Project projects multiple states using projectors with optional cursor
    // cursor == nil: project from beginning of stream
    // cursor != nil: project from specified cursor position (EXCLUSIVE - events after cursor, not including cursor)
    // Returns final aggregated states and append condition for optimistic locking
    Project(ctx context.Context, projectors []StateProjector, cursor *Cursor) (map[string]any, AppendCondition, error)

    // ProjectStream projects multiple states using channel-based streaming with optional cursor
    // cursor == nil: stream from beginning of stream
    // cursor != nil: stream from specified cursor position (EXCLUSIVE - events after cursor, not including cursor)
    // This is optimized for large datasets and provides backpressure through channels
    // for efficient memory usage and Go-idiomatic streaming
    ProjectStream(ctx context.Context, projectors []StateProjector, cursor *Cursor) (<-chan map[string]any, <-chan AppendCondition, error)

    // GetConfig returns the current EventStore configuration
    GetConfig() EventStoreConfig
}

type Cursor struct {
    TransactionID uint64 `json:"transaction_id"`
    Position      int64  `json:"position"`
}
```

## Transaction ID Ordering

go-crablet uses PostgreSQL's `xid8` transaction IDs for event ordering and optimistic locking:

- **True ordering**: No gaps or out-of-order events
- **Optimistic locking**: Uses transaction IDs for conflict detection
- **Cursor-based**: Combines `(transaction_id, position)` for precise positioning

## DCB Decision Model Pattern

We're exploring how a Dynamic Consistency Boundary decision model might work:

1. Define projectors for each business rule or invariant
2. Project all states in a single query
3. Build a combined append condition
4. Append new events only if all invariants still hold

## Example: Course Subscription

```go
projectors := []dcb.StateProjector{
    {
        ID: "courseExists",
        Query: dcb.NewQuery(dcb.NewTags("course_id", courseID), "CourseDefined"),
        InitialState: false,
        TransitionFn: func(state any, event dcb.Event) any {
            return true // If we see a CourseDefined event, course exists
        },
    },
    {
        ID: "numSubscriptions",
        Query: dcb.NewQuery(dcb.NewTags("course_id", courseID), "StudentEnrolled"),
        InitialState: 0,
        TransitionFn: func(state any, event dcb.Event) any {
            return state.(int) + 1
        },
    },
}
states, appendCond, _ := store.Project(ctx, projectors, nil)
if !states["courseExists"].(bool) { 
    store.Append(ctx, []dcb.InputEvent{...}, nil) 
}
if states["numSubscriptions"].(int) < 2 { 
    store.Append(ctx, []dcb.InputEvent{...}, &appendCond) 
}
```

## Transaction Isolation Levels

go-crablet uses configurable PostgreSQL transaction isolation levels:

- **Append (unconditional)**: Uses the default isolation level configured in `EventStoreConfig` (typically Read Committed)
- **Append (conditional)**: Uses the default isolation level configured in `EventStoreConfig` (typically Repeatable Read)

The isolation level can be configured when creating the EventStore via `EventStoreConfig.DefaultAppendIsolation`.

## Configuration

The EventStore can be configured with various settings:

```go
type EventStoreConfig struct {
    MaxBatchSize           int            `json:"max_batch_size"`           // Maximum events per batch
    LockTimeout            int            `json:"lock_timeout"`             // Lock timeout in milliseconds for advisory locks
    StreamBuffer           int            `json:"stream_buffer"`            // Channel buffer size for streaming operations
    DefaultAppendIsolation IsolationLevel `json:"default_append_isolation"` // Default isolation level for Append operations
    QueryTimeout           int            `json:"query_timeout"`            // Query timeout in milliseconds (defensive against hanging queries)
}
```

### Default Values
- `MaxBatchSize`: 1000 events
- `LockTimeout`: 5000ms (5 seconds)
- `StreamBuffer`: 1000 events
- `DefaultAppendIsolation`: Read Committed
- `QueryTimeout`: 15000ms (15 seconds)

## Performance Comparison Across Isolation Levels

Benchmark results from web-app load testing (30-second tests, multiple VUs):

| Metric | Append (unconditional) | Append (conditional) |
|--------|------------------------|---------------------------|
| **Throughput** | 79.2 req/s | 61.7 req/s |
| **Avg Response Time** | 24.87ms | 12.82ms |
| **p95 Response Time** | 49.16ms | 21.86ms |
| **Success Rate** | 100% | 100% |
| **VUs** | 10 | 10 |
| **Use Case** | Simple appends | Conditional appends |

### Key Performance Insights

- **Conditional append is fastest**: Conditional appends with Repeatable Read isolation actually perform better than simple appends
- **Excellent reliability**: Both isolation levels achieve 100% success rate
- **Optimized implementation**: Cursor-based optimistic locking and SQL functions are highly efficient

### When to Use Each Method

- **Append (nil condition)**: Use for simple event appends where no conditions are needed
- **Append (with condition)**: Use for conditional appends requiring optimistic locking

## Implementation Details

- **Database**: PostgreSQL with events table and append functions
- **Streaming**: Multiple approaches for different dataset sizes
- **Extensions**: Channel-based streaming for Go-idiomatic processing

See [examples](examples.md) for complete working examples including course subscriptions and money transfers, and [getting-started](getting-started.md) for setup instructions.