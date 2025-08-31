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

