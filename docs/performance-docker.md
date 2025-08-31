# Docker PostgreSQL Performance

## Performance Results

## Append Performance

**Append Operations Details**:
- **Operation**: Simple event append operations
- **Scenario**: Basic event writing without conditions or business logic
- **Events**: Single event (1) or batch (100 events)
- **Model**: Generic test events with simple JSON data

| Dataset | Concurrency | Events | Throughput (ops/sec) | Latency (ns/op) | Memory (B/op) | Allocations |
|---------|-------------|--------|---------------------|-----------------|---------------|-------------|
| Tiny | 1 | 1 | 4,245 | 235,601 | 1,884 | 56 |
| Medium | 1 | 1 | 4,199 | 238,131 | 1,882 | 56 |
| Small | 1 | 1 | 3,821 | 261,668 | 1,888 | 56 |
| Medium | 10 | 1 | 1,206 | 829,250 | 17,548 | 523 |
| Tiny | 10 | 1 | 1,166 | 857,995 | 17,559 | 523 |
| Small | 10 | 1 | 1,160 | 861,590 | 17,554 | 523 |
| Medium | 1 | 100 | 678 | 1,474,140 | 211,359 | 2,053 |
| Small | 1 | 100 | 522 | 1,914,980 | 211,276 | 2,053 |
| Tiny | 1 | 100 | 476 | 2,098,884 | 211,665 | 2,054 |
| Small | 100 | 1 | 147 | 6,808,841 | 182,705 | 5,275 |
| Tiny | 100 | 1 | 142 | 7,042,692 | 183,156 | 5,285 |
| Medium | 100 | 1 | 130 | 7,717,958 | 182,656 | 5,277 |
| Medium | 10 | 100 | 101 | 9,910,265 | 2,094,603 | 20,491 |
| Small | 10 | 100 | 98 | 10,233,074 | 2,095,527 | 20,500 |
| Tiny | 10 | 100 | 92 | 10,822,192 | 2,097,196 | 20,508 |
| Tiny | 100 | 100 | 9 | 114,265,729 | 20,965,165 | 205,137 |
| Small | 100 | 100 | 9 | 107,913,216 | 20,962,283 | 205,131 |
| Medium | 100 | 100 | 8 | 117,799,050 | 20,956,685 | 205,081 |

## AppendIf Performance (No Conflict)

**AppendIf No Conflict Details**:
- **Attempted Events**: Number of events AppendIf operation tries to append (1 or 100 events per operation)
- **Actual Events**: Number of events successfully appended (1 or 100 events)
- **Past Events**: Number of existing events in database before benchmark (100 events for all scenarios)
- **Conflict Events**: 0 (no conflicts exist)

| Dataset | Concurrency | Attempted Events | Throughput (ops/sec) | Latency (ns/op) | Memory (B/op) | Allocations |
|---------|-------------|------------------|---------------------|-----------------|---------------|-------------|
| Tiny | 10 | 1 | 1,483 | 674,000 | 43,466 | 923 |
| Small | 10 | 1 | 1,483 | 674,000 | 43,466 | 923 |
| Medium | 10 | 1 | 1,483 | 674,000 | 43,466 | 923 |
| Tiny | 1 | 100 | 718 | 1,392,000 | 213,933 | 2,093 |
| Small | 1 | 100 | 718 | 1,392,000 | 213,933 | 2,093 |
| Medium | 1 | 100 | 718 | 1,392,000 | 213,933 | 2,093 |
| Tiny | 1 | 1 | 631 | 1,584,000 | 4,505 | 95 |
| Small | 1 | 1 | 631 | 1,584,000 | 4,505 | 95 |
| Medium | 1 | 1 | 631 | 1,584,000 | 4,505 | 95 |
| Tiny | 10 | 100 | 202 | 4,950,000 | 2,136,535 | 20,902 |
| Small | 10 | 100 | 202 | 4,950,000 | 2,136,535 | 20,902 |
| Medium | 10 | 100 | 202 | 4,950,000 | 2,136,535 | 20,902 |
| Tiny | 100 | 1 | 61 | 16,400,000 | 441,655 | 9,268 |
| Small | 100 | 1 | 61 | 16,400,000 | 441,655 | 9,268 |
| Medium | 100 | 1 | 61 | 16,400,000 | 441,655 | 9,268 |
| Tiny | 100 | 100 | 19 | 52,600,000 | 21,361,007 | 209,098 |
| Small | 100 | 100 | 19 | 52,600,000 | 21,361,007 | 209,098 |
| Medium | 100 | 100 | 19 | 52,600,000 | 21,361,007 | 209,098 |

## AppendIf Performance (With Conflict)

**AppendIf With Conflict Details**:
- **Attempted Events**: Number of events AppendIf operation tries to append (1 or 100 events per operation)
- **Actual Events**: Number of events successfully appended (0 - all operations fail due to conflicts)
- **Past Events**: Number of existing events in database before benchmark (100 events for all scenarios)
- **Conflict Events**: Number of conflicting events created before AppendIf (1, 10, or 100 events, matching concurrency level)

| Dataset | Concurrency | Attempted Events | Conflict Events | Throughput (ops/sec) | Latency (ns/op) | Memory (B/op) | Allocations |
|---------|-------------|------------------|-----------------|---------------------|-----------------|---------------|-------------|
| Tiny | 1 | 1 | 1 | 177 | 5,650,000 | 5,885 | 144 |
| Small | 1 | 1 | 1 | 170 | 5,880,000 | 5,870 | 144 |
| Tiny | 1 | 100 | 1 | 100 | 10,000,000 | 215,457 | 2,144 |
| Small | 1 | 100 | 1 | 100 | 10,000,000 | 214,760 | 2,140 |
| Medium | 1 | 1 | 1 | 100 | 10,000,000 | 5,909 | 144 |
| Small | 10 | 1 | 10 | 108 | 9,260,000 | 57,272 | 1,405 |
| Tiny | 10 | 1 | 10 | 106 | 9,430,000 | 57,260 | 1,405 |
| Small | 10 | 100 | 10 | 82 | 12,200,000 | 2,146,121 | 21,379 |
| Tiny | 10 | 100 | 10 | 80 | 12,500,000 | 2,149,047 | 21,399 |
| Small | 100 | 1 | 100 | 26 | 38,500,000 | 581,459 | 14,178 |
| Tiny | 100 | 1 | 100 | 24 | 41,700,000 | 581,917 | 14,183 |
| Medium | 100 | 1 | 100 | 14 | 71,400,000 | 583,659 | 14,171 |
| Medium | 1 | 100 | 1 | 18 | 55,600,000 | 213,399 | 2,140 |
| Medium | 10 | 1 | 10 | 18 | 55,600,000 | 57,949 | 1,409 |
| Medium | 10 | 100 | 10 | 18 | 55,600,000 | 2,131,168 | 21,370 |
| Tiny | 100 | 100 | 100 | 12 | 83,300,000 | 21,488,142 | 213,965 |
| Small | 100 | 100 | 100 | 12 | 83,300,000 | 21,482,203 | 213,947 |
| Medium | 100 | 100 | 100 | 12 | 83,300,000 | 21,467,655 | 213,808 |
## Read and Projection Performance

| Operation | Dataset | Concurrency | Events | Throughput (ops/sec) | Latency (ns/op) | Memory (B/op) | Allocations |
|-----------|---------|-------------|--------|---------------------|-----------------|---------------|-------------|
| **Read_Single** | Tiny | 1 | - | 49 | 20,070,509 | 99,628 | 124,930 |
| **Read_Single** | Small | 1 | - | 123 | 8,124,236 | 102,439 | 131,365 |
| **Read_Single** | Medium | 1 | - | 117 | 8,543,448 | 101,916 | 130,168 |
| **Read_Batch** | Tiny | 1 | - | 815 | 1,226,009 | 990 | 21 |
| **Read_Batch** | Small | 1 | - | 1,653 | 604,903 | 989 | 21 |
| **Read_Batch** | Medium | 1 | - | 2,400 | 512,384 | 988 | 21 |
| **Projection** | Tiny | 1 | - | 11,700 | 85,401 | 2,035 | 37 |
| **Projection** | Small | 1 | - | 12,100 | 82,591 | 2,036 | 37 |
| **Projection** | Medium | 1 | - | 12,100 | 82,558 | 2,036 | 37 |

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
