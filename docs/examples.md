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

For a complete example of a course subscription system, including state projection, event handling, and best practices, see [Course Subscription Example](docs/course-subscription.md).

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