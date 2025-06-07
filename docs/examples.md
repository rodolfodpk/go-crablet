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

    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/rodolfodpk/go-crablet"
)

func main() {
    // Create a PostgreSQL connection pool
    pool, err := pgxpool.New(context.Background(), "postgres://user:pass@localhost:5432/mydb?sslmode=disable")
    if err != nil {
        log.Fatal(err)
    }
    defer pool.Close()

    // Create a new event store
    store, err := dcb.NewEventStore(context.Background(), pool)
    if err != nil {
        log.Fatal(err)
    }

    ctx := context.Background()

    // Create tags for the event
    tags := dcb.NewTags(
        "course_id", "C123",
        "student_id", "S456",
    )

    // Create a new event
    event := dcb.NewInputEvent(
        "StudentSubscribedToCourse", 
        tags, 
        []byte(`{"subscription_date": "2024-03-20", "payment_method": "credit_card"}`),
    )

    // Define the consistency boundary
    query := dcb.NewQuery(tags, "StudentSubscribedToCourse")

    // Get current stream position
    position, err := store.GetCurrentPosition(ctx, query)
    if err != nil {
        log.Fatal(err)
    }

    // Append the event to the store using the current position
    newPosition, err := store.AppendEvents(ctx, []dcb.InputEvent{event}, query, position)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Event appended at position %d\n", newPosition)
}
```

## Course Subscription System

Here's a more complete example of a course subscription system using go-crablet:

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "time"

    "github.com/jackc/pgx/v5/pgxpool"
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
    // Create a PostgreSQL connection pool
    pool, err := pgxpool.New(context.Background(), "postgres://user:pass@localhost:5432/mydb?sslmode=disable")
    if err != nil {
        log.Fatal(err)
    }
    defer pool.Close()

    // Create a new event store
    store, err := dcb.NewEventStore(context.Background(), pool)
    if err != nil {
        log.Fatal(err)
    }

    ctx := context.Background()

    // Create a new course
    courseID := "C123"
    courseTags := dcb.NewTags("course_id", courseID)
    courseCreated := dcb.NewInputEvent(
        "CourseCreated", 
        courseTags, 
        []byte(`{"name": "Introduction to Go", "description": "Learn Go programming language"}`),
    )

    // Create a new student
    studentID := "S456"
    studentTags := dcb.NewTags("student_id", studentID)
    studentRegistered := dcb.NewInputEvent(
        "StudentRegistered", 
        studentTags, 
        []byte(`{"name": "John Doe", "email": "john@example.com"}`),
    )

    // Subscribe student to course
    subscriptionTags := dcb.NewTags(
        "course_id", courseID,
        "student_id", studentID,
    )
    subscriptionEvent := dcb.NewInputEvent(
        "StudentSubscribedToCourse", 
        subscriptionTags, 
        []byte(`{"subscription_date": "2024-03-20", "payment_method": "credit_card"}`),
    )

    // Append all events with a query that includes all relevant tags
    query := dcb.NewQuery(subscriptionTags, "CourseCreated", "StudentRegistered", "StudentSubscribedToCourse")
    
    // Get current stream position
    position, err := store.GetCurrentPosition(ctx, query)
    if err != nil {
        log.Fatal(err)
    }

    events := []dcb.InputEvent{
        courseCreated,
        studentRegistered,
        subscriptionEvent,
    }
    newPosition, err := store.AppendEvents(ctx, events, query, position)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Events appended at position %d\n", newPosition)

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
                var data struct {
                    Name string `json:"name"`
                }
                if err := json.Unmarshal(event.Data, &data); err != nil {
                    panic(err)
                }
                course.Name = data.Name
                course.IsActive = true
            case "StudentSubscribedToCourse":
                for _, tag := range event.Tags {
                    if tag.Key == "student_id" {
                        course.StudentIDs[tag.Value] = true
                    }
                }
            case "StudentUnsubscribedFromCourse":
                for _, tag := range event.Tags {
                    if tag.Key == "student_id" {
                        delete(course.StudentIDs, tag.Value)
                    }
                }
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
                var data struct {
                    Name string `json:"name"`
                }
                if err := json.Unmarshal(event.Data, &data); err != nil {
                    panic(err)
                }
                student.Name = data.Name
                student.IsActive = true
            case "StudentSubscribedToCourse":
                for _, tag := range event.Tags {
                    if tag.Key == "course_id" {
                        student.CourseIDs[tag.Value] = true
                    }
                }
            case "StudentUnsubscribedFromCourse":
                for _, tag := range event.Tags {
                    if tag.Key == "course_id" {
                        delete(student.CourseIDs, tag.Value)
                    }
                }
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

## Error Handling

Here's an example showing how to handle different types of errors:

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/rodolfodpk/go-crablet"
)

func main() {
    // Create a PostgreSQL connection pool
    pool, err := pgxpool.New(context.Background(), "postgres://user:pass@localhost:5432/mydb?sslmode=disable")
    if err != nil {
        log.Fatal(err)
    }
    defer pool.Close()

    // Create a new event store
    store, err := dcb.NewEventStore(context.Background(), pool)
    if err != nil {
        log.Fatal(err)
    }

    ctx := context.Background()

    // Example of handling validation errors
    courseID := "C123"
    courseTags := dcb.NewTags("course_id", courseID)
    query := dcb.NewQuery(courseTags, "CourseUpdated")

    // Try to append with invalid event data
    invalidEvent := dcb.NewInputEvent(
        "CourseUpdated", 
        courseTags, 
        []byte(`invalid json`), // Invalid JSON data
    )

    _, err = store.AppendEvents(ctx, []dcb.InputEvent{invalidEvent}, query, 0)
    if err != nil {
        if validationErr, ok := err.(*dcb.ValidationError); ok {
            fmt.Printf("Validation error: %v\n", validationErr)
            return
        }
        log.Fatal(err)
    }

    // Example of handling concurrency errors
    // First append
    event1 := dcb.NewInputEvent(
        "CourseUpdated", 
        courseTags, 
        []byte(`{"title": "New Title"}`),
    )
    position, err := store.AppendEvents(ctx, []dcb.InputEvent{event1}, query, 0)
    if err != nil {
        log.Fatal(err)
    }

    // Try to append another event with the same query but old position
    event2 := dcb.NewInputEvent(
        "CourseUpdated", 
        courseTags, 
        []byte(`{"title": "Another Title"}`),
    )
    _, err = store.AppendEvents(ctx, []dcb.InputEvent{event2}, query, 0) // Using position 0 instead of the new position
    if err != nil {
        if _, ok := err.(*dcb.ConcurrencyError); ok {
            fmt.Println("Concurrency error: another event was appended to this stream")
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
4. Proper error handling for validation and concurrency

For more details about specific features, please refer to the other documentation sections. 