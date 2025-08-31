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

## Dataset-Specific Performance

### **Docker PostgreSQL**
- **📊 [Tiny Dataset](./performance-docker-tiny.md)**: 5 courses, 10 students, 17 enrollments
- **📊 [Small Dataset](./performance-docker-small.md)**: 1,000 courses, 10,000 students, 49,871 enrollments

### **Local PostgreSQL**
- **📊 [Tiny Dataset](./performance-local-tiny.md)**: 5 courses, 10 students, 20 enrollments
- **📊 [Small Dataset](./performance-local-small.md)**: 500 courses, 5,000 students, 25,000 enrollments

## Performance Recommendations

### **For Development**
- **Use Local PostgreSQL**: 6-9x faster performance for daily development
- **Faster feedback**: Quicker test cycles and iteration
- **Realistic performance**: Closer to production expectations

### **For CI/CD & Production Planning**
- **Use Docker PostgreSQL**: Consistent, containerized environment
- **Conservative estimates**: Safe for production planning
- **Reproducible**: Same environment across all deployments

### **For Benchmarking**
- **Compare both**: Understand performance differences
- **Local for development**: Use local setup for daily work
- **Docker for validation**: Use Docker for final validation

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

## Performance Recommendations

### **For Development**
- Use **Local PostgreSQL** for fast feedback and testing
- **Append operations** provide excellent performance (8,000+ ops/sec)
- **AppendIf operations** suitable for business rule validation (16+ ops/sec)

### **For Production Planning**
- **Docker PostgreSQL** provides realistic production expectations
- **Concurrency testing** essential for production workloads
- **Memory monitoring** critical for high-concurrency scenarios

### **For High-Performance Scenarios**
- **Local PostgreSQL** offers maximum performance potential
- **Batch operations** provide 2x performance improvement
- **Streaming projections** offer best performance for state reconstruction
