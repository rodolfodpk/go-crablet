package benchmarks

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/rodolfodpk/go-crablet/pkg/dcb"
)

// BenchmarkConcurrentScaling_1User tests single user performance
func BenchmarkConcurrentScaling_1User(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "small", 100)
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Single user performing course registration
		student := benchCtx.Dataset.Students[i%len(benchCtx.Dataset.Students)]

		event := dcb.NewInputEvent("StudentCourseRegistration",
			dcb.NewTags("student_id", student.ID, "action_type", "registration", "concurrent_user", "1"),
			[]byte(fmt.Sprintf(`{
				"studentId": "%s",
				"action": "course_registration",
				"timestamp": "%s",
				"concurrentUser": 1
			}`, student.ID, time.Now().Format(time.RFC3339))))

		err := benchCtx.Store.Append(ctx, []dcb.InputEvent{event})
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkConcurrentScaling_10Users tests 10 concurrent users
func BenchmarkConcurrentScaling_10Users(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "small", 100)
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	concurrentUsers := 10
	var wg sync.WaitGroup

	for i := 0; i < b.N; i++ {
		wg.Add(concurrentUsers)

		for userID := 0; userID < concurrentUsers; userID++ {
			go func(userID int) {
				defer wg.Done()

				student := benchCtx.Dataset.Students[userID%len(benchCtx.Dataset.Students)]

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

// BenchmarkConcurrentScaling_100Users tests 100 concurrent users
func BenchmarkConcurrentScaling_100Users(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "medium", 100)
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	concurrentUsers := 100
	var wg sync.WaitGroup

	for i := 0; i < b.N; i++ {
		wg.Add(concurrentUsers)

		for userID := 0; userID < concurrentUsers; userID++ {
			go func(userID int) {
				defer wg.Done()

				student := benchCtx.Dataset.Students[userID%len(benchCtx.Dataset.Students)]

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

// BenchmarkConcurrentRead_1User tests single user read performance
func BenchmarkConcurrentRead_1User(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "small", 100)
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		query := dcb.NewQuery(dcb.NewTags("test", "single"))
		_, err := benchCtx.Store.Query(ctx, query, nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkConcurrentRead_10Users tests 10 concurrent users reading
func BenchmarkConcurrentRead_10Users(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "small", 100)
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	concurrentUsers := 10
	var wg sync.WaitGroup

	for i := 0; i < b.N; i++ {
		wg.Add(concurrentUsers)

		for userID := 0; userID < concurrentUsers; userID++ {
			go func(userID int) {
				defer wg.Done()

				query := dcb.NewQuery(dcb.NewTags("test", "single"))
				_, err := benchCtx.Store.Query(ctx, query, nil)
				if err != nil {
					b.Fatal(err)
				}
			}(userID)
		}

		wg.Wait()
	}
}

// BenchmarkConcurrentRead_100Users tests 100 concurrent users reading
func BenchmarkConcurrentRead_100Users(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "medium", 100)
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	concurrentUsers := 100
	var wg sync.WaitGroup

	for i := 0; i < b.N; i++ {
		wg.Add(concurrentUsers)

		for userID := 0; userID < concurrentUsers; userID++ {
			go func(userID int) {
				defer wg.Done()

				query := dcb.NewQuery(dcb.NewTags("test", "single"))
				_, err := benchCtx.Store.Query(ctx, query, nil)
				if err != nil {
					b.Fatal(err)
				}
			}(userID)
		}

		wg.Wait()
	}
}

// BenchmarkConcurrentProjection_1User tests single user projection performance
func BenchmarkConcurrentProjection_1User(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "small", 100)
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		projector := dcb.ProjectState("test_projection", "TestEvent", "test", "single", map[string]any{}, func(state any, event dcb.Event) any {
			return state
		})

		_, _, err := benchCtx.Store.Project(ctx, []dcb.StateProjector{projector}, nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkConcurrentProjection_10Users tests 10 concurrent users projecting
func BenchmarkConcurrentProjection_10Users(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "small", 100)
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	concurrentUsers := 10
	var wg sync.WaitGroup

	for i := 0; i < b.N; i++ {
		wg.Add(concurrentUsers)

		for userID := 0; userID < concurrentUsers; userID++ {
			go func(userID int) {
				defer wg.Done()

				projector := dcb.ProjectState("test_projection", "TestEvent", "test", "single", map[string]any{}, func(state any, event dcb.Event) any {
					return state
				})

				_, _, err := benchCtx.Store.Project(ctx, []dcb.StateProjector{projector}, nil)
				if err != nil {
					b.Fatal(err)
				}
			}(userID)
		}

		wg.Wait()
	}
}

// BenchmarkConcurrentProjection_100Users tests 100 concurrent users projecting
func BenchmarkConcurrentProjection_100Users(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "medium", 100)
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	concurrentUsers := 100
	var wg sync.WaitGroup

	for i := 0; i < b.N; i++ {
		wg.Add(concurrentUsers)

		for userID := 0; userID < concurrentUsers; userID++ {
			go func(userID int) {
				defer wg.Done()

				projector := dcb.ProjectState("test_projection", "TestEvent", "test", "single", map[string]any{}, func(state any, event dcb.Event) any {
					return state
				})

				_, _, err := benchCtx.Store.Project(ctx, []dcb.StateProjector{projector}, nil)
				if err != nil {
					b.Fatal(err)
				}
			}(userID)
		}

		wg.Wait()
	}
}
