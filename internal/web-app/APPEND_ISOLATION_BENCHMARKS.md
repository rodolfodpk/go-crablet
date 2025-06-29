# Append Isolation Level Benchmarks

This document describes the k6 benchmarks for testing different transaction isolation levels in the go-crablet DCB implementation.

## Overview

The go-crablet DCB provides different append methods with different transaction isolation levels:

1. **`Append`** (default) - Uses `ReadCommitted` isolation
2. **`AppendIf`** - Uses `RepeatableRead` isolation (default for conditional appends)
3. **`AppendIf` with `X-Append-If-Isolation: serializable` header** - Uses `Serializable` isolation

Each isolation level provides different consistency guarantees and performance characteristics. The API has been simplified to use implicit isolation levels with HTTP headers for Serializable isolation when needed.

## Available Benchmarks

### 1. AppendIf Benchmarks (RepeatableRead Isolation)

#### Quick AppendIf Benchmark (`append-if-quick.js`)
- **Duration**: 30 seconds
- **Virtual Users**: Up to 10 (ramped up gradually)
- **Purpose**: Fast testing of RepeatableRead isolation scenarios
- **Scenarios Tested**:
  - Single event with condition (success case)
  - Single event without condition
  - Small batch with condition (success case)
  - Condition fail (duplicate detection)

**Run with:**
```bash
make append-if-quick
```

#### Full AppendIf Benchmark (`append-if-benchmark.js`)
- **Duration**: 6 minutes
- **Virtual Users**: Up to 200 (ramped up gradually)
- **Purpose**: Comprehensive testing of RepeatableRead isolation scenarios
- **Scenarios Tested**:
  - All quick scenarios plus:
  - Complex conditions with multiple event types
  - High concurrency scenarios
  - Large batch operations with conditions
  - Mixed event types with conditions

**Run with:**
```bash
make append-if-full
```

### 2. AppendIf with Serializable Isolation Benchmarks

#### Quick Serializable Benchmark (`append-if-isolated-quick.js`)
- **Duration**: 30 seconds
- **Virtual Users**: Up to 5 (ramped up gradually)
- **Purpose**: Fast testing of Serializable isolation scenarios
- **Scenarios Tested**:
  - Single event with condition (success case)
  - Single event without condition
  - Small batch with condition (success case)
  - Condition fail (duplicate detection)
  - After position condition test

**Run with:**
```bash
make append-if-isolated-quick
```

#### Full Serializable Benchmark (`append-if-isolated-benchmark.js`)
- **Duration**: 6 minutes
- **Virtual Users**: Up to 100 (ramped up gradually)
- **Purpose**: Comprehensive testing of Serializable isolation scenarios
- **Scenarios Tested**:
  - All quick scenarios plus:
  - Complex conditions with multiple event types
  - Serializable-specific concurrency scenarios
  - Medium batch operations with conditions
  - After position condition tests

**Run with:**
```bash
make append-if-isolated-full
```

## Isolation Level Differences

### RepeatableRead (AppendIf - default)
- **Consistency**: Strong consistency with snapshot isolation, prevents phantom reads
- **Performance**: Moderate overhead due to snapshot creation
- **Concurrency**: Good concurrency with MVCC
- **Use Cases**: Most business logic requiring strong consistency
- **Expected Performance**: ~100-200 req/s under load

### Serializable (AppendIf with header)
- **Consistency**: Highest consistency level, prevents all anomalies
- **Performance**: Higher overhead due to serialization checks
- **Concurrency**: Lower concurrency due to serialization conflicts
- **Use Cases**: Critical operations requiring absolute consistency
- **Expected Performance**: ~50-100 req/s under load

## Performance Expectations

### Throughput Comparison
1. **Standard Append** (ReadCommitted): ~200-500 req/s
2. **AppendIf** (RepeatableRead): ~100-200 req/s
3. **AppendIf with Serializable** (Serializable): ~50-100 req/s

### Response Time Comparison
1. **Standard Append**: < 1000ms (95th percentile)
2. **AppendIf**: < 1000ms (95th percentile)
3. **AppendIf with Serializable**: < 2000ms (95th percentile)

### Error Rate Expectations
1. **Standard Append**: < 5%
2. **AppendIf**: < 10%
3. **AppendIf with Serializable**: < 15% (higher due to serialization conflicts)

## Custom Metrics

### AppendIf Metrics
- `append_if_operations`: Counter for AppendIf operations
- `concurrency_errors`: Rate of concurrency-related failures
- `append_success`: Rate of successful appends

### Serializable Metrics
- `append_if_isolated_operations`: Counter for Serializable operations
- `concurrency_errors`: Rate of concurrency-related failures
- `serialization_errors`: Rate of serialization conflicts
- `append_success`: Rate of successful appends

## Test Scenarios

### Common Scenarios (Both Isolation Levels)
1. **Single Event with Condition**: Tests basic conditional append
2. **Single Event without Condition**: Tests unconditional append
3. **Batch with Condition**: Tests batch operations with conditions
4. **Condition Fail**: Tests duplicate detection and failure handling

### AppendIf-Specific Scenarios
1. **Complex Condition**: Multiple event types and tag combinations
2. **High Concurrency**: Tests RepeatableRead under concurrent load
3. **Large Batch**: Tests performance with larger event batches

### Serializable-Specific Scenarios
1. **Serializable Concurrency**: Tests Serializable isolation under load
2. **After Position Condition**: Tests conditions with position constraints
3. **Medium Batch**: Smaller batches due to Serializable overhead

## Running All Benchmarks

### Quick Comparison
```bash
# Run all quick benchmarks for comparison
make append-quick
make append-if-quick
make append-if-isolated-quick
```

### Full Comparison
```bash
# Run all full benchmarks for comprehensive comparison
make append-full
make append-if-full
make append-if-isolated-full
```

### Individual Tests
```bash
# Test specific isolation levels
make append-if-quick          # Quick RepeatableRead test
make append-if-full           # Full RepeatableRead test
make append-if-isolated-quick  # Quick Serializable test
make append-if-isolated-full   # Full Serializable test
```

## HTTP Endpoints

The web-app exposes append endpoints with simplified isolation level control:

- **`POST /append`** - Standard append (ReadCommitted)
- **`POST /append-if`** - Conditional append (RepeatableRead by default)
- **`POST /append-if` with `X-Append-If-Isolation: serializable`** - Conditional append with Serializable isolation

All endpoints use the same request/response format, maintaining OpenAPI compatibility.

## HTTP Header Usage

To control isolation levels for conditional appends, use the `X-Append-If-Isolation` header:

```bash
# Default (RepeatableRead)
curl -X POST /append-if -H "Content-Type: application/json" -d '{"events":[...]}'

# Serializable isolation
curl -X POST /append-if -H "Content-Type: application/json" -H "X-Append-If-Isolation: serializable" -d '{"events":[...]}'
```

## Simplified API Design

The go-crablet DCB uses a simplified approach to isolation levels:

- **Core API**: Uses implicit isolation levels (not configurable)
- **HTTP Layer**: Uses headers for Serializable isolation when needed
- **Benefits**: Cleaner API, better performance, reduced complexity

### Isolation Level Mapping
- **Append**: Always uses ReadCommitted (fastest)
- **AppendIf**: Always uses RepeatableRead (strong consistency)
- **AppendIf with Serializable**: Uses HTTP header to override to Serializable

## Troubleshooting

### Common Issues
1. **High Error Rates**: Serializable isolation may have higher error rates due to conflicts
2. **Slow Response Times**: Serializable isolation has higher latency
3. **Connection Timeouts**: Increase timeout values for Serializable tests

### Performance Tuning
- Adjust virtual user counts based on your system capabilities
- Modify sleep times to control request rates
- Tune database connection pool settings for your workload

## Expected Results Summary

| Metric | Standard Append | AppendIf | AppendIf with Serializable |
|--------|----------------|----------|---------------------------|
| Throughput | 200-500 req/s | 100-200 req/s | 50-100 req/s |
| Response Time (95%) | < 1000ms | < 1000ms | < 2000ms |
| Error Rate | < 5% | < 10% | < 15% |
| Consistency | ReadCommitted | RepeatableRead | Serializable |
| Use Case | General purpose | Business logic | Critical operations |
| API Complexity | Simple | Simple | Header-based override | 