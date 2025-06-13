# Overview: Dynamic Consistency Boundary (DCB) in go-crablet

go-crablet is a Go library for event sourcing, built around the [Dynamic Consistency Boundary (DCB)](https://dcb.events/) pattern. DCB enables you to:

- Project multiple states and check business invariants in a single query
- Use tag-based, OR-combined queries for cross-entity consistency
- Enforce optimistic concurrency with combined append conditions
- Stream events efficiently for large datasets

## Key Concepts

- **Batch Projection**: Project multiple states (decision model) in one database query
- **Combined Append Condition**: Use a single, OR-combined query for optimistic locking
- **Streaming**: Process events row-by-row, suitable for millions of events
- **Tag-based Queries**: Flexible, cross-entity queries using tags

## DCB Decision Model Pattern

A DCB decision model is built by:
1. Defining projectors for each business rule or invariant
2. Projecting all states in a single query
3. Building a combined append condition
4. Appending new events only if all invariants still hold

## Example: Course Subscription

```go
projectors := []dcb.BatchProjector{
    {ID: "courseExists", StateProjector: dcb.StateProjector{...}},
    {ID: "numSubscriptions", StateProjector: dcb.StateProjector{...}},
}
query := dcb.NewQueryFromItems(...)
states, appendCond, _ := store.ProjectDecisionModel(ctx, query, nil, projectors)
if !states["courseExists"].(bool) { /* append CourseDefined */ }
if states["numSubscriptions"].(int) < 2 { /* append StudentSubscribed */ }
```

## Streaming & Memory Efficiency
- **ProjectDecisionModel**: Projects all states in one query, streams events row-by-row
- **ReadStream**: Streams events for custom processing

## Why DCB?
- **Single-query consistency**: All invariants checked atomically
- **No aggregates required**: Consistency boundaries are defined by your queries
- **Efficient**: One database round trip for all business rules

See the [README](../README.md) and [examples](examples.md) for more.