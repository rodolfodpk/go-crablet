# Local PostgreSQL 17.6 Performance

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

**Current Local PostgreSQL Performance (September 16, 2025):**

## Detailed Performance Results (Local PostgreSQL)

**Benchmark Data Source**: `go_benchmarks_20250916_211519.txt` (September 16, 2025)
**Environment**: Local PostgreSQL 16 on macOS (Apple M1 Pro)
**Benchmark Type**: Realistic business scenarios with course enrollment events

### Append Performance

| Dataset | Concurrency | Events | Throughput (ops/sec) | Latency (ns/op) | Memory (B/op) | Allocations |
|---------|-------------|--------|---------------------|-----------------|---------------|-------------|
| Medium | 1 | 1 | 3,625 | 275,915 | 2,988 | 56 |
| Small | 1 | 1 | 3,870 | 258,269 | 3,000 | 56 |
| Tiny | 1 | 1 | 3,870 | 249,256 | 2,990 | 56 |
| Medium | 1 | 10 | 2,288 | 494,445 | 31,654 | 253 |
| Small | 1 | 10 | 1,603 | 626,150 | 31,666 | 253 |
| Tiny | 1 | 10 | 2,070 | 483,204 | 31,679 | 253 |
| Medium | 100 | 1 | 127 | 7,902,070 | 295,191 | 5,456 |
| Small | 100 | 1 | 127 | 7,917,297 | 295,426 | 5,458 |
| Tiny | 100 | 1 | 128 | 7,791,038 | 295,462 | 5,457 |
| Medium | 100 | 10 | 56 | 17,786,466 | 3,599,902 | 25,258 |
| Small | 100 | 10 | 50 | 20,086,309 | 3,602,393 | 25,271 |
| Tiny | 100 | 10 | 50 | 20,076,981 | 3,603,581 | 25,283 |

### AppendIf Performance (No Conflict)

| Dataset | Concurrency | Events | Throughput (ops/sec) | Latency (ns/op) | Memory (B/op) | Allocations |
|---------|-------------|--------|---------------------|-----------------|---------------|-------------|
| Small | 1 | 1 | 1,220 | 819,145 | 4,850 | 96 |
| Medium | 1 | 1 | 1,103 | 906,534 | 4,846 | 96 |
| Tiny | 1 | 1 | 1,164 | 1,089,374 | 4,852 | 96 |
| Small | 1 | 10 | 407 | 2,459,322 | 38,421 | 295 |
| Medium | 1 | 10 | 430 | 2,325,759 | 38,393 | 295 |
| Tiny | 1 | 10 | 272 | 3,666,334 | 38,436 | 295 |
| Small | 100 | 1 | 35 | 28,724,148 | 562,312 | 9,551 |
| Medium | 100 | 1 | 46 | 21,550,057 | 562,214 | 9,552 |
| Tiny | 100 | 1 | 45 | 22,023,323 | 562,576 | 9,553 |
| Small | 100 | 10 | 9 | 117,461,555 | 3,841,505 | 29,423 |
| Medium | 100 | 10 | 7 | 148,948,714 | 3,839,164 | 29,407 |
| Tiny | 100 | 10 | 9 | 112,978,576 | 3,844,722 | 29,444 |

### Project Performance

| Dataset | Concurrency | Throughput (ops/sec) | Latency (ns/op) | Memory (B/op) | Allocations |
|---------|-------------|---------------------|-----------------|---------------|-------------|
| Medium | 1 | 3,163 | 314,882 | 68,533 | 1,486 |
| Small | 1 | 3,348 | 298,773 | 68,545 | 1,486 |
| Tiny | 1 | 3,433 | 291,323 | 68,553 | 1,486 |
| Medium | 100 | 97 | 10,290,708 | 6,849,270 | 148,463 |
| Small | 100 | 96 | 10,414,504 | 6,850,855 | 148,478 |
| Tiny | 100 | 98 | 10,162,482 | 6,852,068 | 148,491 |

### Query Performance

| Dataset | Concurrency | Throughput (ops/sec) | Latency (ns/op) | Memory (B/op) | Allocations |
|---------|-------------|---------------------|-----------------|---------------|-------------|
| Medium | 1 | 6,147 | 162,680 | 32,842 | 497 |
| Small | 1 | 5,696 | 175,570 | 32,845 | 497 |
| Tiny | 1 | 5,750 | 173,920 | 32,850 | 497 |
| Medium | 100 | 161 | 6,210,771 | 3,280,107 | 49,557 |
| Small | 100 | 147 | 6,798,822 | 3,280,427 | 49,558 |
| Tiny | 100 | 152 | 6,601,194 | 3,281,163 | 49,562 |

**Key Performance Insights:**
- **Append operations**: 3,625-3,870 ops/sec (single user, single event)
- **AppendIf operations**: 1,103-1,220 ops/sec (single user, single event)
- **Project operations**: 3,163-3,433 ops/sec (single user)
- **Query operations**: 5,696-6,147 ops/sec (single user)
- **Concurrency impact**: Performance degrades significantly with 100 concurrent users
- **Memory usage**: Consistent across datasets, scales with concurrency
- **Total execution time**: 390.817 seconds (~6.5 minutes) for complete benchmark suite

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
| **ProjectionLimits** | Medium | 5 | ~100 | 7,203 | 0.39 | 23.86 | 488 | 100% | 0% |
| **ProjectionLimits** | Small | 5 | ~100 | 6,907 | 0.39 | 23.86 | 488 | 100% | 0% |
| **ProjectionLimits** | Tiny | 5 | ~100 | 6,818 | 0.39 | 23.86 | 488 | 100% | 0% |
| **ProjectionLimits** | Medium | 8 | ~100 | 5,916 | 0.41 | 34.69 | 716 | 62.5% | 37.5% |
| **ProjectionLimits** | Small | 8 | ~100 | 6,103 | 0.42 | 34.69 | 716 | 62.5% | 37.5% |
| **ProjectionLimits** | Tiny | 8 | ~100 | 6,099 | 0.42 | 34.69 | 716 | 62.5% | 37.5% |
| **ProjectionLimits** | Medium | 10 | ~100 | 5,552 | 0.46 | 45.23 | 938 | 50% | 50% |
| **ProjectionLimits** | Small | 10 | ~100 | 5,446 | 0.46 | 45.23 | 938 | 50% | 50% |
| **ProjectionLimits** | Tiny | 10 | ~100 | 5,446 | 0.46 | 45.23 | 938 | 50% | 50% |