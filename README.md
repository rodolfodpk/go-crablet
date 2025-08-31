[![Go Report Card](https://goreportcard.com/badge/github.com/rodolfodpk/go-crablet)](https://goreportcard.com/report/github.com/rodolfodpk/go-crablet)
[![Code Coverage](https://img.shields.io/badge/code%20coverage-79.9%25-yellow?logo=go)](https://github.com/rodolfodpk/go-crablet/actions/workflows/coverage.yml)
[![GoDoc](https://godoc.org/github.com/rodolfodpk/go-crablet?status.svg)](https://godoc.org/github.com/rodolfodpk/go-crablet)
[![License](https://img.shields.io/github/license/rodolfodpk/go-crablet)](https://github.com/rodolfodpk/go-crablet/blob/main/LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/rodolfodpk/go-crablet)](https://github.com/rodolfodpk/go-crablet/blob/main/go.mod)

# go-crablet

A Go library for event sourcing, exploring concepts inspired by Sara Pellegrini's Dynamic Consistency Boundary (DCB) approach. 

## 🚀 Key Features

**Core API - EventStore:**
- **DCB-inspired decision models**: Project multiple states and check business invariants in a single query
- **DCB concurrency control**: Append events only if no conflicting events exist within the same query scope (uses the DCB approach, not classic optimistic locking; transaction IDs ensure correct event ordering, inspired by Oskar’s article)
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

## 🏃‍♂️ Running Benchmarks

### Prerequisites
- PostgreSQL 16 (local or Docker)
- Go 1.25+
- Database setup (see Quick Start above)

### Run All Benchmarks
```bash
# Run all benchmarks (Tiny, Small, Medium datasets)
cd internal/benchmarks
go test -bench=. -benchmem -benchtime=1s -timeout=10m .

# Expected execution time: ~2.9 minutes
```

### Run Specific Datasets
```bash
# Run only Small dataset benchmarks
go test -bench=BenchmarkAppend_Small -benchmem -benchtime=1s -timeout=10m .

# Run only Tiny dataset benchmarks  
go test -bench=BenchmarkAppend_Tiny -benchmem -benchtime=1s -timeout=10m .

# Run only Medium dataset benchmarks
go test -bench=BenchmarkAppend_Medium -benchmem -benchtime=1s -timeout=10m .
```

### Run Specific Operations
```bash
# Run only Append operations (all datasets)
go test -bench=Append_Concurrent -benchmem -benchtime=1s -timeout=10m .

# Run only AppendIf operations (all datasets)
go test -bench=AppendIf_ -benchmem -benchtime=1s -timeout=10m .

# Run only Projection operations (all datasets)
go test -bench=Project_ -benchmem -benchtime=1s -timeout=10m .
```

### Benchmark Results
- **Latest Results**: [Local PostgreSQL Performance](./docs/performance-local.md)
- **Benchmark Suite**: 54 standardized tests covering Append, AppendIf, and Projection operations
- **Performance Metrics**: Throughput (ops/sec), Latency (ns/op), Memory (B/op), Allocations

For detailed benchmark documentation, see [internal/benchmarks/README.md](./internal/benchmarks/README.md).

## 📖 References

- [Dynamic Consistency Boundary (DCB)](https://dcb.events/) - An excellent resource to understand the DCB approach and its applications in event-driven systems
- [I am here to kill the aggregate](https://sara.event-thinking.io/2023/04/kill-aggregate-chapter-1-I-am-here-to-kill-the-aggregate.html) - Sara Pellegrini's blog post about moving beyond aggregates in event-driven systems
- [Kill Aggregate - Volume 2 - Sara Pellegrini at JOTB25](https://www.youtube.com/watch?v=AQ5fk4D3u9I)
- [Ordering in Postgres Outbox: Why Transaction IDs Matter](https://event-driven.io/en/ordering_in_postgres_outbox/) - Explains the importance of transaction IDs for event ordering and concurrency control in PostgreSQL

## 📄 License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.