# Getting Started

If you're new to Go and want to run the examples, follow these essential steps:

## Prerequisites

1. **Install Go** (1.24.5+): Download from [golang.org](https://golang.org/dl/)
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
- `internal/examples/streaming/main.go` - Event streaming basics
- `internal/examples/batch/main.go` - Multiple events in single append calls
- `internal/examples/concurrency_comparison/main.go` - Concert ticket booking comparing DCB concurrency control vs PostgreSQL advisory locks
- `internal/examples/utils/main.go` - Utility functions and helpers

### Running Examples with Parameters

Some examples support command-line parameters for testing different scenarios:

```bash
# Ticket booking example with custom parameters
go run internal/examples/concurrency_comparison/main.go -users 50 -seats 30 -tickets 1

# Show help for available options
go run internal/examples/concurrency_comparison/main.go -h
```
