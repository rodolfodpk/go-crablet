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
| **[Docker PostgreSQL Performance](performance-docker.md)** | Docker PostgreSQL 16 | Tiny, Small, Medium | Containerized benchmark results |

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

**Local PostgreSQL vs Docker PostgreSQL Performance Comparison**:

| Operation | Dataset | Local PostgreSQL | Docker PostgreSQL | Performance Gain |
|-----------|---------|------------------|-------------------|------------------|
| **Append** | Tiny | 4,110 ops/sec | 2,406 ops/sec | **1.7x faster** |
| **Append** | Small | 4,380 ops/sec | 2,110 ops/sec | **2.1x faster** |
| **Append** | Medium | 4,100 ops/sec | 2,132 ops/sec | **1.9x faster** |
| **AppendIf No Conflict** | Tiny | 1,164 ops/sec | 2,054 ops/sec | **1.8x faster** |
| **AppendIf No Conflict** | Small | 864 ops/sec | 1,858 ops/sec | **2.2x faster** |
| **AppendIf No Conflict** | Medium | 1,319 ops/sec | 2,061 ops/sec | **1.6x faster** |
| **AppendIf With Conflict** | Tiny | 1,080 ops/sec | 1,132 ops/sec | **1.0x faster** |
| **AppendIf With Conflict** | Small | 1,180 ops/sec | 1,171 ops/sec | **1.0x faster** |
| **AppendIf With Conflict** | Medium | 1,179 ops/sec | 1,088 ops/sec | **1.1x faster** |
| **ProjectStream** | Tiny | 5,665 ops/sec | 4,044 ops/sec | **1.4x faster** |
| **ProjectStream** | Small | 5,665 ops/sec | 4,022 ops/sec | **1.4x faster** |
| **ProjectStream** | Medium | 5,665 ops/sec | 3,992 ops/sec | **1.4x faster** |
| **Project** | Tiny | 3,564 ops/sec | 1,766 ops/sec | **2.0x faster** |
| **Project** | Small | 3,564 ops/sec | 1,704 ops/sec | **2.1x faster** |
| **Project** | Medium | 3,564 ops/sec | 1,630 ops/sec | **2.2x faster** |

**Key Performance Insights**:
- **Local PostgreSQL**: ~1.4-2.2x faster for most operations
- **Docker PostgreSQL**: Additional latency from containerization
- **Memory**: Local uses system memory, Docker has container limits
- **I/O**: Local has direct disk access, Docker uses volume mounts
- **Network**: Local uses localhost, Docker uses container networking
- **Resource Contention**: Local has full system resources, Docker shares with host

## Environment Switching

To switch between PostgreSQL environments:

### Switch to Local PostgreSQL
```bash
# Stop Docker PostgreSQL
docker-compose down

# Start Local PostgreSQL
brew services start postgresql@16

# Run benchmarks
cd internal/benchmarks
go test -bench="BenchmarkAppend" -benchmem -benchtime=2s -timeout=10m .
```

### Switch to Docker PostgreSQL
```bash
# Stop Local PostgreSQL
brew services stop postgresql@16

# Start Docker PostgreSQL
docker-compose up -d

# Run benchmarks
cd internal/benchmarks
go test -bench="BenchmarkAppend" -benchmem -benchtime=2s -timeout=10m .
```

## Dataset Sizes

| Dataset | Courses | Students | Enrollments |
|---------|---------|----------|-------------|
| **Tiny** | 5 | 10 | 20 |
| **Small** | 500 | 5,000 | 25,000 |
| **Medium** | 1,000 | 10,000 | 50,000 |

