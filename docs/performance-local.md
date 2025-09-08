# Local PostgreSQL Performance

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
- **Events**: Single event (1) or batch (10 events)
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
| Medium | 1 | 1 | 4,100 | 0.24 | 1.83 | 55 |
| Tiny | 1 | 1 | 4,110 | 0.24 | 1.83 | 55 |
| Small | 1 | 1 | 4,380 | 0.23 | 1.85 | 56 |
| Medium | 1 | 10 | 2,170 | 0.46 | 19.54 | 244 |
| Tiny | 1 | 10 | 2,200 | 0.45 | 19.54 | 244 |
| Small | 1 | 10 | 2,180 | 0.46 | 19.54 | 244 |
| Tiny | 100 | 1 | 339 | 2.95 | 178.41 | 5,282 |
| Medium | 100 | 1 | 345 | 2.95 | 178.22 | 5,282 |
| Small | 100 | 1 | 332 | 3.01 | 178.31 | 5,280 |
| Tiny | 100 | 10 | 135 | 7.41 | 1,950.63 | 24,079 |
| Medium | 100 | 10 | 133 | 7.52 | 1,948.42 | 24,065 |
| Small | 100 | 10 | 135 | 7.41 | 1,950.63 | 24,077 |

## AppendIf Performance (No Conflict)

**AppendIf No Conflict Details**:
- **Attempted Events**: Number of events AppendIf operation tries to append (1 or 10 events per operation)
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
| Medium | 1 | 1 | 1,319 | 0.76 | 4.36 | 95 |
| Tiny | 1 | 1 | 1,164 | 0.86 | 4.36 | 96 |
| Small | 1 | 1 | 864 | 1.16 | 4.36 | 96 |
| Medium | 1 | 10 | 1,319 | 0.76 | 21.52 | 283 |
| Tiny | 1 | 10 | 1,164 | 0.86 | 21.52 | 283 |
| Small | 1 | 10 | 864 | 1.16 | 21.52 | 283 |
| Medium | 100 | 1 | 147 | 6.80 | 430.47 | 9,259 |
| Tiny | 100 | 1 | 151 | 6.62 | 430.95 | 9,263 |
| Small | 100 | 1 | 128 | 7.81 | 430.68 | 9,262 |
| Medium | 100 | 10 | 100 | 10.00 | 2,144.14 | 27,979 |
| Tiny | 100 | 10 | 100 | 10.00 | 2,147.43 | 28,012 |
| Small | 100 | 10 | 100 | 10.00 | 2,145.13 | 27,995 |

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
| Medium | 1 | 1 | 1 | 1,179 | 0.85 | 5.73 | 145 |
| Tiny | 1 | 1 | 1 | 1,080 | 0.93 | 5.73 | 146 |
| Small | 1 | 1 | 1 | 1,180 | 0.85 | 5.73 | 145 |
| Medium | 1 | 10 | 1 | 1,036 | 0.97 | 22.90 | 332 |
| Tiny | 1 | 10 | 1 | 1,017 | 0.98 | 22.91 | 332 |
| Small | 1 | 10 | 1 | 1,036 | 0.97 | 22.90 | 332 |
| Medium | 100 | 1 | 1 | 345 | 2.90 | 422.87 | 9,532 |
| Tiny | 100 | 1 | 1 | 355 | 2.82 | 423.43 | 9,537 |
| Small | 100 | 1 | 1 | 345 | 2.90 | 422.87 | 9,532 |
| Medium | 100 | 10 | 1 | 205 | 4.88 | 2,138.20 | 28,123 |
| Tiny | 100 | 10 | 1 | 200 | 5.00 | 2,141.18 | 28,142 |
| Small | 100 | 10 | 1 | 205 | 4.88 | 2,138.20 | 28,123 |

## Projection Performance

**Projection Operations Details**:
- **Operation**: State reconstruction from event streams
- **Scenario**: Building aggregate state from historical events
- **Events**: Number of events processed during projection (varies by dataset)

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
| **ProjectStream** | Tiny | 1 | ~100 | 5,665 | 0.18 | 62.96 | 1,460 |
| **ProjectStream** | Small | 1 | ~100 | 5,665 | 0.18 | 62.96 | 1,460 |
| **ProjectStream** | Medium | 1 | ~100 | 5,665 | 0.18 | 62.96 | 1,460 |
| **Project** | Tiny | 1 | ~100 | 3,564 | 0.28 | 54.33 | 1,457 |
| **Project** | Small | 1 | ~100 | 3,564 | 0.28 | 54.33 | 1,457 |
| **Project** | Medium | 1 | ~100 | 3,564 | 0.28 | 54.33 | 1,457 |
| **ProjectStream** | Tiny | 100 | ~100 | 376 | 2.66 | 6,293.20 | 145,882 |
| **ProjectStream** | Small | 100 | ~100 | 376 | 2.66 | 6,293.20 | 145,883 |
| **ProjectStream** | Medium | 100 | ~100 | 376 | 2.66 | 6,293.20 | 145,883 |
| **Project** | Tiny | 100 | ~100 | 280 | 3.57 | 5,561.47 | 145,586 |
| **Project** | Small | 100 | ~100 | 280 | 3.57 | 5,561.47 | 145,593 |
| **Project** | Medium | 100 | ~100 | 280 | 3.57 | 5,561.47 | 145,590 |
