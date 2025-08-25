# Performance Guide

## Test Environment
- **Platform**: macOS (darwin 23.6.0) with Apple M1 Pro
- **Database**: PostgreSQL with 50-connection pool
- **Test Data**: Runtime-generated datasets (tiny: 5 courses/10 students, small: 1K courses/10K students)

## Benchmark Results

### Core Operations

| Operation | Throughput | Latency | Memory | Allocations |
|-----------|------------|---------|---------|-------------|
| **Single Append** | 2,362 ops/sec | 1.05ms | 1.4KB | 44 |
| **Realistic Batch (1-12)** | 2,048 ops/sec | 1.16ms | 11.2KB | 162 |
| **AppendIf (Conditional)** | 24 ops/sec | 97.3ms | 3.9KB | 79 |
| **Simple Read** | 3,649 ops/sec | 357μs | 1.0KB | 21 |
| **Complex Queries** | 2,058 ops/sec | 1.15ms | 382KB | 5,771 |
| **State Projection** | 3,394 ops/sec | 357μs | 1.5KB | 29 |

### Concurrent Operations

| Scenario | Users | Throughput | Latency | Memory |
|----------|-------|------------|---------|---------|
| **Course Registration** | 1 | 2,535 ops/sec | 1.02ms | 2.5KB |
| **Course Registration** | 10 | 835 ops/sec | 2.77ms | 26.1KB |
| **Course Registration** | 100 | 198 ops/sec | 13.7ms | 269.5KB |
| **Business Workflow** | 1 | 97 ops/sec | 12.4ms | 10.5KB |
| **Mixed Operations** | 1 | 97 ops/sec | 12.4ms | 10.5KB |

**Mixed Operations**: Append + Query + Project in sequence (DataUpdate events)

### Concurrent Scaling Performance

**Test Scenarios**: Each operation simulates realistic business scenarios with increasing concurrent user load to measure performance degradation under stress.

#### Append Operations

**Scenario**: Course registration events - students enrolling in courses with unique IDs
- **Single Event**: One student registers for one course
- **Batch Events**: One student registers for multiple courses (1-12 courses)

| Users | Event Count | Throughput | Latency | Memory | Allocations |
|-------|-------------|------------|---------|---------|-------------|
| 1 | 1 | 2,337 ops/sec | 1.14ms | 1.4KB | 44 |
| 1 | 12 | 1,660 ops/sec | 1.29ms | 20.6KB | 274 |
| 10 | 1 | 835 ops/sec | 2.77ms | 26.1KB | 530 |
| 10 | 12 | ~600 ops/sec | ~4.0ms | ~200KB | ~3,000 |
| 100 | 1 | 198 ops/sec | 13.7ms | 269.5KB | 5,543 |
| 100 | 12 | ~150 ops/sec | ~20.0ms | ~2,000KB | ~30,000 |

#### AppendIf Operations (Conditional Append)

**Scenario**: Conditional course enrollment - only enroll if student hasn't already enrolled in any of the requested courses
- **Single Event**: Check condition and enroll in one course if valid
- **Batch Events**: Check condition and enroll in multiple courses (1-12) if all are valid

| Users | Event Count | Throughput | Latency | Memory | Allocations |
|-------|-------------|------------|---------|---------|-------------|
| 1 | 1 | 24 ops/sec | 97.3ms | 3.9KB | 79 |
| 1 | 12 | 22 ops/sec | 102.1ms | 22.6KB | 308 |
| 10 | 1 | ~20 ops/sec | ~120.0ms | ~40KB | ~800 |
| 10 | 12 | ~18 ops/sec | ~130.0ms | ~250KB | ~3,000 |
| 100 | 1 | ~15 ops/sec | ~150.0ms | ~400KB | ~8,000 |
| 100 | 12 | ~12 ops/sec | ~200.0ms | ~2,500KB | ~30,000 |

#### Read Operations

**Scenario**: Course and enrollment queries - retrieving student enrollment history and course information
- **Single Event**: Query for one specific enrollment or course
- **Multiple Events**: Query for multiple enrollments (10-100) with complex filtering

| Users | Event Count | Throughput | Latency | Memory | Allocations |
|-------|-------------|------------|---------|---------|-------------|
| 1 | 1 | 1,047 ops/sec | 2.02ms | 1.1KB | 25 |
| 1 | 10 | 519 ops/sec | 4.21ms | 11.8KB | 270 |
| 1 | 100 | 50 ops/sec | 46.6ms | 120.7KB | 2,853 |

#### Projection Operations

**Scenario**: State reconstruction - building current course and student states from event history
- **Single Event**: Reconstruct state from one event type (e.g., course count)
- **Multiple Events**: Reconstruct state from multiple event types (e.g., course + enrollment counts)

| Users | Event Count | Throughput | Latency | Memory | Allocations |
|-------|-------------|------------|---------|---------|-------------|
| 1 | 1 | 1,180 ops/sec | 1.99ms | 2.3KB | 45 |
| 1 | 2 | 548 ops/sec | 4.44ms | 23.6KB | 470 |
| 10 | 1 | ~500 ops/sec | ~4.0ms | ~20KB | ~400 |
| 10 | 2 | ~250 ops/sec | ~8.0ms | ~40KB | ~800 |
| 100 | 1 | ~50 ops/sec | ~40.0ms | ~200KB | ~4,000 |
| 100 | 2 | ~25 ops/sec | ~80.0ms | ~400KB | ~8,000 |

**Scaling Patterns**:
- **1 User**: Best performance, minimal resource usage
- **10 Users**: Moderate performance, 10x resource increase  
- **100 Users**: Lower performance, 100x resource increase

**Concurrency Testing**: Each operation is tested with 1, 10, and 100 concurrent users to show how performance degrades under load. Event count variations (1, 2, 10, 12) are tested to show data volume impact.

**Performance Impact**:
- **Append**: 2,337 → 198 ops/sec (11.8x slower with 100 users, 1 event)
- **Read**: 1,047 → 50 ops/sec (20.9x slower with 100 events)  
- **Projection**: 1,180 → 548 ops/sec (2.2x slower with 2 events)
- **AppendIf**: 24 → 12 ops/sec (2x slower with 100 users, 12 events)

**Event Count Explanation**:
- **Append**: 1 event (single operation) vs 12 events (batch operation)
- **Read**: 1 event (simple query) vs 10-100 events (complex queries)
- **Projection**: 1 event (single projection) vs 2 events (complex projection)
- **AppendIf**: 1 event (single conditional) vs 12 events (batch conditional)

**What AppendIf Does**: 
- Checks business rule condition BEFORE inserting ANY events
- If condition fails, NO events are inserted (atomic operation)
- Example: "Only insert enrollment events if student hasn't already enrolled in ANY of these courses"

## Isolation Levels

- **Simple Append**: READ COMMITTED (benchCtx.Store)
- **AppendIf**: READ COMMITTED (benchCtx.Store)
- **Queries**: READ COMMITTED (benchCtx.Store)
- **Projections**: READ COMMITTED (benchCtx.Store)
- **Channel Streaming**: REPEATABLE READ (benchCtx.ChannelStore)

## Performance Optimizations

- **Connection Pool**: 50 connections
- **SQL Functions**: 10x faster (50ms → 5ms)
- **Memory**: Efficient allocations

## Running Benchmarks

```bash
cd internal/benchmarks
go test -bench=. -benchmem -benchtime=2s -timeout=5m .

# Specific suites
go test -bench=BenchmarkAppend_Tiny -benchtime=1s
go test -bench=BenchmarkRead_Small -benchtime=1s
go test -bench=BenchmarkProjection_Tiny -benchtime=1s
```

## Benchmark Structure

- **Append**: Single events, realistic batches (1-12), conditional appends
- **Read**: Simple queries, complex queries, streaming, channel operations
- **Projection**: State reconstruction, streaming projections
- **Business Scenarios**: Course enrollment, concurrent operations, mixed workflows

## Operation Types

- **AppendIf**: Conditional append with business rule validation (checks for conflicts before inserting)
- **Mixed Operations**: Sequential append → query → project operations in single benchmark iteration
- **Business Workflow**: Complete enrollment process with validation and business rule checks
