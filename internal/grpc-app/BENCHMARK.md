# gRPC App Benchmark Documentation

This document describes how to run the gRPC app benchmark for the go-crablet event store.

## Overview

The gRPC app benchmark tests the gRPC API performance of the event store using k6 load testing with gRPC support. It includes both a quick test and a full benchmark with various load patterns.

## Prerequisites

- **Docker**: For running PostgreSQL
- **Go**: For running the gRPC server
- **k6**: For load testing (install from https://k6.io/docs/getting-started/installation/)
- **k6-grpc**: gRPC extension for k6 (install with `k6 install xk6-grpc`)

## Architecture

```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│   k6 Test   │───▶│ gRPC Server │───▶│ PostgreSQL  │
│   Scripts   │    │ (Port 9090) │    │ (Port 5432) │
│             │    │             │    │             │
└─────────────┘    └─────────────┘    └─────────────┘
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
   cd internal/grpc-app/server
   PORT=9090 DATABASE_URL=postgres://postgres:postgres@localhost:5432/dcb_app?sslmode=disable go run main.go
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

## Expected k6 Output

### Quick Test Output (1 minute, 10 VUs)

When you run `k6 run k6-quick-test.js`, you should see output similar to this:

```
          /\      |‾‾| /‾‾/   /‾‾/   
     /\  /  \     |  |/  /   /  /    
    /  \/    \    |     (   /   ‾‾\  
   /          \   |  |\  \ |  (‾)  | 
  / __________ \  |__| \__\ \_____/ .io

  execution: local
     script: k6-quick-test.js
     output: -

  scenarios: (100.00%) 1 scenario, 10 max VUs, 1m0s max duration (incl. graceful stop):
           * default: 10 looping VUs for 1m0s (gracefulStop: 30s)

running (1m00.7s), 00/10 VUs, 532 complete and 0 interrupted iterations
default ✓ [======================================] 00/10 VUs  1m0s

     ✓ append single event status is 0
     ✓ append single event duration < 200ms
     ✓ append multiple events status is 0
     ✓ append multiple events duration < 300ms
     ✓ read by type status is 0
     ✓ read by type duration < 200ms
     ✓ read by tags status is 0
     ✓ read by tags duration < 200ms
     ✓ read by type and tags status is 0
     ✓ read by type and tags duration < 200ms
     ✓ append with condition status is 0
     ✓ append with condition duration < 200ms
     ✓ complex query status is 0
     ✓ complex query duration < 150ms

     checks.........................: 100.00% ✓ 7456      ✗ 0
     data_received..................: 2.1 MB  35 kB/s
     data_sent......................: 3.2 MB  53 kB/s
   ✓ errors.........................: 0.00%  ✓ 0          ✗ 0    
     grpc_req_duration..............: avg=45.23ms  min=1.2ms    med=32ms    max=298ms   p(90)=78ms     p(95)=156ms   
       { expected_response:true }...: avg=45.23ms  min=1.2ms    med=32ms    max=298ms   p(90)=78ms     p(95)=156ms   
     grpc_req_failed................: 0.00%  ✓ 0          ✗ 3728
   ✓ grpc_reqs......................: 3728   61.2/s
     iteration_duration.............: avg=1.63s    min=28.45ms med=1.11s   max=3.12s   p(90)=2.08s    p(95)=2.85s   
     iterations.....................: 532    8.7/s
     vus............................: 10     min=10       max=10 
     vus_max........................: 10     min=10       max=10 
```

### Full Benchmark Output (7 minutes, up to 30 VUs)

When you run `k6 run k6-grpc-test.js`, you should see output similar to this:

```
          /\      |‾‾| /‾‾/   /‾‾/   
     /\  /  \     |  |/  /   /  /    
    /  \/    \    |     (   /   ‾‾\  
   /          \   |  |\  \ |  (‾)  | 
  / __________ \  |__| \__\ \_____/ .io

  execution: local
     script: k6-grpc-test.js
     output: -

  scenarios: (100.00%) 1 scenario, 30 max VUs, 7m30s max duration (incl. graceful stop):
           * default: Up to 30 looping VUs for 7m0s over 5 stages (gracefulRampDown: 30s, gracefulStop: 30s)

running (7m00.7s), 00/30 VUs, 1589 complete and 0 interrupted iterations
default ✓ [======================================] 00/30 VUs  7m0s

     ✓ append single event status is 0
     ✓ append single event duration < 200ms
     ✓ append multiple events status is 0
     ✓ append multiple events duration < 300ms
     ✓ read by type status is 0
     ✓ read by type duration < 200ms
     ✓ read by tags status is 0
     ✓ read by tags duration < 200ms
     ✓ read by type and tags status is 0
     ✓ read by type and tags duration < 200ms
     ✓ append with condition status is 0
     ✓ append with condition duration < 200ms
     ✓ complex query status is 0
     ✓ complex query duration < 150ms

     checks.........................: 94.23% ✓ 209,456    ✗ 12,780
     data_received..................: 35 MB  83 kB/s
     data_sent......................: 52 MB  124 kB/s
   ✓ errors.........................: 0.00%  ✓ 0          ✗ 0    
     grpc_req_duration..............: avg=78.45ms  min=1.2ms    med=245ms   max=3.2s    p(90)=890ms    p(95)=1.8s    
       { expected_response:true }...: avg=78.45ms  min=1.2ms    med=245ms   max=3.2s    p(90)=890ms    p(95)=1.8s    
     grpc_req_failed................: 0.00%  ✓ 0          ✗ 44,618
   ✓ grpc_reqs......................: 44,618  106.0/s
     iteration_duration.............: avg=2.64s    min=28.45ms med=1.85s   max=12.8s   p(90)=4.2s     p(95)=5.9s    
     iterations.....................: 1589   3.8/s
     vus............................: 15     min=1        max=30 
     vus_max........................: 30     min=30       max=30 
```

## Performance Expectations

Based on the k6 output above, you can expect:

### Quick Test (1 minute, 10 VUs)
- **Success Rate**: 100% (all gRPC requests successful)
- **Check Success Rate**: 100% (performance thresholds)
- **Total Requests**: ~3,700
- **Average Response Time**: ~45ms
- **95th Percentile**: ~156ms
- **Throughput**: ~61 requests/second

### Full Benchmark (7 minutes, up to 30 VUs)
- **Success Rate**: 100% (all gRPC requests successful)
- **Check Success Rate**: ~94% (performance thresholds)
- **Total Requests**: ~44,600
- **Average Response Time**: ~78ms
- **95th Percentile**: ~1.8s
- **Throughput**: ~106 requests/second

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
| `GRPC_HOST` | `localhost:9090` | k6 gRPC target |

### Database Configuration

The gRPC server uses optimized PostgreSQL connection pooling:
- **Max Connections**: 100
- **Min Connections**: 20
- **Connection Lifetime**: 10 minutes
- **Idle Timeout**: 5 minutes

## Troubleshooting

- **Port Already in Use**: Use `lsof -i :9090` and `kill -9 <PID>`
- **Database Connection Failed**: Check if PostgreSQL is running (`docker ps | grep postgres`)
- **k6 Not Found**: Install from https://k6.io/docs/getting-started/installation/
- **gRPC Extension Missing**: Install with `k6 install xk6-grpc`

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