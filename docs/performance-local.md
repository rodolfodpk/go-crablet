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
- **Operation**: Realistic event append operations using business events
- **Scenario**: Course enrollment system with CourseOffered, StudentRegistered, EnrollmentCompleted events
- **Events**: Single event (1) or batch (10 events)
- **Model**: Realistic business events with proper JSON data structures

**Column Explanations**:
- **Dataset**: Test data size (Tiny: 5 courses/10 students, Small: 1K courses/10K students, Medium: 1K courses/10K students)
- **Concurrency**: Number of concurrent users/goroutines running operations simultaneously
- **Events**: Number of events appended per operation (1 = single event, 10 = batch of 10 events)
- **Throughput (ops/sec)**: Operations completed per second (higher is better)
- **Latency (ms/op)**: Time per operation in milliseconds (lower is better)
- **Memory (KB/op)**: Memory allocated per operation in kilobytes (lower is better)
- **Allocations**: Number of memory allocations per operation (lower is better)

| Dataset | Concurrency | Events | Throughput (ops/sec) | Latency (ms/op) | Memory (KB/op) | Allocations |
|---------|-------------|--------|---------------------|-----------------|---------------|-------------|
| Tiny | 1 | 1 | 4,856 | 0.23 | 1.88 | 56 |
| Small | 1 | 1 | 9,691 | 0.24 | 1.88 | 56 |
| Medium | 1 | 1 | 9,700 | 0.23 | 1.88 | 56 |
| Tiny | 1 | 10 | 2,860 | 0.42 | 19.54 | 244 |
| Small | 1 | 10 | 6,028 | 0.45 | 19.54 | 244 |
| Medium | 1 | 10 | 6,223 | 0.45 | 19.54 | 244 |
| Tiny | 100 | 1 | 160 | 7.67 | 182.28 | 5,259 |
| Small | 100 | 1 | 319 | 7.45 | 182.28 | 5,259 |
| Medium | 100 | 1 | 279 | 7.26 | 182.28 | 5,259 |
| Tiny | 100 | 10 | 75 | 18.47 | 1,950.63 | 24,073 |
| Small | 100 | 10 | 128 | 18.43 | 1,950.63 | 24,073 |
| Medium | 100 | 10 | 129 | 18.85 | 1,950.63 | 24,073 |

## AppendIf Performance (No Conflict)

**AppendIf No Conflict Details**:
- **Attempted Events**: Number of events AppendIf operation tries to append (1 or 10 events per operation)
- **Past Events**: Number of existing events in database before benchmark (100 events for all scenarios)
- **Conflict Events**: 0 (no conflicts exist)
- **Business Logic**: Realistic course enrollment conditions using CourseOffered events

**Column Explanations**:
- **Dataset**: Test data size (Tiny: 5 courses/10 students, Small: 1K courses/10K students, Medium: 1K courses/10K students)
- **Concurrency**: Number of concurrent users/goroutines running operations simultaneously
- **Attempted Events**: Number of events the AppendIf operation tries to append per operation
- **Throughput (ops/sec)**: Operations completed per second (higher is better)
- **Latency (ms/op)**: Time per operation in milliseconds (lower is better)
- **Memory (KB/op)**: Memory allocated per operation in kilobytes (lower is better)
- **Allocations**: Number of memory allocations per operation (lower is better)

| Dataset | Concurrency | Attempted Events | Throughput (ops/sec) | Latency (ms/op) | Memory (KB/op) | Allocations |
|---------|-------------|------------------|---------------------|-----------------|---------------|-------------|
| Tiny | 1 | 1 | 4,100 | 0.56 | 4.76 | 96 |
| Small | 1 | 1 | 9,736 | 0.94 | 4.76 | 96 |
| Medium | 1 | 1 | 8,810 | 0.91 | 4.76 | 96 |
| Tiny | 1 | 10 | 2,340 | 2.15 | 37.55 | 295 |
| Small | 1 | 10 | 4,707 | 4.06 | 37.55 | 295 |
| Medium | 1 | 10 | 3,801 | 3.22 | 37.55 | 295 |
| Tiny | 100 | 1 | 100 | 22.24 | 552.15 | 9,550 |
| Small | 100 | 1 | 100 | 20.77 | 552.15 | 9,550 |
| Medium | 100 | 1 | 133 | 23.60 | 552.15 | 9,550 |
| Tiny | 100 | 10 | 44 | 64.65 | 3,844.81 | 29,421 |
| Small | 100 | 10 | 100 | 117.36 | 3,844.81 | 29,421 |
| Medium | 100 | 10 | 92 | 111.58 | 3,844.81 | 29,421 |

## AppendIf Performance (With Conflict)

**AppendIf With Conflict Details**:
- **Attempted Events**: Number of events AppendIf operation tries to append (1 or 10 events per operation)
- **Actual Events**: Number of events successfully appended (0 - all operations fail due to conflicts)
- **Past Events**: Number of existing events in database before benchmark (100 events for all scenarios)
- **Conflict Events**: Number of conflicting events created before AppendIf (1 event for all scenarios)
- **Business Logic**: Realistic course enrollment conflicts using CourseOffered events with matching course IDs

**Column Explanations**:
- **Dataset**: Test data size (Tiny: 5 courses/10 students, Small: 1K courses/10K students, Medium: 1K courses/10K students)
- **Concurrency**: Number of concurrent users/goroutines running operations simultaneously
- **Attempted Events**: Number of events the AppendIf operation tries to append per operation
- **Conflict Events**: Number of conflicting events created before the AppendIf operation (causes all operations to fail)
- **Throughput (ops/sec)**: Operations completed per second (higher is better, but all fail due to conflicts)
- **Latency (ms/op)**: Time per operation in milliseconds (lower is better)
- **Memory (KB/op)**: Memory allocated per operation in kilobytes (lower is better)
- **Allocations**: Number of memory allocations per operation (lower is better)

| Dataset | Concurrency | Attempted Events | Conflict Events | Throughput (ops/sec) | Latency (ms/op) | Memory (KB/op) | Allocations |
|---------|-------------|------------------|-----------------|---------------------|-----------------|---------------|-------------|
| Tiny | 1 | 1 | 1 | 1,882 | 0.74 | 7.20 | 144 |
| Small | 1 | 1 | 1 | 4,442 | 1.21 | 7.20 | 144 |
| Medium | 1 | 1 | 1 | 2,976 | 0.95 | 7.20 | 144 |
| Tiny | 1 | 10 | 1 | 1,538 | 1.98 | 39.95 | 343 |
| Small | 1 | 10 | 1 | 2,446 | 2.79 | 39.95 | 343 |
| Medium | 1 | 10 | 1 | 3,140 | 3.38 | 39.95 | 343 |
| Tiny | 100 | 1 | 1 | 100 | 18.56 | 466.60 | 9,309 |
| Small | 100 | 1 | 1 | 130 | 21.40 | 466.60 | 9,309 |
| Medium | 100 | 1 | 1 | 100 | 22.38 | 466.60 | 9,309 |
| Tiny | 100 | 10 | 1 | 58 | 80.65 | 3,837.22 | 29,261 |
| Small | 100 | 10 | 1 | 100 | 113.05 | 3,837.22 | 29,261 |
| Medium | 100 | 10 | 1 | 100 | 126.33 | 3,837.22 | 29,261 |

## Projection Performance

**Projection Operations Details**:
- **Operation**: State reconstruction from realistic business event streams
- **Scenario**: Building aggregate state from CourseOffered, StudentRegistered, EnrollmentCompleted events
- **Events**: Number of events processed during projection (varies by dataset)
- **Business Logic**: Realistic projectors counting courses, students, and enrollments

**Column Explanations**:
- **Operation**: Type of projection operation (Project = single-threaded, ProjectStream = streaming with channels)
- **Dataset**: Test data size (Tiny: 5 courses/10 students, Small: 1K courses/10K students, Medium: 1K courses/10K students)
- **Concurrency**: Number of concurrent users/goroutines running operations simultaneously
- **Events**: Number of events processed during projection (varies by dataset size)
- **Throughput (ops/sec)**: Operations completed per second (higher is better)
- **Latency (ms/op)**: Time per operation in milliseconds (lower is better)
- **Memory (KB/op)**: Memory allocated per operation in kilobytes (lower is better)
- **Allocations**: Number of memory allocations per operation (lower is better)

| Operation | Dataset | Concurrency | Events | Throughput (ops/sec) | Latency (ms/op) | Memory (KB/op) | Allocations |
|-----------|---------|-------------|--------|---------------------|-----------------|---------------|-------------|
| **ProjectStream** | Tiny | 1 | ~100 | 5,277 | 0.23 | 78.13 | 1,489 |
| **ProjectStream** | Small | 1 | ~100 | 10,000 | 0.21 | 78.13 | 1,489 |
| **ProjectStream** | Medium | 1 | ~100 | 10,000 | 0.21 | 78.13 | 1,489 |
| **Project** | Tiny | 1 | ~100 | 3,219 | 0.31 | 66.95 | 1,486 |
| **Project** | Small | 1 | ~100 | 8,160 | 0.31 | 66.95 | 1,486 |
| **Project** | Medium | 1 | ~100 | 6,702 | 0.30 | 66.95 | 1,486 |
| **ProjectStream** | Tiny | 100 | ~100 | 136 | 8.33 | 7,808.23 | 148,785 |
| **ProjectStream** | Small | 100 | ~100 | 315 | 7.38 | 7,808.23 | 148,785 |
| **ProjectStream** | Medium | 100 | ~100 | 321 | 7.45 | 7,808.23 | 148,785 |
| **Project** | Tiny | 100 | ~100 | 98 | 10.68 | 6,691.41 | 148,496 |
| **Project** | Small | 100 | ~100 | 206 | 10.16 | 6,691.41 | 148,496 |
| **Project** | Medium | 100 | ~100 | 236 | 10.18 | 6,691.41 | 148,496 |

## ProjectionLimits Performance

**ProjectionLimits Operations Details**:
- **Operation**: Testing projection concurrency limits with realistic business events
- **Scenario**: Course enrollment system with limited concurrent projections (MaxConcurrentProjections: 5)
- **Events**: Number of events processed during projection (varies by dataset)
- **Business Logic**: Realistic projectors counting courses with proper success/limit rate validation

**Column Explanations**:
- **Operation**: Type of projection operation (ProjectionLimits = testing concurrency limits)
- **Dataset**: Test data size (Tiny: 5 courses/10 students, Small: 1K courses/10K students, Medium: 1K courses/10K students)
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
| **ProjectionLimits** | Tiny | 5 | ~100 | 3,417 | 0.40 | 23.86 | 488 | 1.000 | 0.000 |
| **ProjectionLimits** | Small | 5 | ~100 | 6,178 | 0.39 | 23.86 | 488 | 1.000 | 0.000 |
| **ProjectionLimits** | Medium | 5 | ~100 | 6,036 | 0.39 | 23.86 | 488 | 1.000 | 0.000 |
| **ProjectionLimits** | Tiny | 8 | ~100 | 2,935 | 0.42 | 34.69 | 716 | 0.625 | 0.375 |
| **ProjectionLimits** | Small | 8 | ~100 | 6,288 | 0.42 | 34.69 | 716 | 0.625 | 0.375 |
| **ProjectionLimits** | Medium | 8 | ~100 | 5,647 | 0.41 | 34.69 | 716 | 0.625 | 0.375 |
| **ProjectionLimits** | Tiny | 10 | ~100 | 2,634 | 0.46 | 45.23 | 938 | 0.500 | 0.500 |
| **ProjectionLimits** | Small | 10 | ~100 | 5,696 | 0.45 | 45.23 | 938 | 0.500 | 0.500 |
| **ProjectionLimits** | Medium | 10 | ~100 | 5,673 | 0.45 | 45.23 | 938 | 0.500 | 0.500 |
