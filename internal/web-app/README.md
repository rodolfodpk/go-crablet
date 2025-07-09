# DCB Web Application

A high-performance HTTP/REST API implementation of the go-crablet DCB (Dynamic Consistency Boundary) pattern with comprehensive benchmarking and different transaction isolation levels.

## 🚀 Quick Start

```bash
# Start PostgreSQL
docker-compose up -d postgres

# Run web application
cd internal/web-app
make run-server

# Test API
curl http://localhost:8080/health
```

## 📊 Performance Results

See [k6/README.md](k6/README.md) for comprehensive test organization and [k6/benchmarks/](k6/benchmarks/) for detailed benchmark results.

### Recent Benchmark Results
- **Zero HTTP Failures**: All tests achieve 100% HTTP success rate
- **Sub-500ms p95**: 95th percentile response times under 500ms for most operations
- **High Throughput**: Sustained 200+ req/s under load with 50 concurrent users
- **Serializable Conflicts**: ~42% conflict rate expected for Serializable isolation (correct behavior)

## 🔧 Available Commands

```bash
# Server Management
make run-server          # Start web application
make kill-server         # Stop web application
make ensure-server       # Ensure server is running

# Database Management  
make start-db           # Start PostgreSQL
make stop-db            # Stop PostgreSQL
make cleanup-db         # Clean database

# Quick Tests (30s-2m)
make test-quick        # Run all quick validation tests
make quick-basic       # Basic functionality test
make quick-append      # Quick append validation
make quick-append-if   # Quick conditional append test
make quick-append-isolated # Quick isolated append test

# Benchmarks (3-5m)
make test-benchmarks   # Run all benchmark tests
make benchmark-append  # Full append performance benchmark
make benchmark-append-if # Conditional append benchmark
make benchmark-append-isolated # Serializable isolation benchmark
make benchmark-isolation # Compare all isolation levels

# Functional Tests (2-4m)
make test-functional   # Run all functional tests
make test-concurrency  # Basic concurrency testing
make test-advisory-locks # Advisory locks concurrency

# Load Tests (5-10m)
make test-load         # Run all load tests
make load-full         # Complete system load test
make load-full-scan    # Full scan performance under load
```

## 🔌 API Endpoints

### POST /append
Simple append using ReadCommitted isolation (fastest).

```bash
curl -X POST http://localhost:8080/append \
  -H "Content-Type: application/json" \
  -d '{
    "events": {
      "type": "CoursePlanned",
      "data": "{\"courseId\": \"course-123\"}",
      "tags": ["course:course-123", "user:user-123"]
    }
  }'
```

### POST /append-if
Conditional append using RepeatableRead isolation (default) or Serializable isolation (via header).

```bash
# Default (RepeatableRead)
curl -X POST http://localhost:8080/append-if \
  -H "Content-Type: application/json" \
  -d '{
    "events": {
      "type": "StudentEnrolled",
      "data": "{\"studentId\": \"student-123\"}",
      "tags": ["course:course-123", "student:student-123"]
    },
    "condition": {
      "failIfEventsMatch": {
        "items": [{
          "types": ["StudentEnrolled"],
          "tags": ["course:course-123", "student:student-123"]
        }]
      }
    }
  }'

# Serializable isolation
curl -X POST http://localhost:8080/append-if \
  -H "Content-Type: application/json" \
  -H "X-Append-If-Isolation: serializable" \
  -d '{...}'
```

### POST /read
Query events by type and tags.

```bash
curl -X POST http://localhost:8080/read \
  -H "Content-Type: application/json" \
  -d '{
    "query": {
      "items": [{
        "types": ["CoursePlanned", "StudentEnrolled"],
        "tags": ["course:course-123"]
      }]
    }
  }'
```

### GET /health
Health check endpoint.

### POST /cleanup
Clean database (removes all events).

## 🔒 Transaction Isolation Levels

| Endpoint | Isolation Level | Use Case | Performance |
|----------|----------------|----------|-------------|
| **`POST /append`** | ReadCommitted | Simple appends | Fastest |
| **`POST /append-if`** | RepeatableRead | Conditional appends | Balanced |
| **`POST /append-if` + header** | Serializable | Critical operations | Strongest |

**HTTP Header for Serializable**: `X-Append-If-Isolation: serializable`

## 📈 Benchmark Details

### Test Categories
- **Quick Tests** (30s-2m): Fast validation and smoke tests
- **Benchmarks** (3-5m): Comprehensive performance measurement
- **Functional Tests** (2-4m): Functional validation and concurrency testing
- **Load Tests** (5-10m): High-load system testing

### Isolation Level Benchmarks
- **Append Benchmarks**: Test ReadCommitted isolation performance
- **AppendIf Benchmarks**: Test RepeatableRead isolation with conditions
- **Serializable Benchmarks**: Test Serializable isolation (higher conflict rates expected)

### Performance Thresholds
- **Response Time**: 95% < 1000ms, 99% < 2000ms
- **Error Rate**: < 10% for most operations, < 15% for Serializable
- **Success Rate**: 100% HTTP success, >95% performance compliance

## ⚙️ Configuration

### Environment Variables
```bash
PORT=8080                                    # Web application port
DATABASE_URL=postgres://...                  # PostgreSQL connection string
```

### Database Configuration
- **Connection Pool**: 20 max connections, 5 min connections
- **PostgreSQL**: Optimized for high concurrency
- **Indexes**: GIN indexes on tags for fast queries

## 🛠️ Development

### Prerequisites
- Go 1.21+
- PostgreSQL 17.5+
- k6 (for load testing)

### Manual Setup
```bash
export DATABASE_URL="postgres://postgres:postgres@localhost:5432/dcb_app?sslmode=disable"
export PORT="8080"
go run main.go
```

### Load Testing
```bash
# Install k6
brew install k6  # macOS

# Run tests by category
make test-quick        # Quick validation tests
make test-benchmarks   # Performance benchmarks
make test-functional   # Functional tests
make test-load         # Load tests

# Or run individual tests
make quick-basic       # Basic functionality
make benchmark-append  # Append performance
make test-concurrency  # Concurrency testing
make load-full         # Full system load
```

## 📚 Documentation

- **[OpenAPI Specification](openapi.yaml)**: Complete API specification
- **[Main Project README](../../README.md)**: Core library documentation

## 🎯 Key Features

- **Zero HTTP Failures**: 100% success rate across all benchmarks
- **Sub-30ms Average Response**: Excellent performance under load
- **Multiple Isolation Levels**: Choose consistency vs performance trade-offs
- **Comprehensive Testing**: k6 load testing with multiple scenarios
- **Production Ready**: Optimized connection pooling and resource allocation

## NOTE ON ISOLATION LEVEL

Isolation level is now set in the server config, not per request. To test with SERIALIZABLE, restart the server with the config set to IsolationLevelSerializable. All k6 scripts now use the server's configured isolation level.