[![Go Report Card](https://goreportcard.com/badge/github.com/rodolfodpk/go-crablet)](https://goreportcard.com/report/github.com/rodolfodpk/go-crablet)
[![Code Coverage](https://img.shields.io/badge/code%20coverage-86.7%25-green?logo=go)](https://github.com/rodolfodpk/go-crablet/actions/workflows/coverage.yml)
[![GoDoc](https://godoc.org/github.com/rodolfodpk/go-crablet?status.svg)](https://godoc.org/github.com/rodolfodpk/go-crablet)
[![License](https://img.shields.io/github/license/rodolfodpk/go-crablet)](https://github.com/rodolfodpk/go-crablet/blob/main/LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/rodolfodpk/go-crablet)](https://github.com/rodolfodpk/go-crablet/blob/main/go.mod)

# go-crablet

A Go library for event sourcing, exploring and learning about concepts inspired by Sara Pellegrini's Dynamic Consistency Boundary (DCB) pattern. go-crablet enables you to build event-driven systems with:

## Key Features

- **DCB-inspired decision models**: Project multiple states and build append conditions in one step
- **Single streamlined query**: Efficiently project all relevant states using PostgreSQL's native streaming via pgx
- **Optimistic concurrency**: Append events only if no conflicting events have appeared within the same query combination scope
- **Memory-efficient streaming**: Process events row-by-row for large event streams
- **Flexible queries**: Tag-based, OR-combined queries for cross-entity boundaries
- **PostgreSQL-backed**: Uses PostgreSQL for robust, concurrent event storage

## Documentation
- [Overview](docs/overview.md): DCB pattern exploration, batch projection, and streaming
- [Minimal Example](docs/minimal-example.md): Detailed walkthrough of the course subscription example
- [Code Coverage](docs/code-coverage.md): Test coverage analysis and improvement guidelines

## Performance Benchmarks

Comprehensive performance testing and analysis for different API protocols:

- **[Web-App Benchmarks](internal/web-app/BENCHMARK.md)**: HTTP/REST API performance testing with k6
- **[gRPC App Benchmarks](internal/grpc-app/BENCHMARK.md)**: gRPC API performance testing with k6
- **[Go Benchmarks](internal/benchmarks/README.md)**: Core library performance testing and analysis

## Examples

Ready-to-run examples demonstrating different aspects of the DCB pattern:

- **[Transfer Example](internal/examples/transfer/main.go)**: **Batch append demonstration** - Money transfer between accounts with account creation and transfer in a single atomic batch operation
- **[Course Enrollment](internal/examples/enrollment/main.go)**: Student course enrollment with capacity limits and business rules
- **[Streaming Projections](internal/examples/streaming_projection/main.go)**: Memory-efficient event processing with multiple projections
- **[Decision Model](internal/examples/decision_model/main.go)**: Exploring Dynamic Consistency Boundary concepts with multiple projectors
- **[Cursor Streaming](internal/examples/cursor_streaming/main.go)**: Large dataset processing with batching and streaming
- **[ReadStream](internal/examples/readstream/main.go)**: Event streaming with projections and optimistic locking
- **[Batch Operations](internal/examples/batch/main.go)**: Batch event processing and atomic operations
- **[Channel Projection](internal/examples/channel_projection/main.go)**: Channel-based state projections
- **[Channel Streaming](internal/examples/channel_streaming/main.go)**: Channel-based event streaming
- **[Extension Interface](internal/examples/extension_interface/main.go)**: Extending the event store interface

Run any example with: `go run internal/examples/[example-name]/main.go`

## References

- [Dynamic Consistency Boundary (DCB)](https://dcb.events/) - A very good resource to understand the DCB pattern and its applications in event-driven systems
- [I am here to kill the aggregate](https://sara.event-thinking.io/2023/04/kill-aggregate-chapter-1-I-am-here-to-kill-the-aggregate.html) - Sara Pellegrini's blog post about moving beyond aggregates in event-driven systems
- [Kill Aggregate - Volume 2 - Sara Pellegrini at JOTB25](https://www.youtube.com/watch?v=AQ5fk4D3u9I)

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.