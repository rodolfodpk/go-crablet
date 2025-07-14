[![Go Report Card](https://goreportcard.com/badge/github.com/rodolfodpk/go-crablet)](https://goreportcard.com/report/github.com/rodolfodpk/go-crablet)
[![Code Coverage](https://img.shields.io/badge/code%20coverage-76.8%25-yellow?logo=go)](https://github.com/rodolfodpk/go-crablet/actions/workflows/coverage.yml)
[![GoDoc](https://godoc.org/github.com/rodolfodpk/go-crablet?status.svg)](https://godoc.org/github.com/rodolfodpk/go-crablet)
[![License](https://img.shields.io/github/license/rodolfodpk/go-crablet)](https://github.com/rodolfodpk/go-crablet/blob/main/LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/rodolfodpk/go-crablet)](https://github.com/rodolfodpk/go-crablet/blob/main/go.mod)

# go-crablet

A Go library for event sourcing, exploring concepts inspired by Sara Pellegrini's Dynamic Consistency Boundary (DCB) pattern. 

## 🚀 Key Features

**Core API - EventStore:**
- **DCB-inspired decision models**: Project multiple states and check business invariants in a single query
- **DCB concurrency control**: Append events only if no conflicting events exist within the same query scope (uses the DCB approach, not classic optimistic locking; transaction IDs ensure correct event ordering, inspired by Oskar’s article)
- **Memory-efficient streaming**: Process events row-by-row for large event streams
- **PostgreSQL-backed**: Uses PostgreSQL for robust, concurrent event storage
- **Robust error handling**: Two-tier error handling with guaranteed transaction rollback

**Optional API - CommandExecutor:**
- **Atomic command execution**: Execute commands with handler-based event generation using the CommandExecutor pattern
- **Command tracking**: Automatic storage of commands in the `commands` table with transaction ID linking

## 📊 Performance Testing

- **[Performance Benchmarks](docs/benchmarks.md)**: Detailed benchmark results and analysis
- **[Web-App Benchmarks](internal/web-app/README.md)**: HTTP/REST API performance testing
- **[Go Benchmarks](internal/benchmarks/README.md)**: Core library performance testing

*For benchmark execution commands, see [Development Guide](docs/getting-started.md).*

## 📚 Documentation
- [Overview](docs/overview.md): DCB pattern exploration, batch projection, and streaming
- [Quick Start](docs/quick-start.md): Get started using go-crablet in your project
- [Getting Started](docs/getting-started.md): Development setup
- [Command Execution Flow](docs/command-execution-flow.md): Sequence diagram and command processing flow
- [Testing](docs/testing.md): Comprehensive testing guide and test organization
- [Performance Analysis](docs/performance-improvements.md): Detailed performance analysis

## 💡 Examples

Ready-to-run examples demonstrating different aspects of the DCB pattern:

- **[Transfer Example](internal/examples/transfer/main.go)**: Account transfer with command executor
- **[Course Enrollment](internal/examples/enrollment/main.go)**: Student course enrollment with capacity limits
- **[Streaming](internal/examples/streaming/main.go)**: Event streaming and projection approaches
- **[Decision Model](internal/examples/decision_model/main.go)**: DCB decision model with multiple projectors
- **[Multiple Events](internal/examples/batch/main.go)**: Multiple events in single append calls
- **[Advisory Locking](internal/examples/ticket_booking/main.go)**: Concert ticket booking with PostgreSQL advisory locks to prevent overbooking (experimental)

Run any example with: `go run internal/examples/[example-name]/main.go`

## 📖 References

- [Dynamic Consistency Boundary (DCB)](https://dcb.events/) - An excellent resource to understand the DCB pattern and its applications in event-driven systems
- [I am here to kill the aggregate](https://sara.event-thinking.io/2023/04/kill-aggregate-chapter-1-I-am-here-to-kill-the-aggregate.html) - Sara Pellegrini's blog post about moving beyond aggregates in event-driven systems
- [Kill Aggregate - Volume 2 - Sara Pellegrini at JOTB25](https://www.youtube.com/watch?v=AQ5fk4D3u9I)
- [Ordering in Postgres Outbox: Why Transaction IDs Matter](https://event-driven.io/en/ordering_in_postgres_outbox/) - Explains the importance of transaction IDs for event ordering and concurrency control in PostgreSQL

## 📄 License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.