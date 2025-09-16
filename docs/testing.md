# Testing Guide

This document provides a comprehensive overview of the testing structure and organization in go-crablet.

## Test Organization

The project follows a well-organized testing structure with clear separation between internal and external tests:

### External Tests (`pkg/dcb/tests/`)
External tests consume only the public API and verify the library works correctly from a consumer's perspective.

**Test Infrastructure:**
- `setup_test.go` - Test infrastructure, database setup, helper functions, and lifecycle hooks

**Test Files:**
- `append_helpers_test.go` - Tests for append helper functions
- `batch_projection_test.go` - Tests for multiple state projection functionality
- `channel_streaming_test.go` - Tests for channel-based streaming
- `concurrency_test.go` - Tests for concurrent operations
- `constructors_test.go` - Tests for constructor functions
- `coverage_improvement_test.go` - Tests for improving code coverage
- `course_subscription_test.go` - Tests for course subscription scenarios
- `cursor_test.go` - Tests for cursor-based operations
- `errors_test.go` - Tests for error handling
- `helpers_test.go` - Tests for helper functions
- `interface_type_guards_test.go` - Tests for interface type guards
- `ordering_scenarios_test.go` - Tests for event ordering scenarios

### Internal Tests (`pkg/dcb/`)
Internal tests have access to unexported functions and test internal implementation details.

**Test Infrastructure:**
- `setup_test.go` - Test infrastructure, database setup, helper functions, and lifecycle hooks

**Test Files:**
- `z_validation_test.go` - Tests for internal validation logic

## Running Tests

### Run All Tests
```bash
# Run all tests (both internal and external)
go test ./pkg/dcb/... -v

# Run only external tests
go test ./pkg/dcb/tests/... -v

# Run only internal tests
go test ./pkg/dcb/... -v -run "TestDCB" -test.v
```

### Run Specific Test Files
```bash
# Run cursor tests
go test ./pkg/dcb/tests/... -v -run "Cursor"

# Run validation tests
go test ./pkg/dcb/... -v -run "Validation"

# Run multiple state projection tests
go test ./pkg/dcb/tests/... -v -run "Batch"
```

### Run Concurrency Tests
```bash
# Run all concurrency-related tests
go test ./pkg/dcb/tests/concurrency_test.go ./pkg/dcb/tests/setup_test.go -v

# Run DCB concurrency control tests only
go test ./pkg/dcb/tests/... -v -run "Concurrency.*DCB"



# Run the ticket booking example
go run internal/examples/ticket_booking/main.go -users 50 -seats 30
```

### Run Tests with Coverage
```bash
# Generate coverage report
go test ./pkg/dcb/... -coverprofile=coverage.out

# View coverage in browser
go tool cover -html=coverage.out
```

## Test Infrastructure

### Database Setup
Tests use PostgreSQL containers via testcontainers-go for isolated, reproducible test environments:

- **Container**: PostgreSQL 17.5-alpine
- **Schema**: Automatically loaded from `docker-entrypoint-initdb.d/schema.sql`
- **Isolation**: Each test run gets a fresh database instance
- **Cleanup**: Containers are automatically cleaned up after tests

## Performance Testing

For comprehensive performance testing and benchmarks, see the **[Performance Guide](./performance.md)**.

### Realistic Benchmark Testing

The project now uses **realistic benchmarks** that test actual business scenarios with course enrollment system data:

**Benchmark Categories:**
- **Append Operations**: Course offering and student enrollment events
- **AppendIf Operations**: Conditional enrollment with business rule validation
- **Query Operations**: Event querying with realistic tags and filters
- **QueryStream Operations**: Streaming event processing
- **Project Operations**: State reconstruction from events
- **ProjectStream Operations**: Streaming state reconstruction
- **ProjectionLimits Operations**: Concurrency limit testing

**Dataset Sizes:**
- **Tiny**: 5 courses, 10 students, 20 enrollments (quick validation)
- **Small**: 500 courses, 5,000 students, 25,000 enrollments (development)
- **Medium**: 1,000 courses, 10,000 students, 50,000 enrollments (production planning)

**Testing Strategy:**
1. **Phase 1**: Quick validation with Tiny dataset (`go test -bench="BenchmarkAppend_Tiny_Realistic" -benchtime=1s -timeout=30s .`)
2. **Phase 2**: Complete benchmark suite with all datasets (`go test -bench="BenchmarkAppend_.*_Realistic" -benchtime=2s -timeout=120s .`)

### Quick Performance Checks
```bash
# Run quick benchmarks for fast feedback
cd internal/benchmarks
go test -bench=BenchmarkAppend_Tiny_Realistic -benchtime=1s

# Run specific benchmark suites
go test -bench=BenchmarkAppend_.*_Realistic -benchtime=2s
```

### Test Lifecycle
```go
// BeforeSuite - Runs once before all tests
var _ = BeforeSuite(func() {
    // Setup database container
    // Load schema
    // Create event store instance
})

// AfterSuite - Runs once after all tests
var _ = AfterSuite(func() {
    // Cleanup database connection
    // Terminate container
})
```

### Helper Functions
Common test utilities available in both test packages:

- `toJSON(v any) []byte` - Marshal struct to JSON bytes
- `generateRandomPassword(length int) (string, error)` - Generate random passwords
- `setupPostgresContainer(ctx context.Context)` - Create test database
- `truncateEventsTable(ctx context.Context, pool *pgxpool.Pool)` - Reset events table
- `filterPsqlCommands(sql string)` - Filter psql meta-commands from schema

## Test Categories

### 1. Unit Tests
Test individual functions and methods in isolation:
- Validation logic
- Constructor functions
- Helper utilities

### 2. Integration Tests
Test interactions between components:
- Database operations
- Event store operations
- Projection functionality

### 3. End-to-End Tests
Test complete workflows:
- Course subscription scenarios
- Multiple state projection operations
- Streaming operations

### 4. Concurrency Tests
Test concurrent operations and race conditions:
- Multiple concurrent appends with DCB concurrency control
- Concurrent projections

- N-user concurrent scenarios (10+ users) to demonstrate real-world concurrency

**Key Test Files:**
- `concurrency_test.go` - Tests DCB concurrency control with N concurrent users

- `ticket_booking/main.go` - Performance demonstration of DCB concurrency control

**Test Scenarios:**
- **DCB Concurrency Control**: Uses `AppendCondition` to enforce business rules
- **N-User Testing**: Demonstrates real concurrent scenarios (10+ users) instead of just 2

## Example Demonstrations

### Course Enrollment Example
The course enrollment example demonstrates proper DCB compliance and business logic validation:

**Example Structure:**
```
internal/examples/enrollment/main.go
```

**Key Demonstrations:**
- **Course Offering**: Creating courses with proper validation
- **Student Registration**: Registering students with proper validation
- **Enrollment Completion**: Successful enrollments with business rules
- **Business Rules**: Duplicate enrollment prevention and capacity limits
- **Non-existent Courses**: Enrollments to non-existent courses (creates them automatically)
- **Sequential Enrollments**: Multiple enrollments and capacity tracking
- **Concurrency Control**: DCB compliance with `AppendCondition`

**Example Features:**
- **Flat Structure**: Single main.go file with all types and handlers
- **Comprehensive Scenarios**: All business scenarios including edge cases
- **DCB Compliance**: Uses proper `AppendCondition` for concurrency control
- **Realistic Scenarios**: Realistic course enrollment scenarios with proper validation

**Running Course Enrollment Example:**
```bash
# Run course enrollment example
go run internal/examples/enrollment/main.go
```

### Ticket Booking Example
The ticket booking example demonstrates DCB concurrency control performance:

**Usage:**
```bash
# Run with default settings (100 users, 20 seats, 2 tickets per user)
go run internal/examples/ticket_booking/main.go

# Run with custom settings
go run internal/examples/ticket_booking/main.go -users 50 -seats 30 -tickets 1
```

**What It Tests:**
- **DCB Concurrency Control**: Uses `AppendCondition` to enforce business rules
- **Performance Metrics**: Benchmarks timing and throughput metrics
- **Real-world Scenarios**: Concert ticket booking with limited seats

## Test Data Management

### Unique Test Data
Tests use unique identifiers to avoid interference:
```go
uniqueID := fmt.Sprintf("test_%d", time.Now().UnixNano())
```

### Test Isolation
Each test is isolated and doesn't depend on other tests:
- Fresh database state for each test
- Unique event IDs and tags
- Proper cleanup after each test

## Best Practices

### 1. Test Naming
- Use descriptive test names that explain the scenario
- Follow the pattern: "should [expected behavior] when [condition]"

### 2. Test Structure
- Arrange: Set up test data and conditions
- Act: Execute the operation being tested
- Assert: Verify the expected outcomes

### 3. Error Testing
- Test both success and failure scenarios
- Verify error messages and types
- Test edge cases and boundary conditions

### 4. Performance Testing
- Use realistic data sizes
- Test with concurrent operations
- Monitor resource usage

## Debugging Tests

### Enable Verbose Output
```bash
go test ./pkg/dcb/... -v -test.v
```

### Debug Database State
Use the `dumpEvents` helper function to inspect database state:
```go
dumpEvents(pool) // Prints all events in JSON format
```

### Test Isolation
If tests are interfering with each other:
1. Check for hardcoded identifiers
2. Ensure proper cleanup
3. Use unique test data

## Continuous Integration

Tests are automatically run in CI/CD pipelines:
- All tests must pass before merging
- Coverage reports are generated
- Performance benchmarks are executed

## Contributing

When adding new tests:
1. Follow the existing naming conventions
2. Use the established test infrastructure
3. Ensure proper test isolation
4. Add appropriate error testing
5. Update this documentation if needed
