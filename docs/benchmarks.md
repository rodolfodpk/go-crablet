# Performance Benchmarks

This document contains performance benchmark results for the go-crablet event sourcing library, including both internal library benchmarks and web-app load testing results.

## Test Environment

- **Platform**: macOS (darwin 23.6.0) with Apple M1 Pro
- **Database**: PostgreSQL with shared connection pool (10 connections)
- **Web Server**: Go HTTP server on port 8080
- **Load Testing**: k6 with various scenarios
- **Go Version**: 1.24.5
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

### Advisory Lock Performance (Latest Results - July 2025)

#### Single Advisory Lock Operations
- **Small Dataset**: **932 ops/sec** (1.18ms per operation)
- **Tiny Dataset**: **892 ops/sec** (1.15ms per operation)
- **Memory Usage**: ~4KB per operation, 85 allocations

#### Concurrent Advisory Lock Operations
- **5 Goroutines**: **273-315 ops/sec** (3.9-3.9ms per operation)
- **8 Goroutines**: **162-212 ops/sec** (5.8-6.8ms per operation)
- **10 Goroutines**: **86-163 ops/sec** (6.1-7.2ms per operation)
- **20 Goroutines**: **37-84 ops/sec** (13.8-24.3ms per operation)
- **Memory Usage**: ~21-87KB per operation, 449-1815 allocations

#### Advisory Lock Batch Operations
- **10 Events**: **392-484 ops/sec** (1.3-1.5ms per operation)
- **100 Events**: **201-216 ops/sec** (2.8-3.1ms per operation)
- **1000 Events**: **33-42 ops/sec** (29.6-33.9ms per operation)

#### Advisory Lock vs Regular Operations
- **Advisory Lock Overhead**: ~2.3ms additional latency
- **Concurrency Control**: Effective resource locking with reasonable performance impact

### Append Performance (Latest Results - July 2025)

#### Single Event Appends
- **Small Dataset**: **1,058 ops/sec** (1.10ms per operation)
- **Tiny Dataset**: **957 ops/sec** (1.08ms per operation)
- **Memory Usage**: ~1.9KB per operation, 56 allocations

#### Multiple Events Performance
- **10 Events**: **804-958 ops/sec** (1.2-1.3ms per append call)
- **100 Events**: **559-573 ops/sec** (3.0-4.2ms per append call)
- **1000 Events**: **100 ops/sec** (22.0-22.2ms per append call)
- **Memory Scaling**: Linear with event count (~1.8MB for 1000 events)

#### Conditional Append (AppendIf) - Performance Consideration
- **10 Events**: **3-4 ops/sec** (171-180ms per operation)
- **100 Events**: **3 ops/sec** (178-180ms per operation)
- **1000 Events**: **3 ops/sec** (180ms per operation)
- **Performance Note**: Conditional appends are **100-150x slower** due to complex concurrency control
- **Use Case**: Only use when strict concurrency control is required

### Read Performance
- **Single Read**: **2,757-3,328 ops/sec** (350-380μs per operation)
- **Complex Queries**: **2,769-3,328 ops/sec** (350-380μs per operation)
- **Channel Streaming**: **2,844-3,105 ops/sec** (350-390μs per operation)
- **Memory Usage**: ~1.4KB per operation, 27-30 allocations

### Projection Performance
- **Single Projection**: **3,061-3,456 ops/sec** (350-380μs per operation)
- **Multiple Projections**: **3,061-3,456 ops/sec** (350-380μs per operation)
- **Streaming Projections**: **2,998-3,351 ops/sec** (350-380μs per operation)
- **Memory Usage**: ~1.5-11KB per operation, 31-56 allocations

### Memory and Resource Usage
- **Single Operations**: ~1-4KB per operation
- **Multiple Events**: ~1.8MB for 1000 events in single append call
- **Connection Pool**: Efficient shared pool with 10 connections
- **No Memory Leaks**: Clean resource management observed

## Web-App Load Testing Results

### 1. Quick Test (Basic Functionality) - Latest Results
**Scenario**: 2 VUs for 10 seconds, basic append/read operations

**Results**:
- ✅ **6,389 iterations** completed with **0 errors**
- ✅ **637.4 iterations/second** throughput
- ✅ **1,275 requests/second** HTTP throughput
- ✅ **1.47ms average response time**
- ✅ **100% success rate** for all operations

**Performance Metrics**:
- **HTTP Response Time**: avg=1.47ms, p90=2.0ms, p95=2.42ms
- **Iteration Duration**: avg=3.12ms, p90=4.06ms, p95=4.73ms
- **Data Throughput**: 232 kB/s received, 344 kB/s sent

### 2. Append Performance Benchmark (with SQLite Test Data) - Latest Results
**Scenario**: 100 VUs for 4m20s, various append operations with cached test data

**Results**:
- ✅ **16,227 iterations** completed successfully
- ✅ **62.4 requests/second** HTTP throughput
- ✅ **805.5ms average response time**
- ✅ **100% append success rate** for valid operations
- ✅ **SQLite test data loaded**: 5 courses, 10 students, 16 enrollments

**Performance Breakdown**:
- **HTTP Response Time**: avg=805.5ms, p90=1.97s, p95=2.64s, p99=5.15s
- **Data Throughput**: 11 kB/s received, 198 kB/s sent
- **Batch Operations**: 6,043 batch appends (23.2/s)
- **Conditional Operations**: 4,058 conditional appends (15.6/s)
- **All Thresholds Passed**: Error rate < 10%, response time < 2000ms

### 3. Isolation Level Benchmark - Latest Results
**Scenario**: 20 VUs for 4m20s, testing different isolation levels

**Results**:
- ✅ **14,216 iterations** completed with **0 errors**
- ✅ **54.7 iterations/second** throughput
- ✅ **54.7 requests/second** HTTP throughput
- ✅ **106.6ms average response time**
- ✅ **100% success rate** for all operations

**Isolation Level Performance**:
- **Read Committed**: 4,760 appends (18.3 req/s)
- **Repeatable Read**: 4,781 appends (18.4 req/s)
- **Serializable**: 4,675 appends (18.0 req/s)
- **HTTP Response Time**: avg=106.6ms, p90=369ms, p95=515ms

**Key Insight**: All isolation levels perform similarly, with Repeatable Read slightly outperforming others.

### 4. Concurrency Test - Latest Results
**Scenario**: 20 VUs for 4m10s, mixed operations with conflicts

**Results**:
- ✅ **2,297 iterations** completed with **0 errors**
- ✅ **9.2 iterations/second** throughput
- ✅ **55.1 requests/second** HTTP throughput
- ✅ **226.9ms average response time**
- ✅ **100% success rate** for all operations

**Operation Mix**:
- **Simple Appends**: 4,594 operations (100% success rate)
- **Conditional Appends**: 4,592 operations (100% success rate with proper conflict handling)
- **Multiple Events Operations**: Reliable performance
- **Read Operations**: 99% success rate for duration checks

**Performance Metrics**:
- **HTTP Response Time**: avg=226.9ms, p90=747ms, p95=949ms
- **Iteration Duration**: avg=1.66s, p90=2.58s, p95=2.87s
- **Conflict Resolution**: 100% success rate

### 5. Advisory Lock Benchmark - Latest Results
**Scenario**: 50 VUs for 3m30s, advisory lock operations with resource contention

**Results**:
- ✅ **45,300 iterations** completed with **0 errors**
- ✅ **215.7 iterations/second** throughput
- ✅ **215.7 requests/second** HTTP throughput
- ✅ **Fast execution** with optimized logging
- ✅ **100% success rate** for advisory lock operations

**Advisory Lock Performance**:
- **Single Operations**: Fast execution with proper resource locking
- **Batch Operations**: Efficient processing of multiple events with locks
- **Resource Contention**: Effective handling of concurrent resource access
- **Mixed Scenarios**: Balanced performance across different operation types

**Performance Characteristics**:
- **Advisory Lock Overhead**: Minimal impact on HTTP response times
- **Concurrency Control**: Effective resource locking with reasonable performance
- **Scalability**: Good performance up to 50 concurrent virtual users

### 6. AppendIf (Conditional Append) Benchmark - Latest Results
**Scenario**: 100 VUs for 4m20s, conditional append operations with complex conditions

**Results**:
- ✅ **7,795 iterations** completed with **0 errors**
- ✅ **29.9 iterations/second** throughput
- ✅ **29.9 requests/second** HTTP throughput
- ✅ **1.75s average response time**
- ✅ **100% success rate** for all conditional operations

**Conditional Append Performance**:
- **Single Events with Conditions**: 100% success rate
- **Batch Operations with Conditions**: Reliable performance
- **Complex Conditions**: Effective handling of multiple event types and tags
- **Concurrency Errors**: 0% error rate (proper conflict handling)

**Performance Metrics**:
- **HTTP Response Time**: avg=1.75s, p90=3.8s, p95=4.03s
- **Iteration Duration**: avg=1.81s, p90=3.86s, p95=4.09s
- **Conditional Operations**: 7,795 operations (29.9/s)
- **Performance Note**: Conditional appends are significantly slower due to complex concurrency control

## Performance Characteristics

### Strengths
1. **Excellent Single Operations**: 900-1,000+ ops/sec for individual events
2. **Good Multiple Events Performance**: Scales well up to medium event counts (100 events)
3. **Fast Response Times**: 1-2ms for individual operations, 1.5ms for HTTP API
4. **Efficient Memory Usage**: Reasonable allocation patterns
5. **Stable Performance**: Consistent results across test runs
6. **Fast Setup**: SQLite caching eliminates benchmark timeouts
7. **Effective Advisory Locks**: Good performance for resource locking scenarios
8. **HTTP API Performance**: 1,275+ requests/second for basic operations
9. **Load Testing Capability**: Handles 50-100 concurrent virtual users effectively

### Performance Considerations
1. **Conditional Append Overhead**: Significant performance impact (100-150x slower) due to complex concurrency control
2. **Large Event Groups**: Performance degrades with very large event counts (1000+ events)
3. **Concurrency Scaling**: Advisory locks show reasonable performance up to 8-10 concurrent goroutines
4. **Web App Performance**: Excellent performance with SQLite test data system and optimized logging
5. **HTTP API Overhead**: ~1.5ms base latency for HTTP operations vs ~1.1ms for direct library calls
6. **Conditional Append HTTP**: ~1.75s average response time for complex conditional operations
7. **Benchmark Efficiency**: Fast execution with cached datasets and optimized logging

### System Capabilities
- ✅ **Fast Single Operations**: Excellent performance for individual events
- ✅ **Good Multiple Events Handling**: Efficient processing of medium-sized event groups
- ✅ **Memory Efficient**: Optimized memory usage patterns
- ✅ **Connection Management**: Efficient shared database connection pooling
- ✅ **No Deadlocks**: Clean execution without blocking issues
- ✅ **Fast Benchmark Setup**: SQLite caching system eliminates timeouts
- ✅ **Advisory Lock Support**: Effective resource locking with reasonable performance
- ✅ **Concurrency Control**: Stable performance under concurrent load
- ✅ **HTTP API Performance**: 1,275+ requests/second for basic operations
- ✅ **Load Testing Ready**: Handles 50-100 concurrent virtual users effectively
- ✅ **Optimized Logging**: Debug logging removed for faster benchmark execution

## Configuration Recommendations

### For Production Use
1. **Event Group Sizes**: Use 10-100 events per append call for optimal performance
2. **Conditional Appends**: Use sparingly due to significant performance impact (100-150x slower)
3. **Advisory Locks**: Use for resource locking with 5-8 concurrent operations for best performance
4. **Connection Pool**: Current 10-connection shared pool works well for moderate loads
5. **HTTP API**: Expect ~1.5ms base latency for HTTP operations vs ~1.1ms for direct library calls
6. **Monitoring**: Monitor response times and adjust event group sizes accordingly

### Performance Tuning
1. **Avoid Large Event Groups**: Keep event counts under 1000 per append call for best performance
2. **Conditional Operations**: Use only when strict concurrency control is required
3. **Advisory Lock Concurrency**: Limit to 5-8 concurrent operations for optimal performance
4. **Memory Monitoring**: Monitor memory usage for large projection operations
5. **Connection Limits**: Consider increasing pool size for high-concurrency scenarios (tested up to 100 VUs)
6. **HTTP API Optimization**: Use direct library calls for high-frequency operations to avoid HTTP overhead
7. **Conditional Append HTTP**: Expect ~1.75s response times for complex conditional operations

### Benchmark Testing
1. **Use SQLite Cache**: Pre-generate datasets for fast benchmark execution
2. **Dataset Sizes**: Use "tiny" for quick tests, "small" for comprehensive testing
3. **Makefile Targets**: Use `make benchmark-go` and `make benchmark-web-app` for easy execution
4. **Test Data Loading**: Web-app automatically loads test data via `/load-test-data` endpoint
5. **Optimized Logging**: Debug logging removed for faster benchmark execution
6. **Comprehensive Coverage**: All benchmark types now available (Go library, web-app, advisory locks, conditional appends)
7. **Performance Validation**: Both direct library and HTTP API performance measured

## Summary

The go-crablet library demonstrates excellent performance characteristics for typical event sourcing workloads:

- **Single Operations**: 900-1,000+ ops/sec with 1-1.2ms latency
- **Multiple Events**: Good performance up to medium event counts per append call
- **Read Operations**: 2,700-3,300+ ops/sec with 350-380μs latency
- **Projection Performance**: 3,000-3,500+ ops/sec with 350-380μs latency
- **Advisory Locks**: 900+ ops/sec single, 200-300+ ops/sec concurrent (5-8 goroutines)
- **HTTP API Performance**: 1,275+ requests/second for basic operations
- **Memory Efficiency**: Optimized allocation patterns
- **Reliability**: Stable performance across different operation types
- **Fast Testing**: SQLite caching system enables efficient benchmark execution

### Performance Hierarchy (Fastest to Slowest)
1. **Read/Projection Operations**: ~350μs (3,000+ ops/sec)
2. **Basic Append Operations**: ~1.1ms (900+ ops/sec)
3. **Advisory Lock Operations**: ~1.2ms (900+ ops/sec)
4. **HTTP API Operations**: ~1.5ms (1,275+ req/sec)
5. **Concurrent Advisory Locks**: ~4-7ms (200-300 ops/sec)
6. **Conditional Appends (Library)**: ~170-180ms (3-4 ops/sec) - Use sparingly
7. **Conditional Appends (HTTP)**: ~1.75s (29.9 req/sec) - Use sparingly

### Benchmark Coverage
- ✅ **Go Library Benchmarks**: Direct performance measurement of core functionality
- ✅ **Web-App HTTP Benchmarks**: End-to-end HTTP API performance testing
- ✅ **Advisory Lock Benchmarks**: Resource locking performance validation
- ✅ **Conditional Append Benchmarks**: Complex concurrency control testing
- ✅ **Load Testing**: Up to 100 concurrent virtual users
- ✅ **Optimized Execution**: Debug logging removed for faster benchmark runs

The library is well-suited for real-time event processing with fast individual operations and efficient handling of multiple events in single append calls. The advisory lock system provides effective resource locking with reasonable performance overhead. The HTTP API offers good performance for web applications, while direct library calls provide optimal performance for high-frequency operations. The comprehensive benchmark suite validates performance across all use cases with the new SQLite test data system providing consistent, fast benchmark execution.
