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
go test ./pkg/dcb/tests/concurrency_test.go ./pkg/dcb/tests/advisory_locks_test.go ./pkg/dcb/tests/setup_test.go -v

# Run DCB concurrency control tests only
go test ./pkg/dcb/tests/... -v -run "Concurrency.*DCB"

# Run advisory locks tests only
go test ./pkg/dcb/tests/... -v -run "Advisory.*Lock"

# Run the concurrency comparison example
go run internal/examples/concurrency_comparison/main.go -users 50 -seats 30
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
- Advisory locks vs DCB concurrency control comparison
- N-user concurrent scenarios (10+ users) to demonstrate real-world concurrency

**Key Test Files:**
- `concurrency_test.go` - Tests DCB concurrency control with N concurrent users
- `advisory_locks_test.go` - Tests advisory locks with and without AppendCondition
- `concurrency_comparison/main.go` - Performance comparison between DCB and advisory locks

**Test Scenarios:**
- **DCB Concurrency Control**: Uses `AppendCondition` to enforce business rules
- **Advisory Locks**: Serialize access but don't enforce business limits without conditions
- **Both Combined**: Serialize access AND enforce business rules
- **N-User Testing**: Demonstrates real concurrent scenarios (10+ users) instead of just 2

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
