# Local PostgreSQL Performance

## Append Performance

**Append Operations Details**:
- **Operation**: Simple event append operations
- **Scenario**: Basic event writing without conditions or business logic
- **Events**: Single event (1) or batch (100 events)
- **Model**: Generic test events with simple JSON data

| Dataset | Concurrency | Events | Throughput (ops/sec) | Latency (ns/op) | Memory (B/op) | Allocations |
|---------|-------------|--------|---------------------|-----------------|---------------|-------------|
| Tiny | 1 | 1 | 4,577 | 218,504 | 1,879 | 56 |
| Small | 1 | 1 | 4,583 | 218,259 | 1,885 | 56 |
| Medium | 1 | 1 | 4,463 | 224,047 | 1,875 | 56 |
| Small | 10 | 1 | 1,161 | 861,083 | 17,543 | 523 |
| Tiny | 10 | 1 | 1,149 | 870,556 | 17,538 | 523 |
| Medium | 10 | 1 | 1,098 | 910,420 | 17,526 | 523 |
| Medium | 1 | 100 | 691 | 1,447,326 | 211,346 | 2,053 |
| Small | 1 | 100 | 659 | 1,518,544 | 211,411 | 2,053 |
| Tiny | 1 | 100 | 656 | 1,523,947 | 210,763 | 2,053 |
| Small | 100 | 1 | 128 | 7,805,995 | 182,766 | 5,277 |
| Tiny | 100 | 1 | 132 | 7,571,459 | 182,769 | 5,280 |
| Medium | 100 | 1 | 130 | 7,697,765 | 182,565 | 5,279 |
| Small | 10 | 100 | 107 | 9,316,738 | 2,095,309 | 20,495 |
| Tiny | 10 | 100 | 104 | 9,627,567 | 2,095,822 | 20,499 |
| Medium | 10 | 100 | 108 | 9,277,635 | 2,094,281 | 20,492 |
| Small | 100 | 100 | 9 | 110,610,827 | 20,962,085 | 205,130 |
| Medium | 100 | 100 | 10 | 104,878,457 | 20,951,635 | 205,032 |
| Tiny | 100 | 100 | 9 | 108,198,257 | 20,963,963 | 205,141 |

## AppendIf Performance (No Conflict)

**AppendIf No Conflict Details**:
- **Attempted Events**: Number of events AppendIf operation tries to append (1 or 100 events per operation)
- **Actual Events**: Number of events successfully appended (1 or 100 events)
- **Past Events**: Number of existing events in database before benchmark (100 events for all scenarios)
- **Conflict Events**: 0 (no conflicts exist)

| Dataset | Concurrency | Attempted Events | Throughput (ops/sec) | Latency (ns/op) | Memory (B/op) | Allocations |
|---------|-------------|------------------|---------------------|-----------------|---------------|-------------|
| Medium | 1 | 1 | 1,890 | 529,242 | 4,463 | 96 |
| Small | 1 | 1 | 1,837 | 544,085 | 4,463 | 96 |
| Tiny | 1 | 1 | 764 | 1,308,487 | 4,462 | 95 |
| Medium | 1 | 100 | 302 | 3,312,571 | 213,772 | 2,092 |
| Small | 1 | 100 | 284 | 3,527,126 | 213,906 | 2,093 |
| Tiny | 1 | 100 | 300 | 3,333,814 | 214,308 | 2,095 |
| Medium | 10 | 1 | 318 | 3,142,533 | 43,369 | 922 |
| Small | 10 | 1 | 350 | 2,859,020 | 43,400 | 922 |
| Tiny | 10 | 1 | 287 | 3,488,913 | 43,419 | 922 |
| Medium | 10 | 100 | 86 | 11,687,830 | 2,134,691 | 20,890 |
| Small | 10 | 100 | 41 | 24,622,282 | 2,135,735 | 20,903 |
| Tiny | 10 | 100 | 75 | 13,042,655 | 2,139,859 | 20,927 |
| Medium | 100 | 1 | 29 | 34,649,656 | 440,908 | 9,264 |
| Small | 100 | 1 | 29 | 34,773,333 | 440,693 | 9,262 |
| Tiny | 100 | 1 | 27 | 37,239,183 | 441,742 | 9,270 |
| Medium | 100 | 100 | 8 | 119,573,268 | 21,370,694 | 209,105 |
| Small | 100 | 100 | 7 | 134,946,622 | 21,369,626 | 209,103 |
| Tiny | 100 | 100 | 8 | 121,339,275 | 21,378,864 | 209,160 |

## AppendIf Performance (With Conflict)

**AppendIf With Conflict Details**:
- **Attempted Events**: Number of events AppendIf operation tries to append (1 or 100 events per operation)
- **Actual Events**: Number of events successfully appended (0 - all operations fail due to conflicts)
- **Past Events**: Number of existing events in database before benchmark (100 events for all scenarios)
- **Conflict Events**: Number of conflicting events created before AppendIf (1, 10, or 100 events, matching concurrency level)

| Dataset | Concurrency | Attempted Events | Conflict Events | Throughput (ops/sec) | Latency (ns/op) | Memory (B/op) | Allocations |
|---------|-------------|------------------|-----------------|---------------------|-----------------|---------------|-------------|
| Small | 1 | 1 | 1 | 69 | 14,499,571 | 5,856 | 144 |
| Tiny | 1 | 1 | 1 | 59 | 16,941,743 | 5,862 | 144 |
| Medium | 1 | 1 | 1 | 56 | 17,894,538 | 5,887 | 144 |
| Small | 1 | 100 | 1 | 51 | 19,483,219 | 214,950 | 2,141 |
| Tiny | 1 | 100 | 1 | 48 | 20,767,895 | 215,322 | 2,144 |
| Medium | 1 | 100 | 1 | 49 | 20,483,004 | 214,685 | 2,140 |
| Small | 10 | 1 | 10 | 47 | 21,124,148 | 57,221 | 1,405 |
| Medium | 10 | 1 | 10 | 46 | 21,553,922 | 57,229 | 1,404 |
| Tiny | 10 | 1 | 10 | 47 | 21,114,152 | 57,341 | 1,406 |
| Medium | 10 | 100 | 10 | 35 | 28,981,927 | 2,145,438 | 21,374 |
| Small | 10 | 100 | 10 | 35 | 28,900,995 | 2,146,577 | 21,382 |
| Tiny | 10 | 100 | 10 | 33 | 30,319,770 | 2,148,972 | 21,398 |
| Medium | 100 | 1 | 100 | 9 | 108,448,105 | 581,445 | 14,167 |
| Small | 100 | 1 | 100 | 9 | 105,167,811 | 582,803 | 14,177 |
| Tiny | 100 | 1 | 100 | 9 | 111,577,038 | 582,143 | 14,173 |
| Medium | 100 | 100 | 100 | 5 | 198,970,045 | 21,469,677 | 213,824 |
| Small | 100 | 100 | 100 | 5 | 200,315,246 | 21,477,819 | 213,889 |
| Tiny | 100 | 100 | 100 | 5 | 200,298,135 | 21,487,972 | 213,966 |



## Projection Performance

**Projection Operations Details**:
- **Operation**: State reconstruction from event streams
- **Scenario**: Building aggregate state from historical events
- **Events**: Number of events processed during projection (varies by dataset)
- **Model**: Domain-specific state reconstruction with business logic

| Operation | Dataset | Concurrency | Events | Throughput (ops/sec) | Latency (ns/op) | Memory (B/op) | Allocations |
|-----------|---------|-------------|--------|---------------------|-----------------|---------------|-------------|
| **Project** | Tiny | 1 | - | 890 | 1,121,436 | 2,316 | 42 |
| **Project** | Small | 1 | - | 1,864 | 536,253 | 2,316 | 42 |
| **Project** | Medium | 1 | - | 896 | 1,115,005 | 2,314 | 42 |
| **Project** | Tiny | 10 | - | 430 | 2,325,222 | 22,015 | 393 |
| **Project** | Small | 10 | - | 873 | 1,146,420 | 22,014 | 393 |
| **Project** | Medium | 10 | - | 406 | 2,414,127 | 22,010 | 393 |
| **Project** | Tiny | 25 | - | 206 | 4,866,014 | 54,825 | 978 |
| **Project** | Small | 25 | - | 461 | 2,169,102 | 54,809 | 978 |
| **Project** | Medium | 25 | - | 205 | 4,879,897 | 54,827 | 978 |
| **ProjectStream** | Tiny | 1 | - | 880 | 1,134,662 | 11,358 | 53 |
| **ProjectStream** | Small | 1 | - | 2,009 | 497,873 | 11,359 | 53 |
| **ProjectStream** | Medium | 1 | - | 883 | 1,132,402 | 11,357 | 53 |
| **ProjectStream** | Tiny | 10 | - | 420 | 2,399,035 | 112,449 | 503 |
| **ProjectStream** | Small | 10 | - | 929 | 1,077,069 | 112,428 | 503 |
| **ProjectStream** | Medium | 10 | - | 420 | 2,353,277 | 112,430 | 503 |
| **ProjectStream** | Tiny | 25 | - | 210 | 4,966,623 | 280,876 | 1,253 |
| **ProjectStream** | Small | 25 | - | 423 | 2,364,859 | 280,944 | 1,253 |
| **ProjectStream** | Medium | 25 | - | 202 | 4,934,739 | 280,935 | 1,253 |

## Course Registration Performance

**Course Registration Details**:
- **Operation**: Course registration events (StudentCourseRegistration)
- **Scenario**: Multiple students simultaneously registering for courses
- **Events**: 1 event per user (course registration)
- **Model**: Domain-specific business scenario with realistic data

| Operation | Dataset | Concurrency | Events | Throughput (ops/sec) | Latency (ns/op) | Memory (B/op) | Allocations |
|-----------|---------|-------------|--------|---------------------|-----------------|---------------|-------------|
| **Concurrent_1User** | Small | 1 | - | 1,210 | 225,217 | 2,537 | 51 |
| **Concurrent_10Users** | Small | 10 | - | 1,208 | 807,331 | 26,033 | 530 |
| **Concurrent_100Users** | Medium | 100 | - | 146 | 6,854,788 | 269,465 | 5,543 |
