# Docker PostgreSQL Performance

## Performance Results

## Append Performance

**Append Operations Details**:
- **Operation**: Simple event append operations
- **Scenario**: Basic event writing without conditions or business logic
- **Events**: Single event (1) or realistic batch (1-12 events)
- **Model**: Generic test events with simple JSON data

| Operation | Dataset | Concurrency | Events | Throughput (ops/sec) | Latency (ns/op) | Memory (B/op) | Allocations |
|-----------|---------|-------------|--------|---------------------|-----------------|---------------|-------------|
| **Append** | Tiny | 1 | 1 | 4,640 | 215,546 | 1,384 | 44 |
| **Append** | Small | 1 | 1 | 4,830 | 207,512 | 1,383 | 44 |
| **Append** | Medium | 1 | 1 | 4,843 | 206,500 | 1,383 | 44 |
| **Append** | Tiny | 1 | 1-12 | 3,472 | 288,118 | 11,233 | 162 |
| **Append** | Small | 1 | 1-12 | 3,375 | 296,286 | 11,231 | 162 |
| **Append** | Medium | 1 | 1-12 | 3,311 | 302,123 | 11,223 | 162 |

## AppendIf Performance (No Conflict)

**AppendIf No Conflict Details**:
- **Attempted Events**: Number of events AppendIf operation tries to append (1 or 100 events per operation)
- **Actual Events**: Number of events successfully appended (1 or 100 events)
- **Past Events**: Number of existing events in database before benchmark (100 events for all scenarios)
- **Conflict Events**: 0 (no conflicts exist)

| Operation | Dataset | Concurrency | Attempted Events | Throughput (ops/sec) | Latency (ns/op) | Memory (B/op) | Allocations |
|-----------|---------|-------------|------------------|---------------------|-----------------|---------------|-------------|
| **AppendIf_NoConflict_Concurrent_1User_1Event** | Tiny | 1 | 1 | 631 | 1,584,000 | 4,505 | 95 |
| **AppendIf_NoConflict_Concurrent_1User_1Event** | Small | 1 | 1 | 631 | 1,584,000 | 4,505 | 95 |
| **AppendIf_NoConflict_Concurrent_1User_1Event** | Medium | 1 | 1 | 631 | 1,584,000 | 4,505 | 95 |
| **AppendIf_NoConflict_Concurrent_10Users_1Event** | Tiny | 10 | 1 | 1,483 | 674,000 | 43,466 | 923 |
| **AppendIf_NoConflict_Concurrent_10Users_1Event** | Small | 10 | 1 | 1,483 | 674,000 | 43,466 | 923 |
| **AppendIf_NoConflict_Concurrent_10Users_1Event** | Medium | 10 | 1 | 1,483 | 674,000 | 43,466 | 923 |
| **AppendIf_NoConflict_Concurrent_100Users_1Event** | Tiny | 100 | 1 | 61 | 16,400,000 | 441,655 | 9,268 |
| **AppendIf_NoConflict_Concurrent_100Users_1Event** | Small | 100 | 1 | 61 | 16,400,000 | 441,655 | 9,268 |
| **AppendIf_NoConflict_Concurrent_100Users_1Event** | Medium | 100 | 1 | 61 | 16,400,000 | 441,655 | 9,268 |
| **AppendIf_NoConflict_Concurrent_1User_100Events** | Tiny | 1 | 100 | 718 | 1,392,000 | 213,933 | 2,093 |
| **AppendIf_NoConflict_Concurrent_1User_100Events** | Small | 1 | 100 | 718 | 1,392,000 | 213,933 | 2,093 |
| **AppendIf_NoConflict_Concurrent_1User_100Events** | Medium | 1 | 100 | 718 | 1,392,000 | 213,933 | 2,093 |
| **AppendIf_NoConflict_Concurrent_10Users_100Events** | Tiny | 10 | 100 | 202 | 4,950,000 | 2,136,535 | 20,902 |
| **AppendIf_NoConflict_Concurrent_10Users_100Events** | Small | 10 | 100 | 202 | 4,950,000 | 2,136,535 | 20,902 |
| **AppendIf_NoConflict_Concurrent_10Users_100Events** | Medium | 10 | 100 | 202 | 4,950,000 | 2,136,535 | 20,902 |
| **AppendIf_NoConflict_Concurrent_100Users_100Events** | Tiny | 100 | 100 | 19 | 52,600,000 | 21,361,007 | 209,098 |
| **AppendIf_NoConflict_Concurrent_100Users_100Events** | Small | 100 | 100 | 19 | 52,600,000 | 21,361,007 | 209,098 |
| **AppendIf_NoConflict_Concurrent_100Users_100Events** | Medium | 100 | 100 | 19 | 52,600,000 | 21,361,007 | 209,098 |

## AppendIf Performance (With Conflict)

**AppendIf With Conflict Details**:
- **Attempted Events**: Number of events AppendIf operation tries to append (1 or 100 events per operation)
- **Actual Events**: Number of events successfully appended (0 - all operations fail due to conflicts)
- **Past Events**: Number of existing events in database before benchmark (100 events for all scenarios)
- **Conflict Events**: Number of conflicting events created before AppendIf (1, 10, or 100 events, matching concurrency level)

| Operation | Dataset | Concurrency | Attempted Events | Conflict Events | Throughput (ops/sec) | Latency (ns/op) | Memory (B/op) | Allocations |
|-----------|---------|-------------|------------------|-----------------|---------------------|-----------------|---------------|-------------|
| **AppendIf_WithConflict_Concurrent_1User_1Event** | Tiny | 1 | 1 | 1 | 177 | 5,650,000 | 5,885 | 144 |
| **AppendIf_WithConflict_Concurrent_1User_1Event** | Small | 1 | 1 | 1 | 170 | 5,880,000 | 5,870 | 144 |
| **AppendIf_WithConflict_Concurrent_1User_1Event** | Medium | 1 | 1 | 1 | 100 | 10,000,000 | 5,909 | 144 |
| **AppendIf_WithConflict_Concurrent_10Users_1Event** | Tiny | 10 | 1 | 10 | 106 | 9,430,000 | 57,260 | 1,405 |
| **AppendIf_WithConflict_Concurrent_10Users_1Event** | Small | 10 | 1 | 10 | 108 | 9,260,000 | 57,272 | 1,405 |
| **AppendIf_WithConflict_Concurrent_10Users_1Event** | Medium | 10 | 1 | 10 | 18 | 55,600,000 | 57,949 | 1,409 |
| **AppendIf_WithConflict_Concurrent_100Users_1Event** | Tiny | 100 | 1 | 100 | 24 | 41,700,000 | 581,917 | 14,183 |
| **AppendIf_WithConflict_Concurrent_100Users_1Event** | Small | 100 | 1 | 100 | 26 | 38,500,000 | 581,459 | 14,178 |
| **AppendIf_WithConflict_Concurrent_100Users_1Event** | Medium | 100 | 1 | 100 | 14 | 71,400,000 | 583,659 | 14,171 |
| **AppendIf_WithConflict_Concurrent_1User_100Events** | Tiny | 1 | 100 | 1 | 100 | 10,000,000 | 215,457 | 2,144 |
| **AppendIf_WithConflict_Concurrent_1User_100Events** | Small | 1 | 100 | 1 | 100 | 10,000,000 | 214,760 | 2,140 |
| **AppendIf_WithConflict_Concurrent_1User_100Events** | Medium | 1 | 100 | 1 | 18 | 55,600,000 | 213,399 | 2,140 |
| **AppendIf_WithConflict_Concurrent_10Users_100Events** | Tiny | 10 | 100 | 10 | 80 | 12,500,000 | 2,149,047 | 21,399 |
| **AppendIf_WithConflict_Concurrent_10Users_100Events** | Small | 10 | 100 | 10 | 82 | 12,200,000 | 2,146,121 | 21,379 |
| **AppendIf_WithConflict_Concurrent_10Users_100Events** | Medium | 10 | 100 | 10 | 18 | 55,600,000 | 2,131,168 | 21,370 |
| **AppendIf_WithConflict_Concurrent_100Users_100Events** | Tiny | 100 | 100 | 100 | 12 | 83,300,000 | 21,488,142 | 213,965 |
| **AppendIf_WithConflict_Concurrent_100Users_100Events** | Small | 100 | 100 | 100 | 12 | 83,300,000 | 21,482,203 | 213,947 |
| **AppendIf_WithConflict_Concurrent_100Users_100Events** | Medium | 100 | 100 | 100 | 12 | 83,300,000 | 21,467,655 | 213,808 |
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
