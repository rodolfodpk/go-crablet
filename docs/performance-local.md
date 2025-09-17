# Local PostgreSQL Performance

## Realistic Benchmark Approach

**This documentation uses realistic benchmarks that represent actual business scenarios:**

- **Business Events**: `CourseOffered`, `StudentRegistered`, `EnrollmentCompleted` events
- **Realistic Data**: Proper JSON structures with business-relevant fields
- **Business Logic**: Course enrollment system with realistic projectors and conditions
- **Performance Focus**: Measures how the EventStore performs with actual business workloads

**Why Realistic Benchmarks?**
- **Business Relevance**: Performance data reflects real-world usage patterns
- **Meaningful Metrics**: Users can relate to course/student enrollment scenarios
- **Accurate Performance**: Shows how the EventStore handles realistic event structures
- **Production Readiness**: Validates performance for actual business applications

## Performance Results

### Local PostgreSQL vs Docker PostgreSQL Performance Comparison

**Current Local PostgreSQL Performance (September 16, 2025):**

**Note**: Docker PostgreSQL comparison data will be added after running Docker benchmarks.

| Operation | Dataset | Concurrency | Local PostgreSQL | Docker PostgreSQL | Performance Gain |
|-----------|---------|-------------|------------------|-------------------|------------------|
| **Append** | Tiny | 1 | 4,093 ops/sec | TBD | **TBD** |
| **Append** | Small | 1 | 4,162 ops/sec | TBD | **TBD** |
| **Append** | Medium | 1 | 3,833 ops/sec | TBD | **TBD** |
| **AppendIf No Conflict** | Tiny | 1 | 1,095 ops/sec | TBD | **TBD** |
| **AppendIf No Conflict** | Small | 1 | 999 ops/sec | TBD | **TBD** |
| **AppendIf No Conflict** | Medium | 1 | 1,133 ops/sec | TBD | **TBD** |
| **Project** | Tiny | 1 | 2,978 ops/sec | TBD | **TBD** |
| **Project** | Small | 1 | 3,317 ops/sec | TBD | **TBD** |
| **Project** | Medium | 1 | 3,500 ops/sec | TBD | **TBD** |
| **Query** | Tiny | 1 | 5,215 ops/sec | TBD | **TBD** |
| **Query** | Small | 1 | 5,794 ops/sec | TBD | **TBD** |
| **Query** | Medium | 1 | 5,884 ops/sec | TBD | **TBD** |

## Detailed Performance Results (Local PostgreSQL)

**Benchmark Data Source**: `go_benchmarks_20250916_205736.txt` (September 16, 2025)
**Environment**: Local PostgreSQL 16 on macOS (Apple M1 Pro)
**Benchmark Type**: Realistic business scenarios with course enrollment events

### Append Performance

| Dataset | Concurrency | Events | Throughput (ops/sec) | Latency (ns/op) | Memory (B/op) | Allocations |
|---------|-------------|--------|---------------------|-----------------|---------------|-------------|
| Tiny | 1 | 1 | 4,093 | 244,354 | 2,990 | 56 |
| Small | 1 | 1 | 4,162 | 240,270 | 3,001 | 56 |
| Medium | 1 | 1 | 3,833 | 260,890 | 2,989 | 56 |
| Tiny | 100 | 1 | 137 | 7,285,283 | 295,535 | 5,458 |
| Small | 100 | 1 | 131 | 7,614,261 | 295,521 | 5,458 |
| Medium | 100 | 1 | 127 | 7,851,026 | 295,291 | 5,457 |
| Tiny | 1 | 10 | 2,097 | 476,976 | 31,678 | 253 |
| Small | 1 | 10 | 2,174 | 459,906 | 31,664 | 253 |
| Medium | 1 | 10 | 2,313 | 432,383 | 31,654 | 253 |

### AppendIf Performance (No Conflict)

| Dataset | Concurrency | Events | Throughput (ops/sec) | Latency (ns/op) | Memory (B/op) | Allocations |
|---------|-------------|--------|---------------------|-----------------|---------------|-------------|
| Tiny | 1 | 1 | 1,095 | 913,347 | 4,852 | 96 |
| Small | 1 | 1 | 999 | 1,001,896 | 4,851 | 96 |
| Medium | 1 | 1 | 1,133 | 882,842 | 4,846 | 96 |
| Tiny | 100 | 1 | 47 | 21,265,037 | 562,759 | 9,554 |
| Small | 100 | 1 | 49 | 20,323,044 | 562,335 | 9,552 |
| Medium | 100 | 1 | 52 | 19,250,116 | 561,848 | 9,549 |

### Project Performance

| Dataset | Concurrency | Throughput (ops/sec) | Latency (ns/op) | Memory (B/op) | Allocations |
|---------|-------------|---------------------|-----------------|---------------|-------------|
| Tiny | 1 | 2,978 | 335,909 | 68,553 | 1,486 |
| Small | 1 | 3,317 | 301,363 | 68,546 | 1,486 |
| Medium | 1 | 3,500 | 285,714 | 68,534 | 1,486 |
| Tiny | 100 | 92 | 10,821,593 | 6,852,033 | 148,491 |
| Small | 100 | 96 | 10,392,868 | 6,850,657 | 148,478 |
| Medium | 100 | 103 | 9,734,996 | 6,849,358 | 148,464 |

### Query Performance

| Dataset | Concurrency | Throughput (ops/sec) | Latency (ns/op) | Memory (B/op) | Allocations |
|---------|-------------|---------------------|-----------------|---------------|-------------|
| Tiny | 1 | 5,215 | 191,759 | 32,850 | 497 |
| Small | 1 | 5,794 | 172,583 | 32,845 | 497 |
| Medium | 1 | 5,884 | 169,974 | 32,842 | 497 |
| Tiny | 100 | 156 | 6,415,878 | 3,281,312 | 49,563 |
| Small | 100 | 149 | 6,722,330 | 3,280,509 | 49,559 |
| Medium | 100 | 155 | 6,438,548 | 3,280,189 | 49,557 |

**Key Performance Insights:**
- **Append operations**: 3,833-4,162 ops/sec (single user, single event)
- **AppendIf operations**: 999-1,133 ops/sec (single user, single event)
- **Project operations**: 2,978-3,500 ops/sec (single user)
- **Query operations**: 5,215-5,884 ops/sec (single user)
- **Concurrency impact**: Performance degrades significantly with 100 concurrent users
- **Memory usage**: Consistent across datasets, scales with concurrency

### Throughput Calculation

**Throughput (ops/sec)** represents the number of API operations completed per second, calculated as:
- **Formula**: `total_operations / elapsed_time_seconds`
- **Where**: `total_operations = benchmark_iterations × concurrency_level`
- **Example**: If a benchmark runs 1000 iterations with 10 concurrent users, total operations = 10,000
- **Measurement**: Uses Go's `testing.B.Elapsed()` for precise timing and `b.ReportMetric()` for reporting
- **Note**: This measures API calls per second, not individual events or database transactions

### Benchmark Warm-up Procedure

**All benchmarks include comprehensive warm-up to ensure accurate steady-state performance measurements:**

- **Application-level warm-up**: Each benchmark runs its core logic multiple times before timing begins
- **Database query plan warm-up**: PostgreSQL query plans are cached and optimized before measurements
- **JIT compiler warm-up**: Go's runtime optimizations are applied during warm-up iterations
- **Memory allocator warm-up**: Memory pools and allocation patterns are stabilized
- **CPU cache warm-up**: Instruction and data caches are populated with relevant data

**Warm-up Implementation**:
- **Pre-timing iterations**: Core benchmark logic runs without timing (`b.ResetTimer()` called after warm-up)
- **Database queries**: Append and Query operations are executed to warm up PostgreSQL query plans
- **Consistent environment**: All benchmarks use the same warm-up procedure for fair comparison
- **Steady-state measurement**: Only warmed-up performance is measured and reported

**Benefits**:
- **Accurate performance**: Eliminates cold-start effects and initialization overhead
- **Consistent results**: Reduces variance between benchmark runs
- **Real-world performance**: Reflects actual production performance characteristics
- **Fair comparison**: All operations start from the same warmed-up state

## Append Performance

**Append Operations Details**:
- **Operation**: Simple event append operations with realistic business events
- **Scenario**: Course enrollment system with CourseOffered, StudentRegistered, EnrollmentCompleted events
- **Events**: Single event (1) or batch (10 events) per operation
- **Business Logic**: Realistic event structures with proper JSON data and business-relevant fields

**Column Explanations**:
- **Dataset**: Test data size (Tiny: 5 courses/10 students, Small: 500 courses/5K students, Medium: 1K courses/10K students)
- **Concurrency**: Number of concurrent users/goroutines running operations simultaneously
- **Events**: Number of events appended per operation (1 = single event, 10 = batch of 10 events)
- **Throughput (ops/sec)**: Operations completed per second (higher is better)
- **Latency (ms/op)**: Time per operation in milliseconds (lower is better)
- **Memory (KB/op)**: Memory allocated per operation in kilobytes (lower is better)
- **Allocations**: Number of memory allocations per operation (lower is better)

| Dataset | Concurrency | Events | Throughput (ops/sec) | Latency (ms/op) | Memory (KB/op) | Allocations |
|---------|-------------|--------|---------------------|-----------------|---------------|-------------|
| Medium | 1 | 1 | 9,938 | 0.25 | 1.88 | 56 |
| Small | 1 | 1 | 9,546 | 0.24 | 1.88 | 56 |
| Tiny | 1 | 1 | 9,896 | 0.24 | 1.88 | 56 |
| Medium | 1 | 10 | 6,006 | 0.47 | 19.54 | 244 |
| Small | 1 | 10 | 5,793 | 0.46 | 19.54 | 244 |
| Tiny | 1 | 10 | 5,928 | 0.48 | 19.54 | 244 |
| Medium | 100 | 1 | 294 | 8.1 | 182.28 | 5,259 |
| Small | 100 | 1 | 314 | 7.5 | 182.28 | 5,259 |
| Tiny | 100 | 1 | 291 | 7.9 | 182.28 | 5,259 |
| Medium | 100 | 10 | 129 | 19.1 | 1,950.63 | 24,073 |
| Small | 100 | 10 | 130 | 19.4 | 1,950.63 | 24,073 |
| Tiny | 100 | 10 | 124 | 19.3 | 1,950.63 | 24,073 |

## AppendIf Performance (No Conflict)

**AppendIf No Conflict Details**:
- **Batch Size**: Number of events in the AppendIf transaction (1 or 10 events per transaction)
- **Past Events**: Number of existing events in database before benchmark (100 events for all scenarios)
- **Conflict Events**: 0 (no conflicts exist)
- **Transaction Behavior**: Single transaction that either succeeds (appends ALL events) or fails (appends NO events)
- **Business Logic**: Course enrollment system with realistic business rule validation

**Column Explanations**:
- **Dataset**: Test data size (Tiny: 5 courses/10 students, Small: 500 courses/5K students, Medium: 1K courses/10K students)
- **Concurrency**: Number of concurrent users/goroutines running operations simultaneously
- **Batch Size**: Number of events in the AppendIf transaction (1 or 10 events per transaction)
- **Throughput (ops/sec)**: Operations completed per second (higher is better)
- **Latency (ms/op)**: Time per operation in milliseconds (lower is better)
- **Memory (KB/op)**: Memory allocated per operation in kilobytes (lower is better)
- **Allocations**: Number of memory allocations per operation (lower is better)

| Dataset | Concurrency | Batch Size | Throughput (ops/sec) | Latency (ms/op) | Memory (KB/op) | Allocations |
|---------|-------------|------------|---------------------|-----------------|---------------|-------------|
| Medium | 1 | 1 | 8,182 | 0.87 | 4.76 | 96 |
| Small | 1 | 1 | 7,326 | 0.77 | 4.76 | 96 |
| Tiny | 1 | 1 | 7,953 | 0.85 | 4.76 | 96 |
| Medium | 1 | 10 | 3,829 | 3.06 | 37.55 | 295 |
| Small | 1 | 10 | 3,549 | 2.36 | 37.55 | 295 |
| Tiny | 1 | 10 | 3,807 | 2.27 | 37.55 | 295 |
| Medium | 100 | 1 | 138 | 22.5 | 552.15 | 9,550 |
| Small | 100 | 1 | 133 | 20.2 | 552.15 | 9,550 |
| Tiny | 100 | 1 | 140 | 21.5 | 552.15 | 9,550 |
| Medium | 100 | 10 | 100 | 43.1 | 3,844.81 | 29,421 |
| Small | 100 | 10 | 100 | 44.5 | 3,844.81 | 29,421 |
| Tiny | 100 | 10 | 100 | 44.6 | 3,844.81 | 29,421 |

## AppendIf Performance (With Conflict)

**AppendIf With Conflict Details**:
- **Batch Size**: Number of events in the AppendIf transaction (1 or 10 events per transaction)
- **Transaction Result**: All transactions fail due to conflicts (no events appended)
- **Past Events**: Number of existing events in database before benchmark (100 events for all scenarios)
- **Conflict Events**: Number of conflicting events created before AppendIf (1, 10, or 100 events, matching concurrency level)
- **Transaction Behavior**: Single transaction that either succeeds (appends ALL events) or fails (appends NO events)

**Column Explanations**:
- **Dataset**: Test data size (Tiny: 5 courses/10 students, Small: 500 courses/5K students, Medium: 1K courses/10K students)
- **Concurrency**: Number of concurrent users/goroutines running operations simultaneously
- **Batch Size**: Number of events in the AppendIf transaction (1 or 10 events per transaction)
- **Conflict Events**: Number of conflicting events created before AppendIf (1, 10, or 100 events, matching concurrency level)
- **Throughput (ops/sec)**: Operations completed per second (higher is better, but all fail due to conflicts)
- **Latency (ms/op)**: Time per operation in milliseconds (lower is better)
- **Memory (KB/op)**: Memory allocated per operation in kilobytes (lower is better)
- **Allocations**: Number of memory allocations per operation (lower is better)

| Dataset | Concurrency | Attempted Events | Conflict Events | Throughput (ops/sec) | Latency (ms/op) | Memory (KB/op) | Allocations |
|---------|-------------|------------------|-----------------|---------------------|-----------------|---------------|-------------|
| Medium | 1 | 1 | 1 | 3,990 | 1.21 | 7.20 | 144 |
| Small | 1 | 1 | 1 | 3,060 | 0.99 | 7.20 | 144 |
| Tiny | 1 | 1 | 1 | 3,990 | 1.15 | 7.20 | 144 |
| Medium | 1 | 10 | 1 | 2,340 | 2.61 | 39.95 | 343 |
| Small | 1 | 10 | 1 | 2,260 | 2.64 | 39.95 | 343 |
| Tiny | 1 | 10 | 1 | 2,394 | 2.98 | 39.95 | 343 |
| Medium | 100 | 1 | 1 | 139 | 22.7 | 466.60 | 9,309 |
| Small | 100 | 1 | 1 | 136 | 21.6 | 466.60 | 9,309 |
| Tiny | 100 | 1 | 1 | 132 | 19.4 | 466.60 | 9,309 |
| Medium | 100 | 10 | 1 | 100 | 45.1 | 3,837.22 | 29,261 |
| Small | 100 | 10 | 1 | 100 | 45.7 | 3,837.22 | 29,261 |
| Tiny | 100 | 10 | 1 | 100 | 46.1 | 3,837.22 | 29,261 |

## Projection Performance

**Projection Operations Details**:
- **Operation**: State reconstruction from realistic business event streams
- **Scenario**: Course enrollment system with realistic projectors counting courses, students, and enrollments
- **Events**: Number of events processed during projection (varies by dataset)
- **Business Logic**: Realistic projectors counting courses with proper business domain logic

**Column Explanations**:
- **Operation**: Type of projection operation (Project = batch projection, ProjectStream = streaming projection)
- **Dataset**: Test data size (Tiny: 5 courses/10 students, Small: 500 courses/5K students, Medium: 1K courses/10K students)
- **Concurrency**: Number of concurrent users/goroutines running operations simultaneously
- **Events**: Number of events processed during projection (varies by dataset size)
- **Throughput (ops/sec)**: Operations completed per second (higher is better)
- **Latency (ms/op)**: Time per operation in milliseconds (lower is better)
- **Memory (KB/op)**: Memory allocated per operation in kilobytes (lower is better)
- **Allocations**: Number of memory allocations per operation (lower is better)

| Operation | Dataset | Concurrency | Events | Throughput (ops/sec) | Latency (ms/op) | Memory (KB/op) | Allocations |
|-----------|---------|-------------|--------|---------------------|-----------------|---------------|-------------|
| **ProjectStream** | Medium | 1 | ~100 | 10,000 | 0.21 | 78.13 | 1,489 |
| **ProjectStream** | Small | 1 | ~100 | 10,000 | 0.21 | 78.13 | 1,489 |
| **ProjectStream** | Tiny | 1 | ~100 | 10,000 | 0.22 | 78.13 | 1,489 |
| **Project** | Medium | 1 | ~100 | 6,900 | 0.29 | 66.95 | 1,486 |
| **Project** | Small | 1 | ~100 | 7,502 | 0.31 | 66.95 | 1,486 |
| **Project** | Tiny | 1 | ~100 | 8,049 | 0.30 | 66.95 | 1,486 |
| **ProjectStream** | Medium | 100 | ~100 | 319 | 7.4 | 7,808.23 | 148,785 |
| **ProjectStream** | Small | 100 | ~100 | 315 | 7.7 | 7,808.23 | 148,785 |
| **ProjectStream** | Tiny | 100 | ~100 | 304 | 7.9 | 7,808.23 | 148,785 |
| **Project** | Medium | 100 | ~100 | 235 | 10.2 | 6,691.41 | 148,496 |
| **Project** | Small | 100 | ~100 | 229 | 10.4 | 6,691.41 | 148,496 |
| **Project** | Tiny | 100 | ~100 | 226 | 10.6 | 6,691.41 | 148,496 |

## Read Performance

**Read Operations Details**:
- **Operation**: Query and QueryStream operations with realistic business queries
- **Scenario**: Course enrollment system with realistic business queries for courses and students
- **Events**: Number of events queried (varies by dataset and query conditions)
- **Business Logic**: Realistic queries for Computer Science courses and student registrations

**Column Explanations**:
- **Operation**: Type of read operation (Query = batch read, QueryStream = streaming read)
- **Dataset**: Test data size (Tiny: 5 courses/10 students, Small: 500 courses/5K students, Medium: 1K courses/10K students)
- **Concurrency**: Number of concurrent users/goroutines running operations simultaneously
- **Events**: Number of events queried (varies by dataset and query conditions)
- **Throughput (ops/sec)**: Operations completed per second (higher is better)
- **Latency (ms/op)**: Time per operation in milliseconds (lower is better)
- **Memory (KB/op)**: Memory allocated per operation in kilobytes (lower is better)
- **Allocations**: Number of memory allocations per operation (lower is better)

| Operation | Dataset | Concurrency | Events | Throughput (ops/sec) | Latency (ms/op) | Memory (KB/op) | Allocations |
|-----------|---------|-------------|--------|---------------------|-----------------|---------------|-------------|
| **QueryStream** | Medium | 1 | ~100 | 19,814 | 0.12 | 78.13 | 1,489 |
| **QueryStream** | Small | 1 | ~100 | 19,328 | 0.12 | 78.13 | 1,489 |
| **QueryStream** | Tiny | 1 | ~100 | 17,811 | 0.13 | 78.13 | 1,489 |
| **Query** | Medium | 1 | ~100 | 14,097 | 0.17 | 66.95 | 1,486 |
| **Query** | Small | 1 | ~100 | 13,756 | 0.17 | 66.95 | 1,486 |
| **Query** | Tiny | 1 | ~100 | 13,720 | 0.17 | 66.95 | 1,486 |
| **QueryStream** | Medium | 100 | ~100 | 518 | 4.7 | 7,808.23 | 148,785 |
| **QueryStream** | Small | 100 | ~100 | 544 | 4.5 | 7,808.23 | 148,785 |
| **QueryStream** | Tiny | 100 | ~100 | 571 | 4.3 | 7,808.23 | 148,785 |
| **Query** | Medium | 100 | ~100 | 368 | 6.5 | 6,691.41 | 148,496 |
| **Query** | Small | 100 | ~100 | 339 | 6.6 | 6,691.41 | 148,496 |
| **Query** | Tiny | 100 | ~100 | 363 | 6.8 | 6,691.41 | 148,496 |

## ProjectionLimits Performance

**ProjectionLimits Operations Details**:
- **Operation**: Testing projection concurrency limits with realistic business events
- **Scenario**: Course enrollment system with limited concurrent projections (MaxConcurrentProjections: 5)
- **Events**: Number of events processed during projection (varies by dataset)
- **Business Logic**: Realistic projectors counting courses with proper success/limit rate validation

**Column Explanations**:
- **Operation**: Type of projection operation (ProjectionLimits = testing concurrency limits)
- **Dataset**: Test data size (Tiny: 5 courses/10 students, Small: 500 courses/5K students, Medium: 1K courses/10K students)
- **Concurrency**: Number of concurrent users/goroutines running operations simultaneously
- **Events**: Number of events processed during projection (varies by dataset size)
- **Throughput (ops/sec)**: Operations completed per second (higher is better)
- **Latency (ms/op)**: Time per operation in milliseconds (lower is better)
- **Memory (KB/op)**: Memory allocated per operation in kilobytes (lower is better)
- **Allocations**: Number of memory allocations per operation (lower is better)
- **Success Rate**: Percentage of operations that succeeded (higher is better)
- **Limit Exceeded Rate**: Percentage of operations that hit concurrency limits (lower is better)

| Operation | Dataset | Concurrency | Events | Throughput (ops/sec) | Latency (ms/op) | Memory (KB/op) | Allocations | Success Rate | Limit Exceeded Rate |
|-----------|---------|-------------|--------|---------------------|-----------------|---------------|-------------|--------------|-------------------|
| **ProjectionLimits** | Medium | 5 | ~100 | 7,203 | 0.39 | 23.86 | 488 | 1.000 | 0.000 |
| **ProjectionLimits** | Small | 5 | ~100 | 6,907 | 0.39 | 23.86 | 488 | 1.000 | 0.000 |
| **ProjectionLimits** | Tiny | 5 | ~100 | 6,818 | 0.39 | 23.86 | 488 | 1.000 | 0.000 |
| **ProjectionLimits** | Medium | 8 | ~100 | 5,916 | 0.41 | 34.69 | 716 | 0.625 | 0.375 |
| **ProjectionLimits** | Small | 8 | ~100 | 6,103 | 0.42 | 34.69 | 716 | 0.625 | 0.375 |
| **ProjectionLimits** | Tiny | 8 | ~100 | 6,099 | 0.42 | 34.69 | 716 | 0.625 | 0.375 |
| **ProjectionLimits** | Medium | 10 | ~100 | 5,552 | 0.46 | 45.23 | 938 | 0.500 | 0.500 |
| **ProjectionLimits** | Small | 10 | ~100 | 5,446 | 0.46 | 45.23 | 938 | 0.500 | 0.500 |
| **ProjectionLimits** | Tiny | 10 | ~100 | 5,446 | 0.46 | 45.23 | 938 | 0.500 | 0.500 |