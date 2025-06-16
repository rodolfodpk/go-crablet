# go-crablet Benchmarks

This directory contains the Go benchmark test files for the go-crablet library, which implements Dynamic Consistency Boundaries (DCB).

## 📚 Documentation

📖 **[Main Documentation](../README.md)** - Complete setup, usage, and analysis guide  
📊 **[Latest Performance Report](../BENCHMARK_REPORT.md)** - Detailed performance analysis and recommendations

## 🚀 Quick Start

```bash
# Run all benchmarks
go test -bench=. -benchmem

# Run specific benchmark category
go test -bench=BenchmarkAppend -benchmem
go test -bench=BenchmarkRead -benchmem
go test -bench=BenchmarkProjection -benchmem

# Run with specific dataset size
go test -bench=BenchmarkAppend_Small -benchmem
go test -bench=BenchmarkAppend_Medium -benchmem
```

## 📁 Files

- `benchmark_runner.go` - Core benchmark framework and utilities
- `append_benchmarks_test.go` - Append operation benchmarks  
- `read_benchmarks_test.go` - Read operation benchmarks
- `projection_benchmarks_test.go` - Projection benchmarks

## 🎯 What's Tested

- **Append Operations**: Single and batch event appends
- **Read Operations**: Simple, complex, and streaming reads
- **Projection Operations**: Single and multiple projector performance
- **Memory Usage**: Memory allocation and efficiency analysis

## 🔧 Configuration

Benchmark parameters can be modified in `benchmark_runner.go`:
- Dataset sizes (Small, Medium, Large, XLarge)
- Batch sizes for append operations
- Projector counts for projection tests
- Memory measurement settings

## 📄 **License**

This benchmark and documentation are licensed under the Apache License 2.0 - see the [LICENSE](../../../LICENSE) file for details.

---

*For comprehensive documentation, setup instructions, and performance analysis, see the [main README](../README.md).* 