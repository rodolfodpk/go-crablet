# Package Reorganization: Separating Core from Implementation

This document explains the reorganization of the `pkg/dcb` package to separate core interfaces from database-specific implementations.

## Current Structure

Currently, everything is in `pkg/dcb/`:
- Core interfaces (`EventStore`, `ChannelEventStore`)
- Core types (`Event`, `InputEvent`, `Query`, etc.)
- PostgreSQL implementation (`eventStore` struct with `*pgxpool.Pool`)
- All helper functions and validation

## Proposed Structure

The reorganization creates a cleaner separation:

```
pkg/dcb/
├── core.go          # Core interfaces and types only
├── helpers.go       # Core helper functions
├── errors.go        # Core error types
└── postgres/
    ├── store.go     # PostgreSQL implementation
    ├── channel_streaming.go # Channel streaming implementation
    └── README.md    # PostgreSQL-specific documentation
```

## Benefits

### 1. **Dependency Separation**
- **Core package**: No database dependencies
- **PostgreSQL package**: Only PostgreSQL dependencies
- **Future SQLite package**: Only SQLite dependencies

### 2. **Consumer Benefits**
```go
// Current approach - pulls in PostgreSQL even if not needed
import "go-crablet/pkg/dcb"

// Proposed approach - only import what you need
import (
    "go-crablet/pkg/dcb"           // Core interfaces only
    postgres "go-crablet/pkg/dcb/postgres"  // PostgreSQL implementation
)
```

### 3. **Clear Separation of Concerns**
- **Core package**: Defines what an event store should do
- **Implementation packages**: Define how to do it with specific databases

### 4. **Easier Testing**
- Mock the core interfaces without database setup
- Test implementations independently
- No circular dependencies

### 5. **Future-Proofing**
- Add new databases without changing core interfaces
- Maintain backward compatibility
- Follow Go idioms like `database/sql`

## Example Usage

### Current Approach
```go
import (
    "go-crablet/pkg/dcb"
    "github.com/jackc/pgx/v5/pgxpool"
)

func main() {
    pool, _ := pgxpool.New(ctx, "postgres://...")
    store, _ := dcb.NewEventStore(ctx, pool)  // PostgreSQL dependency included
    
    events, _ := store.Read(ctx, query, nil)
}
```

### Proposed Approach
```go
import (
    "go-crablet/pkg/dcb"           // Core interfaces only
    postgres "go-crablet/pkg/dcb/postgres"  // PostgreSQL implementation
    "github.com/jackc/pgx/v5/pgxpool"
)

func main() {
    pool, _ := pgxpool.New(ctx, "postgres://...")
    store, _ := postgres.NewEventStore(ctx, pool)  // Explicit PostgreSQL choice
    
    events, _ := store.Read(ctx, query, nil)
}
```

## Implementation Status

### ✅ Completed
- Created `pkg/dcb/postgres/` package
- Moved PostgreSQL implementation to `postgres/store.go`
- Moved channel streaming to `postgres/channel_streaming.go`
- Created example showing both approaches

### 🔄 In Progress
- Core package cleanup (removing PostgreSQL dependencies)
- Updating examples to use postgres package
- Documentation updates

### 📋 Planned
- SQLite implementation example
- Migration guide for existing users
- Performance benchmarks comparison

## Migration Strategy

The reorganization is designed to be **backward compatible**:

1. **Phase 1**: Add postgres package alongside existing implementation
2. **Phase 2**: Update examples and documentation to use postgres package
3. **Phase 3**: Deprecate PostgreSQL implementation in core package
4. **Phase 4**: Remove PostgreSQL implementation from core package

## Dependency Graph

### Current
```
Your App
└── go-crablet/pkg/dcb
    └── github.com/jackc/pgx/v5 (always included)
```

### Proposed
```
Your App
├── go-crablet/pkg/dcb (core interfaces only)
└── go-crablet/pkg/dcb/postgres (PostgreSQL implementation)
    └── github.com/jackc/pgx/v5 (only if using PostgreSQL)
```

## Testing the Reorganization

Run the example to see both approaches:

```bash
cd internal/examples/postgres_package
go run main.go
```

This demonstrates:
- Current core package approach
- Proposed postgres package approach
- Dependency separation benefits

## Conclusion

The reorganization provides:
- **Better separation of concerns**
- **Reduced dependencies for consumers**
- **Easier testing and mocking**
- **Future extensibility**
- **Backward compatibility**

This follows Go best practices and patterns like `database/sql` and `database/sql/driver`. 