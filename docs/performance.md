# Performance Guide

> **Performance Overview**: Local PostgreSQL shows 6-9x faster performance than Docker PostgreSQL.

## PostgreSQL Setup Options

### **🐳 [Docker PostgreSQL Performance](./performance-docker.md)**
- **Environment**: Docker containerized PostgreSQL
- **Setup**: `docker-compose up -d`

### **💻 [Local PostgreSQL Performance](./performance-local.md)**
- **Environment**: Native PostgreSQL installation (macOS/Homebrew)
- **Setup**: Native PostgreSQL installation

## Performance Comparison

| Operation | Dataset | Docker PostgreSQL | Local PostgreSQL | Performance Gain |
|-----------|---------|-------------------|------------------|------------------|
| **AppendSingle** | Tiny | 850 ops/sec | 8,030 ops/sec | **9.4x faster** |
| **AppendRealistic** | Tiny | 764 ops/sec | 4,825 ops/sec | **6.3x faster** |
| **AppendIf** | Tiny | 10 ops/sec | 16 ops/sec | **1.6x faster** |
| **Projection** | Tiny | 2,395 ops/sec | 16,834 ops/sec | **7.0x faster** |
| **AppendSingle** | Small | 850 ops/sec | 7,900 ops/sec | **9.3x faster** |
| **AppendRealistic** | Small | 764 ops/sec | 5,165 ops/sec | **6.8x faster** |
| **AppendIf** | Small | 10 ops/sec | 11 ops/sec | **1.1x faster** |
| **Projection** | Small | 2,395 ops/sec | 14,340 ops/sec | **6.0x faster** |
| **AppendSingle** | Medium | 850 ops/sec | 8,000 ops/sec | **9.4x faster** |
| **AppendRealistic** | Medium | 764 ops/sec | 5,000 ops/sec | **6.5x faster** |
| **AppendIf** | Medium | 10 ops/sec | 10 ops/sec | **1.0x faster** |
| **Projection** | Medium | 2,395 ops/sec | 12,000 ops/sec | **5.0x faster** |

## Dataset Performance

### **Local PostgreSQL**
- **Tiny**: 8,030 ops/sec Append, 16 ops/sec AppendIf, 16,834 ops/sec Projection
- **Small**: 7,900 ops/sec Append, 11 ops/sec AppendIf, 14,340 ops/sec Projection
- **Medium**: 8,000 ops/sec Append, 10 ops/sec AppendIf, 12,000 ops/sec Projection

### **Docker PostgreSQL**
- **Tiny**: 850 ops/sec Append, 10 ops/sec AppendIf, 2,395 ops/sec Projection
- **Small**: 850 ops/sec Append, 10 ops/sec AppendIf, 2,395 ops/sec Projection
- **Medium**: 850 ops/sec Append, 10 ops/sec AppendIf, 2,395 ops/sec Projection

### **Dataset Sizes**
- **Tiny**: 5 courses, 10 students, 20 enrollments
- **Small**: 500 courses, 5,000 students, 25,000 enrollments
- **Medium**: 1,000 courses, 10,000 students, 50,000 enrollments

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

## Setup Guides

- **🐳 [Docker PostgreSQL Setup](./benchmark-setup-docker.md)**
- **💻 [Local PostgreSQL Setup](./benchmark-setup-local.md)**
- **📊 [Performance Comparison](./performance-comparison.md)**
