# PostgreSQL Performance Optimization for go-crablet Benchmarks

This document explains the PostgreSQL optimizations applied to improve benchmark performance for the go-crablet library.

## 🚀 **Quick Start**

To apply all optimizations and restart PostgreSQL:

```bash
./restart_optimized_db.sh
```

This script will:
1. Stop the current PostgreSQL container
2. Remove the existing database volume
3. Start PostgreSQL with optimized settings
4. Verify connectivity
5. Display resource allocation

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

## ⚙️ **PostgreSQL Configuration Optimizations**

### **Memory Configuration**
```conf
shared_buffers = 256MB                    # PostgreSQL's main memory
effective_cache_size = 1GB                # Estimated OS cache
work_mem = 16MB                           # Per operation memory
maintenance_work_mem = 256MB              # Maintenance operations
```

### **Query Performance**
```conf
default_statistics_target = 100           # Better query planning
random_page_cost = 1.1                    # Optimized for SSD
effective_io_concurrency = 200            # Parallel I/O operations
```

### **Parallel Processing**
```conf
max_worker_processes = 8                  # Total background processes
max_parallel_workers_per_gather = 4       # Parallel workers per query
max_parallel_workers = 8                  # Total parallel workers
max_parallel_maintenance_workers = 4      # Parallel maintenance workers
```

### **Checkpoint Optimization**
```conf
checkpoint_completion_target = 0.9        # Spread checkpoints over time
wal_buffers = 16MB                        # WAL buffer size
checkpoint_timeout = 5min                 # Checkpoint interval
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
  POSTGRES_SHARED_BUFFERS: 256MB
  POSTGRES_EFFECTIVE_CACHE_SIZE: 1GB
  POSTGRES_WORK_MEM: 16MB
  POSTGRES_MAINTENANCE_WORK_MEM: 256MB
  # ... additional optimizations
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

1. **Database Reset**: The optimization script removes the existing database volume, so any existing data will be lost.

2. **Resource Requirements**: Ensure your system has at least 6GB of available RAM and 4 CPU cores for optimal performance.

3. **Docker Resources**: Make sure Docker has enough resources allocated (Docker Desktop → Settings → Resources).

4. **Benchmark Isolation**: Each benchmark run starts with a fresh database to ensure consistent results.

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
- Verify the postgresql.conf file exists
- Check Docker logs: `docker-compose logs postgres`

### **Poor Performance**
- Verify resource allocation: `docker stats postgres_db`
- Check PostgreSQL configuration: `docker-compose exec postgres psql -U postgres -c "SHOW ALL;"`
- Monitor system resources during benchmarks

### **Connection Issues**
- Wait for PostgreSQL to fully start (usually 10-15 seconds)
- Check health status: `docker-compose ps`
- Verify port availability: `netstat -an | grep 5432`

---

**For more information about PostgreSQL performance tuning, see the [PostgreSQL documentation](https://www.postgresql.org/docs/current/runtime-config-resource.html).** 