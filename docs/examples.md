# DCB Example: Course Subscription with Invariants

This example demonstrates the Dynamic Consistency Boundary (DCB) pattern using go-crablet. It shows how to:
- Project multiple states (decision model) in a single query
- Enforce business invariants (course exists, not full, student not already subscribed)
- Use a combined append condition for optimistic concurrency

## Example: Course Subscription Command Handler

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
            Query: dcb.NewQuery(dcb.NewTags("course_id", courseID), "CourseDefined"),
            InitialState: false,
            TransitionFn: func(state any, e dcb.Event) any { return true },
        }},
        {ID: "numSubscriptions", StateProjector: dcb.StateProjector{
            Query: dcb.NewQuery(dcb.NewTags("course_id", courseID), "StudentSubscribed"),
            InitialState: 0,
            TransitionFn: func(state any, e dcb.Event) any { return state.(int) + 1 },
        }},
        {ID: "alreadySubscribed", StateProjector: dcb.StateProjector{
            Query: dcb.NewQuery(dcb.NewTags("student_id", studentID, "course_id", courseID), "StudentSubscribed"),
            InitialState: false,
            TransitionFn: func(state any, e dcb.Event) any { return true },
        }},
    }
    query := dcb.NewQueryFromItems(
        dcb.NewQueryItem([]string{"CourseDefined"}, dcb.NewTags("course_id", courseID)),
        dcb.NewQueryItem([]string{"StudentSubscribed"}, dcb.NewTags("course_id", courseID)),
        dcb.NewQueryItem([]string{"StudentSubscribed"}, dcb.NewTags("student_id", studentID, "course_id", courseID)),
    )
    states, appendCond, _ := store.ProjectDecisionModel(context.Background(), query, nil, projectors)

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

**Key points:**
- All invariants are checked in a single query (batch projection)
- The append condition is the OR-combination of all projector queries
- Only one database round trip is needed for all business rules
- No aggregates or legacy event sourcing patterns required 