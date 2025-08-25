# Performance Guide

> **🚀 Performance Update**: Recent benchmark improvements show significantly better AppendIf performance (124 ops/sec vs previous 0.08 ops/sec) after fixing database event accumulation issues. Results now reflect realistic business rule validation overhead.

## Test Environment
- **Platform**: macOS (darwin 23.6.0) with Apple M1 Pro
- **Database**: PostgreSQL with 50-connection pool
- **Test Data**: Runtime-generated datasets with controlled past event counts

## Dataset-Specific Performance Results

Choose your dataset size to view detailed performance metrics:

### **📊 [Tiny Dataset Performance](./performance-tiny.md)**
- **Size**: 5 courses, 10 students, 17 enrollments
- **Use Case**: Quick testing, development, fast feedback
- **Past Events**: 10 events for AppendIf testing
- **Performance**: Best case scenarios, minimal data volume

### **📊 [Small Dataset Performance](./performance-small.md)**
- **Size**: 1,000 courses, 10,000 students, 49,871 enrollments  
- **Use Case**: Realistic testing, production planning, scalability analysis
- **Past Events**: 100 events for AppendIf testing
- **Performance**: Real-world scenarios, data volume impact

## Performance Summary

**Key Performance Insights**:
- **Append**: 2,000+ ops/sec (single event), scales well with concurrency
- **AppendIf**: 15-124 ops/sec depending on dataset size and conflict scenarios
- **Read**: 400-5,000+ ops/sec depending on query complexity and data volume
- **Projection**: 100-700 ops/sec for state reconstruction from event streams

**Dataset Impact**:
- **Tiny Dataset**: Best performance, minimal resource usage, ideal for development
- **Small Dataset**: Realistic performance, shows data volume impact, production planning

**Concurrency Scaling**: All operations tested with 1, 10, and 100 concurrent users to measure performance degradation under load.

**For detailed performance tables and specific metrics, see the dataset-specific pages above.**
