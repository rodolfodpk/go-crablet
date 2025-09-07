# go-crablet Benchmarks

This directory contains the Go benchmark test files for the go-crablet library, which aims to implement Dynamic Consistency Boundaries (DCB).

## 📚 Documentation

📖 **[Main Documentation](../README.md)** - Complete setup, usage, and analysis guide  
📊 **[Performance Reports](../docs/performance-local.md)** - Detailed performance analysis and recommendations

## 🚀 Quick Start

### Prerequisites
- PostgreSQL 16 (local or Docker)
- Go 1.25+
- Database setup (see main README for setup instructions)

### Run All Benchmarks
```bash
# Run all benchmarks (Tiny, Small, Medium datasets)
cd internal/benchmarks
go test -bench=. -benchmem -benchtime=1s -timeout=10m .

# Expected execution time: ~2.9 minutes
```

### Run Specific Dataset
```bash
# Run only Small dataset benchmarks
go test -bench=BenchmarkAppend_Small -benchmem -benchtime=1s -timeout=10m .

# Run only Tiny dataset benchmarks  
go test -bench=BenchmarkAppend_Tiny -benchmem -benchtime=1s -timeout=10m .

# Run only Medium dataset benchmarks
go test -bench=BenchmarkAppend_Medium -benchmem -benchtime=1s -timeout=10m .
```

### Run Specific Operations
```bash
# Run only Append operations (all datasets)
go test -bench=Append_Concurrent -benchmem -benchtime=1s -timeout=10m .

# Run only AppendIf operations (all datasets)
go test -bench=AppendIf_ -benchmem -benchtime=1s -timeout=10m .

# Run only Projection operations (all datasets)
go test -bench=Project_ -benchmem -benchtime=1s -timeout=10m .
```

## 📁 Files

- `benchmark_runner.go` - Core benchmark framework and standardized tests
- `append_benchmarks_test.go` - Top-level benchmark runners (Tiny, Small, Medium)
- `setup/` - Benchmark data setup and utilities
- `tools/` - Benchmark data generation tools
- `scripts/` - Benchmark execution scripts

## 🎯 What's Tested

### Standardized Benchmark Suite (54 total tests)

**Append Operations (18 tests):**
- `Append_Concurrent_1User_1Event` - Single event, single user
- `Append_Concurrent_10Users_1Event` - Single event, 10 users
- `Append_Concurrent_25Users_1Event` - Single event, 25 users
- `Append_Concurrent_1User_100Events` - Batch events, single user
- `Append_Concurrent_10Users_100Events` - Batch events, 10 users
- `Append_Concurrent_25Users_100Events` - Batch events, 25 users
- **Datasets**: Tiny, Small, Medium

**AppendIf Operations (36 tests):**
- `AppendIf_NoConflict_Concurrent_*` - Conditional append (success scenario)
- `AppendIf_WithConflict_Concurrent_*` - Conditional append (failure scenario)
- **Concurrency**: 1, 10, 100 users
- **Events**: 1, 100 events per operation
- **Datasets**: Tiny, Small, Medium

**Projection Operations (18 tests):**
- `Project_Concurrent_*` - Synchronous state reconstruction
- `ProjectStream_Concurrent_*` - Asynchronous streaming reconstruction
- **Concurrency**: 1, 10, 25 users
- **Datasets**: Tiny, Small, Medium

## 🔧 Configuration

### Benchmark Parameters
- **Dataset Sizes**: Tiny (5 courses), Small (1K courses), Medium (10K courses)
- **Concurrency Levels**: 1, 10, 25, 100 users
- **Event Counts**: 1, 100 events per operation
- **Benchmark Time**: 1 second per test (configurable)

### Database Setup
```bash
# Local PostgreSQL
brew services start postgresql@16

# Docker PostgreSQL
docker-compose up -d
```

## 📊 Performance Metrics

All benchmarks measure:
- **Throughput**: Operations per second (ops/sec)
- **Latency**: Nanoseconds per operation (ns/op)
- **Memory**: Bytes per operation (B/op)
- **Allocations**: Number of allocations per operation

## 🧹 Clean Benchmark Suite

**Optimized with Go 1.25 Features:**
- **WaitGroup.Go()**: Improved concurrent performance
- **context.WithTimeoutCause**: Better error handling
- **Removed artificial delays**: Eliminated overhead
- **Standardized names**: All tests align with performance documentation

**Removed Redundant Tests:**
- Batch scaling tests (redundant with concurrent tests)
- Single-user tests (redundant with concurrent 1-user tests)
- Memory usage tests (redundant with -benchmem flag)
- Business workflow tests (not standardized)

## 📄 **License**

This benchmark and documentation are licensed under the Apache License 2.0 - see the [LICENSE](../../../LICENSE) file for details.

---

*For comprehensive documentation, setup instructions, and performance analysis, see the [main README](../README.md).* 