# DCB Bench Web Application

This is a web application that implements the DCB Bench OpenAPI specification, providing HTTP endpoints for reading and appending events using the core DCB (Domain-Centric Benchmark) API.

**API Specification**: [DCB Bench OpenAPI 1.0.0](https://app.swaggerhub.com/apis/wwwision/dcb-bench/1.0.0#/)

## Features

- **Read Endpoint** (`/read`): Query events by type and tags with performance timing
- **Append Endpoint** (`/append`): Append single or multiple events with optional conditions
- **Performance Metrics**: All operations include microsecond timing measurements
- **Concurrency Support**: Handles append conditions and concurrency errors
- **Docker Support**: Containerized application with PostgreSQL
- **Performance Optimized**: Configured for high-throughput with optimized resource allocation

## Performance Results (2024-06-13)

### High-Load Test (50 VUs, 8m)
- **Throughput**: 217.19 requests/sec
- **Success Rate**: 100% (zero errors)
- **Check Success Rate**: 97.59%
- **Total Events Created**: 74,508 (verified in PostgreSQL)
- **Average Response Time**: 44.49ms
- **95th Percentile**: 199.85ms

### Resource Allocation
- **Web-app**: 4 CPUs, 1GB RAM
- **Postgres**: 4 CPUs, 4GB RAM
- **Database**: PostgreSQL 17.5 with performance tuning

## Docker Compose Files

### Root docker-compose.yaml
- **Location**: `/docker-compose.yaml`
- **Purpose**: Main development setup with PostgreSQL 17.5
- **Usage**: `docker-compose up -d` from project root

### Web-app docker-compose.yml
- **Location**: `/internal/web-app/docker-compose.yml`
- **Purpose**: Optimized for benchmarking with resource limits
- **Usage**: `docker-compose up -d` from web-app directory
- **Features**: Custom resource allocation, performance tuning

## Quick Start

### Using Docker Compose (Recommended)

1. **Start both services from project root:**
   ```bash
   # From the project root directory
   docker-compose -f docker-compose.yaml up -d --build
   ```

2. **Wait for services to be healthy:**
   ```bash
   docker-compose -f docker-compose.yaml ps
   ```

3. **Test the API:**
   ```bash
   # Test health endpoint
   curl http://localhost:8080/health

   # Test append endpoint
   curl -X POST http://localhost:8080/append \
     -H "Content-Type: application/json" \
     -d '{
       "events": {
         "type": "CoursePlanned",
         "data": "{\"courseId\": \"course-1234567890\"}",
         "tags": ["course:course-1234567890", "user:user-1234567890"]
       }
     }'

   # Test read endpoint
   curl -X POST http://localhost:8080/read \
     -H "Content-Type: application/json" \
     -d '{
       "query": {
         "items": [{
           "types": ["CoursePlanned", "StudentEnrolled"],
           "tags": ["course:course-1234567890"]
         }]
       }
     }'
   ```

### Run Performance Tests

```bash
# From the web-app directory
cd internal/web-app
k6 run k6-test.js
```

## Manual Setup

1. **Prerequisites:**
   - Go 1.21+
   - PostgreSQL 17.5+
   - Make sure the core DCB package is available

2. **Set up database:**
   ```bash
   # Use existing docker-compose setup
   docker-compose -f docker-compose.yaml up -d postgres
   ```

3. **Set environment variables:**
   ```bash
   export DATABASE_URL="postgres://postgres:postgres@localhost:5432/dcb_app?sslmode=disable"
   export PORT="8080"
   ```

4. **Run the application:**
   ```bash
   go run ./internal/web-app
   ```

## API Endpoints

### POST /append

Appends one or more events to the event store.

**Request Body:**
```json
{
  "events": {
    "type": "CoursePlanned",
    "data": "{\"courseId\": \"course-1234567890\"}",
    "tags": ["course:course-1234567890", "user:user-1234567890"]
  }
}
```

**Response:**
```json
{
  "durationInMicroseconds": 1250,
  "appendConditionFailed": false
}
```

### POST /read

Reads events matching the specified query.

**Request Body:**
```json
{
  "query": {
    "items": [{
      "types": ["CoursePlanned", "StudentEnrolled"],
      "tags": ["course:course-1234567890"]
    }]
  }
}
```

**Response:**
```json
{
  "durationInMicroseconds": 850,
  "events": [
    {
      "id": "3ff67a09-c85f-4589-aa13-4e977eaa9763",
      "type": "CoursePlanned",
      "data": "{\"courseId\": \"course-1234567890\"}",
      "tags": ["course:course-1234567890", "user:user-1234567890"],
      "timestamp": "2024-06-13T10:30:00Z"
    }
  ]
}
```

### GET /health

Health check endpoint.

**Response:**
```json
{
  "status": "ok"
}
```

## Database Schema

The application uses PostgreSQL with an optimized schema. See [schema.sql](../../docker-entrypoint-initdb.d/schema.sql) for the complete database schema and indexes.

## Performance Optimizations

- **Connection Pooling**: Optimized pool settings for high concurrency
- **Database Indexes**: GIN indexes on tags for fast queries
- **Resource Allocation**: 4 CPUs each for web-app and PostgreSQL
- **Memory Allocation**: 1GB web-app, 4GB PostgreSQL
- **Go Runtime**: Optimized with GOMAXPROCS=4

## Load Testing

The application has been tested with k6 load testing:

- **Test Duration**: 8 minutes
- **Virtual Users**: Up to 50 concurrent users
- **Test Scenarios**: 7 different API operations per iteration
- **Events per Iteration**: 5 events (1 single + 3 multiple + 1 conditional)
- **Total Events**: 74,508 events created and verified in PostgreSQL

See [BENCHMARK.md](BENCHMARK.md) for detailed performance metrics and latest results.

## Development

### Building the Docker Image

```