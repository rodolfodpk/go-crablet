# Performance Guide

## PostgreSQL Setup Options

| Environment | Setup Command | Use Case | Status |
|-------------|---------------|----------|--------|
| **🐳 Docker PostgreSQL** | `docker-compose up -d` | Containerized environment | Available |
| **💻 Local PostgreSQL** | `brew services start postgresql@16` | Development environment | **Currently Active** |

## Current Setup

**Local PostgreSQL 16** is currently active and running via Homebrew services. This provides the best performance for development and benchmarking.

### Local PostgreSQL Details
- **Version**: PostgreSQL 16 (LTS)
- **Installation**: Homebrew (`brew install postgresql@16`)
- **Service**: `brew services start postgresql@16`
- **Database**: `crablet` (created automatically)
- **Port**: 5432 (default)
- **Performance**: Optimized for local development

## Performance Guides

| Guide | Environment | Datasets | Description |
|-------|-------------|----------|-------------|
| **[Local PostgreSQL Performance](performance-local.md)** | Local PostgreSQL 16 | Tiny, Small, Medium | **Latest benchmark results** |
| **[Docker PostgreSQL Performance](performance-docker.md)** | Docker PostgreSQL 16.10 | Tiny, Small, Medium | Containerized benchmark results |

## Local vs Docker PostgreSQL Comparison

| Aspect | Local PostgreSQL | Docker PostgreSQL |
|--------|------------------|-------------------|
| **Performance** | **Faster** - Direct system access | Slower - Container overhead |
| **Setup** | One-time Homebrew install | Docker Compose required |
| **Resource Usage** | Native system resources | Containerized with limits |
| **Persistence** | Native PostgreSQL data directory | Docker volumes |
| **Network** | Localhost (127.0.0.1:5432) | Docker network |
| **Configuration** | System-level PostgreSQL config | Container environment |
| **Development** | **Recommended** for local development | Good for CI/CD and testing |

### Performance Differences

**Local PostgreSQL vs Docker PostgreSQL Performance Comparison (Realistic Benchmarks)**:

**Docker PostgreSQL benchmark data from `go_benchmarks_20250916_213723.txt` (September 16, 2025)**

| Operation | Dataset | Local PostgreSQL | Docker PostgreSQL | Performance Gain |
|-----------|---------|------------------|-------------------|------------------|
| **Append** | Tiny | 3,870 ops/sec | 820 ops/sec | **4.7x faster** |
| **Append** | Small | 3,870 ops/sec | 820 ops/sec | **4.7x faster** |
| **Append** | Medium | 3,625 ops/sec | 820 ops/sec | **4.4x faster** |
| **AppendIf No Conflict** | Tiny | 1,164 ops/sec | 685 ops/sec | **1.7x faster** |
| **AppendIf No Conflict** | Small | 1,220 ops/sec | 680 ops/sec | **1.8x faster** |
| **AppendIf No Conflict** | Medium | 1,103 ops/sec | 680 ops/sec | **1.6x faster** |
| **Project** | Tiny | 3,433 ops/sec | 502 ops/sec | **6.8x faster** |
| **Project** | Small | 3,348 ops/sec | 537 ops/sec | **6.2x faster** |
| **Project** | Medium | 3,163 ops/sec | 537 ops/sec | **5.9x faster** |
| **Query** | Tiny | 5,750 ops/sec | 768 ops/sec | **7.5x faster** |
| **Query** | Small | 5,696 ops/sec | 680 ops/sec | **8.4x faster** |
| **Query** | Medium | 6,147 ops/sec | 680 ops/sec | **9.0x faster** |

**Key Performance Insights**:
- **Local PostgreSQL**: 1.6-9.0x faster across all operations compared to Docker PostgreSQL
- **Query Operations**: Show the largest performance gains (7.5-9.0x faster on Local)
- **Project Operations**: Significant improvements (5.9-6.8x faster on Local)
- **Append Operations**: Consistent performance gains (4.4-4.7x faster on Local)
- **AppendIf Operations**: Moderate improvements (1.6-1.8x faster on Local)
- **Docker Overhead**: Containerization adds significant latency to all operations
- **Realistic Benchmarks**: Performance differences are more pronounced with business scenarios

## Environment Switching

To switch between PostgreSQL environments:

### Switch to Local PostgreSQL
```bash
# Stop Docker PostgreSQL
docker-compose down

# Start Local PostgreSQL
brew services start postgresql@16

# Run realistic benchmarks
cd internal/benchmarks
go test -bench="BenchmarkAppend_.*_Realistic" -benchmem -benchtime=2s -timeout=10m .
```

### Switch to Docker PostgreSQL
```bash
# Stop Local PostgreSQL
brew services stop postgresql@16

# Start Docker PostgreSQL
docker-compose up -d

# Run realistic benchmarks
cd internal/benchmarks
go test -bench="BenchmarkAppend_.*_Realistic" -benchmem -benchtime=2s -timeout=10m .
```

## Dataset Sizes

| Dataset | Courses | Students | Enrollments |
|---------|---------|----------|-------------|
| **Tiny** | 5 | 10 | 20 |
| **Small** | 500 | 5,000 | 25,000 |
| **Medium** | 1,000 | 10,000 | 50,000 |

