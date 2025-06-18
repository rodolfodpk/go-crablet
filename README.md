[![Go Report Card](https://goreportcard.com/badge/github.com/rodolfodpk/go-crablet)](https://goreportcard.com/report/github.com/rodolfodpk/go-crablet)
[![Code Coverage](https://img.shields.io/badge/code%20coverage-86.7%25-green?logo=go)](https://github.com/rodolfodpk/go-crablet/actions/workflows/coverage.yml)
[![GoDoc](https://godoc.org/github.com/rodolfodpk/go-crablet?status.svg)](https://godoc.org/github.com/rodolfodpk/go-crablet)
[![License](https://img.shields.io/github/license/rodolfodpk/go-crablet)](https://github.com/rodolfodpk/go-crablet/blob/main/LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/rodolfodpk/go-crablet)](https://github.com/rodolfodpk/go-crablet/blob/main/go.mod)

# go-crablet

A Go library for event sourcing, exploring and learning about concepts inspired by Sara Pellegrini's Dynamic Consistency Boundary (DCB) pattern. go-crablet enables you to build event-driven systems with:

## Key Features

- **DCB-inspired decision models**: Project multiple states and build append conditions in one step
- **Single streamlined query**: Efficiently project all relevant states using PostgreSQL's native streaming via pgx
- **Optimistic concurrency**: Append events only if no conflicting events have appeared within the same query combination scope
- **Memory-efficient streaming**: Process events row-by-row for large event streams
- **Flexible queries**: Tag-based, OR-combined queries for cross-entity boundaries
- **PostgreSQL-backed**: Uses PostgreSQL for robust, concurrent event storage

## Documentation
- [Overview](docs/overview.md): DCB pattern exploration, batch projection, and streaming
- [Examples](docs/examples.md): DCB-inspired use cases
- [Implementation](docs/implementation.md): Technical details
- [Causation and Correlation](docs/causation-correlation.md): Understanding event relationships and tracing
- [Minimal Example](docs/minimal-example.md): Detailed walkthrough of the course subscription example
- [Code Coverage](docs/code-coverage.md): Test coverage analysis and improvement guidelines
- [Performance Benchmarks](internal/benchmarks/README.md): Comprehensive performance testing and analysis
- [k6 Performance Benchmarks](internal/web-app/k6-benchmark-report.md): Detailed performance test results for REST API

## Minimal Example: Batch Append with DCB Invariants

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "github.com/rodolfodpk/go-crablet/pkg/dcb"
    "github.com/jackc/pgx/v5/pgxpool"
)

func main() {
    ctx := context.Background()
    pool, _ := pgxpool.New(ctx, "postgres://postgres:postgres@localhost:5432/dcb_app?sslmode=disable")
    store, _ := dcb.NewEventStore(ctx, pool)

    // Command 1: Create Course
    createCourseCmd := CreateCourseCommand{
        CourseID: "c1",
        Title:    "Introduction to Event Sourcing",
        Capacity: 2,
    }
    err := handleCreateCourse(ctx, store, createCourseCmd)
    if err != nil {
        log.Fatalf("Create course failed: %v", err)
    }

    // Command 2: Register Student
    registerStudentCmd := RegisterStudentCommand{
        StudentID: "s1",
        Name:      "Alice",
        Email:     "alice@example.com",
    }
    err = handleRegisterStudent(ctx, store, registerStudentCmd)
    if err != nil {
        log.Fatalf("Register student failed: %v", err)
    }

    // Command 3: Enroll Student in Course
    enrollCmd := EnrollStudentCommand{
        StudentID: "s1",
        CourseID:  "c1",
    }
    err = handleEnrollStudent(ctx, store, enrollCmd)
    if err != nil {
        log.Fatalf("Enroll student failed: %v", err)
    }

    fmt.Println("All commands executed successfully!")
}

// Command handlers with their own business rules

func handleCreateCourse(ctx context.Context, store dcb.EventStore, cmd CreateCourseCommand) error {
    // Command-specific projectors
    projectors := []dcb.BatchProjector{
        {ID: "courseExists", StateProjector: dcb.StateProjector{
            Query: dcb.NewQuery(dcb.NewTags("course_id", cmd.CourseID), "CourseCreated"),
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
        dcb.NewInputEvent("CourseCreated", 
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
            Query: dcb.NewQuery(dcb.NewTags("course_id", cmd.CourseID), "CourseCreated", "StudentEnrolled"),
            InitialState: &CourseState{Capacity: 0, Enrolled: 0},
            TransitionFn: func(state any, e dcb.Event) any {
                course := state.(*CourseState)
                switch e.Type {
                case "CourseCreated":
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

## Examples

Ready-to-run examples demonstrating different aspects of the DCB pattern:

- **[Transfer Example](internal/examples/transfer/main.go)**: **Batch append demonstration** - Money transfer between accounts with account creation and transfer in a single atomic batch operation
- **[Course Enrollment](internal/examples/enrollment/main.go)**: Student course enrollment with capacity limits and business rules
- **[Streaming Projections](internal/examples/streaming_projection/main.go)**: Memory-efficient event processing with multiple projections
- **[Decision Model](internal/examples/decision_model/main.go)**: Exploring Dynamic Consistency Boundary concepts with multiple projectors
- **[Cursor Streaming](internal/examples/cursor_streaming/main.go)**: Large dataset processing with batching and streaming
- **[ReadStream](internal/examples/readstream/main.go)**: Event streaming with projections and optimistic locking

**Batch Append Examples:**
- **Transfer Example**: Creates accounts and transfers money atomically in a single batch
- **Decision Model**: Demonstrates batch append with optimistic locking
- **Streaming Projection**: Shows batch processing for large datasets

Run any example with: `go run internal/examples/[example-name]/main.go`

**Note**: Examples require a PostgreSQL 17.5+ database. You can use the provided `docker-compose.yaml` to start a local PostgreSQL instance.

## Getting Started

If you're new to Go and want to run the examples, follow these essential steps:

### Prerequisites
1. **Install Go** (1.24+): Download from [golang.org](https://golang.org/dl/)
2. **Install Docker**: Download from [docker.com](https://docker.com/get-started/)
3. **Install Git**: Download from [git-scm.com](https://git-scm.com/)

### Quick Start
1. **Clone the repository:**
   ```bash
   git clone https://github.com/rodolfodpk/go-crablet.git
   cd go-crablet
   ```

2. **Start PostgreSQL database:**
   ```bash
   docker-compose up -d
   ```

3. **Run an example:**
   ```bash
   go run internal/examples/decision_model/main.go
   ```

## References

- [Dynamic Consistency Boundary (DCB)](https://dcb.events/) - A very good resource to understand the DCB pattern and its applications in event-driven systems
- [I am here to kill the aggregate](https://sara.event-thinking.io/2023/04/kill-aggregate-chapter-1-I-am-here-to-kill-the-aggregate.html) - Sara Pellegrini's blog post about moving beyond aggregates in event-driven systems
- [Kill Aggregate - Volume 2 - Sara Pellegrini at JOTB25](https://www.youtube.com/watch?v=AQ5fk4D3u9I)
- **DCB Bench API Specification**: [OpenAPI 3.0.3](https://app.swaggerhub.com/apis/wwwision/dcb-bench/1.0.0) - Official API specification for DCB Bench

---

## 📄 **License**

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## 📊 **Benchmark Results**

For comprehensive performance benchmarks and analysis, see the [Performance Benchmarks documentation](internal/benchmarks/README.md).