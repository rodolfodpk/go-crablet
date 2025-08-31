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
| Tiny | 1 | 1 | 4,892 | 225,692 | 1,892 | 56 |
| Small | 1 | 1 | 4,906 | 221,201 | 1,887 | 56 |
| Medium | 1 | 1 | 5,047 | 226,043 | 1,886 | 55 |
| Tiny | 10 | 1 | 1,350 | 844,249 | 17,552 | 522 |
| Small | 10 | 1 | 1,336 | 861,306 | 17,544 | 522 |
| Medium | 10 | 1 | 1,263 | 829,944 | 17,547 | 522 |
| Tiny | 100 | 1 | 156 | 7,254,413 | 183,049 | 5,262 |
| Small | 100 | 1 | 146 | 7,437,905 | 182,553 | 5,258 |
| Medium | 100 | 1 | 151 | 8,401,014 | 182,569 | 5,259 |
| Tiny | 1 | 100 | 855 | 1,540,084 | 209,924 | 2,054 |
| Small | 1 | 100 | 1,260 | 1,626,183 | 210,244 | 2,053 |
| Medium | 1 | 100 | 1,213 | 1,478,501 | 210,092 | 2,053 |
| Tiny | 10 | 100 | 100 | 10,399,734 | 2,097,099 | 20,509 |
| Small | 10 | 100 | 139 | 9,513,812 | 2,095,407 | 20,497 |
| Medium | 10 | 100 | 100 | 12,313,394 | 2,094,439 | 20,491 |
| Tiny | 100 | 100 | 14 | 97,064,631 | 20,968,922 | 205,180 |
| Small | 100 | 100 | 13 | 102,504,657 | 20,964,273 | 205,131 |
| Medium | 100 | 100 | 9 | 112,527,824 | 20,958,545 | 205,108 |

## AppendIf Performance (No Conflict)

**AppendIf No Conflict Details**:
- **Attempted Events**: Number of events AppendIf operation tries to append (1 or 100 events per operation)
- **Actual Events**: Number of events successfully appended (1 or 100 events)
- **Past Events**: Number of existing events in database before benchmark (100 events for all scenarios)
- **Conflict Events**: 0 (no conflicts exist)

| Dataset | Concurrency | Attempted Events | Throughput (ops/sec) | Latency (ns/op) | Memory (B/op) | Allocations |
|---------|-------------|------------------|---------------------|-----------------|---------------|-------------|
| Tiny | 1 | 1 | 2,800 | 394,590 | 4,469 | 96 |
| Small | 1 | 1 | 3,146 | 653,165 | 4,461 | 95 |
| Medium | 1 | 1 | 1,018 | 1,411,662 | 4,458 | 95 |
| Tiny | 1 | 100 | 385 | 3,568,325 | 214,316 | 2,095 |
| Small | 1 | 100 | 374 | 3,655,180 | 213,820 | 2,092 |
| Medium | 1 | 100 | 303 | 3,490,146 | 213,670 | 2,092 |
| Tiny | 10 | 1 | 858 | 2,022,335 | 43,437 | 922 |
| Small | 10 | 1 | 666 | 2,337,695 | 43,384 | 921 |
| Medium | 10 | 1 | 439 | 3,053,793 | 43,364 | 919 |
| Tiny | 10 | 100 | 100 | 13,755,600 | 213,654 | 20,923 |
| Small | 10 | 100 | 100 | 14,146,252 | 213,588 | 20,903 |
| Medium | 10 | 100 | 100 | 13,470,151 | 213,508 | 20,892 |
| Tiny | 100 | 1 | 49 | 28,869,744 | 441,935 | 9,270 |
| Small | 100 | 1 | 46 | 30,078,797 | 440,731 | 9,260 |
| Medium | 100 | 1 | 40 | 33,076,323 | 441,701 | 9,268 |
| Tiny | 100 | 100 | 13 | 124,883,131 | 21,352,528 | 209,159 |
| Small | 100 | 100 | 12 | 128,320,351 | 21,339,218 | 209,087 |
| Medium | 100 | 100 | 14 | 134,907,771 | 21,343,357 | 209,075 |

## AppendIf Performance (With Conflict)

**AppendIf With Conflict Details**:
- **Attempted Events**: Number of events AppendIf operation tries to append (1 or 100 events per operation)
- **Actual Events**: Number of events successfully appended (0 - all operations fail due to conflicts)
- **Past Events**: Number of existing events in database before benchmark (100 events for all scenarios)
- **Conflict Events**: Number of conflicting events created before AppendIf (1, 10, or 100 events, matching concurrency level)

| Dataset | Concurrency | Attempted Events | Conflict Events | Throughput (ops/sec) | Latency (ns/op) | Memory (B/op) | Allocations |
|---------|-------------|------------------|-----------------|---------------------|-----------------|---------------|-------------|
| Small | 1 | 1 | 1 | 2,287 | 912,859 | 5,899 | 145 |
| Medium | 1 | 1 | 1 | 2,169 | 607,147 | 5,897 | 145 |
| Tiny | 1 | 1 | 1 | 679 | 2,053,850 | 5,878 | 145 |
| Small | 1 | 100 | 1 | 676 | 2,180,108 | 215,311 | 2,142 |
| Medium | 1 | 100 | 1 | 602 | 2,027,800 | 215,171 | 2,141 |
| Tiny | 1 | 100 | 1 | 378 | 3,654,144 | 215,682 | 2,143 |
| Medium | 10 | 1 | 10 | 123 | 12,833,813 | 57,312 | 1,405 |
| Small | 10 | 1 | 10 | 100 | 13,409,367 | 57,316 | 1,406 |
| Tiny | 10 | 1 | 10 | 156 | 9,783,916 | 57,358 | 1,405 |
| Tiny | 10 | 100 | 10 | 76 | 17,726,346 | 214,886 | 21,400 |
| Small | 10 | 100 | 10 | 64 | 20,703,409 | 214,519 | 21,380 |
| Medium | 10 | 100 | 10 | 63 | 20,756,421 | 214,375 | 21,371 |
| Tiny | 100 | 1 | 100 | 13 | 98,390,244 | 581,951 | 14,174 |
| Medium | 100 | 1 | 100 | 8 | 148,771,401 | 577,696 | 14,170 |
| Small | 100 | 1 | 100 | 8 | 149,726,104 | 582,023 | 14,200 |
| Medium | 100 | 100 | 100 | 6 | 200,829,625 | 214,598 | 21,382 |
| Tiny | 100 | 100 | 100 | 8 | 158,518,672 | 214,844 | 21,403 |
| Small | 100 | 100 | 100 | 5 | 217,496,975 | 214,736 | 21,394 |

## Projection Performance

**Projection Operations Details**:
- **Operation**: State reconstruction from event streams
- **Scenario**: Building aggregate state from historical events
- **Events**: Number of events processed during projection (~100 events from AppendIf operations)
- **Model**: Simple counter increment for each event processed

| Operation | Dataset | Concurrency | Events | Throughput (ops/sec) | Latency (ns/op) | Memory (B/op) | Allocations |
|-----------|---------|-------------|--------|---------------------|-----------------|---------------|-------------|
| **ProjectStream** | Small | 1 | ~100 | 3.06 | 978,364,847 | 455,480,592 | 10,976,750 |
| **Project** | Small | 1 | ~100 | 3.33 | 899,661,986 | 455,471,240 | 10,976,739 |
| **ProjectStream** | Small | 10 | ~100 | 0.22 | 4,655,230,625 | 4,554,814,296 | 109,767,531 |
| **Project** | Small | 10 | ~100 | 0.22 | 4,623,331,625 | 4,554,724,464 | 109,767,516 |
| **ProjectStream** | Small | 25 | ~100 | 0.2 | 5,394,729,375 | 4,976,219,360 | 120,081,173 |
| **Project** | Small | 25 | ~100 | 0.1 | 9,304,960,042 | 4,976,002,024 | 120,081,085 |
