# Minimal Example: Course Subscription

This document provides a detailed walkthrough of the minimal course subscription example, showing how events are created and stored.

## Complete Example

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "time"
    "github.com/rodolfodpk/go-crablet/pkg/dcb"
    "github.com/jackc/pgx/v5/pgxpool"
)

// Generate unique IDs for better concurrency
func generateUniqueID(prefix string) string {
    return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}

func main() {
    ctx := context.Background()
    pool, _ := pgxpool.New(ctx, "postgres://postgres:postgres@localhost:5432/dcb_app?sslmode=disable")
    store, _ := dcb.NewEventStore(ctx, pool)

    // Generate unique IDs for this example
    courseID := generateUniqueID("course")
    studentID := generateUniqueID("student")

    // Define projectors for course capacity check
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
    }

    // Project states and get append condition (exploring Dynamic Consistency Boundary concepts)
    states, appendCond, _ := store.ProjectDecisionModel(ctx, projectors, nil)
    
    // Check business rules
    courseExists := states["courseExists"].(bool)
    numSubscriptions := states["numSubscriptions"].(int)

    fmt.Printf("Course exists: %v, Current subscriptions: %d\n", courseExists, numSubscriptions)

    // Create course if it doesn't exist
    if !courseExists {
        data, _ := json.Marshal(map[string]any{"CourseID": courseID, "Capacity": 2})
        courseEvent := dcb.NewInputEvent("CourseDefined", dcb.NewTags("course_id", courseID), data)
        
        _, err := store.Append(ctx, []dcb.InputEvent{courseEvent}, &appendCond)
        if err != nil {
            log.Fatalf("Failed to create course: %v", err)
        }
        fmt.Printf("Created course %s with capacity 2\n", courseID)
    }

    // Enroll student if capacity allows
    if numSubscriptions < 2 {
        data, _ := json.Marshal(map[string]any{"StudentID": studentID, "CourseID": courseID})
        enrollEvent := dcb.NewInputEvent("StudentSubscribed", dcb.NewTags("student_id", studentID, "course_id", courseID), data)
        
        _, err := store.Append(ctx, []dcb.InputEvent{enrollEvent}, &appendCond)
        if err != nil {
            log.Fatalf("Failed to enroll student: %v", err)
        }
        fmt.Printf("Enrolled student %s in course %s\n", studentID, courseID)
    } else {
        fmt.Printf("Course %s is full, cannot enroll student %s\n", courseID, studentID)
    }

    // Demonstrate capacity change
    changeCourseCapacity(ctx, store, courseID, 5)
}

// Change course capacity
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
    
    // Create capacity change event
    data, _ := json.Marshal(map[string]any{
        "CourseID": courseID,
        "OldCapacity": currentCapacity,
        "NewCapacity": newCapacity,
        "ChangedAt": time.Now(),
    })
    
    capacityEvent := dcb.NewInputEvent("CourseCapacityChanged", dcb.NewTags("course_id", courseID), data)
    
    _, err := store.Append(ctx, []dcb.InputEvent{capacityEvent}, &appendCond)
    return err
}

## Resulting Events

After running the minimal example, the events table will contain:

```sql
SELECT type, tags, data, position 
FROM events 
ORDER BY position;
```

| type | tags | data | position |
|------|------|------|----------|
| CourseDefined | `{"course_id": "course-1234567890"}` | `{"CourseID": "course-1234567890", "Capacity": 2}` | 1 |
| StudentSubscribed | `{"student_id": "student-1234567891", "course_id": "course-1234567890"}` | `{"StudentID": "student-1234567891", "CourseID": "course-1234567890"}` | 2 |
| CourseCapacityChanged | `{"course_id": "course-1234567890"}` | `{"CourseID": "course-1234567890", "OldCapacity": 2, "NewCapacity": 5, "ChangedAt": "..."}` | 3 |

**Event Flow:**
1. **CourseDefined**: Creates course with unique ID and capacity 2
2. **StudentSubscribed**: Enrolls student with unique ID in the course
3. **CourseCapacityChanged**: Increases course capacity from 2 to 5

**Benefits:**
- **Audit Trail**: Complete history of course changes and enrollments
- **Debugging**: Trace exactly which operations occurred and when
- **Concurrency**: Unique IDs prevent conflicts between different examples 