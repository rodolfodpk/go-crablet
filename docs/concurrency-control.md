# Concurrency Control in go-crablet

## Overview

The go-crablet library implements fail-fast concurrency control for projection operations to prevent resource exhaustion, and provides configurable transaction isolation levels for read operations.

## Scope

Concurrency limits apply ONLY to projection methods that create goroutines:

- ✅ **`Project()`** - Limited by `MaxConcurrentProjections`
- ✅ **`ProjectStream()`** - Limited by `MaxConcurrentProjections`  
- ❌ **`Append()`** - NOT limited (fast, no goroutines)
- ❌ **`AppendIf()`** - NOT limited (fast, no goroutines)
- ❌ **`Query()`** - NOT limited (fast, no goroutines)
- ❌ **`QueryStream()`** - NOT limited (fast, no goroutines)

## Configuration

```go
type EventStoreConfig struct {
    MaxConcurrentProjections int `json:"max_concurrent_projections"`
    MaxProjectionGoroutines  int `json:"max_projection_goroutines"`
    DefaultReadIsolation     IsolationLevel `json:"default_read_isolation"`
}
```

**Default Values:**
- `MaxConcurrentProjections: 100` (supports ~200 concurrent users)
- `MaxProjectionGoroutines: 50` (internal goroutines per projection)
- `DefaultReadIsolation: IsolationLevelReadCommitted` (read operations)

## Read Isolation Levels

The library supports configurable transaction isolation levels for read operations:

```go
type IsolationLevel int

const (
    IsolationLevelReadCommitted   IsolationLevel = iota
    IsolationLevelRepeatableRead
    IsolationLevelSerializable
)
```

**Read Operations Affected:**
- `Query()` - Event querying
- `QueryStream()` - Streaming event querying
- `Project()` - State reconstruction
- `ProjectStream()` - Streaming state reconstruction

**Default Behavior:**
- **`READ_COMMITTED`** - Default isolation level for read operations
- Provides good balance between consistency and performance
- Suitable for most event sourcing scenarios

## Behavior

When the concurrency limit is exceeded, operations fail immediately with `TooManyProjectionsError`:

```go
type TooManyProjectionsError struct {
    EventStoreError
    MaxConcurrent int `json:"max_concurrent"`
    CurrentCount  int `json:"current_count"`
}
```

## Usage

```go
// Create EventStore with custom limits and read isolation
config := dcb.EventStoreConfig{
    MaxConcurrentProjections: 10,
    MaxProjectionGoroutines:   50,
    DefaultReadIsolation:     dcb.IsolationLevelReadCommitted,
}

store, err := dcb.NewEventStoreWithConfig(ctx, pool, config)

// Handle concurrency limit errors
states, condition, err := store.Project(ctx, projectors, nil)
if err != nil {
    if tooManyErr, ok := err.(*dcb.TooManyProjectionsError); ok {
        // Handle limit exceeded - implement retry logic
        return err
    }
    return err
}
```

## Best Practices

1. **Configure appropriate limits** based on your system capacity
2. **Implement retry logic** for `TooManyProjectionsError`
3. **Choose appropriate read isolation level**:
   - `READ_COMMITTED` - Default, good balance of consistency and performance
   - `REPEATABLE_READ` - Higher consistency, may impact performance
   - `SERIALIZABLE` - Highest consistency, significant performance impact
4. **Consider external libraries** for production resilience:
   - Circuit breaker: [gobreaker](https://github.com/sony/gobreaker) or [go-circuitbreaker](https://github.com/mercari/go-circuitbreaker)
   - Metrics: [Prometheus Go client](https://github.com/prometheus/client_golang) or [OpenTelemetry](https://github.com/open-telemetry/opentelemetry-go)
   - pgx metrics: [pgxpoolprometheus](https://github.com/IBM/pgxpoolprometheus)