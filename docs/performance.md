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

**Local PostgreSQL vs Docker PostgreSQL Performance Comparison (Realistic Benchmarks)**:

**Note**: Docker PostgreSQL comparison data will be added after running Docker benchmarks.

| Operation | Dataset | Local PostgreSQL | Docker PostgreSQL | Performance Gain |
|-----------|---------|------------------|-------------------|------------------|
| **Append** | Tiny | 3,870 ops/sec | TBD | **TBD** |
| **Append** | Small | 3,870 ops/sec | TBD | **TBD** |
| **Append** | Medium | 3,625 ops/sec | TBD | **TBD** |
| **AppendIf No Conflict** | Tiny | 1,164 ops/sec | TBD | **TBD** |
| **AppendIf No Conflict** | Small | 1,220 ops/sec | TBD | **TBD** |
| **AppendIf No Conflict** | Medium | 1,103 ops/sec | TBD | **TBD** |
| **Project** | Tiny | 3,433 ops/sec | TBD | **TBD** |
| **Project** | Small | 3,348 ops/sec | TBD | **TBD** |
| **Project** | Medium | 3,163 ops/sec | TBD | **TBD** |
| **Query** | Tiny | 5,750 ops/sec | TBD | **TBD** |
| **Query** | Small | 5,696 ops/sec | TBD | **TBD** |
| **Query** | Medium | 6,147 ops/sec | TBD | **TBD** |

**Key Performance Insights**:
- **Local PostgreSQL**: Current performance data available for all operations
- **Append Operations**: 3,625-3,870 ops/sec (single user, single event)
- **AppendIf Operations**: 1,103-1,220 ops/sec (single user, single event)
- **Project Operations**: 3,163-3,433 ops/sec (single user)
- **Query Operations**: 5,696-6,147 ops/sec (single user)
- **Docker Comparison**: TBD - will be updated after running Docker benchmarks
- **Realistic Benchmarks**: Performance data from actual business scenarios

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

