# Database Connection Pool Monitoring

This document explains how to monitor and troubleshoot database connection pool issues in go-crablet.

## Overview

Database connection leaks occur when connections are acquired from the pool but not properly returned. This can lead to:
- Pool exhaustion
- Application slowdowns
- Database connection limits being reached
- Application crashes

## Connection Pool Configuration

The applications use the following connection pool settings:

```go
config.MaxConns = 300        // Maximum connections
config.MinConns = 100        // Minimum connections
config.MaxConnLifetime = 15 * time.Minute
config.MaxConnIdleTime = 10 * time.Minute
config.HealthCheckPeriod = 30 * time.Second
```

## Monitoring Tools

### 1. Connection Pool Monitor Script

Use the monitoring script to track connection health:

```bash
# Basic monitoring
./scripts/monitor-connections.sh

# Custom configuration
DB_HOST=localhost DB_PORT=5432 DB_NAME=dcb_app DB_USER=postgres MONITOR_INTERVAL=10 ./scripts/monitor-connections.sh
```

The script monitors:
- Active connections
- Idle connections
- Long-running queries (>30s)
- Idle in transaction connections (potential leaks)
- Connection pool exhaustion

### 2. Application-Level Monitoring

The library now includes internal connection pool monitoring:

```go
// Connection pool stats are logged automatically
log.Printf("[Read-before] Pool stats - Total: %d, Idle: %d, Acquired: %d, Constructing: %d",
    stats.TotalConns(), stats.IdleConns(), stats.AcquiredConns(), stats.ConstructingConns())
```

## Common Connection Leak Scenarios

### 1. Channel-Based Streaming Without Proper Cleanup

**Problem**: Channel consumers don't read all events, leaving database rows open.

**Solution**: The library now includes proper cleanup:
```go
defer func() {
    if r := recover(); r != nil {
        log.Printf("ReadStreamChannel panic recovered: %v", r)
    }
    rows.Close()
    close(resultChan)
}()
```

### 2. Context Cancellation Without Cleanup

**Problem**: Context cancellation can leave database connections open.

**Solution**: Enhanced context handling:
```go
select {
case <-ctx.Done():
    // Context cancelled - exit cleanly
    return
case resultChan <- event:
    // Event sent successfully
}
```

### 3. High Connection Pool Settings

**Problem**: High settings mask connection leaks initially.

**Solution**: Monitor connection usage and adjust settings:
```bash
# Check current connection usage
psql -h localhost -U postgres -d dcb_app -c "
SELECT 
    count(*) as total_connections,
    count(*) FILTER (WHERE state = 'active') as active_connections,
    count(*) FILTER (WHERE state = 'idle') as idle_connections
FROM pg_stat_activity 
WHERE datname = 'dcb_app';
"
```

## Troubleshooting Steps

### 1. Check for Connection Leaks

```bash
# Run the monitoring script
./scripts/monitor-connections.sh

# Look for:
# - High number of acquired connections
# - Idle in transaction connections
# - Long-running queries
```

### 2. Check Application Logs

Look for connection pool statistics in application logs:
```
[Read-before] Pool stats - Total: 150, Idle: 50, Acquired: 100, Constructing: 0
[Read-after] Pool stats - Total: 150, Idle: 100, Acquired: 50, Constructing: 0
```

### 3. Check PostgreSQL Activity

```sql
-- Check current connections
SELECT 
    pid,
    usename,
    application_name,
    state,
    query_start,
    query
FROM pg_stat_activity 
WHERE datname = 'dcb_app'
ORDER BY query_start DESC;

-- Check for idle in transaction connections
SELECT 
    pid,
    usename,
    application_name,
    state_change,
    query
FROM pg_stat_activity 
WHERE datname = 'dcb_app' 
AND state = 'idle in transaction'
ORDER BY state_change;
```

### 4. Check for Long-Running Queries

```sql
SELECT 
    pid,
    usename,
    application_name,
    state,
    EXTRACT(EPOCH FROM (now() - query_start)) as duration_seconds,
    query
FROM pg_stat_activity 
WHERE datname = 'dcb_app' 
AND state = 'active'
AND query_start < now() - interval '30 seconds'
ORDER BY query_start;
```

## Best Practices

### 1. Always Use Context Timeouts

```go
// Use timeouts for database operations
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

result, err := store.Read(ctx, query, options)
```

### 2. Properly Handle Channel Streaming

```go
// Always read all events from channels
eventChan, err := channelStore.ReadStreamChannel(ctx, query)
if err != nil {
    return err
}

// Read all events to ensure proper cleanup
for event := range eventChan {
    // Process event
}
```

### 3. Monitor Connection Pool Health

```go
// Use the internal monitoring functions
health := dcb.CheckConnectionPoolHealth(pool)
if !health.Healthy {
    log.Printf("Connection pool unhealthy: %s", health.Message)
}
```

### 4. Adjust Pool Settings Based on Usage

```go
// For high-throughput applications
config.MaxConns = 200
config.MinConns = 50
config.MaxConnLifetime = 10 * time.Minute
config.MaxConnIdleTime = 5 * time.Minute

// For low-throughput applications
config.MaxConns = 50
config.MinConns = 10
config.MaxConnLifetime = 30 * time.Minute
config.MaxConnIdleTime = 15 * time.Minute
```

## Emergency Recovery

If you encounter connection pool exhaustion:

### 1. Restart the Application

```bash
# Stop the application
pkill -f "go run"

# Restart with lower connection limits
DB_MAX_CONNS=50 DB_MIN_CONNS=10 go run main.go
```

### 2. Kill Long-Running Connections

```sql
-- Kill connections that have been idle in transaction for more than 5 minutes
SELECT pg_terminate_backend(pid)
FROM pg_stat_activity 
WHERE datname = 'dcb_app' 
AND state = 'idle in transaction'
AND state_change < now() - interval '5 minutes';
```

### 3. Reset Connection Pool

```sql
-- Disconnect all connections (use with caution)
SELECT pg_terminate_backend(pid)
FROM pg_stat_activity 
WHERE datname = 'dcb_app' 
AND pid <> pg_backend_pid();
```

## Monitoring Dashboard

For production environments, consider setting up monitoring dashboards:

- **Prometheus + Grafana**: Track connection pool metrics
- **pg_stat_statements**: Monitor query performance
- **pg_stat_activity**: Real-time connection monitoring

## Summary

The go-crablet library now includes:
- ✅ Automatic connection pool monitoring
- ✅ Enhanced error handling and cleanup
- ✅ Panic recovery in streaming operations
- ✅ Connection leak detection tools
- ✅ Comprehensive monitoring script

These improvements help prevent connection leaks while maintaining the same public API for consumers. 