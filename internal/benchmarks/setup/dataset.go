package setup

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"go-crablet/pkg/dcb"
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
	Category   string
	Popularity float64 // 0.0 to 1.0, higher means more popular
}

// StudentData represents a student in the test dataset
type StudentData struct {
	ID         string
	Name       string
	Email      string
	Major      string
	Year       int // 1-4 for undergraduate years
	MaxCourses int // How many courses this student typically takes
}

// EnrollmentData represents an enrollment in the test dataset
type EnrollmentData struct {
	StudentID  string
	CourseID   string
	EnrolledAt time.Time
	Grade      string // A, B, C, D, F, or empty for current enrollments
}

// Dataset represents the complete test dataset
type Dataset struct {
	Courses     []CourseData
	Students    []StudentData
	Enrollments []EnrollmentData
	Config      DatasetConfig
}

// Course categories for realistic distribution
var CourseCategories = []string{
	"Computer Science", "Mathematics", "Physics", "Chemistry", "Biology",
	"Literature", "History", "Philosophy", "Economics", "Psychology",
	"Engineering", "Art", "Music", "Business", "Political Science",
}

// Student majors for realistic distribution
var StudentMajors = []string{
	"Computer Science", "Mathematics", "Physics", "Chemistry", "Biology",
	"English", "History", "Philosophy", "Economics", "Psychology",
	"Engineering", "Art", "Music", "Business", "Political Science",
}

// GenerateDataset creates a complete test dataset with realistic distributions
func GenerateDataset(config DatasetConfig) *Dataset {
	dataset := &Dataset{
		Config: config,
	}

	// Seed random number generator
	rand.Seed(time.Now().UnixNano())

	// Generate courses with popularity distribution
	dataset.Courses = generateCourses(config.Courses, config.Capacity)

	// Generate students with realistic patterns
	dataset.Students = generateStudents(config.Students)

	// Generate enrollments with realistic distribution
	dataset.Enrollments = generateEnrollments(dataset.Courses, dataset.Students, config.Enrollments)

	return dataset
}

// generateCourses creates courses with realistic popularity distribution
func generateCourses(count, capacity int) []CourseData {
	courses := make([]CourseData, count)

	for i := 0; i < count; i++ {
		// Create popularity distribution: 20% popular, 60% average, 20% less popular
		var popularity float64
		randVal := rand.Float64()
		if randVal < 0.2 {
			// Popular courses (0.7-1.0)
			popularity = 0.7 + rand.Float64()*0.3
		} else if randVal < 0.8 {
			// Average courses (0.3-0.7)
			popularity = 0.3 + rand.Float64()*0.4
		} else {
			// Less popular courses (0.1-0.3)
			popularity = 0.1 + rand.Float64()*0.2
		}

		courses[i] = CourseData{
			ID:         fmt.Sprintf("course-%d", i),
			Name:       fmt.Sprintf("Course %d", i),
			Instructor: fmt.Sprintf("Instructor %d", i),
			Capacity:   capacity,
			Category:   CourseCategories[rand.Intn(len(CourseCategories))],
			Popularity: popularity,
		}
	}

	return courses
}

// generateStudents creates students with realistic enrollment patterns
func generateStudents(count int) []StudentData {
	students := make([]StudentData, count)

	for i := 0; i < count; i++ {
		// Student year distribution: more lower years than upper years
		year := 1
		randVal := rand.Float64()
		if randVal < 0.4 {
			year = 1 // 40% first year
		} else if randVal < 0.7 {
			year = 2 // 30% second year
		} else if randVal < 0.9 {
			year = 3 // 20% third year
		} else {
			year = 4 // 10% fourth year
		}

		// Max courses per student: 3-8 courses typical
		maxCourses := 3 + rand.Intn(6)

		students[i] = StudentData{
			ID:         fmt.Sprintf("student-%d", i),
			Name:       fmt.Sprintf("Student %d", i),
			Email:      fmt.Sprintf("student%d@example.com", i),
			Major:      StudentMajors[rand.Intn(len(StudentMajors))],
			Year:       year,
			MaxCourses: maxCourses,
		}
	}

	return students
}

// generateEnrollments creates enrollments with realistic distribution patterns
func generateEnrollments(courses []CourseData, students []StudentData, count int) []EnrollmentData {
	enrollments := make([]EnrollmentData, 0, count)

	// Track enrollments per student to respect max courses
	studentEnrollments := make(map[string]int)

	// Track enrollments per course to respect capacity
	courseEnrollments := make(map[string]int)

	// Generate enrollments over a realistic time period (last 2 years)
	startDate := time.Now().AddDate(-2, 0, 0)
	endDate := time.Now()

	for i := 0; i < count; i++ {
		// Select student (weighted by remaining capacity)
		student := selectStudent(students, studentEnrollments)
		if student == nil {
			// All students at capacity, skip
			continue
		}

		// Select course (weighted by popularity and remaining capacity)
		course := selectCourse(courses, courseEnrollments, student.Major)
		if course == nil {
			// All courses at capacity, skip
			continue
		}

		// Check if this enrollment already exists
		exists := false
		for _, existing := range enrollments {
			if existing.StudentID == student.ID && existing.CourseID == course.ID {
				exists = true
				break
			}
		}
		if exists {
			continue
		}

		// Generate realistic enrollment date
		enrollmentDate := startDate.Add(time.Duration(rand.Int63n(endDate.Sub(startDate).Nanoseconds())))

		// Generate grade (older enrollments more likely to have grades)
		var grade string
		if enrollmentDate.Before(time.Now().AddDate(0, -3, 0)) {
			// Completed courses get grades
			grades := []string{"A", "B", "C", "D", "F"}
			grade = grades[rand.Intn(len(grades))]
		}

		enrollment := EnrollmentData{
			StudentID:  student.ID,
			CourseID:   course.ID,
			EnrolledAt: enrollmentDate,
			Grade:      grade,
		}

		enrollments = append(enrollments, enrollment)
		studentEnrollments[student.ID]++
		courseEnrollments[course.ID]++
	}

	return enrollments
}

// selectStudent selects a student based on remaining enrollment capacity
func selectStudent(students []StudentData, studentEnrollments map[string]int) *StudentData {
	// Find students with remaining capacity
	var availableStudents []*StudentData
	for i := range students {
		student := &students[i]
		if studentEnrollments[student.ID] < student.MaxCourses {
			availableStudents = append(availableStudents, student)
		}
	}

	if len(availableStudents) == 0 {
		return nil
	}

	// Weight by remaining capacity (students with more capacity more likely)
	totalWeight := 0
	for _, student := range availableStudents {
		remaining := student.MaxCourses - studentEnrollments[student.ID]
		totalWeight += remaining
	}

	randVal := rand.Intn(totalWeight)
	currentWeight := 0
	for _, student := range availableStudents {
		remaining := student.MaxCourses - studentEnrollments[student.ID]
		currentWeight += remaining
		if randVal < currentWeight {
			return student
		}
	}

	return availableStudents[0] // Fallback
}

// selectCourse selects a course based on popularity, capacity, and student major
func selectCourse(courses []CourseData, courseEnrollments map[string]int, studentMajor string) *CourseData {
	// Find courses with remaining capacity
	var availableCourses []*CourseData
	for i := range courses {
		course := &courses[i]
		if courseEnrollments[course.ID] < course.Capacity {
			availableCourses = append(availableCourses, course)
		}
	}

	if len(availableCourses) == 0 {
		return nil
	}

	// Weight by popularity and major alignment
	totalWeight := 0.0
	weights := make([]float64, len(availableCourses))

	for i, course := range availableCourses {
		// Base weight from popularity
		weight := course.Popularity

		// Bonus for major alignment (students prefer courses in their major)
		if course.Category == studentMajor {
			weight *= 1.5
		}

		// Bonus for remaining capacity (less full courses more attractive)
		remainingCapacity := float64(course.Capacity - courseEnrollments[course.ID])
		capacityBonus := remainingCapacity / float64(course.Capacity)
		weight *= (1.0 + capacityBonus)

		weights[i] = weight
		totalWeight += weight
	}

	// Select based on weights
	randVal := rand.Float64() * totalWeight
	currentWeight := 0.0
	for i, course := range availableCourses {
		currentWeight += weights[i]
		if randVal < currentWeight {
			return course
		}
	}

	return availableCourses[0] // Fallback
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
			events[j] = dcb.NewInputEvent("CourseDefined",
				dcb.NewTags("course_id", course.ID, "category", course.Category),
				[]byte(fmt.Sprintf(`{"courseId": "%s", "name": "%s", "capacity": %d, "instructor": "%s", "category": "%s", "popularity": %.2f}`,
					course.ID, course.Name, course.Capacity, course.Instructor, course.Category, course.Popularity)))
		}

		err := store.Append(ctx, events, nil)
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
				dcb.NewTags("student_id", student.ID, "major", student.Major, "year", fmt.Sprintf("%d", student.Year)),
				[]byte(fmt.Sprintf(`{"studentId": "%s", "name": "%s", "email": "%s", "major": "%s", "year": %d, "maxCourses": %d}`,
					student.ID, student.Name, student.Email, student.Major, student.Year, student.MaxCourses)))
		}

		err := store.Append(ctx, events, nil)
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
			// Add grade tag if present
			tags := dcb.NewTags("student_id", enrollment.StudentID, "course_id", enrollment.CourseID)
			if enrollment.Grade != "" {
				tags = dcb.NewTags("student_id", enrollment.StudentID, "course_id", enrollment.CourseID, "grade", enrollment.Grade)
			}

			events[j] = dcb.NewInputEvent("StudentEnrolledInCourse",
				tags,
				[]byte(fmt.Sprintf(`{"studentId": "%s", "courseId": "%s", "enrolledAt": "%s", "grade": "%s"}`,
					enrollment.StudentID, enrollment.CourseID, enrollment.EnrolledAt.Format(time.RFC3339), enrollment.Grade)))
		}

		err := store.Append(ctx, events, nil)
		if err != nil {
			return fmt.Errorf("failed to append enrollment batch %d-%d: %w", i, end-1, err)
		}
	}

	return nil
}

// GenerateRandomQueries creates random queries for testing with realistic patterns
func GenerateRandomQueries(dataset *Dataset, count int) []dcb.Query {
	queries := make([]dcb.Query, count)
	rand.Seed(time.Now().UnixNano())

	for i := 0; i < count; i++ {
		queryType := rand.Intn(8) // More query types for better testing

		switch queryType {
		case 0:
			// Random course query
			courseIdx := rand.Intn(len(dataset.Courses))
			queries[i] = dcb.NewQuery(dcb.NewTags("course_id", dataset.Courses[courseIdx].ID), "CourseDefined")
		case 1:
			// Random student query
			studentIdx := rand.Intn(len(dataset.Students))
			queries[i] = dcb.NewQuery(dcb.NewTags("student_id", dataset.Students[studentIdx].ID), "StudentRegistered")
		case 2:
			// Random enrollment query
			if len(dataset.Enrollments) > 0 {
				enrollmentIdx := rand.Intn(len(dataset.Enrollments))
				enrollment := dataset.Enrollments[enrollmentIdx]
				queries[i] = dcb.NewQuery(dcb.NewTags("student_id", enrollment.StudentID, "course_id", enrollment.CourseID), "StudentEnrolledInCourse")
			}
		case 3:
			// Category-based course query
			category := CourseCategories[rand.Intn(len(CourseCategories))]
			queries[i] = dcb.NewQuery(dcb.NewTags("category", category), "CourseDefined")
		case 4:
			// Major-based student query
			major := StudentMajors[rand.Intn(len(StudentMajors))]
			queries[i] = dcb.NewQuery(dcb.NewTags("major", major), "StudentRegistered")
		case 5:
			// Year-based student query
			year := 1 + rand.Intn(4)
			queries[i] = dcb.NewQuery(dcb.NewTags("year", fmt.Sprintf("%d", year)), "StudentRegistered")
		case 6:
			// Grade-based enrollment query
			grades := []string{"A", "B", "C", "D", "F"}
			grade := grades[rand.Intn(len(grades))]
			queries[i] = dcb.NewQuery(dcb.NewTags("grade", grade), "StudentEnrolledInCourse")
		case 7:
			// Complex OR query across multiple types
			queries[i] = dcb.NewQueryFromItems(
				dcb.NewQueryItem([]string{"CourseDefined"}, dcb.NewTags("category", "Computer Science")),
				dcb.NewQueryItem([]string{"StudentRegistered"}, dcb.NewTags("major", "Computer Science")),
				dcb.NewQueryItem([]string{"StudentEnrolledInCourse"}, dcb.NewTags("grade", "A")),
			)
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
