# Low-Level Implementation Guide

This document provides detailed information about the internal implementation of go-crablet, including database schema, SQL functions, and low-level architectural decisions.

## Table of Contents

1. [Database Schema](#database-schema)
2. [SQL Functions](#sql-functions)
3. [Advisory Locks Implementation](#advisory-locks-implementation)
4. [Transaction Management](#transaction-management)
5. [Error Handling](#error-handling)
6. [Performance Considerations](#performance-considerations)

## Database Schema

### Events Table

The primary table that stores all events in the system:

```sql
CREATE TABLE events (
    id BIGSERIAL PRIMARY KEY,
    type TEXT NOT NULL,
    tags TEXT[] NOT NULL DEFAULT '{}',
    data JSONB NOT NULL,
    transaction_id BIGINT NOT NULL,
    position INTEGER NOT NULL,
    occurred_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX idx_events_type ON events(type);
CREATE INDEX idx_events_transaction_id ON events(transaction_id);
CREATE INDEX idx_events_occurred_at ON events(occurred_at);
CREATE INDEX idx_events_tags_gin ON events USING GIN(tags);
CREATE INDEX idx_events_data_gin ON events USING GIN(data);
CREATE UNIQUE INDEX idx_events_transaction_position ON events(transaction_id, position);
```

**Key Design Decisions:**
- **`transaction_id`**: Groups events that were created in the same transaction
- **`position`**: Order of events within a transaction (1, 2, 3, ...)
- **`tags`**: PostgreSQL TEXT[] array for efficient querying and indexing
- **`data`**: JSONB for flexible event payload storage
- **`occurred_at`**: Business timestamp (when the event logically occurred)
- **`created_at`**: System timestamp (when the event was stored)

### Commands Table

Audit trail for all commands executed:

```sql
CREATE TABLE commands (
    id BIGSERIAL PRIMARY KEY,
    transaction_id BIGINT NOT NULL,
    type TEXT NOT NULL,
    data JSONB NOT NULL,
    metadata JSONB,
    occurred_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_commands_type ON commands(type);
CREATE INDEX idx_commands_transaction_id ON commands(transaction_id);
CREATE INDEX idx_commands_occurred_at ON commands(occurred_at);
CREATE INDEX idx_commands_data_gin ON commands USING GIN(data);
```

**Purpose:**
- **Audit Trail**: Track all commands for debugging and compliance
- **Correlation**: Link commands to their generated events via `transaction_id`
- **Metadata**: Store additional information about command execution

### Sequences Table

Manages transaction ID generation:

```sql
CREATE TABLE sequences (
    name TEXT PRIMARY KEY,
    current_value BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

INSERT INTO sequences (name) VALUES ('transaction_id');
```

**Usage:**
- Ensures unique, monotonically increasing transaction IDs
- Used by all append functions to generate new transaction IDs

## SQL Functions

### Core Append Functions

#### 1. `append_events_batch()` - Unconditional Append

```sql
CREATE OR REPLACE FUNCTION append_events_batch(
    event_types TEXT[],
    event_tags TEXT[],
    event_data JSONB[]
) RETURNS VOID AS $$
DECLARE
    tx_id BIGINT;
    i INTEGER;
BEGIN
    -- Generate new transaction ID
    UPDATE sequences 
    SET current_value = current_value + 1, updated_at = NOW()
    WHERE name = 'transaction_id'
    RETURNING current_value INTO tx_id;
    
    -- Insert all events
    FOR i IN 1..array_length(event_types, 1) LOOP
        INSERT INTO events (type, tags, data, transaction_id, position)
        VALUES (
            event_types[i],
            event_tags[i]::TEXT[],
            event_data[i],
            tx_id,
            i
        );
    END LOOP;
END;
$$ LANGUAGE plpgsql;
```

#### 2. `append_events_with_condition()` - Conditional Append

```sql
CREATE OR REPLACE FUNCTION append_events_with_condition(
    event_types TEXT[],
    event_tags TEXT[],
    event_data JSONB[],
    condition JSONB
) RETURNS JSONB AS $$
DECLARE
    tx_id BIGINT;
    i INTEGER;
    condition_result BOOLEAN;
    result JSONB;
BEGIN
    -- Validate condition first
    IF condition IS NOT NULL THEN
        condition_result := validate_append_condition(condition);
        IF NOT condition_result THEN
            RETURN jsonb_build_object(
                'success', false,
                'message', 'Append condition violated'
            );
        END IF;
    END IF;
    
    -- Generate transaction ID and insert events
    UPDATE sequences 
    SET current_value = current_value + 1, updated_at = NOW()
    WHERE name = 'transaction_id'
    RETURNING current_value INTO tx_id;
    
    FOR i IN 1..array_length(event_types, 1) LOOP
        INSERT INTO events (type, tags, data, transaction_id, position)
        VALUES (
            event_types[i],
            event_tags[i]::TEXT[],
            event_data[i],
            tx_id,
            i
        );
    END LOOP;
    
    RETURN jsonb_build_object('success', true);
END;
$$ LANGUAGE plpgsql;
```

#### 3. `append_events_with_advisory_locks()` - Advisory Lock Append

```sql
CREATE OR REPLACE FUNCTION append_events_with_advisory_locks(
    event_types TEXT[],
    event_tags TEXT[],
    event_data JSONB[],
    lock_tags TEXT[],
    condition JSONB,
    lock_timeout_ms INTEGER DEFAULT 5000
) RETURNS JSONB AS $$
DECLARE
    tx_id BIGINT;
    i INTEGER;
    lock_key TEXT;
    condition_result BOOLEAN;
    result JSONB;
BEGIN
    -- Set lock timeout
    PERFORM set_config('lock_timeout', lock_timeout_ms::TEXT, false);
    
    -- Acquire advisory locks for all lock keys
    FOR i IN 1..array_length(lock_tags, 1) LOOP
        IF lock_tags[i] != '{}' THEN
            -- Parse lock keys from array literal string
            FOR lock_key IN SELECT unnest(lock_tags[i]::TEXT[]) LOOP
                PERFORM pg_advisory_xact_lock(hashtext(lock_key));
            END LOOP;
        END IF;
    END LOOP;
    
    -- Validate condition if provided
    IF condition IS NOT NULL THEN
        condition_result := validate_append_condition(condition);
        IF NOT condition_result THEN
            RAISE EXCEPTION 'DCB01: Append condition violated' USING ERRCODE = 'DCB01';
        END IF;
    END IF;
    
    -- Generate transaction ID and insert events
    UPDATE sequences 
    SET current_value = current_value + 1, updated_at = NOW()
    WHERE name = 'transaction_id'
    RETURNING current_value INTO tx_id;
    
    FOR i IN 1..array_length(event_types, 1) LOOP
        INSERT INTO events (type, tags, data, transaction_id, position)
        VALUES (
            event_types[i],
            event_tags[i]::TEXT[],
            event_data[i],
            tx_id,
            i
        );
    END LOOP;
    
    RETURN jsonb_build_object('success', true);
END;
$$ LANGUAGE plpgsql;
```

### Query Functions

#### `query_events()` - Event Querying

```sql
CREATE OR REPLACE FUNCTION query_events(
    query_type TEXT DEFAULT NULL,
    query_tags TEXT[] DEFAULT NULL,
    query_data JSONB DEFAULT NULL,
    cursor_transaction_id BIGINT DEFAULT NULL,
    cursor_position INTEGER DEFAULT NULL,
    limit_count INTEGER DEFAULT 100
) RETURNS TABLE(
    id BIGINT,
    type TEXT,
    tags TEXT[],
    data JSONB,
    transaction_id BIGINT,
    position INTEGER,
    occurred_at TIMESTAMP WITH TIME ZONE
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        e.id,
        e.type,
        e.tags,
        e.data,
        e.transaction_id,
        e.position,
        e.occurred_at
    FROM events e
    WHERE (query_type IS NULL OR e.type = query_type)
        AND (query_tags IS NULL OR e.tags @> query_tags)
        AND (query_data IS NULL OR e.data @> query_data)
        AND (
            cursor_transaction_id IS NULL 
            OR e.transaction_id > cursor_transaction_id
            OR (e.transaction_id = cursor_transaction_id AND e.position > cursor_position)
        )
    ORDER BY e.transaction_id ASC, e.position ASC
    LIMIT limit_count;
END;
$$ LANGUAGE plpgsql;
```

### Projection Functions

#### `project_state()` - State Projection

```sql
CREATE OR REPLACE FUNCTION project_state(
    projector_configs JSONB,
    cursor_transaction_id BIGINT DEFAULT NULL,
    cursor_position INTEGER DEFAULT NULL
) RETURNS JSONB AS $$
DECLARE
    projector_config JSONB;
    projector_type TEXT;
    projector_tags TEXT[];
    projector_data JSONB;
    projector_result JSONB;
    final_result JSONB := '{}'::JSONB;
BEGIN
    -- Process each projector configuration
    FOR projector_config IN SELECT * FROM jsonb_array_elements(projector_configs) LOOP
        projector_type := projector_config->>'type';
        projector_tags := ARRAY(SELECT jsonb_array_elements_text(projector_config->'tags'));
        projector_data := projector_config->'data';
        
        -- Apply projector logic based on type
        CASE projector_type
            WHEN 'count' THEN
                projector_result := project_count(projector_tags, projector_data, cursor_transaction_id, cursor_position);
            WHEN 'sum' THEN
                projector_result := project_sum(projector_tags, projector_data, cursor_transaction_id, cursor_position);
            WHEN 'custom' THEN
                projector_result := project_custom(projector_tags, projector_data, cursor_transaction_id, cursor_position);
            ELSE
                RAISE EXCEPTION 'Unknown projector type: %', projector_type;
        END CASE;
        
        -- Merge result into final result
        final_result := final_result || projector_result;
    END LOOP;
    
    RETURN final_result;
END;
$$ LANGUAGE plpgsql;
```

## Advisory Locks Implementation

### Lock Key Generation

Lock keys are generated from event tags with the `lock:` prefix:

```go
// In Go code
for _, tag := range event.GetTags() {
    if strings.HasPrefix(tag.GetKey(), "lock:") {
        lockKey := strings.TrimPrefix(tag.GetKey(), "lock:")
        lockKeys = append(lockKeys, lockKey)
    }
}
```

### Lock Acquisition Strategy

```sql
-- In SQL function
FOR lock_key IN SELECT unnest(lock_tags[i]::TEXT[]) LOOP
    PERFORM pg_advisory_xact_lock(hashtext(lock_key));
END LOOP;
```

**Key Characteristics:**
- **Transaction-scoped**: Locks are automatically released when transaction commits/rolls back
- **Hash-based**: Uses `hashtext()` for consistent lock key generation
- **Ordered**: Locks are acquired in a consistent order to prevent deadlocks
- **Timeout-protected**: Uses `lock_timeout` setting to prevent indefinite waiting

### Lock Key Examples

```go
// Example lock keys
"course:CS101"           // Lock specific course
"student:student123"     // Lock specific student
"enrollment:CS101"       // Lock enrollment operations for course
"account:account456"     // Lock specific account
```

## Transaction Management

### Isolation Levels

The system supports three isolation levels:

```go
type IsolationLevel int

const (
    IsolationLevelReadCommitted IsolationLevel = iota
    IsolationLevelRepeatableRead
    IsolationLevelSerializable
)
```

**Default**: `ReadCommitted` for most operations

### Transaction Flow

1. **Begin Transaction**: Start with specified isolation level
2. **Acquire Locks**: If advisory locks are needed
3. **Validate Conditions**: Check append conditions
4. **Generate Transaction ID**: Update sequences table
5. **Insert Events**: Batch insert all events
6. **Insert Command**: Store command for audit trail
7. **Commit**: All changes become visible

### Timeout Management

```go
func (es *eventStore) withTimeout(ctx context.Context, defaultTimeoutMs int) (context.Context, context.CancelFunc) {
    if deadline, ok := ctx.Deadline(); ok {
        // Use caller's timeout
        return context.WithDeadline(context.Background(), deadline)
    }
    // Use default timeout
    return context.WithTimeout(context.Background(), time.Duration(defaultTimeoutMs)*time.Millisecond)
}
```

## Error Handling

### Custom Error Codes

```sql
-- Custom error codes for specific scenarios
DCB01: Append condition violated
DCB02: Lock acquisition timeout
DCB03: Invalid event data
```

### Error Types in Go

```go
type EventStoreError struct {
    Op  string
    Err error
}

type ValidationError struct {
    EventStoreError
    Field string
    Value string
}

type ConcurrencyError struct {
    EventStoreError
}

type ResourceError struct {
    EventStoreError
    Resource string
}
```

### Error Recovery

- **Validation Errors**: Fail fast, no database changes
- **Concurrency Errors**: Retry with exponential backoff
- **Resource Errors**: Check database connectivity and configuration
- **Lock Timeouts**: Increase timeout or reduce concurrency

## Performance Considerations

### Indexing Strategy

```sql
-- Primary query patterns
CREATE INDEX idx_events_type ON events(type);
CREATE INDEX idx_events_transaction_id ON events(transaction_id);
CREATE INDEX idx_events_occurred_at ON events(occurred_at);

-- Tag-based queries
CREATE INDEX idx_events_tags_gin ON events USING GIN(tags);

-- JSON data queries
CREATE INDEX idx_events_data_gin ON events USING GIN(data);

-- Cursor-based pagination
CREATE UNIQUE INDEX idx_events_transaction_position ON events(transaction_id, position);
```

### Batch Operations

- **Batch Size Limit**: Configurable via `MaxBatchSize` (default: 1000)
- **Array Parameters**: Use PostgreSQL arrays for efficient batch inserts
- **Transaction Scope**: All events in a batch share the same transaction ID

### Lock Performance

- **Hash-based Keys**: Fast lock key generation and comparison
- **Transaction-scoped**: No manual lock cleanup required
- **Timeout Protection**: Prevents indefinite waiting
- **Ordered Acquisition**: Prevents deadlocks

### Query Optimization

- **Cursor-based Pagination**: Efficient for large datasets
- **Tag-based Filtering**: Uses GIN indexes for fast array operations
- **JSONB Queries**: Leverages PostgreSQL's JSONB indexing
- **Limit Clauses**: Prevents memory exhaustion

## Monitoring and Debugging

### Key Metrics

```sql
-- Event throughput
SELECT 
    DATE_TRUNC('hour', occurred_at) as hour,
    COUNT(*) as event_count,
    COUNT(DISTINCT transaction_id) as transaction_count
FROM events 
WHERE occurred_at >= NOW() - INTERVAL '24 hours'
GROUP BY hour
ORDER BY hour;

-- Lock contention
SELECT 
    locktype,
    database,
    relation,
    page,
    tuple,
    virtualxid,
    transactionid,
    classid,
    objid,
    objsubid,
    virtualtransaction,
    pid,
    mode,
    granted
FROM pg_locks 
WHERE locktype = 'advisory';

-- Transaction distribution
SELECT 
    COUNT(*) as event_count,
    COUNT(DISTINCT transaction_id) as transaction_count,
    AVG(events_per_transaction) as avg_events_per_tx
FROM (
    SELECT transaction_id, COUNT(*) as events_per_transaction
    FROM events 
    GROUP BY transaction_id
) tx_stats;
```

### Debug Queries

```sql
-- View recent events with full details
SELECT 
    e.*,
    c.type as command_type,
    c.data as command_data
FROM events e
LEFT JOIN commands c ON e.transaction_id = c.transaction_id
WHERE e.occurred_at >= NOW() - INTERVAL '1 hour'
ORDER BY e.transaction_id DESC, e.position DESC
LIMIT 100;

-- Check for duplicate transaction IDs (should never happen)
SELECT transaction_id, COUNT(*) as count
FROM events 
GROUP BY transaction_id 
HAVING COUNT(*) > 1;

-- Find events with specific tags
SELECT * FROM events 
WHERE tags @> ARRAY['course_id:CS101', 'student_id:student123']
ORDER BY occurred_at DESC;
```

This low-level documentation provides the foundation for understanding how go-crablet works internally, enabling developers to optimize, debug, and extend the system effectively. 