# Batch Projection with Optimistic Locking

This document demonstrates how to use batch projection efficiently while maintaining consistency through optimistic locking with combined queries.

## Overview

The key insight is that when using multiple projectors for decision making, the `AppendCondition` should use a combined query that includes all projector queries. This ensures:

1. **Efficiency**: Single database query for all projectors
2. **Consistency**: Optimistic locking covers all relevant events
3. **Simplicity**: No need for complex transactions

## Example: Course Enrollment System

Let's build a course enrollment system that needs to check both course and student states:

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/your-org/go-crablet/pkg/dcb"
)

// CourseEnrollmentService handles course enrollment with batch projection
type CourseEnrollmentService struct {
	store dcb.EventStore
}

// EnrollStudent enrolls a student in a course using batch projection
func (s *CourseEnrollmentService) EnrollStudent(ctx context.Context, courseID, studentID string) error {
	// Define projectors for the decision model
	projectors := []dcb.BatchProjector{
		{
			ID: "course",
			StateProjector: dcb.StateProjector{
				Query: dcb.NewQuery(
					dcb.NewTags("course_id", courseID),
					"CourseLaunched", "CourseUpdated", "Enrollment",
				),
				InitialState: &CourseState{},
				TransitionFn: func(state any, e dcb.Event) any {
					course := state.(*CourseState)
					course.EventCount++

					var data map[string]string
					_ = json.Unmarshal(e.Data, &data)

					switch e.Type {
					case "CourseLaunched", "CourseUpdated":
						course.Title = data["title"]
					case "Enrollment":
						course.EnrollmentCount++
					}
					return course
				},
			},
		},
		{
			ID: "student",
			StateProjector: dcb.StateProjector{
				Query: dcb.NewQuery(
					dcb.NewTags("student_id", studentID),
					"StudentRegistered", "Enrollment",
				),
				InitialState: &StudentState{CourseIDs: make(map[string]bool)},
				TransitionFn: func(state any, e dcb.Event) any {
					student := state.(*StudentState)
					student.EventCount++

					var data map[string]string
					_ = json.Unmarshal(e.Data, &data)

					switch e.Type {
					case "StudentRegistered":
						student.Name = data["name"]
					case "Enrollment":
						for _, tag := range e.Tags {
							if tag.Key == "course_id" {
								student.CourseIDs[tag.Value] = true
							}
						}
					}
					return student
				},
			},
		},
	}

	// Project all states in a single database query
	result, err := s.store.ProjectBatch(ctx, projectors)
	if err != nil {
		return fmt.Errorf("failed to project states: %w", err)
	}

	// Extract states
	courseState := result.States["course"].(*CourseState)
	studentState := result.States["student"].(*StudentState)

	// Business logic validation
	if courseState.EnrollmentCount >= maxStudentsPerCourse {
		return fmt.Errorf("course %q is full", courseID)
	}

	if len(studentState.CourseIDs) >= maxCoursesPerStudent {
		return fmt.Errorf("student %q has reached course limit", studentID)
	}

	// Create enrollment event
	enrollmentEvent := dcb.NewInputEvent(
		"Enrollment",
		dcb.NewTags("course_id", courseID, "student_id", studentID),
		[]byte(`{"status":"active"}`),
	)

	// Create combined query for optimistic locking
	combinedQuery := dcb.CombineProjectorQueries(projectors)
	
	// Create append condition using the combined query
	condition := dcb.AppendCondition{
		FailIfEventsMatch: combinedQuery,
		After:             &result.Position, // Only check for new events after our projection
	}

	// Append the event with optimistic locking
	_, err = s.store.AppendEventsIf(ctx, []dcb.InputEvent{enrollmentEvent}, condition)
	if err != nil {
		return fmt.Errorf("failed to append enrollment event: %w", err)
	}

	return nil
}

// CourseState represents the state of a course
type CourseState struct {
	Title           string
	EnrollmentCount int
	EventCount      int
}

// StudentState represents the state of a student
type StudentState struct {
	Name       string
	CourseIDs  map[string]bool
	EventCount int
}

const (
	maxStudentsPerCourse = 50
	maxCoursesPerStudent = 10
)
```

## Key Benefits

### 1. **Single Database Query for Projection**
Instead of N separate queries for N projectors, we use one combined query:

```go
// Efficient: One query for all projectors
result, err := store.ProjectBatch(ctx, projectors)

// Instead of N separate queries:
// _, courseState, err := store.ProjectState(ctx, courseProjector)
// _, studentState, err := store.ProjectState(ctx, studentProjector)
// _, otherState, err := store.ProjectState(ctx, otherProjector)
```

### 2. **Consistent Optimistic Locking**
The `AppendCondition` uses the same combined query logic:

```go
// Combined query ensures all relevant events are considered
combinedQuery := dcb.CombineProjectorQueries(projectors)
condition := dcb.AppendCondition{
	FailIfEventsMatch: combinedQuery,
	After:             &result.Position,
}
```

### 3. **Event Routing**
Each event is automatically routed to the appropriate projectors:

```go
// Event with course_id tag goes to course projector
// Event with student_id tag goes to student projector  
// Event with both tags goes to both projectors
```

## Performance Comparison

### Before (N queries):
```
1. Query for course events
2. Query for student events  
3. Query for other events
4. Append with separate condition
Total: N+1 database round trips
```

### After (Combined approach):
```
1. Single combined query for all projectors
2. Append with combined condition
Total: 2 database round trips
```

## Advanced Usage: Complex Decision Models

For more complex scenarios with multiple decision models:

```go
// Multiple decision models can share projectors
courseProjectors := []dcb.BatchProjector{...}
studentProjectors := []dcb.BatchProjector{...}

// Combine all projectors for comprehensive locking
allProjectors := append(courseProjectors, studentProjectors...)
combinedQuery := dcb.CombineProjectorQueries(allProjectors)

// Use in append condition
condition := dcb.AppendCondition{
	FailIfEventsMatch: combinedQuery,
	After:             &result.Position,
}
```

## Best Practices

1. **Use descriptive projector IDs**: Makes debugging easier
2. **Keep projectors focused**: Each projector should handle a specific concern
3. **Combine queries consistently**: Always use `CombineProjectorQueries` for `AppendCondition`
4. **Handle position correctly**: Use the position from batch projection in the append condition
5. **Validate business rules**: Check constraints after projection, before append

This approach provides the efficiency of batch operations while maintaining the consistency guarantees of optimistic locking. 