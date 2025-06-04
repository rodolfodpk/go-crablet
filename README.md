# go-crablet

[![codecov](https://codecov.io/gh/rodolfo/go-crablet/branch/main/graph/badge.svg)](https://codecov.io/gh/rodolfo/go-crablet)

A Go implementation of the Dynamic Consistency Boundary (DCB) event store pattern, providing a simpler and more flexible approach to consistency in event-driven systems.

## Overview

go-crablet is a Go library that implements the [Dynamic Consistency Boundary (DCB)](https://dcb.events/) pattern, introduced by Sara Pellegrini in her blog post "Killing the Aggregate". DCB provides a pragmatic approach to balancing strong consistency with flexibility in event-driven systems, without relying on rigid transactional boundaries.

Unlike traditional event sourcing approaches that use strict constraints to maintain immediate consistency, DCB allows for selective enforcement of strong consistency where needed, particularly for operations that span multiple entities. This ensures critical business processes and cross-entity invariants remain reliable while avoiding the constraints of traditional transactional models.

## Key Concepts

- **Single Event Stream**: Instead of multiple event streams per aggregate, DCB uses a single event stream per bounded context
- **Tag-based Events**: Events are tagged when published, allowing one event to affect multiple entities/concepts
- **Dynamic Consistency**: Consistency is enforced through optimistic locking using the same query used for reading events
- **Flexible Boundaries**: No need for predefined aggregates or rigid transactional boundaries

## Features

- **Event Storage**: Store events with unique IDs, types, and JSON payloads
- **Tag-based Querying**: Query events using tags and event types to build different views of the same event stream
- **Optimistic Concurrency**: Built-in support for optimistic locking using event queries and positions
- **Event Causation**: Track event causation and correlation for event chains
- **State Reduction**: Build current state by reducing over events using custom reducers
- **Batch Operations**: Efficient batch operations for appending multiple events
- **PostgreSQL Backend**: Uses PostgreSQL for reliable, ACID-compliant storage
- **Go Native**: Written in Go with idiomatic Go patterns and interfaces

## Example Use Case

Consider a course subscription system where:
- A course cannot accept more than N students
- A student cannot subscribe to more than 10 courses

With traditional event sourcing, this would require:
1. Two separate event streams (for students and courses)
2. A saga to coordinate the subscription process
3. Two separate events for the same fact

With DCB, this becomes simpler:
1. A single event stream for the entire bounded context
2. One event tagged with both student and course identifiers
3. Consistency enforced through optimistic locking using the same query

```go
// Example: Subscribing a student to a course
event := dcb.NewInputEvent(
    "StudentSubscribedToCourse",
    dcb.NewTags(
        "student_id", "s1",
        "course_id", "c1",
    ),
    []byte(`{"timestamp": "2024-03-20T10:00:00Z"}`),
)

// The event affects both student and course entities
// Consistency is enforced through the query
query := dcb.NewQuery(
    dcb.NewTags("student_id", "s1", "course_id", "c1"),
    "StudentSubscribedToCourse",
)

position, err := store.AppendEvents(ctx, []dcb.InputEvent{event}, query, lastKnownPosition)
```

## Requirements

- Go 1.24 or later
- PostgreSQL 15 or later

## Installation

```bash
go get github.com/yourusername/go-crablet
```

## Quick Start

1. Set up PostgreSQL (using Docker):

```bash
docker-compose up -d
```

2. Use the event store in your Go code:

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "github.com/jackc/pgx/v5/pgxpool"
    "go-crablet/internal/dcb"
)

// CourseState represents the current state of a course
type CourseState struct {
    ID          string
    Name        string
    Capacity    int
    StudentIDs  map[string]bool // Set of enrolled student IDs
    IsActive    bool
}

// StudentState represents the current state of a student
type StudentState struct {
    ID          string
    Name        string
    CourseIDs   map[string]bool // Set of enrolled course IDs
    IsActive    bool
}

func main() {
    // Connect to PostgreSQL
    pool, err := pgxpool.New(context.Background(), "postgres://user:secret@localhost:5432/testdb?sslmode=disable")
    if err != nil {
        panic(err)
    }
    defer pool.Close()

    // Create event store
    store, err := dcb.NewEventStore(context.Background(), pool)
    if err != nil {
        panic(err)
    }
    defer store.Close()

    // Create some events
    events := []dcb.InputEvent{
        // Course creation event
        dcb.NewInputEvent(
            "CourseCreated",
            dcb.NewTags("course_id", "c1"),
            []byte(`{"name": "Go Programming", "capacity": 30}`),
        ),
        // Student registration event
        dcb.NewInputEvent(
            "StudentRegistered",
            dcb.NewTags("student_id", "s1"),
            []byte(`{"name": "John Doe", "email": "john@example.com"}`),
        ),
        // Course subscription event
        dcb.NewInputEvent(
            "StudentSubscribedToCourse",
            dcb.NewTags("course_id", "c1", "student_id", "s1"),
            []byte(`{"timestamp": "2024-03-20T10:00:00Z"}`),
        ),
    }

    // Append all events
    query := dcb.NewQuery(nil) // Empty query to match all events
    position, err := store.AppendEvents(context.Background(), events, query, 0)
    if err != nil {
        panic(err)
    }

    // Create a reducer for course state
    courseReducer := dcb.StateReducer{
        InitialState: &CourseState{
            StudentIDs: make(map[string]bool),
        },
        ReducerFn: func(state any, event dcb.Event) any {
            course := state.(*CourseState)
            
            switch event.Type {
            case "CourseCreated":
                var data struct {
                    Name     string `json:"name"`
                    Capacity int    `json:"capacity"`
                }
                json.Unmarshal(event.Data, &data)
                course.ID = event.Tags[0].Value // course_id tag
                course.Name = data.Name
                course.Capacity = data.Capacity
                course.IsActive = true
                
            case "StudentSubscribedToCourse":
                // Only process if this event is for our course
                for _, tag := range event.Tags {
                    if tag.Key == "course_id" && tag.Value == course.ID {
                        // Get student_id from tags
                        for _, t := range event.Tags {
                            if t.Key == "student_id" {
                                course.StudentIDs[t.Value] = true
                                break
                            }
                        }
                        break
                    }
                }
            }
            return course
        },
    }

    // Create a reducer for student state
    studentReducer := dcb.StateReducer{
        InitialState: &StudentState{
            CourseIDs: make(map[string]bool),
        },
        ReducerFn: func(state any, event dcb.Event) any {
            student := state.(*StudentState)
            
            switch event.Type {
            case "StudentRegistered":
                var data struct {
                    Name  string `json:"name"`
                    Email string `json:"email"`
                }
                json.Unmarshal(event.Data, &data)
                student.ID = event.Tags[0].Value // student_id tag
                student.Name = data.Name
                student.IsActive = true
                
            case "StudentSubscribedToCourse":
                // Only process if this event is for our student
                for _, tag := range event.Tags {
                    if tag.Key == "student_id" && tag.Value == student.ID {
                        // Get course_id from tags
                        for _, t := range event.Tags {
                            if t.Key == "course_id" {
                                student.CourseIDs[t.Value] = true
                                break
                            }
                        }
                        break
                    }
                }
            }
            return student
        },
    }

    // Read course state
    courseQuery := dcb.NewQuery(
        dcb.NewTags("course_id", "c1"),
    )
    _, courseState, err := store.ReadState(context.Background(), courseQuery, courseReducer)
    if err != nil {
        panic(err)
    }
    course := courseState.(*CourseState)
    fmt.Printf("Course %s has %d students enrolled\n", 
        course.Name, len(course.StudentIDs))

    // Read student state
    studentQuery := dcb.NewQuery(
        dcb.NewTags("student_id", "s1"),
    )
    _, studentState, err := store.ReadState(context.Background(), studentQuery, studentReducer)
    if err != nil {
        panic(err)
    }
    student := studentState.(*StudentState)
    fmt.Printf("Student %s is enrolled in %d courses\n", 
        student.Name, len(student.CourseIDs))

    // Check if course is at capacity
    if len(course.StudentIDs) >= course.Capacity {
        fmt.Printf("Course %s is at capacity\n", course.Name)
    }
}
```

This example demonstrates several key aspects of DCB:

1. **Single Event Stream**: All events (course creation, student registration, subscriptions) are stored in the same stream
2. **Tag-based Queries**: We can query the same events in different ways using tags
3. **Multiple Views**: The same events are used to build both course and student states
4. **Consistency**: The subscription event affects both course and student states atomically
5. **Business Rules**: We can enforce rules like course capacity by reducing over events

## API Documentation

### EventStore Interface

```go
type EventStore interface {
    // AppendEvents appends events to the store with optimistic concurrency control
    // using the provided query to enforce consistency
    AppendEvents(ctx context.Context, events []InputEvent, query Query, latestKnownPosition int64) (int64, error)
    
    // AppendEventsIfNotExists appends events only if they don't exist, using a reducer to check
    AppendEventsIfNotExists(ctx context.Context, events []InputEvent, query Query, latestKnownPosition int64, reducer StateReducer) (int64, error)
    
    // ReadState reads and reduces events to compute current state
    // using the same query that will be used for consistency checks
    ReadState(ctx context.Context, query Query, stateReducer StateReducer) (int64, any, error)
    
    // ReadStateUpTo reads and reduces events up to a specific position
    ReadStateUpTo(ctx context.Context, query Query, stateReducer StateReducer, maxPosition int64) (int64, any, error)
    
    // Close closes the event store connection
    Close()
}
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## References

- [DCB Official Website](https://dcb.events/)
- ["Killing the Aggregate" by Sara Pellegrini](https://dcb.events/)
