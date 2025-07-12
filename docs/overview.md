# Overview: Dynamic Consistency Boundary (DCB) in go-crablet

go-crablet is a Go library for event sourcing, exploring concepts inspired by the [Dynamic Consistency Boundary (DCB)](https://dcb.events/) pattern. We're exploring how DCB might enable you to:

- Project multiple states and check business invariants in a single query
- Use tag-based, OR-combined queries for cross-entity consistency
- Enforce optimistic concurrency with combined append conditions
- Execute commands with automatic event generation using the CommandExecutor pattern

## Key Concepts

- **Batch Projection**: Project multiple states (decision model) in one database query
- **Combined Append Condition**: Use a single, OR-combined query for optimistic locking
- **Tag-based Queries**: Flexible, cross-entity queries using tags
- **Streaming**: Process events efficiently for large datasets
- **Transaction-based Ordering**: Uses PostgreSQL transaction IDs for true event ordering
- **Atomic Command Execution**: Execute commands with handler-based event generation and command tracking

## Core Interfaces

### EventStore Interface

The `EventStore` is the primary interface that users interact with directly:

```go
type EventStore interface {
    // Query reads events matching the query with optional cursor
    // cursor == nil: query from beginning of stream
    // cursor != nil: query from specified cursor position (EXCLUSIVE - events after cursor, not including cursor)
    Query(ctx context.Context, query Query, cursor *Cursor) ([]Event, error)

    // QueryStream creates a channel-based stream of events matching a query with optional cursor
    // cursor == nil: stream from beginning of stream
    // cursor != nil: stream from specified cursor position (EXCLUSIVE - events after cursor, not including cursor)
    // This is optimized for large datasets and provides backpressure through channels
    // for efficient memory usage and Go-idiomatic streaming
    QueryStream(ctx context.Context, query Query, cursor *Cursor) (<-chan Event, error)

    // Append appends events to the store with optional condition
    // condition == nil: unconditional append
    // condition != nil: conditional append (optimistic locking)
    Append(ctx context.Context, events []InputEvent, condition *AppendCondition) error

    // Project projects multiple states using projectors with optional cursor
    // cursor == nil: project from beginning of stream
    // cursor != nil: project from specified cursor position (EXCLUSIVE - events after cursor, not including cursor)
    // Returns final aggregated states and append condition for optimistic locking
    Project(ctx context.Context, projectors []StateProjector, cursor *Cursor) (map[string]any, AppendCondition, error)

    // ProjectStream projects multiple states using channel-based streaming with optional cursor
    // cursor == nil: stream from beginning of stream
    // cursor != nil: stream from specified cursor position (EXCLUSIVE - events after cursor, not including cursor)
    // This is optimized for large datasets and provides backpressure through channels
    // for efficient memory usage and Go-idiomatic streaming
    ProjectStream(ctx context.Context, projectors []StateProjector, cursor *Cursor) (<-chan map[string]any, <-chan AppendCondition, error)

    // GetConfig returns the current EventStore configuration
    GetConfig() EventStoreConfig

    // GetPool returns the underlying database pool
    GetPool() *pgxpool.Pool
}
```

### CommandExecutor Interface

The `CommandExecutor` is an optional wrapper around `EventStore` that provides command-driven event generation:

```go
type CommandExecutor interface {
    // ExecuteCommand executes a command and generates events atomically
    // The handler receives the EventStore to perform its own projections and business logic
    ExecuteCommand(ctx context.Context, command Command, handler CommandHandler, condition *AppendCondition) error
}

type CommandHandler interface {
    // Handle processes a command and generates events
    // The handler has access to the EventStore for projection and business logic
    Handle(ctx context.Context, store EventStore, command Command) []InputEvent
}
```

### Usage Pattern

Users typically follow this pattern:

```go
// 1. Create EventStore (primary interface)
store, err := dcb.NewEventStore(ctx, pool)

// 2. Optionally create CommandExecutor from EventStore
commandExecutor := dcb.NewCommandExecutor(store)

// 3. Use either interface as needed
// Direct EventStore usage:
err = store.Append(ctx, events, condition)

// Command-driven usage:
err = commandExecutor.ExecuteCommand(ctx, command, handler, condition)
```

### Supporting Types

```go
type Cursor struct {
    TransactionID uint64 `json:"transaction_id"`
    Position      int64  `json:"position"`
}

type Command interface {
    GetType() string
    GetData() []byte
    GetMetadata() map[string]interface{}
}
```

## Transaction ID Ordering

go-crablet uses PostgreSQL's `xid8` transaction IDs for event ordering and optimistic locking:

- **True ordering**: No gaps or out-of-order events
- **Optimistic locking**: Uses transaction IDs for conflict detection
- **Cursor-based**: Combines `(transaction_id, position)` for precise positioning

## DCB Decision Model Pattern

We're exploring how a Dynamic Consistency Boundary decision model might work:

1. Define projectors for each business rule or invariant
2. Project all states in a single query
3. Build a combined append condition
4. Append new events only if all invariants still hold

## Example: Course Subscription

```go
projectors := []dcb.StateProjector{
    {
        ID: "courseExists",
        Query: dcb.NewQuery(dcb.NewTags("course_id", courseID), "CourseDefined"),
        InitialState: false,
        TransitionFn: func(state any, event dcb.Event) any {
            return true // If we see a CourseDefined event, course exists
        },
    },
    {
        ID: "numSubscriptions",
        Query: dcb.NewQuery(dcb.NewTags("course_id", courseID), "StudentEnrolled"),
        InitialState: 0,
        TransitionFn: func(state any, event dcb.Event) any {
            return state.(int) + 1
        },
    },
}
states, appendCond, _ := store.Project(ctx, projectors, nil)
if !states["courseExists"].(bool) { 
    store.Append(ctx, []dcb.InputEvent{...}, nil) 
}
if states["numSubscriptions"].(int) < 2 { 
    store.Append(ctx, []dcb.InputEvent{...}, &appendCond) 
}
```

## Transaction Isolation Levels

go-crablet uses configurable PostgreSQL transaction isolation levels:

- **Append (unconditional)**: Uses the default isolation level configured in `EventStoreConfig` (typically Read Committed)
- **Append (conditional)**: Uses the default isolation level configured in `EventStoreConfig` (typically Repeatable Read)

The isolation level can be configured when creating the EventStore via `EventStoreConfig.DefaultAppendIsolation`.

## Configuration

The EventStore can be configured with various settings:

```go
type EventStoreConfig struct {
    MaxBatchSize           int            `json:"max_batch_size"`           // Maximum events per append call
    LockTimeout            int            `json:"lock_timeout"`             // Lock timeout in milliseconds for advisory locks
    StreamBuffer           int            `json:"stream_buffer"`            // Channel buffer size for streaming operations
    DefaultAppendIsolation IsolationLevel `json:"default_append_isolation"` // Default isolation level for Append operations
    QueryTimeout           int            `json:"query_timeout"`            // Query timeout in milliseconds (defensive against hanging queries)
    AppendTimeout          int            `json:"append_timeout"`           // Append timeout in milliseconds (defensive against hanging appends)
}
```

### Default Values
- `MaxBatchSize`: 1000 events (limits events per append call)
- `LockTimeout`: 5000ms (5 seconds)
- `StreamBuffer`: 1000 events
- `DefaultAppendIsolation`: Read Committed
- `QueryTimeout`: 15000ms (15 seconds)
- `AppendTimeout`: 10000ms (10 seconds)

## Performance Comparison Across Isolation Levels

Benchmark results from web-app load testing (30-second tests, multiple VUs):

| Metric | Append (unconditional) | Append (conditional) |
|--------|------------------------|---------------------------|
| **Throughput** | 79.2 req/s | 61.7 req/s |
| **Avg Response Time** | 24.87ms | 12.82ms |
| **p95 Response Time** | 49.16ms | 21.86ms |
| **Success Rate** | 100% | 100% |
| **VUs** | 10 | 10 |
| **Use Case** | Simple appends | Conditional appends |

### Key Performance Insights

- **Conditional append is fastest**: Conditional appends with Repeatable Read isolation actually perform better than simple appends
- **Excellent reliability**: Both isolation levels achieve 100% success rate
- **Optimized implementation**: Cursor-based optimistic locking and SQL functions are highly efficient

### When to Use Each Method

- **Append (nil condition)**: Use for simple event appends where no conditions are needed
- **Append (with condition)**: Use for conditional appends requiring optimistic locking

## Table Validation

The library validates that the `events` table exists and has the correct structure:

- **Required columns**: `type`, `tags`, `data`, `transaction_id`, `position`, `occurred_at`
- **Data types**: Validates column types and nullable constraints
- **Error handling**: Returns `TableStructureError` with detailed information about validation failures

Example validation errors:
- `table events does not exist`
- `missing required column 'occurred_at'`
- `column 'tags' should be ARRAY type, got TEXT`

## Command Execution

go-crablet supports atomic command execution with handler-based event generation and command tracking. The `CommandExecutor` provides a clean abstraction for command-driven event sourcing:

### Command Execution Flow

1. **Execute command** using the `CommandExecutor`
2. **Handler performs projection** using the provided `EventStore`
3. **Handler generates events** based on business logic and projected state
4. **Store command** in the `commands` table with transaction ID (automatic)
5. **Append events** atomically within the same transaction

### Database Tables

The library uses two main tables:

- **`events` table**: Stores all events with transaction IDs for ordering
- **`commands` table**: Tracks executed commands with metadata and links to events via transaction ID

#### Events Table Structure

```sql
CREATE TABLE events (
    transaction_id BIGINT NOT NULL,           -- PostgreSQL xid8 transaction ID for ordering
    position       BIGINT NOT NULL,           -- Position within transaction (0-based)
    type           TEXT NOT NULL,             -- Event type (e.g., "UserCreated", "OrderPlaced")
    tags           TEXT[] NOT NULL,           -- Array of tags for querying (e.g., ["user_id:123", "order_id:456"])
    data           JSONB NOT NULL,            -- Event payload as JSON
    occurred_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- Event timestamp
    
    PRIMARY KEY (transaction_id, position),   -- Composite primary key for ordering
    UNIQUE (transaction_id, position)         -- Ensure no duplicate positions within transaction
);

-- Indexes for efficient querying
CREATE INDEX idx_events_type ON events(type);
CREATE INDEX idx_events_tags ON events USING GIN(tags);
CREATE INDEX idx_events_occurred_at ON events(occurred_at);
CREATE INDEX idx_events_transaction_id ON events(transaction_id);
```

**Key Features:**
- **Transaction-based ordering**: `transaction_id` ensures true event ordering without gaps
- **Position within transaction**: `position` allows multiple events per transaction
- **Tag-based querying**: `tags` array enables flexible, cross-entity queries
- **JSONB data**: Rich event payloads with PostgreSQL JSONB performance
- **Automatic timestamps**: `occurred_at` provides event timing information

#### Commands Table Structure

```sql
CREATE TABLE commands (
    transaction_id BIGINT NOT NULL,           -- Links to events via transaction ID
    type           TEXT NOT NULL,             -- Command type (e.g., "CreateUser", "TransferMoney")
    data           JSONB NOT NULL,            -- Command payload as JSON
    metadata       JSONB NOT NULL DEFAULT '{}', -- Additional context (correlation ID, source, etc.)
    occurred_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- Command execution timestamp
    
    PRIMARY KEY (transaction_id),             -- One command per transaction
    FOREIGN KEY (transaction_id) REFERENCES events(transaction_id) ON DELETE CASCADE
);

-- Indexes for efficient querying
CREATE INDEX idx_commands_type ON commands(type);
CREATE INDEX idx_commands_occurred_at ON commands(occurred_at);
CREATE INDEX idx_commands_metadata ON commands USING GIN(metadata);
```

**Key Features:**
- **Transaction linking**: `transaction_id` links commands to their generated events
- **Command metadata**: `metadata` stores correlation IDs, source information, etc.
- **Audit trail**: Complete command execution history
- **Cascade deletion**: Commands deleted when related events are deleted

#### Example Data

**Events Table:**
```sql
-- User creation events
INSERT INTO events VALUES 
(123456789, 0, 'UserCreated', ARRAY['user_id:123', 'email:alice@example.com'], 
 '{"user_id": "123", "email": "alice@example.com", "name": "Alice Smith"}', 
 '2025-07-12 15:30:00+00');

-- Course enrollment events  
INSERT INTO events VALUES 
(123456790, 0, 'StudentEnrolled', ARRAY['user_id:123', 'course_id:CS101'], 
 '{"user_id": "123", "course_id": "CS101", "enrolled_at": "2025-07-12 15:35:00"}', 
 '2025-07-12 15:35:00+00');
```

**Commands Table:**
```sql
-- Command that generated the user creation events
INSERT INTO commands VALUES 
(123456789, 'CreateUser', 
 '{"email": "alice@example.com", "name": "Alice Smith"}',
 '{"correlation_id": "corr_123", "source": "web_api", "user_agent": "Mozilla/5.0"}',
 '2025-07-12 15:30:00+00');
```

#### Querying Patterns

**Event Queries:**
```sql
-- Query events by type
SELECT * FROM events WHERE type = 'UserCreated';

-- Query events by tags
SELECT * FROM events WHERE tags @> ARRAY['user_id:123'];

-- Query events by time range
SELECT * FROM events WHERE occurred_at BETWEEN '2025-07-12 00:00:00' AND '2025-07-12 23:59:59';

-- Query events with multiple tag conditions
SELECT * FROM events WHERE tags @> ARRAY['user_id:123'] AND tags @> ARRAY['course_id:CS101'];
```

**Command Queries:**
```sql
-- Query commands by type
SELECT * FROM commands WHERE type = 'CreateUser';

-- Query commands by metadata
SELECT * FROM commands WHERE metadata->>'correlation_id' = 'corr_123';

-- Query commands with their generated events
SELECT c.*, e.* 
FROM commands c 
JOIN events e ON c.transaction_id = e.transaction_id 
WHERE c.type = 'CreateUser';
```

### Basic Usage

```go
// Create command executor
commandExecutor := dcb.NewCommandExecutor(eventStore)

// Create command
cmd := dcb.NewCommand("CreateUser", dcb.ToJSON(userData), map[string]interface{}{
    "correlation_id": "corr_789",
    "source": "web_api",
})

// Define command handler
type CreateUserHandler struct{}

func (h *CreateUserHandler) Handle(ctx context.Context, store dcb.EventStore, command dcb.Command) []dcb.InputEvent {
    // Extract command data
    var cmdData CreateUserCommand
    json.Unmarshal(command.GetData(), &cmdData)
    
    // Perform projection to check current state
    projectors := []dcb.StateProjector{
        {
            ID: "userExists",
            Query: dcb.NewQuery(dcb.NewTags("email", cmdData.Email), "UserCreated"),
            InitialState: false,
            TransitionFn: func(state any, event dcb.Event) any { return true },
        },
    }
    
    states, _, err := store.Project(ctx, projectors, nil)
    if err != nil {
        return nil
    }
    
    // Check business rules using projected state
    if states["userExists"].(bool) {
        return []dcb.InputEvent{
            dcb.NewInputEvent("UserCreationFailed", 
                dcb.NewTags("email", cmdData.Email, "reason", "user_exists"), 
                dcb.ToJSON(map[string]string{"error": "User already exists"})),
        }
    }
    
    // Generate success events
    return []dcb.InputEvent{
        dcb.NewInputEvent("UserCreated", 
            dcb.NewTags("email", cmdData.Email), 
            dcb.ToJSON(userCreatedData)),
    }
}

// Execute command
handler := &CreateUserHandler{}
err := commandExecutor.ExecuteCommand(ctx, cmd, handler, nil)
```

### Command Tracking

Every executed command is automatically stored in the `commands` table with:

- **Transaction ID**: Links the command to its generated events
- **Command type**: Identifies the command type
- **Command data**: Serialized command payload
- **Metadata**: Additional context (correlation ID, source, etc.)
- **Timestamp**: When the command was executed

This enables:
- **Audit trails**: Track which commands led to which events
- **Debugging**: Correlate commands with their outcomes
- **Monitoring**: Analyze command execution patterns
- **CQRS**: Separate command and query concerns

### Type Safety for Projections

Since handlers receive the `EventStore`, they can implement their own projection logic with full type safety:

#### Option 1: Direct Projection in Handler
```go
func (h *MyHandler) Handle(ctx context.Context, store dcb.EventStore, command dcb.Command) []dcb.InputEvent {
    // Define projectors for this specific command
    projectors := []dcb.StateProjector{
        {
            ID: "userState",
            Query: dcb.NewQuery(dcb.NewTags("user_id", userID), "UserCreated", "UserUpdated"),
            InitialState: &UserState{},
            TransitionFn: func(state any, event dcb.Event) any {
                // Type-safe state transitions
                userState := state.(*UserState)
                // ... update user state
                return userState
            },
        },
    }
    
    states, _, err := store.Project(ctx, projectors, nil)
    if err != nil {
        return nil
    }
    
    userState := states["userState"].(*UserState)
    // Use projected state for business logic
    return events
}
```

#### Option 2: Reusable Projector Functions
```go
// Define reusable projectors
func UserStateProjector(userID string) dcb.StateProjector {
    return dcb.StateProjector{
        ID: "userState",
        Query: dcb.NewQuery(dcb.NewTags("user_id", userID), "UserCreated", "UserUpdated"),
        InitialState: &UserState{},
        TransitionFn: func(state any, event dcb.Event) any {
            userState := state.(*UserState)
            // ... state transition logic
            return userState
        },
    }
}

// Use in handler
func (h *MyHandler) Handle(ctx context.Context, store dcb.EventStore, command dcb.Command) []dcb.InputEvent {
    projectors := []dcb.StateProjector{UserStateProjector(userID)}
    states, _, err := store.Project(ctx, projectors, nil)
    // ... use projected state
}
```

## Implementation Details

- **Database**: PostgreSQL with `events` table, `commands` table, and append functions
- **Event Storage**: Events stored with transaction IDs for true ordering and optimistic locking
- **Streaming**: Multiple approaches for different dataset sizes (cursor-based and channel-based)
- **Projections**: DCB decision model pattern with state projectors
- **Optimistic Locking**: Cursor-based append conditions for concurrent safety
- **Command Tracking**: Commands automatically stored in `commands` table with transaction ID linking
- **Command Execution**: Atomic command execution with handler-based event generation using `CommandExecutor`

See [examples](examples.md) for complete working examples including course subscriptions and money transfers, and [getting-started](getting-started.md) for setup instructions.