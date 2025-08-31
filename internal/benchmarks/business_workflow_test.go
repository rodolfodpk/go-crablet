package benchmarks

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/rodolfodpk/go-crablet/pkg/dcb"
)

// BenchmarkStudentEnrollmentWorkflow_Small tests a complete student enrollment workflow
// that mirrors real-world business processes: validation → decision → action
func BenchmarkStudentEnrollmentWorkflow_Small(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "medium", 100)
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Use real data from the loaded dataset
		student := benchCtx.Dataset.Students[i%len(benchCtx.Dataset.Students)]
		course := benchCtx.Dataset.Courses[i%len(benchCtx.Dataset.Courses)]

		// Simulate realistic business workflow: Student Course Enrollment
		// 1. Check if student exists
		// 2. Check if course exists
		// 3. Check prerequisites
		// 4. Attempt enrollment

		// Step 1: Check if student exists (query real student)
		studentQuery := dcb.NewQuery(dcb.NewTags("student_id", student.ID), "StudentRegistered")
		cursor := &dcb.Cursor{}

		_, err := benchCtx.Store.Query(ctx, studentQuery, cursor)
		if err != nil {
			b.Fatal(err)
		}

		// Step 2: Check if course exists (query real course)
		courseQuery := dcb.NewQuery(dcb.NewTags("course_id", course.ID), "CourseDefined")
		_, err = benchCtx.Store.Query(ctx, courseQuery, cursor)
		if err != nil {
			b.Fatal(err)
		}

		// Step 3: Check if student is already enrolled (business rule validation)
		enrollmentQuery := dcb.NewQuery(dcb.NewTags("student_id", student.ID, "course_id", course.ID), "StudentEnrolledInCourse")
		_, err = benchCtx.Store.Query(ctx, enrollmentQuery, cursor)
		// This might return no results, which is expected for new enrollments

		// Step 4: Attempt enrollment (real business event)
		enrollmentEvent := dcb.NewInputEvent("StudentEnrolledInCourse",
			dcb.NewTags("student_id", student.ID, "course_id", course.ID, "enrolled_at", time.Now().Format(time.RFC3339)),
			[]byte(fmt.Sprintf(`{
				"studentId": "%s",
				"courseId": "%s", 
				"enrolledAt": "%s",
				"status": "enrolled"
			}`, student.ID, course.ID, time.Now().Format(time.RFC3339))))

		err = benchCtx.Store.Append(ctx, []dcb.InputEvent{enrollmentEvent})
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkConcurrentCourseRegistration_Small simulates multiple concurrent users
// performing course registration operations simultaneously
func BenchmarkConcurrentCourseRegistration_Small(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "small", 100)
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	// Simulate 10 concurrent users
	concurrentUsers := 10
	var wg sync.WaitGroup

	for i := 0; i < b.N; i++ {
		wg.Add(concurrentUsers)

		for userID := 0; userID < concurrentUsers; userID++ {
			go func(userID int) {
				defer wg.Done()

				// Use real student data from the dataset
				student := benchCtx.Dataset.Students[userID%len(benchCtx.Dataset.Students)]

				// Each concurrent user performs a realistic action: course registration
				event := dcb.NewInputEvent("StudentCourseRegistration",
					dcb.NewTags("student_id", student.ID, "action_type", "registration", "concurrent_user", fmt.Sprintf("%d", userID)),
					[]byte(fmt.Sprintf(`{
						"studentId": "%s",
						"action": "course_registration",
						"timestamp": "%s",
						"concurrentUser": %d
					}`, student.ID, time.Now().Format(time.RFC3339), userID)))

				err := benchCtx.Store.Append(ctx, []dcb.InputEvent{event})
				if err != nil {
					b.Fatal(err)
				}
			}(userID)
		}

		wg.Wait()
	}
}

// BenchmarkConcurrentCourseRegistration_Medium simulates 100 concurrent users
// performing course registration operations simultaneously
func BenchmarkConcurrentCourseRegistration_Medium(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "medium", 100)
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	// Simulate 100 concurrent users
	concurrentUsers := 100
	var wg sync.WaitGroup

	for i := 0; i < b.N; i++ {
		wg.Add(concurrentUsers)

		for userID := 0; userID < concurrentUsers; userID++ {
			go func(userID int) {
				defer wg.Done()

				// Use real student data from the dataset
				student := benchCtx.Dataset.Students[userID%len(benchCtx.Dataset.Students)]

				// Each concurrent user performs a realistic action: course registration
				event := dcb.NewInputEvent("StudentCourseRegistration",
					dcb.NewTags("student_id", student.ID, "action_type", "registration", "concurrent_user", fmt.Sprintf("%d", userID)),
					[]byte(fmt.Sprintf(`{
						"studentId": "%s",
						"action": "course_registration",
						"timestamp": "%s",
						"concurrentUser": %d
					}`, student.ID, time.Now().Format(time.RFC3339), userID)))

				err := benchCtx.Store.Append(ctx, []dcb.InputEvent{event})
				if err != nil {
					b.Fatal(err)
				}
			}(userID)
		}

		wg.Wait()
	}
}

// BenchmarkEnrollmentValidation_Small tests enrollment business rule validation
// including capacity checks, prerequisite validation, and enrollment limits
func BenchmarkEnrollmentValidation_Small(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "small", 100)
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Use real data from the loaded dataset
		student := benchCtx.Dataset.Students[i%len(benchCtx.Dataset.Students)]
		course := benchCtx.Dataset.Courses[i%len(benchCtx.Dataset.Courses)]

		// Simulate business rule validation workflow
		// 1. Check student enrollment count
		// 2. Check course capacity
		// 3. Check prerequisites
		// 4. Validate business rules

		// Step 1: Check student enrollment count
		enrollmentCountQuery := dcb.NewQuery(dcb.NewTags("student_id", student.ID), "StudentEnrolledInCourse")
		enrollments, err := benchCtx.Store.Query(ctx, enrollmentCountQuery, nil)
		if err != nil {
			b.Fatal(err)
		}

		// Business rule: Student can't enroll in more than 10 courses
		if len(enrollments) >= 10 {
			continue // Skip this iteration
		}

		// Step 2: Check course capacity
		courseEnrollmentQuery := dcb.NewQuery(dcb.NewTags("course_id", course.ID), "StudentEnrolledInCourse")
		courseEnrollments, err := benchCtx.Store.Query(ctx, courseEnrollmentQuery, nil)
		if err != nil {
			b.Fatal(err)
		}

		// Business rule: Course capacity is 50 students
		if len(courseEnrollments) >= 50 {
			continue // Skip this iteration
		}

		// Step 3: Check prerequisites (simplified)
		prerequisiteQuery := dcb.NewQuery(dcb.NewTags("course_id", course.ID), "CoursePrerequisite")
		prerequisites, err := benchCtx.Store.Query(ctx, prerequisiteQuery, nil)
		if err != nil {
			b.Fatal(err)
		}

		// Business rule: Check if student has completed prerequisites
		hasPrerequisites := true
		for _, prereq := range prerequisites {
			prereqQuery := dcb.NewQuery(dcb.NewTags("student_id", student.ID, "course_id", prereq.Type), "StudentCompletedCourse")
			completed, err := benchCtx.Store.Query(ctx, prereqQuery, nil)
			if err != nil {
				b.Fatal(err)
			}
			if len(completed) == 0 {
				hasPrerequisites = false
				break
			}
		}

		if !hasPrerequisites {
			continue // Skip this iteration
		}

		// Step 4: All validations passed, proceed with enrollment
		enrollmentEvent := dcb.NewInputEvent("StudentEnrolledInCourse",
			dcb.NewTags("student_id", student.ID, "course_id", course.ID, "enrolled_at", time.Now().Format(time.RFC3339)),
			[]byte(fmt.Sprintf(`{
				"studentId": "%s",
				"courseId": "%s", 
				"enrolledAt": "%s",
				"status": "enrolled"
			}`, student.ID, course.ID, time.Now().Format(time.RFC3339))))

		err = benchCtx.Store.Append(ctx, []dcb.InputEvent{enrollmentEvent})
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkReadWriteEnrollment_Small tests mixed read and write operations
// simulating real-world enrollment application patterns: read state → validate → write → confirm
func BenchmarkReadWriteEnrollment_Small(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "small", 100)
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Use real data from the loaded dataset
		student := benchCtx.Dataset.Students[i%len(benchCtx.Dataset.Students)]
		course := benchCtx.Dataset.Courses[i%len(benchCtx.Dataset.Courses)]

		// Simulate mixed read/write operations
		// 1. Read current state
		// 2. Validate business rules
		// 3. Write new events
		// 4. Read updated state

		// Step 1: Read current enrollment state
		enrollmentQuery := dcb.NewQuery(dcb.NewTags("student_id", student.ID, "course_id", course.ID), "StudentEnrolledInCourse")
		currentEnrollments, err := benchCtx.Store.Query(ctx, enrollmentQuery, nil)
		if err != nil {
			b.Fatal(err)
		}

		// Step 2: If not enrolled, proceed with enrollment
		if len(currentEnrollments) == 0 {
			// Write enrollment event
			enrollmentEvent := dcb.NewInputEvent("StudentEnrolledInCourse",
				dcb.NewTags("student_id", student.ID, "course_id", course.ID, "enrolled_at", time.Now().Format(time.RFC3339)),
				[]byte(fmt.Sprintf(`{
					"studentId": "%s",
					"courseId": "%s", 
					"enrolledAt": "%s",
					"status": "enrolled"
				}`, student.ID, course.ID, time.Now().Format(time.RFC3339))))

			err = benchCtx.Store.Append(ctx, []dcb.InputEvent{enrollmentEvent})
			if err != nil {
				b.Fatal(err)
			}

			// Step 3: Read updated state to confirm
			updatedEnrollments, err := benchCtx.Store.Query(ctx, enrollmentQuery, nil)
			if err != nil {
				b.Fatal(err)
			}

			// Verify the enrollment was successful
			if len(updatedEnrollments) == 0 {
				b.Fatal("Enrollment was not successful")
			}
		}
	}
}
