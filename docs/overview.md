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
    // Simple append without conditions (Read Committed)
    Append(ctx context.Context, events []InputEvent) error
    
    // Conditional append (Repeatable Read)
    AppendIf(ctx context.Context, events []InputEvent, condition AppendCondition) error
    
    // Conditional append with strongest consistency (Serializable)
    AppendIfIsolated(ctx context.Context, events []InputEvent, condition AppendCondition) error
    
    // Read events matching a query
    Read(ctx context.Context, query Query) ([]Event, error)
    ReadWithOptions(ctx context.Context, query Query, options *ReadOptions) ([]Event, error)
    
    // Project multiple states using projectors
    ProjectDecisionModel(ctx context.Context, projectors []BatchProjector) (map[string]any, AppendCondition, error)
}

type ReadOptions struct {
    Cursor    *Cursor `json:"cursor"` // (transaction_id, position) tracking
    Limit     *int    `json:"limit"`
    BatchSize *int    `json:"batch_size"`
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
projectors := []dcb.BatchProjector{
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

go-crablet automatically chooses the optimal PostgreSQL transaction isolation level for each append method:

- **Append**: Uses **Read Committed** (fastest, safe for simple appends)
- **AppendIf**: Uses **Repeatable Read** (strong consistency for conditional appends)
- **AppendIfIsolated**: Uses **Serializable** (strongest consistency for critical operations)

Isolation levels are **implicit and not configurable** in the API. This ensures the best balance of safety and performance for each operation.

## Implementation Details

- **Database**: PostgreSQL with events table and append functions
- **Streaming**: Multiple approaches for different dataset sizes
- **Extensions**: Channel-based streaming for Go-idiomatic processing

See [examples](examples.md) for complete working examples including course subscriptions and money transfers, and [getting-started](getting-started.md) for setup instructions.