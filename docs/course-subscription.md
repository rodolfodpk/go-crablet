# Course Subscription System Example

This example demonstrates how to use go-crablet to implement a course subscription system. It shows how to handle subscriptions, manage state, and use consistency boundaries effectively.

## Implementation

```go
// Define event types
const (
    EventTypeSubscription = "Subscription"
    EventTypeUnsubscription = "Unsubscription"
)

// Define subscription state
type SubscriptionState struct {
    IsActive bool
    Since    time.Time
    Until    time.Time
}

// Create a projector for subscription state
subscriptionProjector := dcb.StateProjector{
    Query: dcb.NewQuery(
        dcb.NewTags("course_id", "C1", "student_id", "S1"),
        EventTypeSubscription,
        EventTypeUnsubscription,
    ),
    InitialState: &SubscriptionState{},
    TransitionFn: func(state any, event dcb.Event) any {
        s := state.(*SubscriptionState)
        switch event.Type {
        case EventTypeSubscription:
            var data struct{ Until time.Time }
            if err := json.Unmarshal(event.Data, &data); err != nil {
                panic(err)
            }
            s.IsActive = true
            s.Since = time.Now()
            s.Until = data.Until
        case EventTypeUnsubscription:
            s.IsActive = false
        }
        return s
    },
}

// Append a subscription event
events := []dcb.InputEvent{
    {
        Type: EventTypeSubscription,
        Tags: dcb.NewTags("course_id", "C1", "student_id", "S1"),
        Data: []byte(`{"until": "2024-12-31T23:59:59Z"}`),
    },
}

// Use a query to define the consistency boundary
query := dcb.NewQuery(
    dcb.NewTags("course_id", "C1", "student_id", "S1"),
    EventTypeSubscription,
    EventTypeUnsubscription,
)

// Append events with optimistic locking
position, err := store.AppendEvents(ctx, events, query, 0)
if err != nil {
    panic(err)
}

// Project current state
_, state, err := store.ProjectState(ctx, subscriptionProjector)
if err != nil {
    panic(err)
}

subscription := state.(*SubscriptionState)
fmt.Printf("Subscription active: %v, until: %v\n", 
    subscription.IsActive, 
    subscription.Until,
)
```

## Key Features Demonstrated

1. **Event Types and State**: Defines clear event types and a state structure to track subscriptions
2. **State Projection**: Uses a projector to build the current subscription state from events
3. **Consistency Boundaries**: Uses tags and event types to define consistency boundaries
4. **Optimistic Locking**: Appends events with position-based concurrency control
5. **Event Data Handling**: Properly marshals and unmarshals event data

## Best Practices

1. **Tag Usage**: Use consistent tag keys (`course_id`, `student_id`) for querying
2. **Event Types**: Define clear event types for different subscription actions
3. **State Structure**: Keep state focused on the subscription domain
4. **Error Handling**: Add proper error handling in production code
5. **Position Management**: Use current stream positions for optimistic locking

For more examples and detailed documentation, see:
- [Overview](overview.md)
- [State Projection](state-projection.md)
- [Examples](examples.md) 