# go-crablet

[![Go Report Card](https://goreportcard.com/badge/github.com/rodolfodpk/go-crablet)](https://goreportcard.com/report/github.com/rodolfodpk/go-crablet)
[![codecov](https://codecov.io/gh/rodolfodpk/go-crablet/branch/main/graph/badge.svg)](https://codecov.io/gh/rodolfodpk/go-crablet)
[![GoDoc](https://godoc.org/github.com/rodolfodpk/go-crablet?status.svg)](https://godoc.org/github.com/rodolfodpk/go-crablet)
[![License](https://img.shields.io/github/license/rodolfodpk/go-crablet)](https://github.com/rodolfodpk/go-crablet/blob/main/LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/rodolfodpk/go-crablet)](https://github.com/rodolfodpk/go-crablet/blob/main/go.mod)

A Go implementation of the Dynamic Consistency Boundary (DCB) event store pattern, providing a simpler and more flexible approach to consistency in event-driven systems.

## Requirements

- Go 1.24 or later
- PostgreSQL 15 or later

## Running Tests

To run all tests in the project, use:

```bash
go test -v ./...
```

This will run all tests in all packages, including integration tests that require a PostgreSQL database (which is automatically started using testcontainers).

## Installation

```bash
go get github.com/rodolfodpk/go-crablet
```

## Quick Start

1. Set up PostgreSQL (using Docker):

```bash
docker-compose up -d
```

## Overview

go-crablet is a Go library that implements the [Dynamic Consistency Boundary (DCB)](https://dcb.events/) pattern, introduced by Sara Pellegrini in her blog post "Killing the Aggregate". DCB provides a pragmatic approach to balancing strong consistency with flexibility in event-driven systems, without relying on rigid transactional boundaries.

Unlike traditional event sourcing approaches that use strict constraints to maintain immediate consistency, DCB allows for selective enforcement of strong consistency where needed, particularly for operations that span multiple entities. This ensures critical business processes and cross-entity invariants remain reliable while avoiding the constraints of traditional transactional models.

## Key Concepts

- **Single Event Stream**: While traditional event sourcing uses one stream per aggregate (e.g., one stream for Course aggregate, another for Student aggregate), DCB uses a single event stream per bounded context. You can still use aggregates if they make sense for your domain, but they're not required to enforce consistency
- **Tag-based Events**: Events are tagged with relevant identifiers, allowing one event to affect multiple concepts without artificial boundaries
- **Dynamic Consistency**: Consistency is enforced by checking if any events matching a query appeared after a known position. This ensures that events affecting the same concept are processed in order
- **Flexible Boundaries**: No need for predefined aggregates or rigid transactional boundaries - consistency boundaries emerge naturally from your queries, though you can still use aggregates where they provide value

The key difference from traditional event sourcing:

Traditional Event Sourcing | DCB Approach
-------------------------|------------
One stream per aggregate (required) | One stream per bounded context (aggregates optional)
Aggregates enforce consistency | Query-based position checks
Rigid aggregate boundaries | Dynamic query-based boundaries
Predefined consistency rules | Emergent consistency through queries

For example, in a course subscription system:

Traditional Approach | DCB Approach
-------------------|------------
Separate streams for `Course` and `Student` aggregates | Single stream with events tagged with both `course_id` and `student_id`
Saga to coordinate subscription | Single event with both tags
Two separate events for the same fact | One event affecting multiple concepts
Aggregate boundaries limit flexibility | Natural consistency through query-based position checks

## State Projection with PostgreSQL Streaming

go-crablet implements efficient state projection by leveraging PostgreSQL's streaming capabilities. Instead of loading all events into memory, events are streamed directly from the database and processed one at a time. This approach provides several benefits:

1. **Memory Efficiency**: Events are processed in a streaming fashion, making it suitable for large event streams
2. **Database Efficiency**: Uses PostgreSQL's native JSONB indexing and querying capabilities
3. **Consistent Views**: The same query used for consistency checks is used for state projection

Here's how it works under the hood:

```go
// The ProjectState method streams events from PostgreSQL
func (es *eventStore) ProjectState(ctx context.Context, query Query, stateProjector StateProjector) (int64, any, error) {
    // Handle empty or nil tags
    var queryTags []byte
    if len(query.Tags) > 0 {
        // Build JSONB query condition from tags
        tagMap := make(map[string]string)
        for _, t := range query.Tags {
            tagMap[t.Key] = t.Value
        }
        var err error
        queryTags, err = json.Marshal(tagMap)
        if err != nil {
            return 0, stateProjector.InitialState, fmt.Errorf("failed to marshal query tags: %w", err)
        }
    }

    // Construct SQL query
    var sqlQuery string
    var args []interface{}
    
    if queryTags != nil {
        // Use JSONB containment operator @> when tags are provided
        sqlQuery = "SELECT id, type, tags, data, position, causation_id, correlation_id FROM events WHERE tags @> $1"
        args = append(args, queryTags)
    } else {
        // When no tags are provided, select all events
        sqlQuery = "SELECT id, type, tags, data, position, causation_id, correlation_id FROM events"
    }

    // Add event type filtering if specified
    if len(query.EventTypes) > 0 {
        if len(args) > 0 {
            sqlQuery += fmt.Sprintf(" AND type = ANY($%d)", len(args)+1)
        } else {
            sqlQuery += fmt.Sprintf(" WHERE type = ANY($%d)", len(args)+1)
        }
        args = append(args, query.EventTypes)
    }

    // Stream rows from PostgreSQL
    rows, err := es.pool.Query(ctx, sqlQuery, args...)
    if err != nil {
        return 0, stateProjector.InitialState, fmt.Errorf("query failed: %w", err)
    }
    defer rows.Close()

    // Initialize state
    state := stateProjector.InitialState
    position := int64(0)

    // Process events one at a time
    for rows.Next() {
        var row rowEvent
        if err := rows.Scan(&row.ID, &row.Type, &row.Tags, &row.Data, &row.Position, &row.CausationID, &row.CorrelationID); err != nil {
            return 0, stateProjector.InitialState, fmt.Errorf("failed to scan row: %w", err)
        }

        // Convert row to Event
        event := convertRowToEvent(row)
        
        // Apply projector
        state = stateProjector.TransitionFn(state, event)
        position = row.Position
    }

    return position, state, nil
}
```

### Query Behavior

The `ProjectState` method provides flexible querying capabilities. Here are examples of how to use it:

1. **Querying All Events**:
   ```go
   // Create a query with no tags to match all events
   query := dcb.NewQuery(nil)
   
   // Project state using the query
   position, state, err := store.ProjectState(ctx, query, projector)
   if err != nil {
       panic(err)
   }
   ```

2. **Querying by Tags**:
   ```go
   // Query events for a specific course
   query := dcb.NewQuery(dcb.NewTags("course_id", "c1"))
   
   // Query events for a specific student and course
   query := dcb.NewQuery(dcb.NewTags(
       "student_id", "s1",
       "course_id", "c1",
   ))
   
   // Project state using the query
   position, state, err := store.ProjectState(ctx, query, projector)
   if err != nil {
       panic(err)
   }
   ```

3. **Querying by Event Type**:
   ```go
   // Query all subscription events
   query := dcb.NewQuery(nil, "StudentSubscribedToCourse")
   
   // Query subscription events for a specific course
   query := dcb.NewQuery(
       dcb.NewTags("course_id", "c1"),
       "StudentSubscribedToCourse",
   )
   
   // Project state using the query
   position, state, err := store.ProjectState(ctx, query, projector)
   if err != nil {
       panic(err)
   }
   ```

4. **Building Different Views**:
   ```go
   // Course view projector
   courseProjector := dcb.StateProjector{
       InitialState: &CourseState{
           StudentIDs: make(map[string]bool),
       },
       TransitionFn: func(state any, event dcb.Event) any {
           course := state.(*CourseState)
           // ... projector implementation ...
           return course
       },
   }

   // Student view projector
   studentProjector := dcb.StateProjector{
       InitialState: &StudentState{
           CourseIDs: make(map[string]bool),
       },
       TransitionFn: func(state any, event dcb.Event) any {
           student := state.(*StudentState)
           // ... projector implementation ...
           return student
       },
   }

   // Project course state
   courseQuery := dcb.NewQuery(dcb.NewTags("course_id", "c1"))
   _, courseState, err := store.ProjectState(ctx, courseQuery, courseProjector)
   if err != nil {
       panic(err)
   }

   // Project student state
   studentQuery := dcb.NewQuery(dcb.NewTags("student_id", "s1"))
   _, studentState, err := store.ProjectState(ctx, studentQuery, studentProjector)
   if err != nil {
       panic(err)
   }
   ```

The query behavior follows these rules:
- Empty or nil tags will match all events in the stream
- When tags are provided, only events containing all specified tags will be matched
- Event types can be combined with tags to further filter the events
- The same query used for projecting state is used for consistency checks when appending events

### Example: Building a Course Enrollment View

Here's how to efficiently build a course enrollment view using streaming state projection:

```go
// EnrollmentView represents the current state of course enrollments
type EnrollmentView struct {
    CourseEnrollments map[string]map[string]bool // course_id -> student_id -> enrolled
    StudentEnrollments map[string]map[string]bool // student_id -> course_id -> enrolled
}

// Create a projector for the enrollment view
enrollmentProjector := dcb.StateProjector{
    InitialState: &EnrollmentView{
        CourseEnrollments: make(map[string]map[string]bool),
        StudentEnrollments: make(map[string]map[string]bool),
    },
    TransitionFn: func(state any, event dcb.Event) any {
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

// Project the enrollment view
query := dcb.NewQuery(nil) // Match all subscription events
_, view, err := store.ProjectState(ctx, query, enrollmentProjector)
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
3. **Query Optimization**: The same query used for consistency checks is used for state projection, ensuring consistency
4. **Position-based Reading**: `ProjectStateUpTo` allows projecting state up to a specific position, which is useful for:
   - Building state at a point in time
   - Implementing event replay
   - Debugging by examining state at specific positions
   - Ensuring consistent reads up to a known position

Here's an example of using `ProjectStateUpTo`:

```go
// Project state up to a specific position
maxPosition := int64(1000) // Project only events up to position 1000
_, state, err := store.ProjectStateUpTo(ctx, query, projector, maxPosition)
if err != nil {
    panic(err)
}

// This is useful for scenarios like:
// 1. Replaying events up to a certain point
// 2. Debugging state at a specific time
// 3. Building state for a specific version
// 4. Ensuring consistent reads up to a known position
```

The key difference between `ProjectState` and `ProjectStateUpTo` is that `ProjectStateUpTo` allows you to limit the event stream to a specific position. This is particularly useful for:
- Debugging: You can examine state at any point in the event stream
- Replay: You can replay events up to a specific position
- Versioning: You can build state for a specific version of your data
- Consistency: You can ensure you're working with a consistent view up to a known position

Note: Events in the stream are always processed in order by their position. The position is automatically assigned when events are appended to the stream, ensuring a consistent and ordered sequence of events.

## Features

- **Event Storage**: Store events with unique IDs, types, and JSON payloads
- **Tag-based Querying**: Query events using tags and event types to build different views of the same event stream
- **Optimistic Concurrency**: Built-in support for optimistic locking using event queries and positions
- **Event Causation**: Track event causation and correlation for event chains
- **State Projection**: Build current state by projecting over events using custom projectors
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
3. Consistency enforced through position checks using the same query

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

## References

- [Dynamic Consistency Boundary (DCB)](https://dcb.events/) - The original blog post by Sara Pellegrini introducing the DCB pattern
- [Killing the Aggregate](https://dcb.events/killing-the-aggregate) - The blog post that inspired this implementation
- [PostgreSQL JSONB](https://www.postgresql.org/docs/current/datatype-json.html) - Documentation on PostgreSQL's JSONB type used for event tags
- [Event Sourcing Pattern](https://martinfowler.com/eaaDev/EventSourcing.html) - Martin Fowler's explanation of event sourcing
