# Performance Benchmarks

This document contains performance benchmark results for the go-crablet event sourcing library, including both internal library benchmarks and web-app load testing results.

## Test Environment

- **Platform**: macOS (darwin 23.6.0) with Apple M1 Pro
- **Database**: PostgreSQL with connection pool (5-20 connections)
- **Web Server**: Go HTTP server on port 8080
- **Load Testing**: k6 with various scenarios
- **Go Version**: 1.24.4
- **Test Data**: SQLite-cached datasets for fast benchmark execution

## Test Data System

### SQLite Caching Architecture
The benchmark system uses SQLite to cache pre-generated test datasets, providing:
- **Fast Setup**: No expensive dataset generation during benchmarks
- **Consistent Data**: Same test datasets across all benchmark types
- **Efficient Loading**: Cached data loads much faster than dynamic generation

### Dataset Sizes
- **"tiny"**: 5 courses, 10 students, 16 enrollments
- **"small"**: 1,000 courses, 10,000 students, 49,869 enrollments

### Test Data Workflow
1. **Generate Datasets**: `cd internal/benchmarks/tools && go run prepare_datasets_main.go`
2. **Cache Storage**: Datasets stored in `cache/benchmark_datasets.db`
3. **Benchmark Loading**: Both Go and web-app benchmarks load from cache
4. **Fast Execution**: No timeout issues, consistent performance

## Internal Library Benchmarks

### Append Performance (Latest Results - July 2025)

#### Single Event Appends
- **Small Dataset**: **2,192 ops/sec** (1.09ms per operation)
- **Tiny Dataset**: **2,476 ops/sec** (1.05ms per operation)
- **Memory Usage**: ~1.9KB per operation, 53 allocations

#### Batch Append Performance
- **Batch Size 10**: ~1,600-2,000 ops/sec (1.2-1.4ms per batch)
- **Batch Size 100**: ~1,000-1,200 ops/sec (3.3-4.2ms per batch)
- **Batch Size 1000**: ~100 ops/sec (22-23ms per batch)
- **Memory Scaling**: Linear with batch size (~1.7MB for 1000 events)

#### Conditional Append (AppendIf)
- **Small batches**: ~8-9 ops/sec (260-330ms per operation)
- **With conflicts**: ~8 ops/sec (250-310ms per operation)
- **Overhead**: Significant due to version checking and conflict resolution

### Read Performance
- **Single Read**: ~400-700 ops/sec (1.4-5.1ms per operation)
- **Batch Read**: ~6-7 ops/sec (340-355ms per operation)
- **Channel Streaming**: Similar performance to regular reads
- **Memory Usage**: ~1-2MB for large datasets

### Projection Performance
- **Single Projection**: ~5,000-6,500 ops/sec (0.4-0.6ms per operation)
- **Large Projections**: ~7-8 ops/sec (290-320ms per operation)
- **Memory Usage**: ~100-140MB for large projections

### Memory and Resource Usage
- **Single Operations**: ~1-2KB per operation
- **Batch Operations**: ~1.7MB for 1000-event batches
- **Connection Pool**: Efficient utilization with multiple pools
- **No Memory Leaks**: Clean resource management observed

## Web-App Load Testing Results

### 1. Quick Test (Basic Functionality)
**Scenario**: 2 VUs for 10 seconds, basic append/read operations

**Results**:
- ✅ **6,372 iterations** completed with **0 errors**
- ✅ **635.4 iterations/second** throughput
- ✅ **1,271 requests/second** HTTP throughput
- ✅ **1.51ms average response time**
- ✅ **100% success rate** for all operations

**Performance Metrics**:
- **HTTP Response Time**: avg=1.51ms, p90=2.05ms, p95=2.24ms
- **Iteration Duration**: avg=3.13ms, p90=3.91ms, p95=4.28ms
- **Data Throughput**: 232 kB/s received, 343 kB/s sent

### 2. Append Performance Benchmark (with SQLite Test Data)
**Scenario**: 100 VUs for 4m20s, various append operations with cached test data

**Results**:
- ✅ **5,120+ iterations** completed successfully
- ✅ **121+ requests/second** HTTP throughput
- ✅ **6.73ms average response time**
- ✅ **100% append success rate** for valid operations
- ✅ **SQLite test data loaded**: 5 courses, 10 students, 16 enrollments

**Performance Breakdown**:
- **HTTP Response Time**: avg=6.73ms, p90=14.01ms, p95=15.94ms, p99=17.48ms
- **Data Throughput**: 21 kB/s received, 21 kB/s sent
- **All Thresholds Passed**: Error rate < 10%, response time < 2000ms

### 3. Isolation Level Benchmark
**Scenario**: 20 VUs for 4m20s, testing different isolation levels

**Results**:
- ✅ **8,375 iterations** completed with **0 errors**
- ✅ **32.2 iterations/second** throughput
- ✅ **32.2 requests/second** HTTP throughput
- ✅ **252.72ms average response time**
- ✅ **100% success rate** for all operations

**Isolation Level Performance**:
- **Read Committed**: 10.8 req/s
- **Repeatable Read**: 10.5 req/s
- **Serializable**: 10.9 req/s
- **HTTP Response Time**: avg=252.72ms, p90=877ms, p95=1.39s

**Key Insight**: All isolation levels perform similarly, with Serializable slightly outperforming others.

### 4. Concurrency Test
**Scenario**: 20 VUs for 4m10s, mixed operations with conflicts

**Results**:
- ✅ **1,217 iterations** completed with **0 errors**
- ✅ **4.9 iterations/second** throughput
- ✅ **29.1 requests/second** HTTP throughput
- ✅ **474.97ms average response time**
- ✅ **87.5% check success rate** (some duration thresholds exceeded)

**Operation Mix**:
- **Simple Appends**: 100% success rate
- **Conditional Appends**: 100% success rate with proper conflict handling
- **Batch Operations**: Reliable performance
- **Read Operations**: 99% success rate for duration checks

**Performance Metrics**:
- **HTTP Response Time**: avg=474.97ms, p90=1.65s, p95=1.95s
- **Iteration Duration**: avg=3.15s, p90=4.64s, p95=4.89s
- **Conflict Resolution**: 100% success rate

## Performance Characteristics

### Strengths
1. **Excellent Single Operations**: 2,000+ ops/sec for individual events
2. **Good Batch Performance**: Scales well up to medium batch sizes (100 events)
3. **Fast Response Times**: 1-2ms for individual operations
4. **Efficient Memory Usage**: Reasonable allocation patterns
5. **Stable Performance**: Consistent results across test runs
6. **Fast Setup**: SQLite caching eliminates benchmark timeouts

### Performance Considerations
1. **Conditional Append Overhead**: Significant performance impact due to version checking
2. **Large Batch Operations**: Performance degrades with very large batches (1000+ events)
3. **Web App Performance**: Excellent performance with SQLite test data system
4. **Benchmark Efficiency**: Fast execution with cached datasets

### System Capabilities
- ✅ **Fast Single Operations**: Excellent performance for individual events
- ✅ **Good Batch Handling**: Efficient processing of medium-sized batches
- ✅ **Memory Efficient**: Optimized memory usage patterns
- ✅ **Connection Management**: Efficient database connection pooling
- ✅ **No Deadlocks**: Clean execution without blocking issues
- ✅ **Fast Benchmark Setup**: SQLite caching system eliminates timeouts

## Configuration Recommendations

### For Production Use
1. **Batch Sizes**: Use batches of 10-100 events for optimal performance
2. **Conditional Appends**: Consider performance impact when using AppendIf operations
3. **Connection Pool**: Current 5-20 connection pool works well for moderate loads
4. **Monitoring**: Monitor response times and adjust batch sizes accordingly

### Performance Tuning
1. **Avoid Large Batches**: Keep batch sizes under 1000 events for best performance
2. **Conditional Operations**: Use sparingly due to significant overhead
3. **Memory Monitoring**: Monitor memory usage for large projection operations
4. **Connection Limits**: Consider increasing pool size for high-concurrency scenarios

### Benchmark Testing
1. **Use SQLite Cache**: Pre-generate datasets for fast benchmark execution
2. **Dataset Sizes**: Use "tiny" for quick tests, "small" for comprehensive testing
3. **Makefile Targets**: Use `make benchmark-go` and `make benchmark-web-app` for easy execution
4. **Test Data Loading**: Web-app automatically loads test data via `/load-test-data` endpoint

## Summary

The go-crablet library demonstrates excellent performance characteristics for typical event sourcing workloads:

- **Single Operations**: 2,000+ ops/sec with 1-2ms latency
- **Batch Operations**: Good performance up to medium batch sizes
- **Memory Efficiency**: Optimized allocation patterns
- **Reliability**: Stable performance across different operation types
- **Fast Testing**: SQLite caching system enables efficient benchmark execution

The library is well-suited for real-time event processing with fast individual operations and efficient batch handling. The new SQLite test data system provides consistent, fast benchmark execution for both Go library and web-app testing. 