# Local PostgreSQL Performance

## Append Performance

**Append Operations Details**:
- **Operation**: Simple event append operations
- **Scenario**: Basic event writing without conditions or business logic
- **Events**: Single event (1) or batch (100 events)
- **Model**: Generic test events with simple JSON data

| Dataset | Concurrency | Events | Throughput (ops/sec) | Latency (ns/op) | Memory (B/op) | Allocations |
|---------|-------------|--------|---------------------|-----------------|---------------|-------------|
| Medium | 1 | 1 | 4,119 | 242,736 | 1,875 | 55 |
| Small | 1 | 1 | 4,190 | 238,620 | 1,885 | 56 |
| Tiny | 1 | 1 | 4,622 | 216,341 | 1,877 | 56 |
| Tiny | 1 | 100 | 4,887 | 204,621 | 211,245 | 2,053 |
| Medium | 1 | 100 | 4,231 | 236,292 | 211,199 | 2,053 |
| Small | 1 | 100 | 3,538 | 282,654 | 209,981 | 2,053 |
| Tiny | 10 | 1 | 1,083 | 923,450 | 17,536 | 523 |
| Small | 10 | 1 | 1,093 | 914,688 | 17,553 | 523 |
| Medium | 10 | 1 | 1,121 | 892,383 | 17,523 | 523 |
| Medium | 10 | 100 | 90 | 11,150,705 | 2,094,267 | 20,488 |
| Tiny | 10 | 100 | 86 | 11,611,182 | 2,095,955 | 20,500 |
| Small | 10 | 100 | 85 | 11,757,119 | 2,095,489 | 20,497 |
| Medium | 100 | 1 | 137 | 7,314,000 | 182,565 | 5,281 |
| Tiny | 100 | 1 | 128 | 7,787,707 | 182,671 | 5,275 |
| Small | 100 | 1 | 137 | 7,291,747 | 182,771 | 5,277 |
| Medium | 100 | 100 | 9 | 109,567,227 | 20,954,504 | 205,062 |
| Small | 100 | 100 | 9 | 116,115,032 | 20,962,771 | 205,132 |
| Tiny | 100 | 100 | 9 | 106,462,125 | 20,959,411 | 205,094 |

## AppendIf Performance (No Conflict)

**AppendIf No Conflict Details**:
- **Attempted Events**: Number of events AppendIf operation tries to append (1 or 100 events per operation)
- **Actual Events**: Number of events successfully appended (1 or 100 events)
- **Past Events**: Number of existing events in database before benchmark (100 events for all scenarios)
- **Conflict Events**: 0 (no conflicts exist)

| Dataset | Concurrency | Attempted Events | Throughput (ops/sec) | Latency (ns/op) | Memory (B/op) | Allocations |
|---------|-------------|------------------|---------------------|-----------------|---------------|-------------|
| Medium | 1 | 1 | 1,457 | 686,165 | 4,460 | 95 |
| Tiny | 1 | 1 | 1,400 | 714,688 | 4,466 | 96 |
| Small | 1 | 1 | 1,574 | 635,688 | 4,461 | 95 |
| Tiny | 1 | 100 | 264 | 3,781,071 | 214,675 | 2,095 |
| Small | 1 | 100 | 266 | 3,759,592 | 214,342 | 2,093 |
| Medium | 1 | 100 | 213 | 4,684,493 | 214,106 | 2,092 |
| Tiny | 10 | 1 | 315 | 3,174,311 | 43,427 | 923 |
| Small | 10 | 1 | 306 | 3,265,270 | 43,426 | 923 |
| Medium | 10 | 1 | 291 | 3,443,409 | 43,382 | 923 |
| Small | 10 | 100 | 53 | 18,835,277 | 2,135,966 | 20,903 |
| Tiny | 10 | 100 | 67 | 14,882,554 | 2,139,238 | 20,927 |
| Medium | 10 | 100 | 53 | 18,923,511 | 2,134,501 | 20,892 |
| Tiny | 100 | 1 | 53 | 18,965,121 | 441,896 | 9,268 |
| Small | 100 | 1 | 43 | 23,407,549 | 440,802 | 9,261 |
| Medium | 100 | 1 | 48 | 20,938,356 | 440,955 | 9,261 |
| Medium | 100 | 100 | 6 | 128,954,533 | 21,362,109 | 209,023 |
| Tiny | 100 | 100 | 8 | 124,781,365 | 21,373,342 | 209,149 |
| Small | 100 | 100 | 6 | 172,831,954 | 21,373,717 | 209,120 |

## AppendIf Performance (With Conflict)

**AppendIf With Conflict Details**:
- **Attempted Events**: Number of events AppendIf operation tries to append (1 or 100 events per operation)
- **Actual Events**: Number of events successfully appended (0 - all operations fail due to conflicts)
- **Past Events**: Number of existing events in database before benchmark (100 events for all scenarios)
- **Conflict Events**: Number of conflicting events created before AppendIf (1, 10, or 100 events, matching concurrency level)

| Dataset | Concurrency | Attempted Events | Conflict Events | Throughput (ops/sec) | Latency (ns/op) | Memory (B/op) | Allocations |
|---------|-------------|------------------|-----------------|---------------------|-----------------|---------------|-------------|
| Medium | 1 | 1 | 1 | 1,251 | 799,542 | 5,907 | 145 |
| Tiny | 1 | 1 | 1 | 1,271 | 786,508 | 5,910 | 145 |
| Small | 1 | 1 | 1 | 1,164 | 859,054 | 5,909 | 145 |
| Small | 1 | 100 | 1 | 1,073 | 932,547 | 216,390 | 2,142 |
| Medium | 1 | 100 | 1 | 1,081 | 925,038 | 216,324 | 2,142 |
| Tiny | 1 | 100 | 1 | 1,008 | 992,745 | 216,713 | 2,144 |
| Tiny | 10 | 1 | 10 | 92 | 10,843,551 | 57,566 | 1,417 |
| Small | 10 | 1 | 10 | 94 | 10,614,842 | 57,522 | 1,416 |
| Medium | 10 | 1 | 10 | 87 | 11,478,349 | 57,506 | 1,416 |
| Tiny | 10 | 100 | 10 | 111 | 9,048,542 | 2,154,924 | 21,408 |
| Small | 10 | 100 | 10 | 104 | 9,660,063 | 2,152,117 | 21,391 |
| Medium | 10 | 100 | 10 | 97 | 10,312,544 | 2,150,714 | 21,380 |
| Tiny | 100 | 1 | 100 | 12 | 83,314,794 | 583,497 | 14,176 |
| Small | 100 | 1 | 100 | 12 | 84,395,612 | 581,912 | 14,165 |
| Medium | 100 | 1 | 100 | 11 | 91,630,902 | 581,738 | 14,165 |
| Medium | 100 | 100 | 100 | 13 | 76,159,041 | 21,505,262 | 213,829 |
| Tiny | 100 | 100 | 100 | 12 | 80,609,910 | 21,522,378 | 213,954 |
| Small | 100 | 100 | 100 | 13 | 79,954,391 | 21,515,959 | 213,907 |

## Projection Performance

**Projection Operations Details**:
- **Operation**: State reconstruction from event streams using core API's built-in concurrency controls
- **Scenario**: Building aggregate state from historical events with proper goroutine limits
- **Events**: Number of events processed during projection (~100 events from Append benchmarks)
- **Model**: Domain-specific state reconstruction with business logic
- **Architecture**: Uses Go 1.25 concurrency features and core API's built-in goroutine limits
- **Performance**: Realistic throughput with proper resource management

| Operation | Dataset | Concurrency | Events | Throughput (ops/sec) | Latency (ns/op) | Memory (B/op) | Allocations |
|-----------|---------|-------------|--------|---------------------|-----------------|---------------|-------------|
| **ProjectStream** | Medium | 1 | ~100 | 6,081 | 164,477 | 64,465 | 1,460 |
| **Project** | Medium | 1 | ~100 | 5,883 | 169,949 | 55,460 | 1,450 |
| **ProjectStream** | Small | 1 | ~100 | 4,453 | 224,543 | 64,469 | 1,460 |
| **Project** | Small | 1 | ~100 | 4,404 | 227,092 | 55,462 | 1,450 |
| **ProjectStream** | Tiny | 1 | ~100 | 4,619 | 216,513 | 64,468 | 1,460 |
| **Project** | Tiny | 1 | ~100 | 4,521 | 221,077 | 55,462 | 1,450 |
| **ProjectStream** | Medium | 10 | ~100 | 1,479 | 675,939 | 643,570 | 14,574 |
| **Project** | Medium | 10 | ~100 | 1,572 | 635,451 | 553,418 | 14,474 |
| **ProjectStream** | Small | 10 | ~100 | 1,299 | 770,673 | 643,595 | 14,574 |
| **Project** | Small | 10 | ~100 | 1,378 | 725,374 | 553,478 | 14,474 |
| **ProjectStream** | Tiny | 10 | ~100 | 1,298 | 770,196 | 643,573 | 14,574 |
| **Project** | Tiny | 10 | ~100 | 1,423 | 702,354 | 553,444 | 14,474 |
| **ProjectStream** | Medium | 25 | ~100 | 678 | 1,475,671 | 1,608,672 | 36,432 |
| **Project** | Medium | 25 | ~100 | 683 | 1,464,470 | 1,383,302 | 36,180 |
| **ProjectStream** | Small | 25 | ~100 | 568 | 1,760,567 | 1,608,697 | 36,432 |
| **Project** | Small | 25 | ~100 | 587 | 1,702,504 | 1,383,359 | 36,181 |
| **ProjectStream** | Tiny | 25 | ~100 | 505 | 1,979,990 | 1,608,564 | 36,431 |
| **Project** | Tiny | 25 | ~100 | 576 | 1,734,903 | 1,383,309 | 36,180 |
