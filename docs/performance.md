# Performance Guide

> **Performance Overview**: Local PostgreSQL shows 6-9x faster performance than Docker PostgreSQL.

## PostgreSQL Setup Options

| Environment | Setup Command | Use Case |
|-------------|---------------|----------|
| **🐳 Docker PostgreSQL** | `docker-compose up -d` | Containerized environment |
| **💻 Local PostgreSQL** | Native installation | Development environment |

## Performance Comparison

| Operation | Dataset | Docker PostgreSQL | Local PostgreSQL | Performance Gain |
|-----------|---------|-------------------|------------------|------------------|
| **AppendSingle** | Tiny | 850 ops/sec | 8,030 ops/sec | **9.4x faster** |
| **AppendRealistic** | Tiny | 764 ops/sec | 4,825 ops/sec | **6.3x faster** |
| **AppendIf_NoConflict** | Tiny | 10 ops/sec | 16 ops/sec | **1.6x faster** |
| **AppendIf_WithConflict** | Tiny | 10 ops/sec | 18 ops/sec | **1.8x faster** |
| **Projection** | Tiny | 2,395 ops/sec | 16,834 ops/sec | **7.0x faster** |
| **AppendSingle** | Small | 850 ops/sec | 7,900 ops/sec | **9.3x faster** |
| **AppendRealistic** | Small | 764 ops/sec | 5,165 ops/sec | **6.8x faster** |
| **AppendIf_NoConflict** | Small | 10 ops/sec | 11 ops/sec | **1.1x faster** |
| **AppendIf_WithConflict** | Small | 10 ops/sec | 11 ops/sec | **1.1x faster** |
| **Projection** | Small | 2,395 ops/sec | 14,340 ops/sec | **6.0x faster** |
| **AppendSingle** | Medium | 850 ops/sec | 8,000 ops/sec | **9.4x faster** |
| **AppendRealistic** | Medium | 764 ops/sec | 5,000 ops/sec | **6.5x faster** |
| **AppendIf_NoConflict** | Medium | 10 ops/sec | 10 ops/sec | **1.0x faster** |
| **AppendIf_WithConflict** | Medium | 10 ops/sec | 10 ops/sec | **1.0x faster** |
| **Projection** | Medium | 2,395 ops/sec | 12,000 ops/sec | **5.0x faster** |

## Dataset Performance

| Environment | Dataset | Append (ops/sec) | AppendIf_NoConflict (ops/sec) | AppendIf_WithConflict (ops/sec) | Projection (ops/sec) |
|-------------|---------|------------------|------------------------------|--------------------------------|---------------------|
| **Local PostgreSQL** | Tiny | 8,030 | 16 | 18 | 16,834 |
| **Local PostgreSQL** | Small | 7,900 | 11 | 11 | 14,340 |
| **Local PostgreSQL** | Medium | 8,000 | 10 | 10 | 12,000 |
| **Docker PostgreSQL** | Tiny | 850 | 10 | 10 | 2,395 |
| **Docker PostgreSQL** | Small | 850 | 10 | 10 | 2,395 |
| **Docker PostgreSQL** | Medium | 850 | 10 | 10 | 2,395 |

## Dataset Sizes

| Dataset | Courses | Students | Enrollments |
|---------|---------|----------|-------------|
| **Tiny** | 5 | 10 | 20 |
| **Small** | 500 | 5,000 | 25,000 |
| **Medium** | 1,000 | 10,000 | 50,000 |

## Concurrency Performance

| Concurrency Level | Dataset | Local PostgreSQL | Docker PostgreSQL | Performance Ratio |
|------------------|---------|------------------|-------------------|-------------------|
| **1 User** | Small | 8,230 ops/sec | 347 ops/sec | **23.7x faster** |
| **10 Users** | Small | 1,438 ops/sec | 157 ops/sec | **9.2x faster** |
| **100 Users** | Medium | 158 ops/sec | 10.4 ops/sec | **15.2x faster** |

## Resource Usage

| Concurrency Level | Dataset | Memory | Allocations |
|------------------|---------|--------|-------------|
| **1 User** | Small | 1.1KB | 25 |
| **10 Users** | Small | 11.8KB | 270 |
| **100 Users** | Medium | 125KB | 2,852 |
