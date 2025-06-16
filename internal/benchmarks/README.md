# go-crablet Performance Benchmarks

This directory contains comprehensive performance benchmarks for the go-crablet library, which implements Dynamic Consistency Boundaries (DCB). The benchmarks test all public API operations with realistic datasets and various configurations.

## 📋 **Latest Benchmark Report**

📊 **[View Latest Performance Analysis](BENCHMARK_REPORT.md)** - Comprehensive analysis of go-crablet performance across all operations with detailed metrics, recommendations, and raw benchmark results.

---

## 📊 **Benchmark Overview**

### **What's Being Tested**

#### **1. Append Operations**
- Single event append performance
- Batch append with different sizes (10, 100, 1000 events)
- Throughput measurements (events/second)

#### **2. Read Operations**
- Simple queries (single event type, single tag)
- Complex queries (OR conditions, multiple tags)
- Traditional batch read vs streaming read
- Iterator-based streaming vs channel-based streaming

#### **3. Projection Operations**
- Single projector performance
- Multiple projector performance (1, 5 projectors)
- Traditional projection vs channel-based projection
- Memory usage analysis

#### **4. Memory Usage**
- Memory allocation per operation
- Memory efficiency comparison between read methods
- Memory usage for different dataset sizes

### **Dataset Sizes**

| Size   | Courses | Students | Enrollments | Total Events |
|--------|---------|----------|-------------|--------------|
| Small  | 1,000   | 10,000   | 50,000      | 61,000       |
| Medium | 5,000   | 50,000   | 250,000     | 305,000      |
| Large  | 10,000  | 100,000  | 500,000     | 610,000      |
| XLarge | 20,000  | 200,000  | 1,000,000   | 1,220,000    |

## 🚀 **Running Benchmarks**

### **Prerequisites**

1. **Docker**: Required for PostgreSQL container
2. **Docker Compose**: Required for managing the database service
3. **Go 1.21+**: For running the benchmarks
4. **Dependencies**: All required Go modules

### **Setup**

The benchmarks use the existing `docker-compose.yaml` file from the project root, which automatically applies the `schema.sql` file.

```bash
# Start the database (if not already running)
cd /path/to/go-crablet
docker-compose up -d

# The benchmarks will automatically use this database
```

### **Quick Start**

```bash
# Run all benchmarks with small dataset
cd internal/benchmarks/benchmarks
go test -bench=. -benchmem -benchtime=10s

# Or use the convenient script
cd internal/benchmarks
./run_benchmarks.sh quick

# Run specific benchmark category
./run_benchmarks.sh append Small

# Run with specific dataset size
./run_benchmarks.sh all Large -t 30s
```

### **Benchmark Options**

```bash
# Run with more iterations for better accuracy
go test -bench=. -benchmem -benchtime=30s -benchtime=1000x

# Run with CPU profiling
go test -bench=. -benchmem -cpuprofile=cpu.prof

# Run with memory profiling
go test -bench=. -benchmem -memprofile=mem.prof

# Run specific benchmark with verbose output
go test -bench=BenchmarkAppendSingle_Small -benchmem -v
```

### **Running Individual Benchmarks**

```bash
# Append benchmarks
go test -bench=BenchmarkAppend -benchmem

# Read benchmarks
go test -bench=BenchmarkRead -benchmem

# Projection benchmarks
go test -bench=BenchmarkProjection -benchmem

# Memory usage benchmarks
go test -bench=BenchmarkMemory -benchmem
```

## 📈 **Understanding Results**

### **Benchmark Output Format**

```
BenchmarkAppendSingle_Small-8         1000           1234567 ns/op        2048 B/op         32 allocs/op
```

- **BenchmarkAppendSingle_Small-8**: Benchmark name and number of CPU cores
- **1000**: Number of iterations
- **1234567 ns/op**: Average time per operation (nanoseconds)
- **2048 B/op**: Average memory allocated per operation (bytes)
- **32 allocs/op**: Average number of allocations per operation

### **Key Metrics**

1. **Throughput**: Events/second (higher is better)
2. **Latency**: Time per operation (lower is better)
3. **Memory Usage**: Bytes allocated per operation (lower is better)
4. **Allocation Count**: Number of allocations per operation (lower is better)

### **Performance Expectations**

#### **Append Operations**
- Single append: ~1,000-10,000 events/second
- Batch append: ~10,000-100,000 events/second (depending on batch size)

#### **Read Operations**
- Simple queries: ~1,000-10,000 events/second
- Complex queries: ~100-1,000 events/second
- Streaming: Similar to batch read but with lower memory usage

#### **Projection Operations**
- Single projector: ~1,000-10,000 events/second
- Multiple projectors: ~100-1,000 events/second (scales with projector count)

## 🔧 **Configuration**

### **Dataset Configuration**

Edit `setup/dataset.go` to modify dataset sizes:

```go
var DatasetSizes = map[string]DatasetConfig{
    "small": {
        Courses:     1_000,
        Students:    10_000,
        Enrollments: 50_000,
        Capacity:    100,
    },
    // Add more configurations...
}
```

### **Benchmark Parameters**

Modify benchmark parameters in `benchmarks/benchmark_runner.go`:

```go
// Change batch sizes
batchSizes := []int{10, 100, 1000, 10000}

// Change concurrency levels
concurrencyLevels := []int{1, 5, 10, 20, 50}

// Change projector counts
projectorCounts := []int{1, 5, 10, 20}
```

## 📊 **Results Analysis**

### **Comparing Results**

```bash
# Save baseline results
go test -bench=. -benchmem > baseline.txt

# After changes, compare results
go test -bench=. -benchmem > new.txt
benchcmp baseline.txt new.txt
```

### **Performance Regression Testing**

```bash
# Run benchmarks in CI/CD pipeline
go test -bench=. -benchmem -benchtime=5s -count=3
```

### **Profiling Analysis**

```bash
# Generate CPU profile
go test -bench=BenchmarkAppendSingle_Small -cpuprofile=cpu.prof

# Analyze with pprof
go tool pprof cpu.prof
```

## 🏗️ **Architecture**

### **File Structure**

```
internal/benchmarks/
├── main.go                    # Simple benchmark runner
├── setup/
│   ├── dataset.go            # Dataset generation utilities
│   └── projectors.go         # Projector definitions
├── benchmarks/
│   ├── benchmark_runner.go   # Core benchmark framework
│   ├── append_benchmarks_test.go
│   ├── read_benchmarks_test.go
│   └── projection_benchmarks_test.go
└── results/                  # Benchmark results (optional)
```

### **Key Components**

1. **BenchmarkContext**: Holds test data and store instances
2. **Dataset Generation**: Creates realistic test data
3. **Projector Definitions**: Various projector types for testing
4. **Benchmark Functions**: Individual benchmark implementations
5. **Test Files**: Go testing.B benchmarks for different operations

## 🎯 **Best Practices**

### **Running Benchmarks**

1. **Use consistent hardware**: Run on the same machine for comparisons
2. **Close other applications**: Minimize background processes
3. **Warm up**: Run benchmarks multiple times, discard first few results
4. **Use appropriate time**: `-benchtime=10s` for most benchmarks
5. **Monitor system resources**: Check CPU, memory, and disk usage

### **Interpreting Results**

1. **Look for trends**: Compare across dataset sizes
2. **Check memory usage**: Lower allocations often mean better performance
3. **Consider scalability**: How does performance change with data size?
4. **Watch for regressions**: Compare against previous results

### **Optimization Tips**

1. **Batch operations**: Use batch append for better throughput
2. **Streaming reads**: Use streaming for large datasets
3. **Projector efficiency**: Minimize projector complexity
4. **Memory management**: Reuse objects when possible

## 🐛 **Troubleshooting**

### **Common Issues**

1. **Docker not running**: Ensure Docker is running for PostgreSQL containers
2. **Docker Compose not running**: Run `docker-compose up -d` from the project root
3. **Database connection errors**: Check if the PostgreSQL container is healthy with `docker-compose ps`
4. **Out of memory**: Reduce dataset size or increase system memory
5. **Slow benchmarks**: Use smaller datasets for development
6. **Schema not applied**: Ensure `docker-entrypoint-initdb.d/schema.sql` exists and docker-compose is restarted

### **Debug Mode**

```bash
# Check docker-compose status
docker-compose ps

# View database logs
docker-compose logs postgres

# Restart the database
docker-compose restart postgres

# Run with verbose output
go test -bench=. -v

# Run single benchmark with debug info
go test -bench=BenchmarkAppendSingle_Small -v -run=^$
```

## 📝 **Contributing**

When adding new benchmarks:

1. Follow the existing naming convention
2. Add appropriate documentation
3. Include memory allocation reporting
4. Test with multiple dataset sizes
5. Add to the appropriate test file

## 📚 **References**

- [Go Benchmarking](https://golang.org/pkg/testing/#hdr-Benchmarks)
- [Performance Profiling](https://golang.org/pkg/runtime/pprof/)
- [TestContainers](https://golang.testcontainers.org/)
- [PostgreSQL Performance](https://www.postgresql.org/docs/current/performance.html)

## 📄 **License**

This benchmark and documentation are licensed under the Apache License 2.0 - see the [LICENSE](../../LICENSE) file for details. 