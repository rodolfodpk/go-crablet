[![Go Report Card](https://goreportcard.com/badge/github.com/rodolfodpk/go-crablet)](https://goreportcard.com/report/github.com/rodolfodpk/go-crablet)
[![Code Coverage](https://img.shields.io/badge/code%20coverage-82.3%25-green?logo=go)](https://github.com/rodolfodpk/go-crablet/actions/workflows/coverage.yml)
[![GoDoc](https://godoc.org/github.com/rodolfodpk/go-crablet?status.svg)](https://godoc.org/github.com/rodolfodpk/go-crablet)
[![License](https://img.shields.io/github/license/rodolfodpk/go-crablet)](https://github.com/rodolfodpk/go-crablet/blob/main/LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/rodolfodpk/go-crablet)](https://github.com/rodolfodpk/go-crablet/blob/main/go.mod)

# go-crablet

A Go library for event sourcing, exploring concepts inspired by Sara Pellegrini's Dynamic Consistency Boundary (DCB) approach. 

## 🚀 Key Features

**Core API - EventStore:**
- **DCB-inspired decision models**: Project multiple states and check business invariants in a single query
- **DCB concurrency control**: Append events only if no conflicting events exist within the same query scope (uses the DCB approach, not classic optimistic locking; transaction IDs ensure correct event ordering, inspired by Oskar's article)
- **Fail-fast concurrency limits**: Resource protection with immediate failure instead of blocking (prevents resource exhaustion under high load)
- **Memory-efficient streaming**: Process events row-by-row for large event streams
- **PostgreSQL-backed**: Uses PostgreSQL for robust, concurrent event storage
- **Robust error handling**: Two-tier error handling with guaranteed transaction rollback

**Optional API - CommandExecutor:**
- **Command execution with business logic**: Execute commands with handler-based event generation using the CommandExecutor API
- **Command tracking**: Automatic storage of commands in the `commands` table with transaction ID linking



## 📚 Documentation
- [Overview](./docs/overview.md): DCB approach exploration, batch projection, and streaming
- [Quick Start](./docs/quick-start.md): Get started using go-crablet in your project
- [Getting Started](./docs/getting-started.md): Development setup
- [Performance Guide](./docs/performance.md): Comprehensive performance information, benchmarks, and optimization details
- [Concurrency Control](./docs/concurrency-control.md): Fail-fast semaphore implementation and resource protection
- [EventStore Flow](./docs/eventstore-flow.md): Direct event operations without commands
- [Command Execution Flow](./docs/command-execution-flow.md): Sequence diagram and command processing flow
- [Low-Level Implementation](./docs/low-level-implementation.md): Database schema, SQL functions, and internal architecture
- [Testing](./docs/testing.md): Comprehensive testing guide and test organization

## 🚀 Quick Start

### 1. Start the Database
```bash
# Start PostgreSQL database
docker-compose up -d

# Wait for database to be ready
docker-compose ps
```

### 2. Run Examples
```bash
# Run any example
go run internal/examples/[example-name]/main.go

# Or use Makefile targets
make example-transfer
make example-enrollment
make example-concurrency  # runs ticket_booking
```

### 3. Cleanup
```bash
# Stop database when done
docker-compose down
```

## 💡 Examples

Ready-to-run examples demonstrating different aspects of the DCB approach:


- **[Transfer Example](internal/examples/transfer/main.go)**: Money transfer with DCB concurrency control
- **[Course Enrollment](internal/examples/enrollment/main.go)**: Student course enrollment with capacity limits
- **[Streaming](internal/examples/streaming/main.go)**: Event streaming and projection approaches
- **[Decision Model](internal/examples/decision_model/main.go)**: DCB decision model with multiple projectors
- **[Multiple Events](internal/examples/batch/main.go)**: Multiple events in single append calls
- **[Ticket Booking](internal/examples/ticket_booking/main.go)**: Concert ticket booking demonstrating DCB concurrency control with performance metrics

### Example Workflow

**Prerequisite: Database must be running!**

```bash
# 1. Start database
docker-compose up -d

# 2. Run examples
go run internal/examples/transfer/main.go
go run internal/examples/enrollment/main.go
go run internal/examples/ticket_booking/main.go

# 3. Cleanup
docker-compose down
```

**Or use Makefile targets:**
```bash
make example-transfer
make example-enrollment
make example-concurrency  # runs ticket_booking
```

## 🏃‍♂️ Performance & Benchmarks

For comprehensive performance information, benchmarks, and detailed instructions:

- **[Performance Guide](./docs/performance.md)**: Main performance index with links to all benchmark results
- **[Local PostgreSQL Performance](./docs/performance-local.md)**: Latest benchmark results and analysis
- **[Benchmark Documentation](./internal/benchmarks/README.md)**: Detailed benchmark instructions and test suite overview

**Quick benchmark command:**
```bash
cd internal/benchmarks && go test -bench=. -benchmem -benchtime=1s -timeout=10m .
```

## 📖 References

- [Dynamic Consistency Boundary (DCB)](https://dcb.events/) - An excellent resource to understand the DCB approach and its applications in event-driven systems
- [I am here to kill the aggregate](https://sara.event-thinking.io/2023/04/kill-aggregate-chapter-1-I-am-here-to-kill-the-aggregate.html) - Sara Pellegrini's blog post about moving beyond aggregates in event-driven systems
- [Kill Aggregate - Volume 2 - Sara Pellegrini at JOTB25](https://www.youtube.com/watch?v=AQ5fk4D3u9I)
- [Ordering in Postgres Outbox: Why Transaction IDs Matter](https://event-driven.io/en/ordering_in_postgres_outbox/) - Explains the importance of transaction IDs for event ordering and concurrency control in PostgreSQL

## 📄 License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.