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

## Installation

### Prerequisites

- **Go 1.21+**: Download from [golang.org](https://golang.org/dl/)
- **Docker**: Required for PostgreSQL database
- **Git**: For cloning the repository

### Quick Install

```bash
# Clone the repository
git clone https://github.com/rodolfodpk/go-crablet.git
cd go-crablet

# Install dependencies
go mod download

# Start PostgreSQL database
docker-compose up -d
```

## Quick Start

1. **Start the database:**
   ```bash
   docker-compose up -d
   ```

2. **Run a simple example:**
   ```bash
   go run internal/examples/decision_model/main.go
   ```

3. **Explore the API:**
   ```bash
   # Start the web application
   cd internal/web-app
   go run main.go
   
   # In another terminal, test the API
   curl http://localhost:8080/health
   ```

## Usage

### Basic Event Store Setup

```go
package main

import (
    "context"
    "github.com/rodolfodpk/go-crablet/pkg/dcb"
    "github.com/jackc/pgx/v5/pgxpool"
)

func main() {
    ctx := context.Background()
    
    // Connect to PostgreSQL
    pool, err := pgxpool.New(ctx, "postgres://user:pass@localhost/db")
    if err != nil {
        panic(err)
    }
    defer pool.Close()
    
    // Create event store
    store, err := dcb.NewEventStore(ctx, pool)
    if err != nil {
        panic(err)
    }
    
    // Your event sourcing logic here...
}
```

### Appending Events

```go
// Create an event
data := []byte(`{"courseId": "course-123", "capacity": 100}`)
event := dcb.NewInputEvent("CourseDefined", 
    dcb.NewTags("course_id", "course-123"), 
    data)

// Append to store
err := store.Append(ctx, []dcb.InputEvent{event}, nil)
```

### Reading Events

```go
// Create a query
query := dcb.NewQuerySimple(dcb.NewTags("course_id", "course-123"), "CourseDefined")

// Read events
events, err := store.Read(ctx, query)
```

For more detailed usage examples, see the [Examples](#examples) section below.

## Documentation

- [Overview](docs/overview.md): DCB pattern exploration, batch projection, and streaming
- [Examples](docs/examples.md): DCB-inspired use cases
- [Getting Started](docs/getting-started.md): Step-by-step setup guide
- [Minimal Example](docs/minimal-example.md): Detailed walkthrough of the course subscription example
- [Code Coverage](docs/code-coverage.md): Test coverage analysis and improvement guidelines
- [Benchmarks](docs/benchmarks.md): Performance testing overview

## Performance Benchmarks

Comprehensive performance testing and analysis for different API protocols:

- **[Web-App Benchmarks](internal/web-app/BENCHMARK.md)**: HTTP/REST API performance testing with k6
- **[gRPC App Benchmarks](internal/grpc-app/BENCHMARK.md)**: gRPC API performance testing with k6
- **[Go Benchmarks](internal/benchmarks/README.md)**: Core library performance testing and analysis

## Examples

Ready-to-run examples demonstrating different aspects of the DCB pattern:

### Core Examples
- **[Decision Model](internal/examples/decision_model/main.go)**: Exploring Dynamic Consistency Boundary concepts with multiple projectors
- **[Transfer](internal/examples/transfer/main.go)**: **Batch append demonstration** - Money transfer between accounts with account creation and transfer in a single atomic batch operation
- **[Enrollment](internal/examples/enrollment/main.go)**: Student course enrollment with capacity limits and business rules

### Streaming Examples
- **[Streaming Projection](internal/examples/streaming_projection/main.go)**: Memory-efficient event processing with multiple projections
- **[Cursor Streaming](internal/examples/cursor_streaming/main.go)**: Large dataset processing with batching and streaming
- **[Channel Streaming](internal/examples/channel_streaming/main.go)**: Channel-based event streaming
- **[ReadStream](internal/examples/readstream/main.go)**: Event streaming with projections and optimistic locking

### Advanced Examples
- **[Batch Operations](internal/examples/batch/main.go)**: Batch event processing and atomic operations
- **[Channel Projection](internal/examples/channel_projection/main.go)**: Channel-based state projections
- **[Extension Interface](internal/examples/extension_interface/main.go)**: Extending the event store interface

Run any example with: `go run internal/examples/[example-name]/main.go`

## Contributing

We welcome contributions! Here's how you can help:

### Development Setup

1. **Fork the repository**
2. **Clone your fork:**
   ```bash
   git clone https://github.com/your-username/go-crablet.git
   cd go-crablet
   ```

3. **Set up the development environment:**
   ```bash
   # Start PostgreSQL
   docker-compose up -d
   
   # Run tests
   go test ./pkg/dcb/...
   
   # Run benchmarks
   cd internal/benchmarks
   go run main.go
   ```

### Making Changes

1. **Create a feature branch:**
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make your changes and test:**
   ```bash
   go test ./pkg/dcb/...
   go run internal/examples/decision_model/main.go
   ```

3. **Commit your changes:**
   ```bash
   git commit -m "Add your feature description"
   ```

4. **Push and create a pull request**

### Code Style

- Follow Go conventions and use `gofmt`
- Add tests for new functionality
- Update documentation as needed
- Ensure all tests pass before submitting

### Testing

```bash
# Run all tests
go test ./pkg/dcb/...

# Run with coverage
go test -cover ./pkg/dcb/...

# Run benchmarks
cd internal/benchmarks
go run main.go
```

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## Support

- **Documentation**: Check the [docs](docs/) directory for detailed guides
- **Examples**: Explore the [examples](internal/examples/) for usage patterns
- **Issues**: Report bugs and request features on [GitHub Issues](https://github.com/rodolfodpk/go-crablet/issues)
- **Discussions**: Join the conversation on [GitHub Discussions](https://github.com/rodolfodpk/go-crablet/discussions)

## Acknowledgments

- Inspired by Sara Pellegrini's Dynamic Consistency Boundary (DCB) pattern
- Built with [pgx](https://github.com/jackc/pgx) for PostgreSQL connectivity
- Performance testing with [k6](https://k6.io/) and Go's built-in benchmarking