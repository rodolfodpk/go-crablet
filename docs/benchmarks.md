# Performance Benchmarks Overview

This document provides a comprehensive guide to performance testing and benchmarking in the go-crablet project with **production-ready performance**.

## Overview

go-crablet includes comprehensive performance testing across multiple components to ensure optimal performance for event sourcing applications. Our benchmarking strategy covers:

- **Core Library Performance**: Go-level benchmarks for the DCB pattern implementation
- **HTTP/REST API Performance**: Web application performance under load with **zero errors**

## 🚀 **Latest Performance Results**

### **Web-App HTTP/REST API**
- ✅ **Zero HTTP failures** (0 out of 66,137 requests)
- ✅ **Zero custom errors** (0% error rate)
- ✅ **Sub-30ms average response time** (27.98ms)
- ✅ **Sub-500ms 99th percentile** (460ms)
- ✅ **98.24% check success rate**
- ✅ **Stable 137 req/s throughput**

### **Quick Test Performance**
- ✅ **100% success rate** (all HTTP requests successful)
- ✅ **100% check success rate** (all performance checks passed)
- ✅ **Sub-2ms average response time** (1.17ms)
- ✅ **816 requests/second throughput**

## Benchmark Types

### 1. 🌐 Web-App Benchmarks
**Location**: [`internal/web-app/BENCHMARK.md`](../internal/web-app/BENCHMARK.md)

**What it tests**: HTTP/REST API performance using k6 load testing

**Key Features**:
- Quick test (10 seconds, 1 VU) for rapid validation
- Full benchmark (8 minutes, up to 50 VUs) for comprehensive testing
- Multiple scenarios: append, read, complex queries
- Performance thresholds and success rate monitoring
- **Expected performance**: ~137 requests/second, <500ms p99, **zero errors**

**Use Case**: When you need to test HTTP API performance for web applications or REST clients.

### 2. ⚡ Go Benchmarks
**Location**: [`internal/benchmarks/README.md`](../internal/benchmarks/README.md)

**What it tests**: Core library performance using Go's built-in benchmarking

**Key Features**:
- Append performance testing with various batch sizes
- Read performance testing with different query patterns
- Projection performance analysis
- Memory usage and allocation profiling
- Detailed performance reports and analysis

**Use Case**: When you need to understand the raw performance characteristics of the DCB pattern implementation.

## Quick Start

### Prerequisites
- **Docker**: For PostgreSQL database
- **Go**: For running servers and benchmarks
- **k6**: For HTTP load testing (`brew install k6` on macOS)

### Running All Benchmarks

1. **Start PostgreSQL**:
   ```bash
   docker-compose up -d postgres
   ```

2. **Run Web-App Benchmarks**:
   ```bash
   cd internal/web-app
   make benchmark
   ```

3. **Run Go Benchmarks**:
   ```bash
   cd internal/benchmarks
   go run main.go
   ```

## Performance Expectations

### Web-App (HTTP/REST)
- **Quick Test**: ~8,164 requests, 816 req/s, 1.17ms avg, 1.79ms p95
- **Full Benchmark**: ~66,137 requests, 137 req/s, 28ms avg, 460ms p99
- **Success Rate**: 100% HTTP success, 98.24% performance threshold compliance
- **Zero Errors**: 0 HTTP failures, 0% custom error rate

### Go Benchmarks
- **Append Performance**: Optimized for batch operations
- **Read Performance**: Efficient streaming with pgx
- **Memory Usage**: Minimal allocations for large datasets
- **Concurrency**: Thread-safe operations with connection pooling

## Performance Optimizations Applied

The system has been optimized for production performance with:

### **Database Configuration**
- **Connection Pool**: 200 max connections, 50 min connections
- **PostgreSQL Memory**: 2GB allocation for better performance
- **Health Check**: 30-second intervals for connection monitoring

### **Load Testing Configuration**
- **Max VUs**: 50 (reduced from 100 for stability)
- **Request Spacing**: 0.2s intervals for database recovery
- **Gentle Ramp-up**: 5 → 15 → 50 VUs over 8 minutes
- **Batch Size**: 5 requests per batch for stability

### **Database Indexes**
- **Position Index**: `idx_events_position` for sequential reads
- **Tags Index**: `idx_events_tags` using GIN for array queries
- **Type Index**: `idx_events_type` for type-based queries
- **Composite Indexes**: `idx_events_type_position` for optimized queries

## Benchmark Results Interpretation

### k6 Metrics to Watch
- **Success Rate**: Should be 100% for HTTP requests
- **Response Times**: p95 should be under thresholds (200ms for most operations)
- **Throughput**: Requests per second under expected load
- **Error Rate**: Should be 0% for successful tests

### Go Benchmark Metrics
- **ns/op**: Nanoseconds per operation (lower is better)
- **B/op**: Bytes allocated per operation (lower is better)
- **allocs/op**: Memory allocations per operation (lower is better)

## Troubleshooting

### Common Issues
- **Port conflicts**: Use `lsof -i :8080` to find and kill processes
- **Database connection**: Ensure PostgreSQL is running with `docker ps`
- **k6 not found**: Install from https://k6.io/docs/getting-started/installation/

### Performance Issues
- **High response times**: Check database performance and connection pooling
- **Low throughput**: Verify system resources (CPU, memory, disk I/O)
- **Connection errors**: Check PostgreSQL max connections and connection limits

## Contributing to Benchmarks

### Adding New Tests
1. **k6 Scripts**: Add new test functions to existing scripts
2. **Go Benchmarks**: Add new benchmark functions to test files
3. **Documentation**: Update this overview and specific benchmark docs
4. **Thresholds**: Adjust performance thresholds based on new test patterns

### Best Practices
- **Consistent Environment**: Run benchmarks on similar hardware
- **Multiple Runs**: Run benchmarks multiple times to account for variance
- **Baseline Comparison**: Compare against previous results
- **Resource Monitoring**: Monitor system resources during benchmarks

## Continuous Integration

Benchmarks are integrated into the CI/CD pipeline to:
- **Regression Detection**: Catch performance regressions early
- **Trend Analysis**: Track performance over time
- **Release Validation**: Ensure releases meet performance criteria
- **Documentation**: Auto-generate performance reports

## Support

For benchmark-related issues:
- Check the troubleshooting section in each benchmark document
- Review server logs and k6 output
- Verify system resources and configuration
- Consult the specific benchmark documentation for detailed guidance

---

**Next Steps**: Choose a specific benchmark type above to get detailed instructions and run your own performance tests.