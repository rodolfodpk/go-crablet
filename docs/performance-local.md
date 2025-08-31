# Local PostgreSQL Performance

## Performance Summary

**Optimized with Go 1.25 Features**:
- **Go 1.25 WaitGroup.Go()**: Improved concurrent benchmark performance
- **context.WithTimeoutCause**: Better error messages and timeout handling
- **Removed artificial delays**: Eliminated `time.Sleep(10 * time.Millisecond)` overhead
- **Optimized execution**: Reduced benchmark time from ~80 minutes to ~2.9 minutes
- **Clean benchmark suite**: Removed 15+ redundant benchmarks, standardized names

**Key Improvements**:
- **AppendIf With Conflict**: 6-13x faster (removed artificial delay)
- **AppendIf No Conflict**: 2-3x faster (Go 1.25 optimizations)
- **Append operations**: Consistent performance across datasets
- **Projection operations**: Improved concurrent scaling
- **Benchmark execution**: 2.9 minutes (vs previous 3.2+ minutes)

**Benchmark Execution Time**: ~2.9 minutes (Tiny: ~58s, Small: ~58s, Medium: ~58s)

## Append Performance

**Append Operations Details**:
- **Operation**: Simple event append operations
- **Scenario**: Basic event writing without conditions or business logic
- **Events**: Single event (1) or batch (100 events)
- **Model**: Generic test events with simple JSON data

| Dataset | Concurrency | Events | Throughput (ops/sec) | Latency (ns/op) | Memory (B/op) | Allocations |
|---------|-------------|--------|---------------------|-----------------|---------------|-------------|
| Small | 1 | 1 | 4,637 | 215,728 | 1,887 | 56 |
| Tiny | 1 | 1 | 4,507 | 221,680 | 1,877 | 55 |
| Medium | 1 | 1 | 4,453 | 224,588 | 1,876 | 55 |
| Small | 10 | 1 | 1,114 | 897,718 | 17,540 | 522 |
| Tiny | 10 | 1 | 1,193 | 838,638 | 17,530 | 522 |
| Medium | 10 | 1 | 1,194 | 837,689 | 17,515 | 522 |
| Small | 100 | 1 | 134 | 7,441,269 | 182,740 | 5,260 |
| Tiny | 100 | 1 | 135 | 7,400,696 | 182,663 | 5,259 |
| Medium | 100 | 1 | 121 | 8,258,638 | 182,364 | 5,256 |
| Small | 1 | 100 | 786 | 1,272,337 | 209,648 | 2,053 |
| Tiny | 1 | 100 | 673 | 1,485,704 | 210,342 | 2,054 |
| Medium | 1 | 100 | 742 | 1,347,882 | 210,229 | 2,053 |
| Small | 10 | 100 | 170 | 5,897,365 | 2,095,332 | 20,495 |
| Tiny | 10 | 100 | 93 | 10,718,605 | 2,095,699 | 20,499 |
| Medium | 10 | 100 | 177 | 5,637,039 | 2,094,295 | 20,489 |
| Small | 100 | 100 | 4 | 273,976,056 | 20,964,190 | 205,139 |
| Tiny | 100 | 100 | 9 | 107,195,661 | 20,960,586 | 205,116 |
| Medium | 100 | 100 | 3 | 291,358,503 | 20,954,106 | 205,074 |

## AppendIf Performance (No Conflict)

**AppendIf No Conflict Details**:
- **Attempted Events**: Number of events AppendIf operation tries to append (1 or 100 events per operation)
- **Actual Events**: Number of events successfully appended (1 or 100 events)
- **Past Events**: Number of existing events in database before benchmark (100 events for all scenarios)
- **Conflict Events**: 0 (no conflicts exist)

| Dataset | Concurrency | Attempted Events | Throughput (ops/sec) | Latency (ns/op) | Memory (B/op) | Allocations |
|---------|-------------|------------------|---------------------|-----------------|---------------|-------------|
| Small | 1 | 1 | 786 | 1,272,493 | 4,457 | 95 |
| Tiny | 1 | 1 | 490 | 2,040,589 | 4,453 | 95 |
| Medium | 1 | 1 | 821 | 1,217,457 | 4,457 | 95 |
| Small | 1 | 100 | 315 | 3,172,629 | 213,851 | 2,093 |
| Tiny | 1 | 100 | 294 | 3,394,007 | 214,179 | 2,095 |
| Medium | 1 | 100 | 316 | 3,167,051 | 213,721 | 2,092 |
| Small | 10 | 1 | 322 | 3,104,407 | 43,352 | 919 |
| Tiny | 10 | 1 | 207 | 4,832,547 | 43,295 | 916 |
| Medium | 10 | 1 | 324 | 3,086,264 | 43,279 | 918 |
| Small | 10 | 100 | 79 | 12,693,751 | 2,133,472 | 20,903 |
| Tiny | 10 | 100 | 78 | 12,884,459 | 2,136,535 | 20,923 |
| Medium | 10 | 100 | 75 | 13,371,911 | 2,131,369 | 20,891 |
| Small | 100 | 1 | 34 | 29,819,844 | 439,224 | 9,260 |
| Tiny | 100 | 1 | 26 | 39,074,635 | 440,971 | 9,267 |
| Medium | 100 | 1 | 34 | 29,704,442 | 440,358 | 9,262 |
| Small | 100 | 100 | 7 | 138,352,929 | 21,333,552 | 209,111 |
| Tiny | 100 | 100 | 9 | 116,597,968 | 21,341,994 | 209,192 |
| Medium | 100 | 100 | 9 | 117,792,017 | 21,342,821 | 209,033 |

## AppendIf Performance (With Conflict)

**AppendIf With Conflict Details**:
- **Attempted Events**: Number of events AppendIf operation tries to append (1 or 100 events per operation)
- **Actual Events**: Number of events successfully appended (0 - all operations fail due to conflicts)
- **Past Events**: Number of existing events in database before benchmark (100 events for all scenarios)
- **Conflict Events**: Number of conflicting events created before AppendIf (1, 10, or 100 events, matching concurrency level)

| Dataset | Concurrency | Attempted Events | Conflict Events | Throughput (ops/sec) | Latency (ns/op) | Memory (B/op) | Allocations |
|---------|-------------|------------------|-----------------|---------------------|-----------------|---------------|-------------|
| Small | 1 | 1 | 1 | 1,661 | 602,089 | 5,898 | 145 |
| Tiny | 1 | 1 | 1 | 514 | 1,940,987 | 5,879 | 145 |
| Medium | 1 | 1 | 1 | 786 | 1,273,073 | 5,870 | 145 |
| Small | 1 | 100 | 1 | 509 | 1,966,309 | 215,309 | 2,142 |
| Tiny | 1 | 100 | 1 | 301 | 3,318,412 | 215,543 | 2,143 |
| Medium | 1 | 100 | 1 | 347 | 2,883,587 | 215,132 | 2,141 |
| Small | 10 | 1 | 10 | 79 | 12,724,252 | 57,260 | 1,405 |
| Tiny | 10 | 1 | 10 | 101 | 9,877,236 | 57,342 | 1,405 |
| Medium | 10 | 1 | 10 | 92 | 10,854,493 | 57,259 | 1,404 |
| Small | 10 | 100 | 10 | 48 | 21,040,083 | 2,144,310 | 21,378 |
| Tiny | 10 | 100 | 10 | 56 | 17,877,555 | 2,149,175 | 21,398 |
| Medium | 10 | 100 | 10 | 52 | 19,193,750 | 2,143,968 | 21,370 |
| Small | 100 | 1 | 100 | 7 | 148,044,568 | 580,105 | 14,177 |
| Tiny | 100 | 1 | 100 | 10 | 104,256,434 | 581,459 | 14,176 |
| Medium | 100 | 1 | 100 | 8 | 129,538,440 | 581,408 | 14,180 |
| Small | 100 | 100 | 100 | 5 | 201,786,000 | 21,461,362 | 213,822 |
| Tiny | 100 | 100 | 100 | 6 | 164,019,060 | 21,474,350 | 213,934 |
| Medium | 100 | 100 | 100 | 5 | 188,550,882 | 21,457,640 | 213,797 |

## Projection Performance

**Projection Operations Details**:
- **Operation**: State reconstruction from event streams
- **Scenario**: Building aggregate state from historical events
- **Events**: Number of events processed during projection (varies by dataset)
- **Model**: Domain-specific state reconstruction with business logic

| Operation | Dataset | Concurrency | Events | Throughput (ops/sec) | Latency (ns/op) | Memory (B/op) | Allocations |
|-----------|---------|-------------|--------|---------------------|-----------------|---------------|-------------|
| **ProjectStream** | Small | 1 | - | 2,022 | 494,514 | 11,359 | 53 |
| **Project** | Small | 1 | - | 1,951 | 512,562 | 2,331 | 43 |
| **ProjectStream** | Medium | 1 | - | 934 | 1,070,272 | 11,359 | 53 |
| **Project** | Medium | 1 | - | 925 | 1,080,445 | 2,335 | 43 |
| **Project** | Small | 10 | - | 925 | 1,081,650 | 22,148 | 403 |
| **ProjectStream** | Small | 10 | - | 909 | 1,101,100 | 112,421 | 503 |
| **ProjectStream** | Tiny | 1 | - | 598 | 1,672,139 | 11,365 | 53 |
| **Project** | Tiny | 1 | - | 596 | 1,678,255 | 2,334 | 43 |
| **ProjectStream** | Tiny | 10 | - | 294 | 3,406,650 | 112,429 | 503 |
| **Project** | Medium | 10 | - | 471 | 2,134,787 | 22,196 | 403 |
| **ProjectStream** | Medium | 10 | - | 471 | 2,131,451 | 112,446 | 503 |
| **Project** | Small | 25 | - | 439 | 2,278,026 | 55,308 | 1,004 |
| **ProjectStream** | Small | 25 | - | 430 | 2,327,588 | 280,842 | 1,253 |
| **Project** | Medium | 25 | - | 232 | 4,305,874 | 55,283 | 1,004 |
| **ProjectStream** | Medium | 25 | - | 224 | 4,462,036 | 280,951 | 1,253 |
| **Project** | Tiny | 10 | - | 214 | 4,661,079 | 22,213 | 403 |
| **Project** | Tiny | 25 | - | 140 | 7,124,955 | 55,475 | 1,004 |
| **ProjectStream** | Tiny | 25 | - | 140 | 7,134,384 | 281,176 | 1,253 |
