# Performance Analysis

## Current System Performance

### Go Library Performance
| Operation | Throughput | Latency | Memory | Allocations |
|-----------|------------|---------|---------|-------------|
| Read Single | 3,200+ ops/sec | 350μs | 2.3MB | 31K |
| Read Batch | 3 ops/sec | 310ms | 267MB | 3M |
| Append Single | 900+ ops/sec | 1.1ms | 2.3MB | 31K |
| Append with Advisory Locks | 900+ ops/sec | 1.2ms | 2.3MB | 31K |
| Append with Advisory Locks (5 goroutines) | 200+ ops/sec | 4.7ms | 2.3MB | 31K |
| AppendIf | 3-4 ops/sec | 170ms | 12MB | 156K |
| Project Single | 3,000+ ops/sec | 350μs | 1.4MB | 35K |

### HTTP API Performance
| Endpoint | Throughput | Latency | Notes |
|----------|------------|---------|-------|
| Quick Test | 1,275 req/sec | 1.47ms | Basic health check |
| Append | 62 req/sec | 805ms | Single event append |
| Append with Advisory Locks | 216 req/sec | 4.6ms | Event append with resource locking tags |
| AppendIf | 30 req/sec | 1.75s | Conditional append |

## Database I/O Operations

### I/O Operations by Append Type

| Append Type | I/O Operations | Performance | Use Case |
|-------------|----------------|-------------|----------|
| **Regular Append** | 1 I/O: INSERT | ~1.1ms | Simple event storage |
| **Advisory Locks (no conditions)** | 1 I/O: INSERT | ~1.2ms | Resource locking only |
| **DCB Conditions (AppendIf)** | 2 I/O: SELECT + INSERT | ~170ms | Business rule validation |
| **Advisory Locks + DCB** | 2 I/O: SELECT + INSERT | ~170ms | Resource locking + business rules |

### Operation Sequence

**Advisory Locks + DCB Conditions**:
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

## System Configuration

### Connection Pool
```go
poolConfig.MaxConns = 20
poolConfig.MinConns = 5
poolConfig.MaxConnLifetime = 5 * time.Minute
```

### Concurrency Limits
- **Optimal**: 5 concurrent goroutines (4.7ms latency)
- **Acceptable**: 8 concurrent goroutines (6.2ms latency)  
- **Degraded**: 10+ concurrent goroutines (8.5ms+ latency)

### Transaction Management
```go
defer tx.Rollback(ctx)  // Guaranteed rollback
// ... operations ...
if err != nil {
    return err  // Transaction rolled back
}
tx.Commit(ctx)  // Only on success
```

## Performance Insights

### Key Characteristics
- **Read/Projection**: Fastest operations (3,000+ ops/sec)
- **Basic Append**: High throughput (900+ ops/sec)
- **Advisory Locks**: Same performance as basic append when used alone
- **Conditional Operations**: Slow but required for business logic (3-4 ops/sec)
- **HTTP API Overhead**: ~1.5ms additional latency vs direct library calls

### Memory Usage
- **Single Operations**: 2.3MB (read/append), 1.4MB (projection)
- **Batch Operations**: 267MB (read), 139MB (projection)
- **Conditional Operations**: 12MB (append), 10MB (read)

### Production Recommendations
- Use direct library calls for production (not HTTP API)
- Limit concurrency to 5-8 goroutines for optimal performance
- Conditional operations required for business logic (unavoidable overhead)
- Monitor memory usage for large batch operations
