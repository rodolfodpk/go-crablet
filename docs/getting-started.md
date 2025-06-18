# Getting Started

If you're new to Go and want to run the examples, follow these essential steps:

## Prerequisites

1. **Install Go** (1.22+): Download from [golang.org](https://golang.org/dl/)
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

- `internal/examples/decision_model/main.go` - Exploring Dynamic Consistency Boundary concepts
- `internal/examples/enrollment/main.go` - Course enrollment with business rules
- `internal/examples/transfer/main.go` - Money transfer between accounts
- `internal/examples/readstream/main.go` - Event streaming basics
- `internal/examples/streaming_projection/main.go` - Streaming projections

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