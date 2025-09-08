# Docker PostgreSQL Performance

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
- **Operation**: Simple event append operations
- **Scenario**: Basic event writing without conditions or business logic
- **Events**: Single event (1) or batch (100 events)
- **Model**: Generic test events with simple JSON data

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
| Tiny | 1 | 1 | 2,406 | 0.42 | 1.83 | 55 |
| Medium | 1 | 1 | 2,132 | 0.47 | 1.83 | 55 |
| Small | 1 | 1 | 2,110 | 0.45 | 1.86 | 55 |
| Medium | 1 | 10 | 1,672 | 0.60 | 19.54 | 243 |
| Small | 1 | 10 | 1,704 | 0.59 | 19.54 | 243 |
| Tiny | 1 | 10 | 1,919 | 0.52 | 19.54 | 243 |
| Tiny | 100 | 1 | 153 | 6.54 | 178.12 | 5,258 |
| Medium | 100 | 1 | 152 | 6.58 | 178.00 | 5,258 |
| Small | 100 | 1 | 163 | 6.14 | 178.00 | 5,257 |
| Tiny | 100 | 10 | 100 | 10.00 | 1,949.22 | 24,067 |
| Medium | 100 | 10 | 100 | 10.00 | 1,948.41 | 24,060 |
| Small | 100 | 10 | 100 | 10.00 | 1,949.22 | 24,076 |

## AppendIf Performance (No Conflict)

**AppendIf No Conflict Details**:
- **Attempted Events**: Number of events AppendIf operation tries to append (1 or 10 events per operation)
- **Actual Events**: Number of events successfully appended (1 or 10 events)
- **Past Events**: Number of existing events in database before benchmark (100 events for all scenarios)
- **Conflict Events**: 0 (no conflicts exist)

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
| Medium | 1 | 1 | 2,061 | 0.49 | 4.36 | 95 |
| Tiny | 1 | 1 | 2,054 | 0.49 | 4.36 | 95 |
| Small | 1 | 1 | 1,858 | 0.54 | 4.36 | 95 |
| Medium | 1 | 10 | 1,848 | 0.54 | 21.51 | 282 |
| Tiny | 1 | 10 | 1,749 | 0.57 | 21.52 | 283 |
| Small | 1 | 10 | 1,776 | 0.56 | 21.51 | 283 |
| Tiny | 100 | 1 | 100 | 10.00 | 431.11 | 9,263 |
| Medium | 100 | 1 | 100 | 10.00 | 430.38 | 9,259 |
| Small | 100 | 1 | 100 | 10.00 | 430.95 | 9,261 |
| Tiny | 100 | 10 | 100 | 10.00 | 2,145.39 | 27,995 |
| Medium | 100 | 10 | 100 | 10.00 | 2,143.37 | 27,976 |
| Small | 100 | 10 | 100 | 10.00 | 2,143.37 | 27,990 |

## AppendIf Performance (With Conflict)

**AppendIf With Conflict Details**:
- **Attempted Events**: Number of events AppendIf operation tries to append (1 or 10 events per operation)
- **Actual Events**: Number of events successfully appended (0 - all operations fail due to conflicts)
- **Past Events**: Number of existing events in database before benchmark (100 events for all scenarios)
- **Conflict Events**: Number of conflicting events created before AppendIf (1 event for all scenarios)

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
| Medium | 1 | 1 | 1 | 1,088 | 0.92 | 5.70 | 145 |
| Tiny | 1 | 1 | 1 | 1,132 | 0.88 | 5.71 | 145 |
| Small | 1 | 1 | 1 | 1,171 | 0.85 | 5.71 | 145 |
| Medium | 1 | 10 | 1 | 1,033 | 0.97 | 22.87 | 331 |
| Tiny | 1 | 10 | 1 | 1,018 | 0.98 | 22.87 | 331 |
| Small | 1 | 10 | 1 | 1,134 | 0.88 | 22.87 | 331 |
| Medium | 100 | 1 | 1 | 154 | 6.49 | 422.55 | 9,501 |
| Tiny | 100 | 1 | 1 | 153 | 6.54 | 422.55 | 9,504 |
| Small | 100 | 1 | 1 | 166 | 6.02 | 422.55 | 9,505 |
| Medium | 100 | 10 | 1 | 100 | 10.00 | 2,133.54 | 28,117 |
| Tiny | 100 | 10 | 1 | 100 | 10.00 | 2,135.74 | 28,135 |
| Small | 100 | 10 | 1 | 115 | 8.70 | 2,133.54 | 28,120 |

## Projection Performance

**Projection Operations Details**:
- **Operation**: State reconstruction from event streams
- **Scenario**: Building aggregate state from historical events
- **Events**: Number of events processed during projection (varies by dataset)
- **Model**: Domain-specific state reconstruction with business logic

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
| **ProjectStream** | Tiny | 1 | ~100 | 4,044 | 0.25 | 62.96 | 1,460 |
| **ProjectStream** | Medium | 1 | ~100 | 3,992 | 0.25 | 62.96 | 1,460 |
| **ProjectStream** | Small | 1 | ~100 | 4,022 | 0.25 | 62.96 | 1,460 |
| **Project** | Tiny | 1 | ~100 | 1,766 | 0.57 | 54.33 | 1,457 |
| **Project** | Medium | 1 | ~100 | 1,630 | 0.61 | 54.33 | 1,457 |
| **Project** | Small | 1 | ~100 | 1,704 | 0.59 | 54.33 | 1,457 |
| **ProjectStream** | Medium | 100 | ~100 | 134 | 7.46 | 6,293.20 | 145,881 |
| **ProjectStream** | Small | 100 | ~100 | 134 | 7.46 | 6,293.20 | 145,881 |
| **ProjectStream** | Tiny | 100 | ~100 | 134 | 7.46 | 6,293.20 | 145,882 |
| **Project** | Medium | 100 | ~100 | 94 | 10.64 | 5,561.47 | 145,590 |
| **Project** | Small | 100 | ~100 | 100 | 10.00 | 5,561.47 | 145,576 |
| **Project** | Tiny | 100 | ~100 | 100 | 10.00 | 5,561.47 | 145,576 |

## Course Registration Performance

**Course Registration Details**:
- **Operation**: Course registration events (StudentCourseRegistration)
- **Scenario**: Multiple students simultaneously registering for courses
- **Events**: 1 event per user (course registration)
- **Model**: Domain-specific business scenario with realistic data

**Column Explanations**:
- **Operation**: Type of course registration operation (Concurrent_1User, Concurrent_10Users, Concurrent_100Users)
- **Dataset**: Test data size (Small: 1K courses/10K students, Medium: 1K courses/10K students)
- **Concurrency**: Number of concurrent users/goroutines running operations simultaneously
- **Events**: Number of events per operation (- indicates variable based on concurrency level)
- **Throughput (ops/sec)**: Operations completed per second (higher is better)
- **Latency (ns/op)**: Time per operation in nanoseconds (lower is better)
- **Memory (B/op)**: Memory allocated per operation in bytes (lower is better)
- **Allocations**: Number of memory allocations per operation (lower is better)

| Operation | Dataset | Concurrency | Events | Throughput (ops/sec) | Latency (ms/op) | Memory (KB/op) | Allocations |
|-----------|---------|-------------|--------|---------------------|-----------------|---------------|-------------|
| **Concurrent_1User** | Small | 1 | - | 1,210 | 225,217 | 2,537 | 51 |
| **Concurrent_10Users** | Small | 10 | - | 1,208 | 807,331 | 26,033 | 530 |
| **Concurrent_100Users** | Medium | 100 | - | 146 | 6,854,788 | 269,465 | 5,543 |
