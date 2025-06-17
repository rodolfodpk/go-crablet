# DCB Bench Web Application

This is a web application that implements the DCB Bench OpenAPI specification, providing HTTP endpoints for reading and appending events using the core DCB (Domain-Centric Benchmark) API.

**API Specification**: [DCB Bench OpenAPI 1.0.0](https://app.swaggerhub.com/apis/wwwision/dcb-bench/1.0.0#/)

## Features

- **Read Endpoint** (`/read`): Query events by type and tags with performance timing
- **Append Endpoint** (`/append`): Append single or multiple events with optional conditions
- **Performance Metrics**: All operations include microsecond timing measurements
- **Concurrency Support**: Handles append conditions and concurrency errors
- **Docker Support**: Containerized application with PostgreSQL

## Quick Start

### Using Docker Compose (Recommended)

1. **Start the PostgreSQL database (from project root):**
   ```bash
   # From the project root directory
   docker-compose up -d postgres
   ```

2. **Start the web application:**
   ```bash
   cd internal/web-app
   docker-compose up -d
   ```

3. **Wait for services to be healthy:**
   ```bash
   docker-compose ps
   ```

4. **Test the API:**
   ```bash
   # Test append endpoint
   curl -X POST http://localhost:8080/append \
     -H "Content-Type: application/json" \
     -d '{
       "events": {
         "type": "CoursePlanned",
         "data": "{\"courseId\": \"c1\", \"name\": \"Introduction to Go\"}",
         "tags": ["course:c1", "user:u1"]
       }
     }'

   # Test read endpoint
   curl -X POST http://localhost:8080/read \
     -H "Content-Type: application/json" \
     -d '{
       "query": {
         "items": [{
           "types": ["CoursePlanned"],
           "tags": ["course:c1"]
         }]
       }
     }'
   ```

### Manual Setup

1. **Prerequisites:**
   - Go 1.21+
   - PostgreSQL 15+ (or use the existing docker-compose setup)
   - Make sure the core DCB package is available

2. **Set up database:**
   ```bash
   # Option 1: Use existing docker-compose setup
   docker-compose up -d postgres
   
   # Option 2: Manual PostgreSQL setup
   createdb dcb_app
   # Or using PostgreSQL CLI
   psql -c "CREATE DATABASE dcb_app;"
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
    "data": "{\"courseId\": \"c1\"}",
    "tags": ["course:c1", "user:u1"]
  },
  "condition": {
    "failIfEventsMatch": {
      "items": [{
        "types": ["CoursePlanned"],
        "tags": ["course:c1"]
      }]
    }
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
      "tags": ["course:c1"]
    }]
  },
  "options": {
    "from": "3ff67a09-c85f-4589-aa13-4e977eaa9763",
    "backwards": false
  }
}
```

**Response:**
```json
{
  "durationInMicroseconds": 850,
  "numberOfMatchingEvents": 5,
  "checkpointEventId": "3ff67a09-c85f-4589-aa13-4e977eaa9763"
}
```

## Performance Testing with k6

The application includes comprehensive k6 load testing.

### Quick Test

```bash
# Clean start (removes all data)
docker-compose down -v

# Start fresh stack
docker-compose up -d

# Run quick test
k6 run quick-test.js
```

### Load Test

```bash
# Clean start (removes all data)
docker-compose down -v

# Start fresh stack
docker-compose up -d

# Run comprehensive load test
k6 run k6-test.js
```

### Test Results (Latest Run)

**Load Test (3m 30s, 20 VUs):**
- **Success Rate**: 100% (all HTTP requests successful)
- **Throughput**: 63.9 requests/second
- **Average Response Time**: 100.5ms
- **Total Requests**: 13,476

**Performance by Operation:**
- **Append Single**: 81% under 100ms target
- **Append Multiple**: 98% under 200ms target ✅
- **Read by Type**: 47% under 100ms target
- **Read by Tags**: 19% under 100ms target
- **Read by Type+Tags**: 66% under 100ms target
- **Append with Condition**: 91% under 100ms target ✅
- **Complex Query**: 61% under 150ms target

**Key Findings:**
- ✅ **100% reliability** - no failed requests
- ✅ **Good append performance** - most operations meet targets
- ⚠️ **Read operations need optimization** - especially tag-based queries

For detailed analysis, see [k6 Benchmark Report](k6-benchmark-report.md).

## k6 Benchmarks

For detailed performance benchmark results and analysis, see the [k6 Benchmark Report](k6-benchmark-report.md).

## Development

### Project Structure

```
internal/web-app/
├── main.go              # Main application entry point
├── Dockerfile           # Container configuration
├── docker-compose.yml   # Local development stack
├── k6-test.js          # Performance test script
└── README.md           # This file
```

### Key Components

- **Server**: HTTP server with `/read` and `/append` endpoints
- **Type Conversion**: Converts OpenAPI types to DCB core types
- **Performance Timing**: Measures operation duration in microseconds
- **Error Handling**: Proper HTTP status codes and error responses

### Environment Variables

- `DATABASE_URL`: PostgreSQL connection string
- `PORT`: HTTP server port (default: 8080)

### Building

```bash
# Build binary
go build -o web-app ./internal/web-app

# Build Docker image
docker build -f internal/web-app/Dockerfile -t dcb-bench .
```

## Monitoring and Health Checks

The application includes health checks:

- **Docker Health Check**: Tests `/read` endpoint availability
- **Database Health Check**: PostgreSQL connection verification
- **Application Metrics**: Request duration and error tracking

## Troubleshooting

### Common Issues

1. **Database Connection Failed:**
   - Verify PostgreSQL is running
   - Check `DATABASE_URL` environment variable
   - Ensure database `dcb_app` exists

2. **Port Already in Use:**
   - Change `PORT` environment variable
   - Or stop other services using port 8080

3. **Permission Denied:**
   - Ensure proper file permissions
   - Check Docker user permissions if using containers

### Logs

```bash
# View application logs
docker-compose logs web-app

# View database logs
docker-compose logs postgres

# Follow logs in real-time
docker-compose logs -f web-app
```

## Contributing

1. Follow the existing code style
2. Add tests for new features
3. Update documentation as needed
4. Ensure k6 tests pass with acceptable performance

## License

This project is part of the go-crablet repository and follows the same license terms. 