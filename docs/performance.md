# Performance Guide

> **🚀 Performance Overview**: Choose your PostgreSQL setup to view detailed performance metrics. **Local PostgreSQL shows 6-9x faster performance** than Docker PostgreSQL for development and testing scenarios.

## PostgreSQL Setup Options

### **🐳 [Docker PostgreSQL Performance](./performance-docker.md)**
- **Environment**: Docker containerized PostgreSQL
- **Use Case**: CI/CD, production-like testing, consistent environments
- **Performance**: Conservative, containerized performance baseline
- **Setup**: `docker-compose up -d`

### **💻 [Local PostgreSQL Performance](./performance-local.md)**
- **Environment**: Native PostgreSQL installation (macOS/Homebrew)
- **Use Case**: Development, high-performance testing, daily work
- **Performance**: 6-9x faster than Docker PostgreSQL
- **Setup**: Native PostgreSQL installation

## Quick Performance Comparison

| Operation | Docker PostgreSQL | Local PostgreSQL | Performance Gain |
|-----------|-------------------|------------------|------------------|
| **AppendSingle** | 850 ops/sec | 8,030 ops/sec | **9.4x faster** |
| **AppendRealistic** | 764 ops/sec | 4,825 ops/sec | **6.3x faster** |
| **AppendIf** | 10 ops/sec | 16 ops/sec | **1.7x faster** |
| **Projection** | 2,395 ops/sec | 16,834 ops/sec | **7x faster** |



## Performance Data Comparison

### **Local PostgreSQL Performance**
- **Append operations**: 8,030 ops/sec (Tiny), 7,900 ops/sec (Small)
- **AppendIf operations**: 16 ops/sec (Tiny), 11 ops/sec (Small)
- **Projection operations**: 16,834 ops/sec (Tiny), 14,340 ops/sec (Small)
- **Concurrency scaling**: 23.7x faster at 1 user, 15.2x faster at 100 users

### **Docker PostgreSQL Performance**
- **Append operations**: 850 ops/sec (Tiny), 850 ops/sec (Small)
- **AppendIf operations**: 10 ops/sec (Tiny), 10 ops/sec (Small)
- **Projection operations**: 2,395 ops/sec (Tiny), 2,395 ops/sec (Small)
- **Concurrency scaling**: 347 ops/sec at 1 user, 10.4 ops/sec at 100 users

### **Dataset Differences**
- **Docker Tiny**: 5 courses, 10 students, 17 enrollments
- **Local Tiny**: 5 courses, 10 students, 20 enrollments
- **Docker Small**: 1,000 courses, 10,000 students, 49,871 enrollments
- **Local Small**: 500 courses, 5,000 students, 25,000 enrollments

## Setup Guides

- **🐳 [Docker PostgreSQL Setup](./benchmark-setup-docker.md)**: Containerized setup
- **💻 [Local PostgreSQL Setup](./benchmark-setup-local.md)**: Native installation
- **📊 [Performance Comparison](./performance-comparison.md)**: Detailed Docker vs Local analysis

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

### **Append vs AppendIf**

#### **Append**
- **What it tests**: Simple event storage without business rule validation
- **Use case**: High-volume event streaming, event-driven architectures
- **Performance**: Fastest operation (8,000+ ops/sec local, 850+ ops/sec Docker)

#### **AppendIf**
- **What it tests**: Conditional event storage with business rule validation
- **Use case**: Event sourcing with business consistency, preventing duplicate operations
- **Performance**: Significantly slower (16+ ops/sec local, 10+ ops/sec Docker) due to:
  - Business rule validation queries
  - Conflict detection logic
  - Past event scanning for conditions

## Concurrency Performance Analysis

### **Read Operations - Concurrency Scaling**

**Test Environment**: macOS (darwin 23.6.0) with Apple M1 Pro  
**Database**: PostgreSQL with 50-connection pool  
**Test Data**: Small dataset (500 courses, 5,000 students, 25,000 enrollments) and Medium dataset (1,000 courses, 10,000 students, 50,000 enrollments)

### **Performance Scaling Pattern**

| Concurrency Level | Local PostgreSQL | Docker PostgreSQL | Performance Ratio |
|------------------|------------------|-------------------|-------------------|
| **1 User** | 8,230 ops/sec | 347 ops/sec | **23.7x faster** |
| **10 Users** | 1,438 ops/sec | 157 ops/sec | **9.2x faster** |
| **100 Users** | 158 ops/sec | 10.4 ops/sec | **15.2x faster** |

### **Resource Usage Scaling**

| Concurrency Level | Local PostgreSQL | Docker PostgreSQL |
|------------------|------------------|-------------------|
| **1 User** | 1.1KB memory, 25 allocations | 1.1KB memory, 25 allocations |
| **10 Users** | 11.8KB memory, 270 allocations | 11.8KB memory, 270 allocations |
| **100 Users** | 125KB memory, 2,852 allocations | 124.5KB memory, 2,853 allocations |

## Performance Data Summary

### **Throughput Performance**
- **Append operations**: Local PostgreSQL 9.4x faster than Docker PostgreSQL
- **AppendIf operations**: Local PostgreSQL 1.6x faster than Docker PostgreSQL
- **Projection operations**: Local PostgreSQL 7.0x faster than Docker PostgreSQL
- **Business workflows**: Local PostgreSQL 6.1x faster than Docker PostgreSQL

### **Concurrency Performance**
- **1 User**: Local PostgreSQL 23.7x faster than Docker PostgreSQL
- **10 Users**: Local PostgreSQL 9.2x faster than Docker PostgreSQL
- **100 Users**: Local PostgreSQL 15.2x faster than Docker PostgreSQL

### **Resource Usage**
- **Memory scaling**: Both setups show similar memory usage patterns
- **Allocation patterns**: Both setups show similar allocation scaling
- **Connection pooling**: Both use 50-connection pools

## Dataset Comparison

| Metric | Tiny Dataset | Small Dataset | Medium Dataset |
|--------|--------------|---------------|----------------|
| **Courses** | 5 | 500 | 1,000 |
| **Students** | 10 | 5,000 | 10,000 |
| **Enrollments** | 20 | 25,000 | 50,000 |
| **Local Append Performance** | 8,030 ops/sec | 7,900 ops/sec | 8,000 ops/sec |
| **Docker Append Performance** | 850 ops/sec | 850 ops/sec | 850 ops/sec |
| **Performance Ratio** | 9.4x | 9.3x | 9.4x |

## Performance Data Summary

### **Dataset Performance Patterns**
- **Tiny dataset**: Higher throughput across all operations
- **Small dataset**: Lower throughput, higher data volume
- **Memory usage**: Scales with dataset size (1.4KB → 33.2MB)
- **Allocation patterns**: Similar scaling across both setups

### **Operation Performance Patterns**
- **Append operations**: Highest throughput, minimal latency
- **AppendIf operations**: Lowest throughput, highest latency
- **Read operations**: Variable performance based on complexity
- **Projection operations**: High throughput with streaming, lower with sync

### **Concurrency Performance Patterns**
- **1 User**: Maximum performance for both setups
- **10 Users**: Moderate performance degradation
- **100 Users**: Significant performance degradation
- **Resource scaling**: Linear increase in memory and allocations
