# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed
- **Performance Documentation Format**: Fixed performance table formatting and units
  - **Latency Units**: Converted from nanoseconds to milliseconds (divided by 1,000,000) for better readability
  - **Memory Units**: Converted from bytes to KB (divided by 1,024) for more practical measurements
  - **Table Sorting**: Fixed ALL performance tables to sort by Throughput (ops/sec) in descending order
  - **Consistency**: Applied formatting fixes to all performance documentation files
  - **Readability**: Improved table readability with more practical units and proper sorting
- **Benchmark Standardization**: Standardized benchmark concurrency levels and event counts for consistency
  - **Concurrency Levels**: Changed from `1, 10, 25` to `1, 100` users for all benchmark operations
  - **Attempted Events**: Changed from `1, 100` to `1, 10` events per operation for Append and AppendIf benchmarks
  - **Conflict Events**: Standardized to `1` event for all AppendIf conflict scenarios (removed variable conflict counts)
  - **Performance Documentation**: Updated all performance tables with new standardized benchmark results
  - **Benchmark Code**: Modified `BenchmarkAppendIfConcurrent` to create exactly 1 conflict event instead of variable counts
  - **Documentation**: Updated performance documentation to reflect new benchmark structure and results

### Added
- **Test Coverage Improvements**: Added comprehensive tests for previously untested functions
  - Added tests for `IsTableStructureError` and `GetTableStructureError` error detection functions
  - Added tests for `AsConcurrencyError`, `AsResourceError`, `AsTableStructureError` error extraction aliases
  - Added tests for `NewEventStoreWithConfig` constructor function existence and configuration validation
  - Added comprehensive tests for `NewCommand` constructor with various parameter combinations
  - Improved test coverage for error handling and constructor functions without changing existing code
        - **Performance Documentation Updates**: Updated performance tables with fresh benchmark data
          - Updated Append Performance table with September 7th benchmark results
          - Updated AppendIf No Conflict Performance table with latest throughput and latency metrics
          - Updated AppendIf With Conflict Performance table with current performance data
          - Updated Projection Performance table with recent benchmark results
          - All performance tables maintain standardized format with factual data only
          - Data extracted from `go_benchmarks_20250907_175510.txt` benchmark results
          - **Fixed throughput column ordering**: All tables now sorted by throughput (ops/sec) in descending order for better readability
          - **Added column explanations**: Detailed explanations for all table columns to improve understanding
            - Dataset sizes: Tiny (5 courses/10 students), Small (1K courses/10K students), Medium (1K courses/10K students)
            - Performance metrics: Throughput (higher better), Latency (lower better), Memory (lower better), Allocations (lower better)
            - Operation-specific details: Events per operation, concurrency levels, conflict scenarios
          - **Docker Performance Documentation**: Applied same improvements to Docker PostgreSQL performance tables
            - Added comprehensive column explanations for all Docker benchmark tables
            - Fixed throughput column ordering: All tables sorted by throughput (ops/sec) in descending order
            - Consistent documentation format between local and Docker performance results
            - Improved readability and understanding of Docker benchmark results
          - **Fixed Docker Projection Performance Table**: Added missing concurrency levels (10 and 25 users)
            - Docker benchmarks were running with concurrency 1, 10, and 25 but documentation only showed concurrency 1
            - Added complete projection performance data for all concurrency levels
            - Docker projection performance: 2,555 ops/sec (1 user) → 620 ops/sec (25 users)
            - Consistent with local performance documentation format

### Fixed
- **Multi-Item AppendCondition Bug**: Fixed `extractConditionPrimitives` function to process all query items instead of just the first one
  - **Issue**: Function only extracted event types and tags from the first item of `failIfEventsMatch` condition
  - **Impact**: Multi-item `AppendCondition` queries using OR logic were not fully evaluated, causing incorrect event appends
  - **Fix**: Modified function to iterate through all query items and collect event types and tags from each item
  - **Testing**: All 207 tests pass, confirming the fix works correctly without regressions

### Updated
- **Performance Documentation**: Updated performance tables with fresh benchmark data from September 16th, 2025
  - **Append Performance**: Updated with latest throughput and latency metrics across all datasets (Tiny, Small, Medium)
  - **AppendIf No Conflict Performance**: Updated with current performance data for conditional append operations
  - **AppendIf With Conflict Performance**: Updated with latest conflict scenario performance metrics
  - **Projection Performance**: Updated with fresh state reconstruction performance data
  - **Read Performance**: Updated with latest query and streaming performance metrics
  - **ProjectionLimits Performance**: Updated with current concurrency limit testing results
  - **Data Source**: All performance data extracted from comprehensive benchmark suite run on September 16th, 2025
  - **Table Formatting**: Maintained standardized table format with throughput sorting and detailed column explanations
- **Local PostgreSQL Performance**: Added comprehensive local PostgreSQL performance data showing dramatic improvements
  - **Performance Comparison**: Added detailed comparison table between Local PostgreSQL and Docker PostgreSQL
  - **Performance Gains**: Local PostgreSQL shows 4-6x faster throughput across all operations
  - **Real-world Performance**: Local PostgreSQL provides production-ready performance characteristics
  - **Benchmark Results**: Updated all performance tables with local PostgreSQL benchmark data
  - **Performance Benefits**: Documented key advantages of local PostgreSQL over Docker PostgreSQL

### Changed
- **Performance Documentation Format**: Fixed performance table formatting and units
  - **Latency Units**: Converted from nanoseconds to milliseconds (divided by 1,000,000) for better readability
  - **Memory Units**: Converted from bytes to KB (divided by 1,024) for more practical measurements
  - **Table Sorting**: Fixed ALL performance tables to sort by Throughput (ops/sec) in descending order
  - **Consistency**: Applied formatting fixes to all performance documentation files
  - **Readability**: Improved table readability with more practical units and proper sorting
- **Benchmark Standardization**: Standardized benchmark concurrency levels and event counts for consistency
  - **Concurrency Levels**: Changed from `1, 10, 25` to `1, 100` users for all benchmark operations
  - **Attempted Events**: Changed from `1, 100` to `1, 10` events per operation for Append and AppendIf benchmarks
  - **Conflict Events**: Standardized to `1` event for all AppendIf conflict scenarios (removed variable conflict counts)
  - **Performance Documentation**: Updated all performance tables with new standardized benchmark results
  - **Benchmark Code**: Modified `BenchmarkAppendIfConcurrent` to create exactly 1 conflict event instead of variable counts
  - **Documentation**: Updated performance documentation to reflect new benchmark structure and results

### Added
- **Test Coverage Improvements**: Added comprehensive tests for previously untested functions
  - Added tests for `IsTableStructureError` and `GetTableStructureError` error detection functions
  - Added tests for `AsConcurrencyError`, `AsResourceError`, `AsTableStructureError` error extraction aliases
  - Added tests for `NewEventStoreWithConfig` constructor function existence and configuration validation
  - Added comprehensive tests for `NewCommand` constructor with various parameter combinations
  - Improved test coverage for error handling and constructor functions without changing existing code
        - **Performance Documentation Updates**: Updated performance tables with fresh benchmark data
          - Updated Append Performance table with September 7th benchmark results
          - Updated AppendIf No Conflict Performance table with latest throughput and latency metrics
          - Updated AppendIf With Conflict Performance table with current performance data
          - Updated Projection Performance table with recent benchmark results
          - All performance tables maintain standardized format with factual data only
          - Data extracted from `go_benchmarks_20250907_175510.txt` benchmark results
          - **Fixed throughput column ordering**: All tables now sorted by throughput (ops/sec) in descending order for better readability
          - **Added column explanations**: Detailed explanations for all table columns to improve understanding
            - Dataset sizes: Tiny (5 courses/10 students), Small (1K courses/10K students), Medium (1K courses/10K students)
            - Performance metrics: Throughput (higher better), Latency (lower better), Memory (lower better), Allocations (lower better)
            - Operation-specific details: Events per operation, concurrency levels, conflict scenarios
          - **Docker Performance Documentation**: Applied same improvements to Docker PostgreSQL performance tables
            - Added comprehensive column explanations for all Docker benchmark tables
            - Fixed throughput column ordering: All tables sorted by throughput (ops/sec) in descending order
            - Consistent documentation format between local and Docker performance results
            - Improved readability and understanding of Docker benchmark results
          - **Fixed Docker Projection Performance Table**: Added missing concurrency levels (10 and 25 users)
            - Docker benchmarks were running with concurrency 1, 10, and 25 but documentation only showed concurrency 1
            - Added complete projection performance data for all concurrency levels
            - Docker projection performance: 2,555 ops/sec (1 user) → 620 ops/sec (25 users)
            - Consistent with local performance documentation format

### Fixed
- **Performance Documentation Accuracy**: Corrected incorrect benchmark numbers in performance documentation
  - Fixed misleading claim that "Complex Queries are 7.6x faster than Simple Read" in Small dataset
  - Updated actual benchmark results: Simple Read (337 ops/sec) vs Complex Business Workflow (361 ops/sec)
  - Clarified that "Complex Queries" is actually testing a 4-step business workflow, not just queries
  - Performance reality: Complex operations are slower (1.1x) due to multiple sequential operations

### Added
- **Enhanced EventStoreConfig**: Added logical grouping for append and query operations
  - Improved organization with clear sections for append vs query configuration
  - Clean, focused configuration with only working, implemented fields
- **Comprehensive Benchmark Documentation**: Updated docs/benchmarks.md with complete benchmark inventory
  - Documented all 68 Go benchmarks with detailed categorization
  - Added benchmark categories: Core Operations (47), Enhanced Business Scenarios (6), Core Functions (13), Framework (2)
  - Detailed breakdown of append, read, projection, and business scenario benchmarks
  - Added dataset integration details and benchmark execution instructions
  - Enhanced use case descriptions for all benchmark types
  - **Added concurrent user metrics**: Documented 10 concurrent users (338 ops/sec) and 50 concurrent users (77 ops/sec) performance results
  - **Enhanced concurrent performance analysis**: Shows real-world scaling characteristics and contention patterns
  - **Added realistic benchmark scenarios**: Implemented benchmarks for common real-world usage (1-12 events per operation)
  - **SQLite caching optimization**: Pre-generated benchmark data eliminates runtime string formatting overhead
  - **Real-world validation**: Benchmarks now reflect actual business usage patterns, not artificial stress tests
- **Comprehensive Concurrent Projection Benchmarks**: Implemented missing projection benchmark functions
  - Added `BenchmarkProject` and `BenchmarkProjectStream` functions that were previously called but not defined
  - Implemented `LoadDatasetIntoStore` function for proper test data loading into PostgreSQL
  - Added concurrent projection testing for 1, 10, and 100 goroutines
  - Test both synchronous (`Project`) and asynchronous (`ProjectStream`) projection methods
  - Optimized benchmarks for speed using tiny dataset and reduced timeouts
  - Show performance scaling with concurrency and goroutine contention patterns

### Changed
- **Documentation Improvements**: 
  - Rewrote overview.md for clarity and conciseness
  - Removed verbose structs vs maps comparison (not relevant)
  - Fixed logical flow: Core Concepts → State Projectors → Command Handlers
  - Replaced undefined UserState with generic map[string]any in examples
  - Updated all documentation links to use correct relative paths
  - Corrected overstatements about library capabilities (emphasized exploration status)
- **Benchmark Documentation**: Updated docs/benchmarks.md with current performance results and comparison disclaimer
  - Added latest Go library benchmark results (2025-08-24)
  - Added latest web app benchmark results (2025-08-24)
  - Fixed AppendIf benchmark status from "failed" to "working successfully"
  - Added clear disclaimer: Go vs Web benchmarks should NOT be compared directly
  - Explained why 700x performance difference is expected and normal
  - Separated use case recommendations for each benchmark type
  - Clarified that both benchmark types are valuable for different purposes
- **Performance Documentation**: Fixed misleading performance claims in docs/performance-comparison.md
  - Removed incorrect "15x slower" claims that didn't match reality
  - Added same disclaimer about not comparing Go vs Web performance
  - Updated with current benchmark numbers (700x difference is normal)
  - Explained why performance differences are expected for different use cases
- **README Updates**: Added performance disclaimer to prevent misleading comparisons
  - Added warning about Go vs Web benchmark comparisons
  - Clarified that 700x performance difference is expected and normal
- **Enhanced Benchmarking**: Implemented comprehensive business scenario benchmarks
  - Added complex business workflow benchmarks (user registration, course enrollment)
  - Added concurrent operation benchmarks (10 concurrent users)
  - Added mixed operation benchmarks (append + query + projection)
  - Added business rule validation benchmarks (DCB conditions)
  - Added load pattern benchmarks (burst traffic, sustained load)
  - Enhanced benchmark runner with statistical analysis (count=3)
  - Added new Makefile targets: `benchmark-go-enhanced`, `benchmark-go-all`
  - Updated documentation with enhanced benchmark capabilities
- **Performance Documentation Restructuring**: Reorganized performance documentation for better clarity
  - Restructured `docs/performance.md` as overview with links to dataset-specific pages
  - Created `docs/performance-tiny.md` for Tiny dataset detailed results (5 courses, 10 students, 17 enrollments)
  - Created `docs/performance-small.md` for Small dataset detailed results (1,000 courses, 10,000 students, 49,871 enrollments)
  - Added navigation links between pages for easy browsing
  - Maintained clear separation between overview and detailed results
  - Organized performance tables by dataset size for clearer comparison
- **CommandExecutor Documentation Clarity**: Improved documentation by removing jargon and invented terms
  - Replaced "CommandExecutor pattern" with clear explanation of what it does
  - Replaced "Atomic command execution" with "database transactions for consistency"
  - Replaced "Atomicity" with "Data consistency using database transactions"
  - Clarified that CommandExecutor helps organize business logic execution
  - Made the purpose and benefits more concrete and understandable
  - Removed invented terms that don't clearly explain functionality

### Fixed
- **Interface Implementation Consistency**: Added missing marker methods
  - `command.isCommand()` - implements Command interface
  - `commandExecutor.isCommandExecutor()` - implements CommandExecutor interface  
  - `eventStore.isEventStore()` - implements EventStore interface
- **Documentation Links**: Fixed broken relative paths in README.md, quick-start.md, and getting-started.md
- **Test Organization**: Resolved duplicate test suite conflicts by properly separating internal vs external tests
- **AppendIf Benchmark**: Fixed endpoint configuration issue in k6 benchmark script
  - Updated script to use correct `/benchmark-data` endpoint instead of non-existent `/load-test-data`
  - Benchmark now runs successfully with 31.8 req/s sustained throughput
- **Broken Projection Benchmarks**: Fixed missing benchmark functions and data loading issues
  - Implemented missing `BenchmarkProject` and `BenchmarkProjectStream` functions
  - Added `LoadDatasetIntoStore` function to properly load test data into PostgreSQL
  - Fixed benchmarks that were calling undefined functions
  - Ensured proper test data loading for accurate benchmark results
- **Performance Documentation Inconsistencies**: Fixed event count mismatches and table organization
  - Corrected event counts across all performance tables in `docs/performance.md`
  - Fixed "Append Operations" scenario description vs table results mismatch
  - Ensured consistent event count values (1, 10, or 100) across all tables
  - Reordered `AppendIf Operations` to be immediately after `Append Operations`
  - Replaced "Batch Size" with "Event Count" for consistency
  - Added tests to increase events consumed for Projection Operations (5, 10, 20, 50, 100, 120 events)
  - Ensured "Core Operations" table includes AppendIf
  - Added "Event Count Explanation" and "Performance Impact" sections with AppendIf positioned after Append

### Internal
- **Test Coverage Improvements**: Added high-priority internal unit tests
  - `pkg/dcb/errors_test.go` - Error handling and marker method tests
  - `pkg/dcb/constructors_test.go` - Alternative constructor and config validation tests
  - Improved coverage for error handling, alternative constructors, and configuration validation

## [0.1.0] - 2024-12-XX

### Added
- **Core DCB Library**: Initial implementation of Dynamic Consistency Boundary pattern
- **EventStore Interface**: Core API for event sourcing operations
  - `Append()` - Basic event appending
  - `AppendIf()` - Conditional event appending with DCB
  - `Query()` - Event querying with tag-based filtering
  - `QueryStream()` - Streaming event queries
  - `Project()` - State projection from events
  - `ProjectStream()` - Streaming state projections
- **CommandExecutor**: High-level API for command handling
  - `ExecuteCommand()` - Execute commands with business logic handlers
- **DCB Concurrency Control**: Event-level conflict detection and prevention
- **PostgreSQL Integration**: Production database support with optimized schema
- **SQLite Benchmark System**: Test data caching for performance testing
- **Comprehensive Testing**: Ginkgo BDD tests with Testcontainers integration

### Architecture
- **Opaque Type Pattern**: Interfaces with private concrete implementations
- **Event Sourcing**: Immutable event storage with append-only semantics
- **State Projection**: Event-driven state reconstruction
- **Tag-based Querying**: Efficient event filtering and retrieval
- **Transaction Management**: PostgreSQL transaction isolation levels

### Performance
- **Append Operations**: ~1,000 ops/s (simple), ~800 ops/s (with DCB)
- **Query Operations**: ~2,000 ops/s (tag-based filtering)
- **Projection Operations**: ~500 ops/s (state reconstruction)
- **Batch Processing**: Up to 1,000 events per batch
- **Streaming Support**: Memory-efficient large dataset handling

### Documentation
- **Comprehensive Guides**: Getting started, examples, testing, benchmarks
- **API Reference**: Complete interface documentation
- **Performance Analysis**: Benchmark results and optimization strategies
- **Best Practices**: Event design, concurrency control, error handling

---

## Version History

- **0.1.0**: Initial release with core DCB functionality
- **Unreleased**: Current development version with ongoing improvements

## Contributing

When adding new features or making significant changes, please update this changelog following the established format. Include:

- **Added**: New features
- **Changed**: Changes in existing functionality  
- **Deprecated**: Soon-to-be removed features
- **Removed**: Removed features
- **Fixed**: Bug fixes
- **Security**: Vulnerability fixes
- **Internal**: Internal changes (refactoring, tests, etc.)

## Notes

- This project is an **exploration of DCB concepts**, not a production-ready solution
- Performance claims are modest and factual based on benchmark results
- Focus is on learning and experimenting with event sourcing patterns
- All changes maintain backward compatibility where possible
