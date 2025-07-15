# Performance Improvements: Data Analysis

## Benchmark Results (July 2025)

### Go Library Performance
| Operation | Throughput | Latency | Memory | Allocations |
|-----------|------------|---------|---------|-------------|
| Read Single | 3,200+ ops/sec | 350μs | 2.3MB | 31K |
| Read Batch | 3 ops/sec | 310ms | 267MB | 3M |
| Append Single | 900+ ops/sec | 1.1ms | 2.3MB | 31K |
| AppendIf | 3-4 ops/sec | 170ms | 12MB | 156K |
| Project Single | 3,000+ ops/sec | 350μs | 1.4MB | 35K |
| Advisory Locks | 900+ ops/sec | 1.2ms | 2.3MB | 31K |
| Advisory Locks (5 goroutines) | 200+ ops/sec | 4.7ms | 2.3MB | 31K |

### HTTP API Performance
| Endpoint | Throughput | Latency | Notes |
|----------|------------|---------|-------|
| Quick Test | 1,275 req/sec | 1.47ms | Basic health check |
| Append | 62 req/sec | 805ms | Single event append |
| Advisory Locks | 216 req/sec | 4.6ms | Resource locking |
| AppendIf | 30 req/sec | 1.75s | Conditional append |

### Concurrency Limits
- **Optimal**: 5 concurrent goroutines (4.7ms latency)
- **Acceptable**: 8 concurrent goroutines (6.2ms latency)  
- **Degraded**: 10+ concurrent goroutines (8.5ms+ latency)
- **Connection Pool**: 20 max connections

## I/O Operation Analysis

### Database I/O Operations by Append Type

| Append Type | I/O Operations | Performance | Use Case |
|-------------|----------------|-------------|----------|
| **Regular Append** | 1 I/O: INSERT | ~1.1ms | Simple event storage |
| **Advisory Locks (no conditions)** | 1 I/O: INSERT | ~1.2ms | Resource locking only |
| **DCB Conditions (AppendIf)** | 2 I/O: SELECT + INSERT | ~170ms | Business rule validation |
| **Advisory Locks + DCB** | 2 I/O: SELECT + INSERT | ~170ms | Resource locking + business rules |

### Advisory Locks vs DCB Conditions: Performance Analysis

#### Operation Sequence

**Advisory Locks + DCB Conditions** (when both are used):
```sql
-- 1. In-memory lock acquisition (no I/O)
PERFORM pg_advisory_xact_lock(hashtext(lock_key));

-- 2. DCB condition check (I/O operation)
condition_result := check_append_condition(fail_if_events_match, after_cursor);

-- 3. Event insertion (I/O operation)
PERFORM append_events_batch(p_types, p_tags, p_data);
```

**DCB Conditions Only** (AppendIf):
```sql
-- 1. DCB condition check (I/O operation)
condition_result := check_append_condition(fail_if_events_match, after_cursor);

-- 2. Event insertion (I/O operation)
PERFORM append_events_batch(p_types, p_tags, p_data);
```

#### Why Advisory Locks Appear More Performant

The performance difference comes from **what scenarios are being benchmarked**:

**Advisory lock benchmarks** typically test simple resource locking:
```go
// Simple advisory lock (no DCB conditions)
event := dcb.NewInputEvent("TestEvent",
    dcb.NewTags("lock:resource", "123"),  // Only advisory lock
    []byte(`{"data": "test"}`))
store.Append(ctx, []dcb.InputEvent{event}, nil)  // No condition = 1 I/O operation
```

**AppendIf benchmarks** test complex business rule validation:
```go
// DCB condition check
condition := dcb.NewAppendCondition(query)
store.Append(ctx, []dcb.InputEvent{event}, &condition)  // With condition = 2 I/O operations
```

#### Performance Comparison

| Scenario | I/O Operations | Performance | Use Case |
|----------|----------------|-------------|----------|
| **Advisory Locks (no conditions)** | 1 I/O: INSERT | ~1.2ms | Simple resource serialization |
| **DCB Conditions (AppendIf)** | 2 I/O: SELECT + INSERT | ~170ms | Business rule validation |
| **Advisory Locks + DCB** | 2 I/O: SELECT + INSERT | ~170ms | Resource locking + business rules |

**Key Insight**: Advisory locks appear more performant because benchmarks typically test **simple scenarios without DCB conditions**. When both advisory locks and DCB conditions are used together, performance equals AppendIf (2 I/O operations).

## Technical Optimizations

### 1. Shared Connection Pool
**Problem**: 64 benchmarks × 20 connections = 1,280 connections (exceeded PostgreSQL limit)

**Solution**: Global shared pool with 20 connections
```go
poolConfig.MaxConns = 20
poolConfig.MinConns = 5
poolConfig.MaxConnLifetime = 5 * time.Minute
```

**Result**: Eliminated connection exhaustion, improved benchmark stability

### 2. Debug Logging Removal
**Problem**: Excessive logging slowed benchmarks by ~30-50%

**Solution**: Removed debug logging from append operations and HTTP handlers

**Result**: Faster benchmark execution, cleaner output

### 3. Advisory Lock Implementation
**Implementation**: PostgreSQL advisory locks via `pg_advisory_xact_lock()`
```sql
PERFORM pg_advisory_xact_lock(hashtext(lock_key));
```

**Performance**: 900+ ops/sec single, 200+ ops/sec concurrent (5 goroutines)

### 4. Database Credential Standardization
**Problem**: Inconsistent credentials caused connection failures

**Solution**: Standardized on `postgres://crablet:crablet@localhost:5432/crablet`

**Result**: Reliable connectivity across all components

## Error Handling

### Two-Tier Architecture
1. **Database Level**: PostgreSQL function errors → ResourceError
2. **Application Level**: JSONB status responses → ConcurrencyError

### Transaction Management
```go
defer tx.Rollback(ctx)  // Guaranteed rollback
// ... operations ...
if err != nil {
    return err  // Transaction rolled back
}
tx.Commit(ctx)  // Only on success
```

## Performance Characteristics

### Throughput Hierarchy (Fastest to Slowest)
1. Read/Projection: 3,000+ ops/sec
2. Basic Append: 900+ ops/sec  
3. Advisory Locks: 900+ ops/sec
4. HTTP API: 1,275+ req/sec
5. Concurrent Locks: 200+ ops/sec
6. Conditional Append: 3-4 ops/sec

### Memory Usage
- **Single Operations**: 2.3MB (read/append), 1.4MB (projection)
- **Batch Operations**: 267MB (read), 139MB (projection)
- **Conditional Operations**: 12MB (append), 10MB (read)

### Concurrency Performance
- **5 goroutines**: Optimal performance (4.7ms latency)
- **8 goroutines**: Acceptable performance (6.2ms latency)
- **10+ goroutines**: Performance degradation (8.5ms+ latency)

## System Status

### Strengths
- 100% success rate across all tests
- Zero concurrency errors
- Consistent performance across isolation levels
- Robust error handling with guaranteed rollback
- Effective advisory lock implementation

### Limitations
- Conditional operations: 3-4 ops/sec (business requirement)
- HTTP API overhead: ~1.5ms base latency
- Large batches: Performance degrades with 1000+ events

### Recommendations
- Use direct library calls for production (not HTTP API)
- Limit concurrency to 5-8 goroutines for optimal performance
- Conditional operations required for business logic (unavoidable overhead)
- Monitor memory usage for large batch operations
