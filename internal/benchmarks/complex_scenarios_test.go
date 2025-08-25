package benchmarks

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/rodolfodpk/go-crablet/pkg/dcb"
)

// BenchmarkComplexBusinessWorkflow_Small tests a complete business workflow
// that mirrors real-world usage patterns
func BenchmarkComplexBusinessWorkflow_Small(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "small", 100)
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

// BenchmarkConcurrentAppends_Small simulates multiple concurrent users
// performing operations simultaneously
func BenchmarkConcurrentAppends_Small(b *testing.B) {
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

// BenchmarkMixedOperations_Small tests a mix of append, query, and projection
// operations that mirror real-world application patterns
func BenchmarkMixedOperations_Small(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "small", 100)
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// 1. Append some events
		events := []dcb.InputEvent{
			dcb.NewInputEvent("DataUpdate",
				dcb.NewTags("record_id", fmt.Sprintf("%d", i), "operation", "update"),
				[]byte(fmt.Sprintf(`{"record_id":%d,"value":"updated","timestamp":1234567890}`, i))),
		}

		err := benchCtx.Store.Append(ctx, events)
		if err != nil {
			b.Fatal(err)
		}

		// 2. Query for events
		query := dcb.NewQuery(dcb.NewTags("operation", "update"), "DataUpdate")
		cursor := &dcb.Cursor{}

		_, err = benchCtx.Store.Query(ctx, query, cursor)
		if err != nil {
			b.Fatal(err)
		}

		// 3. Project state (simplified projection)
		projector := dcb.ProjectState("test_projection", "DataUpdate", "operation", "update", map[string]any{}, func(state any, event dcb.Event) any {
			return state
		})

		_, _, err = benchCtx.Store.Project(ctx, []dcb.StateProjector{projector}, nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkBusinessRuleValidation_Small tests complex business rule validation
// scenarios that require multiple DCB conditions
func BenchmarkBusinessRuleValidation_Small(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "small", 100)
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Use real data from the dataset for realistic business rule validation
		student := benchCtx.Dataset.Students[i%len(benchCtx.Dataset.Students)]
		course := benchCtx.Dataset.Courses[i%len(benchCtx.Dataset.Courses)]

		// Simulate complex business rule: student can only enroll in courses
		// if they haven't already enrolled and the course has capacity

		// 1. Check if student is already enrolled (business rule validation)
		existingEnrollmentQuery := dcb.NewQuery(dcb.NewTags("student_id", student.ID, "course_id", course.ID), "StudentEnrolledInCourse")
		enrollmentCondition := dcb.NewAppendCondition(existingEnrollmentQuery)

		// 2. Try to enroll with condition (must NOT already be enrolled)
		enrollmentEvent := dcb.NewInputEvent("StudentEnrolledInCourse",
			dcb.NewTags("student_id", student.ID, "course_id", course.ID, "enrolled_at", time.Now().Format(time.RFC3339)),
			[]byte(fmt.Sprintf(`{
				"studentId": "%s",
				"courseId": "%s",
				"enrolledAt": "%s",
				"status": "enrolled"
			}`, student.ID, course.ID, time.Now().Format(time.RFC3339))))

		// This will fail if student is already enrolled, but that's expected
		// We're measuring the performance of the business rule validation logic
		_ = benchCtx.Store.AppendIf(ctx, []dcb.InputEvent{enrollmentEvent}, enrollmentCondition)
	}
}

// BenchmarkRequestBurst_Small simulates burst traffic patterns
// that are common in web applications
func BenchmarkRequestBurst_Small(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "small", 100)
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Simulate burst of 50 requests
		burstSize := 50
		var wg sync.WaitGroup
		wg.Add(burstSize)

		for j := 0; j < burstSize; j++ {
			go func(requestID int) {
				defer wg.Done()

				// Each request creates a simple event
				event := dcb.NewInputEvent("BurstRequest",
					dcb.NewTags("request_id", fmt.Sprintf("%d", requestID), "burst_id", fmt.Sprintf("%d", i)),
					[]byte(fmt.Sprintf(`{"request_id":%d,"timestamp":1234567890}`, requestID)))

				err := benchCtx.Store.Append(ctx, []dcb.InputEvent{event})
				if err != nil {
					b.Fatal(err)
				}
			}(j)
		}

		wg.Wait()
	}
}

// BenchmarkSustainedLoad_Small simulates sustained load over time
// to test performance consistency
func BenchmarkSustainedLoad_Small(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "small", 100)
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	// Simulate sustained load with multiple operation types
	for i := 0; i < b.N; i++ {
		// Mix of operations to simulate real application load
		operationType := i % 4

		switch operationType {
		case 0:
			// Simple append
			event := dcb.NewInputEvent("SustainedLoad",
				dcb.NewTags("operation", "append", "sequence", fmt.Sprintf("%d", i)),
				[]byte(fmt.Sprintf(`{"operation":"append","sequence":%d,"timestamp":1234567890}`, i)))

			err := benchCtx.Store.Append(ctx, []dcb.InputEvent{event})
			if err != nil {
				b.Fatal(err)
			}

		case 1:
			// Query operation
			query := dcb.NewQuery(dcb.NewTags("operation", "append"), "SustainedLoad")
			cursor := &dcb.Cursor{}

			_, err := benchCtx.Store.Query(ctx, query, cursor)
			if err != nil {
				b.Fatal(err)
			}

		case 2:
			// Projection operation
			projector := dcb.ProjectState("sustained_projection", "SustainedLoad", "operation", "append", map[string]any{}, func(state any, event dcb.Event) any {
				return state
			})

			_, _, err := benchCtx.Store.Project(ctx, []dcb.StateProjector{projector}, nil)
			if err != nil {
				b.Fatal(err)
			}

		case 3:
			// Conditional append
			condition := dcb.NewAppendCondition(
				dcb.NewQuery(dcb.NewTags("operation", "append"), "SustainedLoad"),
			)

			event := dcb.NewInputEvent("ConditionalLoad",
				dcb.NewTags("operation", "conditional", "sequence", fmt.Sprintf("%d", i)),
				[]byte(fmt.Sprintf(`{"operation":"conditional","sequence":%d,"timestamp":1234567890}`, i)))

			_ = benchCtx.Store.AppendIf(ctx, []dcb.InputEvent{event}, condition)
		}
	}
}
