# Local PostgreSQL Performance

## Append Performance

**Append Operations Details**:
- **Operation**: Simple event append operations
- **Scenario**: Basic event writing without conditions or business logic
- **Events**: Single event (1) or batch (100 events)
- **Model**: Generic test events with simple JSON data

| Dataset | Concurrency | Events | Throughput (ops/sec) | Latency (ns/op) | Memory (B/op) | Allocations |
|---------|-------------|--------|---------------------|-----------------|---------------|-------------|
| Medium | 1 | 1 | 4,773 | 209,508 | 1,875 | 55 |
| Tiny | 1 | 1 | 4,602 | 217,203 | 1,877 | 56 |
| Small | 1 | 1 | 4,302 | 232,429 | 1,883 | 56 |
| Medium | 10 | 1 | 1,127 | 887,256 | 17,532 | 523 |
| Tiny | 10 | 1 | 1,109 | 901,649 | 17,542 | 523 |
| Small | 10 | 1 | 1,081 | 924,593 | 17,547 | 523 |
| Tiny | 1 | 100 | 652 | 1,533,690 | 210,790 | 2,053 |
| Medium | 1 | 100 | 612 | 1,634,961 | 211,381 | 2,053 |
| Small | 1 | 100 | 571 | 1,751,036 | 210,784 | 2,053 |
| Small | 100 | 1 | 141 | 7,105,952 | 182,904 | 5,282 |
| Medium | 100 | 1 | 129 | 7,746,200 | 182,588 | 5,275 |
| Tiny | 100 | 1 | 120 | 8,342,164 | 182,815 | 5,283 |
| Tiny | 10 | 100 | 98 | 10,171,860 | 2,096,951 | 20,506 |
| Medium | 10 | 100 | 98 | 10,210,932 | 2,094,443 | 20,489 |
| Small | 10 | 100 | 97 | 10,284,180 | 2,095,247 | 20,494 |
| Small | 100 | 100 | 10 | 97,230,189 | 20,961,682 | 205,133 |
| Medium | 100 | 100 | 9 | 115,899,177 | 20,955,109 | 205,076 |
| Tiny | 100 | 100 | 9 | 113,270,533 | 20,965,455 | 205,151 |

## AppendIf Performance (No Conflict)

**AppendIf No Conflict Details**:
- **Attempted Events**: Number of events AppendIf operation tries to append (1 or 100 events per operation)
- **Actual Events**: Number of events successfully appended (1 or 100 events)
- **Past Events**: Number of existing events in database before benchmark (100 events for all scenarios)
- **Conflict Events**: 0 (no conflicts exist)

| Dataset | Concurrency | Attempted Events | Throughput (ops/sec) | Latency (ns/op) | Memory (B/op) | Allocations |
|---------|-------------|------------------|---------------------|-----------------|---------------|-------------|
| Small | 1 | 1 | 1,528 | 654,820 | 4,465 | 96 |
| Tiny | 1 | 1 | 732 | 1,366,943 | 4,461 | 95 |
| Medium | 1 | 1 | 661 | 1,514,049 | 4,461 | 95 |
| Small | 1 | 100 | 302 | 3,306,069 | 213,952 | 2,093 |
| Tiny | 1 | 100 | 270 | 3,707,723 | 214,381 | 2,095 |
| Medium | 1 | 100 | 225 | 4,452,797 | 213,795 | 2,092 |
| Small | 10 | 1 | 332 | 3,010,493 | 43,401 | 922 |
| Tiny | 10 | 1 | 286 | 3,492,125 | 43,401 | 921 |
| Medium | 10 | 1 | 275 | 3,641,032 | 43,363 | 921 |
| Small | 10 | 100 | 83 | 12,040,251 | 2,136,629 | 20,903 |
| Medium | 10 | 100 | 63 | 15,832,250 | 2,135,136 | 20,892 |
| Tiny | 10 | 100 | 56 | 17,880,128 | 2,139,685 | 20,929 |
| Small | 100 | 1 | 29 | 34,537,345 | 441,049 | 9,265 |
| Medium | 100 | 1 | 28 | 35,975,661 | 440,653 | 9,262 |
| Tiny | 100 | 1 | 27 | 36,487,455 | 441,985 | 9,272 |
| Small | 100 | 100 | 8 | 129,548,595 | 21,355,971 | 209,084 |
| Medium | 100 | 100 | 8 | 125,817,985 | 21,365,328 | 209,045 |
| Tiny | 100 | 100 | 7 | 151,996,252 | 21,379,785 | 209,199 |

## AppendIf Performance (With Conflict)

**AppendIf With Conflict Details**:
- **Attempted Events**: Number of events AppendIf operation tries to append (1 or 100 events per operation)
- **Actual Events**: Number of events successfully appended (0 - all operations fail due to conflicts)
- **Past Events**: Number of existing events in database before benchmark (100 events for all scenarios)
- **Conflict Events**: Number of conflicting events created before AppendIf (1, 10, or 100 events, matching concurrency level)

| Dataset | Concurrency | Attempted Events | Conflict Events | Throughput (ops/sec) | Latency (ns/op) | Memory (B/op) | Allocations |
|---------|-------------|------------------|-----------------|---------------------|-----------------|---------------|-------------|
| Small | 1 | 1 | 1 | 70 | 14,371,495 | 5,859 | 144 |
| Medium | 1 | 1 | 1 | 46 | 21,828,132 | 5,897 | 144 |
| Tiny | 1 | 1 | 1 | 31 | 32,313,921 | 5,910 | 144 |
| Small | 1 | 100 | 1 | 47 | 21,387,674 | 214,731 | 2,140 |
| Tiny | 1 | 100 | 1 | 43 | 23,255,195 | 215,322 | 2,144 |
| Medium | 1 | 100 | 1 | 40 | 25,062,065 | 214,684 | 2,140 |
| Small | 10 | 1 | 10 | 48 | 20,857,225 | 57,246 | 1,405 |
| Medium | 10 | 1 | 10 | 40 | 25,004,187 | 57,238 | 1,405 |
| Tiny | 10 | 1 | 10 | 25 | 40,366,956 | 57,282 | 1,406 |
| Tiny | 10 | 100 | 10 | 35 | 28,186,945 | 2,149,590 | 21,401 |
| Small | 10 | 100 | 10 | 34 | 29,396,364 | 2,146,454 | 21,380 |
| Medium | 10 | 100 | 10 | 33 | 30,285,182 | 2,145,157 | 21,372 |
| Tiny | 100 | 1 | 100 | 11 | 89,878,760 | 584,294 | 14,188 |
| Medium | 100 | 1 | 100 | 10 | 98,250,736 | 581,578 | 14,169 |
| Small | 100 | 1 | 100 | 9 | 112,494,108 | 581,648 | 14,172 |
| Tiny | 100 | 100 | 100 | 6 | 179,032,479 | 21,493,694 | 213,955 |
| Medium | 100 | 100 | 100 | 6 | 180,638,478 | 21,478,281 | 213,861 |
| Small | 100 | 100 | 100 | 5 | 205,235,701 | 21,475,130 | 213,853 |

## Read and Projection Performance

| Operation | Dataset | Concurrency | Events | Throughput (ops/sec) | Latency (ns/op) | Memory (B/op) | Allocations |
|-----------|---------|-------------|--------|---------------------|-----------------|---------------|-------------|
| **Read_Single** | Tiny | 1 | - | 40 | 25,170,720 | 2,546,894 | 288,487 |
| **Read_Single** | Small | 1 | - | 131 | 7,649,236 | 1,024,685 | 131,363 |
| **Read_Single** | Medium | 1 | - | 113 | 8,850,949 | 1,024,343 | 131,363 |
| **Read_Batch** | Tiny | 1 | - | 6,547 | 152,791 | 989 | 21 |
| **Read_Batch** | Small | 1 | - | 2,075 | 481,945 | 988 | 21 |
| **Read_Batch** | Medium | 1 | - | 558 | 1,791,792 | 989 | 21 |
| **Projection** | Tiny | 1 | - | 7,033 | 142,197 | 2,037 | 37 |
| **Projection** | Small | 1 | - | 1,968 | 507,989 | 2,036 | 37 |
| **Projection** | Medium | 1 | - | 3 | 369,278,396 | 2,212 | 37 |

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
