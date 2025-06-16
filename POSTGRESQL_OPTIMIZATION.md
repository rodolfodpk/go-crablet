# PostgreSQL Performance Optimization for go-crablet Benchmarks

This document explains the PostgreSQL optimizations applied to improve benchmark performance for the go-crablet library using environment variables.

## 🚀 **Quick Start**

The optimizations are automatically applied when you start PostgreSQL with Docker Compose:

```bash
docker-compose up -d
```

All performance optimizations are configured via environment variables in `docker-compose.yaml`, providing reliable and consistent performance without configuration file issues.

## 📊 **Resource Allocation**

### **CPU Resources**
- **Reserved**: 2 CPU cores (guaranteed)
- **Maximum**: 4 CPU cores (can burst up to 4)
- **Parallel Workers**: 8 maximum parallel workers
- **Parallel Workers per Query**: 4 workers per query

### **Memory Resources**
- **Reserved**: 2GB RAM (guaranteed)
- **Maximum**: 4GB RAM (can use up to 4GB)
- **Shared Buffers**: 256MB (PostgreSQL's main memory)
- **Effective Cache Size**: 1GB (estimated OS cache)
- **Work Memory**: 16MB per operation
- **Maintenance Work Memory**: 256MB for maintenance

## ⚙️ **Environment Variable Optimizations**

### **Memory Configuration**
```yaml
POSTGRES_SHARED_BUFFERS: 256MB                    # PostgreSQL's main memory
POSTGRES_EFFECTIVE_CACHE_SIZE: 1GB                # Estimated OS cache
POSTGRES_WORK_MEM: 16MB                           # Per operation memory
POSTGRES_MAINTENANCE_WORK_MEM: 256MB              # Maintenance operations
```

### **Query Performance**
```yaml
POSTGRES_DEFAULT_STATISTICS_TARGET: 100           # Better query planning
POSTGRES_RANDOM_PAGE_COST: 1.1                    # Optimized for SSD
POSTGRES_EFFECTIVE_IO_CONCURRENCY: 200            # Parallel I/O operations
```

### **Parallel Processing**
```yaml
POSTGRES_MAX_WORKER_PROCESSES: 8                  # Total background processes
POSTGRES_MAX_PARALLEL_WORKERS_PER_GATHER: 4       # Parallel workers per query
POSTGRES_MAX_PARALLEL_WORKERS: 8                  # Total parallel workers
POSTGRES_MAX_PARALLEL_MAINTENANCE_WORKERS: 4      # Parallel maintenance workers
```

### **Checkpoint Optimization**
```yaml
POSTGRES_CHECKPOINT_COMPLETION_TARGET: 0.9        # Spread checkpoints over time
POSTGRES_WAL_BUFFERS: 16MB                        # WAL buffer size
```

### **Authentication**
```yaml
POSTGRES_HOST_AUTH_METHOD: trust                  # No password prompts for benchmarks
```

## 🔧 **Docker Compose Configuration**

The `docker-compose.yaml` file includes:

### **Resource Limits**
```yaml
deploy:
  resources:
    limits:
      cpus: '4.0'
      memory: 4G
    reservations:
      cpus: '2.0'
      memory: 2G
```

### **Environment Variables**
```yaml
environment:
  POSTGRES_USER: postgres
  POSTGRES_PASSWORD: postgres
  POSTGRES_DB: dcb_app
  POSTGRES_SHARED_BUFFERS: 256MB
  POSTGRES_EFFECTIVE_CACHE_SIZE: 1GB
  POSTGRES_WORK_MEM: 16MB
  POSTGRES_MAINTENANCE_WORK_MEM: 256MB
  POSTGRES_CHECKPOINT_COMPLETION_TARGET: 0.9
  POSTGRES_WAL_BUFFERS: 16MB
  POSTGRES_DEFAULT_STATISTICS_TARGET: 100
  POSTGRES_RANDOM_PAGE_COST: 1.1
  POSTGRES_EFFECTIVE_IO_CONCURRENCY: 200
  POSTGRES_MAX_WORKER_PROCESSES: 8
  POSTGRES_MAX_PARALLEL_WORKERS_PER_GATHER: 4
  POSTGRES_MAX_PARALLEL_WORKERS: 8
  POSTGRES_MAX_PARALLEL_MAINTENANCE_WORKERS: 4
  POSTGRES_HOST_AUTH_METHOD: trust
```

## 📈 **Expected Performance Improvements**

### **Append Operations**
- **Faster WAL writes** with optimized WAL buffers
- **Reduced checkpoint overhead** with spread checkpoints
- **Better I/O performance** with parallel operations

### **Read Operations**
- **Improved query planning** with better statistics
- **Faster parallel queries** with optimized worker processes
- **Better cache utilization** with optimized memory settings

### **Projection Operations**
- **Parallel processing** for multiple projectors
- **Optimized memory usage** per operation
- **Better I/O concurrency** for large datasets

## 🧪 **Running Optimized Benchmarks**

### **Quick Benchmarks**
```bash
cd internal/benchmarks
./run_benchmarks.sh quick
```

### **Comprehensive Benchmarks**
```bash
cd internal/benchmarks
./run_benchmarks.sh comprehensive
```

### **Specific Benchmark Categories**
```bash
cd internal/benchmarks/benchmarks
go test -bench=BenchmarkAppend -benchmem -benchtime=10s
go test -bench=BenchmarkRead -benchmem -benchtime=10s
go test -bench=BenchmarkProjection -benchmem -benchtime=10s
```

## 📊 **Monitoring Performance**

### **Check Resource Usage**
```bash
# Check container resource usage
docker stats postgres_db

# Check PostgreSQL processes
docker-compose exec postgres ps aux

# Check PostgreSQL configuration
docker-compose exec postgres psql -U postgres -c "SHOW shared_buffers;"
docker-compose exec postgres psql -U postgres -c "SHOW work_mem;"
```

### **Performance Queries**
```bash
# Check active connections
docker-compose exec postgres psql -U postgres -c "SELECT count(*) FROM pg_stat_activity;"

# Check cache hit ratio
docker-compose exec postgres psql -U postgres -c "
SELECT 
    schemaname,
    tablename,
    heap_blks_read,
    heap_blks_hit,
    round(heap_blks_hit::numeric/(heap_blks_hit + heap_blks_read), 4) as hit_ratio
FROM pg_statio_user_tables
WHERE heap_blks_read > 0;
"
```

## ⚠️ **Important Notes**

1. **Automatic Configuration**: All optimizations are applied automatically via environment variables
2. **No Configuration Files**: No need for custom `postgresql.conf` or `pg_hba.conf` files
3. **Reliable Operation**: Environment variables ensure consistent performance across different environments
4. **Resource Requirements**: Ensure your system has at least 6GB of available RAM and 4 CPU cores for optimal performance
5. **Docker Resources**: Make sure Docker has enough resources allocated (Docker Desktop → Settings → Resources)
6. **Benchmark Isolation**: Each benchmark run starts with a fresh database to ensure consistent results

## 🔄 **Reverting to Default Settings**

To revert to default PostgreSQL settings:

```bash
# Stop optimized container
docker-compose down

# Remove optimized volume
docker volume rm go-crablet_postgres_data

# Edit docker-compose.yaml to remove resource limits and environment variables
# Then restart
docker-compose up -d
```

## 📝 **Troubleshooting**

### **Container Won't Start**
- Check Docker has enough resources allocated
- Verify the environment variables are correctly set in docker-compose.yaml
- Check Docker logs: `docker-compose logs postgres`

### **Poor Performance**
- Verify resource allocation: `docker stats postgres_db`
- Check if environment variables are applied: `docker-compose exec postgres psql -U postgres -c "SHOW shared_buffers;"`
- Ensure Docker has enough CPU and memory allocated

### **Authentication Issues**
- The `POSTGRES_HOST_AUTH_METHOD=trust` setting eliminates password prompts
- If you need password authentication, remove this environment variable

## 🎯 **Benefits of Environment Variable Approach**

1. **Reliability**: No configuration file mounting issues
2. **Consistency**: Same settings across all environments
3. **Simplicity**: No need for custom configuration files
4. **Portability**: Works on any system with Docker
5. **Maintainability**: All settings in one place (docker-compose.yaml)

---

**For more information about PostgreSQL performance tuning, see the [PostgreSQL documentation](https://www.postgresql.org/docs/current/runtime-config-resource.html).** 