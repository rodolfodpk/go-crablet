[![Go Report Card](https://goreportcard.com/badge/github.com/rodolfodpk/go-crablet)](https://goreportcard.com/report/github.com/rodolfodpk/go-crablet)
[![Code Coverage](https://img.shields.io/badge/code%20coverage-82.1%25-green?logo=go)](https://github.com/rodolfodpk/go-crablet/actions/workflows/coverage.yml)
[![GoDoc](https://godoc.org/github.com/rodolfodpk/go-crablet?status.svg)](https://godoc.org/github.com/rodolfodpk/go-crablet)
[![License](https://img.shields.io/github/license/rodolfodpk/go-crablet)](https://github.com/rodolfodpk/go-crablet/blob/main/LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/rodolfodpk/go-crablet)](https://github.com/rodolfodpk/go-crablet/blob/main/go.mod)

# go-crablet

A Go library for event sourcing, exploring and learning about concepts inspired by Sara Pellegrini's Dynamic Consistency Boundary (DCB) pattern. 

## Key Features

- **DCB-inspired decision models**: Project multiple states and check business invariants in a single query
- **Optimistic concurrency**: Append events only if no conflicting events exist within the same query scope
- **Memory-efficient streaming**: Process events row-by-row for large event streams
- **PostgreSQL-backed**: Uses PostgreSQL for robust, concurrent event storage

## Documentation
- [Overview](docs/overview.md): DCB pattern exploration, batch projection, and streaming
- [Quick Start](docs/quick-start.md): Get started using go-crablet in your project
- [Getting Started](docs/getting-started.md): Development setup
- [Code Coverage](docs/code-coverage.md): Test coverage analysis and improvement guidelines

## Performance Benchmarks

Comprehensive performance testing and analysis for different API protocols:

- **[Web-App Benchmarks](internal/web-app/BENCHMARK.md)**: HTTP/REST API performance testing with k6
- **[Go Benchmarks](internal/benchmarks/README.md)**: Core library performance testing and analysis

## Examples

Ready-to-run examples demonstrating different aspects of the DCB pattern:

- **[Transfer Example](internal/examples/transfer/main.go)**: **Account transfer demonstration** - Money transfer between accounts with account creation and transfer in a single atomic batch operation
- **[Course Enrollment](internal/examples/enrollment/main.go)**: **Course subscription demonstration** - Student course enrollment with capacity limits and business rules
- **[Streaming](internal/examples/streaming/main.go)**: **Streaming approaches** - Demonstrates core EventStore reading, channel-based streaming, and channel-based projection
- **[Decision Model](internal/examples/decision_model/main.go)**: **DCB decision model** - Exploring Dynamic Consistency Boundary concepts with multiple projectors
- **[Batch Operations](internal/examples/batch/main.go)**: **Batch processing** - Batch event processing and atomic operations

Run any example with: `go run internal/examples/[example-name]/main.go`

## References

- [Dynamic Consistency Boundary (DCB)](https://dcb.events/) - A very good resource to understand the DCB pattern and its applications in event-driven systems
- [I am here to kill the aggregate](https://sara.event-thinking.io/2023/04/kill-aggregate-chapter-1-I-am-here-to-kill-the-aggregate.html) - Sara Pellegrini's blog post about moving beyond aggregates in event-driven systems
- [Kill Aggregate - Volume 2 - Sara Pellegrini at JOTB25](https://www.youtube.com/watch?v=AQ5fk4D3u9I)

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.