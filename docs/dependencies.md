# Dependencies

go-crablet uses a minimal set of carefully selected dependencies to provide robust functionality while maintaining a small footprint.

## Core Dependencies

### **[pgx/v5](https://github.com/jackc/pgx)** - PostgreSQL driver and toolkit

**Purpose**: High-performance PostgreSQL driver for Go

**Why we use it**:
- **Performance**: Significantly faster than the standard `database/sql` driver
- **JSONB Support**: Native support for PostgreSQL's JSONB type with efficient serialization
- **Connection Pooling**: Built-in connection pooling for optimal performance
- **Prepared Statements**: Automatic prepared statement caching
- **Type Safety**: Strong typing for PostgreSQL-specific types
- **Active Development**: Well-maintained with regular updates

**Features we leverage**:
- JSONB for flexible tag storage
- Connection pooling for concurrent operations
- Prepared statements for query optimization
- Transaction support for atomic operations

### **[go.jetify.com/typeid](https://github.com/jetify-com/typeid)** - Type-safe, K-sortable unique identifiers

**Purpose**: Provides meaningful, human-readable event IDs with tag-based prefixes

**Why we use it**:
- **Debugging**: Event IDs include entity information (e.g., `course_id_01jxfvsth3ezwvxjec1xp4ejvb`)
- **K-Sortable**: Chronological ordering for efficient event processing
- **Type Safety**: Compile-time type checking for ID generation
- **Custom Prefixes**: Tag-based prefixes for meaningful identification
- **Compact**: 26-character UUID part with configurable prefix length

**Features we leverage**:
- Tag-based prefix generation from event tags
- K-sortable UUIDs for chronological ordering
- Type-safe ID generation and validation
- Compact representation within VARCHAR(64) limits

## Testing Dependencies

### **[Ginkgo](https://github.com/onsi/ginkgo)** - BDD testing framework

**Purpose**: Behavior-driven development testing with descriptive test structure

**Why we use it**:
- **BDD Style**: Descriptive test structure that reads like specifications
- **Parallel Execution**: Efficient parallel test execution
- **Rich Reporting**: Comprehensive test reporting and failure analysis
- **Integration**: Seamless integration with Gomega for assertions
- **Community**: Well-established testing framework in the Go ecosystem

### **[Gomega](https://github.com/onsi/gomega)** - Matcher library for Ginkgo

**Purpose**: Rich assertion library with expressive matchers

**Why we use it**:
- **Expressive Matchers**: Rich set of built-in matchers for common assertions
- **Custom Matchers**: Easy creation of domain-specific matchers
- **Integration**: Designed to work seamlessly with Ginkgo
- **Readable**: Assertions that read like natural language
- **Comprehensive**: Covers all common testing scenarios

### **[testcontainers-go](https://github.com/testcontainers/testcontainers-go)** - Container-based testing

**Purpose**: Automated PostgreSQL container management for integration tests

**Why we use it**:
- **Isolation**: Each test gets a clean PostgreSQL instance
- **Automation**: Automatic container lifecycle management
- **Cross-Platform**: Works consistently across different operating systems
- **Cleanup**: Automatic cleanup after tests complete
- **Realistic**: Tests against actual PostgreSQL database

## Dependency Philosophy

We follow these principles when selecting dependencies:

### **Minimal Surface Area**
- Only essential dependencies that provide core functionality
- Avoid transitive dependencies that add unnecessary complexity
- Prefer standard library solutions when possible

### **Active Maintenance**
- Dependencies with active development and community support
- Regular updates and security patches
- Strong community backing and documentation

### **Performance Focus**
- High-performance libraries that don't compromise on speed
- Efficient memory usage and resource consumption
- Optimized for production workloads

### **Type Safety**
- Strong typing and compile-time safety where possible
- Clear interfaces and error handling
- Reduced runtime errors through compile-time checks

> **Note:** As this is an alpha library, breaking changes to dependencies and APIs may occur as the project evolves. We recommend pinning versions and reviewing release notes when upgrading.

## Version Compatibility

### **Go Version**
- **Minimum**: Go 1.24+
- **Rationale**: Uses modern Go features like generics for type-safe operations
- **Features**: Generics for type-safe ID generation and query building

### **PostgreSQL Version**
- **Minimum**: PostgreSQL 12+
- **Rationale**: JSONB support and advanced features required
- **Features**: JSONB for flexible tag storage, advanced indexing

### **Dependency Updates**
- **Frequency**: Regular updates to latest stable versions
- **Process**: Automated dependency updates with comprehensive testing
- **Compatibility**: All updates tested for backward compatibility

## Security Considerations

### **Dependency Scanning**
- Regular security scanning of all dependencies
- Automated vulnerability detection and reporting
- Prompt updates for security-related issues

### **Minimal Attack Surface**
- Fewer dependencies reduce potential attack vectors
- Carefully vetted dependencies with strong security practices
- Regular security audits and updates

## Performance Impact

### **Memory Usage**
- Minimal memory footprint from dependencies
- Efficient resource utilization
- No unnecessary memory allocations

### **Startup Time**
- Fast library initialization
- Minimal startup overhead
- Efficient connection pooling

### **Runtime Performance**
- High-performance database operations
- Optimized query execution
- Efficient event processing

All dependencies are regularly updated and tested for compatibility, performance, and security. 