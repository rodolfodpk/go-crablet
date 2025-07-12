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
- `internal/examples/readstream/main.go` - Event streaming basics
- `internal/examples/streaming_projection/main.go` - Streaming projections
- `internal/examples/cursor_streaming/main.go` - Large dataset processing
- `internal/examples/batch/main.go` - Multiple events in single append calls
- `internal/examples/channel_projection/main.go`
