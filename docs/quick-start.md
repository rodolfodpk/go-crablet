# Quick Start: Using go-crablet in Your Project

This guide helps you get started using go-crablet in your Go project.

## Installation

Add go-crablet to your Go module:

```bash
go get github.com/rodolfodpk/go-crablet
```

## Prerequisites

- PostgreSQL database (version 17.5 recommended, 12 or higher supported)
- Go 1.24.5 or higher

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

    // Define a simple event using the fluent API
    event := dcb.NewEvent("UserCreated").
        WithTag("user_id", "123").
        WithData(map[string]string{
            "name":  "John Doe",
            "email": "john@example.com",
        }).
        Build()

    // Append event (simple, no conditions)
    err = store.Append(ctx, []dcb.InputEvent{event})
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Event appended successfully")

    // Query events using the new QueryBuilder
    query := dcb.NewQueryBuilder().
        WithTag("user_id", "123").
        WithType("UserCreated").
        Build()
    
    events, err := store.Query(ctx, query, nil)
    if err != nil {
        log.Fatal(err)
    }

    for _, event := range events {
        log.Printf("Event: %s, Position: %d", event.GetType(), event.GetPosition())
    }

    // Conditional append with DCB concurrency control
    if len(events) > 0 {
        // Use fluent append condition constructor
        condition := dcb.FailIfExists("user_id", "123")
        
        newEvent := dcb.NewEvent("UserUpdated").
            WithTag("user_id", "123").
            WithData(map[string]string{"name": "John Smith"}).
            Build()
            
        err = store.AppendIf(ctx, []dcb.InputEvent{newEvent}, condition)
        if err != nil {
            log.Printf("Conditional append failed: %v", err)
        }
    }
}
```

## Next Steps

After completing this quick start:

- Read the [Overview](docs/overview.md) to understand DCB concepts and transaction ID ordering
- Explore the [Overview](docs/overview.md) for the fluent API patterns
- Check out the [Examples](docs/examples.md) for complete working examples including money transfers

## Configuration

The event store can be configured with various options using `EventStoreConfig`:

```go
config := dcb.EventStoreConfig{
    MaxBatchSize:           1000, // Limits events per append call
    LockTimeout:            5000, // ms
    StreamBuffer:           1000,
    DefaultAppendIsolation: dcb.IsolationLevelReadCommitted,
    QueryTimeout:           15000, // ms
    AppendTimeout:          10000, // ms
}
store, err := dcb.NewEventStoreWithConfig(ctx, pool, config)
```

See the [API documentation](https://pkg.go.dev/github.com/rodolfodpk/go-crablet/pkg/dcb) for all available options.
