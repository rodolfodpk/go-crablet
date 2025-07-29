# EventStore Flow: Direct Event Operations

This document explains the simple flow for using go-crablet's EventStore directly without the CommandExecutor API. This covers the core event sourcing operations: appending events, querying events, and projecting state.

## Table of Contents

1. [Overview](#overview)
2. [Basic EventStore Flow](#basic-eventstore-flow)
3. [Event Append Flow](#event-append-flow)
4. [Event Query Flow](#event-query-flow)
5. [Event Projection Flow](#event-projection-flow)
6. [Database Persistence](#database-persistence)
7. [Examples](#examples)

## Overview

The EventStore provides the core event sourcing functionality without the command pattern. It allows you to:

- **Append events** directly to the event stream
- **Query events** using tag-based filtering
- **Project state** by processing events through state machines
- **Handle concurrency** using DCB (Dynamic Consistency Boundary) patterns

## Basic EventStore Flow

```mermaid
sequenceDiagram
    participant Client
    participant EventStore
    participant Database
    
    Note over Client, Database: Simple Event Append Flow
    Client->>EventStore: Append(events)
    EventStore->>Database: Begin Transaction
    EventStore->>Database: Insert events with transaction_id
    Database-->>EventStore: Success
    EventStore-->>Client: Events appended
    
    Note over Client, Database: Event Query Flow
    Client->>EventStore: Query(tags, after)
    EventStore->>Database: SELECT events WHERE tags @> array
    Database-->>EventStore: Events
    EventStore-->>Client: []Event
    
    Note over Client, Database: Event Projection Flow
    Client->>EventStore: Project(projectors, after)
    EventStore->>Database: Query events
    Database-->>EventStore: Events
    EventStore->>EventStore: Process through projectors
    EventStore-->>Client: Final states + cursor
```

## Event Append Flow

### Simple Append (No Conditions)

```go
// Create events
events := []dcb.InputEvent{
    dcb.NewEvent("CourseCreated").
        WithTag("course_id", "CS101").
        WithData(map[string]interface{}{
            "name": "Introduction to Computer Science",
            "capacity": 30,
        }).
        Build(),
    
    dcb.NewEvent("StudentEnrolled").
        WithTag("student_id", "student123").
        WithTag("course_id", "CS101").
        WithData(map[string]interface{}{
            "enrollment_date": time.Now(),
        }).
        Build(),
}

// Append events directly
err := store.Append(ctx, events)
```

**Flow:**
1. **Validate events** - Check event structure and constraints
2. **Begin transaction** - Start PostgreSQL transaction
3. **Generate transaction ID** - Use `pg_current_xact_id()`
4. **Insert events** - Batch insert with `append_events_batch()`
5. **Commit transaction** - All events become visible atomically

### Conditional Append (DCB Concurrency Control)

```go
// Define append condition
condition := dcb.AppendCondition{
    AfterCursor: &dcb.Cursor{
        TransactionID: 123,
        Position:      5,
    },
    FailIfEventsMatch: &dcb.Query{
        Tags: []dcb.Tag{
            {Key: "course_id", Value: "CS101"},
            {Key: "student_id", Value: "student123"},
        },
    },
}

// Append with condition
err := store.AppendIf(ctx, events, condition)
```

**Flow:**
1. **Validate events** - Check event structure and constraints
2. **Begin transaction** - Start PostgreSQL transaction
3. **Check conditions** - Verify no conflicting events exist
4. **Generate transaction ID** - Use `pg_current_xact_id()`
5. **Insert events** - Batch insert with `append_events_with_condition()`
6. **Commit transaction** - All events become visible atomically

### Advisory Lock Append (Experimental)

```go
// Events with lock tags
events := []dcb.InputEvent{
    dcb.NewEvent("AccountDebited").
        WithTag("account_id", "acc-001").
        WithTag("lock:account_id", "acc-001"). // Advisory lock tag
        WithData(map[string]interface{}{
            "amount": 100,
        }).
        Build(),
}

// Append with advisory locks
err := store.AppendWithLocks(ctx, events, nil)
```

**Flow:**
1. **Validate events** - Check event structure and constraints
2. **Begin transaction** - Start PostgreSQL transaction
3. **Acquire advisory locks** - Lock resources using `lock:` tags
4. **Check conditions** - If specified, verify no conflicts
5. **Generate transaction ID** - Use `pg_current_xact_id()`
6. **Insert events** - Batch insert with `append_events_with_advisory_locks()`
7. **Commit transaction** - Locks released, events become visible

## Event Query Flow

### Batch Query

```go
// Define query
query := dcb.Query{
    Tags: []dcb.Tag{
        {Key: "course_id", Value: "CS101"},
    },
}

// Execute batch query
events, err := store.Query(ctx, query, nil)
```

**Flow:**
1. **Build SQL query** - Generate optimized SQL with tag filtering
2. **Execute query** - Use GIN indexes for fast tag matching
3. **Fetch results** - Load all matching events into memory
4. **Return events** - Convert database rows to Event objects

### Streaming Query

```go
// Execute streaming query
eventChan, err := store.QueryStream(ctx, query, nil)
if err != nil {
    return err
}

// Process events as they arrive
for event := range eventChan {
    fmt.Printf("Received event: %s\n", event.GetType())
}
```

**Flow:**
1. **Build SQL query** - Generate optimized SQL with tag filtering
2. **Start goroutine** - Begin background query execution
3. **Stream results** - Send events through buffered channel
4. **Process events** - Handle events one at a time (memory efficient)
5. **Close channel** - When query completes or context cancels

## Event Projection Flow

### Batch Projection

```go
// Define state projector
projector := dcb.StateProjector{
    Name: "CourseEnrollment",
    Tags: []dcb.Tag{{Key: "course_id", Value: "CS101"}},
    InitialState: map[string]interface{}{
        "enrolled_students": []string{},
        "capacity": 30,
    },
    Apply: func(state map[string]interface{}, event dcb.Event) map[string]interface{} {
        switch event.GetType() {
        case "StudentEnrolled":
            students := state["enrolled_students"].([]string)
            studentID := event.GetData()["student_id"].(string)
            state["enrolled_students"] = append(students, studentID)
        }
        return state
    },
}

// Execute projection
finalState, cursor, err := store.Project(ctx, []dcb.StateProjector{projector}, nil)
```

**Flow:**
1. **Query events** - Fetch all events matching projector tags
2. **Initialize state** - Start with projector's initial state
3. **Process events** - Apply each event through projector function
4. **Return result** - Final state and cursor for next projection

### Streaming Projection

```go
// Execute streaming projection
stateChan, cursorChan, err := store.ProjectStream(ctx, []dcb.StateProjector{projector}, nil)
if err != nil {
    return err
}

// Process intermediate states
for state := range stateChan {
    fmt.Printf("Intermediate state: %+v\n", state)
}

// Get final cursor
cursor := <-cursorChan
```

**Flow:**
1. **Start streaming** - Begin background projection processing
2. **Query events** - Stream events matching projector tags
3. **Process incrementally** - Apply events and emit intermediate states
4. **Stream results** - Send states through channel as they're computed
5. **Return cursor** - Final cursor for next projection

## Database Persistence

### Events Table (Primary Data)

All events are stored in the `events` table:

```sql
-- Example: Course creation and enrollment events
SELECT * FROM events WHERE transaction_id = 456 ORDER BY position;
```

| type | tags | data | transaction_id | position | occurred_at |
|------|------|------|----------------|----------|-------------|
| CourseCreated | {"course_id:CS101"} | {"name":"Intro to CS","capacity":30} | 456 | 1 | 2024-01-15 10:30:00 |
| StudentEnrolled | {"student_id:student123","course_id:CS101"} | {"enrollment_date":"2024-01-15T10:30:00Z"} | 456 | 2 | 2024-01-15 10:30:00 |

**Key Points:**
- **Same transaction_id**: All events in a batch share the same transaction ID
- **Sequential positions**: Events are ordered by position within the transaction
- **Tag-based storage**: Tags stored as PostgreSQL TEXT[] arrays for efficient querying
- **JSON data**: Event payload stored as JSONB for flexibility

### No Commands Table

Unlike the CommandExecutor flow, direct EventStore operations do **not** use the `commands` table. Events are the only data persisted.

## Examples

### Simple Course Management

```go
func main() {
    ctx := context.Background()
    
    // Create EventStore
    store, err := dcb.NewEventStore(ctx, "postgres://user:pass@localhost/crablet")
    if err != nil {
        panic(err)
    }
    defer store.Close()
    
    // Create course
    courseEvents := []dcb.InputEvent{
        dcb.NewEvent("CourseCreated").
            WithTag("course_id", "CS101").
            WithData(map[string]interface{}{
                "name": "Introduction to Computer Science",
                "capacity": 30,
            }).
            Build(),
    }
    
    err = store.Append(ctx, courseEvents)
    if err != nil {
        panic(err)
    }
    
    // Enroll student
    enrollmentEvents := []dcb.InputEvent{
        dcb.NewEvent("StudentEnrolled").
            WithTag("student_id", "student123").
            WithTag("course_id", "CS101").
            WithData(map[string]interface{}{
                "enrollment_date": time.Now(),
            }).
            Build(),
    }
    
    err = store.Append(ctx, enrollmentEvents)
    if err != nil {
        panic(err)
    }
    
    // Query course events
    query := dcb.Query{
        Tags: []dcb.Tag{{Key: "course_id", Value: "CS101"}},
    }
    
    events, err := store.Query(ctx, query, nil)
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Found %d events for course CS101\n", len(events))
    
    // Project course state
    projector := dcb.StateProjector{
        Name: "CourseState",
        Tags: []dcb.Tag{{Key: "course_id", Value: "CS101"}},
        InitialState: map[string]interface{}{
            "name": "",
            "capacity": 0,
            "enrolled_students": []string{},
        },
        Apply: func(state map[string]interface{}, event dcb.Event) map[string]interface{} {
            switch event.GetType() {
            case "CourseCreated":
                state["name"] = event.GetData()["name"]
                state["capacity"] = event.GetData()["capacity"]
            case "StudentEnrolled":
                students := state["enrolled_students"].([]string)
                studentID := event.GetData()["student_id"].(string)
                state["enrolled_students"] = append(students, studentID)
            }
            return state
        },
    }
    
    finalState, _, err := store.Project(ctx, []dcb.StateProjector{projector}, nil)
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Course state: %+v\n", finalState["CourseState"])
}
```

### Conditional Append with Concurrency Control

```go
// Check if student is already enrolled before enrolling
condition := dcb.AppendCondition{
    FailIfEventsMatch: &dcb.Query{
        Tags: []dcb.Tag{
            {Key: "course_id", Value: "CS101"},
            {Key: "student_id", Value: "student123"},
        },
    },
}

enrollmentEvents := []dcb.InputEvent{
    dcb.NewEvent("StudentEnrolled").
        WithTag("student_id", "student123").
        WithTag("course_id", "CS101").
        WithData(map[string]interface{}{
            "enrollment_date": time.Now(),
        }).
        Build(),
}

err := store.AppendIf(ctx, enrollmentEvents, condition)
if err != nil {
    if dcb.IsConcurrencyError(err) {
        fmt.Println("Student already enrolled")
    } else {
        panic(err)
    }
}
```

This EventStore flow provides the core event sourcing functionality without the command pattern, allowing direct event operations with full control over concurrency and consistency. 