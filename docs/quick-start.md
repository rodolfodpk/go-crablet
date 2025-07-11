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
    "time"

    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/rodolfodpk/go-crablet/pkg/dcb"
)

func main() {
    // Create context with timeout for the entire application
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    // Connect to PostgreSQL
    pool, err := pgxpool.New(ctx, "postgres://user:pass@localhost:5432/dbname")
    if err != nil {
        log.Fatal(err)
    }
    defer pool.Close()

    // Create event store
    store, err := dcb.NewEventStore(ctx, pool)
    if err != nil {
        log.Fatal(err)
    }

    // Define a simple event
    event := dcb.NewInputEvent("UserCreated", dcb.NewTags("user_id", "123"), []byte(`{"name": "John Doe", "email": "john@example.com"}`))

    // Append event (simple, no conditions)
    err = store.Append(ctx, []dcb.InputEvent{event}, nil)
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Event appended successfully")

    // Query events (use Query instead of Read)
    query := dcb.NewQuery(dcb.NewTags("user_id", "123"), "UserCreated")
    events, err := store.Query(ctx, query, nil)
    if err != nil {
        log.Fatal(err)
    }

    for _, event := range events {
        log.Printf("Event: %s, Position: %d", event.Type, event.Position)
    }

    // Conditional append with optimistic concurrency
    if len(events) > 0 {
        condition := dcb.NewAppendCondition(query)
        newEvent := dcb.NewInputEvent("UserUpdated", dcb.NewTags("user_id", "123"), []byte(`{"name": "John Smith"}`))
        err = store.Append(ctx, []dcb.InputEvent{newEvent}, &condition)
        if err != nil {
            log.Printf("Conditional append failed: %v", err)
        }
    }
}
```

## Next Steps

After completing this quick start:

- Read the [Overview](overview.md) to understand DCB concepts and transaction ID ordering
- Explore the [Examples](../internal/examples/) directory for complete working examples including money transfers

## Configuration

The event store can be configured with various options using `EventStoreConfig`:

```go
config := dcb.EventStoreConfig{
    MaxBatchSize:           1000,
    LockTimeout:            5000, // ms
    StreamBuffer:           1000,
    DefaultAppendIsolation: dcb.IsolationLevelReadCommitted,
    QueryTimeout:           15000, // ms
}
store, err := dcb.NewEventStoreWithConfig(ctx, pool, config)
```

See the [API documentation](https://godoc.org/github.com/rodolfodpk/go-crablet/pkg/dcb) for all available options. 