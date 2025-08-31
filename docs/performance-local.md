# Local PostgreSQL Performance

## Append Performance

**Append Operations Details**:
- **Operation**: Simple event append operations
- **Scenario**: Basic event writing without conditions or business logic
- **Events**: Single event (1) or realistic batch (1-12 events)
- **Model**: Generic test events with simple JSON data

| Dataset | Concurrency | Events | Throughput (ops/sec) | Latency (ns/op) | Memory (B/op) | Allocations |
|---------|-------------|--------|---------------------|-----------------|---------------|-------------|
| Tiny | 1 | 1 | 4,867 | 205,321 | 1,384 | 44 |
| Small | 1 | 1 | 4,760 | 210,104 | 1,384 | 44 |
| Tiny | 1 | 1-12 | 3,210 | 311,863 | 11,230 | 162 |
| Small | 1 | 1-12 | 3,440 | 290,598 | 11,232 | 162 |
| Medium | 1 | 1-12 | 3,310 | 302,039 | 11,224 | 162 |
| Tiny | 1 | 1 | 4,245 | 235,601 | 1,884 | 56 |
| Small | 1 | 1 | 3,821 | 261,668 | 1,888 | 56 |
| Medium | 1 | 1 | 4,199 | 238,131 | 1,882 | 56 |
| Tiny | 10 | 1 | 1,166 | 857,995 | 17,559 | 523 |
| Small | 10 | 1 | 1,160 | 861,590 | 17,554 | 523 |
| Medium | 10 | 1 | 1,206 | 829,250 | 17,548 | 523 |
| Tiny | 100 | 1 | 142 | 7,042,692 | 183,156 | 5,285 |
| Small | 100 | 1 | 147 | 6,808,841 | 182,705 | 5,275 |
| Medium | 100 | 1 | 130 | 7,717,958 | 182,656 | 5,277 |
| Tiny | 1 | 100 | 476 | 2,098,884 | 211,665 | 2,054 |
| Small | 1 | 100 | 522 | 1,914,980 | 211,276 | 2,053 |
| Medium | 1 | 100 | 678 | 1,474,140 | 211,359 | 2,053 |
| Tiny | 10 | 100 | 92 | 10,822,192 | 2,097,196 | 20,508 |
| Small | 10 | 100 | 98 | 10,233,074 | 2,095,527 | 20,500 |
| Medium | 10 | 100 | 101 | 9,910,265 | 2,094,603 | 20,491 |
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
| Tiny | 1 | 1 | 930 | 1,075,000 | 4,495 | 95 |
| Small | 1 | 1 | 669 | 1,495,000 | 4,488 | 95 |
| Medium | 1 | 1 | 1,432 | 698,000 | 4,476 | 95 |
| Tiny | 10 | 1 | 430 | 2,325,000 | 43,476 | 919 |
| Small | 10 | 1 | 1,066 | 938,000 | 43,475 | 922 |
| Medium | 10 | 1 | 608 | 1,645,000 | 43,448 | 920 |
| Tiny | 100 | 1 | 46 | 21,700,000 | 443,743 | 9,277 |
| Small | 100 | 1 | 86 | 11,600,000 | 441,366 | 9,265 |
| Medium | 100 | 1 | 58 | 17,200,000 | 441,418 | 9,264 |
| Tiny | 1 | 100 | 1,537 | 650,000 | 215,033 | 2,096 |
| Small | 1 | 100 | 730 | 1,370,000 | 213,939 | 2,093 |
| Medium | 1 | 100 | 772 | 1,295,000 | 213,828 | 2,092 |
| Tiny | 10 | 100 | 187 | 5,350,000 | 2,139,663 | 20,925 |
| Small | 10 | 100 | 176 | 5,680,000 | 2,136,595 | 20,905 |
| Medium | 10 | 100 | 186 | 5,380,000 | 2,135,081 | 20,893 |
| Tiny | 100 | 100 | 18 | 55,600,000 | 21,367,125 | 209,183 |
| Small | 100 | 100 | 24 | 41,700,000 | 21,366,958 | 209,105 |
| Medium | 100 | 100 | 25 | 40,000,000 | 21,361,626 | 209,068 |

## AppendIf Performance (With Conflict)

**AppendIf With Conflict Details**:
- **Attempted Events**: Number of events AppendIf operation tries to append (1 or 100 events per operation)
- **Actual Events**: Number of events successfully appended (0 - all operations fail due to conflicts)
- **Past Events**: Number of existing events in database before benchmark (100 events for all scenarios)
- **Conflict Events**: Number of conflicting events created before AppendIf (1, 10, or 100 events, matching concurrency level)

| Dataset | Concurrency | Attempted Events | Conflict Events | Throughput (ops/sec) | Latency (ns/op) | Memory (B/op) | Allocations |
|---------|-------------|------------------|-----------------|---------------------|-----------------|---------------|-------------|
| Tiny | 1 | 1 | 1 | 100 | 10,000,000 | 5,909 | 144 |
| Small | 1 | 1 | 1 | 100 | 10,000,000 | 5,880 | 144 |
| Medium | 1 | 1 | 1 | 169 | 5,920,000 | 5,870 | 144 |
| Tiny | 10 | 1 | 10 | 16 | 62,500,000 | 57,906 | 1,411 |
| Small | 10 | 1 | 10 | 109 | 9,170,000 | 57,240 | 1,405 |
| Medium | 10 | 1 | 10 | 15 | 66,700,000 | 57,918 | 1,410 |
| Tiny | 100 | 1 | 100 | 13 | 76,900,000 | 585,352 | 14,188 |
| Small | 100 | 1 | 100 | 26 | 38,500,000 | 581,756 | 14,175 |
| Medium | 100 | 1 | 100 | 13 | 76,900,000 | 584,568 | 14,176 |
| Tiny | 1 | 100 | 1 | 16 | 62,500,000 | 213,810 | 2,143 |
| Small | 1 | 100 | 1 | 18 | 55,600,000 | 213,323 | 2,141 |
| Medium | 1 | 100 | 1 | 139 | 7,190,000 | 214,816 | 2,140 |
| Tiny | 10 | 100 | 10 | 16 | 62,500,000 | 2,133,544 | 21,400 |
| Small | 10 | 100 | 10 | 18 | 55,600,000 | 2,132,702 | 21,380 |
| Medium | 10 | 100 | 10 | 105 | 9,520,000 | 2,146,011 | 21,371 |
| Tiny | 100 | 100 | 100 | 10 | 100,000,000 | 21,473,610 | 213,918 |
| Small | 100 | 100 | 100 | 8 | 125,000,000 | 21,465,429 | 213,849 |
| Medium | 100 | 100 | 100 | 19 | 52,600,000 | 21,492,126 | 213,877 |

## Read and Projection Performance

| Operation | Dataset | Concurrency | Events | Throughput (ops/sec) | Latency (ns/op) | Memory (B/op) | Allocations |
|-----------|---------|-------------|--------|---------------------|-----------------|---------------|-------------|
| **Read_Single** | Tiny | 1 | - | 123 | 8,130,000 | 2,106,756 | 253,425 |
| **Read_Single** | Small | 1 | - | 294 | 3,400,000 | 1,024,370 | 131,363 |
| **Read_Single** | Medium | 1 | - | 328 | 3,050,000 | 1,024,348 | 131,363 |
| **Read_Batch** | Tiny | 1 | - | 49,030 | 20,400 | 988 | 21 |
| **Read_Batch** | Small | 1 | - | 6,724 | 148,800 | 989 | 21 |
| **Read_Batch** | Medium | 1 | - | 7,898 | 126,600 | 989 | 21 |
| **Projection** | Tiny | 1 | - | 36,338 | 27,500 | 2,037 | 37 |
| **Projection** | Small | 1 | - | 33,769 | 29,600 | 2,036 | 37 |
| **Projection** | Medium | 1 | - | 6,811 | 146,800 | 2,036 | 37 |

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
