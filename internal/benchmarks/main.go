package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"go-crablet/pkg/dcb"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	ctx := context.Background()

	// Use the existing docker-compose setup
	// The docker-compose.yaml file should be running with the schema.sql already applied
	dsn := "postgres://postgres:postgres@localhost:5432/dcb_app?sslmode=disable"

	// Wait for database to be ready
	maxRetries := 30
	for i := 0; i < maxRetries; i++ {
		pool, err := pgxpool.New(ctx, dsn)
		if err == nil {
			// Test the connection
			err = pool.Ping(ctx)
			if err == nil {
				pool.Close()
				break
			}
			pool.Close()
		}

		if i == maxRetries-1 {
			log.Fatalf("Failed to connect to database after %d retries. Make sure docker-compose is running: docker-compose up -d", maxRetries)
		}

		time.Sleep(1 * time.Second)
	}

	// Connect to database
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	// Create event store
	store, err := dcb.NewEventStore(ctx, pool)
	if err != nil {
		log.Fatalf("Failed to create event store: %v", err)
	}

	// Check if ChannelEventStore is available
	channelStore, hasChannel := store.(dcb.ChannelEventStore)

	fmt.Println("=== DCB Performance Benchmarks ===")
	fmt.Printf("Database: localhost:5432/dcb_app\n")
	fmt.Printf("Channel streaming available: %v\n", hasChannel)
	fmt.Println()

	// Run benchmarks
	runAppendBenchmarks(ctx, store)
	runReadBenchmarks(ctx, store)
	runStreamBenchmarks(ctx, store, channelStore, hasChannel)
	runProjectionBenchmarks(ctx, store, channelStore, hasChannel)

	fmt.Println("=== Benchmark Complete ===")
}

func runAppendBenchmarks(ctx context.Context, store dcb.EventStore) {
	fmt.Println("--- Append Benchmarks ---")

	// Single event append
	benchmarkSingleAppend(ctx, store)

	// Batch append
	benchmarkBatchAppend(ctx, store)

	// Concurrent append
	benchmarkConcurrentAppend(ctx, store)

	fmt.Println()
}

func runReadBenchmarks(ctx context.Context, store dcb.EventStore) {
	fmt.Println("--- Read Benchmarks ---")

	// Setup test data
	setupTestData(ctx, store)

	// Simple queries
	benchmarkSimpleQueries(ctx, store)

	// Complex queries
	benchmarkComplexQueries(ctx, store)

	fmt.Println()
}

func runStreamBenchmarks(ctx context.Context, store dcb.EventStore, channelStore dcb.ChannelEventStore, hasChannel bool) {
	fmt.Println("--- Streaming Benchmarks ---")

	// Iterator vs Channel comparison
	if hasChannel {
		benchmarkIteratorVsChannel(ctx, store, channelStore)
	}

	// Memory usage comparison
	benchmarkMemoryUsage(ctx, store, channelStore, hasChannel)

	fmt.Println()
}

func runProjectionBenchmarks(ctx context.Context, store dcb.EventStore, channelStore dcb.ChannelEventStore, hasChannel bool) {
	fmt.Println("--- Projection Benchmarks ---")

	// Single projector
	benchmarkSingleProjector(ctx, store)

	// Multiple projectors
	benchmarkMultipleProjectors(ctx, store)

	// Channel projection
	if hasChannel {
		benchmarkChannelProjection(ctx, channelStore)
	}

	fmt.Println()
}

func benchmarkSingleAppend(ctx context.Context, store dcb.EventStore) {
	fmt.Println("Single Event Append:")

	event := dcb.NewInputEvent("TestEvent", dcb.NewTags("test", "single"), []byte(`{"value": "test"}`))

	start := time.Now()
	// Use batch append with a single event to demonstrate the pattern
	_, err := store.Append(ctx, []dcb.InputEvent{event}, nil)
	duration := time.Since(start)

	if err != nil {
		fmt.Printf("  Error: %v\n", err)
		return
	}

	fmt.Printf("  Duration: %v\n", duration)
	fmt.Printf("  Throughput: %.2f events/sec\n", 1.0/duration.Seconds())
}

func benchmarkBatchAppend(ctx context.Context, store dcb.EventStore) {
	fmt.Println("Batch Append:")

	batchSizes := []int{10, 100, 1000}

	for _, size := range batchSizes {
		events := make([]dcb.InputEvent, size)
		for i := 0; i < size; i++ {
			events[i] = dcb.NewInputEvent("TestEvent", dcb.NewTags("test", "batch", "index", fmt.Sprintf("%d", i)), []byte(`{"value": "test"}`))
		}

		start := time.Now()
		_, err := store.Append(ctx, events, nil)
		duration := time.Since(start)

		if err != nil {
			fmt.Printf("  Batch %d: Error: %v\n", size, err)
			continue
		}

		fmt.Printf("  Batch %d: %v (%.2f events/sec)\n", size, duration, float64(size)/duration.Seconds())
	}
}

func benchmarkConcurrentAppend(ctx context.Context, store dcb.EventStore) {
	fmt.Println("Concurrent Append:")

	concurrencyLevels := []int{1, 5, 10, 20}
	eventsPerGoroutine := 100

	for _, concurrency := range concurrencyLevels {
		start := time.Now()

		// Use errgroup or similar for proper error handling in production
		done := make(chan bool, concurrency)
		for i := 0; i < concurrency; i++ {
			go func(id int) {
				events := make([]dcb.InputEvent, eventsPerGoroutine)
				for j := 0; j < eventsPerGoroutine; j++ {
					events[j] = dcb.NewInputEvent("TestEvent", dcb.NewTags("test", "concurrent", "goroutine", fmt.Sprintf("%d", id), "index", fmt.Sprintf("%d", j)), []byte(`{"value": "test"}`))
				}
				_, err := store.Append(ctx, events, nil)
				if err != nil {
					fmt.Printf("    Goroutine %d error: %v\n", id, err)
				}
				done <- true
			}(i)
		}

		// Wait for all goroutines
		for i := 0; i < concurrency; i++ {
			<-done
		}

		duration := time.Since(start)
		totalEvents := concurrency * eventsPerGoroutine

		fmt.Printf("  %d goroutines: %v (%.2f events/sec)\n", concurrency, duration, float64(totalEvents)/duration.Seconds())
	}
}

func setupTestData(ctx context.Context, store dcb.EventStore) {
	fmt.Println("Setting up test data...")

	// Create courses
	courses := 1000
	students := 10000
	enrollments := 50000

	// Append course events
	courseEvents := make([]dcb.InputEvent, courses)
	for i := 0; i < courses; i++ {
		courseID := fmt.Sprintf("course-%d", i)
		courseEvents[i] = dcb.NewInputEvent("CourseCreated", dcb.NewTags("course_id", courseID), []byte(fmt.Sprintf(`{"courseId": "%s", "name": "Course %d", "capacity": 100, "instructor": "Instructor %d"}`, courseID, i, i)))
	}

	_, err := store.Append(ctx, courseEvents, nil)
	if err != nil {
		fmt.Printf("Error creating courses: %v\n", err)
		return
	}

	// Append student events
	studentEvents := make([]dcb.InputEvent, students)
	for i := 0; i < students; i++ {
		studentID := fmt.Sprintf("student-%d", i)
		studentEvents[i] = dcb.NewInputEvent("StudentRegistered", dcb.NewTags("student_id", studentID), []byte(fmt.Sprintf(`{"studentId": "%s", "name": "Student %d", "email": "student%d@example.com"}`, studentID, i, i)))
	}

	_, err = store.Append(ctx, studentEvents, nil)
	if err != nil {
		fmt.Printf("Error creating students: %v\n", err)
		return
	}

	// Append enrollment events
	enrollmentEvents := make([]dcb.InputEvent, enrollments)
	for i := 0; i < enrollments; i++ {
		studentID := fmt.Sprintf("student-%d", i%students)
		courseID := fmt.Sprintf("course-%d", i%courses)
		enrollmentEvents[i] = dcb.NewInputEvent("StudentEnrolledInCourse", dcb.NewTags("student_id", studentID, "course_id", courseID), []byte(fmt.Sprintf(`{"studentId": "%s", "courseId": "%s", "enrolledAt": "2024-01-01"}`, studentID, courseID)))
	}

	_, err = store.Append(ctx, enrollmentEvents, nil)
	if err != nil {
		fmt.Printf("Error creating enrollments: %v\n", err)
		return
	}

	fmt.Printf("Created %d courses, %d students, %d enrollments\n", courses, students, enrollments)
}

func benchmarkSimpleQueries(ctx context.Context, store dcb.EventStore) {
	fmt.Println("Simple Queries:")

	// Query all course events
	start := time.Now()
	query := dcb.NewQuery(dcb.NewTags(), "CourseCreated")
	result, err := store.Read(ctx, query, nil)
	duration := time.Since(start)

	if err != nil {
		fmt.Printf("  All courses: Error: %v\n", err)
	} else {
		fmt.Printf("  All courses: %v (%d events)\n", duration, len(result.Events))
	}

	// Query by tag
	start = time.Now()
	query = dcb.NewQuery(dcb.NewTags("course_id", "course-1"), "CourseCreated")
	result, err = store.Read(ctx, query, nil)
	duration = time.Since(start)

	if err != nil {
		fmt.Printf("  Course by ID: Error: %v\n", err)
	} else {
		fmt.Printf("  Course by ID: %v (%d events)\n", duration, len(result.Events))
	}
}

func benchmarkComplexQueries(ctx context.Context, store dcb.EventStore) {
	fmt.Println("Complex Queries:")

	// OR query
	start := time.Now()
	query := dcb.Query{
		Items: []dcb.QueryItem{
			{EventTypes: []string{"CourseCreated"}, Tags: dcb.NewTags("course_id", "course-1")},
			{EventTypes: []string{"StudentRegistered"}, Tags: dcb.NewTags("student_id", "student-1")},
		},
	}
	result, err := store.Read(ctx, query, nil)
	duration := time.Since(start)

	if err != nil {
		fmt.Printf("  OR query: Error: %v\n", err)
	} else {
		fmt.Printf("  OR query: %v (%d events)\n", duration, len(result.Events))
	}

	// Query with limit
	start = time.Now()
	limit := 100
	options := &dcb.ReadOptions{Limit: &limit}
	query = dcb.NewQuery(dcb.NewTags(), "StudentEnrolledInCourse")
	result, err = store.Read(ctx, query, options)
	duration = time.Since(start)

	if err != nil {
		fmt.Printf("  Limited query: Error: %v\n", err)
	} else {
		fmt.Printf("  Limited query: %v (%d events)\n", duration, len(result.Events))
	}
}

func benchmarkIteratorVsChannel(ctx context.Context, store dcb.EventStore, channelStore dcb.ChannelEventStore) {
	fmt.Println("Iterator vs Channel:")

	query := dcb.NewQuery(dcb.NewTags(), "StudentEnrolledInCourse")

	// Iterator approach
	start := time.Now()
	iterator, err := store.ReadStream(ctx, query, nil)
	if err != nil {
		fmt.Printf("  Iterator: Error creating stream: %v\n", err)
	} else {
		count := 0
		for iterator.Next() {
			count++
		}
		iterator.Close()
		duration := time.Since(start)
		fmt.Printf("  Iterator: %v (%d events)\n", duration, count)
	}

	// Channel approach
	start = time.Now()
	eventChan, err := channelStore.ReadStreamChannel(ctx, query, nil)
	if err != nil {
		fmt.Printf("  Channel: Error creating stream: %v\n", err)
	} else {
		count := 0
		for range eventChan {
			count++
		}
		duration := time.Since(start)
		fmt.Printf("  Channel: %v (%d events)\n", duration, count)
	}
}

func benchmarkMemoryUsage(ctx context.Context, store dcb.EventStore, channelStore dcb.ChannelEventStore, hasChannel bool) {
	fmt.Println("Memory Usage Comparison:")

	// This would require runtime.ReadMemStats() for actual memory measurement
	// For now, just show the approach
	fmt.Println("  (Memory measurement requires runtime.ReadMemStats())")
}

func benchmarkSingleProjector(ctx context.Context, store dcb.EventStore) {
	fmt.Println("Single Projector:")

	projector := dcb.BatchProjector{
		ID: "courseCount",
		StateProjector: dcb.StateProjector{
			Query:        dcb.NewQuery(dcb.NewTags(), "CourseCreated"),
			InitialState: 0,
			TransitionFn: func(state any, event dcb.Event) any {
				return state.(int) + 1
			},
		},
	}

	start := time.Now()
	states, _, err := store.ProjectDecisionModel(ctx, []dcb.BatchProjector{projector}, nil)
	duration := time.Since(start)

	if err != nil {
		fmt.Printf("  Error: %v\n", err)
	} else {
		fmt.Printf("  Duration: %v (count: %d)\n", duration, states["courseCount"])
	}
}

func benchmarkMultipleProjectors(ctx context.Context, store dcb.EventStore) {
	fmt.Println("Multiple Projectors:")

	projectors := []dcb.BatchProjector{
		{
			ID: "courseCount",
			StateProjector: dcb.StateProjector{
				Query:        dcb.NewQuery(dcb.NewTags(), "CourseCreated"),
				InitialState: 0,
				TransitionFn: func(state any, event dcb.Event) any {
					return state.(int) + 1
				},
			},
		},
		{
			ID: "studentCount",
			StateProjector: dcb.StateProjector{
				Query:        dcb.NewQuery(dcb.NewTags(), "StudentRegistered"),
				InitialState: 0,
				TransitionFn: func(state any, event dcb.Event) any {
					return state.(int) + 1
				},
			},
		},
		{
			ID: "enrollmentCount",
			StateProjector: dcb.StateProjector{
				Query:        dcb.NewQuery(dcb.NewTags(), "StudentEnrolledInCourse"),
				InitialState: 0,
				TransitionFn: func(state any, event dcb.Event) any {
					return state.(int) + 1
				},
			},
		},
	}

	start := time.Now()
	states, _, err := store.ProjectDecisionModel(ctx, projectors, nil)
	duration := time.Since(start)

	if err != nil {
		fmt.Printf("  Error: %v\n", err)
	} else {
		fmt.Printf("  Duration: %v\n", duration)
		for id, state := range states {
			fmt.Printf("    %s: %d\n", id, state)
		}
	}
}

func benchmarkChannelProjection(ctx context.Context, channelStore dcb.ChannelEventStore) {
	fmt.Println("Channel Projection:")

	projectors := []dcb.BatchProjector{
		{
			ID: "courseCount",
			StateProjector: dcb.StateProjector{
				Query:        dcb.NewQuery(dcb.NewTags(), "CourseCreated"),
				InitialState: 0,
				TransitionFn: func(state any, event dcb.Event) any {
					return state.(int) + 1
				},
			},
		},
	}

	start := time.Now()
	resultChan, err := channelStore.ProjectDecisionModelChannel(ctx, projectors, nil)
	if err != nil {
		fmt.Printf("  Error: %v\n", err)
		return
	}

	count := 0
	for range resultChan {
		count++
	}
	duration := time.Since(start)

	fmt.Printf("  Duration: %v (%d results)\n", duration, count)
}
