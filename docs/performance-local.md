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
| Tiny | 1 | 1 | 8,668 | 0.12 | 1.88 | 56 |
| Small | 1 | 1 | 9,096 | 0.11 | 1.88 | 56 |
| Medium | 1 | 1 | 9,061 | 0.11 | 1.88 | 56 |
| Tiny | 1 | 10 | 5,392 | 0.19 | 19.54 | 244 |
| Small | 1 | 10 | 5,131 | 0.20 | 19.54 | 244 |
| Medium | 1 | 10 | 5,650 | 0.18 | 19.54 | 244 |
| Tiny | 100 | 1 | 307 | 3.26 | 182.28 | 5,259 |
| Small | 100 | 1 | 310 | 3.23 | 182.28 | 5,259 |
| Medium | 100 | 1 | 307 | 3.26 | 182.28 | 5,259 |
| Tiny | 100 | 10 | 122 | 8.20 | 1,950.63 | 24,073 |
| Small | 100 | 10 | 122 | 8.20 | 1,950.63 | 24,073 |
| Medium | 100 | 10 | 124 | 8.06 | 1,950.63 | 24,073 |

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
| Tiny | 1 | 1 | 7,604 | 0.13 | 4.76 | 96 |
| Small | 1 | 1 | 7,041 | 0.14 | 4.76 | 96 |
| Medium | 1 | 1 | 7,290 | 0.14 | 4.76 | 96 |
| Tiny | 1 | 10 | 4,269 | 0.23 | 37.55 | 295 |
| Small | 1 | 10 | 3,667 | 0.27 | 37.55 | 295 |
| Medium | 1 | 10 | 3,615 | 0.28 | 37.55 | 295 |
| Tiny | 100 | 1 | 124 | 8.06 | 552.15 | 9,550 |
| Small | 100 | 1 | 141 | 7.09 | 552.15 | 9,550 |
| Medium | 100 | 1 | 126 | 7.94 | 552.15 | 9,550 |
| Tiny | 100 | 10 | 100 | 10.00 | 3,844.81 | 29,421 |
| Small | 100 | 10 | 100 | 10.00 | 3,844.81 | 29,421 |
| Medium | 100 | 10 | 100 | 10.00 | 3,844.81 | 29,421 |

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
| Tiny | 1 | 1 | 1 | 4,221 | 0.24 | 7.20 | 144 |
| Small | 1 | 1 | 1 | 4,021 | 0.25 | 7.20 | 144 |
| Medium | 1 | 1 | 1 | 4,179 | 0.24 | 7.20 | 144 |
| Tiny | 1 | 10 | 1 | 2,301 | 0.43 | 39.95 | 343 |
| Small | 1 | 10 | 1 | 2,827 | 0.35 | 39.95 | 343 |
| Medium | 1 | 10 | 1 | 2,410 | 0.41 | 39.95 | 343 |
| Tiny | 100 | 1 | 1 | 129 | 7.75 | 466.60 | 9,309 |
| Small | 100 | 1 | 1 | 130 | 7.69 | 466.60 | 9,309 |
| Medium | 100 | 1 | 1 | 126 | 7.94 | 466.60 | 9,309 |
| Tiny | 100 | 10 | 1 | 100 | 10.00 | 3,837.22 | 29,261 |
| Small | 100 | 10 | 1 | 100 | 10.00 | 3,837.22 | 29,261 |
| Medium | 100 | 10 | 1 | 100 | 10.00 | 3,837.22 | 29,261 |

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
| **ProjectStream** | Tiny | 1 | ~100 | 8,398 | 0.12 | 78.13 | 1,489 |
| **ProjectStream** | Small | 1 | ~100 | 10,000 | 0.10 | 78.13 | 1,489 |
| **ProjectStream** | Medium | 1 | ~100 | 10,000 | 0.10 | 78.13 | 1,489 |
| **Project** | Tiny | 1 | ~100 | 6,082 | 0.16 | 66.95 | 1,486 |
| **Project** | Small | 1 | ~100 | 7,119 | 0.14 | 66.95 | 1,486 |
| **Project** | Medium | 1 | ~100 | 7,213 | 0.14 | 66.95 | 1,486 |
| **ProjectStream** | Tiny | 100 | ~100 | 267 | 3.75 | 7,808.23 | 148,785 |
| **ProjectStream** | Small | 100 | ~100 | 314 | 3.18 | 7,808.23 | 148,785 |
| **ProjectStream** | Medium | 100 | ~100 | 319 | 3.13 | 7,808.23 | 148,785 |
| **Project** | Tiny | 100 | ~100 | 201 | 4.98 | 6,691.41 | 148,496 |
| **Project** | Small | 100 | ~100 | 218 | 4.59 | 6,691.41 | 148,496 |
| **Project** | Medium | 100 | ~100 | 234 | 4.27 | 6,691.41 | 148,496 |

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
| **QueryStream** | Tiny | 1 | ~100 | 17,242 | 0.06 | 78.13 | 1,489 |
| **QueryStream** | Small | 1 | ~100 | 18,691 | 0.05 | 78.13 | 1,489 |
| **QueryStream** | Medium | 1 | ~100 | 19,407 | 0.05 | 78.13 | 1,489 |
| **Query** | Tiny | 1 | ~100 | 13,219 | 0.08 | 66.95 | 1,486 |
| **Query** | Small | 1 | ~100 | 13,236 | 0.08 | 66.95 | 1,486 |
| **Query** | Medium | 1 | ~100 | 13,242 | 0.08 | 66.95 | 1,486 |
| **QueryStream** | Tiny | 100 | ~100 | 552 | 1.81 | 7,808.23 | 148,785 |
| **QueryStream** | Small | 100 | ~100 | 532 | 1.88 | 7,808.23 | 148,785 |
| **QueryStream** | Medium | 100 | ~100 | 517 | 1.93 | 7,808.23 | 148,785 |
| **Query** | Tiny | 100 | ~100 | 352 | 2.84 | 6,691.41 | 148,496 |
| **Query** | Small | 100 | ~100 | 372 | 2.69 | 6,691.41 | 148,496 |
| **Query** | Medium | 100 | ~100 | 369 | 2.71 | 6,691.41 | 148,496 |

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
| **ProjectionLimits** | Tiny | 5 | ~100 | 6,093 | 0.16 | 23.86 | 488 | 1.000 | 0.000 |
| **ProjectionLimits** | Small | 5 | ~100 | 6,837 | 0.15 | 23.86 | 488 | 1.000 | 0.000 |
| **ProjectionLimits** | Medium | 5 | ~100 | 6,933 | 0.14 | 23.86 | 488 | 1.000 | 0.000 |
| **ProjectionLimits** | Tiny | 8 | ~100 | 5,941 | 0.18 | 34.69 | 716 | 0.625 | 0.375 |
| **ProjectionLimits** | Small | 8 | ~100 | 5,890 | 0.18 | 34.69 | 716 | 0.625 | 0.375 |
| **ProjectionLimits** | Medium | 8 | ~100 | 6,020 | 0.17 | 34.69 | 716 | 0.625 | 0.375 |
| **ProjectionLimits** | Tiny | 10 | ~100 | 4,831 | 0.21 | 45.23 | 938 | 0.500 | 0.500 |
| **ProjectionLimits** | Small | 10 | ~100 | 5,370 | 0.19 | 45.23 | 938 | 0.500 | 0.500 |
| **ProjectionLimits** | Medium | 10 | ~100 | 6,610 | 0.15 | 45.23 | 938 | 0.500 | 0.500 |