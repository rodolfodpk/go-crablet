# Usage Examples

This document provides practical examples of using go-crablet in different scenarios.

## Basic Usage

Here's a simple example of how to use go-crablet to store and query events:

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/rodolfodpk/go-crablet"
)

func main() {
    // Create a new event store with PostgreSQL connection
    store, err := dcb.NewEventStore("postgres://user:pass@localhost:5432/mydb?sslmode=disable")
    if err != nil {
        log.Fatal(err)
    }
    defer store.Close()

    ctx := context.Background()

    // Create tags for the event
    tags := dcb.NewTags(
        "course_id", "C123",
        "student_id", "S456",
    )

    // Create a new event
    event := dcb.NewInputEvent("StudentSubscribedToCourse", tags, map[string]interface{}{
        "subscription_date": "2024-03-20",
        "payment_method": "credit_card",
    })

    // Append the event to the store
    position, err := store.AppendEvents(ctx, []dcb.InputEvent{event}, dcb.NewQuery(nil), 0)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Event appended at position %d\n", position)
}
```

## Course Subscription System

Here's a more complete example of a course subscription system using go-crablet:

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/rodolfodpk/go-crablet"
)

// CourseState represents the current state of a course
type CourseState struct {
    ID          string
    Name        string
    StudentIDs  map[string]bool
    IsActive    bool
}

// StudentState represents the current state of a student
type StudentState struct {
    ID         string
    Name       string
    CourseIDs  map[string]bool
    IsActive   bool
}

func main() {
    store, err := dcb.NewEventStore("postgres://user:pass@localhost:5432/mydb?sslmode=disable")
    if err != nil {
        log.Fatal(err)
    }
    defer store.Close()

    ctx := context.Background()

    // Create a new course
    courseID := "C123"
    courseTags := dcb.NewTags("course_id", courseID)
    courseCreated := dcb.NewInputEvent("CourseCreated", courseTags, map[string]interface{}{
        "name": "Introduction to Go",
        "description": "Learn Go programming language",
    })

    // Create a new student
    studentID := "S456"
    studentTags := dcb.NewTags("student_id", studentID)
    studentRegistered := dcb.NewInputEvent("StudentRegistered", studentTags, map[string]interface{}{
        "name": "John Doe",
        "email": "john@example.com",
    })

    // Subscribe student to course
    subscriptionTags := dcb.NewTags(
        "course_id", courseID,
        "student_id", studentID,
    )
    subscriptionEvent := dcb.NewInputEvent("StudentSubscribedToCourse", subscriptionTags, map[string]interface{}{
        "subscription_date": "2024-03-20",
        "payment_method": "credit_card",
    })

    // Append all events
    events := []dcb.InputEvent{
        courseCreated,
        studentRegistered,
        subscriptionEvent,
    }
    position, err := store.AppendEvents(ctx, events, dcb.NewQuery(nil), 0)
    if err != nil {
        log.Fatal(err)
    }

    // Project course state
    courseProjector := dcb.StateProjector{
        Query: dcb.NewQuery(courseTags),
        InitialState: &CourseState{
            ID: courseID,
            StudentIDs: make(map[string]bool),
        },
        TransitionFn: func(state any, event dcb.Event) any {
            course := state.(*CourseState)
            switch event.Type {
            case "CourseCreated":
                data := event.Data.(map[string]interface{})
                course.Name = data["name"].(string)
                course.IsActive = true
            case "StudentSubscribedToCourse":
                course.StudentIDs[event.Tags["student_id"]] = true
            case "StudentUnsubscribedFromCourse":
                delete(course.StudentIDs, event.Tags["student_id"])
            case "CourseCancelled":
                course.IsActive = false
            }
            return course
        },
    }

    // Project student state
    studentProjector := dcb.StateProjector{
        Query: dcb.NewQuery(studentTags),
        InitialState: &StudentState{
            ID: studentID,
            CourseIDs: make(map[string]bool),
        },
        TransitionFn: func(state any, event dcb.Event) any {
            student := state.(*StudentState)
            switch event.Type {
            case "StudentRegistered":
                data := event.Data.(map[string]interface{})
                student.Name = data["name"].(string)
                student.IsActive = true
            case "StudentSubscribedToCourse":
                student.CourseIDs[event.Tags["course_id"]] = true
            case "StudentUnsubscribedFromCourse":
                delete(student.CourseIDs, event.Tags["course_id"])
            case "StudentDeactivated":
                student.IsActive = false
            }
            return student
        },
    }

    // Get current states
    _, courseState, err := store.ProjectState(ctx, courseProjector)
    if err != nil {
        log.Fatal(err)
    }

    _, studentState, err := store.ProjectState(ctx, studentProjector)
    if err != nil {
        log.Fatal(err)
    }

    // Print results
    course := courseState.(*CourseState)
    student := studentState.(*StudentState)

    fmt.Printf("Course: %s\n", course.Name)
    fmt.Printf("Active: %v\n", course.IsActive)
    fmt.Printf("Students: %d\n", len(course.StudentIDs))

    fmt.Printf("\nStudent: %s\n", student.Name)
    fmt.Printf("Active: %v\n", student.IsActive)
    fmt.Printf("Courses: %d\n", len(student.CourseIDs))
}
```

## Consistency Example

Here's an example showing how to use go-crablet's consistency features:

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/rodolfodpk/go-crablet"
)

func main() {
    store, err := dcb.NewEventStore("postgres://user:pass@localhost:5432/mydb?sslmode=disable")
    if err != nil {
        log.Fatal(err)
    }
    defer store.Close()

    ctx := context.Background()

    // Create a query for a specific course
    courseID := "C123"
    courseQuery := dcb.NewQuery(dcb.NewTags("course_id", courseID))

    // Get current position for the course
    _, position, err := store.GetPosition(ctx, courseQuery)
    if err != nil {
        log.Fatal(err)
    }

    // Create a new subscription event
    subscriptionTags := dcb.NewTags(
        "course_id", courseID,
        "student_id", "S456",
    )
    subscriptionEvent := dcb.NewInputEvent("StudentSubscribedToCourse", subscriptionTags, nil)

    // Append event with consistency check
    // This ensures no other events for this course were added since we last checked
    newPosition, err := store.AppendEvents(ctx, []dcb.InputEvent{subscriptionEvent}, courseQuery, position)
    if err != nil {
        if err == dcb.ErrConcurrentModification {
            fmt.Println("Course was modified by another process, please retry")
            return
        }
        log.Fatal(err)
    }

    fmt.Printf("Event appended at position %d\n", newPosition)
}
```

## Error Handling

Here's an example showing how to handle different types of errors:

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/rodolfodpk/go-crablet"
)

func main() {
    store, err := dcb.NewEventStore("postgres://user:pass@localhost:5432/mydb?sslmode=disable")
    if err != nil {
        log.Fatal(err)
    }
    defer store.Close()

    ctx := context.Background()

    // Example of handling concurrent modification
    courseID := "C123"
    courseQuery := dcb.NewQuery(dcb.NewTags("course_id", courseID))

    // Get current position
    _, position, err := store.GetPosition(ctx, courseQuery)
    if err != nil {
        log.Fatal(err)
    }

    // Try to append with retry logic
    maxRetries := 3
    for i := 0; i < maxRetries; i++ {
        event := dcb.NewInputEvent("CourseUpdated", dcb.NewTags("course_id", courseID), nil)
        newPosition, err := store.AppendEvents(ctx, []dcb.InputEvent{event}, courseQuery, position)
        if err != nil {
            if err == dcb.ErrConcurrentModification {
                // Get the new position and retry
                _, position, err = store.GetPosition(ctx, courseQuery)
                if err != nil {
                    log.Fatal(err)
                }
                continue
            }
            log.Fatal(err)
        }
        fmt.Printf("Event appended at position %d after %d retries\n", newPosition, i)
        break
    }

    // Example of handling invalid event data
    invalidEvent := dcb.NewInputEvent("CourseCreated", dcb.NewTags("course_id", "C999"), map[string]interface{}{
        "name": make(chan int), // Invalid data type
    })

    _, err = store.AppendEvents(ctx, []dcb.InputEvent{invalidEvent}, dcb.NewQuery(nil), 0)
    if err != nil {
        if err == dcb.ErrInvalidEventData {
            fmt.Println("Invalid event data:", err)
            return
        }
        log.Fatal(err)
    }
}
```

These examples demonstrate:
1. Basic event storage and retrieval
2. Building a complete system with multiple entities
3. Using consistency features to handle concurrent operations
4. Proper error handling and retry logic

For more details about specific features, please refer to the other documentation sections. 