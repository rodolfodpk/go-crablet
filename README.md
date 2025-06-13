[![Go Report Card](https://goreportcard.com/badge/github.com/rodolfodpk/go-crablet)](https://goreportcard.com/report/github.com/rodolfodpk/go-crablet)
[![codecov](https://codecov.io/gh/rodolfodpk/go-crablet/branch/main/graph/badge.svg)](https://codecov.io/gh/rodolfodpk/go-crablet)
[![GoDoc](https://godoc.org/github.com/rodolfodpk/go-crablet?status.svg)](https://godoc.org/github.com/rodolfodpk/go-crablet)
[![License](https://img.shields.io/github/license/rodolfodpk/go-crablet)](https://github.com/rodolfodpk/go-crablet/blob/main/LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/rodolfodpk/go-crablet)](https://github.com/rodolfodpk/go-crablet/blob/main/go.mod)

# go-crablet

A Go library for event sourcing, exploring concepts inspired by Sara Pellegrini's Dynamic Consistency Boundary (DCB) pattern. go-crablet enables you to build event-driven systems with:

- **Batch projection**: Project multiple states using a single streamlined PostgreSQL query with native streaming
- **DCB-inspired consistency**: Optimistic locking using the same query combination scope for projection and append
- **Streaming**: Memory-efficient event processing for large event streams
- **Flexible queries**: Tag-based, OR-combined queries for cross-entity invariants

## Key Features

- **DCB-inspired decision models**: Project multiple states and build append conditions in one step
- **Single streamlined query**: Efficiently project all relevant states using PostgreSQL's native streaming via pgx
- **Optimistic concurrency**: Append events only if no conflicting events have appeared within the same query combination scope
- **Streaming**: Process events row-by-row, suitable for millions of events
- **PostgreSQL-backed**: Uses PostgreSQL for robust, concurrent event storage

## Exploring the DCB Pattern in Go

We're learning about the Dynamic Consistency Boundary (DCB) pattern by exploring how to:
- Define projections ("decision models") that provide the data business rules need
- Project all relevant state in a single query
- Build a combined append condition for optimistic locking
- Append new events only if all invariants still hold

## Minimal Example: Course Subscription

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "github.com/rodolfodpk/go-crablet/pkg/dcb"
    "github.com/jackc/pgx/v5/pgxpool"
)

type CourseDefined struct {
    CourseID string
    Capacity int
}

type StudentSubscribed struct {
    StudentID string
    CourseID  string
}

func main() {
    pool, _ := pgxpool.New(context.Background(), "postgres://user:pass@localhost/db")
    store, _ := dcb.NewEventStore(context.Background(), pool)

    // Projectors for DCB-inspired decision model
    projectors := []dcb.BatchProjector{
        {ID: "courseExists", StateProjector: dcb.StateProjector{
            Query: dcb.NewQuery(dcb.NewTags("course_id", "c1"), "CourseDefined"),
            InitialState: false,
            TransitionFn: func(state any, e dcb.Event) any { return true },
        }},
        {ID: "numSubscriptions", StateProjector: dcb.StateProjector{
            Query: dcb.NewQuery(dcb.NewTags("course_id", "c1"), "StudentSubscribed"),
            InitialState: 0,
            TransitionFn: func(state any, e dcb.Event) any { return state.(int) + 1 },
        }},
    }
    query := dcb.NewQueryFromItems(
        dcb.NewQueryItem([]string{"CourseDefined"}, dcb.NewTags("course_id", "c1")),
        dcb.NewQueryItem([]string{"StudentSubscribed"}, dcb.NewTags("course_id", "c1")),
    )
    states, appendCond, _ := store.ProjectDecisionModel(context.Background(), query, nil, projectors)
    if !states["courseExists"].(bool) {
        // Append CourseDefined event
        data, _ := json.Marshal(CourseDefined{"c1", 2})
        store.Append(context.Background(), []dcb.InputEvent{
            dcb.NewInputEvent("CourseDefined", dcb.NewTags("course_id", "c1"), data),
        }, &appendCond)
    }
    // Subscribe a student if not full
    if states["numSubscriptions"].(int) < 2 {
        data, _ := json.Marshal(StudentSubscribed{"s1", "c1"})
        store.Append(context.Background(), []dcb.InputEvent{
            dcb.NewInputEvent("StudentSubscribed", dcb.NewTags("student_id", "s1", "course_id", "c1"), data),
        }, &appendCond)
    }
}
```

## Documentation
- [Overview](docs/overview.md): DCB pattern exploration, batch projection, and streaming
- [Examples](docs/examples.md): DCB-inspired use cases
- [Implementation](docs/implementation.md): Technical details

## References

- [Dynamic Consistency Boundary (DCB)](https://dcb.events/) - A very good resource to understand the DCB pattern and its applications in event-driven systems
- [I am here to kill the aggregate](https://sara.event-thinking.io/2023/04/kill-aggregate-chapter-1-I-am-here-to-kill-the-aggregate.html) - Sara Pellegrini's blog post about moving beyond aggregates in event-driven systems
- [Kill Aggregate - Volume 2 - Sara Pellegrini at JOTB25](https://www.youtube.com/watch?v=AQ5fk4D3u9I) 