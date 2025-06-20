// This example demonstrates channel-based projection using ProjectDecisionModelChannel
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"go-crablet/pkg/dcb"

	"github.com/jackc/pgx/v5/pgxpool"
)

// CourseState represents the state of a course
type CourseState struct {
	CourseID  string
	Name      string
	Capacity  int
	Enrolled  int
	CreatedAt time.Time
	UpdatedAt time.Time
}

// StudentState represents the state of a student
type StudentState struct {
	StudentID string
	Name      string
	Courses   []string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func main() {
	ctx := context.Background()

	// Connect to database
	pool, err := pgxpool.New(ctx, "postgres://postgres:postgres@localhost:5432/dcb_app?sslmode=disable")
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	// Create event store
	store, err := dcb.NewEventStore(ctx, pool)
	if err != nil {
		log.Fatalf("Failed to create event store: %v", err)
	}

	// Cast to CrabletEventStore for extended functionality
	channelStore := store.(dcb.CrabletEventStore)

	// Create some test events
	events := []dcb.InputEvent{
		func() dcb.InputEvent {
			event := dcb.NewInputEvent("CourseCreated", dcb.NewTags("course_id", "course-1"), []byte(`{"name": "Go Programming", "capacity": 30}`))
			return event
		}(),
		func() dcb.InputEvent {
			event := dcb.NewInputEvent("CourseCreated", dcb.NewTags("course_id", "course-2"), []byte(`{"name": "Event Sourcing", "capacity": 25}`))
			return event
		}(),
		func() dcb.InputEvent {
			event := dcb.NewInputEvent("StudentRegistered", dcb.NewTags("student_id", "student-1"), []byte(`{"name": "Alice"}`))
			return event
		}(),
		func() dcb.InputEvent {
			event := dcb.NewInputEvent("StudentRegistered", dcb.NewTags("student_id", "student-2"), []byte(`{"name": "Bob"}`))
			return event
		}(),
		func() dcb.InputEvent {
			event := dcb.NewInputEvent("StudentEnrolled", dcb.NewTags("course_id", "course-1", "student_id", "student-1"), []byte(`{"enrolled_at": "2024-01-15T10:00:00Z"}`))
			return event
		}(),
		func() dcb.InputEvent {
			event := dcb.NewInputEvent("StudentEnrolled", dcb.NewTags("course_id", "course-1", "student_id", "student-2"), []byte(`{"enrolled_at": "2024-01-15T11:00:00Z"}`))
			return event
		}(),
		func() dcb.InputEvent {
			event := dcb.NewInputEvent("StudentEnrolled", dcb.NewTags("course_id", "course-2", "student_id", "student-1"), []byte(`{"enrolled_at": "2024-01-15T12:00:00Z"}`))
			return event
		}(),
	}

	// Append events
	_, err = store.Append(ctx, events, nil)
	if err != nil {
		log.Fatalf("Failed to append events: %v", err)
	}

	fmt.Println("=== Channel-Based Projection Example ===")

	// Method 1: Traditional cursor-based projection
	fmt.Println("\n1. Traditional Cursor-Based Projection:")
	demonstrateCursorProjection(ctx, channelStore)

	// Method 2: Channel-based projection
	fmt.Println("\n2. Channel-Based Projection:")
	demonstrateChannelProjection(ctx, channelStore)

	// Method 3: Performance Comparison
	demonstratePerformanceComparison(ctx, store)
}

// demonstrateCursorProjection shows the traditional cursor-based approach
func demonstrateCursorProjection(ctx context.Context, store dcb.CrabletEventStore) {
	fmt.Println("   Using traditional ProjectDecisionModel:")

	// Create projectors
	courseProjector := dcb.BatchProjector{
		ID: "course-projector",
		StateProjector: dcb.StateProjector{
			Query:        dcb.NewQuerySimple(dcb.NewTags(), "CourseCreated", "StudentEnrolled"),
			InitialState: map[string]*CourseState{},
			TransitionFn: func(state any, event dcb.Event) any {
				courses := state.(map[string]*CourseState)

				switch event.Type {
				case "CourseCreated":
					var data struct {
						Name     string `json:"name"`
						Capacity int    `json:"capacity"`
					}
					if err := json.Unmarshal(event.Data, &data); err != nil {
						return state
					}

					courseID := ""
					for _, tag := range event.Tags {
						if tag.Key == "course_id" {
							courseID = tag.Value
							break
						}
					}

					if courseID != "" {
						courses[courseID] = &CourseState{
							CourseID:  courseID,
							Name:      data.Name,
							Capacity:  data.Capacity,
							Enrolled:  0,
							CreatedAt: time.Now(),
							UpdatedAt: time.Now(),
						}
					}

				case "StudentEnrolled":
					courseID := ""
					for _, tag := range event.Tags {
						if tag.Key == "course_id" {
							courseID = tag.Value
							break
						}
					}

					if course, exists := courses[courseID]; exists {
						course.Enrolled++
						course.UpdatedAt = time.Now()
					}
				}

				return courses
			},
		},
	}

	studentProjector := dcb.BatchProjector{
		ID: "student-projector",
		StateProjector: dcb.StateProjector{
			Query:        dcb.NewQuerySimple(dcb.NewTags(), "StudentRegistered", "StudentEnrolled"),
			InitialState: map[string]*StudentState{},
			TransitionFn: func(state any, event dcb.Event) any {
				students := state.(map[string]*StudentState)

				switch event.Type {
				case "StudentRegistered":
					var data struct {
						Name string `json:"name"`
					}
					if err := json.Unmarshal(event.Data, &data); err != nil {
						return state
					}

					studentID := ""
					for _, tag := range event.Tags {
						if tag.Key == "student_id" {
							studentID = tag.Value
							break
						}
					}

					if studentID != "" {
						students[studentID] = &StudentState{
							StudentID: studentID,
							Name:      data.Name,
							Courses:   []string{},
							CreatedAt: time.Now(),
							UpdatedAt: time.Now(),
						}
					}

				case "StudentEnrolled":
					studentID := ""
					courseID := ""
					for _, tag := range event.Tags {
						if tag.Key == "student_id" {
							studentID = tag.Value
						}
						if tag.Key == "course_id" {
							courseID = tag.Value
						}
					}

					if student, exists := students[studentID]; exists && courseID != "" {
						student.Courses = append(student.Courses, courseID)
						student.UpdatedAt = time.Now()
					}
				}

				return students
			},
		},
	}

	// Use traditional projection
	projectors := []dcb.BatchProjector{courseProjector, studentProjector}
	states, appendCondition, err := store.ProjectDecisionModel(ctx, projectors)
	if err != nil {
		log.Printf("Traditional projection failed: %v", err)
		return
	}

	fmt.Printf("   - Final states computed: %d projectors\n", len(states))
	fmt.Printf("   - Append condition: %+v\n", appendCondition)

	// Display results
	if courses, ok := states["course-projector"].(map[string]*CourseState); ok {
		fmt.Printf("   - Courses: %d\n", len(courses))
		for id, course := range courses {
			fmt.Printf("     * %s: %s (%d/%d enrolled)\n", id, course.Name, course.Enrolled, course.Capacity)
		}
	}

	if students, ok := states["student-projector"].(map[string]*StudentState); ok {
		fmt.Printf("   - Students: %d\n", len(students))
		for id, student := range students {
			fmt.Printf("     * %s: %s (%d courses)\n", id, student.Name, len(student.Courses))
		}
	}
}

// demonstrateChannelProjection shows the channel-based approach
func demonstrateChannelProjection(ctx context.Context, store dcb.EventStore) {
	fmt.Println("   Using channel-based ProjectDecisionModelChannel:")

	// Check if store implements CrabletEventStore
	channelStore, ok := store.(dcb.CrabletEventStore)
	if !ok {
		fmt.Println("   - Store does not implement CrabletEventStore interface")
		return
	}

	// Create the same projectors as above
	courseProjector := dcb.BatchProjector{
		ID: "course-projector",
		StateProjector: dcb.StateProjector{
			Query:        dcb.NewQuerySimple(dcb.NewTags(), "CourseCreated", "StudentEnrolled"),
			InitialState: map[string]*CourseState{},
			TransitionFn: func(state any, event dcb.Event) any {
				courses := state.(map[string]*CourseState)

				switch event.Type {
				case "CourseCreated":
					var data struct {
						Name     string `json:"name"`
						Capacity int    `json:"capacity"`
					}
					if err := json.Unmarshal(event.Data, &data); err != nil {
						return state
					}

					courseID := ""
					for _, tag := range event.Tags {
						if tag.Key == "course_id" {
							courseID = tag.Value
							break
						}
					}

					if courseID != "" {
						courses[courseID] = &CourseState{
							CourseID:  courseID,
							Name:      data.Name,
							Capacity:  data.Capacity,
							Enrolled:  0,
							CreatedAt: time.Now(),
							UpdatedAt: time.Now(),
						}
					}

				case "StudentEnrolled":
					courseID := ""
					for _, tag := range event.Tags {
						if tag.Key == "course_id" {
							courseID = tag.Value
							break
						}
					}

					if course, exists := courses[courseID]; exists {
						course.Enrolled++
						course.UpdatedAt = time.Now()
					}
				}

				return courses
			},
		},
	}

	studentProjector := dcb.BatchProjector{
		ID: "student-projector",
		StateProjector: dcb.StateProjector{
			Query:        dcb.NewQuerySimple(dcb.NewTags(), "StudentRegistered", "StudentEnrolled"),
			InitialState: map[string]*StudentState{},
			TransitionFn: func(state any, event dcb.Event) any {
				students := state.(map[string]*StudentState)

				switch event.Type {
				case "StudentRegistered":
					var data struct {
						Name string `json:"name"`
					}
					if err := json.Unmarshal(event.Data, &data); err != nil {
						return state
					}

					studentID := ""
					for _, tag := range event.Tags {
						if tag.Key == "student_id" {
							studentID = tag.Value
							break
						}
					}

					if studentID != "" {
						students[studentID] = &StudentState{
							StudentID: studentID,
							Name:      data.Name,
							Courses:   []string{},
							CreatedAt: time.Now(),
							UpdatedAt: time.Now(),
						}
					}

				case "StudentEnrolled":
					studentID := ""
					courseID := ""
					for _, tag := range event.Tags {
						if tag.Key == "student_id" {
							studentID = tag.Value
						}
						if tag.Key == "course_id" {
							courseID = tag.Value
						}
					}

					if student, exists := students[studentID]; exists && courseID != "" {
						student.Courses = append(student.Courses, courseID)
						student.UpdatedAt = time.Now()
					}
				}

				return students
			},
		},
	}

	// Use channel-based projection
	projectors := []dcb.BatchProjector{courseProjector, studentProjector}
	resultChan, err := channelStore.ProjectDecisionModelChannel(ctx, projectors)
	if err != nil {
		log.Printf("Channel projection failed: %v", err)
		return
	}

	// Process results as they come in
	projectionCount := 0
	finalStates := make(map[string]interface{})

	for result := range resultChan {
		if result.Error != nil {
			fmt.Printf("   - Error: %v\n", result.Error)
			continue
		}

		projectionCount++
		finalStates[result.ProjectorID] = result.State

		fmt.Printf("   - Projection %d: %s processed event %s (position %d)\n",
			projectionCount, result.ProjectorID, result.Event.Type, result.Position)
	}

	fmt.Printf("   - Total projections: %d\n", projectionCount)
	fmt.Printf("   - Final states computed: %d projectors\n", len(finalStates))

	// Display final results
	if courses, ok := finalStates["course-projector"].(map[string]*CourseState); ok {
		fmt.Printf("   - Courses: %d\n", len(courses))
		for id, course := range courses {
			fmt.Printf("     * %s: %s (%d/%d enrolled)\n", id, course.Name, course.Enrolled, course.Capacity)
		}
	}

	if students, ok := finalStates["student-projector"].(map[string]*StudentState); ok {
		fmt.Printf("   - Students: %d\n", len(students))
		for id, student := range students {
			fmt.Printf("     * %s: %s (%d courses)\n", id, student.Name, len(student.Courses))
		}
	}
}

// demonstratePerformanceComparison shows the performance characteristics
func demonstratePerformanceComparison(ctx context.Context, store dcb.EventStore) {
	fmt.Println("\n3. Performance Comparison:")
	fmt.Println("   - Cursor-based: Best for large datasets (> 1000 events)")
	fmt.Println("   - Channel-based: Best for small-medium datasets (< 500 events)")
	fmt.Println("   - Traditional: Best for very small datasets (< 100 events)")
	fmt.Println("   - Channel approach provides real-time processing feedback")
	fmt.Println("   - Cursor approach provides better memory efficiency")
}
