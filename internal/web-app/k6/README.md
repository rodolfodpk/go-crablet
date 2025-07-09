# K6 Test Suite Organization

This directory contains all k6 performance and load testing scripts for the go-crablet web-app, organized by purpose and complexity.

## Directory Structure

```
k6/
├── README.md                    # This file
├── benchmarks/                  # Comprehensive benchmark tests
│   ├── append-benchmark.js      # Full append performance benchmark
│   ├── append-if-benchmark.js   # Conditional append benchmark
│   ├── append-if-isolated-benchmark.js  # Serializable isolation benchmark
│   └── isolation-level-benchmark.js     # Isolation level comparison
├── tests/                       # Functional and concurrency tests
│   ├── k6-concurrency-test.js   # Basic concurrency testing
│   └── k6-advisory-locks-concurrency-test.js  # Advisory locks concurrency
├── quick/                       # Quick validation and smoke tests
│   ├── quick.js                 # Basic functionality test
│   ├── append-quick.js          # Quick append validation
│   ├── isolation-levels-quick.js # Consolidated isolation levels test
│   └── conditional-append-quick.js # Consolidated conditional append test
├── full.js                      # Full system load test
└── full-scan.js                 # Full scan performance test
```

## Test Categories

### 🏃‍♂️ Quick Tests (`quick/`)
**Purpose**: Fast validation and smoke tests
**Duration**: 30 seconds to 2 minutes
**Use Case**: Pre-deployment validation, development testing

- **quick.js** - Basic health and functionality check
- **append-quick.js** - Quick append operation validation
- **isolation-levels-quick.js** - Consolidated test for all isolation levels (READ_COMMITTED, REPEATABLE_READ, SERIALIZABLE)
- **conditional-append-quick.js** - Consolidated test for conditional append across all isolation levels

### 📊 Benchmarks (`benchmarks/`)
**Purpose**: Comprehensive performance measurement
**Duration**: 3-5 minutes
**Use Case**: Performance analysis, capacity planning

- **append-benchmark.js** - Full append performance with multiple scenarios
- **append-if-benchmark.js** - Conditional append performance
- **append-if-isolated-benchmark.js** - Serializable isolation performance
- **isolation-level-benchmark.js** - Compare all isolation levels

### 🧪 Functional Tests (`tests/`)
**Purpose**: Functional validation and concurrency testing
**Duration**: 2-4 minutes
**Use Case**: Integration testing, concurrency validation

- **k6-concurrency-test.js** - Basic concurrency scenarios
- **k6-advisory-locks-concurrency-test.js** - Advisory locks concurrency

### 🔥 Load Tests (root level)
**Purpose**: High-load system testing
**Duration**: 5-10 minutes
**Use Case**: Stress testing, capacity limits

- **full.js** - Complete system load test
- **full-scan.js** - Full scan performance under load

## Usage Examples

### Quick Validation (Development)
```bash
# Basic functionality check
k6 run k6/quick/quick.js

# Quick append validation
k6 run k6/quick/append-quick.js

# Consolidated isolation levels test
k6 run k6/quick/isolation-levels-quick.js

# Consolidated conditional append test
k6 run k6/quick/conditional-append-quick.js
```

### Performance Benchmarking
```bash
# Full append performance
k6 run k6/benchmarks/append-benchmark.js

# Isolation level comparison
k6 run k6/benchmarks/isolation-level-benchmark.js
```

### Concurrency Testing
```bash
# Basic concurrency
k6 run k6/tests/k6-concurrency-test.js

# Advisory locks concurrency
k6 run k6/tests/k6-advisory-locks-concurrency-test.js
```

### Load Testing
```bash
# Full system load
k6 run k6/full.js

# Full scan performance
k6 run k6/full-scan.js
```

## Test Configuration

All tests use the following environment variables:
- `BASE_URL` - Target server URL (default: http://localhost:8080)
- `K6_OUT` - Output format for results

## Isolation Levels

Different tests use different isolation levels:

- **READ_COMMITTED** (default) - Used by most append tests
- **REPEATABLE_READ** - Used by some conditional tests
- **SERIALIZABLE** - Used by isolated append tests

## Performance Thresholds

Most benchmarks include these thresholds:
- `http_req_duration: p(95)<1000ms` - 95% of requests under 1 second
- `http_req_duration: p(99)<2000ms` - 99% of requests under 2 seconds
- `errors: rate<0.10` - Error rate below 10%
- `http_reqs: rate>100` - Minimum 100 requests per second

## Running All Tests

You can run all tests in sequence using the Makefile:

```bash
make test-quick      # Run all quick tests
make test-benchmarks # Run all benchmarks
make test-all        # Run all tests
``` 