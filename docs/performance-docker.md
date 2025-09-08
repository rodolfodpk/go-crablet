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
| Tiny | 1 | 1 | 2,241 | 1.08 | 1.83 | 55 |
| Medium | 1 | 1 | 2,154 | 1.07 | 1.83 | 55 |
| Small | 1 | 1 | 2,013 | 1.02 | 1.86 | 55 |
| Tiny | 1 | 10 | 1,905 | 1.34 | 19.54 | 243 |
| Medium | 1 | 10 | 1,876 | 1.27 | 19.54 | 243 |
| Small | 1 | 10 | 1,810 | 1.22 | 19.54 | 243 |
| Tiny | 100 | 1 | 171 | 5.85 | 178.34 | 5,258 |
| Small | 100 | 1 | 172 | 5.81 | 178.56 | 5,259 |
| Medium | 100 | 1 | 156 | 6.41 | 178.08 | 5,256 |
| Medium | 100 | 10 | 100 | 10.00 | 1,948.80 | 24,065 |
| Tiny | 100 | 10 | 100 | 10.00 | 1,949.41 | 24,072 |
| Small | 100 | 10 | 97 | 10.31 | 1,948.70 | 24,070 |

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
| Tiny | 1 | 1 | 2,167 | 1.44 | 4.36 | 95 |
| Medium | 1 | 1 | 2,038 | 1.39 | 4.36 | 95 |
| Small | 1 | 1 | 1,719 | 1.39 | 4.36 | 95 |
| Tiny | 1 | 10 | 1,876 | 3.14 | 21.53 | 283 |
| Medium | 1 | 10 | 1,754 | 2.65 | 21.51 | 282 |
| Small | 1 | 10 | 1,707 | 2.98 | 21.53 | 283 |
| Medium | 100 | 1 | 100 | 10.00 | 430.36 | 9,260 |
| Tiny | 100 | 1 | 100 | 10.00 | 431.58 | 9,267 |
| Small | 100 | 1 | 86 | 11.63 | 431.37 | 9,262 |
| Medium | 100 | 10 | 92 | 10.87 | 2,143.37 | 27,977 |
| Tiny | 100 | 10 | 87 | 11.49 | 2,147.07 | 28,013 |
| Small | 100 | 10 | 69 | 14.49 | 2,144.35 | 27,994 |
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
| Small | 1 | 1 | 1 | 1,202 | 2.27 | 5.71 | 145 |
| Tiny | 1 | 1 | 1 | 1,131 | 2.26 | 5.71 | 145 |
| Medium | 1 | 1 | 1 | 1,089 | 2.35 | 5.70 | 145 |
| Small | 1 | 10 | 1 | 1,185 | 2.39 | 22.88 | 331 |
| Tiny | 1 | 10 | 1 | 1,132 | 2.38 | 22.88 | 331 |
| Medium | 1 | 10 | 1 | 1,140 | 2.38 | 22.86 | 331 |
| Small | 100 | 1 | 1 | 178 | 5.62 | 422.98 | 9,504 |
| Tiny | 100 | 1 | 1 | 172 | 5.81 | 422.91 | 9,506 |
| Medium | 100 | 1 | 1 | 160 | 6.25 | 422.28 | 9,499 |
| Small | 100 | 10 | 1 | 160 | 6.25 | 2,137.41 | 28,125 |
| Tiny | 100 | 10 | 1 | 152 | 6.58 | 2,139.27 | 28,145 |
| Medium | 100 | 10 | 1 | 147 | 6.80 | 2,136.25 | 28,114 |


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
| **ProjectStream** | Medium | 1 | ~100 | 4,256 | 0.63 | 62.96 | 1,460 |
| **Project** | Medium | 1 | ~100 | 4,075 | 0.61 | 54.18 | 1,450 |
| **ProjectStream** | Tiny | 1 | ~100 | 4,266 | 0.66 | 62.96 | 1,460 |
| **Project** | Tiny | 1 | ~100 | 4,072 | 0.64 | 54.16 | 1,450 |
| **ProjectStream** | Small | 1 | ~100 | 4,256 | 0.63 | 62.96 | 1,460 |
| **Project** | Small | 1 | ~100 | 3,973 | 0.64 | 54.18 | 1,450 |
| **Project** | Medium | 100 | ~100 | 154 | 6.49 | 5,414.13 | 144,895 |
| **Project** | Tiny | 100 | ~100 | 154 | 6.50 | 5,414.02 | 144,897 |
| **ProjectStream** | Medium | 100 | ~100 | 159 | 6.29 | 6,293.20 | 145,882 |
| **ProjectStream** | Tiny | 100 | ~100 | 154 | 6.50 | 6,293.20 | 145,882 |
| **Project** | Small | 100 | ~100 | 152 | 6.58 | 5,414.13 | 144,897 |
| **ProjectStream** | Small | 100 | ~100 | 156 | 6.41 | 6,293.41 | 145,881 |

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
