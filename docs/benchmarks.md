# Performance Benchmarks

## Test Environment
- **Platform**: macOS (darwin 23.6.0) with Apple M1 Pro
- **Database**: PostgreSQL with shared connection pool (20 connections)
- **Web Server**: Go HTTP server on port 8080
- **Load Testing**: k6 with various scenarios
- **Test Data**: SQLite-cached datasets (tiny: 5 courses/10 students, small: 1K courses/10K students)

## Go Library Benchmarks

### Advisory Lock Performance
| Operation | Throughput | Latency | Memory | Allocations |
|-----------|------------|---------|---------|-------------|
| Single (Small) | 932 ops/sec | 1.18ms | 4KB | 85 |
| Single (Tiny) | 892 ops/sec | 1.15ms | 4KB | 85 |
| 5 Goroutines | 273-315 ops/sec | 3.9ms | 21KB | 449 |
| 8 Goroutines | 162-212 ops/sec | 5.8-6.8ms | 35KB | 750 |
| 10 Goroutines | 86-163 ops/sec | 6.1-7.2ms | 50KB | 1,100 |
| 20 Goroutines | 37-84 ops/sec | 13.8-24.3ms | 87KB | 1,815 |
| Batch 10 | 392-484 ops/sec | 1.3-1.5ms | 4KB | 85 |
| Batch 100 | 201-216 ops/sec | 2.8-3.1ms | 4KB | 85 |
| Batch 1000 | 33-42 ops/sec | 29.6-33.9ms | 4KB | 85 |

### Append Performance
| Operation | Throughput | Latency | Memory | Allocations |
|-----------|------------|---------|---------|-------------|
| Single (Small) | 1,058 ops/sec | 1.10ms | 1.9KB | 56 |
| Single (Tiny) | 957 ops/sec | 1.08ms | 1.9KB | 56 |
| 10 Events | 804-958 ops/sec | 1.2-1.3ms | 18KB | 560 |
| 100 Events | 559-573 ops/sec | 3.0-4.2ms | 180KB | 5,600 |
| 1000 Events | 100 ops/sec | 22.0-22.2ms | 1.8MB | 56,000 |
| AppendIf (10) | 3-4 ops/sec | 171-180ms | 12MB | 156K |
| AppendIf (100) | 3 ops/sec | 178-180ms | 12MB | 156K |

### Read Performance
| Operation | Throughput | Latency | Memory | Allocations |
|-----------|------------|---------|---------|-------------|
| Single Read | 2,757-3,328 ops/sec | 350-380μs | 1.4KB | 27-30 |
| Complex Queries | 2,769-3,328 ops/sec | 350-380μs | 1.4KB | 27-30 |
| Channel Streaming | 2,844-3,105 ops/sec | 350-390μs | 1.4KB | 27-30 |

### Projection Performance
| Operation | Throughput | Latency | Memory | Allocations |
|-----------|------------|---------|---------|-------------|
| Single Projection | 3,061-3,456 ops/sec | 350-380μs | 1.5-11KB | 31-56 |
| Multiple Projections | 3,061-3,456 ops/sec | 350-380μs | 1.5-11KB | 31-56 |
| Streaming Projections | 2,998-3,351 ops/sec | 350-380μs | 1.5-11KB | 31-56 |

## Web-App Load Testing

### Quick Test (Basic Functionality)
- **6,389 iterations** completed with **0 errors**
- **637.4 iterations/second** throughput
- **1,275 requests/second** HTTP throughput
- **1.47ms average response time**
- **100% success rate**

### Append Performance Benchmark
- **16,227 iterations** completed successfully
- **62.4 requests/second** HTTP throughput
- **805.5ms average response time**
- **100% append success rate**
- **Batch Operations**: 6,043 batch appends (23.2/s)
- **Conditional Operations**: 4,058 conditional appends (15.6/s)

### Isolation Level Benchmark
- **14,216 iterations** completed with **0 errors**
- **54.7 requests/second** HTTP throughput
- **106.6ms average response time**
- **100% success rate**

**Isolation Level Performance**:
- Read Committed: 4,760 appends (18.3 req/s)
- Repeatable Read: 4,781 appends (18.4 req/s)
- Serializable: 4,675 appends (18.0 req/s)

### Concurrency Test
- **2,297 iterations** completed with **0 errors**
- **55.1 requests/second** HTTP throughput
- **226.9ms average response time**
- **100% success rate**

**Operation Mix**:
- Simple Appends: 4,594 operations (100% success rate)
- Conditional Appends: 4,592 operations (100% success rate)
- Read Operations: 99% success rate

### Advisory Lock Benchmark
- **45,300 iterations** completed with **0 errors**
- **215.7 requests/second** HTTP throughput
- **100% success rate** for advisory lock operations

### AppendIf (Conditional Append) Benchmark
- **7,795 iterations** completed with **0 errors**
- **29.9 requests/second** HTTP throughput
- **1.75s average response time**
- **100% success rate** for all conditional operations

## Performance Summary

### Throughput Hierarchy (Fastest to Slowest)
1. Read/Projection: 3,000+ ops/sec
2. Basic Append: 900+ ops/sec
3. Advisory Locks: 900+ ops/sec
4. HTTP API: 1,275+ req/sec
5. Concurrent Locks: 200+ ops/sec
6. Conditional Append: 3-4 ops/sec

### Key Insights
- **Optimal Concurrency**: 5 goroutines (4.7ms latency)
- **Conditional Operations**: 100-150x slower but required for business logic
- **HTTP API Overhead**: ~1.5ms base latency vs ~1.1ms for direct library calls
- **Memory Scaling**: Linear with event count (~1.8MB for 1000 events)
- **Connection Pool**: 20 connections optimal for concurrent benchmarks
