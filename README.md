# go-crablet

[![Go Report Card](https://goreportcard.com/badge/github.com/rodolfodpk/go-crablet)](https://goreportcard.com/report/github.com/rodolfodpk/go-crablet)
[![codecov](https://codecov.io/gh/rodolfodpk/go-crablet/branch/main/graph/badge.svg)](https://codecov.io/gh/rodolfodpk/go-crablet)
[![GoDoc](https://godoc.org/github.com/rodolfodpk/go-crablet?status.svg)](https://godoc.org/github.com/rodolfodpk/go-crablet)
[![License](https://img.shields.io/github/license/rodolfodpk/go-crablet)](https://github.com/rodolfodpk/go-crablet/blob/main/LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/rodolfodpk/go-crablet)](https://github.com/rodolfodpk/go-crablet/blob/main/go.mod)

A Go implementation of the Dynamic Consistency Boundary (DCB) event store pattern, providing a simpler and more flexible approach to consistency in event-driven systems.

## Overview

go-crablet is a Go library that implements the [Dynamic Consistency Boundary (DCB)](https://dcb.events/) pattern, introduced by Sara Pellegrini in her blog post "Killing the Aggregate". DCB provides a pragmatic approach to balancing strong consistency with flexibility in event-driven systems, without relying on rigid transactional boundaries.

Unlike traditional event sourcing approaches that use strict constraints to maintain immediate consistency, DCB allows for selective enforcement of strong consistency where needed, particularly for operations that span multiple entities. This ensures critical business processes and cross-entity invariants remain reliable while avoiding the constraints of traditional transactional models.

## Key Concepts

- **Single Event Stream**: Instead of multiple event streams per aggregate, DCB uses a single event stream per bounded context
- **Tag-based Events**: Events are tagged when published, allowing one event to affect multiple entities/concepts
- **Dynamic Consistency**: Consistency is enforced through optimistic locking using the same query used for reading events
- **Flexible Boundaries**: No need for predefined aggregates or rigid transactional boundaries

## State Reduction with PostgreSQL Streaming

go-crablet implements efficient state reduction by leveraging PostgreSQL's streaming capabilities. Instead of loading all events into memory, events are streamed directly from the database and processed one at a time. This approach provides several benefits:

1. **Memory Efficiency**: Events are processed in a streaming fashion, making it suitable for large event streams
2. **Database Efficiency**: Uses PostgreSQL's native JSONB indexing and querying capabilities
3. **Consistent Views**: The same query used for consistency checks is used for state reduction

Here's how it works under the hood:

```go
// The ReadState method streams events from PostgreSQL
func (es *eventStore) ReadState(ctx context.Context, query Query, stateReducer StateReducer) (int64, any, error) {
    // Build JSONB query condition from tags
    tagMap := make(map[string]string)
    for _, t := range query.Tags {
        tagMap[t.Key] = t.Value
    }
    queryTags, err := json.Marshal(tagMap)
    if err != nil {
        return 0, stateReducer.InitialState, fmt.Errorf("failed to marshal query tags: %w", err)
    }

    // Construct SQL query using PostgreSQL's JSONB containment operator @>
    sqlQuery := "SELECT id, type, tags, data, position, causation_id, correlation_id FROM events WHERE tags @> $1"
    args := []interface{}{queryTags}

    // Add event type filtering if specified
    if len(query.EventTypes) > 0 {
        sqlQuery += fmt.Sprintf(" AND type = ANY($%d)", len(args)+1)
        args = append(args, query.EventTypes)
    }

    // Stream rows from PostgreSQL
    rows, err := es.pool.Query(ctx, sqlQuery, args...)
    if err != nil {
        return 0, stateReducer.InitialState, fmt.Errorf("query failed: %w", err)
    }
    defer rows.Close()

    // Initialize state
    state := stateReducer.InitialState
    position := int64(0)

    // Process events one at a time
    for rows.Next() {
        var row rowEvent
        if err := rows.Scan(&row.ID, &row.Type, &row.Tags, &row.Data, &row.Position, &row.CausationID, &row.CorrelationID); err != nil {
            return 0, stateReducer.InitialState, fmt.Errorf("failed to scan row: %w", err)
        }

        // Convert row to Event
        event := convertRowToEvent(row)
        
        // Apply reducer
        state = stateReducer.ReducerFn(state, event)
        position = row.Position
    }

    return position, state, nil
}
```

### Example: Building a Course Enrollment View

Here's how to efficiently build a course enrollment view using streaming state reduction:

```go
// EnrollmentView represents the current state of course enrollments
type EnrollmentView struct {
    CourseEnrollments map[string]map[string]bool // course_id -> student_id -> enrolled
    StudentEnrollments map[string]map[string]bool // student_id -> course_id -> enrolled
}

// Create a reducer for the enrollment view
enrollmentReducer := dcb.StateReducer{
    InitialState: &EnrollmentView{
        CourseEnrollments: make(map[string]map[string]bool),
        StudentEnrollments: make(map[string]map[string]bool),
    },
    ReducerFn: func(state any, event dcb.Event) any {
        view := state.(*EnrollmentView)
        
        switch event.Type {
        case "StudentSubscribedToCourse":
            var courseID, studentID string
            // Extract IDs from tags
            for _, tag := range event.Tags {
                switch tag.Key {
                case "course_id":
                    courseID = tag.Value
                case "student_id":
                    studentID = tag.Value
                }
            }
            
            // Update course enrollments
            if _, exists := view.CourseEnrollments[courseID]; !exists {
                view.CourseEnrollments[courseID] = make(map[string]bool)
            }
            view.CourseEnrollments[courseID][studentID] = true
            
            // Update student enrollments
            if _, exists := view.StudentEnrollments[studentID]; !exists {
                view.StudentEnrollments[studentID] = make(map[string]bool)
            }
            view.StudentEnrollments[studentID][courseID] = true
            
        case "StudentUnsubscribedFromCourse":
            // Similar logic for unsubscription
            // ...
        }
        return view
    },
}

// Read the enrollment view
query := dcb.NewQuery(nil) // Match all subscription events
_, view, err := store.ReadState(ctx, query, enrollmentReducer)
if err != nil {
    panic(err)
}

enrollmentView := view.(*EnrollmentView)
fmt.Printf("Total courses with enrollments: %d\n", len(enrollmentView.CourseEnrollments))
fmt.Printf("Total students with enrollments: %d\n", len(enrollmentView.StudentEnrollments))
```

### Performance Considerations

1. **Indexing**: The implementation uses PostgreSQL's GIN indexes on the `tags` JSONB column for efficient querying
2. **Streaming**: Events are processed one at a time, keeping memory usage constant regardless of event stream size
3. **Query Optimization**: The same query used for consistency checks is used for state reduction, ensuring consistency
4. **Position-based Reading**: `ReadStateUpTo` allows reading events up to a specific position, which is useful for:
   - Building state at a point in time
   - Implementing event replay
   - Handling out-of-order event processing
   - Debugging by examining state at specific positions

Here's an example of using `ReadStateUpTo`:

```go
// Read state up to a specific position
maxPosition := int64(1000) // Read only events up to position 1000
_, state, err := store.ReadStateUpTo(ctx, query, reducer, maxPosition)
if err != nil {
    panic(err)
}

// This is useful for scenarios like:
// 1. Replaying events up to a certain point
// 2. Debugging state at a specific time
// 3. Building state for a specific version
// 4. Handling out-of-order event processing
```

The key difference between `ReadState` and `ReadStateUpTo` is that `ReadStateUpTo` allows you to limit the event stream to a specific position, which is particularly useful for:
- Debugging: You can examine state at any point in the event stream
- Replay: You can replay events up to a specific position
- Versioning: You can build state for a specific version of your data
- Consistency: You can ensure you're working with a consistent view up to a known position

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
go get github.com/rodolfodpk/go-crablet
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
- [GitHub Repository](https://github.com/rodolfodpk/go-crablet)
