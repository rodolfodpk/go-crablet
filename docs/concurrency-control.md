# Concurrency Control in go-crablet

## Overview

The go-crablet library implements fail-fast concurrency control for projection operations to prevent resource exhaustion.

## Scope

Concurrency limits apply ONLY to projection methods that create goroutines:

- ✅ **`Project()`** - Limited by `MaxConcurrentProjections`
- ✅ **`ProjectStream()`** - Limited by `MaxConcurrentProjections`  
- ❌ **`Append()`** - NOT limited (fast, no goroutines)
- ❌ **`AppendIf()`** - NOT limited (fast, no goroutines)
- ❌ **`Query()`** - NOT limited (fast, no goroutines)

## Configuration

```go
type EventStoreConfig struct {
    MaxConcurrentProjections int `json:"max_concurrent_projections"`
    MaxProjectionGoroutines  int `json:"max_projection_goroutines"`
}
```

**Default Values:**
- `MaxConcurrentProjections: 100` (supports ~200 concurrent users)
- `MaxProjectionGoroutines: 50` (internal goroutines per projection)

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
// Create EventStore with custom limits
config := dcb.EventStoreConfig{
    MaxConcurrentProjections: 10,
    MaxProjectionGoroutines:   50,
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
3. **Use external circuit breakers** for system-wide protection
4. **Monitor concurrency metrics** to tune limits