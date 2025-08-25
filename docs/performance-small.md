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
| **Simple Read** | 678 ops/sec | 3.5ms | 2.2MB | 30,100 |
| **Complex Queries** | 5,179 ops/sec | 0.44ms | 1.0KB | 21 |
| **State Projection** | 673 ops/sec | 3.5ms | 1.4MB | 34,462 |

## Concurrent Scaling Performance

### Append Operations

**Scenario**: Course registration events - students enrolling in courses with unique IDs
- **Single Event**: One student registers for one course
- **Batch Events**: One student registers for multiple courses (1-12 courses)

| Users | Event Count | Throughput | Latency | Memory | Allocations |
|-------|-------------|------------|---------|---------|-------------|
| 1 | 1 | 2,211 ops/sec | 0.45ms | 1.4KB | 44 |
| 1 | 5 | 2,029 ops/sec | 0.49ms | 11.2KB | 162 |
| 1 | 12 | ~1,800 ops/sec | ~1.2ms | ~15KB | ~200 |
| 10 | 1 | ~800 ops/sec | ~3.0ms | ~30KB | ~600 |
| 10 | 5 | ~600 ops/sec | ~4.0ms | ~200KB | ~3,000 |
| 10 | 12 | ~400 ops/sec | ~6.0ms | ~1,500KB | ~20,000 |
| 100 | 1 | ~200 ops/sec | ~15.0ms | ~300KB | ~6,000 |
| 100 | 5 | ~150 ops/sec | ~20.0ms | ~2,000KB | ~30,000 |
| 100 | 12 | ~100 ops/sec | ~30.0ms | ~15,000KB | ~200,000 |

### AppendIf Operations

**Scenario**: Conditional course enrollment - only enroll if student hasn't already enrolled in any of the requested courses

**Two Sub-Scenarios**:
1. **No Conflict**: Business rule passes - student can enroll (should perform closer to regular Append)
2. **With Conflict**: Business rule fails - student already enrolled, rollback occurs (slower due to error handling)

- **Single Event**: Check condition and enroll in one course if valid
- **Batch Events**: Check condition and enroll in multiple courses (1-12 courses) if all are valid

#### AppendIf - No Conflict (Business Rule Passes)

| Users | Event Count | Throughput | Latency | Memory | Allocations |
|-------|-------------|------------|---------|---------|-------------|
| 1 | 1 | 15 ops/sec | 67.3ms | 4.4KB | 80 |
| 1 | 5 | 14 ops/sec | 71.4ms | 12.7KB | 167 |
| 1 | 12 | 14 ops/sec | 71.4ms | 22.5KB | 309 |

#### AppendIf - With Conflict (Business Rule Fails)

| Users | Event Count | Throughput | Latency | Memory | Allocations |
|-------|-------------|------------|---------|---------|-------------|
| 1 | 1 | 14 ops/sec | 71.4ms | 6.1KB | 136 |
| 1 | 5 | 13 ops/sec | 76.9ms | 14.7KB | 221 |
| 1 | 12 | 13 ops/sec | 76.9ms | 29.1KB | 364 |

### Read Operations

**Scenario**: Course and enrollment queries - retrieving student enrollment history and course information
- **Single Event**: Query for one specific enrollment or course
- **Multiple Events**: Query for multiple enrollments (1-12) with complex filtering

| Users | Event Count | Throughput | Latency | Memory | Allocations |
|-------|-------------|------------|---------|---------|-------------|
| 1 | 1 | 678 ops/sec | 3.5ms | 2.2MB | 30,100 |
| 1 | 5 | 5,179 ops/sec | 0.44ms | 1.0KB | 21 |
| 1 | 12 | 2,475 ops/sec | 0.87ms | 225KB | 3,690 |
| 10 | 1 | ~300 ops/sec | ~7.0ms | ~22MB | ~300,000 |
| 10 | 5 | ~2,500 ops/sec | ~0.8ms | ~10KB | ~200 |
| 10 | 12 | ~1,200 ops/sec | ~1.7ms | ~225KB | ~3,700 |
| 100 | 1 | ~30 ops/sec | ~70.0ms | ~220MB | ~3,000,000 |
| 100 | 5 | ~250 ops/sec | ~8.0ms | ~10KB | ~200 |
| 100 | 12 | ~120 ops/sec | ~17.0ms | ~225KB | ~3,700 |

### Projection Operations

**Scenario**: State reconstruction - building current course and student states from event history
- **Single Event**: Reconstruct state from one event type (e.g., course count)
- **Multiple Events**: Reconstruct state from multiple event types (e.g., course + enrollment counts, 1-12 events)

| Users | Event Count | Throughput | Latency | Memory | Allocations |
|-------|-------------|------------|---------|---------|-------------|
| 1 | 1 | 673 ops/sec | 3.5ms | 1.4MB | 34,462 |
| 1 | 2 | 672 ops/sec | 3.5ms | 1.4MB | 34,476 |
| 1 | 5 | ~500 ops/sec | ~4.0ms | ~1.4MB | ~34,000 |
| 1 | 12 | ~400 ops/sec | ~5.0ms | ~1.4MB | ~34,000 |
| 10 | 1 | ~300 ops/sec | ~7.0ms | ~14MB | ~340,000 |
| 10 | 2 | ~300 ops/sec | ~7.0ms | ~14MB | ~340,000 |
| 10 | 5 | ~250 ops/sec | ~8.0ms | ~14MB | ~340,000 |
| 10 | 12 | ~200 ops/sec | ~10.0ms | ~14MB | ~340,000 |
| 100 | 1 | ~30 ops/sec | ~70.0ms | ~140MB | ~3,400,000 |
| 100 | 2 | ~30 ops/sec | ~70.0ms | ~140MB | ~3,400,000 |
| 100 | 5 | ~25 ops/sec | ~80.0ms | ~140MB | ~3,400,000 |
| 100 | 12 | ~20 ops/sec | ~100.0ms | ~140MB | ~3,400,000 |

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

## Dataset Comparison

| Metric | Tiny Dataset | Small Dataset | Ratio |
|--------|--------------|---------------|-------|
| **Courses** | 5 | 1,000 | 200x |
| **Students** | 10 | 10,000 | 1,000x |
| **Enrollments** | 17 | 49,871 | 2,933x |
| **Append Performance** | 2,124 ops/sec | 2,211 ops/sec | 1.04x |
| **AppendIf Performance** | 124 ops/sec | 15 ops/sec | 8.3x slower |
| **Memory Usage** | 1.4KB | 2.2MB | 1,571x |

---

[← Back to Performance Guide](./performance.md)
