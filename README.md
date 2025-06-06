# go-crablet

[![Go Report Card](https://goreportcard.com/badge/github.com/rodolfodpk/go-crablet)](https://goreportcard.com/report/github.com/rodolfodpk/go-crablet)
[![codecov](https://codecov.io/gh/rodolfodpk/go-crablet/branch/main/graph/badge.svg)](https://codecov.io/gh/rodolfodpk/go-crablet)
[![GoDoc](https://godoc.org/github.com/rodolfodpk/go-crablet?status.svg)](https://godoc.org/github.com/rodolfodpk/go-crablet)
[![License](https://img.shields.io/github/license/rodolfodpk/go-crablet)](https://github.com/rodolfodpk/go-crablet/blob/main/LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/rodolfodpk/go-crablet)](https://github.com/rodolfodpk/go-crablet/blob/main/go.mod)

A Go implementation of the Dynamic Consistency Boundary (DCB) event store pattern, providing a simpler and more flexible approach to consistency in event-driven systems.

## Documentation

The documentation has been split into several files for better organization:

- [Installation and Development Tools](docs/installation.md) - How to install and set up the development environment
- [Overview and Key Concepts](docs/overview.md) - Introduction to DCB and its key concepts
- [State Projection](docs/state-projection.md) - Details about state projection and PostgreSQL streaming
- [Examples](docs/examples.md) - Usage examples and patterns

## Quick Start

```bash
# Install the package
go get github.com/rodolfodpk/go-crablet

# See [Installation](docs/installation.md) for development setup
```

For more detailed information, please refer to the documentation sections above.

## Installation

```bash
go get github.com/rodolfodpk/go-crablet
```

## Development Tools

This project includes a Makefile to simplify common development tasks. Here are the available commands:

```bash
# Build the application
make build

# Run tests
make test

# Run tests with coverage report
make test-coverage

# Start Docker containers (PostgreSQL)
make docker-up

# Stop Docker containers
make docker-down

# Run linter
make lint

# Generate and serve documentation
make docs

# Clean build artifacts
make clean

# Show all available commands
make help
```

### Prerequisites

To use these commands, you'll need:
- Go 1.24 or later
- Docker and Docker Compose (required for both running PostgreSQL and running integration tests with testcontainers)
- golangci-lint (for the `make lint` command)

## Overview

go-crablet is a Go library that implements the [Dynamic Consistency Boundary (DCB)](https://dcb.events/) pattern, introduced by Sara Pellegrini in her blog post "Killing the Aggregate". DCB provides a pragmatic approach to balancing strong consistency with flexibility in event-driven systems, without relying on rigid transactional boundaries.

Unlike traditional event sourcing approaches that use strict constraints to maintain immediate consistency, DCB allows for selective enforcement of strong consistency where needed, particularly for operations that span multiple entities. This ensures critical business processes and cross-entity invariants remain reliable while avoiding the constraints of traditional transactional models.

The implementation leverages PostgreSQL's robust concurrency control mechanisms (MVCC and optimistic locking) to handle concurrent operations efficiently, while maintaining ACID guarantees at the database level.

## Key Concepts

- **Single Event Stream**: While traditional event sourcing uses one stream per aggregate (e.g., one stream for Course aggregate, another for Student aggregate), DCB uses a single event stream per bounded context. You can still use aggregates if they make sense for your domain, but they're not required to enforce consistency
- **Tag-based Events**: Events are tagged with relevant identifiers, allowing one event to affect multiple concepts without artificial boundaries
- **Dynamic Consistency**: Consistency is enforced by checking if any events matching a query appeared after a known position. This ensures that events affecting the same concept are processed in order
- **Flexible Boundaries**: No need for predefined aggregates or rigid transactional boundaries - consistency boundaries emerge naturally from your queries, though you can still use aggregates where they provide value
- **Concurrent Operations**: The implementation allows true concurrent operations by leveraging PostgreSQL's concurrency control mechanisms, rather than using application-level locks

The key difference from traditional event sourcing:

Traditional Event Sourcing | DCB Approach
-------------------------|------------
One stream per aggregate (required) | One stream per bounded context (aggregates optional)
Aggregates enforce consistency | Query-based position checks
Rigid aggregate boundaries | Dynamic query-based boundaries
Predefined consistency rules | Emergent consistency through queries
Application-level locking | Database-level concurrency control

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
func (es *eventStore) ProjectState(ctx context.Context, projector StateProjector) (int64, any, error) {
    // Handle empty or nil tags
    var queryTags []byte
    if len(projector.Tags) > 0 {
        // Build JSONB query condition from tags
        tagMap := make(map[string]string)
        for _, t := range projector.Tags {
            tagMap[t.Key] = t.Value
        }
        var err error
        queryTags, err = json.Marshal(tagMap)
        if err != nil {
            return 0, projector.InitialState, fmt.Errorf("failed to marshal query tags: %w", err)
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
    if len(projector.EventTypes) > 0 {
        if len(args) > 0 {
            sqlQuery += fmt.Sprintf(" AND type = ANY($%d)", len(args)+1)
        } else {
            sqlQuery += fmt.Sprintf(" WHERE type = ANY($%d)", len(args)+1)
        }
        args = append(args, projector.EventTypes)
    }

    // Stream rows from PostgreSQL
    rows, err := es.pool.Query(ctx, sqlQuery, args...)
    if err != nil {
        return 0, projector.InitialState, fmt.Errorf("query failed: %w", err)
    }
    defer rows.Close()

    // Initialize state
    state := projector.InitialState
    position := int64(0)

    // Process events one at a time
    for rows.Next() {
        var row rowEvent
        if err := rows.Scan(&row.ID, &row.Type, &row.Tags, &row.Data, &row.Position, &row.CausationID, &row.CorrelationID); err != nil {
            return 0, projector.InitialState, fmt.Errorf("failed to scan row: %w", err)
        }

        // Convert row to Event
        event := convertRowToEvent(row)
        
        // Apply projector
        state = projector.TransitionFn(state, event)
        position = row.Position
    }

    return position, state, nil
}
```

### Query Behavior

The `ProjectState` method provides flexible state projection capabilities. Here are examples of how to use it:

1. **Projecting All Events**:
   ```go
   // Create a projector that handles all events
   projector := dcb.StateProjector{
       Query: dcb.NewQuery(nil), // Empty query matches all events
       InitialState: &MyState{},
       TransitionFn: func(state any, event dcb.Event) any {
           // Handle all events
           return state
       },
   }
   
   // Project state using the projector
   position, state, err := store.ProjectState(ctx, projector)
   if err != nil {
       panic(err)
   }
   ```

2. **Projecting Specific Event Types**:
   ```go
   // Create a projector that handles specific event types
   projector := dcb.StateProjector{
       Query: dcb.NewQuery(nil, "StudentSubscribedToCourse", "StudentUnsubscribedFromCourse"),
       InitialState: &SubscriptionState{},
       TransitionFn: func(state any, event dcb.Event) any {
           // Only subscription events will be received due to Query.EventTypes
           switch event.Type {
           case "StudentSubscribedToCourse":
               // Handle subscription event
           case "StudentUnsubscribedFromCourse":
               // Handle unsubscription event
           }
           return state
       },
   }
   
   // Project state using the projector
   position, state, err := store.ProjectState(ctx, projector)
   if err != nil {
       panic(err)
   }
   ```

3. **Building Different Views**:
   ```go
   // Course view projector
   courseProjector := dcb.StateProjector{
       Query: dcb.NewQuery(dcb.NewTags("course_id", "c1")), // Filter by course_id at database level
       InitialState: &CourseState{
           StudentIDs: make(map[string]bool),
       },
       TransitionFn: func(state any, event dcb.Event) any {
           course := state.(*CourseState)
           // Only events for course c1 will be received due to Query.Tags
           switch event.Type {
           case "StudentSubscribedToCourse":
               // Add student to course
           case "StudentUnsubscribedFromCourse":
               // Remove student from course
           }
           return course
       },
   }

   // Student view projector
   studentProjector := dcb.StateProjector{
       Query: dcb.NewQuery(dcb.NewTags("student_id", "s1")), // Filter by student_id at database level
       InitialState: &StudentState{
           CourseIDs: make(map[string]bool),
       },
       TransitionFn: func(state any, event dcb.Event) any {
           student := state.(*StudentState)
           // Only events for student s1 will be received due to Query.Tags
           switch event.Type {
           case "StudentSubscribedToCourse":
               // Add course to student
           case "StudentUnsubscribedFromCourse":
               // Remove course from student
           }
           return student
       },
   }

   // Project course state
   _, courseState, err := store.ProjectState(ctx, courseProjector)
   if err != nil {
       panic(err)
   }

   // Project student state
   _, studentState, err := store.ProjectState(ctx, studentProjector)
   if err != nil {
       panic(err)
   }
   ```

The projector behavior follows these rules:
- The projector's `Query` field filters events at the database level for better performance
- The projector's `TransitionFn` receives only the events that match the query
- The projector can combine tag and event type filtering in its query
- The projector maintains its own state and can handle any event type it needs to

### Generic Example: Filtering by Tag and Event Type

Suppose you want to project state only for events related to a specific course and only for certain event types. Here's a minimal example:

```go
// Define a simple state type to track course activity
type CourseActivity struct {
    Subscriptions int
    Unsubscriptions int
}

// Append some events (pseudo-code, assumes you have a store and context)
courseTags := dcb.NewTags("course_id", "C123")
otherCourseTags := dcb.NewTags("course_id", "C999")

// These events would be appended to the store (see your event appending API)
events := []dcb.InputEvent{
    dcb.NewInputEvent("StudentSubscribedToCourse", courseTags, nil),    // Should be counted
    dcb.NewInputEvent("StudentUnsubscribedFromCourse", courseTags, nil), // Should be counted
    dcb.NewInputEvent("CourseCancelled", courseTags, nil),              // Should NOT be counted
    dcb.NewInputEvent("StudentSubscribedToCourse", otherCourseTags, nil), // Should NOT be counted
}
// _ = store.AppendEvents(ctx, events, dcb.NewQuery(nil), 0) // (pseudo-code)

// Create a projector that only counts subscription events for course_id=C123
projector := dcb.StateProjector{
    Query: dcb.NewQuery(
        dcb.NewTags("course_id", "C123"), // Only events with this tag
        "StudentSubscribedToCourse", "StudentUnsubscribedFromCourse", // Only these event types
    ),
    InitialState: &CourseActivity{},
    TransitionFn: func(state any, event dcb.Event) any {
        c := state.(*CourseActivity)
        switch event.Type {
        case "StudentSubscribedToCourse":
            c.Subscriptions++
        case "StudentUnsubscribedFromCourse":
            c.Unsubscriptions++
        }
        return c
    },
}

// Project the state
_, state, err := store.ProjectState(ctx, projector)
if err != nil {
    panic(err)
}
activity := state.(*CourseActivity)
fmt.Printf("Subscriptions: %d, Unsubscriptions: %d\n", activity.Subscriptions, activity.Unsubscriptions)
// Output: Subscriptions: 1, Unsubscriptions: 1
```

## Features

- **Event Storage**: Store events with unique IDs, types, and JSON payloads
- **Tag-based Querying**: Query events using tags and event types to build different views of the same event stream
- **Optimistic Concurrency**: Built-in support for optimistic locking using event queries and positions
- **Event Causation**: Track event causation and correlation for event chains
- **State Projection**: Build current state by projecting over events using custom projectors
- **Batch Operations**: Efficient batch operations for appending multiple events
- **PostgreSQL Backend**: Uses PostgreSQL for reliable, ACID-compliant storage with native concurrency control
- **Go Native**: Written in Go with idiomatic Go patterns and interfaces
- **Resource Management**: Clean separation of concerns with database pool lifecycle managed by the caller

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

- [Dynamic Consistency Boundary (DCB)](https://dcb.events/) - The official website about the DCB pattern
- [Sara Pellegrini's Talk at DDD Europe 2024](https://dddeurope.com/2024/sara-pellegrini/) - Recent talk about DCB and its practical applications

### Event Store Interface

The event store provides a simple interface for managing events:

```go
type EventStore interface {
    // AppendEvents adds multiple events to the stream and returns the latest position
    AppendEvents(ctx context.Context, events []Event) (int64, error)
    
    // ProjectState projects the current state using the provided projector
    ProjectState(ctx context.Context, projector StateProjector) (int64, any, error)
}

// StateProjector defines how to project state from events
type StateProjector struct {
    // Query defines criteria for selecting events at the database level
    Query Query
    
    // InitialState is the starting state for the projection
    InitialState any
    
    // TransitionFn defines how to update state for each event
    TransitionFn func(state any, event Event) any
}

// Query defines criteria for selecting events
type Query struct {
    // Tags must match all specified tags (empty means match any tag)
    Tags []Tag
    
    // EventTypes must match one of these types (empty means match any type)
    EventTypes []string
}

// Tag is a key-value pair for querying events
type Tag struct {
    Key   string
    Value string
}
```
