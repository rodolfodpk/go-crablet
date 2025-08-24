# Performance Guide

This document provides comprehensive performance information for go-crablet, including benchmarks, analysis, and optimization details.

## 🚀 **Performance Overview**

go-crablet is designed for high-performance event sourcing with realistic real-world scenarios. Our benchmarks focus on common business patterns rather than artificial stress tests.

### **Key Performance Characteristics**
- **Single Events**: ~2,200 ops/sec with 1.1-1.2ms latency
- **Realistic Batches**: 1-12 events per transaction (most common real-world usage)
- **Memory Efficient**: ~1.4KB per operation with minimal allocations
- **PostgreSQL Optimized**: 50-connection pool for optimal performance

## 📊 **Benchmark Results**

### **Test Environment**
- **Platform**: macOS (darwin 23.6.0) with Apple M1 Pro
- **Database**: PostgreSQL with optimized connection pool (50 connections)
- **Test Data**: Runtime-generated datasets (tiny: 5 courses/10 students, small: 1K courses/10K students)

### **Core Operations Performance**

| Operation | Throughput | Latency | Memory | Allocations |
|-----------|------------|---------|---------|-------------|
| **Single Append** | 2,200 ops/sec | 1.1-1.2ms | 1.4KB | 50 |
| **Realistic Batch (1-12)** | 800-1,500 ops/sec | 2-5ms | 2-8KB | 100-300 |
| **Simple Read** | 2,680 ops/sec | 373-404μs | 1.0KB | 22 |
| **Complex Queries** | 2,576 ops/sec | 387-424μs | 1.0KB | 22 |
| **State Projection** | 2,800 ops/sec | 357-356μs | 1.5KB | 31 |

### **Realistic Business Scenarios**

| Scenario | Batch Size | Throughput | Use Case |
|----------|------------|------------|----------|
| **User Login** | 1 event | 2,200 ops/sec | Authentication events |
| **Order Creation** | 2-3 events | 1,800 ops/sec | Simple workflows |
| **Course Enrollment** | 5-8 events | 1,200 ops/sec | Business operations |
| **Complex Workflow** | 12 events | 800 ops/sec | Multi-step processes |

## 🔧 **Performance Optimizations**

### **Database Connection Pooling**
- **Pool Size**: 50 connections for optimal throughput
- **Connection Life**: 10 minutes with 5-minute idle time
- **Result**: Consistent performance under load

### **SQL Function Optimization**
- **Before**: Complex CTEs with multiple subqueries (~50ms per operation)
- **After**: Single optimized query (~5ms per operation)
- **Improvement**: **10x faster** with 10x less memory usage

### **Memory Management**
- **Efficient Allocations**: Minimized memory allocations per operation
- **Garbage Collection**: Optimized for low GC pressure
- **Result**: Predictable memory usage patterns

## 📈 **Benchmark Structure**

### **Core Benchmark Suites**
- **`BenchmarkAppend_Small/Tiny`**: Comprehensive append testing
- **`BenchmarkRead_Small/Tiny`**: Query and streaming performance
- **`BenchmarkProjection_Small/Tiny`**: State reconstruction testing

### **Business Scenario Tests**
- **Course Enrollment**: Real student registration workflows
- **Ticket Booking**: Concurrent booking with capacity limits
- **Mixed Operations**: Combined append, query, and projection sequences

### **Quick Tests**
- **Fast Feedback**: Essential operations for development iteration
- **Performance Validation**: Quick performance checks during development

## 🏃‍♂️ **Running Benchmarks**

### **Basic Commands**
```bash
# Run all benchmarks
cd internal/benchmarks
go test -bench=. -benchmem -benchtime=2s -timeout=5m .

# Run specific suites
go test -bench=BenchmarkAppend_Tiny -benchtime=1s
go test -bench=BenchmarkRead_Small -benchtime=1s
go test -bench=BenchmarkProjection_Tiny -benchtime=1s

# Quick benchmarks for fast feedback
go test -bench=BenchmarkQuick -benchtime=1s
```

### **Benchmark Data**
- **Runtime Generation**: Clean, simple data generation without external dependencies
- **Realistic Scenarios**: Tests common batch sizes (1-12 events per transaction)
- **PostgreSQL Backed**: Uses the same database schema as production code

## 🎯 **Performance Recommendations**

### **For High-Throughput Applications**
- Use **single events** for logging and audit trails
- Implement **batch processing** for bulk operations
- Monitor **connection pool** utilization

### **For Business-Critical Operations**
- Use **DCB concurrency control** for business rule validation
- Implement **realistic batch sizes** (1-12 events)
- Test with **concurrent user scenarios**

### **For Development and Testing**
- Run **quick benchmarks** for fast feedback
- Use **tiny datasets** for development iteration
- Monitor **memory usage** and allocations

## 📊 **Performance Monitoring**

### **Key Metrics to Track**
- **Throughput**: Operations per second
- **Latency**: Response time per operation
- **Memory Usage**: Memory consumption per operation
- **Allocations**: Number of memory allocations

### **Performance Baselines**
- **Single Events**: 2,000+ ops/sec, <2ms latency
- **Realistic Batches**: 800+ ops/sec, <10ms latency
- **Complex Queries**: 2,500+ ops/sec, <1ms latency

## 🔍 **Performance Analysis**

### **Why Realistic Scenarios Matter**
- **Real-world Usage**: Benchmarks reflect actual business patterns
- **Predictable Performance**: Consistent results across different scenarios
- **Resource Planning**: Accurate capacity planning for production

### **Performance vs. Complexity Trade-offs**
- **Simple Append**: Fastest option, no business rule validation
- **DCB Control**: Slower but ensures business rule compliance
- **Batch Operations**: Balance between performance and efficiency

## 📚 **Further Reading**

- **[Getting Started](./getting-started.md)**: Setup and basic usage
- **[Testing Guide](./testing.md)**: Comprehensive testing information
- **[Overview](./overview.md)**: DCB pattern concepts and architecture

---

**Note**: All benchmarks use realistic real-world scenarios. Performance may vary based on hardware, database configuration, and workload characteristics.
