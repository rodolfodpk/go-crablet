// This example is standalone. Run with: go run examples/enrollment/main.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"go-crablet/internal/examples/utils"
	"go-crablet/pkg/dcb"

	"github.com/jackc/pgx/v5/pgxpool"
)

// CourseState holds the state for a course
type CourseState struct {
	Title            string
	MaxStudents      int
	EnrolledStudents int
}

// StudentState holds the state for a student
type StudentState struct {
	Name      string
	Email     string
	CourseIDs map[string]bool
}

func main() {
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, "postgres://postgres:postgres@localhost:5432/dcb_app?sslmode=disable")
	if err != nil {
		log.Fatalf("failed to connect to db: %v", err)
	}
	store, err := dcb.NewEventStore(ctx, pool)
	if err != nil {
		log.Fatalf("failed to create event store: %v", err)
	}

	cmd := struct {
		CourseID  string
		StudentID string
	}{CourseID: "course101", StudentID: "student42"}

	// Define projectors
	courseProjector := dcb.StateProjector{
		Query: dcb.NewQuery(
			dcb.NewTags("course_id", cmd.CourseID),
			"CourseCreated", "StudentEnrolled", "StudentUnenrolled",
		),
		InitialState: &CourseState{MaxStudents: 30},
		TransitionFn: func(state any, e dcb.Event) any {
			course := state.(*CourseState)
			switch e.Type {
			case "CourseCreated":
				var data struct {
					Title       string
					MaxStudents int
				}
				if err := json.Unmarshal(e.Data, &data); err == nil {
					course.Title = data.Title
					if data.MaxStudents > 0 {
						course.MaxStudents = data.MaxStudents
					}
				}
			case "StudentEnrolled":
				course.EnrolledStudents++
			case "StudentUnenrolled":
				if course.EnrolledStudents > 0 {
					course.EnrolledStudents--
				}
			}
			return course
		},
	}
	studentProjector := dcb.StateProjector{
		Query: dcb.NewQuery(
			dcb.NewTags("student_id", cmd.StudentID),
			"StudentRegistered", "StudentEnrolled", "StudentUnenrolled",
		),
		InitialState: &StudentState{CourseIDs: make(map[string]bool)},
		TransitionFn: func(state any, e dcb.Event) any {
			student := state.(*StudentState)
			switch e.Type {
			case "StudentRegistered":
				var data struct{ Name, Email string }
				if err := json.Unmarshal(e.Data, &data); err == nil {
					student.Name = data.Name
					student.Email = data.Email
				}
			case "StudentEnrolled":
				var data struct{ CourseID string }
				if err := json.Unmarshal(e.Data, &data); err == nil {
					student.CourseIDs[data.CourseID] = true
				}
			case "StudentUnenrolled":
				var data struct{ CourseID string }
				if err := json.Unmarshal(e.Data, &data); err == nil {
					delete(student.CourseIDs, data.CourseID)
				}
			}
			return student
		},
	}

	// Project both states using the DCB decision model pattern
	states, appendCondition, err := store.ProjectDecisionModel(ctx, []dcb.BatchProjector{
		{ID: "course", StateProjector: courseProjector},
		{ID: "student", StateProjector: studentProjector},
	}, nil)
	if err != nil {
		log.Fatalf("projection failed: %v", err)
	}
	course := states["course"].(*CourseState)
	student := states["student"].(*StudentState)

	// Business rules
	if course.EnrolledStudents >= course.MaxStudents {
		log.Fatalf("course is full")
	}
	if len(student.CourseIDs) >= 10 {
		log.Fatalf("student is already enrolled in 10 courses")
	}

	// Create events for batch append based on business logic
	events := []dcb.InputEvent{}

	// Add course creation event if course doesn't exist
	if course.Title == "" {
		courseEvent := dcb.NewInputEvent(
			"CourseCreated",
			dcb.NewTags("course_id", cmd.CourseID),
			mustJSON(map[string]any{
				"Title":       "Introduction to Event Sourcing",
				"MaxStudents": 25,
			}),
		)
		events = append(events, courseEvent)
		fmt.Println("Adding course creation event to batch")
	}

	// Add student registration event if student doesn't exist
	if student.Name == "" {
		studentEvent := dcb.NewInputEvent(
			"StudentRegistered",
			dcb.NewTags("student_id", cmd.StudentID),
			mustJSON(map[string]any{
				"Name":  "Alice Johnson",
				"Email": "alice@example.com",
			}),
		)
		events = append(events, studentEvent)
		fmt.Println("Adding student registration event to batch")
	}

	// Add enrollment event
	enrollEvent := dcb.NewInputEvent(
		"StudentEnrolled",
		dcb.NewTags(
			"course_id", cmd.CourseID,
			"student_id", cmd.StudentID,
		),
		mustJSON(map[string]any{"CourseID": cmd.CourseID, "StudentID": cmd.StudentID}),
	)
	events = append(events, enrollEvent)
	fmt.Println("Adding enrollment event to batch")

	// Use the append condition from the decision model for optimistic locking
	// All events (course creation + student registration + enrollment) are appended atomically
	position, err := store.Append(ctx, events, &appendCondition)
	if err != nil {
		log.Fatalf("append failed: %v", err)
	}

	fmt.Printf("Successfully appended %d events up to position: %d\n", len(events), position)
	fmt.Println("Student enrolled successfully!")

	// Dump all events to show what was created
	fmt.Println("\n=== Events in Database ===")
	utils.DumpEvents(ctx, pool)
}

func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}
