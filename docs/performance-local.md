# Local PostgreSQL Performance

## Append Performance

**Append Operations Details**:
- **Operation**: Simple event append operations
- **Scenario**: Basic event writing without conditions or business logic
- **Events**: Single event (1) or realistic batch (1-12 events)
- **Model**: Generic test events with simple JSON data

| Operation | Dataset | Concurrency | Events | Throughput (ops/sec) | Latency (ns/op) | Memory (B/op) | Allocations |
|-----------|---------|-------------|--------|---------------------|-----------------|---------------|-------------|
| **Append** | Tiny | 1 | 1 | 4,867 | 205,321 | 1,384 | 44 |
| **Append** | Small | 1 | 1 | 4,760 | 210,104 | 1,384 | 44 |
| **Append** | Tiny | 1 | 1-12 | 3,210 | 311,863 | 11,230 | 162 |
| **Append** | Small | 1 | 1-12 | 3,440 | 290,598 | 11,232 | 162 |
| **Append** | Medium | 1 | 1-12 | 3,310 | 302,039 | 11,224 | 162 |

## AppendIf Performance (No Conflict)

**AppendIf No Conflict Details**:
- **Attempted Events**: Number of events AppendIf operation tries to append (1 or 100 events per operation)
- **Actual Events**: Number of events successfully appended (1 or 100 events)
- **Past Events**: Number of existing events in database before benchmark (100 events for all scenarios)
- **Conflict Events**: 0 (no conflicts exist)

| Operation | Dataset | Concurrency | Attempted Events | Throughput (ops/sec) | Latency (ns/op) | Memory (B/op) | Allocations |
|-----------|---------|-------------|------------------|---------------------|-----------------|---------------|-------------|
| **AppendIf_NoConflict_Concurrent_1User_1Event** | Tiny | 1 | 1 | 930 | 1,075,000 | 4,495 | 95 |
| **AppendIf_NoConflict_Concurrent_1User_1Event** | Small | 1 | 1 | 669 | 1,495,000 | 4,488 | 95 |
| **AppendIf_NoConflict_Concurrent_1User_1Event** | Medium | 1 | 1 | 1,432 | 698,000 | 4,476 | 95 |
| **AppendIf_NoConflict_Concurrent_10Users_1Event** | Tiny | 10 | 1 | 430 | 2,325,000 | 43,476 | 919 |
| **AppendIf_NoConflict_Concurrent_10Users_1Event** | Small | 10 | 1 | 1,066 | 938,000 | 43,475 | 922 |
| **AppendIf_NoConflict_Concurrent_10Users_1Event** | Medium | 10 | 1 | 608 | 1,645,000 | 43,448 | 920 |
| **AppendIf_NoConflict_Concurrent_100Users_1Event** | Tiny | 100 | 1 | 46 | 21,700,000 | 443,743 | 9,277 |
| **AppendIf_NoConflict_Concurrent_100Users_1Event** | Small | 100 | 1 | 86 | 11,600,000 | 441,366 | 9,265 |
| **AppendIf_NoConflict_Concurrent_100Users_1Event** | Medium | 100 | 1 | 58 | 17,200,000 | 441,418 | 9,264 |
| **AppendIf_NoConflict_Concurrent_1User_100Events** | Tiny | 1 | 100 | 1,537 | 650,000 | 215,033 | 2,096 |
| **AppendIf_NoConflict_Concurrent_1User_100Events** | Small | 1 | 100 | 730 | 1,370,000 | 213,939 | 2,093 |
| **AppendIf_NoConflict_Concurrent_1User_100Events** | Medium | 1 | 100 | 772 | 1,295,000 | 213,828 | 2,092 |
| **AppendIf_NoConflict_Concurrent_10Users_100Events** | Tiny | 10 | 100 | 187 | 5,350,000 | 2,139,663 | 20,925 |
| **AppendIf_NoConflict_Concurrent_10Users_100Events** | Small | 10 | 100 | 176 | 5,680,000 | 2,136,595 | 20,905 |
| **AppendIf_NoConflict_Concurrent_10Users_100Events** | Medium | 10 | 100 | 186 | 5,380,000 | 2,135,081 | 20,893 |
| **AppendIf_NoConflict_Concurrent_100Users_100Events** | Tiny | 100 | 100 | 18 | 55,600,000 | 21,367,125 | 209,183 |
| **AppendIf_NoConflict_Concurrent_100Users_100Events** | Small | 100 | 100 | 24 | 41,700,000 | 21,366,958 | 209,105 |
| **AppendIf_NoConflict_Concurrent_100Users_100Events** | Medium | 100 | 100 | 25 | 40,000,000 | 21,361,626 | 209,068 |

## AppendIf Performance (With Conflict)

**AppendIf With Conflict Details**:
- **Attempted Events**: Number of events AppendIf operation tries to append (1 or 100 events per operation)
- **Actual Events**: Number of events successfully appended (0 - all operations fail due to conflicts)
- **Past Events**: Number of existing events in database before benchmark (100 events for all scenarios)
- **Conflict Events**: Number of conflicting events created before AppendIf (1, 10, or 100 events, matching concurrency level)

| Operation | Dataset | Concurrency | Attempted Events | Conflict Events | Throughput (ops/sec) | Latency (ns/op) | Memory (B/op) | Allocations |
|-----------|---------|-------------|------------------|-----------------|---------------------|-----------------|---------------|-------------|
| **AppendIf_WithConflict_Concurrent_1User_1Event** | Tiny | 1 | 1 | 1 | 100 | 10,000,000 | 5,909 | 144 |
| **AppendIf_WithConflict_Concurrent_1User_1Event** | Small | 1 | 1 | 1 | 100 | 10,000,000 | 5,880 | 144 |
| **AppendIf_WithConflict_Concurrent_1User_1Event** | Medium | 1 | 1 | 1 | 169 | 5,920,000 | 5,870 | 144 |
| **AppendIf_WithConflict_Concurrent_10Users_1Event** | Tiny | 10 | 1 | 10 | 16 | 62,500,000 | 57,906 | 1,411 |
| **AppendIf_WithConflict_Concurrent_10Users_1Event** | Small | 10 | 1 | 10 | 109 | 9,170,000 | 57,240 | 1,405 |
| **AppendIf_WithConflict_Concurrent_10Users_1Event** | Medium | 10 | 1 | 10 | 15 | 66,700,000 | 57,918 | 1,410 |
| **AppendIf_WithConflict_Concurrent_100Users_1Event** | Tiny | 100 | 1 | 100 | 13 | 76,900,000 | 585,352 | 14,188 |
| **AppendIf_WithConflict_Concurrent_100Users_1Event** | Small | 100 | 1 | 100 | 26 | 38,500,000 | 581,756 | 14,175 |
| **AppendIf_WithConflict_Concurrent_100Users_1Event** | Medium | 100 | 1 | 100 | 13 | 76,900,000 | 584,568 | 14,176 |
| **AppendIf_WithConflict_Concurrent_1User_100Events** | Tiny | 1 | 100 | 1 | 16 | 62,500,000 | 213,810 | 2,143 |
| **AppendIf_WithConflict_Concurrent_1User_100Events** | Small | 1 | 100 | 1 | 18 | 55,600,000 | 213,323 | 2,141 |
| **AppendIf_WithConflict_Concurrent_1User_100Events** | Medium | 1 | 100 | 1 | 139 | 7,190,000 | 214,816 | 2,140 |
| **AppendIf_WithConflict_Concurrent_10Users_100Events** | Tiny | 10 | 100 | 10 | 16 | 62,500,000 | 2,133,544 | 21,400 |
| **AppendIf_WithConflict_Concurrent_10Users_100Events** | Small | 10 | 100 | 10 | 18 | 55,600,000 | 2,132,702 | 21,380 |
| **AppendIf_WithConflict_Concurrent_10Users_100Events** | Medium | 10 | 100 | 10 | 105 | 9,520,000 | 2,146,011 | 21,371 |
| **AppendIf_WithConflict_Concurrent_100Users_100Events** | Tiny | 100 | 100 | 100 | 10 | 100,000,000 | 21,473,610 | 213,918 |
| **AppendIf_WithConflict_Concurrent_100Users_100Events** | Small | 100 | 100 | 100 | 8 | 125,000,000 | 21,465,429 | 213,849 |
| **AppendIf_WithConflict_Concurrent_100Users_100Events** | Medium | 100 | 100 | 100 | 19 | 52,600,000 | 21,492,126 | 213,877 |

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
