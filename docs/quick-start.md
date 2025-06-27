# Quick Start: Using go-crablet in Your Project

This guide helps you get started using go-crablet in your Go project.

## Installation

Add go-crablet to your Go module:

```bash
go get github.com/rodolfodpk/go-crablet
```

## Prerequisites

- PostgreSQL database (version 12 or higher)
- Go 1.21 or higher

## Database Setup

1. Create a PostgreSQL database
2. Run the schema setup:

```bash
# Using docker-compose (recommended for development)
docker-compose up -d postgres

# Or manually run the schema
psql -d your_database -f docker-entrypoint-initdb.d/schema.sql
```

## Basic Usage

```go
package main

import (
    "context"
    "log"

    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/rodolfodpk/go-crablet/pkg/dcb"
)

func main() {
    // Connect to PostgreSQL
    pool, err := pgxpool.New(context.Background(), "postgres://user:pass@localhost:5432/dbname")
    if err != nil {
        log.Fatal(err)
    }
    defer pool.Close()

    // Create event store
    store, err := dcb.NewEventStore(pool)
    if err != nil {
        log.Fatal(err)
    }

    // Define a simple event
    event := dcb.InputEvent{
        Type: "UserCreated",
        Tags: []string{"user", "user:123"},
        Data: map[string]any{"name": "John Doe", "email": "john@example.com"},
    }

    // Append event
    position, err := store.Append(context.Background(), []dcb.InputEvent{event}, nil)
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Event appended at position: %d", position)

    // Read events
    query := dcb.NewQuery(dcb.NewQueryItem("user", "user:123"))
    events, err := store.Read(context.Background(), query, nil)
    if err != nil {
        log.Fatal(err)
    }

    for _, event := range events.Events {
        log.Printf("Event: %s, Data: %v", event.Type, event.Data)
    }
}
```

## Next Steps

- Read the [Overview](overview.md) to understand DCB concepts
- Check out the [Minimal Example](minimal-example.md) for a complete walkthrough
- Explore the [examples](../internal/examples/) for more advanced usage patterns

## Configuration

The event store can be configured with various options:

```go
store, err := dcb.NewEventStore(pool, dcb.WithMaxConnections(10))
```

See the [API documentation](https://godoc.org/github.com/rodolfodpk/go-crablet/pkg/dcb) for all available options. 