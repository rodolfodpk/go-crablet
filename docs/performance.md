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

**Local PostgreSQL**: Excellent single-threaded performance (8,230 ops/sec) with moderate concurrency impact (1,438 ops/sec at 10 users) and reasonable degradation under high load (158 ops/sec at 100 users).

**Docker PostgreSQL**: Good single-threaded performance (347 ops/sec) with moderate concurrency impact (157 ops/sec at 10 users) and significant degradation under high load (10.4 ops/sec at 100 users).

### **Performance Scaling Pattern**

| Concurrency Level | Local PostgreSQL | Docker PostgreSQL | Performance Ratio |
|------------------|------------------|-------------------|-------------------|
| **1 User** | 8,230 ops/sec | 347 ops/sec | **23.7x faster** |
| **10 Users** | 1,438 ops/sec | 157 ops/sec | **9.2x faster** |
| **100 Users** | 158 ops/sec | 10.4 ops/sec | **15.2x faster** |

## Key Performance Insights

### **Local PostgreSQL Advantages**
- **No Docker overhead**: Direct hardware access
- **Optimized configuration**: Homebrew PostgreSQL tuned for macOS
- **Better resource allocation**: Full system resources available
- **Faster filesystem access**: Direct disk access vs Docker layers

### **Docker PostgreSQL Benefits**
- **Consistent environment**: Same setup across all deployments
- **Production-like**: Containerized deployment simulation
- **Resource isolation**: Controlled resource allocation
- **Easy setup**: `docker-compose up -d`

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
