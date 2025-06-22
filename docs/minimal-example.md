# Minimal Example: Course Enrollment with DCB Pattern

This document provides a detailed walkthrough of a comprehensive course enrollment example, demonstrating the Dynamic Consistency Boundary (DCB) pattern with proper command handlers, business logic separation, and optimistic concurrency.

## Complete Example

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "github.com/rodolfodpk/go-crablet/pkg/dcb"
    "github.com/jackc/pgx/v5/pgxpool"
    "time"
)

func main() {
    ctx := context.Background()
    pool, _ := pgxpool.New(ctx, "postgres://postgres:postgres@localhost:5432/dcb_app?sslmode=disable")
    store, _ := dcb.NewEventStore(ctx, pool)

    // Command 1: Create Course
    createCourseCmd := CreateCourseCommand{
        CourseID: generateUniqueID("course"),
        Title:    "Introduction to Event Sourcing",
        Capacity: 2,
    }
    err := handleCreateCourse(ctx, store, createCourseCmd)
    if err != nil {
        log.Fatalf("Create course failed: %v", err)
    }

    // Command 2: Register Student
    registerStudentCmd := RegisterStudentCommand{
        StudentID: generateUniqueID("student"),
        Name:      "Alice",
        Email:     "alice@example.com",
    }
    err = handleRegisterStudent(ctx, store, registerStudentCmd)
    if err != nil {
        log.Fatalf("Register student failed: %v", err)
    }

    // Command 3: Enroll Student in Course
    enrollCmd := EnrollStudentCommand{
        StudentID: registerStudentCmd.StudentID,
        CourseID:  createCourseCmd.CourseID,
    }
    err = handleEnrollStudent(ctx, store, enrollCmd)
    if err != nil {
        log.Fatalf("Enroll student failed: %v", err)
    }

    fmt.Println("All commands executed successfully!")
}

// Generate unique IDs for better concurrency
func generateUniqueID(prefix string) string {
    return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}

// Command handlers with their own business rules

func handleCreateCourse(ctx context.Context, store dcb.EventStore, cmd CreateCourseCommand) error {
    // Command-specific projectors
    projectors := []dcb.BatchProjector{
        {ID: "courseExists", StateProjector: dcb.StateProjector{
            Query: dcb.NewQuery(dcb.NewTags("course_id", cmd.CourseID), "CourseDefined"),
            InitialState: false,
            TransitionFn: func(state any, e dcb.Event) any { return true },
        }},
    }

    states, appendCondition, _ := store.ProjectDecisionModel(ctx, projectors, nil)
    
    // Command-specific business rule: course must not already exist
    if states["courseExists"].(bool) {
        return fmt.Errorf("course %s already exists", cmd.CourseID)
    }

    // Create events for this command
    events := []dcb.InputEvent{
        dcb.NewInputEvent("CourseDefined", 
            dcb.NewTags("course_id", cmd.CourseID), 
            mustJSON(map[string]any{"Title": cmd.Title, "Capacity": cmd.Capacity})),
    }

    // Append events atomically for this command
    _, err := store.Append(ctx, events, &appendCondition)
    if err != nil {
        return fmt.Errorf("failed to create course: %w", err)
    }

    fmt.Printf("Created course %s with capacity %d\n", cmd.CourseID, cmd.Capacity)
    return nil
}

func handleRegisterStudent(ctx context.Context, store dcb.EventStore, cmd RegisterStudentCommand) error {
    // Command-specific projectors
    projectors := []dcb.BatchProjector{
        {ID: "studentExists", StateProjector: dcb.StateProjector{
            Query: dcb.NewQuery(dcb.NewTags("student_id", cmd.StudentID), "StudentRegistered"),
            InitialState: false,
            TransitionFn: func(state any, e dcb.Event) any { return true },
        }},
    }

    states, appendCondition, _ := store.ProjectDecisionModel(ctx, projectors, nil)
    
    // Command-specific business rule: student must not already exist
    if states["studentExists"].(bool) {
        return fmt.Errorf("student %s already exists", cmd.StudentID)
    }

    // Create events for this command
    events := []dcb.InputEvent{
        dcb.NewInputEvent("StudentRegistered", 
            dcb.NewTags("student_id", cmd.StudentID), 
            mustJSON(map[string]any{"Name": cmd.Name, "Email": cmd.Email})),
    }

    // Append events atomically for this command
    _, err := store.Append(ctx, events, &appendCondition)
    if err != nil {
        return fmt.Errorf("failed to register student: %w", err)
    }

    fmt.Printf("Registered student %s (%s)\n", cmd.Name, cmd.Email)
    return nil
}

func handleEnrollStudent(ctx context.Context, store dcb.EventStore, cmd EnrollStudentCommand) error {
    // Command-specific projectors
    projectors := []dcb.BatchProjector{
        {ID: "courseState", StateProjector: dcb.StateProjector{
            Query: dcb.NewQuery(dcb.NewTags("course_id", cmd.CourseID), "CourseDefined", "StudentEnrolled"),
            InitialState: &CourseState{Capacity: 0, Enrolled: 0},
            TransitionFn: func(state any, e dcb.Event) any {
                course := state.(*CourseState)
                switch e.Type {
                case "CourseDefined":
                    var data struct{ Capacity int }
                    json.Unmarshal(e.Data, &data)
                    course.Capacity = data.Capacity
                case "StudentEnrolled":
                    course.Enrolled++
                }
                return course
            },
        }},
        {ID: "studentEnrollmentCount", StateProjector: dcb.StateProjector{
            Query: dcb.NewQuery(dcb.NewTags("student_id", cmd.StudentID, "course_id", cmd.CourseID), "StudentEnrolled"),
            InitialState: 0,
            TransitionFn: func(state any, e dcb.Event) any { return state.(int) + 1 },
        }},
    }

    states, appendCondition, _ := store.ProjectDecisionModel(ctx, projectors, nil)
    
    course := states["courseState"].(*CourseState)
    enrollmentCount := states["studentEnrollmentCount"].(int)

    // Command-specific business rules
    if course.Enrolled >= course.Capacity {
        return fmt.Errorf("course %s is full (capacity: %d, enrolled: %d)", cmd.CourseID, course.Capacity, course.Enrolled)
    }
    if enrollmentCount > 0 {
        return fmt.Errorf("student %s is already enrolled in course %s", cmd.StudentID, cmd.CourseID)
    }

    // Create events for this command
    events := []dcb.InputEvent{
        dcb.NewInputEvent("StudentEnrolled", 
            dcb.NewTags("student_id", cmd.StudentID, "course_id", cmd.CourseID), 
            mustJSON(map[string]any{"StudentID": cmd.StudentID, "CourseID": cmd.CourseID})),
    }

    // Append events atomically for this command
    _, err := store.Append(ctx, events, &appendCondition)
    if err != nil {
        return fmt.Errorf("failed to enroll student: %w", err)
    }

    fmt.Printf("Enrolled student %s in course %s\n", cmd.StudentID, cmd.CourseID)
    return nil
}

// Command types
type CreateCourseCommand struct {
    CourseID string
    Title    string
    Capacity int
}

type RegisterStudentCommand struct {
    StudentID string
    Name      string
    Email     string
}

type EnrollStudentCommand struct {
    StudentID string
    CourseID  string
}

type CourseState struct {
    Capacity int
    Enrolled int
}

func mustJSON(v any) []byte {
    data, _ := json.Marshal(v)
    return data
}
```

## Key Features Demonstrated

### 1. **Command Pattern**
Each business operation is encapsulated in a command handler:
- `handleCreateCourse`: Creates a new course with validation
- `handleRegisterStudent`: Registers a new student with duplicate checking
- `handleEnrollStudent`: Enrolls a student with capacity and duplicate enrollment checks

### 2. **DCB Pattern Implementation**
- **Batch Projectors**: Each command defines its own projectors to read relevant state
- **Optimistic Concurrency**: Uses `appendCondition` to ensure atomic operations
- **Business Rules**: Validates business constraints before appending events

### 3. **State Projection**
- **Course State**: Tracks course capacity and current enrollment count
- **Student Enrollment Count**: Prevents duplicate enrollments
- **Existence Checks**: Validates that entities exist before operations

### 4. **Event Sourcing Benefits**
- **Audit Trail**: Complete history of all operations
- **Concurrency Safety**: Optimistic locking prevents race conditions
- **Business Rule Enforcement**: Rules are enforced at the event level

## Resulting Events

After running the example, the events table will contain:

```sql
SELECT type, tags, data, position 
FROM events 
ORDER BY position;
```

| type | tags | data | position |
|------|------|------|----------|
| CourseDefined | `{"course_id": "course-1234567890"}` | `{"Title": "Introduction to Event Sourcing", "Capacity": 2}` | 1 |
| StudentRegistered | `{"student_id": "student-1234567891"}` | `{"Name": "Alice", "Email": "alice@example.com"}` | 2 |
| StudentEnrolled | `{"student_id": "student-1234567891", "course_id": "course-1234567890"}` | `{"StudentID": "student-1234567891", "CourseID": "course-1234567890"}` | 3 |

## Business Rules Enforced

1. **Course Creation**: Cannot create a course that already exists
2. **Student Registration**: Cannot register a student that already exists
3. **Course Enrollment**: 
   - Cannot enroll in a course that's at capacity
   - Cannot enroll the same student twice in the same course
   - Course must exist before enrollment

## Benefits of This Approach

- **Separation of Concerns**: Each command handler is focused on its specific business logic
- **Reusability**: Command handlers can be called independently
- **Testability**: Each handler can be tested in isolation
- **Maintainability**: Business rules are clearly defined and easy to modify
- **Scalability**: Commands can be processed in parallel with proper concurrency control

## Running the Example

To run this complete course enrollment example:

```bash
go run internal/examples/enrollment/main.go
```

This will execute the full course enrollment workflow with all business rules and optimistic concurrency controls. 