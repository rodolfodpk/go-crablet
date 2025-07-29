# EventStore Flow

This document explains the simple EventStore flow for direct event operations without using the CommandExecutor.

## Overview

The EventStore provides a direct API for event sourcing operations:
- **Append**: Simple event appending without consistency checks
- **AppendIf**: Event appending with DCB concurrency control
- **Query**: Event retrieval with filtering
- **Project**: State projection from events

## Basic EventStore Usage

### 1. Simple Append (No Consistency Checks)

```go
package main

import (
    "context"
    "log"
    
    "github.com/rodolfodpk/go-crablet/pkg/dcb"
)

func main() {
    ctx := context.Background()
    
    // Create EventStore
    store, err := dcb.NewEventStore(ctx, "postgres://user:pass@localhost:5432/db")
    if err != nil {
        log.Fatal(err)
    }
    defer store.Close()
    
    // Create events
    events := []dcb.InputEvent{
        dcb.NewEvent("AccountCreated").
            WithTag("account_id", "acc-001").
            WithData(map[string]any{
                "name": "John Doe",
                "balance": 100.0,
            }).
            Build(),
    }
    
    // Simple append (no consistency checks)
    err = store.Append(ctx, events)
    if err != nil {
        log.Fatal(err)
    }
    
    log.Println("Events appended successfully")
}
```

### 2. DCB Concurrency Control Append

```go
// Events with business rule validation
events := []dcb.InputEvent{
    dcb.NewEvent("TransferCompleted").
        WithTag("account_id", "acc-001").
        WithData(map[string]any{
            "amount": 50.0,
            "to_account": "acc-002",
        }).
        Build(),
}

// Create condition to ensure account exists and has sufficient balance
query := dcb.NewQuery(
    dcb.NewTags("account_id", "acc-001"),
    "AccountCreated",
)
condition := dcb.NewAppendCondition(query)

// Append with DCB concurrency control
err = store.AppendIf(ctx, events, condition)
if err != nil {
    log.Fatal(err)
}
```

## EventStore Flow Steps

### Simple Append Flow:
1. **Validate events** - Check event types, tags, and data
2. **Start transaction** - Begin database transaction
3. **Insert events** - Batch insert events into database
4. **Commit transaction** - Commit changes

### DCB Concurrency Control Flow:
1. **Validate events** - Check event types, tags, and data
2. **Start transaction** - Begin database transaction
3. **Check conditions** - Verify business rules haven't changed
4. **Insert events** - Batch insert events if conditions pass
5. **Commit transaction** - Commit changes

## Key Differences

| Aspect | Simple Append | DCB Concurrency Control |
|--------|---------------|-------------------------|
| **Consistency** | None | Business rule validation |
| **Performance** | Fastest | Slightly slower due to condition checks |
| **Use Case** | Event logging, audit trails | Business operations with rules |
| **Concurrency** | No protection | Fail-fast on conflicts |

## Best Practices

1. **Use Simple Append** for:
   - Event logging and audit trails
   - Non-critical operations
   - When no business rules apply

2. **Use DCB Concurrency Control** for:
   - Business operations with rules
   - Operations that depend on existing state
   - When consistency is critical

3. **Event Design**:
   - Use descriptive event types
   - Include relevant tags for querying
   - Keep data JSON-serializable

This direct EventStore approach provides maximum flexibility and performance for event sourcing operations. 