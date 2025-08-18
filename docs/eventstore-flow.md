# EventStore Flow: Direct Event Operations

This document explains the simple flow for using go-crablet's EventStore directly without the CommandExecutor API. This covers the core event sourcing operations: appending events, querying events, and projecting state.

**Note: This is an exploration project for learning and experimenting with DCB concepts, not a production-ready solution.**

## Table of Contents

1. [Overview](#overview)
2. [Basic EventStore Flow](#basic-eventstore-flow)
3. [Event Append Flow](#event-append-flow)
4. [Event Query Flow](#event-query-flow)
5. [Event Projection Flow](#event-projection-flow)
6. [Database Persistence](#database-persistence)
7. [Examples](#examples)

## Overview

The EventStore provides the core event sourcing functionality without the command pattern. It allows you to **explore**:

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
// BEST PRACTICE: Define event data as structs for type safety and performance
type CourseCreatedData struct {
    Name     string `json:"name"`
    Capacity int    `json:"capacity"`
}

type StudentEnrolledData struct {
    EnrollmentDate time.Time `json:"enrollment_date"`
}

// Create events with struct-based data (RECOMMENDED)
events := []dcb.InputEvent{
    dcb.NewEvent("CourseCreated").
        WithTag("course_id", "CS101").
        WithData(CourseCreatedData{
            Name:     "Introduction to Computer Science",
            Capacity: 30,
        }).
        Build(),
    
    dcb.NewEvent("StudentEnrolled").
        WithTag("student_id", "student123").
        WithTag("course_id", "CS101").
        WithData(StudentEnrolledData{
            EnrollmentDate: time.Now(),
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
// Define append condition using QueryBuilder
condition := dcb.NewAppendCondition(
    dcb.NewQueryBuilder().
        WithTag("course_id", "CS101").
        WithTag("student_id", "student123").
        WithType("StudentEnrolled").
        Build(),
)

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

### Conditional Append with DCB

```go
// BEST PRACTICE: Define event data as structs
type AccountDebitedData struct {
    Amount int `json:"amount"`
}

// Events with business logic validation
events := []dcb.InputEvent{
    dcb.NewEvent("AccountDebited").
        WithTag("account_id", "acc-001").
        WithData(AccountDebitedData{
            Amount: 100,
        }).
        Build(),
}

// Append with DCB concurrency control
err := store.AppendIf(ctx, events, condition)
```

**Flow:**
1. **Validate events** - Check event structure and constraints
2. **Begin transaction** - Start PostgreSQL transaction
3. **Check conditions** - If specified, verify no conflicts using DCB
4. **Generate transaction ID** - Use `pg_current_xact_id()`
5. **Insert events** - Batch insert with appropriate SQL function
6. **Commit transaction** - Events become visible atomically

## Event Query Flow

### Batch Query

```go
// Define query using QueryBuilder
query := dcb.NewQueryBuilder().
    WithTag("course_id", "CS101").
    WithType("CourseCreated").
    Build()

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
    fmt.Printf("Received event: %s\n", event.GetEventType())
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
// BEST PRACTICE: Use typed constants for event types and typed structs for state projection
const (
	EventTypeStudentEnrolled = "StudentEnrolled"
)

type CourseEnrollmentState struct {
	EnrolledStudents []string `json:"enrolled_students"`
	Capacity         int      `json:"capacity"`
}

// Define state projector using QueryBuilder with typed state
projector := dcb.StateProjector{
	ID: "CourseEnrollment",
	Query: dcb.NewQueryBuilder().
		WithTag("course_id", "CS101").
		Build(),
	InitialState: CourseEnrollmentState{
		EnrolledStudents: []string{},
		Capacity:         30,
	},
	TransitionFn: func(state any, event dcb.Event) any {
		currentState := state.(CourseEnrollmentState)
		
		switch event.GetEventType() {
		case EventTypeStudentEnrolled:
			var data StudentEnrolledData
			if err := json.Unmarshal(event.GetData(), &data); err == nil {
				// Note: In a real implementation, you'd extract student_id from tags
				// This is simplified for demonstration
				currentState.EnrolledStudents = append(currentState.EnrolledStudents, "student123")
			}
		}
		return currentState
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
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "time"
    
    "github.com/rodolfodpk/go-crablet/pkg/dcb"
)

// BEST PRACTICE: Define event data as structs for type safety and performance
type CourseCreatedData struct {
    Name     string `json:"name"`
    Capacity int    `json:"capacity"`
}

type StudentEnrolledData struct {
    EnrollmentDate time.Time `json:"enrollment_date"`
}

type CourseState struct {
    Name             string   `json:"name"`
    Capacity         int      `json:"capacity"`
    EnrolledStudents []string `json:"enrolled_students"`
}

func main() {
    ctx := context.Background()
    
    // Create EventStore
    store, err := dcb.NewEventStore(ctx, "postgres://user:pass@localhost/crablet")
    if err != nil {
        panic(err)
    }
    defer store.Close()
    
    // Create course with struct-based data (RECOMMENDED)
    courseEvents := []dcb.InputEvent{
        dcb.NewEvent("CourseCreated").
            WithTag("course_id", "CS101").
            WithData(CourseCreatedData{
                Name:     "Introduction to Computer Science",
                Capacity: 30,
            }).
            Build(),
    }
    
    err = store.Append(ctx, courseEvents)
    if err != nil {
        panic(err)
    }
    
    // Enroll student with struct-based data (RECOMMENDED)
    enrollmentEvents := []dcb.InputEvent{
        dcb.NewEvent("StudentEnrolled").
            WithTag("student_id", "student123").
            WithTag("course_id", "CS101").
            WithData(StudentEnrolledData{
                EnrollmentDate: time.Now(),
            }).
            Build(),
    }
    
    err = store.Append(ctx, enrollmentEvents)
    if err != nil {
        panic(err)
    }
    
    // Query course events using QueryBuilder
    query := dcb.NewQueryBuilder().
        WithTag("course_id", "CS101").
        Build()
    
    events, err := store.Query(ctx, query, nil)
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Found %d events for course CS101\n", len(events))
    
    // Project course state using QueryBuilder with struct-based state
    projector := dcb.StateProjector{
        ID: "CourseState",
        Query: dcb.NewQueryBuilder().
            WithTag("course_id", "CS101").
            Build(),
        InitialState: CourseState{
            Name:             "",
            Capacity:         0,
            EnrolledStudents: []string{},
        },
        TransitionFn: func(state any, event dcb.Event) any {
            currentState := state.(CourseState)
            
            switch event.GetEventType() {
            case EventTypeCourseScheduled:
                var data CourseScheduledData
                if err := json.Unmarshal(event.GetData(), &data); err == nil {
                    currentState.Name = data.Name
                    currentState.Capacity = data.Capacity
                }
            case EventTypeStudentEnrolled:
                var data StudentEnrolledData
                if err := json.Unmarshal(event.GetData(), &data); err == nil {
                    // Note: In a real implementation, you'd extract student_id from tags
                    // This is simplified for demonstration
                    currentState.EnrolledStudents = append(currentState.EnrolledStudents, "student123")
                }
            }
            return currentState
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
// Check if student is already enrolled before enrolling using QueryBuilder
condition := dcb.NewAppendCondition(
    dcb.NewQueryBuilder().
        WithTag("course_id", "CS101").
        WithTag("student_id", "student123").
        WithType("StudentEnrolled").
        Build(),
)

enrollmentEvents := []dcb.InputEvent{
    dcb.NewEvent("StudentEnrolled").
        WithTag("student_id", "student123").
        WithTag("course_id", "CS101").
        WithData(StudentEnrolledData{
            EnrollmentDate: time.Now(),
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

This EventStore flow provides the core event sourcing functionality without the command pattern, allowing you to **explore** direct event operations with full control over concurrency and consistency. This approach is useful for learning and experimenting with DCB patterns. 