# Performance Benchmarks Overview

This document provides a comprehensive guide to performance testing and benchmarking in the go-crablet project.

## Overview

go-crablet includes comprehensive performance testing across multiple components to ensure optimal performance for event sourcing applications. Our benchmarking strategy covers:

- **Core Library Performance**: Go-level benchmarks for the DCB pattern implementation
- **HTTP/REST API Performance**: Web application performance under load
- **gRPC API Performance**: High-performance gRPC service testing

## Benchmark Types

### 1. 🌐 Web-App Benchmarks
**Location**: [`internal/web-app/BENCHMARK.md`](../internal/web-app/BENCHMARK.md)

**What it tests**: HTTP/REST API performance using k6 load testing

**Key Features**:
- Quick test (1 minute, 10 VUs) for rapid validation
- Full benchmark (7 minutes, up to 30 VUs) for comprehensive testing
- Multiple scenarios: append, read, complex queries
- Performance thresholds and success rate monitoring
- Expected performance: ~60-100 requests/second, <200ms p95

**Use Case**: When you need to test HTTP API performance for web applications or REST clients.

### 2. 🔌 gRPC App Benchmarks  
**Location**: [`internal/grpc-app/BENCHMARK.md`](../internal/grpc-app/BENCHMARK.md)

**What it tests**: gRPC API performance using k6 with gRPC extension

**Key Features**:
- Quick test (1 minute, 10 VUs) for rapid validation
- Full benchmark (7 minutes, up to 30 VUs) for comprehensive testing
- gRPC-specific metrics and performance analysis
- Higher throughput than HTTP due to binary protocol
- Expected performance: ~60-106 requests/second, <200ms p95

**Use Case**: When you need high-performance, low-latency communication between services.

### 3. ⚡ Go Benchmarks
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
- **k6**: For HTTP/gRPC load testing (`brew install k6` on macOS)
- **k6-grpc**: For gRPC testing (`k6 install xk6-grpc`)

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

3. **Run gRPC Benchmarks**:
   ```bash
   cd internal/grpc-app
   make benchmark
   ```

4. **Run Go Benchmarks**:
   ```bash
   cd internal/benchmarks
   go run main.go
   ```

## Performance Expectations

### Web-App (HTTP/REST)
- **Quick Test**: ~3,700 requests, 60 req/s, 60ms avg, 200ms p95
- **Full Benchmark**: ~42,000 requests, 100 req/s, 100ms avg, 2.5s p95
- **Success Rate**: 100% HTTP success, ~92-97% performance threshold compliance

### gRPC App
- **Quick Test**: ~3,700 requests, 61 req/s, 45ms avg, 156ms p95
- **Full Benchmark**: ~44,600 requests, 106 req/s, 78ms avg, 1.8s p95
- **Success Rate**: 100% gRPC success, ~94-100% performance threshold compliance

### Go Benchmarks
- **Append Performance**: Optimized for batch operations
- **Read Performance**: Efficient streaming with pgx
- **Memory Usage**: Minimal allocations for large datasets
- **Concurrency**: Thread-safe operations with connection pooling

## Benchmark Results Interpretation

### k6 Metrics to Watch
- **Success Rate**: Should be 100% for HTTP/gRPC requests
- **Response Times**: p95 should be under thresholds (200ms for most operations)
- **Throughput**: Requests per second under expected load
- **Error Rate**: Should be 0% for successful tests

### Go Benchmark Metrics
- **ns/op**: Nanoseconds per operation (lower is better)
- **B/op**: Bytes allocated per operation (lower is better)
- **allocs/op**: Memory allocations per operation (lower is better)

## Troubleshooting

### Common Issues
- **Port conflicts**: Use `lsof -i :8080` or `lsof -i :9090` to find and kill processes
- **Database connection**: Ensure PostgreSQL is running with `docker ps`
- **k6 not found**: Install from https://k6.io/docs/getting-started/installation/
- **gRPC extension missing**: Install with `k6 install xk6-grpc`

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