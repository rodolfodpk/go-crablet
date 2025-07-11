# Getting Started

If you're new to Go and want to run the examples, follow these essential steps:

## Prerequisites

1. **Install Go** (1.24+): Download from [golang.org](https://golang.org/dl/)
2. **Install Docker**: Download from [docker.com](https://docker.com/get-started/)
3. **Install Git**: Download from [git-scm.com](https://git-scm.com/)

## Quick Start

1. **Clone the repository:**
   ```bash
   git clone https://github.com/rodolfodpk/go-crablet.git
   cd go-crablet
   ```

2. **Start PostgreSQL database:**
   ```bash
   docker-compose up -d
   ```

3. **Run an example:**
   ```bash
   go run internal/examples/decision_model/main.go
   ```

## Available Examples

All examples are located in `internal/examples/` and demonstrate different aspects of the DCB pattern:

- `internal/examples/decision_model/main.go` - Exploring Dynamic Consistency Boundary concepts
- `internal/examples/enrollment/main.go` - Course enrollment with business rules
- `internal/examples/transfer/main.go` - Money transfer between accounts
- `internal/examples/readstream/main.go` - Event streaming basics
- `internal/examples/streaming_projection/main.go` - Streaming projections
- `internal/examples/cursor_streaming/main.go` - Large dataset processing
- `internal/examples/batch/main.go` - Batch event processing
- `internal/examples/channel_projection/main.go` - Channel-based projections
- `internal/examples/channel_streaming/main.go` - Channel-based streaming
- `internal/examples/extension_interface/main.go` - Extending the event store

## Project Structure

This project uses a single Go module with organized internal packages:

### Core Library (`pkg/dcb`)
```bash
cd pkg/dcb
go build ./...    # Build core library
go test ./...     # Run all tests
```

### Web Application (`internal/web-app`)
```bash
cd internal/web-app
go build ./...    # Build REST API server
make test         # Run k6 performance benchmarks
make run          # Start server locally
```

### Benchmarks (`internal/benchmarks`)
```bash
cd internal/benchmarks
go build ./...    # Build benchmark tools
go run main.go    # Run performance benchmarks
```

## Testing

The project has a comprehensive test suite with clear organization:

### Test Structure
- **External Tests** (`pkg/dcb/tests/`): Tests that consume only the public API
- **Internal Tests** (`pkg/dcb/`): Tests with access to internal implementation details

### Running Tests
```bash
# Run all tests
go test ./pkg/dcb/... -v

# Run only external tests
go test ./pkg/dcb/tests/... -v

# Run with coverage
go test ./pkg/dcb/... -coverprofile=coverage.out
```

For detailed testing information, see the [Testing Guide](testing.md).

## Troubleshooting

- **Database connection error**: Make sure PostgreSQL is running with `docker-compose ps`
- **Database does not exist error**: If you see an error like `database "dcb_app" does not exist`, you may need to reset the database volume to trigger initialization:
  ```bash
  docker-compose down -v
  docker-compose up -d
  ```
  This will remove the old database and re-create it with the correct schema.
- **Go module error**: Run `go mod download` to download dependencies
- **Permission error**: Make sure Docker is running and you have permissions

## Next Steps

For more detailed examples and documentation, see the [Examples](examples.md) guide.

`Query` and `QueryItem` must be constructed using helper functions such as `NewQuery`, `NewQueryItem`, and `NewQueryFromItems`. You cannot use struct literals or access fields directly. This ensures DCB compliance and improves type safety.