# Performance Guide

> **🚀 Performance Update**: Recent benchmark improvements show significantly better AppendIf performance (124 ops/sec vs previous 0.08 ops/sec) after fixing database event accumulation issues. Results now reflect realistic business rule validation overhead. **New dataset configuration enables 100-user concurrent benchmarks to complete successfully.**

## Test Environment
- **Platform**: macOS (darwin 23.6.0) with Apple M1 Pro
- **Database**: PostgreSQL with 50-connection pool
- **Test Data**: Runtime-generated datasets with controlled past event counts

## Dataset-Specific Performance Results

Choose your dataset size to view detailed performance metrics:

### **📊 [Tiny Dataset Performance](./performance-tiny.md)**
- **Size**: 5 courses, 10 students, 20 enrollments
- **Use Case**: Quick testing, development, fast feedback
- **Past Events**: 10 events for AppendIf testing
- **Performance**: Best case scenarios, minimal data volume

### **📊 [Small Dataset Performance](./performance-small.md)**
- **Size**: 500 courses, 5,000 students, 25,000 enrollments  
- **Use Case**: High concurrency testing, balanced performance
- **Past Events**: 100 events for AppendIf testing
- **Performance**: Optimized for concurrent user scenarios

### **📊 [Medium Dataset Performance](./performance-medium.md)**
- **Size**: 1,000 courses, 10,000 students, 50,000 enrollments  
- **Use Case**: Production planning, scalability analysis
- **Past Events**: 100 events for AppendIf testing
- **Performance**: Real-world scenarios, maximum data volume

## Performance Summary

**Key Performance Insights**:
- **Append**: 2,000+ ops/sec (single event), scales well with concurrency
- **AppendIf**: 15-124 ops/sec depending on dataset size and conflict scenarios
- **Read**: 400-5,000+ ops/sec depending on query complexity and data volume
- **Projection**: 100-700 ops/sec for state reconstruction from event streams

**Dataset Impact**:
- **Tiny Dataset**: Best performance, minimal resource usage, ideal for development
- **Small Dataset**: Balanced performance, optimized for high concurrency testing
- **Medium Dataset**: Realistic performance, shows data volume impact, production planning

**Concurrency Scaling**: All operations tested with 1, 10, and 100 concurrent users to measure performance degradation under load. **Verified Results**: 10 users (2.7ms/op), 100 users (14.1ms/op) - both completing successfully.

**For detailed performance tables and specific metrics, see the dataset-specific pages above.**

## Operation Types Explained

### **Simple Read vs Complex Queries**

The performance tables show two different types of read operations:

#### **Simple Read**
- **What it tests**: Single query operations with basic tag filtering
- **Example**: `Query events with tag "user_id" = "123"`
- **Use case**: Basic event retrieval, simple filtering
- **Performance**: Fastest read operations

#### **Complex Queries** 
- **What it tests**: Multi-step business workflows with sequential queries
- **Example**: Complete course enrollment process:
  1. Query if student exists (`StudentRegistered` event)
  2. Query if course exists (`CourseDefined` event)
  3. Query if student is already enrolled (`StudentEnrolledInCourse` event)
  4. Append enrollment event
- **Use case**: Business rule validation, multi-step workflows, real-world scenarios
- **Performance**: Slower than simple reads due to multiple sequential operations

#### **Why Performance Differs Between Datasets**

| Dataset | Simple Read | Complex Queries | Performance Pattern |
|---------|-------------|-----------------|-------------------|
| **Tiny** | 3,649 ops/sec | 2,058 ops/sec | Complex queries are 1.8x slower than Simple Read |
| **Small** | 337 ops/sec | 361 ops/sec | Complex queries are 1.1x slower than Simple Read |

**Tiny Dataset**: Complex queries are slower because they perform 4 sequential operations, and the overhead of multiple queries is more significant with minimal data.

**Small Dataset**: Complex queries are slower (1.1x) because they still perform 4 sequential operations, but the performance difference is smaller due to:
- **Better query optimization** with more data
- **Improved indexing efficiency** 
- **Reduced per-operation overhead** at scale
- **Real-world data patterns** that optimize better

### **Append vs AppendIf**

#### **Append**
- **What it tests**: Simple event storage without business rule validation
- **Use case**: High-volume event streaming, event-driven architectures
- **Performance**: Fastest operation (2,000+ ops/sec)

#### **AppendIf**
- **What it tests**: Conditional event storage with business rule validation
- **Use case**: Event sourcing with business consistency, preventing duplicate operations
- **Performance**: Significantly slower (15-124 ops/sec) due to:
  - Business rule validation queries
  - Conflict detection logic
  - Past event scanning for conditions

**Performance Impact**: AppendIf is 8-147x slower than Append, but provides business consistency guarantees that simple Append cannot.

## Dataset Comparison

| Metric | Tiny Dataset | Small Dataset | Medium Dataset | Ratio (Tiny→Medium) |
|--------|--------------|---------------|----------------|-------------------|
| **Courses** | 5 | 500 | 1,000 | 200x |
| **Students** | 10 | 5,000 | 10,000 | 1,000x |
| **Enrollments** | 20 | 25,000 | 50,000 | 2,500x |
| **Append Performance** | 2,124 ops/sec | 2,200 ops/sec | 2,100 ops/sec | 1.01x |
| **AppendIf Performance** | 124 ops/sec | 30 ops/sec | 15 ops/sec | 8.3x slower |
| **Memory Usage** | 1.4KB | 1.1MB | 2.2MB | 1,571x |

**Key Insights**:
- **Append operations** are nearly identical in performance across all datasets
- **AppendIf operations** are significantly slower with larger datasets (8.3x slower)
- **Memory usage** scales dramatically with data volume (1,571x increase)
- **Tiny dataset** provides best-case performance for development and testing
- **Small dataset** optimized for high concurrency testing (100+ users)
- **Medium dataset** shows realistic production performance expectations
