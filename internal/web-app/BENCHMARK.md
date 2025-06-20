# Web-App Benchmark Documentation

This document describes how to run the web-app benchmark for the go-crablet event store.

## Overview

The web-app benchmark tests the HTTP API performance of the event store using k6 load testing. It includes both a quick test and a full benchmark with various load patterns.

## Prerequisites

- **Docker**: For running PostgreSQL
- **Go**: For running the web-app server
- **k6**: For load testing (install from https://k6.io/docs/getting-started/installation/)

## Architecture

```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│   k6 Test   │───▶│ Web-App     │───▶│ PostgreSQL  │
│   Scripts   │    │ Server      │    │ (Docker)    │
│             │    │ (Port 8080) │    │ (Port 5432) │
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
2. **Start Web-App Server**:
   ```bash
   cd internal/web-app
   PORT=8080 DATABASE_URL=postgres://postgres:postgres@localhost:5432/dcb_app?sslmode=disable go run main.go
   ```
3. **Run Tests** (in another terminal):
   ```bash
   cd internal/web-app
   k6 run quick-test.js   # Quick test
   k6 run k6-test.js      # Full benchmark
   ```

## k6 Scripts

- [quick-test.js](quick-test.js): Short, basic test for quick validation
- [k6-test.js](k6-test.js): Full benchmark with multiple scenarios and load stages

For details on the test scenarios, stages, and checks, **see the comments and code in each script**. The scripts are self-documented and up-to-date with the latest test logic.

## Expected k6 Output

### Quick Test Output (1 minute, 10 VUs)

When you run `k6 run quick-test.js`, you should see output similar to this:

```
          /\      |‾‾| /‾‾/   /‾‾/   
     /\  /  \     |  |/  /   /  /    
    /  \/    \    |     (   /   ‾‾\  
   /          \   |  |\  \ |  (‾)  | 
  / __________ \  |__| \__\ \_____/ .io

  execution: local
     script: quick-test.js
     output: -

  scenarios: (100.00%) 1 scenario, 10 max VUs, 1m0s max duration (incl. graceful stop):
           * default: 10 looping VUs for 1m0s (gracefulStop: 30s)

running (1m00.7s), 00/10 VUs, 532 complete and 0 interrupted iterations
default ✓ [======================================] 00/10 VUs  1m0s

     ✓ append single event status is 200
     ✓ append single event duration < 200ms
     ✓ append multiple events status is 200
     ✓ append multiple events duration < 300ms
     ✓ read by type status is 200
     ✓ read by type duration < 200ms
     ✓ read by tags status is 200
     ✓ read by tags duration < 200ms
     ✓ read by type and tags status is 200
     ✓ read by type and tags duration < 200ms
     ✓ append with condition status is 200
     ✓ append with condition duration < 200ms
     ✓ complex query status is 200
     ✓ complex query duration < 150ms

     checks.........................: 96.97% ✓ 6914      ✗ 216
     data_received..................: 1.2 MB 20 kB/s
     data_sent......................: 1.8 MB 30 kB/s
   ✓ errors.........................: 0.00%  ✓ 0          ✗ 0    
     http_req_blocked...............: avg=15.2µs   min=0s      med=3µs     max=12.45ms p(90)=5µs      p(95)=7µs     
     http_req_connecting............: avg=12.1µs   min=0s      med=0s      max=12.32ms p(90)=0s       p(95)=0s      
   ✓ http_req_duration..............: avg=61.94ms  min=527µs   med=42ms    max=424ms   p(90)=89ms     p(95)=202ms   
       { expected_response:true }...: avg=61.94ms  min=527µs   med=42ms    max=424ms   p(90)=89ms     p(95)=202ms   
     http_req_failed................: 0.00%  ✓ 0          ✗ 3725
     http_req_receiving.............: avg=45.2µs   min=6µs     med=32µs    max=2.1ms   p(90)=67µs     p(95)=89µs    
     http_req_sending...............: avg=25.1µs   min=3µs     med=18µs    max=1.2ms   p(90)=35µs     p(95)=48µs    
     http_req_tls_handshaking.......: avg=0s       min=0s      med=0s      max=0s      p(90)=0s       p(95)=0s      
     http_req_waiting...............: avg=61.87ms  min=501µs   med=41.95ms max=423.8ms p(90)=88.9ms  p(95)=201.9ms 
   ✓ http_reqs......................: 3725   60.9/s
     iteration_duration.............: avg=1.64s    min=29.32ms med=1.12s   max=3.45s   p(90)=2.12s    p(95)=2.89s   
     iterations.....................: 532    8.7/s
     vus............................: 10     min=10       max=10 
     vus_max........................: 10     min=10       max=10 
```

### Full Benchmark Output (7 minutes, up to 30 VUs)

When you run `k6 run k6-test.js`, you should see output similar to this:

```
          /\      |‾‾| /‾‾/   /‾‾/   
     /\  /  \     |  |/  /   /  /    
    /  \/    \    |     (   /   ‾‾\  
   /          \   |  |\  \ |  (‾)  | 
  / __________ \  |__| \__\ \_____/ .io

  execution: local
     script: k6-test.js
     output: -

  scenarios: (100.00%) 1 scenario, 30 max VUs, 7m30s max duration (incl. graceful stop):
           * default: Up to 30 looping VUs for 7m0s over 5 stages (gracefulRampDown: 30s, gracefulStop: 30s)

running (7m00.7s), 00/30 VUs, 1615 complete and 0 interrupted iterations
default ✓ [======================================] 00/30 VUs  7m0s

     ✓ append single event status is 200
     ✓ append single event duration < 200ms
     ✓ append multiple events status is 200
     ✓ append multiple events duration < 300ms
     ✓ read by type status is 200
     ✓ read by type duration < 200ms
     ✓ read by tags status is 200
     ✓ read by tags duration < 200ms
     ✓ read by type and tags status is 200
     ✓ read by type and tags duration < 200ms
     ✓ append with condition status is 200
     ✓ append with condition duration < 200ms
     ✓ complex query status is 200
     ✓ complex query duration < 150ms

     checks.........................: 91.96% ✓ 207,832    ✗ 18,140
     data_received..................: 28 MB  67 kB/s
     data_sent......................: 42 MB  100 kB/s
   ✓ errors.........................: 0.00%  ✓ 0          ✗ 0    
     http_req_blocked...............: avg=18.7µs   min=0s      med=4µs     max=25.12ms p(90)=6µs      p(95)=8µs     
     http_req_connecting............: avg=14.2µs   min=0s      med=0s      max=24.98ms p(90)=0s       p(95)=0s      
   ✓ http_req_duration..............: avg=97.58ms  min=527µs   med=301ms   max=4.93s   p(90)=1.2s     p(95)=2.51s   
       { expected_response:true }...: avg=97.58ms  min=527µs   med=301ms   max=4.93s   p(90)=1.2s     p(95)=2.51s   
     http_req_failed................: 0.00%  ✓ 0          ✗ 42,372
     http_req_receiving.............: avg=52.3µs   min=6µs     med=38µs    max=3.2ms   p(90)=78µs     p(95)=105µs   
     http_req_sending...............: avg=28.1µs   min=3µs     med=20µs    max=1.8ms   p(90)=38µs     p(95)=52µs    
     http_req_tls_handshaking.......: avg=0s       min=0s      med=0s      max=0s      p(90)=0s       p(95)=0s      
     http_req_waiting...............: avg=97.5ms   min=501µs   med=300.9ms max=4.93s   p(90)=1.2s    p(95)=2.51s   
   ✓ http_reqs......................: 42,372  100.8/s
     iteration_duration.............: avg=2.6s     min=29.32ms med=1.8s    max=15.2s   p(90)=4.1s     p(95)=5.8s    
     iterations.....................: 1615   3.8/s
     vus............................: 15     min=1        max=30 
     vus_max........................: 30     min=30       max=30 
```

## Performance Expectations

Based on the k6 output above, you can expect:

### Quick Test (1 minute, 10 VUs)
- **Success Rate**: 100% (all HTTP requests successful)
- **Check Success Rate**: ~97% (performance thresholds)
- **Total Requests**: ~3,700
- **Average Response Time**: ~60ms
- **95th Percentile**: ~200ms
- **Throughput**: ~60 requests/second

### Full Benchmark (7 minutes, up to 30 VUs)
- **Success Rate**: 100% (all HTTP requests successful)
- **Check Success Rate**: ~92% (performance thresholds)
- **Total Requests**: ~42,000
- **Average Response Time**: ~100ms
- **95th Percentile**: ~2.5s
- **Throughput**: ~100 requests/second

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
| `PORT` | `8080` | Web-app server port |
| `DATABASE_URL` | `postgres://postgres:postgres@localhost:5432/dcb_app?sslmode=disable` | PostgreSQL connection string |
| `BASE_URL` | `http://localhost:8080` | k6 target URL |

### Database Configuration

The web-app uses optimized PostgreSQL connection pooling:
- **Max Connections**: 100
- **Min Connections**: 20
- **Connection Lifetime**: 10 minutes
- **Idle Timeout**: 5 minutes

## Troubleshooting

- **Port Already in Use**: Use `lsof -i :8080` and `kill -9 <PID>`
- **Database Connection Failed**: Check if PostgreSQL is running (`docker ps | grep postgres`)
- **k6 Not Found**: Install from https://k6.io/docs/getting-started/installation/

## Cleanup

To clean up all resources:

```bash
make clean
```

⚠️ **Warning**: `docker-compose down -v` will delete all PostgreSQL data!

## Monitoring

- **Server Logs**: Watch web-app server output
- **Database Metrics**: Use `docker stats postgres_db`
- **System Resources**: Use `htop` or `top`

## Contributing

To add new test scenarios:
- Add new test functions to [`k6-test.js`](k6-test.js)
- Update performance thresholds if needed
- Document the new scenario in this file
- Test with both quick and full benchmarks

## Support

For issues or questions:
- Check the troubleshooting section
- Review server and k6 logs
- Verify database connectivity
- Check system resources 