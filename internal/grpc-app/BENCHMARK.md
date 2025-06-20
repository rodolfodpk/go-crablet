# gRPC App Benchmark Documentation

This document describes how to run the gRPC benchmark for the go-crablet event store.

## Overview

The gRPC benchmark tests the gRPC API performance of the event store using k6 with the gRPC extension. It includes both a quick test and a full benchmark with various load patterns.

## Prerequisites

- **Docker**: For running PostgreSQL
- **Go**: For running the gRPC server
- **k6**: For load testing (with gRPC support, see https://k6.io/docs/using-k6/protocols/grpc/)

## Architecture

```
┌─────────────┐    ┌──────────────┐    ┌─────────────┐
│   k6 gRPC   │───▶│ gRPC Server  │───▶│ PostgreSQL  │
│   Scripts   │    │ (Port 9090)  │    │ (Docker)    │
└─────────────┘    └──────────────┘    └─────────────┘
```

## Quick Start

### Using Makefile (Recommended)

```bash
# Run the complete benchmark suite
make benchmark

# Run only the quick test
make quick-test

# Run only the full benchmark
make full-benchmark

# Clean up (with safety prompt)
make clean
```

### Manual Steps

1. **Start PostgreSQL**:
   ```bash
   cd /path/to/go-crablet
   docker-compose up -d postgres
   ```
2. **Start gRPC Server**:
   ```bash
   cd internal/grpc-app
   PORT=9090 DATABASE_URL=postgres://postgres:postgres@localhost:5432/dcb_app?sslmode=disable go run server/main.go
   ```
3. **Run Tests** (in another terminal):
   ```bash
   cd internal/grpc-app
   k6 run k6-quick-test.js   # Quick test
   k6 run k6-grpc-test.js    # Full benchmark
   ```

## k6 Scripts

- [k6-quick-test.js](k6-quick-test.js): Short, basic test for quick validation
- [k6-grpc-test.js](k6-grpc-test.js): Full benchmark with multiple scenarios and load stages

For details on the test scenarios, stages, and checks, **see the comments and code in each script**. The scripts are self-documented and up-to-date with the latest test logic.

## How to Use k6 Results

After running a test, k6 will output a detailed summary including:
- Request rates
- Response times (avg, p90, p95, p99, max)
- Success/error rates
- Thresholds and checks

**For reporting or analysis:**
- Copy-paste the k6 output from your terminal.
- The output contains all relevant metrics and statistics.
- No need to duplicate numbers in this markdown; always refer to the k6 output for the actual results.

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `9090` | gRPC server port |
| `DATABASE_URL` | `postgres://postgres:postgres@localhost:5432/dcb_app?sslmode=disable` | PostgreSQL connection string |

### Database Configuration

The gRPC app uses optimized PostgreSQL connection pooling:
- **Max Connections**: 100
- **Min Connections**: 20
- **Connection Lifetime**: 10 minutes
- **Idle Timeout**: 5 minutes

## Troubleshooting

- **Port Already in Use**: Use `lsof -i :9090` and `kill -9 <PID>`
- **Database Connection Failed**: Check if PostgreSQL is running (`docker ps | grep postgres`)
- **k6 Not Found or gRPC Not Supported**: See https://k6.io/docs/using-k6/protocols/grpc/

## Cleanup

To clean up all resources:

```bash
make clean
```

⚠️ **Warning**: `docker-compose down -v` will delete all PostgreSQL data!

## Monitoring

- **Server Logs**: Watch gRPC server output
- **Database Metrics**: Use `docker stats postgres_db`
- **System Resources**: Use `htop` or `top`

## Contributing

To add new test scenarios:
- Add new test functions to [`k6-grpc-test.js`](k6-grpc-test.js)
- Update performance thresholds if needed
- Document the new scenario in this file
- Test with both quick and full benchmarks

## Support

For issues or questions:
- Check the troubleshooting section
- Review server and k6 logs
- Verify database connectivity
- Check system resources 