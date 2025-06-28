# Overview: Dynamic Consistency Boundary (DCB) in go-crablet

go-crablet is a Go library for event sourcing, exploring and learning about concepts inspired by the [Dynamic Consistency Boundary (DCB)](https://dcb.events/) pattern. We're learning how DCB enables you to:

- Project multiple states and check business invariants in a single query
- Use tag-based, OR-combined queries for cross-entity consistency
- Enforce optimistic concurrency with combined append conditions

## Key Concepts

- **Batch Projection**: Project multiple states (decision model) in one database query
- **Combined Append Condition**: Use a single, OR-combined query for optimistic locking
- **Tag-based Queries**: Flexible, cross-entity queries using tags
- **Streaming**: Process events efficiently for large datasets

## Core Interface

```go
type EventStore interface {
    // Simple append without conditions
    Append(ctx context.Context, events []InputEvent) error
    
    // Conditional append with READ COMMITTED isolation
    AppendIf(ctx context.Context, events []InputEvent, condition AppendCondition) error
    
    // Conditional append with explicit isolation level control
    AppendIfSerializable(ctx context.Context, events []InputEvent, condition AppendCondition) error
    
    // Read events matching a query
    Read(ctx context.Context, query Query) ([]Event, error)
    ReadWithOptions(ctx context.Context, query Query, options *ReadOptions) ([]Event, error)
    
    // Project multiple states using projectors
    ProjectDecisionModel(ctx context.Context, projectors []BatchProjector) (map[string]any, AppendCondition, error)
}
```

## DCB Decision Model Pattern

We're exploring how a Dynamic Consistency Boundary decision model works:

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

## Implementation Details

- **Database**: PostgreSQL with events table and append functions
- **Streaming**: Multiple approaches for different dataset sizes
- **Extensions**: Channel-based streaming for Go-idiomatic processing
- **Isolation Levels**: Support for READ COMMITTED, REPEATABLE READ, and SERIALIZABLE

See [examples](examples.md) for complete working examples and [getting-started](getting-started.md) for setup instructions.