package setup

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/rodolfodpk/go-crablet/pkg/dcb"
)

// DatasetConfig defines the size and distribution of test data
type DatasetConfig struct {
	Courses     int
	Students    int
	Enrollments int
	Capacity    int
}

// DatasetSizes provides predefined dataset configurations
var DatasetSizes = map[string]DatasetConfig{
	"small": {
		Courses:     1_000,
		Students:    10_000,
		Enrollments: 50_000,
		Capacity:    100,
	},
	"medium": {
		Courses:     5_000,
		Students:    50_000,
		Enrollments: 250_000,
		Capacity:    100,
	},
	"large": {
		Courses:     10_000,
		Students:    100_000,
		Enrollments: 500_000,
		Capacity:    100,
	},
	"xlarge": {
		Courses:     20_000,
		Students:    200_000,
		Enrollments: 1_000_000,
		Capacity:    100,
	},
}

// CourseData represents a course in the test dataset
type CourseData struct {
	ID         string
	Name       string
	Instructor string
	Capacity   int
}

// StudentData represents a student in the test dataset
type StudentData struct {
	ID    string
	Name  string
	Email string
}

// EnrollmentData represents an enrollment in the test dataset
type EnrollmentData struct {
	StudentID  string
	CourseID   string
	EnrolledAt string
}

// Dataset represents the complete test dataset
type Dataset struct {
	Courses     []CourseData
	Students    []StudentData
	Enrollments []EnrollmentData
	Config      DatasetConfig
}

// GenerateDataset creates a complete test dataset
func GenerateDataset(config DatasetConfig) *Dataset {
	dataset := &Dataset{
		Config: config,
	}

	// Generate courses
	dataset.Courses = make([]CourseData, config.Courses)
	for i := 0; i < config.Courses; i++ {
		dataset.Courses[i] = CourseData{
			ID:         fmt.Sprintf("course-%d", i),
			Name:       fmt.Sprintf("Course %d", i),
			Instructor: fmt.Sprintf("Instructor %d", i),
			Capacity:   config.Capacity,
		}
	}

	// Generate students
	dataset.Students = make([]StudentData, config.Students)
	for i := 0; i < config.Students; i++ {
		dataset.Students[i] = StudentData{
			ID:    fmt.Sprintf("student-%d", i),
			Name:  fmt.Sprintf("Student %d", i),
			Email: fmt.Sprintf("student%d@example.com", i),
		}
	}

	// Generate enrollments
	dataset.Enrollments = make([]EnrollmentData, config.Enrollments)
	rand.Seed(time.Now().UnixNano())

	for i := 0; i < config.Enrollments; i++ {
		studentIdx := rand.Intn(config.Students)
		courseIdx := rand.Intn(config.Courses)

		dataset.Enrollments[i] = EnrollmentData{
			StudentID:  dataset.Students[studentIdx].ID,
			CourseID:   dataset.Courses[courseIdx].ID,
			EnrolledAt: "2024-01-01T00:00:00Z",
		}
	}

	return dataset
}

// LoadDatasetIntoStore loads the dataset into the event store
func LoadDatasetIntoStore(ctx context.Context, store dcb.EventStore, dataset *Dataset) error {
	fmt.Printf("Loading dataset: %d courses, %d students, %d enrollments\n",
		len(dataset.Courses), len(dataset.Students), len(dataset.Enrollments))

	// Load courses
	if err := loadCourses(ctx, store, dataset.Courses); err != nil {
		return fmt.Errorf("failed to load courses: %w", err)
	}

	// Load students
	if err := loadStudents(ctx, store, dataset.Students); err != nil {
		return fmt.Errorf("failed to load students: %w", err)
	}

	// Load enrollments
	if err := loadEnrollments(ctx, store, dataset.Enrollments); err != nil {
		return fmt.Errorf("failed to load enrollments: %w", err)
	}

	fmt.Println("Dataset loaded successfully")
	return nil
}

// loadCourses loads course events into the store
func loadCourses(ctx context.Context, store dcb.EventStore, courses []CourseData) error {
	const batchSize = 1000

	for i := 0; i < len(courses); i += batchSize {
		end := i + batchSize
		if end > len(courses) {
			end = len(courses)
		}

		batch := courses[i:end]
		events := make([]dcb.InputEvent, len(batch))

		for j, course := range batch {
			events[j] = dcb.NewInputEvent("CourseCreated",
				dcb.NewTags("course_id", course.ID),
				[]byte(fmt.Sprintf(`{"courseId": "%s", "name": "%s", "capacity": %d, "instructor": "%s"}`,
					course.ID, course.Name, course.Capacity, course.Instructor)))
		}

		_, err := store.Append(ctx, events, nil)
		if err != nil {
			return fmt.Errorf("failed to append course batch %d-%d: %w", i, end-1, err)
		}
	}

	return nil
}

// loadStudents loads student events into the store
func loadStudents(ctx context.Context, store dcb.EventStore, students []StudentData) error {
	const batchSize = 1000

	for i := 0; i < len(students); i += batchSize {
		end := i + batchSize
		if end > len(students) {
			end = len(students)
		}

		batch := students[i:end]
		events := make([]dcb.InputEvent, len(batch))

		for j, student := range batch {
			events[j] = dcb.NewInputEvent("StudentRegistered",
				dcb.NewTags("student_id", student.ID),
				[]byte(fmt.Sprintf(`{"studentId": "%s", "name": "%s", "email": "%s"}`,
					student.ID, student.Name, student.Email)))
		}

		_, err := store.Append(ctx, events, nil)
		if err != nil {
			return fmt.Errorf("failed to append student batch %d-%d: %w", i, end-1, err)
		}
	}

	return nil
}

// loadEnrollments loads enrollment events into the store
func loadEnrollments(ctx context.Context, store dcb.EventStore, enrollments []EnrollmentData) error {
	const batchSize = 1000

	for i := 0; i < len(enrollments); i += batchSize {
		end := i + batchSize
		if end > len(enrollments) {
			end = len(enrollments)
		}

		batch := enrollments[i:end]
		events := make([]dcb.InputEvent, len(batch))

		for j, enrollment := range batch {
			events[j] = dcb.NewInputEvent("StudentEnrolledInCourse",
				dcb.NewTags("student_id", enrollment.StudentID, "course_id", enrollment.CourseID),
				[]byte(fmt.Sprintf(`{"studentId": "%s", "courseId": "%s", "enrolledAt": "%s"}`,
					enrollment.StudentID, enrollment.CourseID, enrollment.EnrolledAt)))
		}

		_, err := store.Append(ctx, events, nil)
		if err != nil {
			return fmt.Errorf("failed to append enrollment batch %d-%d: %w", i, end-1, err)
		}
	}

	return nil
}

// GenerateRandomQueries creates random queries for testing
func GenerateRandomQueries(dataset *Dataset, count int) []dcb.Query {
	queries := make([]dcb.Query, count)
	rand.Seed(time.Now().UnixNano())

	for i := 0; i < count; i++ {
		queryType := rand.Intn(4)

		switch queryType {
		case 0:
			// Random course query
			courseIdx := rand.Intn(len(dataset.Courses))
			queries[i] = dcb.NewQuery(dcb.NewTags("course_id", dataset.Courses[courseIdx].ID), "CourseCreated")
		case 1:
			// Random student query
			studentIdx := rand.Intn(len(dataset.Students))
			queries[i] = dcb.NewQuery(dcb.NewTags("student_id", dataset.Students[studentIdx].ID), "StudentRegistered")
		case 2:
			// Random enrollment query
			enrollmentIdx := rand.Intn(len(dataset.Enrollments))
			enrollment := dataset.Enrollments[enrollmentIdx]
			queries[i] = dcb.NewQuery(dcb.NewTags("student_id", enrollment.StudentID, "course_id", enrollment.CourseID), "StudentEnrolledInCourse")
		case 3:
			// All events of a type
			eventTypes := []string{"CourseCreated", "StudentRegistered", "StudentEnrolledInCourse"}
			eventType := eventTypes[rand.Intn(len(eventTypes))]
			queries[i] = dcb.NewQuery(dcb.NewTags(), eventType)
		}
	}

	return queries
}

// TruncateDatabase truncates the events table and resets the position sequence
func TruncateDatabase(ctx context.Context, store dcb.EventStore) error {
	// Use the proper DCB API for truncating events
	if err := dcb.TruncateEvents(ctx, store); err != nil {
		return fmt.Errorf("failed to truncate events: %w", err)
	}

	fmt.Println("Database truncated successfully")
	return nil
}
