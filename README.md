[![Go Report Card](https://goreportcard.com/badge/github.com/rodolfodpk/go-crablet)](https://goreportcard.com/report/github.com/rodolfodpk/go-crablet)
[![Code Coverage](https://img.shields.io/badge/code%20coverage-86.7%25-green?logo=go)](https://github.com/rodolfodpk/go-crablet/actions/workflows/coverage.yml)
[![GoDoc](https://godoc.org/github.com/rodolfodpk/go-crablet?status.svg)](https://godoc.org/github.com/rodolfodpk/go-crablet)
[![License](https://img.shields.io/github/license/rodolfodpk/go-crablet)](https://github.com/rodolfodpk/go-crablet/blob/main/LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/rodolfodpk/go-crablet)](https://github.com/rodolfodpk/go-crablet/blob/main/go.mod)

# go-crablet

A Go library for event sourcing, exploring concepts inspired by Sara Pellegrini's Dynamic Consistency Boundary (DCB) pattern. go-crablet enables you to build event-driven systems with:

## Key Features

- **DCB-inspired decision models**: Project multiple states and build append conditions in one step
- **Single streamlined query**: Efficiently project all relevant states using PostgreSQL's native streaming via pgx
- **Optimistic concurrency**: Append events only if no conflicting events have appeared within the same query combination scope
- **Memory-efficient streaming**: Process events row-by-row for large event streams
- **Flexible queries**: Tag-based, OR-combined queries for cross-entity boundaries
- **PostgreSQL-backed**: Uses PostgreSQL for robust, concurrent event storage

## Exploring the DCB Pattern in Go

We're learning about the Dynamic Consistency Boundary (DCB) pattern by exploring how to:
- Define projections ("decision models") that provide the data business rules need
- Project all relevant state in a single query
- Build a combined append condition for optimistic locking
- Append new events only if all invariants still hold

## Documentation
- [Overview](docs/overview.md): DCB pattern exploration, batch projection, and streaming
- [Examples](docs/examples.md): DCB-inspired use cases
- [Implementation](docs/implementation.md): Technical details
- [Causation and Correlation](docs/causation-correlation.md): Understanding event relationships and tracing
- [Minimal Example](docs/minimal-example.md): Detailed walkthrough of the course subscription example
- [Performance Benchmarks](internal/benchmarks/README.md): Comprehensive performance testing and analysis
- [Code Coverage](docs/code-coverage.md): Test coverage analysis and improvement guidelines

## Minimal Example: Course Subscription

```go
package main

import (
    "context"
    "encoding/json"
    "github.com/rodolfodpk/go-crablet/pkg/dcb"
    "github.com/jackc/pgx/v5/pgxpool"
)

func main() {
    ctx := context.Background()
    pool, _ := pgxpool.New(ctx, "postgres://user:pass@localhost/db")
    store, _ := dcb.NewEventStore(ctx, pool)

    // Define projectors for business rules
    projectors := []dcb.BatchProjector{
        {ID: "courseExists", StateProjector: dcb.StateProjector{
            Query: dcb.NewQueryFromItems(dcb.QItemKV("CourseDefined", "course_id", "c1")),
            InitialState: false,
            TransitionFn: func(state any, e dcb.Event) any { return true },
        }},
        {ID: "studentExists", StateProjector: dcb.StateProjector{
            Query: dcb.NewQueryFromItems(dcb.QItemKV("StudentRegistered", "student_id", "s1")),
            InitialState: false,
            TransitionFn: func(state any, e dcb.Event) any { return true },
        }},
        {ID: "numSubscriptions", StateProjector: dcb.StateProjector{
            Query: dcb.NewQueryFromItems(dcb.QItemKV("StudentSubscribed", "course_id", "c1")),
            InitialState: 0,
            TransitionFn: func(state any, e dcb.Event) any { return state.(int) + 1 },
        }},
        {ID: "studentCourseCount", StateProjector: dcb.StateProjector{
            Query: dcb.NewQueryFromItems(dcb.QItemKV("StudentSubscribed", "student_id", "s1")),
            InitialState: 0,
            TransitionFn: func(state any, e dcb.Event) any { return state.(int) + 1 },
        }},
    }

    // Project states and get append condition (DCB pattern)
    // The query is automatically combined from all projectors using OR logic
    states, appendCondition, _ := store.ProjectDecisionModel(ctx, projectors, nil)
    
    // Business logic: create course if it doesn't exist
    if !states["courseExists"].(bool) {
        data, _ := json.Marshal(map[string]any{"CourseID": "c1", "Capacity": 2})
        courseEvent := dcb.NewInputEvent("CourseDefined", dcb.NewTags("course_id", "c1"), data)
        store.Append(ctx, []dcb.InputEvent{courseEvent}, &appendCondition)
    }
    
    // Business logic: create student if doesn't exist
    if !states["studentExists"].(bool) {
        data, _ := json.Marshal(map[string]any{"StudentID": "s1", "Name": "Alice", "Email": "alice@example.com"})
        studentEvent := dcb.NewInputEvent("StudentRegistered", dcb.NewTags("student_id", "s1"), data)
        store.Append(ctx, []dcb.InputEvent{studentEvent}, &appendCondition)
    }
    
    // Business logic: check course capacity (max 2 students)
    if states["numSubscriptions"].(int) >= 2 {
        panic("course is full")
    }
    
    // Business logic: check student course limit (max 10 courses)
    if states["studentCourseCount"].(int) >= 10 {
        panic("student cannot subscribe to more than 10 courses")
    }
    
    // Business logic: subscribe student (all invariants satisfied)
    data, _ := json.Marshal(map[string]any{"StudentID": "s1", "CourseID": "c1"})
    enrollEvent := dcb.NewInputEvent("StudentSubscribed", dcb.NewTags("student_id", "s1", "course_id", "c1"), data)
    store.Append(ctx, []dcb.InputEvent{enrollEvent}, &appendCondition)
}
```

**What we're exploring:**
- **ProjectDecisionModel**: Projects multiple states in one query
- **AppendCondition**: Optimistic locking for consistency
- **BatchProjector**: Defines business rules and state transitions

## Examples

Ready-to-run examples demonstrating different aspects of the DCB pattern:

- **[Transfer Example](internal/examples/transfer/main.go)**: Money transfer between accounts with balance validation and optimistic locking
- **[Course Enrollment](internal/examples/enrollment/main.go)**: Student course enrollment with capacity limits and business rules
- **[Streaming Projections](internal/examples/streaming_projection/main.go)**: Memory-efficient event processing with multiple projections
- **[Decision Model](internal/examples/decision_model/main.go)**: Complete DCB pattern implementation with multiple projectors
- **[Cursor Streaming](internal/examples/cursor_streaming/main.go)**: Large dataset processing with batching and streaming
- **[ReadStream](internal/examples/readstream/main.go)**: Event streaming with projections and optimistic locking

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

### Available Examples
- `internal/examples/decision_model/main.go` - Complete DCB pattern

## References

- [Dynamic Consistency Boundary (DCB)](https://dcb.events/) - A very good resource to understand the DCB pattern and its applications in event-driven systems
- [I am here to kill the aggregate](https://sara.event-thinking.io/2023/04/kill-aggregate-chapter-1-I-am-here-to-kill-the-aggregate.html) - Sara Pellegrini's blog post about moving beyond aggregates in event-driven systems
- [Kill Aggregate - Volume 2 - Sara Pellegrini at JOTB25](https://www.youtube.com/watch?v=AQ5fk4D3u9I)

---

## 📄 **License**

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.