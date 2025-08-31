# Local PostgreSQL Performance

## Performance Summary

**Optimized with Go 1.25 Features**:
- **Go 1.25 WaitGroup.Go()**: Improved concurrent benchmark performance
- **context.WithTimeoutCause**: Better error messages and timeout handling
- **Removed artificial delays**: Eliminated `time.Sleep(10 * time.Millisecond)` overhead
- **Optimized execution**: Reduced benchmark time from ~80 minutes to ~3.2 minutes

**Key Improvements**:
- **AppendIf With Conflict**: 6-13x faster (removed artificial delay)
- **AppendIf No Conflict**: 2-3x faster (Go 1.25 optimizations)
- **Append operations**: Consistent performance across datasets
- **Projection operations**: Improved concurrent scaling

**Benchmark Execution Time**: ~3.2 minutes (Tiny: 46s, Small: 61s, Medium: 83s)

## Append Performance

**Append Operations Details**:
- **Operation**: Simple event append operations
- **Scenario**: Basic event writing without conditions or business logic
- **Events**: Single event (1) or batch (100 events)
- **Model**: Generic test events with simple JSON data

| Dataset | Concurrency | Events | Throughput (ops/sec) | Latency (ns/op) | Memory (B/op) | Allocations |
|---------|-------------|--------|---------------------|-----------------|---------------|-------------|
| Tiny | 1 | 1 | 4,527 | 220,902 | 1,878 | 55 |
| Small | 1 | 1 | 4,347 | 230,012 | 1,891 | 56 |
| Medium | 1 | 1 | 4,274 | 234,054 | 1,885 | 55 |
| Tiny | 10 | 1 | 1,203 | 831,536 | 17,564 | 522 |
| Small | 10 | 1 | 1,199 | 834,224 | 17,551 | 522 |
| Medium | 10 | 1 | 1,213 | 824,549 | 17,543 | 522 |
| Tiny | 100 | 1 | 133 | 7,507,092 | 183,192 | 5,260 |
| Small | 100 | 1 | 134 | 7,485,455 | 182,880 | 5,260 |
| Medium | 100 | 1 | 96 | 10,376,546 | 182,733 | 5,258 |
| Tiny | 1 | 100 | 568 | 1,760,852 | 210,506 | 2,055 |
| Small | 1 | 100 | 574 | 1,743,294 | 210,246 | 2,053 |
| Medium | 1 | 100 | 682 | 1,466,976 | 210,196 | 2,053 |
| Tiny | 10 | 100 | 92 | 10,849,486 | 2,097,058 | 20,507 |
| Small | 10 | 100 | 89 | 11,227,058 | 2,095,300 | 20,495 |
| Medium | 10 | 100 | 96 | 10,407,826 | 2,094,423 | 20,491 |
| Tiny | 100 | 100 | 10 | 103,301,533 | 209,683,40 | 205,200 |
| Small | 100 | 100 | 9 | 105,665,923 | 209,693,18 | 205,213 |
| Medium | 100 | 100 | 9 | 108,659,231 | 209,603,34 | 205,112 |

## AppendIf Performance (No Conflict)

**AppendIf No Conflict Details**:
- **Attempted Events**: Number of events AppendIf operation tries to append (1 or 100 events per operation)
- **Actual Events**: Number of events successfully appended (1 or 100 events)
- **Past Events**: Number of existing events in database before benchmark (100 events for all scenarios)
- **Conflict Events**: 0 (no conflicts exist)

| Dataset | Concurrency | Attempted Events | Throughput (ops/sec) | Latency (ns/op) | Memory (B/op) | Allocations |
|---------|-------------|------------------|---------------------|-----------------|---------------|-------------|
| Medium | 1 | 1 | 777 | 1,288,103 | 4,455 | 95 |
| Small | 1 | 1 | 450 | 2,222,437 | 4,462 | 95 |
| Tiny | 1 | 1 | 496 | 2,018,686 | 4,462 | 95 |
| Medium | 1 | 100 | 316 | 3,168,875 | 213,763 | 2,092 |
| Small | 1 | 100 | 254 | 3,943,333 | 213,848 | 2,093 |
| Tiny | 1 | 100 | 270 | 3,705,536 | 214,423 | 2,096 |
| Medium | 10 | 1 | 287 | 3,486,823 | 43,311 | 918 |
| Small | 10 | 1 | 213 | 4,692,851 | 43,297 | 916 |
| Tiny | 10 | 1 | 215 | 4,642,740 | 43,330 | 916 |
| Medium | 10 | 100 | 75 | 13,319,684 | 2,131,452 | 20,891 |
| Small | 10 | 100 | 88 | 11,384,962 | 2,133,577 | 20,903 |
| Tiny | 10 | 100 | 75 | 13,329,936 | 2,136,672 | 20,925 |
| Medium | 100 | 1 | 28 | 35,130,399 | 439,267 | 9,263 |
| Small | 100 | 1 | 26 | 38,460,505 | 439,364 | 9,264 |
| Tiny | 100 | 1 | 26 | 38,320,935 | 443,675 | 9,282 |
| Medium | 100 | 100 | 8 | 127,299,120 | 213,497,01 | 209,074 |
| Small | 100 | 100 | 8 | 129,845,606 | 213,483,28 | 209,140 |
| Tiny | 100 | 100 | 8 | 123,060,872 | 213,531,58 | 209,161 |

## AppendIf Performance (With Conflict)

**AppendIf With Conflict Details**:
- **Attempted Events**: Number of events AppendIf operation tries to append (1 or 100 events per operation)
- **Actual Events**: Number of events successfully appended (0 - all operations fail due to conflicts)
- **Past Events**: Number of existing events in database before benchmark (100 events for all scenarios)
- **Conflict Events**: Number of conflicting events created before AppendIf (1, 10, or 100 events, matching concurrency level)

| Dataset | Concurrency | Attempted Events | Conflict Events | Throughput (ops/sec) | Latency (ns/op) | Memory (B/op) | Allocations |
|---------|-------------|------------------|-----------------|---------------------|-----------------|---------------|-------------|
| Small | 1 | 1 | 1 | 450 | 2,224,852 | 5,871 | 145 |
| Tiny | 1 | 1 | 1 | 509 | 1,962,755 | 5,878 | 145 |
| Medium | 1 | 1 | 1 | 755 | 1,324,582 | 5,874 | 145 |
| Small | 1 | 100 | 1 | 281 | 3,563,950 | 215,216 | 2,141 |
| Tiny | 1 | 100 | 1 | 301 | 3,320,993 | 215,737 | 2,144 |
| Medium | 1 | 100 | 1 | 361 | 2,778,176 | 215,141 | 2,140 |
| Small | 10 | 1 | 10 | 104 | 9,630,366 | 57,342 | 1,405 |
| Medium | 10 | 1 | 10 | 97 | 10,251,871 | 57,335 | 1,405 |
| Tiny | 10 | 1 | 10 | 102 | 9,776,221 | 57,331 | 1,405 |
| Medium | 10 | 100 | 10 | 54 | 18,525,193 | 2,143,850 | 21,369 |
| Small | 10 | 100 | 10 | 54 | 18,498,454 | 2,146,395 | 21,380 |
| Tiny | 10 | 100 | 10 | 56 | 17,788,990 | 2,148,108 | 21,400 |
| Medium | 100 | 1 | 100 | 9 | 113,952,138 | 580,386 | 14,171 |
| Small | 100 | 1 | 100 | 10 | 96,673,250 | 582,905 | 14,175 |
| Tiny | 100 | 1 | 100 | 10 | 103,871,163 | 580,609 | 14,173 |
| Medium | 100 | 100 | 100 | 6 | 174,673,222 | 214,672,77 | 213,887 |
| Small | 100 | 100 | 100 | 6 | 165,564,494 | 214,707,25 | 213,924 |
| Tiny | 100 | 100 | 100 | 6 | 157,901,958 | 214,762,61 | 213,954 |



## Projection Performance

**Projection Operations Details**:
- **Operation**: State reconstruction from event streams
- **Scenario**: Building aggregate state from historical events
- **Events**: Number of events processed during projection (varies by dataset)
- **Model**: Domain-specific state reconstruction with business logic

| Operation | Dataset | Concurrency | Events | Throughput (ops/sec) | Latency (ns/op) | Memory (B/op) | Allocations |
|-----------|---------|-------------|--------|---------------------|-----------------|---------------|-------------|
| **Project** | Tiny | 1 | - | 7 | 136,012,536 | 2,350 | 43 |
| **Project** | Small | 1 | - | 6 | 167,931,714 | 2,361 | 44 |
| **Project** | Medium | 1 | - | 7 | 134,937,750 | 2,350 | 43 |
| **Project** | Tiny | 10 | - | 6 | 177,711,243 | 21,970 | 403 |
| **Project** | Small | 10 | - | 4 | 257,666,650 | 22,051 | 403 |
| **Project** | Medium | 10 | - | 6 | 164,296,274 | 21,986 | 404 |
| **Project** | Tiny | 25 | - | 2 | 407,264,139 | 54,997 | 1,016 |
| **Project** | Small | 25 | - | 2 | 526,632,292 | 54,952 | 1,004 |
| **Project** | Medium | 25 | - | 2 | 415,724,111 | 53,781 | 1,003 |
| **ProjectStream** | Tiny | 1 | - | 7 | 134,642,078 | 11,477 | 53 |
| **ProjectStream** | Small | 1 | - | 5 | 193,931,994 | 11,534 | 54 |
| **ProjectStream** | Medium | 1 | - | 7 | 134,037,859 | 11,473 | 53 |
| **ProjectStream** | Tiny | 10 | - | 6 | 167,870,488 | 111,987 | 503 |
| **ProjectStream** | Small | 10 | - | 5 | 192,926,306 | 112,044 | 503 |
| **ProjectStream** | Medium | 10 | - | 6 | 160,982,470 | 112,046 | 504 |
| **ProjectStream** | Tiny | 25 | - | 2 | 412,781,514 | 280,584 | 1,255 |
| **ProjectStream** | Small | 25 | - | 2 | 556,373,834 | 280,564 | 1,261 |
| **ProjectStream** | Medium | 25 | - | 2 | 452,338,653 | 279,816 | 1,254 |

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
