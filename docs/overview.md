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
    // Query events with optional cursor (nil = from beginning, non-nil = after cursor)
    Query(ctx context.Context, query Query, cursor *Cursor) ([]Event, error)

    // Stream events for large datasets with backpressure
    QueryStream(ctx context.Context, query Query, cursor *Cursor) (<-chan Event, error)

    // Append events with optional optimistic locking condition
    Append(ctx context.Context, events []InputEvent, condition *AppendCondition) error

    // Project multiple states in single query (DCB pattern)
    Project(ctx context.Context, projectors []StateProjector, cursor *Cursor) (map[string]any, AppendCondition, error)

    // Stream projections for large datasets
    ProjectStream(ctx context.Context, projectors []StateProjector, cursor *Cursor) (<-chan map[string]any, <-chan AppendCondition, error)

    GetConfig() EventStoreConfig
    GetPool() *pgxpool.Pool
}
```

### CommandExecutor Interface (Optional API)

The `CommandExecutor` is an **optional** wrapper around `EventStore` that provides command-driven event generation. It's not required for basic event sourcing - you can use the `EventStore` directly:

```go
type CommandExecutor interface {
    // Execute command and generate events atomically
    ExecuteCommand(ctx context.Context, command Command, handler CommandHandler, condition *AppendCondition) error
}

type CommandHandler interface {
    // Process command and return events to append
    Handle(ctx context.Context, store EventStore, command Command) []InputEvent
}
```

### Usage Pattern

Users typically follow this pattern:

```go
// 1. Create EventStore (primary interface)
store, err := dcb.NewEventStore(ctx, pool)

// 2. Optionally create CommandExecutor from EventStore (not required)
commandExecutor := dcb.NewCommandExecutor(store)

// 3. Use either interface as needed
// Direct EventStore usage (core API):
err = store.Append(ctx, events, condition)

// Command-driven usage (optional convenience API):
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

## Transaction ID Ordering and Locking

go-crablet uses PostgreSQL's `xid8` transaction IDs for event ordering and optimistic locking:

- **True ordering**: No gaps or out-of-order events
- **Optimistic locking**: Uses transaction IDs for conflict detection (primary mechanism)
- **Cursor-based**: Combines `(transaction_id, position)` for precise positioning
- **Advisory locks**: Optional additional locking via `lock:` prefixed tags in event tags

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

## Transaction Isolation and Locking

### Primary: Optimistic Locking
go-crablet primarily uses **optimistic locking** via transaction IDs and append conditions:
- **Conflict detection**: Uses `AppendCondition` to check for existing events before appending
- **Concurrent safety**: Only one append can succeed when conditions match existing events
- **No blocking**: Failed appends return immediately with `ConcurrencyError`

### Optional: Advisory Locks
For additional concurrency control, you can use PostgreSQL advisory locks via event tags:
- **Tag-based locking**: Add tags with `lock:` prefix (e.g., `"lock:user-123"`, `"lock:account-456"`)
- **Automatic acquisition**: Database functions automatically acquire locks on these keys
- **Deadlock prevention**: Locks are sorted and acquired in consistent order
- **Transaction-scoped**: Locks are automatically released when transaction commits/rolls back

**Example with advisory locks:**
```go
// This event will acquire advisory lock on "user-123"
event := dcb.NewInputEvent("UserAction",
    dcb.NewTags("user_id", "123", "lock:user-123"),
    dcb.ToJSON(data))
```

**Note**: Advisory locks are currently available in the database functions but not actively used by the Go implementation.

### Isolation Levels
Configurable PostgreSQL isolation levels via `EventStoreConfig.DefaultAppendIsolation` (default: Read Committed).

## Configuration

The EventStore can be configured with various settings:

```go
type EventStoreConfig struct {
    MaxBatchSize           int            `json:"max_batch_size"`           // Maximum events per append call
    LockTimeout            int            `json:"lock_timeout"`             // Lock timeout in milliseconds for advisory locks (optional feature)
    StreamBuffer           int            `json:"stream_buffer"`            // Channel buffer size for streaming operations
    DefaultAppendIsolation IsolationLevel `json:"default_append_isolation"` // Default isolation level for Append operations
    QueryTimeout           int            `json:"query_timeout"`            // Query timeout in milliseconds (defensive against hanging queries)
    AppendTimeout          int            `json:"append_timeout"`           // Append timeout in milliseconds (defensive against hanging appends)
}
```

### Default Values
- `MaxBatchSize`: 1000 events (limits events per append call)
- `LockTimeout`: 5000ms (5 seconds) - **Optional feature, currently unused**
- `StreamBuffer`: 1000 events
- `DefaultAppendIsolation`: Read Committed
- `QueryTimeout`: 15000ms (15 seconds)
- `AppendTimeout`: 10000ms (10 seconds)

## Performance

See [benchmarks documentation](benchmarks.md) for detailed performance analysis and isolation level comparisons.

## Table Validation

The library validates that the `events` table exists and has the correct structure:

- **Required columns**: `type`, `tags`, `data`, `transaction_id`, `position`, `occurred_at`
- **Data types**: Validates column types and nullable constraints
- **Error handling**: Returns `TableStructureError` with detailed information about validation failures

Example validation errors:
- `table events does not exist`
- `missing required column 'occurred_at'`
- `column 'tags' should be ARRAY type, got TEXT`

## Command Execution (Optional Feature)

go-crablet supports atomic command execution with handler-based event generation and command tracking. The `CommandExecutor` provides an **optional** convenience layer for command-driven event sourcing. You can use the `EventStore` directly for basic event sourcing without the command pattern:

### Command Execution Flow

1. **Execute command** using the `CommandExecutor`
2. **Handler performs projection** using the provided `EventStore`
3. **Handler generates events** based on business logic and projected state
4. **Store command** in the `commands` table with transaction ID (automatic)
5. **Append events** atomically within the same transaction

### Database Tables

The library uses two main tables:

- **`events` table**: Stores all events with transaction IDs for ordering (required for all usage)
- **`commands` table**: Tracks executed commands with metadata and links to events via transaction ID (only used when CommandExecutor is used)

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
-- CREATE INDEX idx_events_type ON events(type);                    -- Not currently used
CREATE INDEX idx_events_tags ON events USING GIN(tags);            -- Used for tag-based queries
-- CREATE INDEX idx_events_occurred_at ON events(occurred_at);      -- Not currently used
CREATE INDEX idx_events_transaction_position_btree ON events(transaction_id, position); -- Used for ordering and cursors
```

**Key Features:**
- **Transaction-based ordering**: `transaction_id` ensures true event ordering without gaps
- **Position within transaction**: `position` allows multiple events per transaction
- **Tag-based querying**: `tags` array enables flexible, cross-entity queries
- **JSONB data**: Rich event payloads with PostgreSQL JSONB performance
- **Audit timestamps**: `occurred_at` provides event timing for audit purposes (not used for ordering/filtering)

#### Commands Table Structure

```sql
CREATE TABLE commands (
    transaction_id BIGINT NOT NULL,           -- Links to events via transaction ID
    type           TEXT NOT NULL,             -- Command type (e.g., "CreateUser", "TransferMoney")
    data           JSONB NOT NULL,            -- Command payload as JSON
    metadata       JSONB NOT NULL DEFAULT '{}', -- Additional context (correlation ID, source, etc.)
    occurred_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- Command execution timestamp

    PRIMARY KEY (transaction_id)              -- One command per transaction
);

-- Indexes for efficient querying
-- CREATE INDEX idx_commands_type ON commands(type);                -- Not currently used
-- CREATE INDEX idx_commands_occurred_at ON commands(occurred_at);  -- Not currently used
-- CREATE INDEX idx_commands_metadata ON commands USING GIN(metadata); -- Not currently used
```

**Key Features:**
- **Transaction linking**: `transaction_id` links commands to their generated events (logical relationship)
- **Command metadata**: `metadata` stores correlation IDs, source information, etc.
- **Audit trail**: Complete command execution history with timestamps
- **Independent tables**: Commands and events are inserted atomically in the same transaction

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

-- Query events with multiple tag conditions
SELECT * FROM events WHERE tags @> ARRAY['user_id:123'] AND tags @> ARRAY['course_id:CS101'];

-- Note: occurred_at is available for audit purposes but not used for filtering/ordering
-- Events are ordered by (transaction_id, position) for true event ordering
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

-- Note: occurred_at is available for audit purposes but not used for filtering/ordering
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

// Define command handler function
func handleCreateUser(ctx context.Context, store dcb.EventStore, command dcb.Command) []dcb.InputEvent {
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

// Execute command using function-based handler
err := commandExecutor.ExecuteCommand(ctx, cmd, dcb.CommandHandlerFunc(handleCreateUser), nil)
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

#### Option 1: Direct Projection in Handler Function
```go
func handleUserAction(ctx context.Context, store dcb.EventStore, command dcb.Command) []dcb.InputEvent {
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

// Usage with CommandExecutor
err := commandExecutor.ExecuteCommand(ctx, cmd, dcb.CommandHandlerFunc(handleUserAction), nil)
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

// Use in handler function
func handleUserAction(ctx context.Context, store dcb.EventStore, command dcb.Command) []dcb.InputEvent {
    projectors := []dcb.StateProjector{UserStateProjector(userID)}
    states, _, err := store.Project(ctx, projectors, nil)
    // ... use projected state
    return events
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
- **Lock Acquisition**: Advisory locks available via `lock:` prefixed tags in event tags (optional feature, currently unused in Go implementation)

See [examples](examples.md) for complete working examples including course subscriptions and money transfers, and [getting-started](getting-started.md) for setup instructions.
