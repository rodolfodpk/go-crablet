# go-crablet

[![Go Report Card](https://goreportcard.com/badge/github.com/rodolfodpk/go-crablet)](https://goreportcard.com/report/github.com/rodolfodpk/go-crablet)
[![codecov](https://codecov.io/gh/rodolfodpk/go-crablet/branch/main/graph/badge.svg)](https://codecov.io/gh/rodolfodpk/go-crablet)
[![GoDoc](https://godoc.org/github.com/rodolfodpk/go-crablet?status.svg)](https://godoc.org/github.com/rodolfodpk/go-crablet)
[![License](https://img.shields.io/github/license/rodolfodpk/go-crablet)](https://github.com/rodolfodpk/go-crablet/blob/main/LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/rodolfodpk/go-crablet)](https://github.com/rodolfodpk/go-crablet/blob/main/go.mod)

A Go library implementing the Dynamic Consistency Boundary (DCB) event store pattern, providing a simpler and more flexible approach to consistency in event-driven systems. This library is perfect for event sourcing applications that need:
- Reliable audit trail of all state changes
- Flexible querying across event streams
- Easy state reconstruction at any point in time
- Optimistic concurrency control with consistency boundaries

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
- [Examples](docs/examples.md): Practical examples and use cases, including a complete course subscription system

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

## References

- [Dynamic Consistency Boundary (DCB)](https://dcb.events/) - The official website about the DCB pattern
- [I am here to kill the aggregate](https://sara.event-thinking.io/2023/04/kill-aggregate-chapter-1-I-am-here-to-kill-the-aggregate.html) - Sara Pellegrini's blog post about moving beyond aggregates in event-driven systems
- [Event Sourcing Pattern](https://martinfowler.com/eaaDev/EventSourcing.html) - Martin Fowler's comprehensive explanation of the event sourcing pattern
- [Event Sourcing Made Simple](https://www.eventstore.com/blog/event-sourcing-made-simple) - A practical guide to understanding and implementing event sourcing
