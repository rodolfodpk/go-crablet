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
| **Append** | Tiny | 8,668 ops/sec | 2,142 ops/sec | **4.0x faster** |
| **Append** | Small | 9,096 ops/sec | 2,348 ops/sec | **3.9x faster** |
| **Append** | Medium | 9,061 ops/sec | 2,175 ops/sec | **4.2x faster** |
| **AppendIf No Conflict** | Tiny | 7,604 ops/sec | 2,139 ops/sec | **3.6x faster** |
| **AppendIf No Conflict** | Small | 7,041 ops/sec | 2,172 ops/sec | **3.2x faster** |
| **AppendIf No Conflict** | Medium | 7,290 ops/sec | 2,020 ops/sec | **3.6x faster** |
| **AppendIf With Conflict** | Tiny | 4,221 ops/sec | 1,026 ops/sec | **4.1x faster** |
| **AppendIf With Conflict** | Small | 4,021 ops/sec | 1,140 ops/sec | **3.5x faster** |
| **AppendIf With Conflict** | Medium | 4,179 ops/sec | 1,027 ops/sec | **4.1x faster** |
| **ProjectStream** | Tiny | 8,398 ops/sec | 3,586 ops/sec | **2.3x faster** |
| **ProjectStream** | Small | 10,000 ops/sec | 3,628 ops/sec | **2.8x faster** |
| **ProjectStream** | Medium | 10,000 ops/sec | 3,369 ops/sec | **3.0x faster** |
| **Project** | Tiny | 6,082 ops/sec | 1,558 ops/sec | **3.9x faster** |
| **Project** | Small | 7,119 ops/sec | 1,464 ops/sec | **4.9x faster** |
| **Project** | Medium | 7,213 ops/sec | 1,419 ops/sec | **5.1x faster** |
| **Query** | Tiny | 13,219 ops/sec | 2,041 ops/sec | **6.5x faster** |
| **Query** | Small | 13,236 ops/sec | 2,594 ops/sec | **5.1x faster** |
| **Query** | Medium | 13,242 ops/sec | 2,038 ops/sec | **6.5x faster** |
| **QueryStream** | Tiny | 17,242 ops/sec | 4,539 ops/sec | **3.8x faster** |
| **QueryStream** | Small | 18,691 ops/sec | 4,598 ops/sec | **4.1x faster** |
| **QueryStream** | Medium | 19,407 ops/sec | 4,932 ops/sec | **3.9x faster** |
| **ProjectionLimits** | Tiny | 6,093 ops/sec | 1,281 ops/sec | **4.8x faster** |
| **ProjectionLimits** | Small | 6,837 ops/sec | 1,344 ops/sec | **5.1x faster** |
| **ProjectionLimits** | Medium | 6,933 ops/sec | 1,294 ops/sec | **5.4x faster** |

**Key Performance Insights**:
- **Local PostgreSQL**: 3.2-6.5x faster across all operations with realistic benchmarks
- **Read Operations**: Query and QueryStream show the largest performance gains (5.1-6.5x faster)
- **Write Operations**: Append and AppendIf show significant improvements (3.2-4.2x faster)
- **Projection Operations**: Project and ProjectStream perform 2.3-5.1x faster
- **Realistic Benchmarks**: Performance differences are more pronounced than generic benchmarks
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

