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
- Optimistic concurrency control with consistency boundaries

Event sourcing is a pattern where all changes to application state are appended as a sequence of immutable events. Instead of updating the current state, you append new events that represent state changes. This append-only approach creates a complete, tamper-evident history that allows you to reconstruct past states, analyze how the system evolved, and build new views of the data without modifying the original event log.

## Documentation

The documentation has been split into several files for better organization:

- [Overview](docs/overview.md): High-level overview of go-crablet
- [Installation](docs/installation.md): Installation and setup guide
- [Tutorial](docs/tutorial.md): Step-by-step guide to get started with go-crablet
- [State Projection](docs/state-projection.md): Detailed guide on state projection
- [Examples](docs/examples.md): Practical examples and use cases

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
- **PostgreSQL Backend**: Uses PostgreSQL for reliable, ACID-compliant storage with optimistic concurrency control
- **Go Native**: Written in Go with idiomatic Go patterns and interfaces
- **Stream Position Management**: Automatic handling of stream positions to ensure event ordering and prevent race conditions. All event appends use the current stream position for optimistic concurrency control, maintaining consistency across concurrent operations.

### Stream Position Handling

When appending events to the store, it's crucial to use the current stream position rather than a fixed position (like 0). This ensures:

1. **Event Ordering**: Events are processed in the correct sequence
2. **Race Condition Prevention**: Concurrent updates are handled safely
3. **Consistency**: The final state reflects the most recent update

Example of proper stream position handling:

```go
// Get current stream position
query := dcb.NewQuery(dcb.NewTags("account_id", "acc123"))
position, err := store.GetCurrentPosition(ctx, query)
if err != nil {
    return err
}

// Append events using the current position
events := []dcb.InputEvent{
    {
        Type: "AccountBalanceUpdated",
        Tags: dcb.NewTags("account_id", "acc123"),
        Data: []byte(`{"balance": 1000}`),
    },
}
newPosition, err := store.AppendEvents(ctx, events, query, position)
```

### Event Store Interface

The core interface for event management:

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

### Example Use Case: Course Subscription System

Here's a practical example of using go-crablet to implement a course subscription system:

```go
// Define event types
const (
    EventTypeSubscription = "Subscription"
    EventTypeUnsubscription = "Unsubscription"
)

// Define subscription state
type SubscriptionState struct {
    IsActive bool
    Since    time.Time
    Until    time.Time
}

// Create a projector for subscription state
subscriptionProjector := dcb.StateProjector{
    Query: dcb.NewQuery(
        dcb.NewTags("course_id", "C1", "student_id", "S1"),
        EventTypeSubscription,
        EventTypeUnsubscription,
    ),
    InitialState: &SubscriptionState{},
    TransitionFn: func(state any, event dcb.Event) any {
        s := state.(*SubscriptionState)
        switch event.Type {
        case EventTypeSubscription:
            var data struct{ Until time.Time }
            if err := json.Unmarshal(event.Data, &data); err != nil {
                panic(err)
            }
            s.IsActive = true
            s.Since = time.Now()
            s.Until = data.Until
        case EventTypeUnsubscription:
            s.IsActive = false
        }
        return s
    },
}

// Append a subscription event
events := []dcb.InputEvent{
    {
        Type: EventTypeSubscription,
        Tags: dcb.NewTags("course_id", "C1", "student_id", "S1"),
        Data: []byte(`{"until": "2024-12-31T23:59:59Z"}`),
    },
}

// Use a query to define the consistency boundary
query := dcb.NewQuery(
    dcb.NewTags("course_id", "C1", "student_id", "S1"),
    EventTypeSubscription,
    EventTypeUnsubscription,
)

// Append events with optimistic locking
position, err := store.AppendEvents(ctx, events, query, 0)
if err != nil {
    panic(err)
}

// Project current state
_, state, err := store.ProjectState(ctx, subscriptionProjector)
if err != nil {
    panic(err)
}

subscription := state.(*SubscriptionState)
fmt.Printf("Subscription active: %v, until: %v\n", 
    subscription.IsActive, 
    subscription.Until,
)
```

### State Projection

go-crablet implements efficient state projection by leveraging PostgreSQL's streaming capabilities. Instead of loading all events into memory, events are streamed directly from the database and processed one at a time. This approach provides several benefits:

1. **Memory Efficiency**: Events are processed in a streaming fashion, making it suitable for large event streams
2. **Database Efficiency**: Uses PostgreSQL's native JSONB indexing and querying capabilities
3. **Consistent Views**: The same query used for consistency checks is used for state projection

Example of state projection:

```go
// Create a projector for account balances
projector := dcb.StateProjector{
    Query: dcb.NewQuery(dcb.NewTags("account_id", "acc123")),
    InitialState: &AccountState{},
    TransitionFn: func(state any, event dcb.Event) any {
        // Handle events and update state
        return state
    },
}

// Project the current state
position, state, err := store.ProjectState(ctx, projector)
```

### Appending Events

go-crablet provides a robust mechanism for appending events with optimistic concurrency control. This ensures:

1. **Event Ordering**: Events are processed in the correct sequence
2. **Race Condition Prevention**: Concurrent updates are handled safely
3. **Consistency**: The final state reflects the most recent update

Example of appending events:

```go
// Get current stream position
query := dcb.NewQuery(dcb.NewTags("account_id", "acc123"))
position, err := store.GetCurrentPosition(ctx, query)
if err != nil {
    return err
}

// Create and append events
events := []dcb.InputEvent{
    {
        Type: "AccountBalanceUpdated",
        Tags: dcb.NewTags("account_id", "acc123"),
        Data: []byte(`{"balance": 1000}`),
    },
}

// Append events using the current position
newPosition, err := store.AppendEvents(ctx, events, query, position)
if err != nil {
    // Handle error - might be due to concurrent modification
    return err
}
```

The event store automatically handles optimistic concurrency control by:
1. Checking if the provided position matches the current stream position
2. Rejecting the append if there are concurrent modifications
3. Updating the stream position atomically with the event append

## References

- [Dynamic Consistency Boundary (DCB)](https://dcb.events/) - The official website about the DCB pattern
- [Sara Pellegrini's Talk at DDD Europe 2024](https://dddeurope.com/2024/sara-pellegrini/) - Recent talk about DCB and its practical applications
