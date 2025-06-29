# Append Benchmark Tests

This directory contains k6 scripts specifically designed to benchmark append scenarios for the go-crablet web application.

## Available Benchmarks

### 1. Quick Append Benchmark (`append-quick.js`)
- **Duration**: 30 seconds
- **Virtual Users**: 10
- **Purpose**: Fast testing of basic append scenarios
- **Scenarios Tested**:
  - Single event append
  - Small batch append (3 events)
  - Conditional append (success case)

**Run with:**
```bash
make append-quick
```

### 2. Full Append Benchmark (`append-benchmark.js`)
- **Duration**: 6 minutes
- **Virtual Users**: Up to 200 (ramped up gradually)
- **Purpose**: Comprehensive testing of all append scenarios
- **Scenarios Tested**:
  - Single event appends
  - Small batch appends (2-3 events)
  - Medium batch appends (5 events)
  - Large batch appends (25 events)
  - Conditional appends (success and failure cases)
  - Mixed event types
  - High-frequency events (50 events per batch)

**Run with:**
```bash
make append-full
```

## Append Scenarios Covered

### Single Event Appends
- Basic event creation with unique IDs
- Tests: `UserCreated`, `AccountOpened`, etc.

### Batch Event Appends
- **Small batches**: 2-3 events (e.g., account operations)
- **Medium batches**: 5 events (e.g., order processing)
- **Large batches**: 25 events (e.g., log entries)
- **High-frequency**: 50 events (e.g., sensor readings)

### Conditional Appends
- **Success case**: Append with condition that should pass
- **Failure case**: Append with condition that should fail
- Tests DCB append condition functionality

### Mixed Event Types
- Different event types in same batch
- Tests: `UserCreated`, `AccountOpened`, `TransactionInitiated`, etc.

## Performance Metrics

### Thresholds
- **Response Time**: 95% of requests < 1000ms, 99% < 2000ms
- **Error Rate**: < 10%
- **Request Rate**: > 100 req/s
- **Append Success Rate**: > 95%

### Custom Metrics
- `append_success`: Rate of successful appends
- `batch_appends`: Counter for batch append operations
- `conditional_appends`: Counter for conditional append operations

## Test Data Generation

Each test generates unique data to avoid conflicts:
- **Unique IDs**: Timestamp + random string for each event
- **Random Values**: Numeric values and timestamps
- **Varied Tags**: Different tag combinations for each event
- **Event Types**: Multiple event types to test variety

## Setup and Teardown

### Setup
- Health check to ensure web-app is running
- Database cleanup to start with clean state
- Automatic database preparation

### Teardown
- Automatic cleanup and logging
- Performance summary output

## Running Benchmarks

### Prerequisites
1. **k6 installed**: `brew install k6` (macOS) or follow [k6 installation guide](https://k6.io/docs/getting-started/installation/)
2. **Web-app running**: `make run-server` or `make ensure-server`
3. **Database ready**: `make start-db`

### Quick Test
```bash
# Run quick append benchmark
make append-quick
```

### Full Benchmark
```bash
# Run comprehensive append benchmark
make append-full
```

### Manual Run
```bash
# Start web-app if not running
make ensure-server

# Run specific benchmark
k6 run append-quick.js
k6 run append-benchmark.js
```

## Expected Results

### Quick Benchmark (30s)
- **Throughput**: ~100-200 req/s
- **Response Time**: < 500ms (95th percentile)
- **Error Rate**: < 5%

### Full Benchmark (6m)
- **Peak Throughput**: ~200-500 req/s
- **Response Time**: < 1000ms (95th percentile)
- **Error Rate**: < 10%
- **Total Events**: ~50,000-100,000 events appended

## Isolation Level Context

The append benchmarks test the standard `Append` method which uses **ReadCommitted** isolation:

- **Isolation Level**: ReadCommitted (fastest available)
- **Use Case**: Simple appends where basic consistency is sufficient
- **Performance**: Highest throughput among all append methods
- **Consistency**: Basic consistency, may see phantom reads

For conditional appends with stronger consistency, see:
- **AppendIf benchmarks**: RepeatableRead isolation
- **AppendIf with Serializable**: Serializable isolation (via HTTP header)

## Troubleshooting

### Common Issues
1. **Port conflicts**: Ensure port 8080 is available
2. **Database connection**: Check PostgreSQL is running
3. **Memory issues**: Large batch tests may require more memory

### Debug Mode
```bash
# Run with verbose output
k6 run --verbose append-quick.js
```

## Customization

### Environment Variables
- `BASE_URL`: Override default URL (default: http://localhost:8080)

### Modifying Scenarios
Edit the `APPEND_SCENARIOS` object in `append-benchmark.js` to:
- Add new event types
- Change batch sizes
- Modify tag patterns
- Adjust test frequencies

### Performance Tuning
- Modify `options.stages` for different load patterns
- Adjust `thresholds` for different performance requirements
- Change `sleep()` times for different request rates 