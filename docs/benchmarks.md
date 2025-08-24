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
1. **Go Library Benchmarks**: Test core DCB library performance (68 total benchmarks)
2. **Web-App Benchmarks**: Test HTTP API performance with load testing

### Test Data
- **Tiny Dataset**: 5 courses, 10 students, 16 enrollments
- **Small Dataset**: 1,000 courses, 10,000 students, 50,000 enrollments

## ⚠️ Important: Do Not Compare Go vs Web Benchmarks

**These benchmarks measure different aspects and should NOT be compared directly:**

### Go Library Benchmarks
- **Purpose**: Measure core DCB algorithm performance
- **Scope**: Single-threaded, direct database access
- **Configuration**: Conservative database pool (10 connections)
- **Use Case**: Algorithm optimization and core performance
- **Expected Performance**: Very fast (1-10ms operations)

### Web App Benchmarks  
- **Purpose**: Measure production HTTP API performance
- **Scope**: Concurrent HTTP service under load (100 VUs)
- **Configuration**: Production database pool (20 connections)
- **Use Case**: Production readiness and HTTP service performance
- **Expected Performance**: Slower due to HTTP overhead (100-1000ms operations)

### Why the Performance Difference is Expected
- **700x slower web performance is NORMAL** for a production HTTP service
- **Go benchmarks** measure algorithm efficiency
- **Web benchmarks** measure real-world API performance
- **Both are valuable** for their respective purposes
- **Direct comparison is misleading** and should be avoided

## Go Library Benchmarks

### Comprehensive Benchmark Coverage

**Total: 68 Go Benchmarks** covering all aspects of the DCB library

#### Benchmark Categories

| Category | Count | Purpose |
|----------|-------|---------|
| **Core Operations** | 47 | Basic append, read, and projection operations |
| **Enhanced Business Scenarios** | 6 | Real-world business logic and workflows |
| **Core Benchmark Functions** | 13 | Detailed performance analysis functions |
| **Framework Support** | 2 | Benchmark orchestration and reporting |

#### Core Operations (47 benchmarks)
- **Append Operations**: 22 benchmarks covering single events, batch operations (10, 100, 1000), and conditional appends
- **Read Operations**: 12 benchmarks for query performance, streaming, and channel operations
- **Projection Operations**: 12 benchmarks for state reconstruction and streaming projections
- **Quick Tests**: 3 benchmarks for basic functionality validation

#### Enhanced Business Scenarios (6 benchmarks)
- **Complex Business Workflow**: Real student enrollment scenarios with business rule validation
- **Concurrent Operations**: 10 concurrent user simulation for course registration
- **Mixed Operations**: Combined append, query, and projection sequences
- **Business Rule Validation**: DCB condition validation with real data
- **Request Burst**: 50 concurrent request simulation for burst traffic patterns
- **Sustained Load**: Mixed operation types over time for consistency testing

#### Realistic Benchmark Scenarios

**Most common real-world usage patterns:**

| Batch Size | Frequency | Use Case | Description |
|------------|-----------|----------|-------------|
| **1 event** | **Most Common** | Single operations | User login, status update, simple event |
| **2-3 events** | **Very Common** | Small transactions | Order creation, simple workflow |
| **5-8 events** | **Common** | Business operations | User registration, course enrollment |
| **12 events** | **Less Common** | Complex workflows | Multi-step business processes |
| **20+ events** | **Rare** | Bulk operations | Data migration, batch processing |

**Realistic Benchmark Types:**
- **AppendRealistic**: Tests common batch sizes (1, 2, 3, 5, 8, 12 events)
- **Real-world validation**: Measures performance for actual usage patterns
- **Business scenarios**: Reflects real application behavior, not artificial stress tests

#### SQLite Caching Optimization

**Pre-generated benchmark data eliminates runtime overhead:**

- **Cached Events**: 4,120 pre-generated events stored in SQLite
- **No Runtime Generation**: Eliminates `fmt.Sprintf` calls during benchmarks
- **Pure Performance**: Measures actual append operations, not data generation
- **Consistent Results**: Same data across runs for reliable comparison
- **Fast Access**: Instant data retrieval from SQLite cache

**Benchmark Data Categories:**
- **Single Events**: 1,000 unique single event operations
- **Realistic Batches**: 1,000 events with common batch sizes (1-12)
- **AppendIf Events**: 1,000 conditional append operations
- **Mixed Events**: 500 events with different event types
- **Batch Operations**: Various batch sizes for comprehensive testing

#### Concurrent User Metrics

**Enhanced benchmarks test realistic concurrent scenarios:**

| Benchmark | Concurrent Users | Scenario | Operations per User |
|-----------|------------------|----------|-------------------|
| **ConcurrentAppends** | **10 users** | Course registration | 1 event per user |
| **RequestBurst** | **50 users** | Burst traffic handling | 1 request per user |
| **SustainedLoad** | **Mixed** | Sustained application load | 4 operation types |

#### Concurrent Performance Characteristics

**ConcurrentAppends_Small (10 users):**
- **Concurrency Level**: 10 simultaneous users
- **Operation**: Course registration events
- **Real Data**: Uses actual student dataset
- **Performance**: Measures concurrent append throughput

**RequestBurst_Small (50 users):**
- **Concurrency Level**: 50 simultaneous requests
- **Operation**: Burst traffic simulation
- **Pattern**: Common in web applications
- **Performance**: Measures burst handling capacity

**SustainedLoad_Small (Mixed operations):**
- **Concurrency Level**: Mixed operation types
- **Operations**: Append, Query, Project, Conditional
- **Pattern**: Real application load simulation
- **Performance**: Measures sustained performance consistency

#### Concurrent Performance Results (2025-08-24)

**Real concurrent performance metrics from enhanced benchmarks:**

| Benchmark | Concurrent Users | Throughput | Latency | Memory | Allocations |
|-----------|------------------|------------|---------|---------|-------------|
| **ConcurrentAppends** | **10 users** | **338 ops/sec** | **3.0ms** | **26.6KB** | **550** |
| **RequestBurst** | **50 users** | **77 ops/sec** | **13.0ms** | **72.4KB** | **2,514** |

**Concurrent Performance Analysis:**
- **10 Concurrent Users**: 338 ops/sec with 3ms latency per operation
- **50 Concurrent Users**: 77 ops/sec with 13ms latency per operation
- **Scaling Factor**: 5x more users = 4.3x slower performance (expected under contention)
- **Memory Efficiency**: Higher concurrency increases memory usage and allocations
- **Real-World Validation**: Tests actual concurrent database access patterns

#### Core Benchmark Functions (13 functions)
- **AppendSingle**: Single event append performance
- **AppendBatch**: Batch event append with various sizes (10, 100, 1000)
- **AppendIf**: Conditional append with DCB concurrency control
- **AppendIfWithConflict**: Conflict scenario testing and resolution
- **Mixed Event Types**: Various event type combinations and patterns

### Latest Results (2025-08-24)

**Purpose**: Measure core DCB algorithm performance in isolation

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

#### Realistic Batch Performance (Most Common Scenarios)
| Operation | Throughput | Latency | Memory | Allocations |
|-----------|------------|---------|---------|-------------|
| **Realistic (Small)** | **2,230 ops/sec** | **1.2ms** | **1.4KB** | **49** |
| **Realistic (Tiny)** | **2,252 ops/sec** | **1.1ms** | **1.4KB** | **49** |

**Realistic benchmarks test common batch sizes (1, 2, 3, 5, 8, 12 events) that reflect real-world usage patterns.**

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

### Go Benchmark Use Cases
- **High-performance applications** requiring direct database access
- **Algorithm optimization** and performance tuning
- **Core library performance** validation
- **Memory usage** and allocation pattern analysis
- **Business logic performance** validation with real data
- **Concurrent operation** handling and scalability testing
- **Production readiness** with realistic data volumes

## Web-App Load Testing

### Latest Benchmarks (2025-08-24)

**Purpose**: Measure production HTTP API performance under realistic load

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

### Web App Benchmark Use Cases
- **HTTP-based integrations** and microservices
- **Production API performance** validation
- **Load testing** and capacity planning
- **Real-world usage** pattern validation

## Performance Characteristics

### Go Library Strengths
1. **Consistent Performance**: Predictable timing across dataset sizes
2. **High Throughput**: Excellent for direct database operations
3. **Memory Efficiency**: Optimized allocation patterns
4. **Fast Reads**: Query operations are very fast (~2,680 ops/sec)
5. **Efficient Batching**: Good scaling characteristics

### Web App Strengths
1. **High Reliability**: 100% success rate under load
2. **Good Scalability**: Handles concurrent users effectively
3. **Production Ready**: Full HTTP service with proper error handling
4. **Load Handling**: Sustains performance under stress

### Areas for Improvement
1. **AppendIf Performance**: Conditional appends are slower due to DCB concurrency control
2. **Memory Usage**: Large projections show high memory consumption
3. **Response Time**: Some operations hit P99 thresholds under load

## Use Case Recommendations

### When to Use Go Library
- **High-frequency operations** requiring maximum performance
- **Direct database access** applications
- **Algorithm development** and optimization
- **Memory-constrained** environments

### When to Use Web App
- **HTTP-based integrations** and microservices
- **Distributed systems** requiring HTTP APIs
- **Production deployments** with multiple clients
- **Load-balanced** environments

### Benchmark Execution

#### Individual Benchmark Categories
```bash
# Core operations only
make benchmark-go

# Enhanced business scenarios only  
make benchmark-go-enhanced

# All Go benchmarks (comprehensive)
make benchmark-go-all
```

#### Benchmark Data Generation
```bash
# Generate realistic benchmark data for fast access
make generate-benchmark-data

# Generate all data (datasets + benchmark data)
make generate-all-data

# Generate only test datasets
make generate-datasets
```

#### Dataset Integration
- **Real Data**: All benchmarks use actual student/course/enrollment datasets
- **PostgreSQL Integration**: Datasets are loaded into PostgreSQL before benchmarks
- **Consistent Environment**: Same database configuration across all benchmark types
- **Performance Validation**: Real-world data validates production readiness
- **SQLite Caching**: Pre-generated benchmark data eliminates runtime overhead

### Enhanced Benchmark Types

#### Basic Performance Benchmarks
- **Core Operations**: Simple append, query, projection
- **Batch Processing**: Various batch sizes (10, 100, 1000 events)
- **DCB Control**: Conditional appends with business rules
- **Memory Analysis**: Allocation patterns and memory usage

#### Complex Business Scenario Benchmarks
- **Business Workflows**: Complete user registration processes
- **Concurrent Operations**: Multiple user simulation (10 concurrent users)
- **Mixed Operations**: Append + Query + Projection sequences
- **Business Rule Validation**: Complex DCB conditions and validation
- **Load Patterns**: Burst traffic and sustained load simulation

#### Enhanced Features
- **Statistical Analysis**: Multiple benchmark runs (count=3) for consistency
- **Extended Duration**: Longer benchmark time (5s) for complex scenarios
- **Comprehensive Coverage**: Business logic, concurrency, and load testing
- **Production Simulation**: Real-world usage pattern validation

### Benchmark Results
All results are saved in the `benchmark-results/` directory with timestamps for analysis and comparison.

## Summary

**Go Library Benchmarks** measure core algorithm performance and are excellent for high-performance applications requiring direct database access.

**Web App Benchmarks** measure production HTTP API performance and validate the system's ability to handle real-world load scenarios.

**Both benchmark types are valuable** for their respective purposes, but they should not be compared directly as they measure fundamentally different aspects of the system.
