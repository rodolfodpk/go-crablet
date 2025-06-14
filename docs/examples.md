# DCB-Inspired Example: Course Subscription with Invariants

This example demonstrates our exploration of the Dynamic Consistency Boundary (DCB) pattern using go-crablet. It shows how we're learning to:
- Project multiple states (decision model) in a single query
- Enforce business invariants (course exists, not full, student not already subscribed)
- Use a combined append condition for optimistic concurrency
- Use channel-based streaming for Go-idiomatic event processing

## Example: Course Subscription Command Handler

### Traditional Cursor-Based Approach

```go
package main

import (
    "context"
    "encoding/json"
    "github.com/rodolfodpk/go-crablet/pkg/dcb"
    "github.com/jackc/pgx/v5/pgxpool"
)

type CourseDefined struct {
    CourseID string
    Capacity int
}

type StudentSubscribed struct {
    StudentID string
    CourseID  string
}

func main() {
    pool, _ := pgxpool.New(context.Background(), "postgres://user:pass@localhost/db")
    store, _ := dcb.NewEventStore(context.Background(), pool)

    courseID := "c1"
    studentID := "s1"

    // Define projectors for the decision model
    projectors := []dcb.BatchProjector{
        {ID: "courseExists", StateProjector: dcb.StateProjector{
            Query: dcb.NewQuerySimple(dcb.NewTags("course_id", courseID), "CourseDefined"),
            InitialState: false,
            TransitionFn: func(state any, e dcb.Event) any { return true },
        }},
        {ID: "numSubscriptions", StateProjector: dcb.StateProjector{
            Query: dcb.NewQuerySimple(dcb.NewTags("course_id", courseID), "StudentSubscribed"),
            InitialState: 0,
            TransitionFn: func(state any, e dcb.Event) any { return state.(int) + 1 },
        }},
        {ID: "alreadySubscribed", StateProjector: dcb.StateProjector{
            Query: dcb.NewQuerySimple(dcb.NewTags("student_id", studentID, "course_id", courseID), "StudentSubscribed"),
            InitialState: false,
            TransitionFn: func(state any, e dcb.Event) any { return true },
        }},
    }

    // Project all states in single query (traditional cursor-based approach)
    states, appendCond, _ := store.ProjectDecisionModel(context.Background(), projectors, nil)

    if !states["courseExists"].(bool) {
        // Append CourseDefined event
        data, _ := json.Marshal(CourseDefined{courseID, 2})
        store.Append(context.Background(), []dcb.InputEvent{
            dcb.NewInputEvent("CourseDefined", dcb.NewTags("course_id", courseID), data),
        }, &appendCond)
    }
    if states["alreadySubscribed"].(bool) {
        panic("student already subscribed")
    }
    if states["numSubscriptions"].(int) >= 2 {
        panic("course is full")
    }
    // Subscribe student
    data, _ := json.Marshal(StudentSubscribed{studentID, courseID})
    store.Append(context.Background(), []dcb.InputEvent{
        dcb.NewInputEvent("StudentSubscribed", dcb.NewTags("student_id", studentID, "course_id", courseID), data),
    }, &appendCond)
}
```

### Channel-Based Approach (New!)

```go
func channelBasedExample() {
    pool, _ := pgxpool.New(context.Background(), "postgres://user:pass@localhost/db")
    store, _ := dcb.NewEventStore(context.Background(), pool)
    
    // Get channel-based store
    channelStore := store.(dcb.ChannelEventStore)

    courseID := "c1"
    studentID := "s1"

    // Define the same projectors
    projectors := []dcb.BatchProjector{
        {ID: "courseExists", StateProjector: dcb.StateProjector{
            Query: dcb.NewQuerySimple(dcb.NewTags("course_id", courseID), "CourseDefined"),
            InitialState: false,
            TransitionFn: func(state any, e dcb.Event) any { return true },
        }},
        {ID: "numSubscriptions", StateProjector: dcb.StateProjector{
            Query: dcb.NewQuerySimple(dcb.NewTags("course_id", courseID), "StudentSubscribed"),
            InitialState: 0,
            TransitionFn: func(state any, e dcb.Event) any { return state.(int) + 1 },
        }},
        {ID: "alreadySubscribed", StateProjector: dcb.StateProjector{
            Query: dcb.NewQuerySimple(dcb.NewTags("student_id", studentID, "course_id", courseID), "StudentSubscribed"),
            InitialState: false,
            TransitionFn: func(state any, e dcb.Event) any { return true },
        }},
    }

    // Channel-based projection with real-time feedback
    resultChan, _ := channelStore.ProjectDecisionModelChannel(context.Background(), projectors, nil)
    
    // Process results as they come in
    finalStates := make(map[string]interface{})
    for result := range resultChan {
        if result.Error != nil {
            fmt.Printf("Error: %v\n", result.Error)
            continue
        }
        
        finalStates[result.ProjectorID] = result.State
        
        fmt.Printf("Projector %s processed event %s (position %d)\n", 
            result.ProjectorID, result.Event.Type, result.Position)
    }

    // Apply business rules using final states
    if !finalStates["courseExists"].(bool) {
        // Append CourseDefined event
    }
    if finalStates["alreadySubscribed"].(bool) {
        panic("student already subscribed")
    }
    if finalStates["numSubscriptions"].(int) >= 2 {
        panic("course is full")
    }
}
```

### Channel-Based Event Streaming

```go
func channelStreamingExample() {
    pool, _ := pgxpool.New(context.Background(), "postgres://user:pass@localhost/db")
    store, _ := dcb.NewEventStore(context.Background(), pool)
    
    // Get channel-based store
    channelStore := store.(dcb.ChannelEventStore)

    // Create query for course events
    query := dcb.NewQuerySimple(dcb.NewTags("course_id", "c1"), "CourseDefined", "StudentSubscribed")

    // Channel-based streaming
    eventChan, err := channelStore.ReadStreamChannel(context.Background(), query, nil)
    if err != nil {
        panic(err)
    }

    // Process events in real-time
    for event := range eventChan {
        fmt.Printf("Processing event: %s at position %d\n", event.Type, event.Position)
        
        // Process event based on type
        switch event.Type {
        case "CourseDefined":
            fmt.Println("Course was defined")
        case "StudentSubscribed":
            fmt.Println("Student was subscribed")
        }
    }
}
```

## Key Points We're Exploring

- **All invariants are checked in a single query** (batch projection)
- **The append condition is the OR-combination of all projector queries**
- **Only one database round trip is needed for all business rules**
- **No aggregates or legacy event sourcing patterns required**
- **Channel-based streaming provides real-time processing feedback**
- **Choose the right streaming approach for your dataset size**

## Performance Comparison

| Approach | Best For | Real-time Feedback | Memory Usage |
|----------|----------|-------------------|--------------|
| **Traditional** | < 100 events | ❌ No | High |
| **Cursor-based** | > 1000 events | ❌ No | Low |
| **Channel-based** | 100-500 events | ✅ Yes | Moderate |

## Available Examples

- **`examples/cursor_streaming/`** - Cursor-based streaming for large datasets
- **`examples/channel_streaming/`** - Channel-based streaming for small-medium datasets
- **`examples/channel_projection/`** - Channel-based projection with real-time feedback
- **`examples/extension_interface/`** - Extension interface pattern demonstration
- **`examples/transfer/`** - Event sourcing with semantic event names
- **`examples/enrollment/`** - Course enrollment with business rules 