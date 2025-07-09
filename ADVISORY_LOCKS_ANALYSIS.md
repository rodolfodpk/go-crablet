# Advisory Locks Analysis: Safety and Scalability

## Current Implementation Overview

Our advisory locks implementation uses PostgreSQL's `pg_advisory_xact_lock()` function to provide aggregate-level locking for event sourcing operations.

### Key Components

1. **Function**: `append_events_with_advisory_locks()` in PostgreSQL
2. **Lock Type**: `pg_advisory_xact_lock(hashtext(lock_key))`
3. **Scope**: Transaction-scoped locks
4. **Trigger**: Automatic when events have `lock:` prefixed tags

## Safety Assessment

### ✅ Strengths

1. **Automatic Cleanup**: Transaction-scoped locks are automatically released on commit/rollback
2. **Deadlock Prevention**: Lock keys are sorted before acquisition
3. **Hash-based Keys**: `hashtext()` prevents special character issues
4. **No Manual Management**: No risk of orphaned locks
5. **Atomic Operations**: Lock acquisition and event append are atomic

### ⚠️ Potential Issues

1. **Lock Granularity**: All events in a batch share the same lock keys
2. **Hash Collisions**: Extremely rare but theoretically possible
3. **No Explicit Timeout**: Could block indefinitely (now fixed with timeout parameter)

## Scalability Assessment

### ❌ Major Concerns

1. **Global Lock Space**: All advisory locks compete in the same global PostgreSQL namespace
2. **Limited Lock Count**: PostgreSQL typically supports ~64K advisory locks
3. **No Partitioning**: High contention on popular lock keys
4. **Blocking Behavior**: Contending transactions block until locks are released

### 📊 Scalability Limits

```sql
-- Check current advisory lock usage
SELECT get_advisory_lock_count();

-- Monitor active advisory locks
SELECT * FROM get_advisory_lock_stats();

-- PostgreSQL settings affecting locks
SELECT name, setting, unit, context 
FROM pg_settings 
WHERE name IN ('max_locks_per_transaction', 'lock_timeout');
```

## Improvements Made

### 1. Added Lock Timeout

```sql
-- Function now accepts timeout parameter
CREATE OR REPLACE FUNCTION append_events_with_advisory_locks(
    p_types TEXT[],
    p_tags TEXT[],
    p_data JSONB[],
    p_condition JSONB DEFAULT NULL,
    p_lock_timeout_ms INTEGER DEFAULT 5000 -- 5 second default
) RETURNS VOID
```

### 2. Added Monitoring Functions

```sql
-- Monitor active advisory locks
SELECT * FROM get_advisory_lock_stats();

-- Get current lock count
SELECT get_advisory_lock_count();
```

### 3. Enhanced Error Handling

- Timeout errors are now properly handled and returned as `ConcurrencyError`
- Context-based timeout configuration
- Proper cleanup on exceptions

## Recommendations for Production Use

### 1. **Monitor Lock Usage**

```go
// Add monitoring to your application
func (s *Server) monitorAdvisoryLocks() {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    
    for range ticker.C {
        var count int
        err := s.pool.QueryRow(context.Background(), 
            "SELECT get_advisory_lock_count()").Scan(&count)
        if err != nil {
            log.Printf("Failed to get lock count: %v", err)
            continue
        }
        
        if count > 1000 { // Alert threshold
            log.Printf("WARNING: High advisory lock count: %d", count)
        }
    }
}
```

### 2. **Use Appropriate Timeouts**

```go
// Set context with timeout for critical operations
ctx := context.WithValue(context.Background(), "lock_timeout_ms", 2000) // 2 seconds
err := s.appendWithAdvisoryLocks(ctx, events, condition)
```

### 3. **Consider Lock Key Design**

```go
// Good: Specific, scoped lock keys
tags := []dcb.Tag{
    dcb.NewTag("lock:user:123", "true"),
    dcb.NewTag("lock:tenant:acme", "true"),
}

// Avoid: Too broad lock keys
tags := []dcb.Tag{
    dcb.NewTag("lock:global", "true"), // Too broad!
}
```

### 4. **Implement Circuit Breaker**

```go
// Add circuit breaker for advisory lock operations
type AdvisoryLockCircuitBreaker struct {
    failureThreshold int
    failureCount     int
    lastFailureTime  time.Time
    timeout          time.Duration
}

func (cb *AdvisoryLockCircuitBreaker) Execute(operation func() error) error {
    if cb.isOpen() {
        return fmt.Errorf("circuit breaker is open")
    }
    
    err := operation()
    if err != nil {
        cb.recordFailure()
    } else {
        cb.recordSuccess()
    }
    return err
}
```

## Alternative Approaches

### 1. **Database-Level Row Locks**

```sql
-- Use SELECT FOR UPDATE on aggregate tables
SELECT * FROM aggregates WHERE id = $1 FOR UPDATE;
```

**Pros**: More granular, database-native
**Cons**: Requires separate aggregate tables, more complex

### 2. **Application-Level Distributed Locks**

```go
// Use Redis or similar for distributed locking
func (s *Server) acquireDistributedLock(key string) (bool, error) {
    return s.redis.SetNX(key, "locked", 30*time.Second).Result()
}
```

**Pros**: Can scale across multiple databases
**Cons**: Additional infrastructure, eventual consistency

### 3. **Optimistic Locking Only**

```go
// Rely only on append conditions without advisory locks
err := s.store.AppendIf(ctx, events, condition)
```

**Pros**: No blocking, better scalability
**Cons**: More retries, eventual consistency

## Conclusion

### Current State: **Moderately Safe, Limited Scalability**

**Safety**: ✅ Good with recent timeout improvements
**Scalability**: ⚠️ Limited by global lock space and blocking behavior

### Recommendations

1. **For Small to Medium Scale**: Current implementation is adequate
2. **For High Scale**: Consider alternatives like distributed locks or optimistic locking
3. **Always Monitor**: Implement the monitoring functions in production
4. **Use Timeouts**: Always set appropriate lock timeouts
5. **Design Lock Keys Carefully**: Avoid overly broad lock keys

### Next Steps

1. Implement monitoring in production
2. Set up alerting for high lock counts
3. Consider performance testing under load
4. Evaluate alternative approaches for high-scale deployments 