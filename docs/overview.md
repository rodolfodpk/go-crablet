# go-crablet Overview

go-crablet is a Go library **exploring** event sourcing concepts with Dynamic Consistency Boundary (DCB) patterns. This project focuses on learning and experimenting with DCB concepts rather than providing a production-ready solution.

**Note: This is an exploration project for learning and experimenting with DCB concepts, not a production-ready solution.**

## 🚀 Quick Start

### 1. Start Database
```bash
docker-compose up -d
docker-compose ps  # Wait for ready
```

### 2. Run Examples
```bash
# Try the course enrollment example
go run internal/examples/enrollment/main.go

# Or use Makefile
make example-enrollment
```

### 3. Cleanup
```bash
docker-compose down
```

## Core Concepts

### Event Sourcing
- **Events**: Immutable records of what happened
- **Event Store**: Append-only storage for events
- **Projections**: State reconstruction from events
- **DCB**: Dynamic Consistency Boundary for concurrency control

### Core Types

```go
// Fundamental types used throughout the system
type Tag interface {
    GetKey() string
    GetValue() string
}

// Event represents a single event in the event store
type Event struct {
    Type          string    // Event type identifier (e.g., "CourseOffered", "StudentRegistered", "EnrollmentCompleted")
    Tags          []Tag     // Key-value pairs for filtering and categorization
    Data          []byte    // Event payload as JSON bytes
    TransactionID uint64    // Database transaction ID for ordering
    Position      int64     // Position within transaction for ordering
    OccurredAt    time.Time // When the event occurred
}

type InputEvent interface {
    GetType() string
    GetTags() []Tag
    GetData() []byte
}

type Query interface {
    GetItems() []QueryItem
}

type QueryItem interface {
    GetEventTypes() []string
    GetTags() []Tag
}

type AppendCondition struct {
    Query Query
}

type Cursor struct {
    TransactionID uint64
    Position      int64
}
```

### Key Components

#### 1. EventStore (Core API)
```go
type EventStore interface {
    Append(ctx context.Context, events []InputEvent) error
    AppendIf(ctx context.Context, events []InputEvent, condition AppendCondition) error
    Query(ctx context.Context, query Query, after *Cursor) ([]Event, error)
    QueryStream(ctx context.Context, query Query, after *Cursor) (<-chan Event, error)
    Project(ctx context.Context, projectors []StateProjector, after *Cursor) (map[string]any, AppendCondition, error)
    ProjectStream(ctx context.Context, projectors []StateProjector, after *Cursor) (<-chan map[string]any, <-chan AppendCondition, error)
}
```

**Append vs AppendIf: Architectural Approach**

- **Append**: **Event Driven** - High-volume, simple event creation and storage
  - Use when: You need speed and throughput for event streaming, logging, notifications
  - Example: "Student registered for course" → store registration event
  
- **AppendIf**: **Event Sourcing** - Business rule validation + conditional event creation
  - Use when: You need business consistency and validation rules
  - Example: "Enroll student in course" → only if prerequisites met, capacity available
  
- **Trade-off**: Speed/volume vs business integrity and consistency

#### 2. StateProjector (State Reconstruction)
```go
type StateProjector struct {
    ID           string                    // Unique identifier for this projection
    InitialState any                       // Starting state (e.g., empty map, struct, or nil)
    EventTypes   []string                  // Event types to process (e.g., ["CourseOffered", "StudentRegistered"])
    Tags         []Tag                     // Filter events by these tags (e.g., course_id="CS101")
    Project      func(state any, event Event) any  // Function that updates state based on each event
}
```

**Project Function**: This function receives the current state and an event, then returns the updated state. It's called for each event in chronological order to reconstruct the current state.

#### 3. CommandExecutor (Optional High-Level API)
```go
type CommandExecutor interface {
    ExecuteCommand(ctx context.Context, command Command, handler CommandHandler, condition *AppendCondition) ([]InputEvent, error)
}

type Command interface {
    GetType() string
    GetData() []byte
    GetMetadata() map[string]interface{}
}

type CommandHandler interface {
    Handle(ctx context.Context, store EventStore, command Command) ([]InputEvent, error)
}
```

**Note**: CommandExecutor is an optional convenience layer that helps you organize business logic. Instead of manually calling your business logic and then appending events, you can:

1. **Define commands** that represent business actions (like "EnrollStudent", "OfferCourse")
2. **Write handlers** that contain your business logic and decide what events to create
3. **Execute commands** - the system automatically runs your handler, stores the generated events, and keeps an audit trail

This is purely optional - you can always use the core EventStore API directly for full control.

## Code Examples

### 1. Basic Event Storage
```go
// Create EventStore
store, err := dcb.NewEventStore(ctx, pool)
if err != nil {
    log.Fatal(err)
}

// Create and append a course offering event
event := dcb.NewEvent("CourseOffered").
    WithTag("course_id", "CS101").
    WithTag("department", "Computer Science").
    WithData(map[string]any{
        "title": "Introduction to Computer Science",
        "credits": 3,
        "capacity": 30,
    }).
    Build()

err = store.Append(ctx, []dcb.InputEvent{event})
```

### 2. Event Querying
```go
// Query events by tags
query := dcb.NewQueryBuilder().
    WithTag("course_id", "CS101").
    WithType("CourseOffered").
    Build()

events, err := store.Query(ctx, query, nil)
```

### 3. State Projection
```go
// Project course state from events
courseProjector := dcb.ProjectState("course_state", "CourseOffered", "course_id", "CS101", 
    map[string]any{}, 
    func(state any, event dcb.Event) any {
        courseState := state.(map[string]any)
        // Update state based on event
        return courseState
    })

state, condition, err := store.Project(ctx, []dcb.StateProjector{courseProjector}, nil)
```

### 4. DCB Concurrency Control
```go
// Prevent duplicate course offerings
condition := dcb.NewAppendCondition(
    dcb.NewQueryBuilder().
        WithTag("course_id", "CS101").
        WithType("CourseOffered").
        Build(),
)

courseEvent := dcb.NewEvent("CourseOffered").
    WithTag("course_id", "CS101").
    WithData(map[string]any{
        "title": "Introduction to Computer Science",
        "credits": 3,
        "capacity": 30,
    }).
    Build()

// Only succeeds if course doesn't exist
err := store.AppendIf(ctx, []dcb.InputEvent{courseEvent}, condition)
```

### 5. Command Pattern (Optional)
```go
// Define command handler
handler := dcb.CommandHandlerFunc(func(ctx context.Context, store dcb.EventStore, cmd dcb.Command) ([]dcb.InputEvent, error) {
    // Business logic to generate events
    return events, nil
})

// Execute command with concurrency control
events, err := commandExecutor.ExecuteCommand(ctx, command, handler, &condition)
```

## Configuration

### EventStore Configuration

The EventStore can be configured with various settings for append and query operations:

```go
config := dcb.EventStoreConfig{
    // =============================================================================
    // APPEND OPERATIONS CONFIGURATION
    // =============================================================================
    
    // MaxBatchSize controls the maximum number of events per batch
    MaxBatchSize: 1000,
    
    // DefaultAppendIsolation sets PostgreSQL transaction isolation level
    DefaultAppendIsolation: dcb.IsolationLevelReadCommitted,
    
    // AppendTimeout sets maximum time for append operations (milliseconds)
    AppendTimeout: 10000, // 10 seconds
    
    // =============================================================================
    // QUERY OPERATIONS CONFIGURATION  
    // =============================================================================
    
    // DefaultReadIsolation sets PostgreSQL transaction isolation level for read operations
    DefaultReadIsolation: dcb.IsolationLevelReadCommitted,
    
    // QueryTimeout sets maximum time for query operations (milliseconds)
    QueryTimeout: 15000, // 15 seconds
    
    // StreamBuffer sets channel buffer size for streaming operations
    StreamBuffer: 1000,
}

store, err := dcb.NewEventStoreWithConfig(ctx, pool, config)

## Usage Patterns

**Common event batch sizes**: 1 (single), 2-3 (simple), 5-8 (business), 12 (complex workflows)
```





## Database Schema

The library uses PostgreSQL with an optimized schema:

```sql
-- Events table (primary data)
CREATE TABLE events (
    type VARCHAR(64) NOT NULL,
    tags TEXT[] NOT NULL,
    data JSON NOT NULL,
    transaction_id xid8 NOT NULL,
    position BIGSERIAL NOT NULL PRIMARY KEY,
    occurred_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Commands table (audit trail for CommandExecutor)
CREATE TABLE commands (
    transaction_id xid8 NOT NULL PRIMARY KEY,
    type VARCHAR(64) NOT NULL,
    data JSONB NOT NULL,
    metadata JSONB, -- Additional context (user_id, timestamp, request_id, etc.)
    occurred_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

**Schema Features:**
- **Optimized indexing** on tags and event types
- **JSON data storage** for flexible event payloads
- **Transaction tracking** for consistency and debugging
- **Position-based ordering** for reliable event sequencing

## Architecture

The library provides two levels of API:

### Core Layer (EventStore)
```go
// Direct database operations
store.Append(ctx, events)                    // Simple append
store.AppendIf(ctx, events, condition)       // Conditional append with DCB
store.Query(ctx, query, cursor)              // Event querying
store.Project(ctx, projectors, cursor)       // State reconstruction
```

### Optional Layer (CommandExecutor)
```go
// High-level command handling
commandExecutor.ExecuteCommand(ctx, command, handler, condition)
// Internally uses EventStore for all operations
```

### Data Flow
```go
// 1. Events flow directly to PostgreSQL
Client → EventStore → PostgreSQL (events table)

// 2. Commands flow through optional CommandExecutor
Client → CommandExecutor → CommandHandler → EventStore → PostgreSQL (commands + events tables)
```



## Performance Characteristics

The library has been benchmarked with realistic course enrollment system data:

**Local PostgreSQL Performance (ops/sec):**
- **Append**: 3,821-4,199 (single event), 476-678 (batch)
- **AppendIf**: 669-1,432 (no conflict), 100-169 (with conflict)
- **Query**: 294-328 (single), 6,724-7,898 (batch)
- **QueryStream**: 123-328 (single), 49,030-7,898 (batch)
- **Project**: 6,811-36,338 (state reconstruction)
- **ProjectStream**: 6,811-36,338 (streaming state reconstruction)

**Key Performance Insights:**
- **Read operations excel**: Query and QueryStream show high throughput
- **Streaming outperforms batch**: ProjectStream > Project for large datasets
- **Local PostgreSQL**: 3.2-6.5x faster than Docker PostgreSQL
- **Concurrency scaling**: Performance degrades gracefully under load

This library explores event sourcing concepts with DCB concurrency control. It's a learning project that experiments with DCB approaches using PostgreSQL, suitable for understanding event sourcing principles and testing DCB concepts.
