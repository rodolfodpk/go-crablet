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
    ctx := context.Background()
    pool, _ := pgxpool.New(ctx, "postgres://user:pass@localhost/db")
    store, _ := dcb.NewEventStore(ctx, pool)

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
    states, appendCond, _ := store.ProjectDecisionModel(ctx, projectors, nil)

    if !states["courseExists"].(bool) {
        // Append CourseDefined event
        data, _ := json.Marshal(CourseDefined{courseID, 2})
        store.Append(ctx, []dcb.InputEvent{
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
    store.Append(ctx, []dcb.InputEvent{
        dcb.NewInputEvent("StudentSubscribed", dcb.NewTags("student_id", studentID, "course_id", courseID), data),
    }, &appendCond)
}
```

### Channel-Based Approach (New!)

```go
func channelBasedExample() {
    ctx := context.Background()
    pool, _ := pgxpool.New(ctx, "postgres://user:pass@localhost/db")
    store, _ := dcb.NewEventStore(ctx, pool)
    
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

    // Channel-based projection with immediate feedback
    resultChan, _ := channelStore.ProjectDecisionModelChannel(ctx, projectors, nil)
    
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
    ctx := context.Background()
    pool, _ := pgxpool.New(ctx, "postgres://user:pass@localhost/db")
    store, _ := dcb.NewEventStore(ctx, pool)
    
    // Get channel-based store
    channelStore := store.(dcb.ChannelEventStore)

    // Create query for course events
    query := dcb.NewQuerySimple(dcb.NewTags("course_id", "c1"), "CourseDefined", "StudentSubscribed")

    // Channel-based streaming
    eventChan, err := channelStore.ReadStreamChannel(ctx, query, nil)
    if err != nil {
        panic(err)
    }

    // Process events with immediate delivery
    for event := range eventChan {
        fmt.Printf("Event: %s at position %d\n", event.Type, event.Position)
        
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
- **Channel-based streaming provides immediate processing feedback**
- **Choose the right streaming approach for your dataset size**

## Query Building with Helper Functions

go-crablet provides concise helper functions to simplify query building:

### Using QItem and QItemKV Helpers

**Before (verbose):**
```go
Query: dcb.NewQuerySimple(dcb.NewTags("course_id", courseID), "CourseDefined")
```

**After (concise):**
```go
Query: dcb.NewQueryFromItems(dcb.QItemKV("CourseDefined", "course_id", courseID))
```

**Complete example with helpers:**
```go
// Define projectors using the new helper functions
projectors := []dcb.BatchProjector{
    {ID: "courseExists", StateProjector: dcb.StateProjector{
        Query: dcb.NewQueryFromItems(dcb.QItemKV("CourseDefined", "course_id", courseID)),
        InitialState: false,
        TransitionFn: func(state any, e dcb.Event) any { return true },
    }},
    {ID: "numSubscriptions", StateProjector: dcb.StateProjector{
        Query: dcb.NewQueryFromItems(dcb.QItemKV("StudentSubscribed", "course_id", courseID)),
        InitialState: 0,
        TransitionFn: func(state any, e dcb.Event) any { return state.(int) + 1 },
    }},
    {ID: "alreadySubscribed", StateProjector: dcb.StateProjector{
        Query: dcb.NewQueryFromItems(dcb.QItemKV("StudentSubscribed", "student_id", studentID, "course_id", courseID)),
        InitialState: false,
        TransitionFn: func(state any, e dcb.Event) any { return true },
    }},
}
```

### Building Complex Queries

For more complex queries with multiple conditions:

```go
// Build a query with multiple event types and tags
query := dcb.NewQueryFromItems(
    dcb.QItemKV("CourseDefined", "course_id", "c1"),
    dcb.QItemKV("StudentRegistered", "student_id", "s1"),
    dcb.QItemKV("StudentSubscribed", "course_id", "c1"),
    dcb.QItemKV("StudentSubscribed", "student_id", "s1"),
)

// Read events with the combined query
events, err := store.Read(ctx, query, nil)
```

## Performance Comparison

| Approach | Best For | Immediate Feedback | Memory Usage |
|----------|----------|-------------------|--------------|
| **Cursor-based** | Large datasets | ❌ No | Low |
| **Channel-based** | Small-medium datasets | ✅ Yes | Moderate |

## Available Examples

- **`internal/examples/cursor_streaming/`** - Cursor-based streaming for large datasets
- **`internal/examples/channel_streaming/`** - Channel-based streaming for small-medium datasets
- **`internal/examples/channel_projection/`** - Channel-based projection with immediate feedback
- **`internal/examples/extension_interface/`** - Extension interface pattern demonstration
- **`internal/examples/transfer/`** - Event sourcing with semantic event names
- **`internal/examples/enrollment/`** - Course enrollment with business rules