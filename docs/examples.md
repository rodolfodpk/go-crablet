# Usage Examples

This document provides practical examples of using go-crablet in different scenarios.

## Basic Usage

Here's a simple example of how to use go-crablet to store and query events:

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/rodolfodpk/go-crablet"
)

func main() {
    // Create a PostgreSQL connection pool
    pool, err := pgxpool.New(context.Background(), "postgres://user:pass@localhost:5432/mydb?sslmode=disable")
    if err != nil {
        log.Fatal(err)
    }
    defer pool.Close()

    // Create a new event store
    store, err := dcb.NewEventStore(context.Background(), pool)
    if err != nil {
        log.Fatal(err)
    }

    ctx := context.Background()

    // Create tags for the event
    tags := dcb.NewTags(
        "course_id", "C123",
        "student_id", "S456",
    )

    // Create a new event
    event := dcb.NewInputEvent(
        "StudentSubscribedToCourse", 
        tags, 
        []byte(`{"subscription_date": "2024-03-20", "payment_method": "credit_card"}`),
    )

    // Define the consistency boundary
    query := dcb.NewQuery(tags, "StudentSubscribedToCourse")

    // Get current stream position
    position, err := store.GetCurrentPosition(ctx, query)
    if err != nil {
        log.Fatal(err)
    }

    // Append the event to the store using the current position
    newPosition, err := store.AppendEvents(ctx, []dcb.InputEvent{event}, query, position)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Event appended at position %d\n", newPosition)
}
```

## Course Subscription System with Invariant Rules

Here's a complete example of a course subscription system using go-crablet with two important invariant rules:

1. **A student cannot enroll in more than 10 courses**
2. **A course cannot have more than 30 students**

This example demonstrates:

- Event creation and appending
- State projection for multiple entities
- Consistency boundaries
- Tag-based querying
- **Comprehensive error handling**
- **Business rule enforcement**

```go
package main

import (
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "log"
    "time"

    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/rodolfodpk/go-crablet"
)

// Business rule errors
var (
    ErrStudentCourseLimitExceeded = errors.New("student cannot enroll in more than 10 courses")
    ErrCourseStudentLimitExceeded = errors.New("course cannot have more than 30 students")
    ErrStudentAlreadyEnrolled     = errors.New("student is already enrolled in this course")
    ErrStudentNotEnrolled         = errors.New("student is not enrolled in this course")
)

// CourseState represents the current state of a course
type CourseState struct {
    ID          string
    Name        string
    StudentIDs  map[string]bool
    IsActive    bool
}

// StudentState represents the current state of a student
type StudentState struct {
    ID         string
    Name       string
    CourseIDs  map[string]bool
    IsActive   bool
}

// CourseEnrollmentService handles course enrollment with business rules
type CourseEnrollmentService struct {
    store dcb.EventStore
}

// NewCourseEnrollmentService creates a new enrollment service
func NewCourseEnrollmentService(store dcb.EventStore) *CourseEnrollmentService {
    return &CourseEnrollmentService{store: store}
}

// EnrollStudent enrolls a student in a course with business rule validation
func (s *CourseEnrollmentService) EnrollStudent(ctx context.Context, courseID, studentID string) error {
    // Get current states to validate business rules
    courseState, err := s.getCourseState(ctx, courseID)
    if err != nil {
        return fmt.Errorf("failed to get course state: %w", err)
    }

    studentState, err := s.getStudentState(ctx, studentID)
    if err != nil {
        return fmt.Errorf("failed to get student state: %w", err)
    }

    // Validate business rules
    if err := s.validateEnrollmentRules(courseState, studentState, courseID, studentID); err != nil {
        return err
    }

    // Create enrollment event
    enrollmentTags := dcb.NewTags(
        "course_id", courseID,
        "student_id", studentID,
    )
    
    enrollmentData := map[string]interface{}{
        "enrollment_date": time.Now().Format(time.RFC3339),
        "status":          "active",
    }
    
    data, err := json.Marshal(enrollmentData)
    if err != nil {
        return fmt.Errorf("failed to marshal enrollment data: %w", err)
    }

    enrollmentEvent := dcb.NewInputEvent(
        "StudentEnrolledInCourse",
        enrollmentTags,
        data,
    )

    // Get current position for consistency
    query := dcb.NewQuery(enrollmentTags, "StudentEnrolledInCourse", "StudentUnenrolledFromCourse")
    position, err := s.store.GetCurrentPosition(ctx, query)
    if err != nil {
        return fmt.Errorf("failed to get current position: %w", err)
    }

    // Append the enrollment event
    _, err = s.store.AppendEvents(ctx, []dcb.InputEvent{enrollmentEvent}, query, position)
    if err != nil {
        return fmt.Errorf("failed to append enrollment event: %w", err)
    }

    return nil
}

// UnenrollStudent removes a student from a course
func (s *CourseEnrollmentService) UnenrollStudent(ctx context.Context, courseID, studentID string) error {
    // Get current states to validate business rules
    courseState, err := s.getCourseState(ctx, courseID)
    if err != nil {
        return fmt.Errorf("failed to get course state: %w", err)
    }

    // Check if student is enrolled
    if !courseState.StudentIDs[studentID] {
        return ErrStudentNotEnrolled
    }

    // Create unenrollment event
    unenrollmentTags := dcb.NewTags(
        "course_id", courseID,
        "student_id", studentID,
    )
    
    unenrollmentData := map[string]interface{}{
        "unenrollment_date": time.Now().Format(time.RFC3339),
        "status":            "inactive",
    }
    
    data, err := json.Marshal(unenrollmentData)
    if err != nil {
        return fmt.Errorf("failed to marshal unenrollment data: %w", err)
    }

    unenrollmentEvent := dcb.NewInputEvent(
        "StudentUnenrolledFromCourse",
        unenrollmentTags,
        data,
    )

    // Get current position for consistency
    query := dcb.NewQuery(unenrollmentTags, "StudentEnrolledInCourse", "StudentUnenrolledFromCourse")
    position, err := s.store.GetCurrentPosition(ctx, query)
    if err != nil {
        return fmt.Errorf("failed to get current position: %w", err)
    }

    // Append the unenrollment event
    _, err = s.store.AppendEvents(ctx, []dcb.InputEvent{unenrollmentEvent}, query, position)
    if err != nil {
        return fmt.Errorf("failed to append unenrollment event: %w", err)
    }

    return nil
}

// validateEnrollmentRules checks business rules before enrollment
func (s *CourseEnrollmentService) validateEnrollmentRules(courseState *CourseState, studentState *StudentState, courseID, studentID string) error {
    // Check if student is already enrolled
    if courseState.StudentIDs[studentID] {
        return ErrStudentAlreadyEnrolled
    }

    // Check course student limit (max 30 students)
    if len(courseState.StudentIDs) >= 30 {
        return ErrCourseStudentLimitExceeded
    }

    // Check student course limit (max 10 courses)
    if len(studentState.CourseIDs) >= 10 {
        return ErrStudentCourseLimitExceeded
    }

    return nil
}

// getCourseState retrieves the current state of a course
func (s *CourseEnrollmentService) getCourseState(ctx context.Context, courseID string) (*CourseState, error) {
    courseTags := dcb.NewTags("course_id", courseID)
    courseProjector := dcb.StateProjector{
        Query: dcb.NewQuery(courseTags),
        InitialState: &CourseState{
            ID:         courseID,
            StudentIDs: make(map[string]bool),
        },
        TransitionFn: func(state any, event dcb.Event) any {
            course := state.(*CourseState)
            switch event.Type {
            case "CourseCreated":
                var data struct {
                    Name string `json:"name"`
                }
                if err := json.Unmarshal(event.Data, &data); err != nil {
                    panic(err)
                }
                course.Name = data.Name
                course.IsActive = true
            case "StudentEnrolledInCourse":
                for _, tag := range event.Tags {
                    if tag.Key == "student_id" {
                        course.StudentIDs[tag.Value] = true
                    }
                }
            case "StudentUnenrolledFromCourse":
                for _, tag := range event.Tags {
                    if tag.Key == "student_id" {
                        delete(course.StudentIDs, tag.Value)
                    }
                }
            case "CourseCancelled":
                course.IsActive = false
            }
            return course
        },
    }

    _, courseState, err := s.store.ProjectState(ctx, courseProjector)
    if err != nil {
        return nil, err
    }

    return courseState.(*CourseState), nil
}

// getStudentState retrieves the current state of a student
func (s *CourseEnrollmentService) getStudentState(ctx context.Context, studentID string) (*StudentState, error) {
    studentTags := dcb.NewTags("student_id", studentID)
    studentProjector := dcb.StateProjector{
        Query: dcb.NewQuery(studentTags),
        InitialState: &StudentState{
            ID:        studentID,
            CourseIDs: make(map[string]bool),
        },
        TransitionFn: func(state any, event dcb.Event) any {
            student := state.(*StudentState)
            switch event.Type {
            case "StudentRegistered":
                var data struct {
                    Name string `json:"name"`
                }
                if err := json.Unmarshal(event.Data, &data); err != nil {
                    panic(err)
                }
                student.Name = data.Name
                student.IsActive = true
            case "StudentEnrolledInCourse":
                for _, tag := range event.Tags {
                    if tag.Key == "course_id" {
                        student.CourseIDs[tag.Value] = true
                    }
                }
            case "StudentUnenrolledFromCourse":
                for _, tag := range event.Tags {
                    if tag.Key == "course_id" {
                        delete(student.CourseIDs, tag.Value)
                    }
                }
            case "StudentDeactivated":
                student.IsActive = false
            }
            return student
        },
    }

    _, studentState, err := s.store.ProjectState(ctx, studentProjector)
    if err != nil {
        return nil, err
    }

    return studentState.(*StudentState), nil
}

func main() {
    // Create a PostgreSQL connection pool
    pool, err := pgxpool.New(context.Background(), "postgres://user:pass@localhost:5432/mydb?sslmode=disable")
    if err != nil {
        log.Fatal(err)
    }
    defer pool.Close()

    // Create a new event store
    store, err := dcb.NewEventStore(context.Background(), pool)
    if err != nil {
        log.Fatal(err)
    }

    // Create enrollment service
    enrollmentService := NewCourseEnrollmentService(store)
    ctx := context.Background()

    // Example usage with error handling
    courseID := "C123"
    studentID := "S456"

    // Try to enroll a student
    err = enrollmentService.EnrollStudent(ctx, courseID, studentID)
    if err != nil {
        switch {
        case errors.Is(err, ErrStudentCourseLimitExceeded):
            fmt.Printf("Cannot enroll: %v\n", err)
        case errors.Is(err, ErrCourseStudentLimitExceeded):
            fmt.Printf("Cannot enroll: %v\n", err)
        case errors.Is(err, ErrStudentAlreadyEnrolled):
            fmt.Printf("Cannot enroll: %v\n", err)
        default:
            fmt.Printf("Unexpected error: %v\n", err)
        }
        return
    }

    fmt.Printf("Successfully enrolled student %s in course %s\n", studentID, courseID)
}
```

## Error Handling

Here's an example showing how to handle different types of errors that can occur when using go-crablet:

### Validation Errors
These occur when event data doesn't meet the required format or constraints. For example, when JSON data is invalid or required fields are missing.

```go
// Example of handling validation errors
courseID := "C123"
courseTags := dcb.NewTags("course_id", courseID)
query := dcb.NewQuery(courseTags, "CourseUpdated")

// Try to append with invalid event data
invalidEvent := dcb.NewInputEvent(
    "CourseUpdated", 
    courseTags, 
    []byte(`invalid json`), // Invalid JSON data
)

_, err = store.AppendEvents(ctx, []dcb.InputEvent{invalidEvent}, query, 0)
if err != nil {
    if validationErr, ok := err.(*dcb.ValidationError); ok {
        fmt.Printf("Validation error: %v\n", validationErr)
        return
    }
    log.Fatal(err)
}
```

### Concurrency Errors
These occur when multiple processes try to modify the same event stream simultaneously. The event store uses optimistic concurrency control to detect and prevent conflicts.

```go
// Example of handling concurrency errors
// First append
event1 := dcb.NewInputEvent(
    "CourseUpdated", 
    courseTags, 
    []byte(`{"title": "New Title"}`),
)
position, err := store.AppendEvents(ctx, []dcb.InputEvent{event1}, query, 0)
if err != nil {
    log.Fatal(err)
}

// Try to append another event with the same query but old position
event2 := dcb.NewInputEvent(
    "CourseUpdated", 
    courseTags, 
    []byte(`{"title": "Another Title"}`),
)
_, err = store.AppendEvents(ctx, []dcb.InputEvent{event2}, query, 0) // Using position 0 instead of the new position
if err != nil {
    if _, ok := err.(*dcb.ConcurrencyError); ok {
        fmt.Println("Concurrency error: another event was appended to this stream")
        return
    }
    log.Fatal(err)
}
```

### Business Rule Errors
These occur when operations violate domain-specific business rules. The example above demonstrates handling enrollment limit errors.

```go
// Example of handling business rule errors
enrollmentService := NewCourseEnrollmentService(store)

// Try to enroll a student
err := enrollmentService.EnrollStudent(ctx, courseID, studentID)
if err != nil {
    switch {
    case errors.Is(err, ErrStudentCourseLimitExceeded):
        fmt.Printf("Business rule violation: %v\n", err)
        // Handle student course limit exceeded
    case errors.Is(err, ErrCourseStudentLimitExceeded):
        fmt.Printf("Business rule violation: %v\n", err)
        // Handle course student limit exceeded
    case errors.Is(err, ErrStudentAlreadyEnrolled):
        fmt.Printf("Business rule violation: %v\n", err)
        // Handle duplicate enrollment
    default:
        fmt.Printf("Unexpected error: %v\n", err)
        // Handle other errors
    }
    return
}
```

### Resource Errors
These occur when there are issues with the underlying database or network connectivity.

```go
// Example of handling resource errors
store, err := dcb.NewEventStore(ctx, pool)
if err != nil {
    if resourceErr, ok := err.(*dcb.ResourceError); ok {
        fmt.Printf("Resource error: %v\n", resourceErr)
        // Handle database connection issues
        return
    }
    log.Fatal(err)
}
```

### Comprehensive Error Handling Pattern
Here's a pattern for comprehensive error handling in your application:

```go
func handleEventStoreError(err error, operation string) {
    if err == nil {
        return
    }

    switch {
    case errors.Is(err, &dcb.ValidationError{}):
        fmt.Printf("Validation error in %s: %v\n", operation, err)
        // Log validation errors for debugging
        log.Printf("Validation error details: %+v", err)
        
    case errors.Is(err, &dcb.ConcurrencyError{}):
        fmt.Printf("Concurrency error in %s: %v\n", operation, err)
        // Implement retry logic or notify user
        // Consider implementing exponential backoff
        
    case errors.Is(err, &dcb.ResourceError{}):
        fmt.Printf("Resource error in %s: %v\n", operation, err)
        // Check database connectivity
        // Implement circuit breaker pattern
        
    case errors.Is(err, ErrStudentCourseLimitExceeded):
        fmt.Printf("Business rule violation in %s: %v\n", operation, err)
        // Notify user about course limit
        
    case errors.Is(err, ErrCourseStudentLimitExceeded):
        fmt.Printf("Business rule violation in %s: %v\n", operation, err)
        // Notify user about course capacity
        
    default:
        fmt.Printf("Unexpected error in %s: %v\n", operation, err)
        // Log unexpected errors for investigation
        log.Printf("Unexpected error details: %+v", err)
    }
}
```

## Key Features Demonstrated

1. **Event Types and Data**
   - Course events: `CourseCreated`, `CourseCancelled`
   - Student events: `StudentRegistered`, `StudentDeactivated`
   - Enrollment events: `StudentEnrolledInCourse`, `StudentUnenrolledFromCourse`

2. **State Projection**
   - Separate projectors for course and student states
   - Efficient tag-based filtering
   - Type-safe event handling

3. **Consistency Boundaries**
   - Events for the same course/student are processed atomically
   - Concurrent modifications are detected and handled
   - Event ordering is maintained

4. **Tag Management**
   - Using tags to link related events
   - Efficient querying by course and student IDs
   - Building different views of the same event stream

5. **Business Rule Enforcement**
   - Student course limit (max 10 courses)
   - Course student limit (max 30 students)
   - Duplicate enrollment prevention
   - Proper error handling for rule violations

6. **Comprehensive Error Handling**
   - Validation errors
   - Concurrency errors
   - Business rule errors
   - Resource errors
   - Structured error handling patterns

## Best Practices

1. **Tag Usage**
   - Use consistent tag keys (`course_id`, `student_id`)
   - Include all relevant IDs in event tags
   - Use tags for efficient querying

2. **Event Types**
   - Use descriptive event type names
   - Group related events by domain concept
   - Maintain consistent naming conventions

3. **State Structure**
   - Keep state objects focused and minimal
   - Use maps for efficient lookups
   - Include only necessary fields

4. **Error Handling**
   - Check for errors after each operation
   - Handle concurrency errors appropriately
   - Validate event data before appending
   - Implement business rule validation
   - Use structured error handling patterns

5. **Position Management**
   - Always get current position before appending
   - Use batch operations for related events
   - Handle position updates atomically

6. **Business Rules**
   - Validate rules before appending events
   - Use clear error messages for rule violations
   - Implement proper error handling for each rule type
   - Consider implementing retry logic for transient errors

For more details about specific features used in these examples, see:
- [Appending Events](docs/appending-events.md): Learn about event appending and concurrency control
- [State Projection](docs/state-projection.md): Understand how state projection works
- [Tutorial](docs/tutorial.md): Get started with go-crablet
- [Course Subscription Example](docs/course-subscription.md): See a complete implementation of the course subscription system 