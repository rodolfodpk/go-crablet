# Local PostgreSQL Performance

## Append Performance

**Append Operations Details**:
- **Operation**: Simple event append operations
- **Scenario**: Basic event writing without conditions or business logic
- **Events**: Single event (1) or batch (100 events)
- **Model**: Generic test events with simple JSON data

| Dataset | Concurrency | Events | Throughput (ops/sec) | Latency (ns/op) | Memory (B/op) | Allocations |
|---------|-------------|--------|---------------------|-----------------|---------------|-------------|
| Tiny | 1 | 1 | 4,636 | 215,796 | 1,879 | 55 |
| Small | 1 | 1 | 4,442 | 225,202 | 1,884 | 56 |
| Medium | 1 | 1 | 4,329 | 230,996 | 1,876 | 55 |
| Small | 10 | 1 | 1,177 | 849,272 | 17,558 | 523 |
| Tiny | 10 | 1 | 1,129 | 886,032 | 17,533 | 523 |
| Medium | 10 | 1 | 1,116 | 895,710 | 17,527 | 523 |
| Medium | 1 | 100 | 819 | 1,221,282 | 211,311 | 2,053 |
| Tiny | 1 | 100 | 695 | 1,438,508 | 210,708 | 2,054 |
| Small | 1 | 100 | 617 | 1,621,076 | 211,385 | 2,053 |
| Small | 100 | 1 | 122 | 8,177,390 | 182,716 | 5,263 |
| Medium | 100 | 1 | 114 | 8,749,296 | 182,726 | 5,279 |
| Tiny | 100 | 1 | 131 | 7,632,288 | 182,709 | 5,281 |
| Small | 10 | 100 | 96 | 10,366,952 | 2,095,370 | 20,495 |
| Tiny | 10 | 100 | 94 | 10,653,667 | 2,096,977 | 20,506 |
| Medium | 10 | 100 | 77 | 13,015,612 | 2,094,547 | 20,494 |
| Small | 100 | 100 | 10 | 104,328,282 | 20,962,898 | 205,128 |
| Medium | 100 | 100 | 10 | 96,468,008 | 2,095,427 | 205,062 |
| Tiny | 100 | 100 | 9 | 108,822,150 | 2,096,423 | 205,143 |

## AppendIf Performance (No Conflict)

**AppendIf No Conflict Details**:
- **Attempted Events**: Number of events AppendIf operation tries to append (1 or 100 events per operation)
- **Actual Events**: Number of events successfully appended (1 or 100 events)
- **Past Events**: Number of existing events in database before benchmark (100 events for all scenarios)
- **Conflict Events**: 0 (no conflicts exist)

| Dataset | Concurrency | Attempted Events | Throughput (ops/sec) | Latency (ns/op) | Memory (B/op) | Allocations |
|---------|-------------|------------------|---------------------|-----------------|---------------|-------------|
| Medium | 1 | 1 | 1,184 | 844,971 | 4,462 | 96 |
| Small | 1 | 1 | 470 | 2,129,438 | 4,465 | 95 |
| Tiny | 1 | 1 | 390 | 2,566,532 | 4,462 | 95 |
| Small | 1 | 100 | 294 | 3,397,311 | 213,931 | 2,093 |
| Medium | 1 | 100 | 253 | 3,946,188 | 213,787 | 2,092 |
| Tiny | 1 | 100 | 235 | 4,256,762 | 214,968 | 2,096 |
| Medium | 10 | 1 | 318 | 3,146,323 | 43,371 | 922 |
| Small | 10 | 1 | 203 | 4,936,844 | 43,366 | 920 |
| Tiny | 10 | 1 | 197 | 5,079,846 | 43,369 | 920 |
| Medium | 10 | 100 | 80 | 12,437,512 | 2,134,779 | 20,892 |
| Small | 10 | 100 | 77 | 13,053,566 | 2,135,695 | 20,901 |
| Tiny | 10 | 100 | 75 | 13,410,766 | 2,139,642 | 20,929 |
| Medium | 100 | 1 | 29 | 34,331,318 | 440,658 | 9,262 |
| Small | 100 | 1 | 28 | 35,501,782 | 440,957 | 9,265 |
| Tiny | 100 | 1 | 24 | 41,524,747 | 441,696 | 9,269 |
| Medium | 100 | 100 | 9 | 116,315,980 | 2,135,264 | 209,052 |
| Small | 100 | 100 | 8 | 131,816,613 | 2,136,679 | 209,096 |
| Tiny | 100 | 100 | 8 | 121,959,336 | 2,138,098 | 209,192 |

## AppendIf Performance (With Conflict)

**AppendIf With Conflict Details**:
- **Attempted Events**: Number of events AppendIf operation tries to append (1 or 100 events per operation)
- **Actual Events**: Number of events successfully appended (0 - all operations fail due to conflicts)
- **Past Events**: Number of existing events in database before benchmark (100 events for all scenarios)
- **Conflict Events**: Number of conflicting events created before AppendIf (1, 10, or 100 events, matching concurrency level)

| Dataset | Concurrency | Attempted Events | Conflict Events | Throughput (ops/sec) | Latency (ns/op) | Memory (B/op) | Allocations |
|---------|-------------|------------------|-----------------|---------------------|-----------------|---------------|-------------|
| Small | 1 | 1 | 1 | 57 | 17,401,571 | 5,902 | 144 |
| Tiny | 1 | 1 | 1 | 57 | 17,608,228 | 5,873 | 144 |
| Medium | 1 | 1 | 1 | 49 | 20,311,454 | 5,912 | 144 |
| Small | 1 | 100 | 1 | 45 | 22,462,174 | 214,814 | 2,141 |
| Tiny | 1 | 100 | 1 | 42 | 23,868,552 | 215,306 | 2,143 |
| Medium | 1 | 100 | 1 | 40 | 25,215,397 | 214,690 | 2,140 |
| Small | 10 | 1 | 10 | 47 | 21,276,223 | 57,221 | 1,405 |
| Medium | 10 | 1 | 10 | 39 | 25,421,037 | 57,136 | 1,404 |
| Tiny | 10 | 1 | 10 | 40 | 24,828,917 | 57,241 | 1,405 |
| Medium | 10 | 100 | 10 | 35 | 28,819,276 | 2,145,438 | 21,374 |
| Small | 10 | 100 | 10 | 33 | 29,885,572 | 2,146,577 | 21,382 |
| Tiny | 10 | 100 | 10 | 33 | 30,092,145 | 2,148,972 | 21,398 |
| Medium | 100 | 1 | 100 | 10 | 96,621,963 | 582,073 | 14,170 |
| Small | 100 | 1 | 100 | 9 | 106,681,188 | 581,433 | 14,167 |
| Tiny | 100 | 1 | 100 | 9 | 115,906,170 | 584,484 | 14,192 |
| Medium | 100 | 100 | 100 | 6 | 174,658,241 | 2,148,060 | 213,851 |
| Tiny | 100 | 100 | 100 | 5 | 210,099,087 | 2,149,004 | 214,000 |
| Small | 100 | 100 | 100 | 5 | 194,951,683 | 2,148,114 | 213,877 |

## Read and Projection Performance

| Operation | Dataset | Concurrency | Events | Throughput (ops/sec) | Latency (ns/op) | Memory (B/op) | Allocations |
|-----------|---------|-------------|--------|---------------------|-----------------|---------------|-------------|
| **Read_Single** | Tiny | 1 | - | 1,699 | 588,548 | 988 | 21 |
| **Read_Single** | Small | 1 | - | 901 | 1,109,483 | 988 | 21 |
| **Read_Single** | Medium | 1 | - | 557 | 1,794,783 | 989 | 21 |
| **Read_Batch** | Tiny | 1 | - | 1,751 | 571,190 | 988 | 21 |
| **Read_Batch** | Small | 1 | - | 901 | 1,109,220 | 990 | 21 |
| **Read_Batch** | Medium | 1 | - | 549 | 1,819,667 | 990 | 21 |
| **Projection** | Tiny | 1 | - | 15,534 | 64,393 | 2,036 | 37 |
| **Projection** | Small | 1 | - | 896 | 1,117,115 | 2,036 | 37 |
| **Projection** | Medium | 1 | - | 542 | 1,844,966 | 2,038 | 37 |

## Concurrent Operations Performance

**Concurrent Operations Details**:
- **Operation**: Course registration events (StudentCourseRegistration)
- **Scenario**: Multiple students simultaneously registering for courses
- **Events**: 1 event per user (course registration)
- **Model**: Domain-specific business scenario with realistic data

| Operation | Dataset | Concurrency | Events | Throughput (ops/sec) | Latency (ns/op) | Memory (B/op) | Allocations |
|-----------|---------|-------------|--------|---------------------|-----------------|---------------|-------------|
| **Concurrent_1User** | Small | 1 | - | 1,210 | 225,217 | 2,537 | 51 |
| **Concurrent_10Users** | Small | 10 | - | 1,208 | 807,331 | 26,033 | 530 |
| **Concurrent_100Users** | Medium | 100 | - | 146 | 6,854,788 | 269,465 | 5,543 |
