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
    // Read reads events matching the query (no options)
    Read(ctx context.Context, query Query) ([]Event, error)

    // ReadChannel creates a channel-based stream of events matching a query
    // This replaces ReadWithOptions functionality - the caller manages complexity
    // like limits and cursors through channel consumption patterns
    // This is optimized for small to medium datasets (< 500 events) and provides
    // a more Go-idiomatic interface using channels
    ReadChannel(ctx context.Context, query Query) (<-chan Event, error)

    // Append appends events to the store (always succeeds if no validation errors)
    // Uses the default isolation level configured in EventStoreConfig
    Append(ctx context.Context, events []InputEvent) error

    // AppendIf appends events to the store only if the condition is met
    // Uses the default isolation level configured in EventStoreConfig
    AppendIf(ctx context.Context, events []InputEvent, condition AppendCondition) error

    // ProjectDecisionModel projects multiple states using projectors and returns final states and append condition
    // This is a go-crablet feature for building decision models in command handlers
    ProjectDecisionModel(ctx context.Context, projectors []StateProjector) (map[string]any, AppendCondition, error)

    // ProjectDecisionModelChannel projects multiple states using channel-based streaming
    // This is optimized for small to medium datasets (< 500 events) and provides
    // a more Go-idiomatic interface using channels for state projection
    // Returns final aggregated states (same as batch version) via streaming
    ProjectDecisionModelChannel(ctx context.Context, projectors []StateProjector) (<-chan map[string]any, <-chan AppendCondition, error)

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
    {ID: "courseExists", StateProjector: dcb.StateProjector{...}},
    {ID: "numSubscriptions", StateProjector: dcb.StateProjector{...}},
}
states, appendCond, _ := store.ProjectDecisionModel(ctx, projectors)
if !states["courseExists"].(bool) { 
    store.Append(ctx, []dcb.InputEvent{...}) 
}
if states["numSubscriptions"].(int) < 2 { 
    store.AppendIf(ctx, []dcb.InputEvent{...}, appendCond) 
}
```

## Transaction Isolation Levels

go-crablet uses configurable PostgreSQL transaction isolation levels:

- **Append**: Uses the default isolation level configured in `EventStoreConfig` (typically Read Committed)
- **AppendIf**: Uses the default isolation level configured in `EventStoreConfig` (typically Repeatable Read)

The isolation level can be configured when creating the EventStore via `EventStoreConfig.DefaultAppendIsolation`.

## Performance Comparison Across Isolation Levels

Benchmark results from web-app load testing (30-second tests, multiple VUs):

| Metric | Append (Read Committed) | AppendIf (Repeatable Read) |
|--------|------------------------|---------------------------|
| **Throughput** | 79.2 req/s | 61.7 req/s |
| **Avg Response Time** | 24.87ms | 12.82ms |
| **p95 Response Time** | 49.16ms | 21.86ms |
| **Success Rate** | 100% | 100% |
| **VUs** | 10 | 10 |
| **Use Case** | Simple appends | Conditional appends |

### Key Performance Insights

- **AppendIf is fastest**: Conditional appends with Repeatable Read isolation actually perform better than simple appends
- **Excellent reliability**: Both isolation levels achieve 100% success rate
- **Optimized implementation**: Cursor-based optimistic locking and SQL functions are highly efficient

### When to Use Each Method

- **Append**: Use for simple event appends where no conditions are needed
- **AppendIf**: Use for conditional appends requiring optimistic locking

## Implementation Details

- **Database**: PostgreSQL with events table and append functions
- **Streaming**: Multiple approaches for different dataset sizes
- **Extensions**: Channel-based streaming for Go-idiomatic processing

See [examples](examples.md) for complete working examples including course subscriptions and money transfers, and [getting-started](getting-started.md) for setup instructions.

## Implementation Details

- **Database**: PostgreSQL with events table and append functions
- **Streaming**: Multiple approaches for different dataset sizes
- **Extensions**: Channel-based streaming for Go-idiomatic processing

See [examples](examples.md) for complete working examples including course subscriptions and money transfers, and [getting-started](getting-started.md) for setup instructions.