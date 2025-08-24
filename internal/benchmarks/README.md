# go-crablet Performance Benchmarks

This directory contains comprehensive performance benchmarks for the go-crablet library, which explores and learns about Dynamic Consistency Boundaries (DCB). The benchmarks test all public API operations with realistic datasets and DCB-focused queries that demonstrate proper consistency boundary patterns.

## 📋 **Latest Benchmark Report**

📊 **[View Latest Performance Analysis](reports/BENCHMARK_REPORT.md)** - Comprehensive analysis of go-crablet performance across all operations with DCB pattern exploration, detailed metrics, and raw benchmark results.

---

## 🏗️ **Directory Structure**

This directory contains comprehensive performance benchmarks organized for clarity:

```
internal/benchmarks/
├── main.go                   # Standalone benchmark application
├── performance/              # Go benchmark-based performance tools (68 benchmarks)
├── setup/                    # Dataset and projector setup
├── tools/                    # Dataset preparation tools
├── scripts/                  # Execution scripts
└── reports/                  # Benchmark reports
```

### **Benchmark Types**
- **Standalone Application** (`main.go`): Custom performance analysis with detailed reporting
- **Go Performance Tools** (`performance/`): 68 benchmarks using Go's testing framework for CI/CD integration
- **Setup Utilities** (`setup/`, `tools/`): Dataset generation and management
- **Execution Scripts** (`scripts/`): Automated benchmark running

## 🚀 **Quick Start with Dataset Caching**

The benchmarks now use **dataset caching** for much faster execution. Datasets are pre-generated and stored in SQLite for instant loading.

### **First Time Setup (One-time)**

```bash
# Pre-generate and cache all datasets (tiny and small)
cd internal/benchmarks
go run tools/prepare_datasets_main.go
```

This will create cached datasets that make subsequent benchmark runs **much faster**.

**Note**: The cache directory (`internal/benchmarks/cache/`) is gitignored, so each developer needs to run this setup once on their machine.

### **Running Benchmarks**

```bash
# Run all benchmarks with cached datasets
go run main.go

# Or use the convenient script
./scripts/run_benchmarks.sh quick

# Run specific benchmark category
./scripts/run_benchmarks.sh append small

# Run with specific dataset size
./scripts/run_benchmarks.sh all tiny -t 30s
```

---

## 📊 **Benchmark Overview**

### **Current API Methods**

The benchmarks test comprehensive append operations including advisory locks:

- **`Append`**: Basic append with ReadCommitted isolation
- **`AppendIf`**: Conditional append with RepeatableRead isolation  
- **`AppendIfIsolated`**: Conditional append with Serializable isolation
- **`Append with Advisory Locks`**: Append with PostgreSQL advisory locks via `lock:` tags

### **Advisory Lock Benchmarks**

New comprehensive advisory lock benchmarks are now available:

- **Single Advisory Lock**: Individual events with advisory locks
- **Batch Advisory Lock**: Batch events with advisory locks (10, 100, 1000)
- **Concurrent Advisory Lock**: Multiple goroutines with advisory locks
- **Advisory Lock Contention**: High contention scenarios with shared resources
- **Advisory Lock vs Regular**: Performance comparison between advisory locks and regular appends

### **What's Being Tested**

#### **1. Append Operations**
- **Single Event Append**: Tests `store.Append(ctx, []dcb.InputEvent{event}, nil)` - baseline performance
- **Batch Event Append**: Tests `store.Append(ctx, events, nil)` with different sizes (10, 100, 1000 events)
- **Concurrent Append**: Tests append performance with multiple goroutines
- **Performance Comparison**: Shows the efficiency gains of batch operations
- **Throughput measurements**: Events/second for different batch sizes

**Key Insight**: Batch append is ~26x more efficient than single event appends (1000-event batches vs single events)

#### **2. Advisory Lock Operations**
- **Single Advisory Lock**: Tests `store.Append()` with `lock:` tags - advisory lock performance
- **Batch Advisory Lock**: Tests batch operations with advisory locks (10, 100, 1000 events)
- **Concurrent Advisory Lock**: Tests concurrent access with advisory locks (10, 50 goroutines)
- **Advisory Lock Contention**: Tests high contention scenarios with shared resources
- **Advisory Lock vs Regular**: Direct performance comparison between advisory locks and regular appends

**Key Insight**: Advisory locks provide serialization guarantees with measurable overhead

#### **3. Read Operations (DCB-Focused)**
- **Targeted Queries**: All queries use specific consistency boundaries (no empty tags)
- **Course Queries**: `category="Computer Science"`, `course_id="course-1"`
- **Student Queries**: `major="Computer Science"`, `student_id="student-1"`
- **Enrollment Queries**: `grade="A"`, `semester="Fall2024"`
- **Cross-Entity Consistency**: OR queries demonstrating proper DCB boundaries

#### **4. Streaming Operations**
- **Channel Streaming**: DCB-focused streaming with specific student enrollments
- **Memory Efficiency**: Only processes relevant events (5 vs 50,000+)
- **Performance**: Sub-10ms for targeted streaming operations

#### **5. Projection Operations (DCB-Focused)**
- **Business Decision Boundaries**: Projectors represent real business scenarios
- **Single Projector**: CS course count, student major analysis
- **Multiple Projectors**: CS courses + CS students + A-grade enrollments
- **Channel Projection**: Real-time state projection with DCB patterns

### **Dataset Sizes & Realistic Distribution**

| Size   | Courses | Students | Enrollments | Total Events | Use Case |
|--------|---------|----------|-------------|--------------|----------|
| Tiny   | 5       | 10       | 20          | 35           | Quick validation, smoke tests |
| Small  | 1,000   | 10,000   | 50,000      | 61,000       | Performance testing, realistic scenarios |

**Realistic Data Features:**
- **Course Popularity**: Computer Science courses are more popular (30% of enrollments)
- **Student Behavior**: 70% of students enroll in 3-7 courses, 20% in 1-2, 10% in 8+
- **Temporal Patterns**: Fall semester has 60% of enrollments, Spring 40%
- **Grade Distribution**: A (25%), B (35%), C (25%), D/F (15%)
- **Major Distribution**: CS (30%), Engineering (25%), Business (20%), Arts (15%), Other (10%)

### **Dataset Caching Benefits**

- **First run**: Datasets are generated and cached (one-time cost)
- **Subsequent runs**: Instant loading from SQLite cache
- **Consistent data**: Same dataset across all benchmark runs
- **Faster iteration**: No need to regenerate data for each test

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
# Run all benchmarks with DCB-focused queries
cd internal/benchmarks
go run main.go

# Or use the convenient script
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

# Run advisory lock benchmarks
go test -bench=BenchmarkAdvisoryLock -benchmem

# Run all advisory lock benchmarks
go test -bench=BenchmarkAdvisoryLocks -benchmem
```

### **Running Individual Benchmarks**

```bash
# Append benchmarks
go test -bench=BenchmarkAppend -benchmem

# Read benchmarks (DCB-focused)
go test -bench=BenchmarkRead -benchmem

# Projection benchmarks (DCB-focused)
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

### **Performance Expectations (DCB-Focused)**

#### **Append Operations**
- **Single append**: ~100 events/second (baseline)
- **Batch append**: ~1,800-2,600 events/second (26x more efficient for large batches)
- **Concurrent append**: Up to 8,285 events/second with 20 goroutines
- **Optimal batch size**: 1000 events provides best throughput

#### **Read Operations (DCB-Focused)**
- **Targeted queries**: 9-14ms for all DCB-focused queries
- **No full scans**: Zero empty tag queries that would cause performance issues
- **Cross-entity consistency**: 12-14ms for OR queries with proper boundaries

#### **Streaming Operations**
- **Channel streaming**: 9ms for targeted student enrollments
- **Memory efficient**: Only processes relevant events (5 vs 50,000+)

#### **Projection Operations (DCB-Focused)**
- **Single projector**: 6-10ms for business decision scenarios
- **Multiple projectors**: 14ms for complex multi-projector scenarios
- **Realistic scenarios**: CS courses, CS students, A-grade enrollments

### **DCB Pattern Benefits Demonstrated**

#### **1. Consistency Boundaries**
```go
// ✅ DCB Pattern: Specific course category instead of all courses
query := dcb.NewQuery(dcb.NewTags("category", "Computer Science"), "CourseDefined")

// ✅ DCB Pattern: Specific student major instead of all students  
query := dcb.NewQuery(dcb.NewTags("major", "Computer Science"), "StudentRegistered")

// ✅ DCB Pattern: Specific enrollment grade instead of all enrollments
query := dcb.NewQuery(dcb.NewTags("grade", "A"), "StudentEnrolledInCourse")
```

#### **2. Business Decision Boundaries**
```go
// ✅ Business Decision: Count CS courses
projector := dcb.StateProjector{
    ID: "csCourseCount",
    Query: dcb.NewQuery(dcb.NewTags("category", "Computer Science"), "CourseDefined"),
    InitialState: 0,
    TransitionFn: func(state any, event dcb.Event) any { return state.(int) + 1 },
}
```

#### **3. Cross-Entity Consistency**
```go
// ✅ DCB Pattern: Cross-entity consistency check
query := dcb.Query{
    Items: []dcb.QueryItem{
        {EventTypes: []string{"CourseDefined"}, Tags: dcb.NewTags("course_id", "course-1")},
        {EventTypes: []string{"StudentRegistered"}, Tags: dcb.NewTags("student_id", "student-1")},
    },
}
```

### **Batch Append Best Practices**

Based on benchmark results, here are the recommended practices:

#### **1. Always Use Batch Append**
```go
// ❌ Avoid: Single event append
store.Append(ctx, []dcb.InputEvent{event}, nil)

// ✅ Prefer: Batch append even for single events
events := []dcb.InputEvent{event1, event2, event3}
store.Append(ctx, events, nil)
```

#### **2. Optimal Batch Sizes**
- **Small batches (10-100)**: Good for real-time processing
- **Large batches (1000+)**: Best for bulk operations and maximum throughput
- **Concurrent processing**: Up to 8,285 events/sec with 20 goroutines

#### **3. Related Events in Same Batch**
```go
// ✅ Group related events atomically
events := []dcb.InputEvent{
    dcb.NewInputEvent("CourseDefined", tags1, data1),
    dcb.NewInputEvent("StudentRegistered", tags2, data2),
    dcb.NewInputEvent("StudentEnrolledInCourse", tags3, data3),
}
store.AppendIf(ctx, events, appendCondition) // All events processed atomically
```

#### **4. DCB Pattern Exploration**
- **Use specific tags**: Always query with specific consistency boundaries
- **Avoid empty tags**: Never use queries that would cause full table scans
- **Business logic**: Projectors should represent real decision scenarios
- **Cross-entity consistency**: Use OR queries for proper boundary exploration

## 🔧 **Configuration**

### **Dataset Configuration**

Edit `setup/dataset.go` to modify dataset sizes and distribution:

```go
// Realistic course popularity distribution
var courseCategories = []string{
    "Computer Science",    // 30% of courses
    "Engineering",         // 25% of courses
    "Business",            // 20% of courses
    "Arts",                // 15% of courses
    "Other",               // 10% of courses
}

// Student behavior patterns
var enrollmentPatterns = []struct {
    minCourses int
    maxCourses int
    percentage  float64
}{
    {1, 2, 0.20},   // 20% enroll in 1-2 courses
    {3, 7, 0.70},   // 70% enroll in 3-7 courses
    {8, 12, 0.10},  // 10% enroll in 8+ courses
}
```

### **Query Configuration**

All benchmarks use DCB-focused queries:

```go
// DCB-focused query examples
queries := []dcb.Query{
    dcb.NewQuery(dcb.NewTags("category", "Computer Science"), "CourseDefined"),
    dcb.NewQuery(dcb.NewTags("major", "Computer Science"), "StudentRegistered"),
    dcb.NewQuery(dcb.NewTags("grade", "A"), "StudentEnrolledInCourse"),
    dcb.NewQuery(dcb.NewTags("course_id", "course-1"), "CourseDefined"),
}
```

## 📊 **Performance Analysis**

### **Strengths**
1. **DCB Pattern Exploration**: All queries use specific consistency boundaries
2. **No Full Scans**: Zero empty tag queries that would cause performance issues
3. **Excellent Append Performance**: Up to 8,285 events/sec with concurrency
4. **Fast Targeted Queries**: Sub-15ms performance for all DCB-focused queries
5. **Realistic Business Scenarios**: Projectors represent actual use cases
6. **Memory Efficient Streaming**: Only processes relevant events

### **Performance Characteristics**
1. **Batch Append Scaling**: Linear performance improvement with batch size
2. **Concurrency Performance**: Excellent scaling with multiple goroutines
3. **Query Performance**: Consistent sub-15ms performance for targeted queries
4. **Projection Efficiency**: Fast multi-projector scenarios
5. **Streaming Performance**: Efficient channel-based processing

### **Production Readiness**
- ✅ **Performance**: Excellent throughput and latency
- ✅ **Reliability**: Consistent, predictable performance
- ✅ **Scalability**: Good concurrency and batch performance
- ✅ **DCB Exploration**: Proper pattern exploration
- ✅ **Memory Efficiency**: Optimized for large datasets

---

*For the latest benchmark results, run: `go run main.go` from the benchmarks directory*

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