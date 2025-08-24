# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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
