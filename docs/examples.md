# Examples

This document provides comprehensive examples of using go-crablet for **exploring** event sourcing concepts with DCB approaches.

**Note: This is an exploration project for learning and experimenting with DCB concepts, not a production-ready solution.**

## Quick Start Examples

### 1. Simple Event Store Usage

```go
package main

import (
    "context"
    "log"
    "time"
    
    "github.com/rodolfodpk/go-crablet/pkg/dcb"
)

// BEST PRACTICE: Define event data as structs for type safety and performance
type CourseOfferedData struct {
    Title    string `json:"title"`
    Credits  int    `json:"credits"`
    Capacity int    `json:"capacity"`
}

func main() {
    ctx := context.Background()
    
    // Create EventStore
    store, err := dcb.NewEventStore(ctx, "postgres://user:pass@localhost:5432/db")
    if err != nil {
        log.Fatal(err)
    }
    defer store.Close()
    
    // Create events with struct-based data (RECOMMENDED)
    events := []dcb.InputEvent{
        dcb.NewEvent("CourseOffered").
            WithTag("course_id", "CS101").
            WithTag("department", "Computer Science").
            WithData(CourseOfferedData{
                Title:    "Introduction to Computer Science",
                Credits:  3,
                Capacity: 30,
            }).
            Build(),
    }
    
    // Append events
    err = store.Append(ctx, events)
    if err != nil {
        log.Fatal(err)
    }
    
    log.Println("Course offered successfully")
}
```

### 2. DCB Concurrency Control

```go
// BEST PRACTICE: Use structs for event data
type EnrollmentCompletedData struct {
    StudentID string `json:"student_id"`
    CourseID  string `json:"course_id"`
    Grade     string `json:"grade"`
}

// Create events with business rule validation
events := []dcb.InputEvent{
    dcb.NewEvent("EnrollmentCompleted").
        WithTag("student_id", "student123").
        WithTag("course_id", "CS101").
        WithData(EnrollmentCompletedData{
            StudentID: "student123",
            CourseID:  "CS101",
            Grade:     "A",
        }).
        Build(),
}

// Create condition to ensure student is registered using QueryBuilder
query := dcb.NewQueryBuilder().
    WithTag("student_id", "student123").
    WithType("StudentRegistered").
    Build()
condition := dcb.NewAppendCondition(query)

// Append with DCB concurrency control
err = store.AppendIf(ctx, events, condition)
if err != nil {
    if dcb.IsConcurrencyError(err) {
        log.Println("Enrollment failed: student not registered")
    } else {
        log.Fatal(err)
    }
}
```

## Command Execution Examples

### 1. Basic Command Execution

```go
// Define command type
type EnrollStudentCommand struct {
    StudentID string `json:"student_id"`
    CourseID  string `json:"course_id"`
}

// Define command handler
func handleEnrollStudent(ctx context.Context, store dcb.EventStore, cmd dcb.Command) ([]dcb.InputEvent, error) {
    var data EnrollStudentCommand
    if err := json.Unmarshal(cmd.GetData(), &data); err != nil {
        return nil, fmt.Errorf("failed to unmarshal command: %w", err)
    }
    
    // Business logic validation
    if data.StudentID == "" {
        return nil, errors.New("student_id required")
    }
    if data.CourseID == "" {
        return nil, errors.New("course_id required")
    }
    
    // Create enrollment event
    event := dcb.NewEvent("EnrollmentCompleted").
        WithTag("student_id", data.StudentID).
        WithTag("course_id", data.CourseID).
        WithData(map[string]any{
            "enrolled_at": time.Now(),
        }).
        Build()
    
    return []dcb.InputEvent{event}, nil
}

// Execute command
command := dcb.NewCommand("EnrollStudent", dcb.ToJSON(EnrollStudentCommand{
    StudentID: "student123",
    CourseID:  "CS101",
}), nil)

commandExecutor := dcb.NewCommandExecutor(store)
events, err := commandExecutor.ExecuteCommand(ctx, command, handleEnrollStudent, nil)
if err != nil {
    log.Fatal(err)
}
```

### 2. Command with Concurrency Control

```go
// Create condition to prevent duplicate enrollment using QueryBuilder
enrollmentCondition := dcb.NewAppendCondition(
    dcb.NewQueryBuilder().
        WithTag("student_id", "student123").
        WithTag("course_id", "CS101").
        WithType("EnrollmentCompleted").
        Build(),
)

// Execute command with condition
events, err := commandExecutor.ExecuteCommand(ctx, command, handleEnrollStudent, &enrollmentCondition)
if err != nil {
    if dcb.IsConcurrencyError(err) {
        log.Println("Student already enrolled")
    } else {
        log.Fatal(err)
    }
}
```

## Query and Projection Examples

### 1. Simple Query

```go
// Query events by tags using QueryBuilder
query := dcb.NewQueryBuilder().
    WithTag("course_id", "CS101").
    WithType("EnrollmentCompleted").
    Build()

events, err := store.Query(ctx, query, nil)
if err != nil {
    log.Fatal(err)
}

log.Printf("Found %d enrollments for course CS101", len(events))
```

### 2. Streaming Query

```go
// Stream events as they arrive
eventChan, err := store.QueryStream(ctx, query, nil)
if err != nil {
    log.Fatal(err)
}

for event := range eventChan {
    log.Printf("Received event: %s", event.GetType())
}
```

### 3. Time-Based Query

```go
// Query events from the last hour using QueryBuilder
recentQuery := dcb.NewQueryBuilder().
    WithTag("course_id", "CS101").
    WithType("EnrollmentCompleted").
    SinceDuration(1 * time.Hour).
    Build()

recentEvents, err := store.Query(ctx, recentQuery, nil)
if err != nil {
    log.Fatal(err)
}

log.Printf("Found %d recent enrollments for course CS101", len(recentEvents))
```

### 4. State Projection

```go
// BEST PRACTICE: Use typed structs for state projection
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
		case EventTypeEnrollmentCompleted:
			var data EnrollmentCompletedData
			if err := json.Unmarshal(event.GetData(), &data); err == nil {
				studentID := data.StudentID
				currentState.EnrolledStudents = append(currentState.EnrolledStudents, studentID)
			}
		}
		return currentState
	},
}

// Execute projection
finalState, cursor, err := store.Project(ctx, []dcb.StateProjector{projector}, nil)
if err != nil {
    log.Fatal(err)
}

courseState := finalState["CourseEnrollment"].(map[string]any)
log.Printf("Course has %d enrolled students", len(courseState["enrolled_students"].([]string)))
```

## Advanced Examples

### 1. Batch Operations

```go
// Create multiple events in a batch
events := []dcb.InputEvent{
    dcb.NewEvent("CourseOffered").
        WithTag("course_id", "CS101").
        WithData(map[string]any{
            "title": "Introduction to Computer Science",
            "capacity": 30,
        }).
        Build(),
    
    dcb.NewEvent("EnrollmentCompleted").
        WithTag("student_id", "student123").
        WithTag("course_id", "CS101").
        WithData(map[string]any{
            "enrolled_at": time.Now(),
        }).
        Build(),
}

// Append batch
err = store.Append(ctx, events)
```

### 2. Complex Conditions

```go
// Create complex condition with multiple queries using QueryBuilder
// This creates an OR condition: (course exists) OR (student is registered)
complexQuery := dcb.NewQueryBuilder().
    WithTag("course_id", "CS101").
    WithType("CourseOffered").
    AddItem().
    WithTag("student_id", "student123").
    WithType("StudentRegistered").
    Build()

condition := dcb.NewAppendCondition(complexQuery)
```

### 3. Complex Queries with Time Filtering

```go
// Query events with multiple conditions and time filtering
complexQuery := dcb.NewQueryBuilder().
    WithTag("student_id", "123").
    WithTypes("EnrollmentCompleted", "CourseOffered").
    SinceDuration(24 * time.Hour).
    Build()

events, err := store.Query(ctx, complexQuery, nil)
if err != nil {
    log.Fatal(err)
}

log.Printf("Found %d course events for student 123 in the last 24 hours", len(events))
```

### 4. Error Handling

```go
// Handle different types of errors
err = store.AppendIf(ctx, events, condition)
if err != nil {
    switch {
    case dcb.IsValidationError(err):
        log.Println("Validation error:", err)
    case dcb.IsConcurrencyError(err):
        log.Println("Concurrency error:", err)
    case dcb.IsResourceError(err):
        log.Println("Resource error:", err)
    default:
        log.Fatal("Unexpected error:", err)
    }
}
```

## Configuration Examples

### 1. Custom EventStore Configuration

```go
config := dcb.EventStoreConfig{
    MaxBatchSize:           500,
    StreamBuffer:           500,
    DefaultAppendIsolation: dcb.IsolationLevelRepeatableRead,
    DefaultReadIsolation:   dcb.IsolationLevelReadCommitted,
    QueryTimeout:           10000, // 10 seconds
    AppendTimeout:          5000,  // 5 seconds
}

store, err := dcb.NewEventStoreWithConfig(ctx, pool, config)
```

### 2. Connection Pool Configuration

```go
pool, err := pgxpool.New(ctx, "postgres://user:pass@localhost:5432/db")
if err != nil {
    log.Fatal(err)
}
defer pool.Close()

// Configure pool
pool.Config().MaxConns = 20
pool.Config().MinConns = 5
```

## Best Practices

### 1. Event Design
- Use descriptive event types
- Include relevant tags for querying
- Keep data JSON-serializable
- Avoid large event payloads

### 2. Concurrency Control
- Use DCB conditions for business rules
- Design idempotent operations
- Handle concurrency errors gracefully

### 3. Performance
- Batch events when possible
- Use streaming for large datasets
- Optimize tag-based queries
- Monitor transaction sizes

### 4. Error Handling
- Check for specific error types
- Implement retry logic for transient failures
- Log errors with context
- Handle validation errors appropriately

## Complete Application Example

### Course Management System

```go
package main

import (
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "log"
    "time"
    
    "github.com/rodolfodpk/go-crablet/pkg/dcb"
)

// Command types
type CreateCourseCommand struct {
    CourseID string `json:"course_id"`
    Name     string `json:"name"`
    Capacity int    `json:"capacity"`
}

type EnrollStudentCommand struct {
    StudentID string `