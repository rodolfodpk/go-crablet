# Installation and Development Tools

## Installation

```bash
go get github.com/rodolfodpk/go-crablet
```

## Development Tools

This project includes a Makefile to simplify common development tasks. Here are the available commands:

```bash
# Build the application
make build

# Run tests
make test

# Run tests with coverage report
make test-coverage

# Start Docker containers (PostgreSQL)
make docker-up

# Stop Docker containers
make docker-down

# Run linter
make lint

# Generate and serve documentation
make docs

# Clean build artifacts
make clean

# Show all available commands
make help
```

### Prerequisites

To use these commands, you'll need:
- Go 1.24 or later
- Docker and Docker Compose (required for both running PostgreSQL and running integration tests with testcontainers)
- golangci-lint (for the `make lint` command) 