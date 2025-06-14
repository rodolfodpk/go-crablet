# Minimal Example: Course Subscription

This document provides a detailed walkthrough of the minimal course subscription example, showing how events are created and stored.

## Complete Example

```go
package main

import (
    "context"
    "encoding/json"
    "time"
    "github.com/rodolfodpk/go-crablet/pkg/dcb"
    "github.com/jackc/pgx/v5/pgxpool"
)

func main() {
    pool, _ := pgxpool.New(context.Background(), "postgres://user:pass@localhost/db")
    store, _ := dcb.NewEventStore(context.Background(), pool)

    // Define projectors for business rules
    projectors := []dcb.BatchProjector{
        {ID: "courseExists", StateProjector: dcb.StateProjector{
            Query: dcb.NewQuery(dcb.NewTags("course_id", "c1"), "CourseDefined"),
            InitialState: false,
            TransitionFn: func(state any, e dcb.Event) any { return true },
        }},
        {ID: "studentExists", StateProjector: dcb.StateProjector{
            Query: dcb.NewQuery(dcb.NewTags("student_id", "s1"), "StudentRegistered"),
            InitialState: false,
            TransitionFn: func(state any, e dcb.Event) any { return true },
        }},
        {ID: "numSubscriptions", StateProjector: dcb.StateProjector{
            Query: dcb.NewQuery(dcb.NewTags("course_id", "c1"), "StudentSubscribed"),
            InitialState: 0,
            TransitionFn: func(state any, e dcb.Event) any { return state.(int) + 1 },
        }},
    }

    // Project states and get append condition (DCB pattern)
    states, appendCond, _ := store.ProjectDecisionModel(context.Background(), projectors, nil)
    
    // Business logic: create course if it doesn't exist
    if !states["courseExists"].(bool) {
        data, _ := json.Marshal(map[string]any{"CourseID": "c1", "Capacity": 2})
        courseEvent := dcb.NewInputEvent("CourseDefined", dcb.NewTags("course_id", "c1"), data)
        courseEvent.CorrelationID = "enrollment-123"
        store.Append(context.Background(), []dcb.InputEvent{courseEvent}, &appendCond)
    }
    
    // Business logic: create student if doesn't exist
    if !states["studentExists"].(bool) {
        data, _ := json.Marshal(map[string]any{"StudentID": "s1", "Name": "Alice", "Email": "alice@example.com"})
        studentEvent := dcb.NewInputEvent("StudentRegistered", dcb.NewTags("student_id", "s1"), data)
        studentEvent.CorrelationID = "enrollment-123"
        store.Append(context.Background(), []dcb.InputEvent{studentEvent}, &appendCond)
    }
    
    // Business logic: subscribe student if course not full
    if states["numSubscriptions"].(int) < 2 {
        data, _ := json.Marshal(map[string]any{"StudentID": "s1", "CourseID": "c1"})
        enrollEvent := dcb.NewInputEvent("StudentSubscribed", dcb.NewTags("student_id", "s1", "course_id", "c1"), data)
        enrollEvent.CorrelationID = "enrollment-123"
        store.Append(context.Background(), []dcb.InputEvent{enrollEvent}, &appendCond)
    }

    // Change course capacity with proper causation/correlation
    changeCourseCapacity(context.Background(), store, "c1", 5)
}

// Change course capacity with proper causation/correlation
func changeCourseCapacity(ctx context.Context, store dcb.EventStore, courseID string, newCapacity int) error {
    // Project current course state
    projectors := []dcb.BatchProjector{
        {ID: "courseCapacity", StateProjector: dcb.StateProjector{
            Query: dcb.NewQuery(dcb.NewTags("course_id", courseID), "CourseDefined", "CourseCapacityChanged"),
            InitialState: 0,
            TransitionFn: func(state any, e dcb.Event) any {
                switch e.Type {
                case "CourseDefined":
                    var data map[string]any
                    json.Unmarshal(e.Data, &data)
                    return int(data["Capacity"].(float64))
                case "CourseCapacityChanged":
                    var data map[string]any
                    json.Unmarshal(e.Data, &data)
                    return int(data["NewCapacity"].(float64))
                }
                return state
            },
        }},
    }
    
    states, appendCond, _ := store.ProjectDecisionModel(ctx, projectors, nil)
    currentCapacity := states["courseCapacity"].(int)
    
    // Create capacity change event with causation/correlation
    correlationID := "capacity-change-" + courseID
    data, _ := json.Marshal(map[string]any{
        "CourseID": courseID,
        "OldCapacity": currentCapacity,
        "NewCapacity": newCapacity,
        "ChangedAt": time.Now(),
    })
    
    capacityEvent := dcb.NewInputEvent("CourseCapacityChanged", dcb.NewTags("course_id", courseID), data)
    capacityEvent.CorrelationID = correlationID
    
    // If we had a previous event, we could set causation:
    // capacityEvent.CausationID = previousEventID
    
    _, err := store.Append(ctx, []dcb.InputEvent{capacityEvent}, &appendCond)
    return err
}
```

## Resulting Events

After running the minimal example, the events table will contain:

```sql
SELECT id, type, tags, data, position, causation_id, correlation_id 
FROM events 
ORDER BY position;
```

| id | type | tags | data | position | causation_id | correlation_id |
|----|------|------|------|----------|--------------|----------------|
| 1 | CourseDefined | `{"course_id": "c1"}` | `{"CourseID": "c1", "Capacity": 2}` | 1 | course_id_01h2xcejqtf2nbrexx3vqjhp41 | course_id_01h2xcejqtf2nbrexx3vqjhp41 |
| 2 | StudentRegistered | `{"student_id": "s1"}` | `{"StudentID": "s1", "Name": "Alice", "Email": "alice@example.com"}` | 2 | student_id_01h2xcejqtf2nbrexx3vqjhp42 | student_id_01h2xcejqtf2nbrexx3vqjhp42 |
| 3 | StudentSubscribed | `{"student_id": "s1", "course_id": "c1"}` | `{"StudentID": "s1", "CourseID": "c1"}` | 3 | course_id_student_id_01h2xcejqtf2nbrexx3vqjhp43 | course_id_student_id_01h2xcejqtf2nbrexx3vqjhp43 |
| 4 | CourseCapacityChanged | `{"course_id": "c1"}` | `{"CourseID": "c1", "OldCapacity": 2, "NewCapacity": 5, "ChangedAt": "..."}` | 4 | course_id_01h2xcejqtf2nbrexx3vqjhp44 | capacity-change-c1 |

**Event Flow:**
1. **CourseDefined**: Creates course "c1" with capacity 2
2. **StudentRegistered**: Registers student "s1" (Alice)
3. **StudentSubscribed**: Enrolls student "s1" in course "c1"
4. **CourseCapacityChanged**: Increases course capacity from 2 to 5

**Causation and Correlation Benefits:**
- **Correlation ID**: Groups all capacity change operations for the same course
- **Causation ID**: Can link to the event that triggered the capacity change
- **Audit Trail**: Complete history of who changed what and when
- **Debugging**: Trace exactly which operation caused the capacity change 