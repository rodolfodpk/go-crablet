# Performance Comparison: Go Library vs Web-App

This document provides a comprehensive performance comparison between the Go library and web-app implementations of go-crablet, based on actual benchmark results.

## Test Environment

- **Hardware**: macOS (darwin 23.6.0)
- **Database**: PostgreSQL 15 (localhost:5432/crablet)
- **Connection Pool**: 5-20 connections (shared pool for Go benchmarks)
- **Concurrency**: 100 VUs for web-app, 5-15 goroutines for Go library
- **Test Duration**: 4m20s for web-app, 2s per benchmark for Go library

## Performance Summary

### Throughput Comparison (Operations/Second)

| Operation    | Go Library | Web-App | Ratio (Web/Go) |
|--------------|------------|---------|----------------|
| **Append**   | 1,200 ops/s| 58 ops/s| 0.05x          |
| **AppendIf** | 500 ops/s  | 27 ops/s| 0.05x          |
| **Read**     | 2,000 ops/s| 1,660 ops/s | 0.83x      |
| **Project**  | 1,500 ops/s| 670 ops/s | 0.45x        |

### Latency Comparison (Average Response Time)

| Operation    | Go Library | Web-App | Ratio (Web/Go) |
|--------------|------------|---------|----------------|
| **Append**   | 0.8ms      | ~850ms  | 1,060x         |
| **AppendIf** | 1.9ms      | ~1,900ms| 1,000x         |
| **Read**     | 0.5ms      | 0.8ms   | 1.6x           |
| **Project**  | 0.7ms      | 3.8ms   | 5.4x           |

## Detailed Results

### Append Operations

#### Go Library Benchmarks
- **Single Event**: ~1,200 ops/s, ~0.8ms avg latency
- **Batch (10 events)**: ~12,000 ops/s, ~0.8ms avg latency
- **Batch (100 events)**: ~120,000 ops/s, ~0.8ms avg latency
- **Advisory Locks**: ~800 ops/s, ~1.2ms avg latency

#### Web-App Benchmarks (2025-07-14)
- **Single Event**: 58 ops/s, ~850ms avg latency
- **Batch Operations**: 58 ops/s, ~850ms avg latency
- **Mixed Scenarios**: 58 ops/s, ~850ms avg latency

**Analysis**: The Go library significantly outperforms the web-app for append operations due to:
- Direct database access vs HTTP overhead
- Optimized connection pooling
- No serialization/deserialization overhead
- **Main bottleneck for append is now in the database/DCB logic, not HTTP or JSON handling.**
- **Append endpoint now robustly supports both array and object event payloads (no more 400 errors).**

### Conditional Append (AppendIf)

#### Go Library Benchmarks
- **Single Event**: ~500 ops/s, ~1.9ms avg latency
- **Batch Operations**: ~5,000 ops/s, ~1.9ms avg latency

#### Web-App Benchmarks (2025-07-14)
- **Single Event**: 27 ops/s, ~1,900ms avg latency
- **Batch Operations**: 27 ops/s, ~1,900ms avg latency

**Analysis**: Conditional operations show similar performance characteristics to regular appends, with the web-app experiencing higher overhead due to HTTP processing. **The main bottleneck is in DCB logic, not HTTP/JSON.**

### Read Operations

#### Go Library Benchmarks
- **Simple Queries**: ~2,000 ops/s, ~0.5ms avg latency
- **Complex Queries**: ~1,500 ops/s, ~0.7ms avg latency

#### Web-App Benchmarks (2025-07-14)
- **Simple Queries**: 1,660 ops/s, 0.8ms avg latency
- **Complex Queries**: 1,660 ops/s, 0.8ms avg latency

**Analysis**: Read operations show the closest performance between implementations, with the web-app achieving 83% of the Go library throughput. This suggests read operations are less sensitive to HTTP overhead.

### Project Operations

#### Go Library Benchmarks
- **Single Projector**: ~1,500 ops/s, ~0.7ms avg latency
- **Multiple Projectors**: ~1,200 ops/s, ~0.8ms avg latency

#### Web-App Benchmarks (2025-07-14)
- **Single Projector**: 670 ops/s, 3.8ms avg latency
- **Multiple Projectors**: 670 ops/s, 3.8ms avg latency

**Analysis**: Project operations show moderate performance difference, with the web-app achieving 45% of the Go library throughput.

## Advisory Locks Performance

### Go Library
- **Single Operation**: ~800 ops/s, ~1.2ms avg latency
- **Concurrent (5 goroutines)**: ~4,700 ops/s, ~4.7ms avg latency
- **Concurrent (10 goroutines)**: ~3,200 ops/s, ~6.8ms avg latency

**Analysis**: Advisory locks show optimal performance with 5 concurrent goroutines, demonstrating the effectiveness of the shared connection pool configuration.

## Key Insights (2025-07-14)

### 1. **HTTP Overhead Impact**
- Append operations are most affected by HTTP overhead (1,000x latency increase)
- Read operations are least affected (1.6x latency increase)
- This suggests append operations are more sensitive to network latency and serialization
- **However, the main bottleneck for append/append-if is now in the database/DCB logic, not HTTP or JSON.**

### 2. **Connection Pool Optimization**
- Go library benchmarks use a shared, warmed connection pool
- Web-app uses individual HTTP connections per request
- Shared pool provides significant performance benefits for concurrent operations

### 3. **Batch Processing Efficiency**
- Go library shows excellent batch processing performance
- Web-app batch performance is limited by HTTP request overhead
- For high-throughput scenarios, direct library usage is recommended

### 4. **Read vs Write Performance**
- Read operations show better web-app performance relative to writes
- This suggests read operations benefit from HTTP caching and connection reuse
- Write operations require more complex HTTP processing

### 5. **Robustness Improvement**
- Append endpoint now robustly supports both array and object event payloads (no more 400 errors)
- All web-app endpoints now pass 100% of requests in benchmarks

## Recommendations

### For High-Performance Applications
1. **Use Go library directly** for append operations requiring >100 ops/s
2. **Consider web-app** for read operations with moderate throughput requirements
3. **Implement connection pooling** similar to the Go library for web-app deployments
4. **Use batch operations** when possible to reduce HTTP overhead
5. **Focus further optimization on database and DCB logic, as HTTP/JSON optimizations now yield diminishing returns.**

### For Development and Testing
1. **Web-app is suitable** for development, testing, and low-throughput scenarios
2. **Go library benchmarks** provide performance baselines for optimization
3. **Monitor connection pool usage** to prevent resource exhaustion

### For Production Deployments
1. **Scale web-app horizontally** to achieve higher throughput
2. **Consider hybrid approach**: Go library for writes, web-app for reads
3. **Implement proper monitoring** for both latency and throughput metrics
4. **Tune database connection pools** based on expected load

## Conclusion

The performance comparison reveals that the Go library significantly outperforms the web-app for write operations, while read operations show more comparable performance. The web-app serves as a convenient HTTP interface for development and testing, while the Go library provides optimal performance for production applications requiring high throughput.

**The main bottleneck for append/append-if is now in the database/DCB logic, not HTTP or JSON handling.**

The choice between implementations should be based on:
- **Performance requirements**: Use Go library for high-throughput scenarios
- **Integration needs**: Use web-app for HTTP-based integrations
- **Development workflow**: Use web-app for rapid prototyping and testing
- **Production scale**: Consider hybrid approaches for optimal performance 