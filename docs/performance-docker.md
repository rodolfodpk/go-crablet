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
- **Events**: Number of events appended per operation (1 = single event, 100 = batch of 100 events)
- **Throughput (ops/sec)**: Operations completed per second (higher is better)
- **Latency (ns/op)**: Time per operation in nanoseconds (lower is better)
- **Memory (B/op)**: Memory allocated per operation in bytes (lower is better)
- **Allocations**: Number of memory allocations per operation (lower is better)

| Dataset | Concurrency | Events | Throughput (ops/sec) | Latency (ns/op) | Memory (B/op) | Allocations |
|---------|-------------|--------|---------------------|-----------------|---------------|-------------|
| Medium | 1 | 1 | 984 | 1,016,378 | 1,877 | 55 |
| Small | 1 | 1 | 979 | 1,021,591 | 1,901 | 55 |
| Tiny | 1 | 1 | 940 | 1,063,937 | 1,875 | 55 |
| Small | 10 | 1 | 337 | 2,964,403 | 17,553 | 521 |
| Tiny | 10 | 1 | 328 | 3,046,159 | 17,525 | 521 |
| Medium | 10 | 1 | 299 | 3,339,308 | 17,522 | 521 |
| Medium | 1 | 100 | 261 | 3,837,844 | 209,569 | 2,053 |
| Small | 1 | 100 | 246 | 4,060,320 | 209,981 | 2,053 |
| Tiny | 1 | 100 | 244 | 4,097,649 | 210,149 | 2,054 |
| Medium | 100 | 1 | 73 | 13,711,205 | 182,523 | 5,257 |
| Tiny | 100 | 1 | 72 | 13,835,732 | 182,497 | 5,258 |
| Small | 100 | 1 | 62 | 16,006,970 | 182,861 | 5,258 |
| Tiny | 10 | 100 | 40 | 24,919,232 | 2,096,906 | 20,505 |
| Small | 10 | 100 | 39 | 25,352,330 | 2,095,482 | 20,496 |
| Medium | 10 | 100 | 34 | 29,217,043 | 2,094,413 | 20,490 |
| Medium | 100 | 100 | 4 | 234,358,689 | 2,095,436 | 205,059 |
| Small | 100 | 100 | 4 | 265,204,075 | 2,095,318 | 205,066 |
| Tiny | 100 | 100 | 4 | 253,122,474 | 2,096,825 | 205,114 |

## AppendIf Performance (No Conflict)

**AppendIf No Conflict Details**:
- **Attempted Events**: Number of events AppendIf operation tries to append (1 or 100 events per operation)
- **Actual Events**: Number of events successfully appended (1 or 100 events)
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
| Tiny | 1 | 1 | 658 | 1,518,450 | 4,465 | 95 |
| Small | 1 | 1 | 651 | 1,535,504 | 4,459 | 95 |
| Tiny | 10 | 1 | 169 | 5,920,910 | 43,420 | 921 |
| Small | 10 | 1 | 161 | 6,202,469 | 43,309 | 918 |
| Medium | 10 | 1 | 62 | 16,137,540 | 43,184 | 914 |
| Medium | 1 | 100 | 99 | 10,136,572 | 213,737 | 2,092 |
| Small | 1 | 100 | 93 | 10,748,651 | 213,802 | 2,092 |
| Tiny | 1 | 100 | 92 | 10,877,061 | 214,163 | 2,094 |
| Small | 10 | 100 | 28 | 35,541,658 | 2,133,903 | 20,905 |
| Tiny | 10 | 100 | 28 | 35,448,964 | 2,136,494 | 20,922 |
| Medium | 10 | 100 | 27 | 37,589,242 | 2,132,165 | 20,893 |
| Small | 100 | 1 | 17 | 59,991,582 | 442,743 | 9,274 |
| Tiny | 100 | 1 | 16 | 63,685,427 | 441,085 | 9,270 |
| Medium | 100 | 1 | 8 | 132,163,729 | 440,483 | 9,269 |
| Medium | 100 | 100 | 3 | 350,805,663 | 2,133,741 | 209,041 |
| Small | 100 | 100 | 3 | 311,342,023 | 2,134,947 | 209,075 |
| Tiny | 100 | 100 | 3 | 297,525,060 | 2,135,738 | 209,122 |

## AppendIf Performance (With Conflict)

**AppendIf With Conflict Details**:
- **Attempted Events**: Number of events AppendIf operation tries to append (1 or 100 events per operation)
- **Actual Events**: Number of events successfully appended (0 - all operations fail due to conflicts)
- **Past Events**: Number of existing events in database before benchmark (100 events for all scenarios)
- **Conflict Events**: Number of conflicting events created before AppendIf (1, 10, or 100 events, matching concurrency level)

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
| Medium | 1 | 1 | 1 | 57 | 17,503,340 | 5,865 | 144 |
| Medium | 1 | 100 | 1 | 53 | 18,956,997 | 214,815 | 2,140 |
| Medium | 10 | 1 | 10 | 34 | 29,250,650 | 57,151 | 1,405 |
| Medium | 10 | 100 | 10 | 27 | 36,491,811 | 2,144,395 | 21,372 |
| Medium | 100 | 1 | 100 | 6 | 166,019,014 | 581,526 | 14,178 |
| Medium | 100 | 100 | 100 | 5 | 216,381,090 | 2,146,916 | 213,806 |
| Small | 1 | 1 | 1 | 47 | 21,263,682 | 5,907 | 144 |
| Tiny | 1 | 1 | 1 | 47 | 21,266,647 | 5,890 | 144 |
| Small | 1 | 100 | 1 | 43 | 23,517,665 | 214,845 | 2,141 |
| Tiny | 1 | 100 | 1 | 44 | 22,743,746 | 215,146 | 2,143 |
| Small | 10 | 1 | 10 | 25 | 40,611,760 | 57,216 | 1,406 |
| Tiny | 10 | 1 | 10 | 26 | 38,563,592 | 57,385 | 1,406 |
| Tiny | 10 | 100 | 10 | 24 | 42,527,353 | 2,146,439 | 21,389 |
| Small | 10 | 100 | 10 | 23 | 43,642,002 | 2,143,954 | 21,379 |
| Small | 100 | 1 | 100 | 4 | 230,691,143 | 579,527 | 14,176 |
| Tiny | 100 | 1 | 100 | 4 | 231,909,009 | 585,446 | 14,214 |
| Small | 100 | 100 | 100 | 4 | 256,274,453 | 2,146,931 | 213,822 |
| Tiny | 100 | 100 | 100 | 3 | 306,809,588 | 2,147,628 | 213,946 |


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
- **Latency (ns/op)**: Time per operation in nanoseconds (lower is better)
- **Memory (B/op)**: Memory allocated per operation in bytes (lower is better)
- **Allocations**: Number of memory allocations per operation (lower is better)

| Operation | Dataset | Concurrency | Events | Throughput (ops/sec) | Latency (ns/op) | Memory (B/op) | Allocations |
|-----------|---------|-------------|--------|---------------------|-----------------|---------------|-------------|
| **Project** | Medium | 1 | 50,000 | 2,555 | 391,495 | 2,036 | 37 |
| **Project** | Small | 1 | 25,000 | 2,550 | 391,810 | 2,036 | 37 |
| **Project** | Tiny | 1 | 20 | 2,300 | 434,857 | 2,036 | 37 |
| **ProjectStream** | Small | 1 | 25,000 | 2,400 | 416,667 | 2,038 | 38 |
| **ProjectStream** | Tiny | 1 | 20 | 2,200 | 454,545 | 2,040 | 38 |
| **ProjectStream** | Medium | 1 | 50,000 | 2,130 | 469,442 | 11,079 | 48 |
| **Project** | Medium | 10 | 50,000 | 1,305 | 765,517 | 553,405 | 14,474 |
| **Project** | Small | 10 | 25,000 | 1,390 | 719,424 | 553,473 | 14,474 |
| **Project** | Tiny | 10 | 20 | 1,347 | 742,391 | 553,404 | 14,474 |
| **ProjectStream** | Medium | 10 | 50,000 | 1,414 | 707,345 | 643,522 | 14,574 |
| **ProjectStream** | Small | 10 | 25,000 | 1,395 | 716,797 | 643,585 | 14,574 |
| **ProjectStream** | Tiny | 10 | 20 | 1,375 | 727,273 | 643,539 | 14,574 |
| **Project** | Medium | 25 | 50,000 | 620 | 1,612,903 | 1,383,303 | 36,180 |
| **Project** | Small | 25 | 25,000 | 645 | 1,550,388 | 1,383,368 | 36,181 |
| **Project** | Tiny | 25 | 20 | 633 | 1,579,778 | 1,383,279 | 36,181 |
| **ProjectStream** | Medium | 25 | 50,000 | 573 | 1,745,636 | 1,608,548 | 36,431 |
| **ProjectStream** | Small | 25 | 25,000 | 664 | 1,506,024 | 1,608,713 | 36,432 |
| **ProjectStream** | Tiny | 25 | 20 | 649 | 1,540,831 | 1,608,580 | 36,431 |

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

| Operation | Dataset | Concurrency | Events | Throughput (ops/sec) | Latency (ns/op) | Memory (B/op) | Allocations |
|-----------|---------|-------------|--------|---------------------|-----------------|---------------|-------------|
| **Concurrent_1User** | Small | 1 | - | 1,210 | 225,217 | 2,537 | 51 |
| **Concurrent_10Users** | Small | 10 | - | 1,208 | 807,331 | 26,033 | 530 |
| **Concurrent_100Users** | Medium | 100 | - | 146 | 6,854,788 | 269,465 | 5,543 |
