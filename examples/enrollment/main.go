// This example is standalone. Run with: go run examples/projection_based_enrollment_example.go
package enrollment

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

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
	pool, err := pgxpool.New(ctx, "postgres://user:password@localhost:5432/db")
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

	// Project both states in a single query
	result, err := store.ProjectBatch(ctx, []dcb.BatchProjector{
		{ID: "course", StateProjector: courseProjector},
		{ID: "student", StateProjector: studentProjector},
	})
	if err != nil {
		log.Fatalf("projection failed: %v", err)
	}
	course := result.States["course"].(*CourseState)
	student := result.States["student"].(*StudentState)

	// Business rules
	if course.EnrolledStudents >= course.MaxStudents {
		log.Fatalf("course is full")
	}
	if len(student.CourseIDs) >= 10 {
		log.Fatalf("student is already enrolled in 10 courses")
	}

	// Prepare events
	enrollEvent := dcb.InputEvent{
		Type: "StudentEnrolled",
		Tags: dcb.NewTags(
			"course_id", cmd.CourseID,
			"student_id", cmd.StudentID,
		),
		Data: mustJSON(map[string]any{"CourseID": cmd.CourseID, "StudentID": cmd.StudentID}),
	}

	// Append event with optimistic locking
	_, err = store.Append(ctx, []dcb.InputEvent{enrollEvent}, &dcb.AppendCondition{
		FailIfEventsMatch: dcb.NewQuery(dcb.NewTags("course_id", cmd.CourseID)),
		After:             &result.Position,
	})
	if err != nil {
		log.Fatalf("append failed: %v", err)
	}

	fmt.Println("Student enrolled successfully!")
}

func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}
