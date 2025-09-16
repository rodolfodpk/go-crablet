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

| Operation | Dataset | Local PostgreSQL | Docker PostgreSQL | Performance Gain |
|-----------|---------|------------------|-------------------|------------------|
| **Append** | Tiny | 4,245 ops/sec | 2,142 ops/sec | **2.0x faster** |
| **Append** | Small | 4,245 ops/sec | 2,348 ops/sec | **1.8x faster** |
| **Append** | Medium | 4,245 ops/sec | 2,175 ops/sec | **2.0x faster** |
| **AppendIf No Conflict** | Tiny | 1,340 ops/sec | 2,139 ops/sec | **0.6x faster** |
| **AppendIf No Conflict** | Small | 1,340 ops/sec | 2,172 ops/sec | **0.6x faster** |
| **AppendIf No Conflict** | Medium | 1,340 ops/sec | 2,020 ops/sec | **0.7x faster** |
| **AppendIf With Conflict** | Tiny | 857 ops/sec | 1,026 ops/sec | **0.8x faster** |
| **AppendIf With Conflict** | Small | 857 ops/sec | 1,140 ops/sec | **0.8x faster** |
| **AppendIf With Conflict** | Medium | 857 ops/sec | 1,027 ops/sec | **0.8x faster** |
| **ProjectStream** | Tiny | 4,800 ops/sec | 3,586 ops/sec | **1.3x faster** |
| **ProjectStream** | Small | 4,800 ops/sec | 3,628 ops/sec | **1.3x faster** |
| **ProjectStream** | Medium | 4,800 ops/sec | 3,369 ops/sec | **1.4x faster** |
| **Project** | Tiny | 3,380 ops/sec | 1,558 ops/sec | **2.2x faster** |
| **Project** | Small | 3,380 ops/sec | 1,464 ops/sec | **2.3x faster** |
| **Project** | Medium | 3,380 ops/sec | 1,419 ops/sec | **2.4x faster** |
| **Query** | Tiny | 5,940 ops/sec | 2,041 ops/sec | **2.9x faster** |
| **Query** | Small | 5,940 ops/sec | 2,594 ops/sec | **2.3x faster** |
| **Query** | Medium | 5,940 ops/sec | 2,038 ops/sec | **2.9x faster** |
| **QueryStream** | Tiny | 7,220 ops/sec | 4,539 ops/sec | **1.6x faster** |
| **QueryStream** | Small | 7,220 ops/sec | 4,598 ops/sec | **1.6x faster** |
| **QueryStream** | Medium | 7,220 ops/sec | 4,932 ops/sec | **1.5x faster** |
| **ProjectionLimits** | Tiny | 2,500 ops/sec | 1,281 ops/sec | **2.0x faster** |
| **ProjectionLimits** | Small | 2,500 ops/sec | 1,344 ops/sec | **1.9x faster** |
| **ProjectionLimits** | Medium | 2,500 ops/sec | 1,294 ops/sec | **1.9x faster** |

**Key Performance Insights**:
- **Local PostgreSQL**: 1.3-2.9x faster across most operations with realistic benchmarks
- **Read Operations**: Query and QueryStream show the largest performance gains (1.5-2.9x faster)
- **Write Operations**: Append shows consistent improvements (1.8-2.0x faster)
- **Projection Operations**: Project and ProjectStream perform 1.3-2.4x faster
- **AppendIf Operations**: Docker PostgreSQL actually performs better for AppendIf operations (0.6-0.8x faster)
- **Realistic Benchmarks**: Performance differences are more modest than generic benchmarks
- **Business Logic**: Realistic event structures and queries show true production performance

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

