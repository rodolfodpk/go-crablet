# Local PostgreSQL Performance

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
- **Latency (ns/op)**: Time per operation in nanoseconds (lower is better)
- **Memory (B/op)**: Memory allocated per operation in bytes (lower is better)
- **Allocations**: Number of memory allocations per operation (lower is better)

| Dataset | Concurrency | Events | Throughput (ops/sec) | Latency (ns/op) | Memory (B/op) | Allocations |
|---------|-------------|--------|---------------------|-----------------|---------------|-------------|
| Tiny | 1 | 1 | 2,264 | 441,358 | 1,880 | 55 |
| Small | 1 | 1 | 2,259 | 442,324 | 1,880 | 55 |
| Medium | 1 | 1 | 2,067 | 483,109 | 1,874 | 55 |
| Tiny | 1 | 10 | 2,010 | 497,142 | 20,013 | 243 |
| Small | 1 | 10 | 2,013 | 496,902 | 20,004 | 243 |
| Medium | 1 | 10 | 2,048 | 488,359 | 20,005 | 243 |
| Tiny | 100 | 1 | 182 | 5,494,620 | 182,623 | 5,258 |
| Small | 100 | 1 | 178 | 5,617,259 | 182,996 | 5,260 |
| Medium | 100 | 1 | 180 | 5,551,665 | 182,508 | 5,258 |
| Tiny | 100 | 10 | 100 | 10,000,000 | 1,998,007 | 24,078 |
| Small | 100 | 10 | 100 | 10,000,000 | 1,996,216 | 24,070 |
| Medium | 100 | 10 | 100 | 10,000,000 | 1,995,823 | 24,064 |

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
- **Latency (ns/op)**: Time per operation in nanoseconds (lower is better)
- **Memory (B/op)**: Memory allocated per operation in bytes (lower is better)
- **Allocations**: Number of memory allocations per operation (lower is better)

| Dataset | Concurrency | Attempted Events | Throughput (ops/sec) | Latency (ns/op) | Memory (B/op) | Allocations |
|---------|-------------|------------------|---------------------|-----------------|---------------|-------------|
| Tiny | 1 | 1 | 1,942 | 514,631 | 4,468 | 95 |
| Small | 1 | 1 | 2,068 | 483,231 | 4,464 | 95 |
| Medium | 1 | 1 | 2,160 | 462,858 | 4,460 | 95 |
| Tiny | 1 | 10 | 1,890 | 528,950 | 22,052 | 283 |
| Small | 1 | 10 | 1,714 | 583,395 | 22,035 | 282 |
| Medium | 1 | 10 | 1,879 | 531,371 | 22,031 | 282 |
| Tiny | 100 | 1 | 100 | 10,000,000 | 440,997 | 9,264 |
| Small | 100 | 1 | 100 | 10,000,000 | 441,369 | 9,266 |
| Medium | 100 | 1 | 100 | 10,000,000 | 440,406 | 9,260 |
| Tiny | 100 | 10 | 100 | 10,000,000 | 2,199,546 | 28,008 |
| Small | 100 | 10 | 100 | 10,000,000 | 2,195,417 | 27,987 |
| Medium | 100 | 10 | 100 | 10,000,000 | 2,195,191 | 27,980 |

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
- **Latency (ns/op)**: Time per operation in nanoseconds (lower is better)
- **Memory (B/op)**: Memory allocated per operation in bytes (lower is better)
- **Allocations**: Number of memory allocations per operation (lower is better)

| Dataset | Concurrency | Attempted Events | Conflict Events | Throughput (ops/sec) | Latency (ns/op) | Memory (B/op) | Allocations |
|---------|-------------|------------------|-----------------|---------------------|-----------------|---------------|-------------|
| Tiny | 1 | 1 | 1 | 1,185 | 843,519 | 5,856 | 145 |
| Small | 1 | 1 | 1 | 1,276 | 783,212 | 5,848 | 145 |
| Medium | 1 | 1 | 1 | 1,160 | 862,069 | 5,848 | 145 |
| Tiny | 1 | 10 | 1 | 1,124 | 889,248 | 23,438 | 331 |
| Small | 1 | 10 | 1 | 1,159 | 862,069 | 23,426 | 331 |
| Medium | 1 | 10 | 1 | 1,159 | 862,069 | 23,426 | 331 |
| Tiny | 100 | 1 | 1 | 166 | 6,024,096 | 433,127 | 9,505 |
| Small | 100 | 1 | 1 | 182 | 5,493,406 | 433,009 | 9,503 |
| Medium | 100 | 1 | 1 | 182 | 5,493,406 | 433,009 | 9,503 |
| Tiny | 100 | 10 | 1 | 152 | 6,578,947 | 2,191,417 | 28,142 |
| Small | 100 | 10 | 1 | 154 | 6,493,506 | 2,189,519 | 28,125 |
| Medium | 100 | 10 | 1 | 154 | 6,493,506 | 2,189,519 | 28,125 |

## Projection Performance

**Projection Operations Details**:
- **Operation**: State reconstruction from event streams using core API's built-in concurrency controls
- **Scenario**: Building aggregate state from historical events with proper goroutine limits
- **Events**: Number of events processed during projection (~100 events from Append benchmarks)
- **Model**: Domain-specific state reconstruction with business logic
- **Architecture**: Uses Go 1.25 concurrency features and core API's built-in goroutine limits
- **Performance**: Realistic throughput with proper resource management

**Column Explanations**:
- **Operation**: Type of projection operation (Project = single-threaded, ProjectStream = streaming with channels)
- **Dataset**: Test data size (Tiny: 5 courses/10 students, Small: 1K courses/10K students, Medium: 1K courses/10K students)
- **Concurrency**: Number of concurrent users/goroutines running operations simultaneously
- **Events**: Approximate number of events processed during projection (~100 events from previous Append benchmarks)
- **Throughput (ops/sec)**: Operations completed per second (higher is better)
- **Latency (ns/op)**: Time per operation in nanoseconds (lower is better)
- **Memory (B/op)**: Memory allocated per operation in bytes (lower is better)
- **Allocations**: Number of memory allocations per operation (lower is better)

| Operation | Dataset | Concurrency | Events | Throughput (ops/sec) | Latency (ns/op) | Memory (B/op) | Allocations |
|-----------|---------|-------------|--------|---------------------|-----------------|---------------|-------------|
| **ProjectStream** | Medium | 1 | ~100 | 4,134 | 241,755 | 64,476 | 1,460 |
| **Project** | Medium | 1 | ~100 | 3,684 | 271,389 | 55,478 | 1,450 |
| **ProjectStream** | Tiny | 1 | ~100 | 4,266 | 234,324 | 64,475 | 1,460 |
| **Project** | Tiny | 1 | ~100 | 3,529 | 283,245 | 55,464 | 1,450 |
| **ProjectStream** | Small | 1 | ~100 | 4,134 | 241,755 | 64,476 | 1,460 |
| **Project** | Small | 1 | ~100 | 3,684 | 271,389 | 55,478 | 1,450 |
| **Project** | Medium | 100 | ~100 | 152 | 6,578,947 | 5,544,465 | 144,897 |
| **Project** | Tiny | 100 | ~100 | 152 | 6,578,947 | 5,544,356 | 144,894 |
| **ProjectStream** | Medium | 100 | ~100 | 153 | 6,535,948 | 6,444,449 | 145,883 |
| **ProjectStream** | Tiny | 100 | ~100 | 158 | 6,328,125 | 6,444,235 | 145,883 |
| **Project** | Small | 100 | ~100 | 152 | 6,578,947 | 5,544,465 | 144,897 |
| **ProjectStream** | Small | 100 | ~100 | 153 | 6,535,948 | 6,444,449 | 145,883 |
