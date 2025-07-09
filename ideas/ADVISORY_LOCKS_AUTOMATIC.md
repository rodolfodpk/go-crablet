# Automatic Advisory Locks Based on Tag Prefixes

## Overview
Add automatic advisory lock support to the EventStore by detecting "lock:" prefixed tags in events and applying advisory locks **in addition to** existing optimistic locking logic.

## Key Principles
- **Automatic detection**: No explicit flags or API changes needed
- **In addition to**: Advisory locks complement existing optimistic checks, don't replace them
- **Zero Go code changes**: All logic handled in PostgreSQL functions
- **Backward compatible**: Existing code works unchanged

## Implementation Options

### Option 1: Modify Existing PostgreSQL Function (Recommended)
- Modify `append_events_with_condition` to detect "lock:" prefixed tags
- Add advisory lock acquisition/release logic
- Keep Go code completely unchanged
- Single function to maintain

### Option 2: Create New PostgreSQL Function
- Create `append_events_with_advisory_locks` function
- Modify Go code to call new function when "lock:" tags detected
- Maintain two similar functions

## Technical Details

### Tag Detection
- Scan all event tags for keys starting with "lock:"
- Extract lock tags in format: `"lock:key:value"`
- Sort lock tags to prevent deadlocks

### Lock Strategy
- Acquire advisory locks on all "lock:" prefixed tags from events
- Support multiple aggregates per event (e.g., order, customer, warehouse)
- Use transaction-level advisory locks (automatic release on commit/rollback)

### Isolation Levels
- Respect client's chosen isolation level
- Advisory locks handle concurrency, isolation levels handle consistency
- No forced isolation level changes

## Example Usage

```go
// Automatic advisory locking - no API changes needed
event := NewInputEvent("OrderShipped", NewTags(
    "lock:order", "123",           // Locks order aggregate
    "lock:customer", "456",        // Locks customer aggregate
    "lock:warehouse", "789",       // Locks warehouse aggregate
    "status", "shipped",           // Non-lock tag
), data)

// Existing API call - automatically uses advisory locks
err := eventStore.AppendIf(ctx, []InputEvent{event}, condition)
```

## Benefits
1. **Intuitive**: "lock:" prefix naturally implies locking
2. **Self-documenting**: Tag names show intent
3. **Flexible**: Support multiple aggregates per event
4. **Backward compatible**: Existing code unchanged
5. **Zero coupling**: No pgx knowledge required by clients

## PostgreSQL Function Changes
```sql
-- In append_events_with_condition:
-- 1. Parse input tags to find "lock:" prefixed ones
-- 2. Acquire advisory locks on those tags
-- 3. Perform existing condition checking
-- 4. Append events
-- 5. Release locks (automatic with transaction)
```

## Future Considerations
- Performance impact of advisory lock acquisition
- Deadlock prevention strategies
- Monitoring and observability
- Integration with existing monitoring tools 