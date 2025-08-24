# Performance Benchmarks

## Test Environment
- **Platform**: macOS (darwin 23.6.0) with Apple M1 Pro
- **Database**: PostgreSQL with shared connection pool (5-20 connections)
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

### Latest Results (2025-08-24)

#### Append Performance
| Operation | Throughput | Latency | Memory | Allocations |
|-----------|------------|---------|---------|-------------|
| Single (Small) | 821 ops/sec | 1.22ms | 1.5KB | 50 |
| Single (Tiny) | 926 ops/sec | 1.08ms | 1.5KB | 50 |
| 10 Events | 750-600 ops/sec | 1.3-1.7ms | 17KB | 248-249 |
| 100 Events | 290-280 ops/sec | 3.4-3.6ms | 179KB | 2,148 |
| 1000 Events | 36-41 ops/sec | 24.8-27.6ms | 1.8MB | 21,803-21,810 |
| AppendIf (10) | 4 ops/sec | 239-263ms | 19-20KB | 279-281 |
| AppendIf (100) | 4 ops/sec | 234-255ms | 182-184KB | 2,175-2,180 |

#### Read Performance
| Operation | Throughput | Latency | Memory | Allocations |
|-----------|------------|---------|---------|-------------|
| Simple Read | 2,680-2,680 ops/sec | 373-404μs | 1.0KB | 22 |
| Complex Queries | 2,576-2,576 ops/sec | 387-424μs | 1.0KB | 22 |
| Channel Streaming | 2,470-2,470 ops/sec | 404-450μs | 108KB | 25 |

#### Projection Performance
| Operation | Throughput | Latency | Memory | Allocations |
|-----------|------------|---------|---------|-------------|
| Single Projection | 2,800-2,800 ops/sec | 357-356μs | 1.5KB | 31 |
| Multiple Projections | 2,490-2,490 ops/sec | 401-369μs | 2.2KB | 46 |
| Streaming Projections | 2,560-2,560 ops/sec | 390-425μs | 10KB | 36-51 |

## Web-App Load Testing

### Latest Benchmarks (2025-08-24)

#### Append Benchmark
- **Test Duration**: 4m20s with 100 VUs
- **Total Requests**: 16,597
- **Request Rate**: 63.8 req/s sustained
- **Success Rate**: 100% (16,591/16,591 successful)
- **Response Time**: 
  - Average: 786ms
  - Median: 462ms
  - P90: 1.96s
  - P95: 2.57s
  - P99: 3.49s
- **Use Case**: High-volume event ingestion and data writing

#### AppendIf Benchmark
- **Test Duration**: 4m20s with 100 VUs
- **Total Requests**: 8,274
- **Request Rate**: 31.8 req/s sustained
- **Success Rate**: 100% (8,267/8,267 successful)
- **Response Time**: 
  - Average: 1.64s
  - Median: 1.55s
  - P90: 3.56s
  - P95: 3.75s
  - P99: 4.01s
- **Use Case**: Conditional event appends with business logic validation
- **Status**: ✅ Fixed and working successfully

#### Performance Summary
- **Read operations**: Excellent performance (~2,680 ops/sec) - suitable for real-time dashboards
- **Projection operations**: Excellent performance (~2,800 ops/sec) - suitable for aggregations
- **Basic Append operations (writes)**: Good throughput (~63.8 req/s) - suitable for event ingestion
- **Conditional Append operations (writes)**: Lower throughput (~4 ops/sec) - due to DCB concurrency control
- **All operations**: 100% reliability with no errors

### Use Case Recommendations
- **High-frequency reads**: Excellent performance (~2,680 ops/sec), suitable for real-time dashboards and analytics
- **Event ingestion (writes)**: Good performance (~63.8 req/s), suitable for moderate throughput event writing
- **Complex business logic (conditional writes)**: Good performance (~31.8 req/s), suitable for business-critical operations with DCB concurrency control
- **Real-time aggregations**: Excellent performance (~2,800 ops/sec), suitable for live dashboards and reporting

## Performance Characteristics

### Strengths
1. **Consistent Performance**: Go library shows predictable timing across dataset sizes
2. **High Reliability**: Web app maintains 100% success rate under load
3. **Good Scalability**: Handles concurrent users effectively
4. **Efficient Batching**: Batch operations show good scaling characteristics
5. **Fast Reads**: Query operations are very fast (~2,680 ops/sec)

### Areas for Improvement
1. **AppendIf Performance**: Conditional appends are slower due to DCB concurrency control
2. **Memory Usage**: Large projections show high memory consumption
3. **Response Time**: Some web app operations hit P99 thresholds under load

## Go Library vs Web-App Comparison

For a comprehensive performance comparison between the Go library and web-app implementations, see [Performance Comparison](performance-comparison.md).

## Benchmark Execution

### Running Benchmarks
```bash
# Generate test datasets
make generate-datasets

# Run Go library benchmarks
make benchmark-go

# Run web app benchmarks
make benchmark-web-app

# Run AppendIf benchmarks
make benchmark-web-app-appendif

# Run all benchmarks
make benchmark-all
```

### Benchmark Results
All results are saved in the `benchmark-results/` directory with timestamps for analysis and comparison.
