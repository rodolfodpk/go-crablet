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
- **Request Rate:** ~58 req/s
- **p99 Latency:** 6.2s (threshold crossed)
- **Success:** 100%
- **Use Case:** High-volume event streaming and data ingestion

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
- **Read/Projection operations:** Excellent performance (1,600+ req/s)
- **Basic Append operations:** Good throughput (~58 req/s)
- **Conditional Append operations:** Lower throughput (~27 req/s) due to DCB logic complexity
- **All operations:** 100% reliability with no errors

### Use Case Recommendations
- **High-frequency reads:** Excellent performance, suitable for real-time dashboards
- **Event streaming:** Good performance for moderate throughput requirements
- **Complex business logic:** Acceptable performance with DCB concurrency control

## Go Library vs Web-App Comparison

For a comprehensive performance comparison between the Go library and web-app implementations, see [Performance Comparison](performance-comparison.md).

### Use Case Guidance
- **Go Library**: Best for high-performance applications requiring direct database access
- **Web-App API**: Best for distributed systems, microservices, and HTTP-based integrations

### Performance Characteristics
- **Write Operations**: Go library significantly outperforms web-app (1,028x lower latency)
- **Read Operations**: More comparable performance (1.6x latency difference)
- **Reliability**: Both achieve 100% success rates in benchmarks
- Web-app suitable for development/testing, Go library recommended for high-throughput production
