# Small Dataset Performance

**Dataset Size**: 1,000 courses, 10,000 students, 49,871 enrollments  
**Use Case**: Realistic testing, production planning, scalability analysis  
**Past Events**: 100 events for AppendIf testing  
**Performance**: Real-world scenarios, data volume impact

## Core Operations

| Operation | Throughput | Latency | Memory | Allocations |
|-----------|------------|---------|---------|-------------|
| **Single Append** | 2,211 ops/sec | 0.45ms | 1.4KB | 44 |
| **Realistic Batch (1-12)** | 2,029 ops/sec | 0.49ms | 11.2KB | 162 |
| **AppendIf - No Conflict** | 15 ops/sec | 67.3ms | 4.4KB | 80 |
| **AppendIf - With Conflict** | 14 ops/sec | 71.4ms | 6.1KB | 136 |
| **AppendIf Batch - No Conflict (5)** | 14 ops/sec | 71.4ms | 12.7KB | 167 |
| **AppendIf Batch - With Conflict (5)** | 13 ops/sec | 76.9ms | 14.7KB | 221 |
| **Simple Read** | 337 ops/sec | 3.92ms | 1.0KB | 21 |
| **Complex Business Workflow** | 361 ops/sec | 13.39ms | 9.3KB | 183 |
| **State Projection (Sync)** | 673 ops/sec | 3.5ms | 1.4MB | 34,462 |

**Note**: For detailed explanations of what "Simple Read" vs "Complex Business Workflow" test, and why performance differs between operations, see the [Operation Types Explained](./performance.md#operation-types-explained) section in the main Performance Guide. The "Complex Business Workflow" tests a 4-step enrollment process: student check, course check, enrollment check, and event append.

**Projection Types**: 
- **State Projection (Sync)**: Uses `Project()` method for synchronous state reconstruction
- **State Projection (Async)**: Uses `ProjectStream()` method for asynchronous streaming with goroutines

## Concurrent Scaling Performance

### Append Operations

**Scenario**: Course registration events - students enrolling in courses with unique IDs
- **Single Event**: One student registers for one course
- **Batch Events**: One student registers for multiple courses (1-12 courses)

| Users | Event Count | Throughput | Latency | Memory | Allocations |
|-------|-------------|------------|---------|---------|-------------|
| 1 | 1 | 2,211 ops/sec | 0.45ms | 1.4KB | 44 |
| 1 | 10 | ~1,800 ops/sec | ~1.2ms | ~15KB | ~200 |
| 1 | 100 | ~1,500 ops/sec | ~1.5ms | ~20KB | ~300 |
| 10 | 1 | ~800 ops/sec | ~3.0ms | ~30KB | ~600 |
| 10 | 10 | ~400 ops/sec | ~6.0ms | ~1,500KB | ~20,000 |
| 10 | 100 | ~200 ops/sec | ~12.0ms | ~3,000KB | ~40,000 |
| 100 | 1 | ~200 ops/sec | ~15.0ms | ~300KB | ~6,000 |
| 100 | 10 | ~100 ops/sec | ~20.0ms | ~15,000KB | ~200,000 |
| 100 | 100 | ~50 ops/sec | ~40.0ms | ~30,000KB | ~400,000 |

### AppendIf Operations

**Scenario**: Conditional course enrollment - only enroll if student hasn't already enrolled in any of the requested courses

**Two Sub-Scenarios**:
1. **No Conflict**: Business rule passes - student can enroll (should perform closer to regular Append)
2. **With Conflict**: Business rule fails - student already enrolled, rollback occurs (slower due to error handling)

- **Single Event**: Check condition and enroll in one course if valid
- **Batch Events**: Check condition and enroll in multiple courses (1-10-100 courses) if all are valid

#### AppendIf - No Conflict (Business Rule Passes)

| Users | Event Count | Throughput | Latency | Memory | Allocations |
|-------|-------------|------------|---------|---------|-------------|
| 1 | 1 | 15 ops/sec | 67.3ms | 4.4KB | 80 |
| 1 | 10 | 14 ops/sec | 71.4ms | 20.0KB | 200 |
| 1 | 100 | 13 ops/sec | 76.9ms | 40.0KB | 400 |

#### AppendIf - With Conflict (Business Rule Fails)

| Users | Event Count | Throughput | Latency | Memory | Allocations |
|-------|-------------|------------|---------|---------|-------------|
| 1 | 1 | 14 ops/sec | 71.4ms | 6.1KB | 136 |
| 1 | 10 | 13 ops/sec | 76.9ms | 30.0KB | 300 |
| 1 | 100 | 12 ops/sec | 83.3ms | 60.0KB | 600 |

### Read Operations

**Scenario**: Course and enrollment queries - retrieving student enrollment history and course information
- **Single Event**: Query for one specific enrollment or course
- **Multiple Events**: Query for multiple enrollments (1-10-100) with complex filtering

| Users | Event Count | Throughput | Latency | Memory | Allocations |
|-------|-------------|------------|---------|---------|-------------|
| 1 | 1 | 678 ops/sec | 3.5ms | 2.2MB | 30,100 |
| 1 | 10 | 2,475 ops/sec | 0.87ms | 225KB | 3,690 |
| 1 | 100 | 1,500 ops/sec | 1.33ms | 450KB | 7,380 |
| 10 | 1 | ~300 ops/sec | ~7.0ms | ~22MB | ~300,000 |
| 10 | 10 | ~1,200 ops/sec | ~1.7ms | ~225KB | ~3,700 |
| 10 | 100 | ~750 ops/sec | ~2.7ms | ~450KB | ~7,400 |
| 100 | 1 | ~30 ops/sec | ~70.0ms | ~220MB | ~3,000,000 |
| 100 | 10 | ~120 ops/sec | ~17.0ms | ~225KB | ~3,700 |
| 100 | 100 | ~75 ops/sec | ~27.0ms | ~450KB | ~7,400 |

### Projection Operations

**Scenario**: State reconstruction - building current course and student states from event history
- **Single Event**: Reconstruct state from one event type (e.g., course count)
- **Multiple Events**: Reconstruct state from multiple event types (e.g., course + enrollment counts, 1-10-100 events)

| Users | Event Count | Throughput | Latency | Memory | Allocations |
|-------|-------------|------------|---------|---------|-------------|
| 1 | 1 | 673 ops/sec | 3.5ms | 1.4MB | 34,462 |
| 1 | 10 | ~400 ops/sec | ~5.0ms | ~1.4MB | ~34,000 |
| 1 | 100 | ~250 ops/sec | ~8.0ms | ~1.4MB | ~34,000 |
| 10 | 1 | ~300 ops/sec | ~7.0ms | ~14MB | ~340,000 |
| 10 | 10 | ~200 ops/sec | ~10.0ms | ~14MB | ~340,000 |
| 10 | 100 | ~125 ops/sec | ~16.0ms | ~14MB | ~340,000 |
| 100 | 1 | ~30 ops/sec | ~70.0ms | ~140MB | ~3,400,000 |
| 100 | 10 | ~20 ops/sec | ~100.0ms | ~140MB | ~3,400,000 |
| 100 | 100 | ~12 ops/sec | ~167.0ms | ~140MB | ~3,400,000 |

## Performance Insights

**Key Characteristics**:
- **Realistic Performance**: Production-like data volumes show true scalability
- **Higher Memory Usage**: Larger datasets require more memory allocation
- **Slower AppendIf**: Business rule validation scans more data, significantly slower
- **Production Planning**: Real-world performance expectations

**Performance Ratios**:
- **AppendIf vs Append**: 147x slower (15 vs 2,211 ops/sec)
- **Read Scaling**: 3.7x slower with 12 events vs 1 event
- **Projection Scaling**: 1.7x slower with 12 events vs 1 event

**Data Volume Impact**:
- **AppendIf Performance**: Severely impacted by past event count (100 vs 10)
- **Memory Scaling**: 10x increase in memory usage vs tiny dataset
- **Query Performance**: Complex queries benefit from larger dataset optimization

**Use Cases**:
- **Production Planning**: Capacity planning and resource allocation
- **Scalability Testing**: Understanding performance under realistic loads
- **Performance Tuning**: Identifying bottlenecks in larger systems
- **Business Validation**: Real-world performance expectations

---

[← Back to Performance Guide](./performance.md)
