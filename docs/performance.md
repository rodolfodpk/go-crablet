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
| **Course Registration** | 10 | 423 ops/sec | 2.9ms | 26.2KB |
| **Business Workflow** | 1 | 97 ops/sec | 12.4ms | 10.5KB |
| **Mixed Operations** | 1 | 97 ops/sec | 12.4ms | 10.5KB |

## Isolation Levels

- **Simple Append**: READ COMMITTED
- **AppendIf**: REPEATABLE READ (for DCB concurrency control)
- **Queries**: READ COMMITTED
- **Projections**: READ COMMITTED

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
