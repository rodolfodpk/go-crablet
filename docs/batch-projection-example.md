# Batch Projection - A go-crablet Feature

Batch projection is a feature provided by go-crablet that allows you to:
- Project multiple states (decision model) in a single query
- Enforce all business invariants atomically
- Use a combined append condition for optimistic concurrency

## Example: Course Subscription

```go
projectors := []dcb.BatchProjector{
    {ID: "courseExists", StateProjector: dcb.StateProjector{...}},
    {ID: "numSubscriptions", StateProjector: dcb.StateProjector{...}},
    {ID: "alreadySubscribed", StateProjector: dcb.StateProjector{...}},
}
query := dcb.NewQueryFromItems(...)
states, appendCond, _ := store.ProjectDecisionModel(ctx, query, nil, projectors)
if !states["courseExists"].(bool) { /* append CourseDefined */ }
if states["alreadySubscribed"].(bool) { panic("student already subscribed") }
if states["numSubscriptions"].(int) >= 2 { panic("course is full") }
// Append StudentSubscribed event
```

**Benefits:**
- All invariants checked in one query
- One append condition for optimistic locking
- One database round trip for all business rules 