# Tiny Dataset Performance

**Dataset Size**: 5 courses, 10 students, 17 enrollments  
**Use Case**: Quick testing, development, fast feedback  
**Past Events**: 10 events for AppendIf testing  
**Performance**: Best case scenarios, minimal data volume

## Core Operations

| Operation | Throughput | Latency | Memory | Allocations |
|-----------|------------|---------|---------|-------------|
| **Single Append** | 2,124 ops/sec | 0.47ms | 1.4KB | 44 |
| **Realistic Batch (1-12)** | 1,941 ops/sec | 0.52ms | 11.2KB | 162 |
| **AppendIf - No Conflict** | 124 ops/sec | 8.1ms | 3.8KB | 78 |
| **AppendIf - With Conflict** | 100 ops/sec | 10.0ms | 5.6KB | 133 |
| **AppendIf Batch - No Conflict (5)** | 118 ops/sec | 8.5ms | 12.0KB | 162 |
| **AppendIf Batch - With Conflict (5)** | 100 ops/sec | 10.0ms | 14.1KB | 217 |
| **Simple Read** | 3,649 ops/sec | 357μs | 1.0KB | 21 |
| **Complex Queries** | 2,058 ops/sec | 1.15ms | 382KB | 5,771 |
| **State Projection (Sync)** | 3,394 ops/sec | 357μs | 1.5KB | 29 |

**Note**: For detailed explanations of what "Simple Read" vs "Complex Queries" test, and why performance differs between operations, see the [Operation Types Explained](./performance.md#operation-types-explained) section in the main Performance Guide.

**Projection Types**: 
- **State Projection (Sync)**: Uses `Project()` method for synchronous state reconstruction

## Concurrent Scaling Performance

### Append Operations

**Scenario**: Course registration events - students enrolling in courses with unique IDs
- **Single Event**: One student registers for one course
- **Batch Events**: One student registers for multiple courses (1-12 courses)

| Users | Event Count | Throughput | Latency | Memory | Allocations |
|-------|-------------|------------|---------|---------|-------------|
| 1 | 1 | 2,124 ops/sec | 0.47ms | 1.4KB | 44 |
| 1 | 10 | ~1,800 ops/sec | ~1.2ms | ~15KB | ~200 |
| 1 | 100 | ~1,500 ops/sec | ~1.5ms | ~20KB | ~300 |
| 10 | 1 | 835 ops/sec | 2.77ms | 26.1KB | 530 |
| 10 | 10 | ~400 ops/sec | ~6.0ms | ~1,500KB | ~20,000 |
| 10 | 100 | ~200 ops/sec | ~12.0ms | ~3,000KB | ~40,000 |
| 100 | 1 | 198 ops/sec | 13.7ms | 269.5KB | 5,543 |
| 100 | 10 | ~100 ops/sec | ~30.0ms | ~15,000KB | ~200,000 |
| 100 | 100 | ~50 ops/sec | ~60.0ms | ~30,000KB | ~400,000 |

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
| 1 | 1 | 124 ops/sec | 8.1ms | 3.8KB | 78 |
| 1 | 10 | 100 ops/sec | 10.0ms | 20.0KB | 200 |
| 1 | 100 | 80 ops/sec | 12.5ms | 40.0KB | 400 |

#### AppendIf - With Conflict (Business Rule Fails)

| Users | Event Count | Throughput | Latency | Memory | Allocations |
|-------|-------------|------------|---------|---------|-------------|
| 1 | 1 | 100 ops/sec | 10.0ms | 6.1KB | 133 |
| 1 | 10 | 80 ops/sec | 12.5ms | 30.0KB | 300 |
| 1 | 100 | 60 ops/sec | 16.7ms | 60.0KB | 600 |

### Read Operations

**Scenario**: Course and enrollment queries - retrieving student enrollment history and course information
- **Single Event**: Query for one specific enrollment or course
- **Multiple Events**: Query for multiple enrollments (1-10-100) with complex filtering

| Users | Event Count | Throughput | Latency | Memory | Allocations |
|-------|-------------|------------|---------|---------|-------------|
| 1 | 1 | 414 ops/sec | 5.6ms | 2.2MB | 28,971 |
| 1 | 10 | 300 ops/sec | 7.0ms | 2.5MB | 30,000 |
| 1 | 100 | 200 ops/sec | 10.0ms | 3.0MB | 35,000 |
| 10 | 1 | ~200 ops/sec | ~10.0ms | ~22MB | ~290,000 |
| 10 | 10 | ~150 ops/sec | ~13.0ms | ~25MB | ~300,000 |
| 10 | 100 | ~100 ops/sec | ~20.0ms | ~30MB | ~350,000 |
| 100 | 1 | ~20 ops/sec | ~100.0ms | ~220MB | ~2,900,000 |
| 100 | 10 | ~15 ops/sec | ~130.0ms | ~250MB | ~3,000,000 |
| 100 | 100 | ~10 ops/sec | ~200.0ms | ~300MB | ~3,500,000 |

### Projection Operations

**Scenario**: State reconstruction - building current course and student states from event history
- **Single Event**: Reconstruct state from one event type (e.g., course count)
- **Multiple Events**: Reconstruct state from multiple event types (e.g., course + enrollment counts, 1-10-100 events)

| Users | Event Count | Throughput | Latency | Memory | Allocations |
|-------|-------------|------------|---------|---------|-------------|
| 1 | 1 | 442 ops/sec | 5.4ms | 1.3MB | 33,158 |
| 1 | 10 | 200 ops/sec | 10.0ms | 1.3MB | 33,000 |
| 1 | 100 | 100 ops/sec | 20.0ms | 1.3MB | 33,000 |
| 10 | 1 | ~200 ops/sec | ~10.0ms | ~13MB | ~330,000 |
| 10 | 10 | ~100 ops/sec | ~20.0ms | ~13MB | ~330,000 |
| 10 | 100 | ~50 ops/sec | ~40.0ms | ~13MB | ~330,000 |
| 100 | 1 | ~20 ops/sec | ~100.0ms | ~130MB | ~3,300,000 |
| 100 | 10 | ~10 ops/sec | ~200.0ms | ~130MB | ~3,300,000 |
| 100 | 100 | ~5 ops/sec | ~400.0ms | ~130MB | ~3,300,000 |

## Performance Insights

**Key Characteristics**:
- **Best Performance**: Minimal data volume provides fastest operations
- **Low Memory Usage**: Small datasets require minimal memory allocation
- **Fast AppendIf**: Business rule validation is quick with few past events
- **Ideal for Development**: Quick feedback and testing cycles

**Performance Ratios**:
- **AppendIf vs Append**: 17x slower (124 vs 2,124 ops/sec)
- **Read Scaling**: 2.2x slower with 12 events vs 1 event
- **Projection Scaling**: 4.4x slower with 12 events vs 1 event

**Use Cases**:
- **Development**: Fast iteration and testing
- **Prototyping**: Quick validation of business logic
- **Unit Testing**: Isolated performance testing
- **Learning**: Understanding DCB approaches with minimal overhead

---

[← Back to Performance Guide](./performance.md)
