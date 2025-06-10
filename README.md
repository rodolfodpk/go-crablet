# go-crablet

[![Go Report Card](https://goreportcard.com/badge/github.com/rodolfodpk/go-crablet)](https://goreportcard.com/report/github.com/rodolfodpk/go-crablet)
[![codecov](https://codecov.io/gh/rodolfodpk/go-crablet/branch/main/graph/badge.svg)](https://codecov.io/gh/rodolfodpk/go-crablet)
[![GoDoc](https://godoc.org/github.com/rodolfodpk/go-crablet?status.svg)](https://godoc.org/github.com/rodolfodpk/go-crablet)
[![License](https://img.shields.io/github/license/rodolfodpk/go-crablet)](https://github.com/rodolfodpk/go-crablet/blob/main/LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/rodolfodpk/go-crablet)](https://github.com/rodolfodpk/go-crablet/blob/main/go.mod)

A Go library inspired by Sara Pellegrini's Dynamic Consistency Boundary (DCB) pattern, providing a simpler and more flexible approach to consistency in event-driven systems. This library aims to help with event sourcing applications that need:
- Reliable audit trail of all state changes
- Flexible querying across event streams
- Easy state reconstruction at any point in time
- Optimistic concurrency control with consistency boundaries
- **Streaming event processing** for memory-efficient operations
- **Complex query support** with multiple query items and OR logic

Event sourcing is a pattern where all changes to application state are appended as a sequence of immutable events. Instead of updating the current state, you append new events that represent state changes. This append-only approach creates a complete, tamper-evident history that allows you to reconstruct past states, analyze how the system evolved, and build new views of the data without modifying the original event log.

The library provides a focused, single-responsibility component that can be easily integrated into any Go application. It gives you full control over your event structure and state management while handling the complexities of event storage, consistency boundaries, and state projection.

## Documentation

The documentation has been split into several files for better organization:

- [Overview](docs/overview.md): High-level overview of go-crablet
- [Installation](docs/installation.md): Installation and setup guide
- [Tutorial](docs/tutorial.md): Step-by-step guide to get started with go-crablet
- [Implementation Details](docs/implementation.md): Detailed technical documentation about the implementation
- [State Projection](docs/state-projection.md): Detailed guide on state projection
- [Appending Events](docs/appending-events.md): Guide on appending events and handling concurrency
- [Reading Events](docs/reading-events.md): Guide on streaming events efficiently
- [Examples](docs/examples.md): Practical examples and use cases, including a complete course subscription system

## Features

- **Event Sourcing**: Append-only event store with optimistic locking
- **Complex Queries**: Support for multiple query items with OR logic
- **Streaming Events**: Memory-efficient event iteration with keyset pagination
- **State Projection**: Real-time state reconstruction from event streams
- **Optimistic Locking**: Robust concurrent event appending with conflict detection
- **DCB Compliance**: Inspired by and aims to follow the Database Change Broker pattern

## Streaming & Memory Efficiency

Both `ReadEvents` and `ProjectState` are designed for **memory-efficient streaming** of large event datasets:

### **ReadEvents - Application-Level Streaming**
```go
// Stream events in configurable batches (default: 1000)
iterator, err := store.ReadEvents(ctx, query, nil)
defer iterator.Close()

for {
    event, err := iterator.Next()
    if err != nil {
        return err
    }
    if event == nil {
        break // No more events
    }
    // Process single event
    processEvent(event)
}
```

**Benefits:**
- **Constant Memory Usage**: Only configurable batch size in memory at any time
- **Keyset Pagination**: Efficient `position > lastPosition` queries
- **Scalable**: Handles millions of events without memory issues
- **Configurable**: Adjust batch size via `ReadOptions.BatchSize`

### **ProjectState - Database-Level Streaming**
```go
// Stream events directly from database
state, position, err := store.ProjectState(ctx, projector)
```

**Benefits:**
- **Native PostgreSQL Streaming**: Uses `rows.Next()` for row-by-row processing
- **Zero Accumulation**: Processes one event at a time, no intermediate storage
- **Real-time State**: Incremental state updates as events are processed
- **Memory Efficient**: Constant memory usage regardless of event count

### **Memory Usage Comparison**

| Method | Memory Pattern | Use Case |
|--------|---------------|----------|
| `ReadEvents` | Configurable batch size (default: 1000) | Event iteration, custom processing |
| `ProjectState` | 1 event at a time | State reconstruction, projections |

Both approaches ensure **O(1) memory complexity** even with millions of events.

## Quick Start

```go
package main

import (
    "context"
    "log"
    
    "github.com/rodolfodpk/go-crablet/pkg/dcb"
    "github.com/jackc/pgx/v5/pgxpool"
)

func main() {
    // Create database connection
    pool, err := pgxpool.New(context.Background(), "postgres://user:pass@localhost/db")
    if err != nil {
        log.Fatal(err)
    }
    
    // Create event store
    store, err := dcb.NewEventStore(context.Background(), pool)
    if err != nil {
        log.Fatal(err)
    }
    
    // Read events with complex query
    query := dcb.NewQueryFromItems(
        dcb.NewQueryItem([]string{"AccountRegistered"}, []dcb.Tag{{Key: "user_id", Value: "123"}}),
        dcb.NewQueryItem([]string{"AccountDetailsChanged"}, []dcb.Tag{{Key: "account_id", Value: "456"}}),
    )
    
    // Read events using streaming interface
    iterator, err := store.ReadEvents(context.Background(), query, nil)
    if err != nil {
        log.Fatal(err)
    }
    defer iterator.Close()
    
    // Process events one by one
    for {
        event, err := iterator.Next()
        if err != nil {
            log.Fatal(err)
        }
        if event == nil {
            break // No more events
        }
        
        log.Printf("Event: %s at position %d", event.Type, event.Position)
    }
}
```

## References

- [Dynamic Consistency Boundary (DCB)](https://dcb.events/) - A very good resource to understand the DCB pattern and its applications in event-driven systems
- [I am here to kill the aggregate](https://sara.event-thinking.io/2023/04/kill-aggregate-chapter-1-I-am-here-to-kill-the-aggregate.html) - Sara Pellegrini's blog post about moving beyond aggregates in event-driven systems
- [Kill Aggregate - Volume 2 - Sara Pellegrini at JOTB25](https://www.youtube.com/watch?v=AQ5fk4D3u9I)
