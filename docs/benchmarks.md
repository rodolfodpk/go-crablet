# Performance Benchmarks

## Test Environment
- **Platform**: macOS (darwin 23.6.0) with Apple M1 Pro
- **Database**: PostgreSQL with shared connection pool (20 connections)
- **Web Server**: Go HTTP server on port 8080
- **Load Testing**: k6 with various scenarios
- **Test Data**: SQLite-cached datasets (tiny: 5 courses/10 students, small: 1K courses/10K students)

## Go Library Benchmarks

### Advisory Lock Performance (Simple Resource Locking)
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

**Note**: These benchmarks test advisory locks without DCB conditions (1 I/O operation). When combined with DCB conditions, performance equals AppendIf (2 I/O operations).

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

### Latest Benchmarks (2025-07-14)
- **Tested Endpoints:** append, append-if, read, project
- **All endpoints:** 100% success, no errors

#### Append Benchmark
- **Iterations:** ~15,200
- **Request Rate:** ~58 req/s
- **p99 Latency:** 6.2s (threshold crossed)
- **Success:** 100%
- **Note:** Throughput is solid, but latency spikes at high concurrency. Main bottleneck is in database/DCB logic, not HTTP or JSON handling.

#### Append-If Benchmark
- **Iterations:** ~7,080
- **Request Rate:** ~27 req/s
- **p99 Latency:** 6.3s (threshold crossed)
- **Success:** 100%
- **Note:** Conditional appends are slower, as expected. Bottleneck is in DCB logic.

#### Read Benchmark
- **Iterations:** ~432,400
- **Request Rate:** ~1,660 req/s
- **p99 Latency:** 5.7ms (well within threshold)
- **Success:** 100%
- **Note:** Read performance is excellent.

#### Project Benchmark
- **Iterations:** ~174,300
- **Request Rate:** ~670 req/s
- **p99 Latency:** 8.2ms (well within threshold)
- **Success:** 100%
- **Note:** Projection queries are fast and scale well.

---

### Robustness Improvement
- The append endpoint now robustly supports both array and object event payloads (thanks to handler fix).
- All web-app endpoints now pass 100% of requests in benchmarks.
- No more 400 errors or failed requests due to payload format.

---

### Throughput Hierarchy (Fastest to Slowest)
1. Read/Projection: 1,600+ ops/sec (read), 670+ ops/sec (project)
2. Basic Append: ~58 ops/sec
3. Conditional Append: ~27 ops/sec

### Key Insights (2025-07-14)
- **Append/Append-If bottleneck is in database/DCB logic, not HTTP/JSON.**
- **Read and project endpoints are extremely fast and scale well.**
- **Handler now robustly supports both array and object event payloads.**
- **No regressions or errors in latest benchmarks.**
- **HTTP/JSON optimizations yield diminishing returns; focus future tuning on database and DCB logic.**

## Go Library vs Web-App Comparison

For a comprehensive performance comparison between the Go library and web-app implementations, see [Performance Comparison](performance-comparison.md).

**Key Findings:**
- Go library significantly outperforms web-app for write operations (1,028x lower latency)
- Read operations show more comparable performance (1.6x latency difference)
- Web-app suitable for development/testing, Go library recommended for high-throughput production
