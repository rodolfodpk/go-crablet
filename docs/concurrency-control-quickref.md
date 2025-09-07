# Concurrency Control Quick Reference

## Fail-Fast Semaphore Pattern

This library uses a **fail-fast semaphore** to limit concurrent projections and protect system resources.

### Key Concepts

- **Fail-fast**: Immediate failure instead of blocking when limits exceeded
- **Resource protection**: Prevents goroutine and memory exhaustion
- **Predictable behavior**: Clear success/failure responses
- **Retry-friendly**: Applications can implement retry logic

### Configuration

```go
config := dcb.EventStoreConfig{
    MaxConcurrentProjections: 200 concurrent projections
    MaxProjectionGoroutines:   100, // Default: 100 internal goroutines
}
```

### Usage Pattern

```go
// Project with automatic concurrency control
states, condition, err := store.Project(ctx, projectors, nil)
if err != nil {
    if tooManyErr, ok := err.(*dcb.TooManyProjectionsError); ok {
        // Handle concurrency limit exceeded
        log.Printf("Too many projections: %d/%d", 
            tooManyErr.CurrentCount, tooManyErr.MaxConcurrent)
        // Implement retry logic
        return retryProjection(store, projectors)
    }
    return err
}
```

### Error Handling

```go
type TooManyProjectionsError struct {
    EventStoreError
    MaxConcurrent int `json:"max_concurrent"`
    CurrentCount  int `json:"current_count"`
}
```

### Retry Logic Example

```go
func retryProjection(store dcb.EventStore, projectors []dcb.StateProjector) error {
    for attempt := 0; attempt < 3; attempt++ {
        _, _, err := store.Project(ctx, projectors, nil)
        if err == nil {
            return nil
        }
        
        if _, ok := err.(*dcb.TooManyProjectionsError); ok {
            delay := time.Duration(1<<attempt) * 100 * time.Millisecond
            time.Sleep(delay)
            continue
        }
        
        return err
    }
    return fmt.Errorf("max retries exceeded")
}
```

### Performance Characteristics

| Metric | Value |
|--------|-------|
| **Acquisition time** | ~1ns |
| **Memory per semaphore** | ~8 bytes |
| **Memory per token** | 0 bytes |

### Best Practices

1. **Configure appropriate limits** based on your workload
2. **Implement retry logic** with exponential backoff
3. **Monitor concurrency metrics** for optimization
4. **Handle failures gracefully** with user-friendly errors

For detailed documentation, see [Concurrency Control](./concurrency-control.md).
