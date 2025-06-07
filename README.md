# go-crablet

[![Go Report Card](https://goreportcard.com/badge/github.com/rodolfodpk/go-crablet)](https://goreportcard.com/report/github.com/rodolfodpk/go-crablet)
[![codecov](https://codecov.io/gh/rodolfodpk/go-crablet/branch/main/graph/badge.svg)](https://codecov.io/gh/rodolfodpk/go-crablet)
[![GoDoc](https://godoc.org/github.com/rodolfodpk/go-crablet?status.svg)](https://godoc.org/github.com/rodolfodpk/go-crablet)
[![License](https://img.shields.io/github/license/rodolfodpk/go-crablet)](https://github.com/rodolfodpk/go-crablet/blob/main/LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/rodolfodpk/go-crablet)](https://github.com/rodolfodpk/go-crablet/blob/main/go.mod)

A Go implementation of the Dynamic Consistency Boundary (DCB) event store pattern, providing a simpler and more flexible approach to consistency in event-driven systems. Perfect for event sourcing applications that need:
- Reliable audit trail of all state changes
- Flexible querying across event streams
- Easy state reconstruction at any point in time
- Strong consistency guarantees
- Support for complex business workflows

Event sourcing is a pattern where all changes to application state are appended as a sequence of immutable events. Instead of updating the current state, you append new events that represent state changes. This append-only approach creates a complete, tamper-evident history that allows you to reconstruct past states, analyze how the system evolved, and build new views of the data without modifying the original event log.

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

## Features

- **Event Storage**: Append events with unique IDs, types, and JSON payloads
- **Consistency Boundaries**: Define and manage consistency boundaries for your events
- **State Projection**: PostgreSQL-streamed event projection for efficient state reconstruction
- **Flexible Querying**: Query events by type and tags to build different views of the same event stream
- **Concurrency Control**: Handle concurrent event appends with optimistic locking
- **Event Causation**: Track event causation and correlation for event chains
- **Batch Operations**: Efficient batch operations for appending multiple events
- **PostgreSQL Backend**: Uses PostgreSQL for reliable, ACID-compliant storage with native concurrency control
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

- [Dynamic Consistency Boundary (DCB)](https://dcb.events/) - The official website about the DCB pattern
- [Sara Pellegrini's Talk at DDD Europe 2024](https://dddeurope.com/2024/sara-pellegrini/) - Recent talk about DCB and its practical applications

### Event Store Interface

The event store provides a simple interface for managing events:

```go
// EventStore provides methods to append and read events in a PostgreSQL database.
// It implements the Dynamic Consistency Boundary pattern, ensuring that events
// within the same boundary are processed atomically and maintaining consistency
// through optimistic locking.
type EventStore interface {
    // AppendEvents adds multiple events to the stream and returns the latest position.
    // It ensures that no conflicting events have been appended since latestKnownPosition
    // for the given query, maintaining consistency boundaries.
    // Returns the new latest position or an error if the append fails.
    AppendEvents(ctx context.Context, events []InputEvent, query Query, latestKnownPosition int64) (int64, error)
    
    // ProjectState projects the current state using the provided projector.
    // It streams events from PostgreSQL that match the projector's query,
    // applying the transition function to build the current state.
    // Returns the latest position processed, the final state, and any error.
    ProjectState(ctx context.Context, projector StateProjector) (int64, any, error)
}

// Event represents a persisted event in the system
type Event struct {
    ID            string // Unique event identifier (UUID)
    Type          string // Event type (e.g., "Subscription")
    Tags          []Tag  // Tags for querying (e.g., {"course_id": "C1"})
    Data          []byte // Event payload
    Position      int64  // Position in the event stream
    CausationID   string // UUID of the event that caused this event
    CorrelationID string // UUID linking to the root event or process
}

// InputEvent represents an event to be appended to the store
type InputEvent struct {
    Type string // Event type (e.g., "Subscription")
    Tags []Tag  // Tags for querying (e.g., {"course_id": "C1"})
    Data []byte // JSON-encoded event payload
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
