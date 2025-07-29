# Performance Benchmarks

## Test Environment
- **Platform**: macOS (darwin 23.6.0) with Apple M1 Pro
- **Database**: PostgreSQL with shared connection pool (20 connections)
- **Web Server**: Go HTTP server on port 8080
- **Load Testing**: k6 with various scenarios
- **Test Data**: SQLite-cached datasets (tiny: 5 courses/10 students, small: 1K courses/10K students)

## Benchmark Overview

This project provides comprehensive performance testing for the DCB event sourcing library:

### Benchmark Types
1. **Go Library Benchmarks**: Test core DCB library performance
2. **Web-App Benchmarks**: Test HTTP API performance with load testing

### Test Data
- **Tiny Dataset**: 5 courses, 10 students, 16 enrollments
- **Small Dataset**: 1,000 courses, 10,000 students, 50,000 enrollments

## Go Library Benchmarks



### Append Performance
| Operation | Throughput | Latency | Memory | Allocations |
|-----------|------------|---------|---------|-------------|
| Single (Small) | 939 ops/sec | 1.06ms | 1.9KB | 55 |
| Single (Tiny) | 897 ops/sec | 1.11ms | 1.9KB | 55 |
| 10 Events | 787-594 ops/sec | 1.3-1.7ms | 18KB | 253-254 |
| 100 Events | 282-270 ops/sec | 3.5-3.7ms | 179KB | 2,153-2,154 |
| 1000 Events | 44-42 ops/sec | 22.3-23.9ms | 1.8MB | 21,807-21,815 |
| AppendIf (10) | 4-4 ops/sec | 239-241ms | 20KB | 286 |
| AppendIf (100) | 4-4 ops/sec | 239-241ms | 182KB | 2,181-2,182 |

### Read Performance
| Operation | Throughput | Latency | Memory | Allocations |
|-----------|------------|---------|---------|-------------|
| Simple Read | 2,789-2,680 ops/sec | 358-373μs | 1.4KB | 27 |
| Complex Queries | 2,630-2,576 ops/sec | 366-387μs | 1.4KB | 27 |
| Channel Streaming | 2,515-2,470 ops/sec | 397-404μs | 108KB | 30 |

### Projection Performance
| Operation | Throughput | Latency | Memory | Allocations |
|-----------|------------|---------|---------|-------------|
| Single Projection | 2,755-2,800 ops/sec | 357-415μs | 1.5KB | 31 |
| Multiple Projections | 2,555-2,490 ops/sec | 394-401μs | 2.2KB | 46 |
| Streaming Projections | 2,400-2,560 ops/sec | 390-448μs | 10KB | 41-56 |

## Web-App Load Testing

### Latest Benchmarks (2025-07-29)
- **Tested Endpoints:** append, append-if, read, project
- **All endpoints:** 100% success, no errors

#### Append Benchmark
- **Iterations:** ~15,200
- **Request Rate:** ~62.3 req/s
- **p99 Latency:** 6.2s (threshold crossed)
- **Success:** 100%
- **Use Case:** High-volume event ingestion and data writing

#### Append-If Benchmark
- **Iterations:** ~7,080
- **Request Rate:** ~27 req/s
- **p99 Latency:** 6.3s (threshold crossed)
- **Success:** 100%
- **Use Case:** Conditional event appends with business logic validation

#### Read Benchmark
- **Iterations:** ~432,400
- **Request Rate:** ~1,660 req/s
- **p99 Latency:** 5.7ms (well within threshold)
- **Success:** 100%
- **Use Case:** Event querying and historical data retrieval

#### Project Benchmark
- **Iterations:** ~174,300
- **Request Rate:** ~670 req/s
- **p99 Latency:** 8.2ms (well within threshold)
- **Success:** 100%
- **Use Case:** Real-time state projections and aggregations

### Performance Summary
- **Read operations:** Excellent performance (~1,660 req/s) - suitable for real-time dashboards
- **Projection operations:** Excellent performance (~670 req/s) - suitable for aggregations
- **Basic Append operations (writes):** Good throughput (~62.3 req/s) - suitable for event ingestion
- **Conditional Append operations (writes):** Lower throughput (~27 req/s) - due to DCB concurrency control
- **All operations:** 100% reliability with no errors

### Use Case Recommendations
- **High-frequency reads:** Excellent performance (~1,660 req/s), suitable for real-time dashboards and analytics
- **Event ingestion (writes):** Good performance (~62.3 req/s), suitable for moderate throughput event writing
- **Complex business logic (conditional writes):** Acceptable performance (~27 req/s), suitable for business-critical operations with DCB concurrency control
- **Real-time aggregations:** Excellent performance (~670 req/s), suitable for live dashboards and reporting

## Go Library vs Web-App Comparison

For a comprehensive performance comparison between the Go library and web-app implementations, see [Performance Comparison](performance-comparison.md).

### Use Case Guidance
- **Go Library**: Best for high-performance applications requiring direct database access
- **Web-App API**: Best for distributed systems, microservices, and HTTP-based integrations

### Performance Characteristics
- **Write Operations (Append)**: Go library (939 ops/sec) vs Web-app (~62.3 req/s) - Go library 15x faster
- **Read Operations**: Go library (2,789 ops/sec) vs Web-app (~1,660 req/s) - Go library 1.7x faster
- **Projection Operations**: Go library (2,755 ops/sec) vs Web-app (~670 req/s) - Go library 4.1x faster
- **Reliability**: Both achieve 100% success rates in benchmarks
- **Use Case**: Web-app suitable for HTTP-based integrations, Go library for high-throughput production
