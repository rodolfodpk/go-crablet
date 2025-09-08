# Docker PostgreSQL Performance

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
| Tiny | 1 | 1 | 2,142 | 0.47 | 1.88 | 56 |
| Small | 1 | 1 | 2,348 | 0.43 | 1.88 | 56 |
| Medium | 1 | 1 | 2,175 | 0.46 | 1.88 | 56 |
| Tiny | 1 | 10 | 1,989 | 0.50 | 19.54 | 244 |
| Small | 1 | 10 | 2,014 | 0.50 | 19.54 | 244 |
| Medium | 1 | 10 | 1,778 | 0.56 | 19.54 | 244 |
| Tiny | 100 | 1 | 186 | 5.38 | 182.28 | 5,259 |
| Small | 100 | 1 | 177 | 5.64 | 182.28 | 5,259 |
| Medium | 100 | 1 | 171 | 5.84 | 182.28 | 5,259 |
| Tiny | 100 | 10 | 100 | 10.00 | 1,950.63 | 24,073 |
| Small | 100 | 10 | 100 | 10.00 | 1,950.63 | 24,073 |
| Medium | 100 | 10 | 100 | 10.00 | 1,950.63 | 24,073 |

## AppendIf Performance (No Conflict)

**AppendIf No Conflict Details**:
- **Attempted Events**: Number of events AppendIf operation tries to append (1 or 10 events per operation)
- **Past Events**: Number of existing events in database before benchmark (100 events for all scenarios)
- **Conflict Events**: 0 (no conflicts exist)
- **Business Logic**: Course enrollment system with realistic business rule validation

**Column Explanations**:
- **Dataset**: Test data size (Tiny: 5 courses/10 students, Small: 500 courses/5K students, Medium: 1K courses/10K students)
- **Concurrency**: Number of concurrent users/goroutines running operations simultaneously
- **Attempted Events**: Number of events AppendIf operation tries to append (1 or 10 events per operation)
- **Throughput (ops/sec)**: Operations completed per second (higher is better)
- **Latency (ms/op)**: Time per operation in milliseconds (lower is better)
- **Memory (KB/op)**: Memory allocated per operation in kilobytes (lower is better)
- **Allocations**: Number of memory allocations per operation (lower is better)

| Dataset | Concurrency | Attempted Events | Throughput (ops/sec) | Latency (ms/op) | Memory (KB/op) | Allocations |
|---------|-------------|------------------|---------------------|-----------------|---------------|-------------|
| Tiny | 1 | 1 | 2,139 | 0.47 | 4.76 | 96 |
| Small | 1 | 1 | 2,172 | 0.46 | 4.76 | 96 |
| Medium | 1 | 1 | 2,020 | 0.50 | 4.76 | 96 |
| Tiny | 1 | 10 | 1,791 | 0.56 | 37.55 | 295 |
| Small | 1 | 10 | 1,716 | 0.58 | 37.55 | 295 |
| Medium | 1 | 10 | 1,605 | 0.62 | 37.55 | 295 |
| Tiny | 100 | 1 | 100 | 10.00 | 552.15 | 9,550 |
| Small | 100 | 1 | 100 | 10.00 | 552.15 | 9,550 |
| Medium | 100 | 1 | 100 | 10.00 | 552.15 | 9,550 |
| Tiny | 100 | 10 | 94 | 10.64 | 3,844.81 | 29,421 |
| Small | 100 | 10 | 97 | 10.31 | 3,844.81 | 29,421 |
| Medium | 100 | 10 | 99 | 10.10 | 3,844.81 | 29,421 |

## AppendIf Performance (With Conflict)

**AppendIf With Conflict Details**:
- **Attempted Events**: Number of events AppendIf operation tries to append (1 or 10 events per operation)
- **Actual Events**: Number of events successfully appended (0 - all operations fail due to conflicts)
- **Past Events**: Number of existing events in database before benchmark (100 events for all scenarios)
- **Conflict Events**: Number of conflicting events created before AppendIf (1, 10, or 100 events, matching concurrency level)

**Column Explanations**:
- **Dataset**: Test data size (Tiny: 5 courses/10 students, Small: 500 courses/5K students, Medium: 1K courses/10K students)
- **Concurrency**: Number of concurrent users/goroutines running operations simultaneously
- **Attempted Events**: Number of events AppendIf operation tries to append (1 or 10 events per operation)
- **Conflict Events**: Number of conflicting events created before AppendIf (1, 10, or 100 events, matching concurrency level)
- **Throughput (ops/sec)**: Operations completed per second (higher is better, but all fail due to conflicts)
- **Latency (ms/op)**: Time per operation in milliseconds (lower is better)
- **Memory (KB/op)**: Memory allocated per operation in kilobytes (lower is better)
- **Allocations**: Number of memory allocations per operation (lower is better)

| Dataset | Concurrency | Attempted Events | Conflict Events | Throughput (ops/sec) | Latency (ms/op) | Memory (KB/op) | Allocations |
|---------|-------------|------------------|-----------------|---------------------|-----------------|---------------|-------------|
| Tiny | 1 | 1 | 1 | 1,026 | 0.97 | 7.20 | 144 |
| Small | 1 | 1 | 1 | 1,140 | 0.88 | 7.20 | 144 |
| Medium | 1 | 1 | 1 | 1,027 | 0.97 | 7.20 | 144 |
| Tiny | 1 | 10 | 1 | 943 | 1.06 | 39.95 | 343 |
| Small | 1 | 10 | 1 | 963 | 1.04 | 39.95 | 343 |
| Medium | 1 | 10 | 1 | 934 | 1.07 | 39.95 | 343 |
| Tiny | 100 | 1 | 1 | 100 | 10.00 | 466.60 | 9,309 |
| Small | 100 | 1 | 1 | 100 | 10.00 | 466.60 | 9,309 |
| Medium | 100 | 1 | 1 | 100 | 10.00 | 466.60 | 9,309 |
| Tiny | 100 | 10 | 1 | 66 | 15.15 | 3,837.22 | 29,261 |
| Small | 100 | 10 | 1 | 67 | 14.93 | 3,837.22 | 29,261 |
| Medium | 100 | 10 | 1 | 54 | 18.52 | 3,837.22 | 29,261 |

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
| **ProjectStream** | Tiny | 1 | ~100 | 3,586 | 0.28 | 78.13 | 1,489 |
| **ProjectStream** | Small | 1 | ~100 | 3,628 | 0.28 | 78.13 | 1,489 |
| **ProjectStream** | Medium | 1 | ~100 | 3,369 | 0.30 | 78.13 | 1,489 |
| **Project** | Tiny | 1 | ~100 | 1,558 | 0.64 | 66.95 | 1,486 |
| **Project** | Small | 1 | ~100 | 1,464 | 0.68 | 66.95 | 1,486 |
| **Project** | Medium | 1 | ~100 | 1,419 | 0.70 | 66.95 | 1,486 |
| **ProjectStream** | Tiny | 100 | ~100 | 100 | 10.00 | 7,808.23 | 148,785 |
| **ProjectStream** | Small | 100 | ~100 | 100 | 10.00 | 7,808.23 | 148,785 |
| **ProjectStream** | Medium | 100 | ~100 | 100 | 10.00 | 7,808.23 | 148,785 |
| **Project** | Tiny | 100 | ~100 | 91 | 10.99 | 6,691.41 | 148,496 |
| **Project** | Small | 100 | ~100 | 87 | 11.49 | 6,691.41 | 148,496 |
| **Project** | Medium | 100 | ~100 | 90 | 11.11 | 6,691.41 | 148,496 |

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
| **QueryStream** | Tiny | 1 | ~100 | 4,539 | 0.22 | 78.13 | 1,489 |
| **QueryStream** | Small | 1 | ~100 | 4,598 | 0.22 | 78.13 | 1,489 |
| **QueryStream** | Medium | 1 | ~100 | 4,932 | 0.20 | 78.13 | 1,489 |
| **Query** | Tiny | 1 | ~100 | 2,041 | 0.49 | 66.95 | 1,486 |
| **Query** | Small | 1 | ~100 | 2,594 | 0.39 | 66.95 | 1,486 |
| **Query** | Small | 1 | ~100 | 2,038 | 0.49 | 66.95 | 1,486 |
| **QueryStream** | Tiny | 100 | ~100 | 212 | 4.72 | 7,808.23 | 148,785 |
| **QueryStream** | Small | 100 | ~100 | 229 | 4.37 | 7,808.23 | 148,785 |
| **QueryStream** | Medium | 100 | ~100 | 232 | 4.31 | 7,808.23 | 148,785 |
| **Query** | Tiny | 100 | ~100 | 139 | 7.19 | 6,691.41 | 148,496 |
| **Query** | Small | 100 | ~100 | 126 | 7.94 | 6,691.41 | 148,496 |
| **Query** | Medium | 100 | ~100 | 146 | 6.85 | 6,691.41 | 148,496 |

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
| **ProjectionLimits** | Tiny | 5 | ~100 | 1,281 | 0.78 | 23.86 | 488 | 1.000 | 0.000 |
| **ProjectionLimits** | Small | 5 | ~100 | 1,344 | 0.74 | 23.86 | 488 | 1.000 | 0.000 |
| **ProjectionLimits** | Medium | 5 | ~100 | 1,294 | 0.77 | 23.86 | 488 | 1.000 | 0.000 |
| **ProjectionLimits** | Tiny | 8 | ~100 | 1,294 | 0.77 | 34.69 | 716 | 0.625 | 0.375 |
| **ProjectionLimits** | Small | 8 | ~100 | 1,242 | 0.81 | 34.69 | 716 | 0.625 | 0.375 |
| **ProjectionLimits** | Medium | 8 | ~100 | 1,249 | 0.80 | 34.69 | 716 | 0.625 | 0.375 |
| **ProjectionLimits** | Tiny | 10 | ~100 | 1,231 | 0.81 | 45.23 | 938 | 0.500 | 0.500 |
| **ProjectionLimits** | Small | 10 | ~100 | 1,233 | 0.81 | 45.23 | 938 | 0.500 | 0.500 |
| **ProjectionLimits** | Medium | 10 | ~100 | 1,090 | 0.92 | 45.23 | 938 | 0.500 | 0.500 |