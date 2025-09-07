# Local PostgreSQL Performance

## Append Performance

**Append Operations Details**:
- **Operation**: Simple event append operations
- **Scenario**: Basic event writing without conditions or business logic
- **Events**: Single event (1) or batch (100 events)
- **Model**: Generic test events with simple JSON data

| Dataset | Concurrency | Events | Throughput (ops/sec) | Latency (ns/op) | Memory (B/op) | Allocations |
|---------|-------------|--------|---------------------|-----------------|---------------|-------------|
| Medium | 1 | 1 | 4,163 | 240,115 | 1,875 | 55 |
| Small | 1 | 1 | 4,181 | 239,175 | 1,892 | 56 |
| Tiny | 1 | 1 | 4,227 | 236,658 | 1,875 | 55 |
| Tiny | 1 | 100 | 4,102 | 243,902 | 211,399 | 2,053 |
| Medium | 1 | 100 | 4,163 | 240,115 | 211,295 | 2,053 |
| Small | 1 | 100 | 4,102 | 243,902 | 211,359 | 2,053 |
| Tiny | 10 | 1 | 1,115 | 897,184 | 17,548 | 523 |
| Small | 10 | 1 | 1,134 | 881,469 | 17,548 | 523 |
| Medium | 10 | 1 | 1,168 | 856,167 | 17,530 | 523 |
| Medium | 10 | 100 | 103 | 9,682,726 | 2,094,330 | 20,490 |
| Tiny | 10 | 100 | 96 | 10,388,341 | 2,096,375 | 20,502 |
| Small | 10 | 100 | 93 | 10,845,719 | 2,095,403 | 20,496 |
| Medium | 100 | 1 | 125 | 7,637,824 | 182,499 | 5,275 |
| Tiny | 100 | 1 | 123 | 8,141,162 | 182,678 | 5,271 |
| Small | 100 | 1 | 125 | 8,020,184 | 182,843 | 5,266 |
| Medium | 100 | 100 | 9 | 116,582,513 | 20,954,190 | 205,059 |
| Small | 100 | 100 | 9 | 112,974,023 | 20,962,527 | 205,152 |
| Tiny | 100 | 100 | 9 | 111,427,837 | 20,963,011 | 205,138 |

## AppendIf Performance (No Conflict)

**AppendIf No Conflict Details**:
- **Attempted Events**: Number of events AppendIf operation tries to append (1 or 100 events per operation)
- **Actual Events**: Number of events successfully appended (1 or 100 events)
- **Past Events**: Number of existing events in database before benchmark (100 events for all scenarios)
- **Conflict Events**: 0 (no conflicts exist)

| Dataset | Concurrency | Attempted Events | Throughput (ops/sec) | Latency (ns/op) | Memory (B/op) | Allocations |
|---------|-------------|------------------|---------------------|-----------------|---------------|-------------|
| Medium | 1 | 1 | 1,278 | 782,174 | 4,462 | 96 |
| Small | 1 | 1 | 1,392 | 717,974 | 4,465 | 96 |
| Tiny | 1 | 1 | 1,594 | 627,621 | 4,464 | 96 |
| Tiny | 1 | 100 | 89 | 11,207,948 | 214,744 | 2,095 |
| Small | 1 | 100 | 98 | 10,241,461 | 214,217 | 2,093 |
| Medium | 1 | 100 | 101 | 9,941,695 | 214,216 | 2,092 |
| Tiny | 10 | 1 | 266 | 3,754,732 | 43,431 | 923 |
| Small | 10 | 1 | 252 | 3,935,961 | 43,409 | 923 |
| Medium | 10 | 1 | 252 | 3,962,822 | 43,380 | 923 |
| Small | 10 | 100 | 45 | 21,983,578 | 2,134,864 | 20,906 |
| Tiny | 10 | 100 | 45 | 22,345,893 | 2,138,365 | 20,927 |
| Medium | 10 | 100 | 44 | 22,636,543 | 2,133,460 | 20,891 |
| Tiny | 100 | 1 | 49 | 20,552,792 | 441,692 | 9,267 |
| Small | 100 | 1 | 46 | 21,848,396 | 441,156 | 9,263 |
| Medium | 100 | 1 | 45 | 22,333,843 | 440,918 | 9,263 |
| Medium | 100 | 100 | 4 | 206,938,770 | 21,359,597 | 209,027 |
| Tiny | 100 | 100 | 4 | 227,154,105 | 21,378,524 | 209,187 |
| Small | 100 | 100 | 4 | 234,958,336 | 21,370,916 | 209,131 |

## AppendIf Performance (With Conflict)

**AppendIf With Conflict Details**:
- **Attempted Events**: Number of events AppendIf operation tries to append (1 or 100 events per operation)
- **Actual Events**: Number of events successfully appended (0 - all operations fail due to conflicts)
- **Past Events**: Number of existing events in database before benchmark (100 events for all scenarios)
- **Conflict Events**: Number of conflicting events created before AppendIf (1, 10, or 100 events, matching concurrency level)

| Dataset | Concurrency | Attempted Events | Conflict Events | Throughput (ops/sec) | Latency (ns/op) | Memory (B/op) | Allocations |
|---------|-------------|------------------|-----------------|---------------------|-----------------|---------------|-------------|
| Medium | 1 | 1 | 1 | 1,171 | 853,820 | 5,905 | 145 |
| Small | 1 | 1 | 1 | 1,202 | 831,439 | 5,908 | 145 |
| Tiny | 1 | 1 | 1 | 1,193 | 838,785 | 5,914 | 146 |
| Small | 1 | 100 | 1 | 1,018 | 982,471 | 216,452 | 2,143 |
| Medium | 1 | 100 | 1 | 1,105 | 905,867 | 216,310 | 2,142 |
| Tiny | 1 | 100 | 1 | 1,064 | 938,336 | 216,659 | 2,144 |
| Tiny | 10 | 1 | 10 | 99 | 10,149,054 | 57,572 | 1,417 |
| Small | 10 | 1 | 10 | 100 | 10,007,610 | 57,505 | 1,416 |
| Medium | 10 | 1 | 10 | 99 | 10,079,142 | 57,472 | 1,416 |
| Tiny | 10 | 100 | 10 | 110 | 9,110,548 | 2,155,078 | 21,409 |
| Small | 10 | 100 | 10 | 112 | 8,965,148 | 2,152,027 | 21,389 |
| Medium | 10 | 100 | 10 | 112 | 8,920,976 | 2,150,702 | 21,380 |
| Tiny | 100 | 1 | 100 | 12 | 82,246,174 | 583,219 | 14,175 |
| Small | 100 | 1 | 100 | 12 | 81,240,740 | 582,678 | 14,171 |
| Medium | 100 | 1 | 100 | 12 | 85,652,522 | 581,817 | 14,165 |
| Medium | 100 | 100 | 100 | 11 | 87,552,629 | 21,506,783 | 213,813 |
| Tiny | 100 | 100 | 100 | 13 | 78,122,002 | 21,522,322 | 213,952 |
| Small | 100 | 100 | 100 | 13 | 77,745,206 | 21,514,603 | 213,898 |

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
| **ProjectStream** | Medium | 1 | ~100 | 5,665 | 176,554 | 64,465 | 1,460 |
| **Project** | Medium | 1 | ~100 | 5,035 | 198,522 | 55,460 | 1,450 |
| **ProjectStream** | Small | 1 | ~100 | 4,290 | 233,359 | 64,467 | 1,460 |
| **Project** | Small | 1 | ~100 | 4,198 | 238,241 | 55,463 | 1,450 |
| **ProjectStream** | Tiny | 1 | ~100 | 4,374 | 228,678 | 64,467 | 1,460 |
| **Project** | Tiny | 1 | ~100 | 4,310 | 232,214 | 55,462 | 1,450 |
| **ProjectStream** | Medium | 10 | ~100 | 1,328 | 753,367 | 643,575 | 14,574 |
| **Project** | Medium | 10 | ~100 | 1,444 | 692,991 | 553,447 | 14,474 |
| **ProjectStream** | Small | 10 | ~100 | 1,246 | 802,418 | 643,584 | 14,574 |
| **Project** | Small | 10 | ~100 | 1,299 | 769,654 | 553,464 | 14,474 |
| **ProjectStream** | Tiny | 10 | ~100 | 1,322 | 756,340 | 643,577 | 14,574 |
| **Project** | Tiny | 10 | ~100 | 1,439 | 695,025 | 553,447 | 14,474 |
| **ProjectStream** | Medium | 25 | ~100 | 612 | 1,633,405 | 1,608,646 | 36,431 |
| **Project** | Medium | 25 | ~100 | 651 | 1,536,585 | 1,383,318 | 36,180 |
| **ProjectStream** | Small | 25 | ~100 | 536 | 1,866,661 | 1,608,716 | 36,432 |
| **Project** | Small | 25 | ~100 | 563 | 1,774,790 | 1,383,371 | 36,181 |
| **ProjectStream** | Tiny | 25 | ~100 | 574 | 1,722,230 | 1,608,714 | 36,432 |
| **Project** | Tiny | 25 | ~100 | 694 | 1,653,638 | 1,383,320 | 36,180 |
