// This example is standalone. Run with: go run examples/enrollment/main.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/rodolfodpk/go-crablet/internal/examples/utils"
	"github.com/rodolfodpk/go-crablet/pkg/dcb"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Command types
type CreateCourseCommand struct {
	CourseID    string
	Title       string
	MaxStudents int
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

type UnenrollStudentCommand struct {
	StudentID string
	CourseID  string
}

// Helper functions for flatter code structure

// setupDatabase initializes the database connection and event store
func setupDatabase(ctx context.Context) (*pgxpool.Pool, dcb.EventStore, error) {
	pool, err := pgxpool.New(ctx, "postgres://crablet:crablet@localhost:5432/crablet?sslmode=disable")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to db: %w", err)
	}

	store, err := dcb.NewEventStore(ctx, pool)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create event store: %w", err)
	}

	// Truncate events table before running the example
	_, err = pool.Exec(ctx, "TRUNCATE TABLE events RESTART IDENTITY CASCADE")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to truncate tables: %w", err)
	}

	return pool, store, nil
}

// executeCommand executes a command and handles errors with early returns
func executeCommand[T any](ctx context.Context, store dcb.EventStore, cmd T, handler func(context.Context, dcb.EventStore, T) error, description string) error {
	if err := handler(ctx, store, cmd); err != nil {
		return fmt.Errorf("%s failed: %w", description, err)
	}
	return nil
}
func main() {
	ctx := context.Background()

	// Setup database
	pool, store, err := setupDatabase(ctx)
	if err != nil {
		log.Fatalf("Setup failed: %v", err)
	}
	defer pool.Close()

	// Execute commands with early returns for failures
	fmt.Println("=== Command 1: Create Course ===")
	createCourseCmd := CreateCourseCommand{
		CourseID:    "course101",
		Title:       "Introduction to Event Sourcing",
		MaxStudents: 25,
	}
	if err := executeCommand(ctx, store, createCourseCmd, handleCreateCourse, "Create course"); err != nil {
		log.Fatalf("%v", err)
	}

	fmt.Println("=== Command 2: Register Student ===")
	registerStudentCmd := RegisterStudentCommand{
		StudentID: "student42",
		Name:      "Alice Johnson",
		Email:     "alice@example.com",
	}
	if err := executeCommand(ctx, store, registerStudentCmd, handleRegisterStudent, "Register student"); err != nil {
		log.Fatalf("%v", err)
	}

	fmt.Println("=== Command 3: Enroll Student in Course ===")
	enrollCmd := EnrollStudentCommand{
		StudentID: "student42",
		CourseID:  "course101",
	}
	if err := executeCommand(ctx, store, enrollCmd, handleEnrollStudent, "Enroll student"); err != nil {
		log.Fatalf("%v", err)
	}

	fmt.Println("=== Command 4: Register another student ===")
	registerStudent2Cmd := RegisterStudentCommand{
		StudentID: "student43",
		Name:      "Bob Smith",
		Email:     "bob@example.com",
	}
	if err := executeCommand(ctx, store, registerStudent2Cmd, handleRegisterStudent, "Register student 2"); err != nil {
		log.Fatalf("%v", err)
	}

	fmt.Println("=== Command 5: Enroll second student ===")
	enroll2Cmd := EnrollStudentCommand{
		StudentID: "student43",
		CourseID:  "course101",
	}
	if err := executeCommand(ctx, store, enroll2Cmd, handleEnrollStudent, "Enroll student 2"); err != nil {
		log.Fatalf("%v", err)
	}

	fmt.Println("=== Command 6: Unenroll first student ===")
	unenrollCmd := UnenrollStudentCommand{
		StudentID: "student42",
		CourseID:  "course101",
	}
	if err := executeCommand(ctx, store, unenrollCmd, handleUnenrollStudent, "Unenroll student"); err != nil {
		log.Fatalf("%v", err)
	}

	fmt.Println("All enrollment commands executed successfully!")

	// Dump all events to show what was created
	fmt.Println("\n=== Events in Database ===")
	utils.DumpEvents(ctx, pool)
}

// Command handlers with their own business rules

func handleCreateCourse(ctx context.Context, store dcb.EventStore, cmd CreateCourseCommand) error {
	// Command-specific projectors
	projectors := []dcb.StateProjector{
		{
			ID: "courseExists",
			Query: dcb.NewQuery(
				dcb.NewTags("course_id", cmd.CourseID),
				"CourseDefined",
			),
			InitialState: false,
			TransitionFn: func(state any, event dcb.Event) any {
				return true // If we see a CourseDefined event, course exists
			},
		},
	}

	states, _, err := store.Project(ctx, projectors, nil)
	if err != nil {
		return fmt.Errorf("failed to check course existence: %w", err)
	}

	// Command-specific business rule: course must not already exist
	if states["courseExists"].(bool) {
		return fmt.Errorf("course %s already exists", cmd.CourseID)
	}

	// Create events for this command using EventBuilder
	events := []dcb.InputEvent{
		dcb.NewEvent("CourseDefined").
			WithTag("course_id", cmd.CourseID).
			WithData(map[string]any{
				"Title":       cmd.Title,
				"MaxStudents": cmd.MaxStudents,
			}).
			Build(),
	}

	// Append events atomically for this command
	err = store.Append(ctx, events)
	if err != nil {
		return fmt.Errorf("failed to create course: %w", err)
	}

	fmt.Printf("Created course %s (%s) with max students %d\n", cmd.CourseID, cmd.Title, cmd.MaxStudents)
	return nil
}

func handleRegisterStudent(ctx context.Context, store dcb.EventStore, cmd RegisterStudentCommand) error {
	// Command-specific projectors
	projectors := []dcb.StateProjector{
		{
			ID: "studentExists",
			Query: dcb.NewQuery(
				dcb.NewTags("student_id", cmd.StudentID),
				"StudentRegistered",
			),
			InitialState: false,
			TransitionFn: func(state any, event dcb.Event) any {
				return true // If we see a StudentRegistered event, student exists
			},
		},
		{
			ID: "emailExists",
			Query: dcb.NewQuery(
				dcb.NewTags("email", cmd.Email),
				"StudentRegistered",
			),
			InitialState: false,
			TransitionFn: func(state any, event dcb.Event) any {
				return true // If we see a StudentRegistered event with this email, email exists
			},
		},
	}

	states, _, err := store.Project(ctx, projectors, nil)
	if err != nil {
		return fmt.Errorf("failed to check student existence: %w", err)
	}

	// Command-specific business rules
	if states["studentExists"].(bool) {
		return fmt.Errorf("student %s already exists", cmd.StudentID)
	}
	if states["emailExists"].(bool) {
		return fmt.Errorf("email %s already exists", cmd.Email)
	}

	// Create events for this command using EventBuilder
	events := []dcb.InputEvent{
		dcb.NewEvent("StudentRegistered").
			WithTag("student_id", cmd.StudentID).
			WithTag("email", cmd.Email).
			WithData(map[string]any{
				"Name":  cmd.Name,
				"Email": cmd.Email,
			}).
			Build(),
	}

	// Append events atomically for this command
	err = store.Append(ctx, events)
	if err != nil {
		return fmt.Errorf("failed to register student: %w", err)
	}

	fmt.Printf("Registered student %s (%s)\n", cmd.Name, cmd.Email)
	return nil
}

func handleEnrollStudent(ctx context.Context, store dcb.EventStore, cmd EnrollStudentCommand) error {
	// Command-specific projectors
	courseProjector := dcb.StateProjector{
		Query: dcb.NewQuery(
			dcb.NewTags("course_id", cmd.CourseID),
			"CourseDefined", "StudentEnrolled", "StudentUnenrolled",
		),
		InitialState: &CourseState{MaxStudents: 30},
		TransitionFn: func(state any, event dcb.Event) any {
			course := state.(*CourseState)
			switch event.Type {
			case "CourseDefined":
				var data struct {
					Title       string
					MaxStudents int
				}
				if err := json.Unmarshal(event.Data, &data); err == nil {
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
		TransitionFn: func(state any, event dcb.Event) any {
			student := state.(*StudentState)
			switch event.Type {
			case "StudentRegistered":
				var data struct{ Name, Email string }
				if err := json.Unmarshal(event.Data, &data); err == nil {
					student.Name = data.Name
					student.Email = data.Email
				}
			case "StudentEnrolled":
				var data struct{ CourseID string }
				if err := json.Unmarshal(event.Data, &data); err == nil {
					student.CourseIDs[data.CourseID] = true
				}
			case "StudentUnenrolled":
				var data struct{ CourseID string }
				if err := json.Unmarshal(event.Data, &data); err == nil {
					delete(student.CourseIDs, data.CourseID)
				}
			}
			return student
		},
	}

	enrollmentProjector := dcb.StateProjector{
		Query: dcb.NewQuery(
			dcb.NewTags("course_id", cmd.CourseID, "student_id", cmd.StudentID),
			"StudentEnrolled",
		),
		InitialState: false,
		TransitionFn: func(state any, event dcb.Event) any {
			return true // If we see a StudentEnrolled event, enrollment exists
		},
	}

	// Project all states using the DCB decision model pattern
	states, appendCond, err := store.Project(ctx, []dcb.StateProjector{
		{
			ID:           "course",
			Query:        courseProjector.Query,
			InitialState: courseProjector.InitialState,
			TransitionFn: courseProjector.TransitionFn,
		},
		{
			ID:           "student",
			Query:        studentProjector.Query,
			InitialState: studentProjector.InitialState,
			TransitionFn: studentProjector.TransitionFn,
		},
		{
			ID:           "enrollmentExists",
			Query:        enrollmentProjector.Query,
			InitialState: enrollmentProjector.InitialState,
			TransitionFn: enrollmentProjector.TransitionFn,
		},
	}, nil)
	if err != nil {
		return fmt.Errorf("projection failed: %w", err)
	}

	course := states["course"].(*CourseState)
	student := states["student"].(*StudentState)
	enrollmentExists := states["enrollmentExists"].(bool)

	// Command-specific business rules
	if course.Title == "" {
		return fmt.Errorf("course %s does not exist", cmd.CourseID)
	}
	if student.Name == "" {
		return fmt.Errorf("student %s does not exist", cmd.StudentID)
	}
	if enrollmentExists {
		return fmt.Errorf("student %s is already enrolled in course %s", cmd.StudentID, cmd.CourseID)
	}
	if course.EnrolledStudents >= course.MaxStudents {
		return fmt.Errorf("course %s is full (capacity: %d, enrolled: %d)", cmd.CourseID, course.MaxStudents, course.EnrolledStudents)
	}
	if len(student.CourseIDs) >= 10 {
		return fmt.Errorf("student %s is already enrolled in 10 courses", cmd.StudentID)
	}

	// Create events for this command using EventBuilder
	events := []dcb.InputEvent{
		dcb.NewEvent("StudentEnrolled").
			WithTag("course_id", cmd.CourseID).
			WithTag("student_id", cmd.StudentID).
			WithData(map[string]any{"CourseID": cmd.CourseID, "StudentID": cmd.StudentID}).
			Build(),
	}

	// Append events atomically for this command with optimistic concurrency control
	err = store.AppendIf(ctx, events, appendCond)
	if err != nil {
		return fmt.Errorf("failed to enroll student: %w", err)
	}

	fmt.Printf("Enrolled student %s in course %s\n", cmd.StudentID, cmd.CourseID)
	return nil
}

func handleUnenrollStudent(ctx context.Context, store dcb.EventStore, cmd UnenrollStudentCommand) error {
	// Command-specific projectors
	enrollmentProjector := dcb.StateProjector{
		Query: dcb.NewQuery(
			dcb.NewTags("course_id", cmd.CourseID, "student_id", cmd.StudentID),
			"StudentEnrolled", "StudentUnenrolled",
		),
		InitialState: &EnrollmentState{Enrolled: false},
		TransitionFn: func(state any, event dcb.Event) any {
			enrollment := state.(*EnrollmentState)
			switch event.Type {
			case "StudentEnrolled":
				enrollment.Enrolled = true
			case "StudentUnenrolled":
				enrollment.Enrolled = false
			}
			return enrollment
		},
	}

	// Project enrollment state
	states, appendCond, err := store.Project(ctx, []dcb.StateProjector{
		{
			ID:           "enrollment",
			Query:        enrollmentProjector.Query,
			InitialState: enrollmentProjector.InitialState,
			TransitionFn: enrollmentProjector.TransitionFn,
		},
	}, nil)
	if err != nil {
		return fmt.Errorf("projection failed: %w", err)
	}

	enrollment := states["enrollment"].(*EnrollmentState)

	// Command-specific business rule: student must be enrolled
	if !enrollment.Enrolled {
		return fmt.Errorf("student %s is not enrolled in course %s", cmd.StudentID, cmd.CourseID)
	}

	// Create events for this command using EventBuilder
	events := []dcb.InputEvent{
		dcb.NewEvent("StudentUnenrolled").
			WithTag("course_id", cmd.CourseID).
			WithTag("student_id", cmd.StudentID).
			WithData(map[string]any{"CourseID": cmd.CourseID, "StudentID": cmd.StudentID}).
			Build(),
	}

	// Append events atomically for this command with optimistic concurrency control
	err = store.AppendIf(ctx, events, appendCond)
	if err != nil {
		return fmt.Errorf("failed to unenroll student: %w", err)
	}

	fmt.Printf("Unenrolled student %s from course %s\n", cmd.StudentID, cmd.CourseID)
	return nil
}

// Helper types
type CourseState struct {
	Title            string
	MaxStudents      int
	EnrolledStudents int
}

type StudentState struct {
	Name      string
	Email     string
	CourseIDs map[string]bool
}

type EnrollmentState struct {
	Enrolled bool
}

func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}
