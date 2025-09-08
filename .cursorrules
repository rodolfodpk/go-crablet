# Cursor Project Rules for go-crablet

# Go-Crablet Cursor Configuration

## Project Overview
This is a Go event sourcing library **exploring** the Dynamic Consistency Boundary (DCB) pattern with PostgreSQL for production and SQLite for benchmark test data.

**IMPORTANT: This is an exploration project, not a production-ready solution.**
- We are **learning and experimenting** with DCB concepts
- Performance claims should be **modest and factual**
- Emphasize **exploration** over **production readiness**
- Be **honest about limitations** and areas for improvement

## Comprehensive Testing Requirements

### Test Categories
- **Internal Tests**: Core DCB package tests (`pkg/dcb/`)
- **External Tests**: DCB test suite (`pkg/dcb/tests/`) and example applications (`internal/examples/`)
- **Benchmark Tests**: Performance validation (`internal/benchmarks/`)

### Mandatory Test Execution
**ALWAYS run both internal and external tests before considering any changes complete:**

```bash
# Run all internal tests (core DCB functionality)
go test ./pkg/dcb -v

# Run all external tests (DCB test suite and examples)
go test ./pkg/dcb/tests -v
go test ./internal/examples/... -v

# Run all tests comprehensively
go test ./... -v
```

### Test Validation Checklist
Before marking any task complete:
- [ ] Internal DCB tests pass (core functionality)
- [ ] External DCB test suite passes (advanced scenarios)
- [ ] External example tests pass (integration scenarios)
- [ ] All Ginkgo BDD tests pass (if applicable)
- [ ] No test failures or skipped tests
- [ ] Testcontainers integration working
- [ ] Database schema and functions created successfully

### Test Framework Requirements
- **Internal Tests**: Use Ginkgo v2 + Gomega for BDD testing
- **External Tests**: Support both standard Go tests and Ginkgo BDD
- **Database Tests**: Always use Testcontainers for isolated PostgreSQL instances
- **Coverage**: Maintain comprehensive test coverage for all public APIs

## Code Coverage System

### Coverage Calculation
The project uses a sophisticated coverage system with **gocovmerge** to combine coverage from multiple test suites:

1. **Internal Tests** (`pkg/dcb/`): ~15% coverage (31 Ginkgo specs)
2. **External Tests** (`pkg/dcb/tests/`): ~77% coverage (154 Ginkgo specs)
3. **Combined Coverage**: ~81% (using gocovmerge to merge coverage files)

### Coverage Scripts
- `scripts/generate-coverage.sh` - Generates comprehensive coverage using gocovmerge
- `scripts/update-coverage-badge.sh` - Updates README badge with coverage percentage
- GitHub Actions workflow (`.github/workflows/coverage.yml`) - Automated coverage reporting

### Coverage Workflow
```bash
# Generate comprehensive coverage
./scripts/generate-coverage.sh

# Update badge (optional)
./scripts/generate-coverage.sh update-badge
```

### Key Functions with Low Coverage (0%)
- `isConcurrencyError` (append.go:267) - Private helper, tested indirectly
- `NewEventStoreWithConfig` (constructors.go:40) - Used in docs/examples, tested through main constructor
- `NewCommandSimple` (constructors.go:216) - Simple wrapper, tested through NewCommand
- `IsTableStructureError` (errors.go:83) - Public API, may need tests if used
- `GetTableStructureError` (errors.go:120) - Public API, may need tests if used
- `AsConcurrencyError`, `AsResourceError`, `AsTableStructureError` (errors.go) - Simple aliases, tested through Get* functions
- `isInputEvent`, `isTag`, `isQuery`, `isQueryItem`, `isAppendCondition` (types.go) - Interface markers, no implementation to test

### Coverage Accuracy
The 80% coverage is legitimate because:
- **External tests are comprehensive** - they test main public APIs extensively
- **gocovmerge combines best coverage** - if a line is covered in either test suite, it counts
- **Core functionality is well-tested** - main EventStore, Append, Query, and Projection methods have good coverage
- **Low-coverage functions are edge cases** - many 0% functions are helper methods or alternative constructors

### Coverage Improvement Strategy
To improve coverage further:
1. **Add tests for 0% coverage functions** - especially error handling and alternative constructors
2. **Test edge cases** - concurrency errors, table structure errors, etc.
3. **Add integration tests** - for the config-based constructors
4. **Test helper functions** - the `is*` type checking functions

## SQLite Test Data System

### Key Directories
- `internal/benchmarks/setup/` - Dataset generation and SQLite caching
- `internal/benchmarks/tools/` - Dataset preparation tools
- `cache/` - SQLite database with pre-generated test datasets
- `internal/benchmarks/benchmarks/` - Go benchmark tests

### SQLite Test Data Workflow

1. **Generate Test Datasets**:
   ```bash
   cd internal/benchmarks/tools
   go run prepare_datasets_main.go
   ```
   This creates SQLite cache with "tiny" and "small" datasets.

2. **Run Go Benchmarks**:
   ```bash
   make benchmark-go
   # or
   cd internal/benchmarks/benchmarks
   go test -bench=. -benchmem -benchtime=2s -timeout=5m .
   ```

3. **Clear Cache** (if needed):
   ```bash
   rm -rf cache/
   ```

### Dataset Sizes
- **"tiny"**: 5 courses, 10 students, 20 enrollments
- **"small"**: 1,000 courses, 10,000 students, 50,000 enrollments

### Important Notes
- SQLite is ONLY used for benchmark test data caching
- API consumers only see PostgreSQL dependency
- Benchmarks use cached datasets for fast execution
- No expensive dataset regeneration during benchmarks

### Database Setup
- PostgreSQL: Production database (localhost:5432/crablet)
- SQLite: Benchmark cache (cache/benchmark_datasets.db)

### Common Commands
- `make benchmark-go` - Run Go library benchmarks
- `make benchmark-all` - Run all benchmarks (web-app + Go)
- `make test` - Run unit tests
- `make build` - Build all packages

## Testing Stack

### Test Framework
- **Ginkgo v2**: BDD testing framework for Go
- **Gomega**: Matcher library for assertions
- **Testcontainers**: Containerized test dependencies
- **Docker Compose**: Local development environment

### Test Structure
```go
// Example test structure
var _ = Describe("EventStore", func() {
    var (
        ctx    context.Context
        store  dcb.EventStore
        pool   *pgxpool.Pool
    )

    BeforeEach(func() {
        ctx = context.Background()
        // Setup test containers or local PostgreSQL
    })

    AfterEach(func() {
        // Cleanup
    })

    Describe("Append", func() {
        It("should append events successfully", func() {
            // Test implementation
            Expect(err).To(BeNil())
            Expect(events).To(HaveLen(1))
        })
    })
})
```

### Running Tests
```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run comprehensive coverage
make coverage

# Run specific test package
go test -v ./pkg/dcb/tests/...
```

## Development Guidelines
- Keep SQLite usage internal to benchmarks only
- Use PostgreSQL for all production code
- Cache datasets to avoid regeneration overhead
- Run benchmarks with reasonable timeouts (2-5 minutes)
- **Use Ginkgo + Gomega for all new tests**
- **Use Testcontainers for database dependencies**
- **Follow BDD style with Describe/Context/It blocks**
- **Write comprehensive assertions with Gomega matchers**

## Communication Guidelines
- **Always be modest** about performance and capabilities
- **Emphasize exploration** of DCB concepts, not production readiness
- **Be honest about limitations** and areas that need improvement
- **Avoid overstating** performance claims or system capabilities
- **Present this as a learning project** rather than a finished solution
- **Acknowledge that DCB is still evolving** and we're exploring its application

## Command Execution Terminology
- **NEVER say "Convert commands to events"** - this is misleading and incorrect
- **ALWAYS say "Execute command handlers to generate events"** or similar
- **Emphasize that command handlers apply business logic** to decide what events to create
- **Clarify there is no automatic conversion** - it's a deliberate business decision
- **The command handler is where business logic lives** and decides what events to create
- **Use phrases like**:
  - "Execute command logic to generate events"
  - "Command handler returns events"
  - "Process command and create events"
  - "Command execution produces events"
  - "Handler applies business logic and generates events"

## Critical Approval Requirements
- **ALL changes to `docker-entrypoint-initdb.d/schema.sql` require user approval**
- **ALL changes to core API in `pkg/dcb/` require user approval**
- **Database schema changes must be reviewed before implementation**
- **API breaking changes must be explicitly approved**
- **Never modify production database schema without approval** 

# Large File Policy
- **Never commit large binary files or build artifacts** (e.g., Go binaries, SQLite databases, large JSON/CSV, or benchmark result files) to git. These files bloat the repository, slow down operations, and make collaboration difficult.
- **Keep git history clean**: If large files are accidentally committed, use tools like `git filter-repo` to remove them from the entire history.
- **Always add large files and build artifacts to `.gitignore`** to prevent accidental commits.
- **Review `.gitignore` regularly** to ensure new large files or patterns are excluded.
- **If in doubt, ask before committing any file over 1MB**. 

# Cursor Project Rule: Event Tag Array Contract

- The contract for event tags in batch appends is:
  - Each event's tags are passed as a single Postgres array-literal string (e.g., '{"key1:value1","key2:value2"}').
  - The Go code (using pgx) must pass a []string where each element is a valid array-literal string for one event's tags.
  - The SQL function must accept TEXT[] (not TEXT[][]), and cast each element to TEXT[] inside the function.
  - The function must process, filter, and reconstruct tags as a valid array-literal string for each event, not as a comma-separated string or a nested array.

- Never require Go code to pass [][]string or Postgres text[][] for tags. Always use the array-literal string contract. 

# Performance Documentation Format Rules

## Mandatory Performance Report Structure
**ALL performance documentation MUST follow this exact table-based format:**

### Required Sections (in order)
1. **Performance Overview** - Brief one-line summary with key metrics
2. **Environment Setup** - Table comparing different environments
3. **Performance Comparison** - Main comparison table with all operations and datasets
4. **Dataset Performance** - Detailed performance by environment and dataset
5. **Dataset Sizes** - Dataset dimensions table
6. **Concurrency Performance** - Multi-user scaling table
7. **Resource Usage** - Memory and allocation table

### Table Format Requirements
- **NO bullet points or free-form text** - Only structured tables
- **Consistent column headers** across all tables
- **Dataset column** in all performance tables
- **Performance ratios** calculated and displayed
- **Units clearly specified** (ops/sec, KB, MB, etc.)
- **ALL tables MUST be sorted by Throughput (ops/sec) in descending order** - Highest throughput first
- **CRITICAL: Every performance table must be sorted by throughput column** - No exceptions
- **Table sorting validation**: Before committing, verify all tables are sorted by throughput descending

### Performance Comparison Table Structure

### Performance Comparison Table Structure
```markdown
| Operation | Dataset | Environment A | Environment B | Performance Gain |
|-----------|---------|---------------|---------------|------------------|
| **OperationName** | Tiny | 100 ops/sec | 1000 ops/sec | **10x faster** |
| **OperationName** | Small | 100 ops/sec | 1000 ops/sec | **10x faster** |
| **OperationName** | Medium | 100 ops/sec | 1000 ops/sec | **10x faster** |
```

### Dataset Performance Table Structure
```markdown
| Environment | Dataset | Operation A (ops/sec) | Operation B (ops/sec) | Operation C (ops/sec) |
|-------------|---------|----------------------|----------------------|----------------------|
| **Environment A** | Tiny | 1000 | 100 | 5000 |
| **Environment A** | Small | 1000 | 100 | 5000 |
| **Environment A** | Medium | 1000 | 100 | 5000 |
```

### Dataset Sizes Table Structure
```markdown
| Dataset | Dimension A | Dimension B | Dimension C |
|---------|-------------|-------------|-------------|
| **Tiny** | 5 | 10 | 20 |
| **Small** | 500 | 5000 | 25000 |
| **Medium** | 1000 | 10000 | 50000 |
```

### Concurrency Performance Table Structure
```markdown
| Concurrency Level | Dataset | Environment A | Environment B | Performance Ratio |
|------------------|---------|---------------|---------------|-------------------|
| **1 User** | Small | 1000 ops/sec | 100 ops/sec | **10x faster** |
| **10 Users** | Small | 500 ops/sec | 50 ops/sec | **10x faster** |
| **100 Users** | Medium | 100 ops/sec | 10 ops/sec | **10x faster** |
```

### Resource Usage Table Structure
```markdown
| Concurrency Level | Dataset | Memory | Allocations |
|------------------|---------|--------|-------------|
| **1 User** | Small | 1KB | 25 |
| **10 Users** | Small | 10KB | 250 |
| **100 Users** | Medium | 100KB | 2500 |
```

## Performance Documentation Content Rules

### Operation Naming Conventions
- **AppendSingle** - Simple single event append
- **AppendRealistic** - Batch append with realistic data
- **AppendIf_NoConflict** - Conditional append (success scenario)
- **AppendIf_WithConflict** - Conditional append (failure scenario)
- **Projection** - State reconstruction from events
- **Read** - Query operations
- **ReadChannel** - Streaming read operations

### Dataset Naming Conventions
- **Tiny** - Minimal dataset for quick testing
- **Small** - Balanced dataset for development
- **Medium** - Larger dataset for production planning

### Performance Metrics Format
- **Throughput**: Always in ops/sec (operations per second)
- **Latency**: In milliseconds (ms) or nanoseconds (ns)
- **Memory**: In KB or MB with clear units
- **Allocations**: Number of allocations per operation
- **Performance Gain**: Always show as "Xx faster" or "Xx slower"

### Objectivity Requirements
- **NO subjective opinions** - Only factual data
- **NO performance recommendations** - Only measurable metrics
- **NO qualitative descriptions** - Only quantitative results
- **NO redundant information** - Each table should add unique value
- **NO broken links** - Only reference existing files

### Consistency Requirements
- **Same dataset sizes** across all tables
- **Same environment names** across all tables
- **Same operation names** across all tables
- **Same performance metrics** across all tables
- **Same table structure** for similar data types

## Performance Documentation Validation Checklist
Before considering any performance documentation complete:
- [ ] All sections use table format (no bullet points)
- [ ] Dataset column present in all performance tables
- [ ] Performance ratios calculated and displayed
- [ ] Units clearly specified for all metrics
- [ ] No subjective opinions or recommendations
- [ ] No broken links or references to non-existent files
- [ ] Consistent naming across all tables
- [ ] All datasets (Tiny, Small, Medium) represented
- [ ] All environments compared consistently
- [ ] Resource usage data included
- [ ] Concurrency scaling data included

## Standardized Performance Documentation Format

### Mandatory Table Structure
**ALL performance documentation MUST follow this standardized format:**

### Append Performance Table Structure
```markdown
## Append Performance

**Append Operations Details**:
- **Operation**: Simple event append operations
- **Scenario**: Basic event writing without conditions or business logic
- **Events**: Single event (1) or batch (100 events)
- **Model**: Generic test events with simple JSON data

| Dataset | Concurrency | Events | Throughput (ops/sec) | Latency (ns/op) | Memory (B/op) | Allocations |
|---------|-------------|--------|---------------------|-----------------|---------------|-------------|
| Tiny | 1 | 1 | 4,245 | 235,601 | 1,884 | 56 |
| Small | 1 | 1 | 3,821 | 261,668 | 1,888 | 56 |
| Medium | 1 | 1 | 4,199 | 238,131 | 1,882 | 56 |
| Tiny | 10 | 1 | 1,166 | 857,995 | 17,559 | 523 |
| Small | 10 | 1 | 1,160 | 861,590 | 17,554 | 523 |
| Medium | 10 | 1 | 1,206 | 829,250 | 17,548 | 523 |
| Tiny | 100 | 1 | 142 | 7,042,692 | 183,156 | 5,285 |
| Small | 100 | 1 | 147 | 6,808,841 | 182,705 | 5,275 |
| Medium | 100 | 1 | 130 | 7,717,958 | 182,656 | 5,277 |
| Tiny | 1 | 100 | 476 | 2,098,884 | 211,665 | 2,054 |
| Small | 1 | 100 | 522 | 1,914,980 | 211,276 | 2,053 |
| Medium | 1 | 100 | 678 | 1,474,140 | 211,359 | 2,053 |
| Tiny | 10 | 100 | 92 | 10,822,192 | 2,097,196 | 20,508 |
| Small | 10 | 100 | 92 | 10,233,074 | 2,095,527 | 20,500 |
| Medium | 10 | 100 | 101 | 9,910,265 | 2,094,603 | 20,491 |
| Tiny | 100 | 100 | 9 | 114,265,729 | 20,965,165 | 205,137 |
| Small | 100 | 100 | 9 | 107,913,216 | 20,962,283 | 205,131 |
| Medium | 100 | 100 | 8 | 117,799,050 | 20,956,685 | 205,081 |
```

### AppendIf Performance Table Structure (No Conflict)
```markdown
## AppendIf Performance (No Conflict)

**AppendIf No Conflict Details**:
- **Attempted Events**: Number of events AppendIf operation tries to append (1 or 100 events per operation)
- **Actual Events**: Number of events successfully appended (1 or 100 events)
- **Past Events**: Number of existing events in database before benchmark (100 events for all scenarios)
- **Conflict Events**: 0 (no conflicts exist)

| Dataset | Concurrency | Attempted Events | Throughput (ops/sec) | Latency (ns/op) | Memory (B/op) | Allocations |
|---------|-------------|------------------|---------------------|-----------------|---------------|-------------|
| Tiny | 1 | 1 | 930 | 1,075,000 | 4,495 | 95 |
| Small | 1 | 1 | 669 | 1,495,000 | 4,488 | 95 |
| Medium | 1 | 1 | 1,432 | 698,000 | 4,476 | 95 |
| Tiny | 10 | 1 | 430 | 2,325,000 | 43,476 | 919 |
| Small | 10 | 1 | 1,066 | 938,000 | 43,475 | 922 |
| Medium | 10 | 1 | 608 | 1,645,000 | 43,448 | 920 |
| Tiny | 100 | 1 | 46 | 21,700,000 | 443,743 | 9,277 |
| Small | 100 | 1 | 86 | 11,600,000 | 441,366 | 9,265 |
| Medium | 100 | 1 | 58 | 17,200,000 | 441,418 | 9,264 |
| Tiny | 1 | 100 | 1,537 | 650,000 | 215,033 | 2,096 |
| Small | 1 | 100 | 730 | 1,370,000 | 213,939 | 2,093 |
| Medium | 1 | 100 | 772 | 1,295,000 | 213,828 | 2,092 |
| Tiny | 10 | 100 | 187 | 5,350,000 | 2,139,663 | 20,925 |
| Small | 10 | 100 | 176 | 5,680,000 | 2,136,595 | 20,905 |
| Medium | 10 | 100 | 186 | 5,380,000 | 2,135,081 | 20,893 |
| Tiny | 100 | 100 | 18 | 55,600,000 | 21,367,125 | 209,183 |
| Small | 100 | 100 | 24 | 41,700,000 | 21,366,958 | 209,105 |
| Medium | 100 | 100 | 25 | 40,000,000 | 21,361,626 | 209,068 |
```

### AppendIf Performance Table Structure (With Conflict)
```markdown
## AppendIf Performance (With Conflict)

**AppendIf With Conflict Details**:
- **Attempted Events**: Number of events AppendIf operation tries to append (1 or 100 events per operation)
- **Actual Events**: Number of events successfully appended (0 - all operations fail due to conflicts)
- **Past Events**: Number of existing events in database before benchmark (100 events for all scenarios)
- **Conflict Events**: Number of conflicting events created before AppendIf (1, 10, or 100 events, matching concurrency level)

| Dataset | Concurrency | Attempted Events | Conflict Events | Throughput (ops/sec) | Latency (ns/op) | Memory (B/op) | Allocations |
|---------|-------------|------------------|-----------------|---------------------|-----------------|---------------|-------------|
| Tiny | 1 | 1 | 1 | 100 | 10,000,000 | 5,909 | 144 |
| Small | 1 | 1 | 1 | 100 | 10,000,000 | 5,880 | 144 |
| Medium | 1 | 1 | 1 | 169 | 5,920,000 | 5,870 | 144 |
| Tiny | 10 | 1 | 10 | 16 | 62,500,000 | 57,906 | 1,411 |
| Small | 10 | 1 | 10 | 109 | 9,170,000 | 57,240 | 1,405 |
| Medium | 10 | 1 | 10 | 15 | 66,700,000 | 57,918 | 1,410 |
| Tiny | 100 | 1 | 100 | 13 | 76,900,000 | 585,352 | 14,188 |
| Small | 100 | 1 | 100 | 26 | 38,500,000 | 581,756 | 14,175 |
| Medium | 100 | 1 | 100 | 13 | 76,900,000 | 584,568 | 14,176 |
| Tiny | 1 | 100 | 1 | 16 | 62,500,000 | 213,810 | 2,143 |
| Small | 1 | 100 | 1 | 18 | 55,600,000 | 213,323 | 2,141 |
| Medium | 1 | 100 | 1 | 139 | 7,190,000 | 214,816 | 2,140 |
| Tiny | 10 | 100 | 10 | 16 | 62,500,000 | 2,133,544 | 21,400 |
| Small | 10 | 100 | 10 | 18 | 55,600,000 | 2,132,702 | 21,380 |
| Medium | 10 | 100 | 10 | 105 | 9,520,000 | 2,146,011 | 21,371 |
| Tiny | 100 | 100 | 100 | 10 | 100,000,000 | 21,473,610 | 213,918 |
| Small | 100 | 100 | 100 | 8 | 125,000,000 | 21,465,429 | 213,849 |
| Medium | 100 | 100 | 100 | 19 | 52,600,000 | 21,492,126 | 213,877 |
```

### Read and Projection Performance Table Structure
```markdown
## Read and Projection Performance

| Operation | Dataset | Concurrency | Events | Throughput (ops/sec) | Latency (ns/op) | Memory (B/op) | Allocations |
|-----------|---------|-------------|--------|---------------------|-----------------|---------------|-------------|
| **Read_Single** | Tiny | 1 | - | 123 | 8,130,000 | 2,106,756 | 253,425 |
| **Read_Single** | Small | 1 | - | 294 | 3,400,000 | 1,024,370 | 131,363 |
| **Read_Single** | Medium | 1 | - | 328 | 3,050,000 | 1,024,348 | 131,363 |
| **Read_Batch** | Tiny | 1 | - | 49,030 | 20,400 | 988 | 21 |
| **Read_Batch** | Small | 1 | - | 6,724 | 148,800 | 989 | 21 |
| **Read_Batch** | Medium | 1 | - | 7,898 | 126,600 | 989 | 21 |
| **Projection** | Tiny | 1 | - | 36,338 | 27,500 | 2,037 | 37 |
| **Projection** | Small | 1 | - | 33,769 | 29,600 | 2,036 | 37 |
| **Projection** | Medium | 1 | - | 6,811 | 146,800 | 2,036 | 37 |
```

### Mandatory Format Requirements
**ALL performance documentation MUST follow these rules:**

1. **Table Structure**:
   - **NO "Operation" column** for Append and AppendIf tables (operation identified by table title)
   - **KEEP "Operation" column** for Read and Projection tables (different operation types)
   - **Dataset column** in all tables
   - **Concurrency column** in all tables
   - **Events column** in all tables (1, 100, or "-" for read/projection)

2. **Event Count Standardization**:
   - **Append**: Use 1 or 100 events only (no mixed ranges like "1-12")
   - **AppendIf**: Use 1 or 100 attempted events only
   - **Read/Projection**: Use "-" in Events column

3. **Concurrency Levels**:
   - **1 user** (single-threaded)
   - **10 users** (concurrent)
   - **100 users** (high concurrency)

4. **Dataset Coverage**:
   - **Tiny** dataset
   - **Small** dataset
   - **Medium** dataset

5. **Description Requirements**:
   - **Detailed operation descriptions** before each table
   - **Clear explanation** of event counts and scenarios
   - **No subjective opinions** - only factual data

6. **Data Extraction**:
   - **Keep detailed operation names** in benchmark code for easy extraction
   - **Use simplified table presentation** in documentation
   - **Maintain correlation** between benchmark results and table cells

7. **Table Sorting**:
   - **ALL performance tables MUST be sorted by Throughput (ops/sec) in descending order**
   - **Highest throughput operations first**
   - **Consistent sorting across all tables**
   - **No exceptions to this rule**

### Validation Checklist for New Performance Documentation
Before considering any performance documentation complete:
- [ ] Follows standardized table structure (no Operation column for Append/AppendIf)
- [ ] Uses standardized event counts (1 or 100 events only)
- [ ] Includes all concurrency levels (1, 10, 100 users)
- [ ] Covers all datasets (Tiny, Small, Medium)
- [ ] Has detailed operation descriptions
- [ ] No duplicate rows or inconsistent data
- [ ] Maintains data extraction capability
- [ ] Uses objective, factual language only 

# Projection Benchmarking Best Practices

## Core API Concurrency Architecture
The EventStore has **built-in goroutine limits** that must be respected:

### Built-in Concurrency Controls
- **`MaxConcurrentProjections`** (default: 50) - Limits concurrent projection operations
- **`MaxProjectionGoroutines`** (default: 100) - Limits internal goroutines per projection
- **`StreamBuffer`** - Controls channel buffer size for streaming operations
- **`projectionSemaphore`** - Internal semaphore for concurrency control

### Mandatory Benchmarking Rules
**ALWAYS use the core API's built-in concurrency controls:**

```go
// CORRECT: Let the API handle concurrency
func BenchmarkProjectStreamConcurrent(b *testing.B, benchCtx *BenchmarkContext, goroutines int) {
    for i := 0; i < b.N; i++ {
        // Use Go 1.25 WaitGroup.Go() for concurrent operations
        var wg sync.WaitGroup
        results := make(chan error, goroutines)
        
        for j := 0; j < goroutines; j++ {
            wg.Go(func() {
                // Let the core API handle goroutine limits and streaming
                stateChan, conditionChan, err := benchCtx.Store.ProjectStream(ctx, projectors, nil)
                if err != nil {
                    results <- fmt.Errorf("concurrent ProjectStream failed: %v", err)
                    return
                }
                
                // Consume from channels (API handles concurrency internally)
                select {
                case state := <-stateChan:
                    _ = state // Use state to prevent optimization
                case <-time.After(5 * time.Second):
                    results <- fmt.Errorf("ProjectStream timeout")
                    return
                }
                
                results <- nil
            })
        }
        
        wg.Wait()
        close(results)
    }
}
```

### Go 1.25 Concurrency Features
**ALWAYS use Go 1.25 concurrency features:**

1. **`WaitGroup.Go()`** instead of manual `wg.Add(1); go func()`
2. **`context.WithTimeoutCause()`** for better error handling
3. **Let the API handle streaming** - don't implement your own

### What NOT to Do
**NEVER implement manual concurrency controls:**

```go
// WRONG: Don't add your own goroutine limits
semaphore := make(chan struct{}, goroutines) // Don't do this
wg.Add(1)
go func() {
    defer wg.Done()
    semaphore <- struct{}{} // Don't do this
    defer func() { <-semaphore }() // Don't do this
    // ...
}(j)
```

### Performance Expectations
**Realistic projection performance:**
- **Single-threaded**: ~10-15 ops/sec (full scan)
- **Concurrent (1 user)**: ~8-10 ops/sec (with API overhead)
- **Concurrent (10+ users)**: ~0.5-2 ops/sec (resource contention)
- **Memory usage**: ~200-500MB per operation (large event streams)
- **Allocations**: ~4-11M per operation (complex state reconstruction)

### Benchmark Architecture
**The correct approach:**
1. **Use core API's built-in limits** - don't fight the architecture
2. **Let API handle streaming** - it's designed for it
3. **Use Go 1.25 features** - cleaner concurrency
4. **Respect the API design** - don't add manual controls
5. **Measure realistic performance** - not artificial optimizations

### Common Mistakes to Avoid
- **Adding manual goroutine limits** when API already has them
- **Implementing custom streaming** when API handles it
- **Fighting the API architecture** instead of working with it
- **Expecting unrealistic performance** for large event streams
- **Not using Go 1.25 concurrency features** 

# Benchmark Testing Rules

## Mandatory Benchmark Testing Strategy

### Two-Phase Testing Approach
**ALWAYS follow this two-phase testing strategy:**

1. **Phase 1: Quick Validation with Tiny Dataset**
   ```bash
   go test -bench="BenchmarkAppend_Tiny_Realistic" -benchtime=1s -timeout=30s .
   ```
   - **Purpose**: Quick validation that all realistic benchmarks work
   - **Dataset**: Tiny (5 courses, 10 students, 20 enrollments) - fastest execution
   - **Verify**: All operations (Append, AppendIf, Project, ProjectStream, ProjectionLimits)
   - **Timeout**: 30 seconds maximum

2. **Phase 2: Complete Benchmark Suite with All Datasets**
   ```bash
   go test -bench="BenchmarkAppend_.*_Realistic" -benchtime=2s -timeout=120s .
   ```
   - **Purpose**: Generate comprehensive performance data for documentation
   - **Datasets**: Tiny, Small, Medium (all three datasets)
   - **Operations**: Complete realistic benchmark suite
   - **Timeout**: 120 seconds maximum

### Benchmark Testing Requirements
- **ALWAYS test with Tiny dataset first** for quick validation
- **ALWAYS run all three datasets** (Tiny, Small, Medium) for complete data
- **ALWAYS use realistic benchmarks** for documentation (not generic)
- **ALWAYS include all operations**: Append, AppendIf, Project, ProjectStream, ProjectionLimits
- **ALWAYS verify success/limit rates** for ProjectionLimits benchmarks

### Benchmark Data Collection
- **Save results to files**: `/tmp/tiny_realistic_benchmarks.txt`, `/tmp/small_realistic_benchmarks.txt`, `/tmp/medium_realistic_benchmarks.txt`
- **Extract key metrics**: ns/op, B/op, allocs/op, success_rate, limit_exceeded_rate
- **Validate all operations work** before proceeding to documentation
- **Use results for performance documentation** updates

### Why This Approach?
- **Tiny dataset first** = quick validation (5 courses, 10 students, 20 enrollments)
- **All datasets** = complete performance picture across different data sizes
- **All benchmarks** = comprehensive coverage of all EventStore operations
- **Realistic events** = business-focused performance data for documentation
- **Efficient workflow** = validate first, then collect complete data