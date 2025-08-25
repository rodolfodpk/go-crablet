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

| Operation | Users | Batch Size | Throughput | Latency | Memory | Allocations |
|-----------|-------|------------|------------|---------|---------|-------------|
| **Append** | 1 | - | 2,535 ops/sec | 1.02ms | 2.5KB | 51 |
| **Append** | 10 | - | 835 ops/sec | 2.77ms | 26.1KB | 530 |
| **Append** | 100 | - | 198 ops/sec | 13.7ms | 269.5KB | 5,543 |
| **Read** | 1 | - | 1,047 ops/sec | 2.02ms | 1.1KB | 25 |
| **Read** | 10 | - | 519 ops/sec | 4.21ms | 11.8KB | 270 |
| **Read** | 100 | - | 50 ops/sec | 46.6ms | 120.7KB | 2,853 |
| **Projection** | 1 | - | 1,180 ops/sec | 1.99ms | 2.3KB | 45 |
| **Projection** | 10 | - | 548 ops/sec | 4.44ms | 23.6KB | 470 |
| **Projection** | 100 | - | 52 ops/sec | 48.8ms | 246.2KB | 4,855 |
| **AppendIf** | - | 1 | 24 ops/sec | 97.3ms | 3.9KB | 79 |
| **AppendIf** | - | 5 | 24 ops/sec | 104.3ms | 12.3KB | 164 |
| **AppendIf** | - | 12 | 22 ops/sec | 102.1ms | 22.6KB | 308 |

**Scaling Patterns**:
- **1 User**: Best performance, minimal resource usage
- **10 Users**: Moderate performance, 10x resource increase
- **100 Users**: Lower performance, 100x resource increase

**Performance Impact**:
- **Append**: 2,535 → 198 ops/sec (12.8x slower with 100 users)
- **Read**: 1,047 → 50 ops/sec (20.9x slower with 100 users)  
- **Projection**: 1,180 → 52 ops/sec (22.7x slower with 100 users)
- **AppendIf**: Consistent ~24 ops/sec regardless of batch size

**Batch Size Explanation**:
- **AppendIf_1**: Process 1 event at a time (e.g., enroll 1 student in 1 course)
- **AppendIf_5**: Process 5 events at a time (e.g., enroll 1 student in 5 courses)  
- **AppendIf_12**: Process 12 events at a time (e.g., enroll 1 student in 12 courses)

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
