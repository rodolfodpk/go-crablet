# Performance Comparison

This document provides a comprehensive performance analysis of go-crablet's different operation modes and concurrency control mechanisms.

## Go Library Performance

### Concurrency Control Performance

| Method | Throughput | Latency | Success Rate | Memory |
|--------|------------|---------|--------------|---------|
| **Simple Append** | 1,000 ops/s| 1.0ms   | 100%         | 6.0KB/op     |
| **DCB Concurrency Control** | 800 ops/s| 1.3ms   | 100%         | 6.2KB/op     |

### Detailed Metrics

#### Simple Append (No Consistency Checks)
- **Throughput**: ~1,000 operations/second
- **Latency**: ~1.0ms average
- **Memory Usage**: ~6KB per operation
- **Allocations**: ~114 allocations per operation
- **Use Case**: Event logging, audit trails, non-critical operations

#### DCB Concurrency Control
- **Throughput**: ~800 operations/second
- **Latency**: ~1.3ms average
- **Memory Usage**: ~6.2KB per operation
- **Allocations**: ~120 allocations per operation
- **Use Case**: Business operations with rules, consistency requirements

## Web App Performance

### HTTP API Performance

| Endpoint | Throughput | Latency | Success Rate | Memory |
|----------|------------|---------|--------------|---------|
| **POST /append** | 64.21 req/s| 15.6ms  | 100%         | ~6KB/req |
| **POST /appendIf** | 32.5 req/s| 30.8ms  | 100%         | ~6KB/req |

### Performance Analysis

#### HTTP Overhead Impact
The web app performance is significantly lower than the Go library due to:

1. **HTTP Serialization**: JSON marshaling/unmarshaling overhead
2. **Network Latency**: HTTP request/response cycles
3. **Connection Pooling**: Database connection management
4. **Middleware Processing**: Logging, validation, error handling

#### Performance Comparison
- **Go Library**: ~1,000 ops/s (direct database access)
- **Web App**: ~64 req/s (HTTP API overhead)
- **Overhead**: ~15x slower due to HTTP layer

## Concurrency Control Analysis

### DCB Concurrency Control

#### Performance Characteristics
- **Throughput**: ~800 ops/s (Go library)
- **Latency**: ~1.3ms average
- **Success Rate**: 100% under normal conditions
- **Memory Usage**: ~6.2KB per operation

#### Use Cases
1. **Business Rule Validation**: Prevent duplicate enrollments
2. **State Consistency**: Ensure prerequisites exist
3. **Conflict Detection**: Fail-fast on concurrent modifications
4. **Domain Logic**: Enforce business constraints

#### What DCB Provides
- **Conflict Detection**: Identifies when business rules are violated during event appends
- **Domain Constraints**: Allows you to define conditions that must be met before events can be stored
- **Multi-instance Support**: Can work across different application instances
- **Consistent Performance**: Predictable behavior under load

#### Trade-offs
- **Performance**: Slightly slower than simple append
- **Complexity**: Requires condition definition
- **Memory**: Slightly higher memory usage

### Simple Append

#### Performance Characteristics
- **Throughput**: ~1,000 ops/s (Go library)
- **Latency**: ~1.0ms average
- **Success Rate**: 100%
- **Memory Usage**: ~6.0KB per operation

#### Use Cases
1. **Event Logging**: Audit trails, activity logs
2. **Non-critical Operations**: Background processing
3. **High-throughput Scenarios**: Bulk data ingestion
4. **Simple Workflows**: No business rule requirements

#### Characteristics
- **Higher Performance**: Faster than DCB operations due to no condition checking
- **Simplicity**: No condition setup required
- **Low Memory**: Minimal overhead
- **Reliability**: Consistent performance

#### Trade-offs
- **No Consistency**: No business rule validation
- **No Conflict Detection**: Concurrent modifications possible
- **Limited Use Cases**: Not suitable for business operations

## Performance Recommendations

### 1. Choose Based on Requirements

#### Use Simple Append When:
- **Performance is critical**: Maximum throughput needed
- **No business rules**: Simple event logging
- **High volume**: Bulk operations, audit trails
- **Non-critical**: Background processing

#### Use DCB Concurrency Control When:
- **Business rules matter**: Domain constraints required
- **Consistency is important**: State validation needed
- **Conflict detection**: Concurrent modification prevention
- **Production systems**: Business-critical operations

### 2. Performance Optimization

#### Database Level
- **Indexes**: Ensure GIN indexes on tags column
- **Connection pooling**: Optimize pool size for workload
- **Query analysis**: Monitor slow queries
- **Transaction size**: Batch operations when possible

#### Application Level
- **Event batching**: Group related events
- **Connection reuse**: Minimize connection overhead
- **Memory management**: Monitor allocation patterns
- **Error handling**: Implement retry logic

### 3. Monitoring and Alerting

#### Key Metrics to Track
- **Throughput**: Operations per second
- **Latency**: Response time percentiles
- **Success Rate**: Error percentage
- **Memory Usage**: Allocation patterns
- **Database Connections**: Pool utilization

#### Alert Thresholds
- **Latency**: Alert if > 10ms (Go library) or > 100ms (web app)
- **Success Rate**: Alert if < 99%
- **Memory**: Alert if > 10MB per operation
- **Throughput**: Alert if < 50% of baseline

## Benchmark Methodology

### Test Environment
- **Hardware**: Standard development machine
- **Database**: PostgreSQL 17.5 with default settings
- **Network**: Localhost (minimal network overhead)
- **Load**: Single-threaded benchmarks for consistency

### Test Data
- **Event Types**: Simple JSON objects
- **Tag Count**: 2-3 tags per event
- **Data Size**: ~100 bytes per event
- **Batch Size**: 1-10 events per operation

### Concurrency Testing
- **Concurrent Users**: 100 simulated users
- **Test Duration**: 30 seconds per test
- **Warm-up**: 5 seconds before measurement
- **Cool-down**: 5 seconds after measurement

## Conclusion

### Performance Summary

1. **Go Library Performance**:
   - Simple Append: ~1,000 ops/s, ~1.0ms latency
   - DCB Concurrency Control: ~800 ops/s, ~1.3ms latency
   - Both methods provide excellent performance for most use cases

2. **Web App Performance**:
   - HTTP overhead reduces performance by ~15x
   - Still suitable for most web applications
   - Consider direct library usage for high-performance scenarios

3. **Concurrency Control**:
   - DCB provides business rule validation with minimal performance impact
   - Simple append offers maximum performance for non-critical operations
   - Choose based on consistency requirements, not performance constraints

### Recommendations

1. **Use Simple Append** for event logging and non-critical operations
2. **Use DCB Concurrency Control** for business operations with rules
3. **Monitor performance** and optimize based on actual usage patterns
4. **Consider direct library usage** for high-performance requirements
5. **Implement proper error handling** and retry logic for production use

The performance characteristics demonstrate that go-crablet provides excellent performance for both simple event logging and complex business operations with DCB concurrency control. 