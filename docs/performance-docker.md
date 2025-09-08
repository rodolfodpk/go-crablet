# Docker PostgreSQL Performance

## Performance Results

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
| Medium | 1 | 1 | 2,218 | 0.45 | 1.83 | 55 |
| Tiny | 1 | 1 | 2,106 | 0.46 | 1.84 | 55 |
| Small | 1 | 1 | 2,066 | 0.48 | 1.87 | 56 |
| Tiny | 1 | 10 | 1,977 | 0.51 | 19.55 | 244 |
| Medium | 1 | 10 | 1,972 | 0.51 | 19.54 | 243 |
| Small | 1 | 10 | 1,773 | 0.56 | 19.55 | 243 |
| Medium | 100 | 1 | 182 | 5.50 | 178.15 | 5,256 |
| Tiny | 100 | 1 | 172 | 5.82 | 178.13 | 5,258 |
| Small | 100 | 1 | 128 | 7.81 | 178.37 | 5,258 |
| Medium | 100 | 10 | 100 | 10.00 | 1,949.29 | 24,067 |
| Tiny | 100 | 10 | 100 | 10.00 | 1,950.08 | 24,074 |
| Small | 100 | 10 | 80 | 12.50 | 1,949.56 | 24,075 |

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
| Medium | 1 | 1 | 2,017 | 0.50 | 4.36 | 95 |
| Tiny | 1 | 1 | 1,958 | 0.51 | 4.36 | 95 |
| Small | 1 | 1 | 1,742 | 0.57 | 4.35 | 95 |
| Tiny | 1 | 10 | 1,830 | 0.55 | 21.52 | 283 |
| Medium | 1 | 10 | 1,623 | 0.62 | 21.51 | 282 |
| Small | 1 | 10 | 1,723 | 0.58 | 21.52 | 282 |
| Medium | 10 | 1 | 100 | 10.00 | 43.18 | 914 |
| Tiny | 10 | 1 | 100 | 10.00 | 43.15 | 921 |
| Small | 10 | 1 | 90 | 11.11 | 43.31 | 918 |
| Medium | 10 | 10 | 100 | 10.00 | 214.16 | 2,092 |
| Tiny | 10 | 10 | 100 | 10.00 | 214.16 | 2,094 |
| Small | 10 | 10 | 90 | 11.11 | 214.16 | 2,092 |
| Medium | 100 | 1 | 100 | 10.00 | 429.90 | 9,259 |
| Tiny | 100 | 1 | 100 | 10.00 | 431.56 | 9,268 |
| Small | 100 | 1 | 90 | 11.11 | 431.37 | 9,263 |
| Medium | 100 | 10 | 100 | 10.00 | 2,144.25 | 27,980 |
| Tiny | 100 | 10 | 96 | 10.42 | 2,147.07 | 28,008 |
| Small | 100 | 10 | 84 | 11.90 | 2,144.35 | 27,981 |
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
| Medium | 1 | 1 | 1 | 1,125 | 0.89 | 5.70 | 145 |
| Tiny | 1 | 1 | 1 | 1,064 | 0.94 | 5.71 | 145 |
| Small | 1 | 1 | 1 | 1,100 | 0.91 | 5.71 | 145 |
| Medium | 1 | 10 | 1 | 1,066 | 0.94 | 22.85 | 331 |
| Tiny | 1 | 10 | 1 | 1,086 | 0.92 | 22.88 | 331 |
| Small | 1 | 10 | 1 | 1,009 | 0.99 | 22.86 | 331 |
| Medium | 10 | 1 | 1 | 100 | 10.00 | 57.15 | 1,405 |
| Tiny | 10 | 1 | 1 | 100 | 10.00 | 57.22 | 1,406 |
| Small | 10 | 1 | 1 | 90 | 11.11 | 57.24 | 1,406 |
| Medium | 10 | 10 | 1 | 100 | 10.00 | 214.16 | 2,140 |
| Tiny | 10 | 10 | 1 | 100 | 10.00 | 214.16 | 2,143 |
| Small | 10 | 10 | 1 | 90 | 11.11 | 214.16 | 2,141 |
| Medium | 100 | 1 | 1 | 159 | 6.29 | 422.59 | 9,501 |
| Tiny | 100 | 1 | 1 | 160 | 6.25 | 422.98 | 9,506 |
| Small | 100 | 1 | 1 | 163 | 6.13 | 422.55 | 9,504 |
| Medium | 100 | 10 | 1 | 141 | 7.09 | 2,136.25 | 28,114 |
| Tiny | 100 | 10 | 1 | 144 | 6.94 | 2,139.27 | 28,143 |
| Small | 100 | 10 | 1 | 148 | 6.76 | 2,137.41 | 28,122 |


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
