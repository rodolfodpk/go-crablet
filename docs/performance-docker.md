# Docker PostgreSQL Performance

## Performance Results

### Throughput Calculation

**Throughput (ops/sec)** represents the number of API operations completed per second, calculated as:
- **Formula**: `total_operations / elapsed_time_seconds`
- **Where**: `total_operations = benchmark_iterations × concurrency_level`
- **Example**: If a benchmark runs 1000 iterations with 10 concurrent users, total operations = 10,000
- **Measurement**: Uses Go's `testing.B.Elapsed()` for precise timing and `b.ReportMetric()` for reporting
- **Note**: This measures API calls per second, not individual events or database transactions

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
| Tiny | 1 | 1 | 2,302 | 1.05 | 1.87 | 55 |
| Medium | 1 | 1 | 2,238 | 1.06 | 1.88 | 55 |
| Small | 1 | 1 | 2,193 | 1.01 | 1.90 | 56 |
| Medium | 1 | 100 | 1,144 | 3.46 | 204.99 | 2,053 |
| Small | 1 | 100 | 1,003 | 3.76 | 204.74 | 2,053 |
| Tiny | 1 | 100 | 900 | 3.95 | 204.80 | 2,053 |
| Small | 100 | 1 | 174 | 5.75 | 178.55 | 5,260 |
| Medium | 100 | 1 | 176 | 6.72 | 178.00 | 5,256 |
| Tiny | 100 | 1 | 176 | 7.12 | 178.45 | 5,260 |

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
| Medium | 1 | 1 | 2,032 | 1.39 | 4.36 | 95 |
| Tiny | 1 | 1 | 1,941 | 1.38 | 4.36 | 95 |
| Small | 1 | 1 | 1,900 | 1.30 | 4.36 | 95 |
| Medium | 1 | 100 | 823 | 8.37 | 208.78 | 2,092 |
| Small | 1 | 100 | 789 | 9.98 | 208.96 | 2,093 |
| Tiny | 1 | 100 | 634 | 8.58 | 209.16 | 2,094 |
| Small | 100 | 1 | 100 | 10.00 | 430.16 | 9,262 |
| Tiny | 100 | 1 | 100 | 10.00 | 431.41 | 9,271 |
| Medium | 100 | 1 | 81 | 12.35 | 430.30 | 9,260 |
| Medium | 100 | 100 | 3 | 350,805,663 | 2,133,741 | 209,041 |
| Small | 100 | 100 | 3 | 311,342,023 | 2,134,947 | 209,075 |
| Tiny | 100 | 100 | 3 | 297,525,060 | 2,135,738 | 209,122 |

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
| Small | 1 | 1 | 1 | 1,219 | 2.25 | 5.75 | 145 |
| Tiny | 1 | 1 | 1 | 1,131 | 2.20 | 5.75 | 145 |
| Medium | 1 | 1 | 1 | 990 | 2.31 | 5.74 | 145 |
| Tiny | 1 | 100 | 1 | 946 | 2.71 | 210.56 | 2,143 |
| Small | 1 | 100 | 1 | 909 | 2.84 | 210.29 | 2,142 |
| Medium | 1 | 100 | 1 | 892 | 2.73 | 210.16 | 2,141 |
| Medium | 100 | 1 | 1 | 19 | 52.63 | 567.48 | 14,166 |
| Small | 100 | 1 | 1 | 18 | 55.56 | 567.70 | 14,172 |
| Tiny | 100 | 1 | 1 | 18 | 55.56 | 567.48 | 14,171 |


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
| **ProjectStream** | Tiny | 1 | ~100 | 4,266 | 0.66 | 62.96 | 1,460 |
| **ProjectStream** | Medium | 1 | ~100 | 4,256 | 0.63 | 62.96 | 1,460 |
| **ProjectStream** | Small | 1 | ~100 | 4,256 | 0.63 | 62.96 | 1,460 |
| **Project** | Tiny | 1 | ~100 | 4,072 | 0.64 | 54.16 | 1,450 |
| **Project** | Medium | 1 | ~100 | 4,075 | 0.61 | 54.18 | 1,450 |
| **Project** | Small | 1 | ~100 | 3,973 | 0.64 | 54.18 | 1,450 |
| **ProjectStream** | Medium | 100 | ~100 | 159 | 6.29 | 6,293.20 | 145,882 |
| **ProjectStream** | Small | 100 | ~100 | 156 | 6.41 | 6,293.41 | 145,881 |
| **ProjectStream** | Tiny | 100 | ~100 | 154 | 6.50 | 6,293.20 | 145,882 |
| **Project** | Medium | 100 | ~100 | 154 | 6.49 | 5,414.13 | 144,895 |
| **Project** | Tiny | 100 | ~100 | 154 | 6.50 | 5,414.02 | 144,897 |
| **Project** | Small | 100 | ~100 | 152 | 6.58 | 5,414.13 | 144,897 |

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
