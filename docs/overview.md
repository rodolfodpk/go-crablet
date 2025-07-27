# Overview: Dynamic Consistency Boundary (DCB) in go-crablet

go-crablet is a Go library for event sourcing, exploring concepts inspired by the [Dynamic Consistency Boundary (DCB)](https://dcb.events/) pattern.

## Key Concepts

- **Batch Projection**: Project multiple states in one database query
- **Combined Append Condition**: Use OR-combined queries for DCB concurrency control
- **Tag-based Queries**: Flexible, cross-entity queries using tags
- **Streaming**: Process events efficiently for large datasets
- **Transaction-based Ordering**: Uses PostgreSQL transaction IDs for true event ordering
- **Atomic Command Execution**: Execute commands with handler-based event generation
- **Fluent API**: Intuitive interfaces for events, queries, and projections with 50% less boilerplate

## Core Interfaces

### Primary: EventStore Interface

The EventStore is the main interface for event sourcing operations:

```go
type EventStore interface {
    // Query operations
    Query(ctx context.Context, query Query, cursor *Cursor) ([]Event, error)
    QueryStream(ctx context.Context, query Query, cursor *Cursor) (<-chan Event, error)
    
    // Append operations
    Append(ctx context.Context, events []InputEvent, condition *AppendCondition) error
    AppendIf(ctx context.Context, events []InputEvent, condition AppendCondition) error
    
    // Projection operations
    Project(ctx context.Context, projectors []StateProjector, cursor *Cursor) (map[string]any, AppendCondition, error)
    ProjectStream(ctx context.Context, projectors []StateProjector, cursor *Cursor) (<-chan map[string]any, <-chan AppendCondition, error)
    
    // Configuration
    GetConfig() EventStoreConfig
    GetPool() *pgxpool.Pool
}
```

### Optional: CommandExecutor Interface

The CommandExecutor provides command-driven architecture support:

```go
type CommandExecutor interface {
    ExecuteCommand(ctx context.Context, command Command, handler CommandHandler, condition *AppendCondition) ([]InputEvent, error)
}

type CommandHandler interface {
    Handle(ctx context.Context, store EventStore, command Command) ([]InputEvent, error)
}
```

## Usage Patterns

### Primary: EventStore Pattern (Direct Event Sourcing)

The EventStore is the primary interface for event sourcing. Use this pattern when you want direct control over event creation and business logic:

```go
// 1. Create EventStore
store, err := dcb.NewEventStore(ctx, pool)

// 2. Use fluent API for events and queries
event := dcb.NewEvent("CourseDefined").
    WithTag("course_id", "CS101").
    WithData(courseData).
    Build()

query := dcb.NewQueryBuilder().
    WithTag("course_id", "CS101").
    WithType("CourseDefined").
    Build()

condition := dcb.FailIfExists("course_id", "CS101")

// 3. Direct event operations
err = store.AppendIf(ctx, []dcb.InputEvent{event}, condition)  // Conditional append
err = store.Append(ctx, events, nil)  // Unconditional append
events, err := store.Query(ctx, query, nil)  // Query events
states, err := store.Project(ctx, projectors, nil)  // Project state
```

### Optional: CommandExecutor Pattern (Command-Query Separation)

The CommandExecutor is an optional pattern for Command-Query Separation (CQS). Use this when you want to separate command handling from event creation:

```go
// 1. Create EventStore and CommandExecutor
store, err := dcb.NewEventStore(ctx, pool)
commandExecutor := dcb.NewCommandExecutor(store)

// 2. Define command types
type DefineCourseCommand struct {
    CourseID   string `json:"course_id"`
    Name       string `json:"name"`
    Capacity   int    `json:"capacity"`
}

type EnrollStudentCommand struct {
    StudentID string `json:"student_id"`
    CourseID  string `json:"course_id"`
}

// 3. Define command handlers
func handleDefineCourse(ctx context.Context, store dcb.EventStore, cmd dcb.Command) ([]dcb.InputEvent, error) {
    var data DefineCourseCommand
    if err := json.Unmarshal(cmd.GetData(), &data); err != nil {
        return nil, fmt.Errorf("failed to unmarshal command: %w", err)
    }
    
    // Business logic validation
    if data.Name == "" {
        return nil, errors.New("course name required")
    }
    if data.Capacity <= 0 {
        return nil, errors.New("capacity must be positive")
    }
    
    // Create course definition event
    event := dcb.NewEvent("CourseDefined").
        WithTag("course_id", data.CourseID).
        WithData(data).
        Build()
    
    return []dcb.InputEvent{event}, nil
}

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
    
    // Project course and enrollment states
    projectors := []dcb.StateProjector{
        dcb.ProjectBoolean("course_exists", "CourseDefined", "course_id", data.CourseID),
        dcb.ProjectCounter("enrollment_count", "StudentEnrolled", "course_id", data.CourseID),
        dcb.ProjectBoolean("already_enrolled", "StudentEnrolled", "student_id", data.StudentID),
    }
    
    states, _, err := store.Project(ctx, projectors, nil)
    if err != nil {
        return nil, fmt.Errorf("failed to project course state: %w", err)
    }
    
    if !states["course_exists"].(bool) {
        return nil, fmt.Errorf("course %s does not exist", data.CourseID)
    }
    
    if states["already_enrolled"].(bool) {
        return nil, fmt.Errorf("student %s already enrolled in course %s", data.StudentID, data.CourseID)
    }
    
    enrollmentCount := states["enrollment_count"].(int)
    // Assume course capacity is 30 for this example
    if enrollmentCount >= 30 {
        return nil, fmt.Errorf("course %s is full (capacity: 30, enrolled: %d)", data.CourseID, enrollmentCount)
    }
    
    // Create enrollment event
    event := dcb.NewEvent("StudentEnrolled").
        WithTag("student_id", data.StudentID).
        WithTag("course_id", data.CourseID).
        WithData(data).
        Build()
    
    return []dcb.InputEvent{event}, nil
}

// 4. Execute commands
courseCommand := dcb.NewCommand("DefineCourse", courseData)
courseCondition := dcb.FailIfExists("course_id", courseID)
err = commandExecutor.ExecuteCommand(ctx, courseCommand, handleDefineCourse, &courseCondition)

enrollmentCommand := dcb.NewCommand("EnrollStudent", enrollmentData)
enrollmentCondition := dcb.FailIfExists("student_id", studentID)
err = commandExecutor.ExecuteCommand(ctx, enrollmentCommand, handleEnrollStudent, &enrollmentCondition)
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

## Concurrency Control

### Primary: DCB Concurrency Control
- Uses `AppendCondition` to check for existing events before appending
- Conflict detection: Only one append succeeds when conditions match
- No blocking: Failed appends return immediately with `ConcurrencyError`
- Event ordering: Transaction IDs ensure correct, gapless ordering

### Optional: Advisory Locks
- Tag-based locking: Add tags with `lock:` prefix (e.g., "lock:user-123")
- Automatic acquisition: Database functions acquire locks on these keys before DCB condition checks
- Deadlock prevention: Locks sorted and acquired in consistent order
- Transaction-scoped: Automatically released on commit/rollback
- Performance: 1 I/O operation when used alone, 2 I/O operations when combined with DCB conditions
- Use case: Resource serialization without complex business logic validation

## Fluent API

The library provides a fluent API for common operations, reducing boilerplate by 50%:

### EventBuilder
```go
event := dcb.NewEvent("CourseDefined").
    WithTag("course_id", "CS101").
    WithTags(map[string]string{
        "department": "computer_science",
        "semester": "fall_2024",
    }).
    WithData(courseData).
    Build()
```

### QueryBuilder
```go
query := dcb.NewQueryBuilder().
    WithTag("course_id", "CS101").
    WithType("CourseDefined").
    AddItem().
    WithTag("course_id", "CS102").
    WithType("CourseDefined").
    Build()
```

### Simplified AppendConditions
```go
condition := dcb.FailIfExists("course_id", "CS101")
condition := dcb.FailIfEventType("CourseDefined", "course_id", "CS101")
```

### Projection Helpers
```go
projector := dcb.ProjectCounter("user_count", "UserRegistered", "status", "active")
projector := dcb.ProjectBoolean("user_exists", "UserRegistered", "user_id", "123")
```

### BatchBuilder
```go
batch := dcb.NewBatch().
    AddEvent(event1).
    AddEvent(event2).
    AddEventFromBuilder(eventBuilder).
    Build()
```

### Convenience Functions
```go
// Append single event with tags
err := dcb.AppendSingleEvent(ctx, store, "UserLogin", map[string]string{
    "user_id": "123",
    "ip": "192.168.1.1",
}, loginData)

// Append single event with condition
err := dcb.AppendSingleEventIf(ctx, store, "UserProfileUpdated", 
    map[string]string{"user_id": "123"}, 
    userData, 
    dcb.FailIfExists("user_id", "123"))
```

See the [Quick Start](quick-start.md) and [Examples](../internal/examples/) for complete usage examples.

## Migration from Legacy API

If you're familiar with the legacy API, here's how to migrate to the new fluent API:

### Event Creation
```go
// Legacy way
event := dcb.NewInputEvent("UserRegistered", dcb.NewTags("user_id", "123"), dcb.ToJSON(userData))

// New way
event := dcb.NewEvent("UserRegistered").
    WithTag("user_id", "123").
    WithData(userData).
    Build()
```

### Query Building
```go
// Legacy way
query := dcb.NewQuery(dcb.NewTags("user_id", "123"), "UserRegistered")

// New way
query := dcb.NewQueryBuilder().
    WithTag("user_id", "123").
    WithType("UserRegistered").
    Build()
```

### Append Conditions
```go
// Legacy way
condition := dcb.NewAppendCondition(dcb.NewQuery(dcb.NewTags("user_id", "123"), "UserRegistered"))

// New way
condition := dcb.FailIfExists("user_id", "123")
```

### Projections
```go
// Legacy way
projector := dcb.StateProjector{
    ID: "user_count",
    Query: dcb.NewQuery(dcb.NewTags("status", "active"), "UserRegistered"),
    InitialState: 0,
    TransitionFn: func(state any, event dcb.Event) any { return state.(int) + 1 },
}

// New way
projector := dcb.ProjectCounter("user_count", "UserRegistered", "status", "active")
```

## Configuration

```go
type EventStoreConfig struct {
    MaxBatchSize           int            `json:"max_batch_size"`           // Default: 1000
    LockTimeout            int            `json:"lock_timeout"`             // Default: 5000ms
    StreamBuffer           int            `json:"stream_buffer"`            // Default: 1000
    DefaultAppendIsolation IsolationLevel `json:"default_append_isolation"` // Default: Read Committed
    QueryTimeout           int            `json:"query_timeout"`            // Default: 15000ms
    AppendTimeout          int            `json:"append_timeout"`           // Default: 10000ms
}
```

## Examples

### EventStore Pattern: Course Subscription

Direct event sourcing approach using the EventStore interface:

```go
// Define projectors using fluent API
projectors := []dcb.StateProjector{
    dcb.ProjectBoolean("courseExists", "CourseDefined", "course_id", courseID),
    dcb.ProjectCounter("numSubscriptions", "StudentEnrolled", "course_id", courseID),
}

states, appendCond, _ := store.Project(ctx, projectors, nil)

if !states["courseExists"].(bool) {
    // Create course using fluent API with condition to prevent duplicates
    courseEvent := dcb.NewEvent("CourseDefined").
        WithTag("course_id", courseID).
        WithData(CourseDefined{courseID, 2}).
        Build()
    courseCondition := dcb.FailIfExists("course_id", courseID)
    store.AppendIf(ctx, []dcb.InputEvent{courseEvent}, courseCondition)
}

if states["numSubscriptions"].(int) < 2 {
    // Enroll student using fluent API with condition
    enrollmentEvent := dcb.NewEvent("StudentEnrolled").
        WithTag("student_id", studentID).
        WithTag("course_id", courseID).
        WithData(StudentEnrolled{studentID, courseID}).
        Build()
    store.AppendIf(ctx, []dcb.InputEvent{enrollmentEvent}, appendCond)
}
```

### CommandExecutor Pattern: User Registration

Command-driven approach using the CommandExecutor interface:

```go
// Define command type
type RegisterUserCommand struct {
    UserID string `json:"user_id"`
    Email  string `json:"email"`
    Name   string `json:"name"`
}

// Define command handler for user registration
func handleRegisterUser(ctx context.Context, store dcb.EventStore, cmd dcb.Command) ([]dcb.InputEvent, error) {
    var data RegisterUserCommand
    if err := json.Unmarshal(cmd.GetData(), &data); err != nil {
        return nil, fmt.Errorf("failed to unmarshal command: %w", err)
    }
    
    // Business logic validation
    if data.Email == "" {
        return nil, errors.New("email required")
    }
    if data.Name == "" {
        return nil, errors.New("name required")
    }
    if data.UserID == "" {
        return nil, errors.New("user_id required")
    }
    
    // Create user registration event
    event := dcb.NewEvent("UserRegistered").
        WithTag("user_id", data.UserID).
        WithTag("email", data.Email).
        WithData(data).
        Build()
    
    return []dcb.InputEvent{event}, nil
}

// Execute command with condition
command := dcb.NewCommand("RegisterUser", commandData)
condition := dcb.FailIfExists("email", userEmail)
err = commandExecutor.ExecuteCommand(ctx, command, handleRegisterUser, &condition)
```

## Performance

See [benchmarks documentation](benchmarks.md) for detailed performance analysis.

## Table Validation

The library validates that the `events` table exists and has the correct structure:
- Required columns: `type`, `tags`, `data`, `transaction_id`, `position`, `occurred_at`
- Returns `TableStructureError` with detailed validation failure information
