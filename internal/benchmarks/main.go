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
	// Create context with timeout for the entire benchmark application
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Use the existing docker-compose setup
	// The docker-compose.yaml file should be running with the schema.sql already applied
	dsn := "postgres://postgres:postgres@localhost:5432/dcb_app?sslmode=disable"

	// Wait for database to be ready
	maxRetries := 30
	for i := 0; i < maxRetries; i++ {
		// Configure connection pool for performance (matching web app)
		config, err := pgxpool.ParseConfig(dsn)
		if err != nil {
			log.Fatalf("Failed to parse database URL: %v", err)
		}

		// Use same pool configuration as web app for fair benchmarking
		config.MaxConns = 50                      // Reduced from 300 to prevent exhaustion
		config.MinConns = 10                      // Reduced from 100 to prevent exhaustion
		config.MaxConnLifetime = 10 * time.Minute // Reduced from 15 minutes
		config.MaxConnIdleTime = 5 * time.Minute  // Reduced from 10 minutes
		config.HealthCheckPeriod = 30 * time.Second

		pool, err := pgxpool.NewWithConfig(ctx, config)
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

	// Connect to database with same pool configuration
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		log.Fatalf("Failed to parse database URL: %v", err)
	}

	// Use same pool configuration as web app for fair benchmarking
	config.MaxConns = 50                      // Reduced from 300 to prevent exhaustion
	config.MinConns = 10                      // Reduced from 100 to prevent exhaustion
	config.MaxConnLifetime = 10 * time.Minute // Reduced from 15 minutes
	config.MaxConnIdleTime = 5 * time.Minute  // Reduced from 10 minutes
	config.HealthCheckPeriod = 30 * time.Second

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	// Create event store
	store, err := dcb.NewEventStore(ctx, pool)
	if err != nil {
		log.Fatalf("Failed to create event store: %v", err)
	}

	// Check if EventStore is available
	channelStore, hasChannel := store.(dcb.EventStore)

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

func runStreamBenchmarks(ctx context.Context, store dcb.EventStore, channelStore dcb.EventStore, hasChannel bool) {
	fmt.Println("--- Streaming Benchmarks ---")

	// Iterator vs Channel comparison
	if hasChannel {
		benchmarkIteratorVsChannel(ctx, store, channelStore)
	}

	// Memory usage comparison
	benchmarkMemoryUsage(ctx, store, channelStore, hasChannel)

	fmt.Println()
}

func runProjectionBenchmarks(ctx context.Context, store dcb.EventStore, channelStore dcb.EventStore, hasChannel bool) {
	fmt.Println("--- Projection Benchmarks ---")

	// Single projector
	benchmarkSingleProjector(ctx, store, channelStore)

	// Multiple projectors
	benchmarkMultipleProjectors(ctx, store, channelStore)

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
	err := store.Append(ctx, []dcb.InputEvent{event})
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
		err := store.Append(ctx, events)
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
				err := store.Append(ctx, events)
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

	// Append course events in batches
	const batchSize = 1000
	for i := 0; i < courses; i += batchSize {
		end := i + batchSize
		if end > courses {
			end = courses
		}

		courseEvents := make([]dcb.InputEvent, end-i)
		for j := 0; j < end-i; j++ {
			courseID := fmt.Sprintf("course-%d", i+j)
			courseEvents[j] = dcb.NewInputEvent("CourseDefined", dcb.NewTags("course_id", courseID), []byte(fmt.Sprintf(`{"courseId": "%s", "name": "Course %d", "capacity": 100, "instructor": "Instructor %d"}`, courseID, i+j, i+j)))
		}

		err := store.Append(ctx, courseEvents)
		if err != nil {
			fmt.Printf("Error creating courses batch %d-%d: %v\n", i, end-1, err)
			return
		}
	}

	// Append student events in batches
	for i := 0; i < students; i += batchSize {
		end := i + batchSize
		if end > students {
			end = students
		}

		studentEvents := make([]dcb.InputEvent, end-i)
		for j := 0; j < end-i; j++ {
			studentID := fmt.Sprintf("student-%d", i+j)
			studentEvents[j] = dcb.NewInputEvent("StudentRegistered", dcb.NewTags("student_id", studentID), []byte(fmt.Sprintf(`{"studentId": "%s", "name": "Student %d", "email": "student%d@example.com"}`, studentID, i+j, i+j)))
		}

		err := store.Append(ctx, studentEvents)
		if err != nil {
			fmt.Printf("Error creating students batch %d-%d: %v\n", i, end-1, err)
			return
		}
	}

	// Append enrollment events in batches
	for i := 0; i < enrollments; i += batchSize {
		end := i + batchSize
		if end > enrollments {
			end = enrollments
		}

		enrollmentEvents := make([]dcb.InputEvent, end-i)
		for j := 0; j < end-i; j++ {
			studentID := fmt.Sprintf("student-%d", (i+j)%students)
			courseID := fmt.Sprintf("course-%d", (i+j)%courses)
			enrollmentEvents[j] = dcb.NewInputEvent("StudentEnrolledInCourse", dcb.NewTags("student_id", studentID, "course_id", courseID), []byte(fmt.Sprintf(`{"studentId": "%s", "courseId": "%s", "enrolledAt": "2024-01-01"}`, studentID, courseID)))
		}

		err := store.Append(ctx, enrollmentEvents)
		if err != nil {
			fmt.Printf("Error creating enrollments batch %d-%d: %v\n", i, end-1, err)
			return
		}
	}

	fmt.Printf("Created %d courses, %d students, %d enrollments\n", courses, students, enrollments)
}

func benchmarkSimpleQueries(ctx context.Context, store dcb.EventStore) {
	fmt.Println("Simple Queries:")

	// Query courses by category (DCB-focused: specific category instead of all courses)
	start := time.Now()
	query := dcb.NewQuery(dcb.NewTags("category", "Computer Science"), "CourseDefined")
	events, err := store.Read(ctx, query)
	duration := time.Since(start)

	if err != nil {
		fmt.Printf("  Courses by category: Error: %v\n", err)
	} else {
		fmt.Printf("  Courses by category: %v (%d events)\n", duration, len(events))
	}

	// Query by specific course ID (DCB-focused: targeted query)
	start = time.Now()
	query = dcb.NewQuery(dcb.NewTags("course_id", "course-1"), "CourseDefined")
	events, err = store.Read(ctx, query)
	duration = time.Since(start)

	if err != nil {
		fmt.Printf("  Course by ID: Error: %v\n", err)
	} else {
		fmt.Printf("  Course by ID: %v (%d events)\n", duration, len(events))
	}
}

func benchmarkComplexQueries(ctx context.Context, store dcb.EventStore) {
	fmt.Println("Complex Queries:")

	// OR query with specific tags (DCB-focused: targeted cross-entity query)
	start := time.Now()
	query := dcb.NewQueryFromItems(
		dcb.NewQueryItem([]string{"CourseDefined"}, dcb.NewTags("course_id", "course-1")),
		dcb.NewQueryItem([]string{"StudentRegistered"}, dcb.NewTags("student_id", "student-1")),
	)
	events, err := store.Read(ctx, query)
	duration := time.Since(start)

	if err != nil {
		fmt.Printf("  OR query: Error: %v\n", err)
	} else {
		fmt.Printf("  OR query: %v (%d events)\n", duration, len(events))
	}

	// Query enrollments by grade (DCB-focused: specific grade instead of all enrollments)
	start = time.Now()
	limit := 100
	options := &dcb.ReadOptions{Limit: &limit}
	query = dcb.NewQuery(dcb.NewTags("grade", "A"), "StudentEnrolledInCourse")
	events, err = store.ReadWithOptions(ctx, query, options)
	duration = time.Since(start)

	if err != nil {
		fmt.Printf("  ReadWithOptions: Error: %v\n", err)
	} else {
		fmt.Printf("  ReadWithOptions: %v (%d events)\n", duration, len(events))
	}
}

func benchmarkIteratorVsChannel(ctx context.Context, store dcb.EventStore, channelStore dcb.EventStore) {
	fmt.Println("Iterator vs Channel:")

	// DCB-focused query: specific student's enrollments instead of all enrollments
	query := dcb.NewQuery(dcb.NewTags("student_id", "student-1"), "StudentEnrolledInCourse")

	// ReadStream has been removed - use Read for batch reading instead
	fmt.Println("  Iterator: ReadStream method has been removed - use Read for batch operations")

	// Channel approach
	start := time.Now()
	eventChan, _, err := channelStore.ReadStreamChannel(ctx, query)
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

func benchmarkMemoryUsage(ctx context.Context, store dcb.EventStore, channelStore dcb.EventStore, hasChannel bool) {
	fmt.Println("Memory Usage Comparison:")

	// This would require runtime.ReadMemStats() for actual memory measurement
	// For now, just show the approach
	fmt.Println("  (Memory measurement requires runtime.ReadMemStats())")
}

func benchmarkSingleProjector(ctx context.Context, store dcb.EventStore, channelStore dcb.EventStore) {
	fmt.Println("Single Projector:")

	// DCB-focused projector: count courses in specific category instead of all courses
	projector := dcb.BatchProjector{
		ID: "csCourseCount",
		StateProjector: dcb.StateProjector{
			Query:        dcb.NewQuery(dcb.NewTags("category", "Computer Science"), "CourseDefined"),
			InitialState: 0,
			TransitionFn: func(state any, event dcb.Event) any {
				return state.(int) + 1
			},
		},
	}

	start := time.Now()
	states, _, err := channelStore.ProjectDecisionModel(ctx, []dcb.BatchProjector{projector})
	duration := time.Since(start)

	if err != nil {
		fmt.Printf("  Error: %v\n", err)
	} else {
		fmt.Printf("  Duration: %v (count: %d)\n", duration, states["csCourseCount"])
	}
}

func benchmarkMultipleProjectors(ctx context.Context, store dcb.EventStore, channelStore dcb.EventStore) {
	fmt.Println("Multiple Projectors:")

	// DCB-focused projectors: specific targeted queries instead of full scans
	projectors := []dcb.BatchProjector{
		{
			ID: "csCourseCount",
			StateProjector: dcb.StateProjector{
				Query:        dcb.NewQuery(dcb.NewTags("category", "Computer Science"), "CourseDefined"),
				InitialState: 0,
				TransitionFn: func(state any, event dcb.Event) any {
					return state.(int) + 1
				},
			},
		},
		{
			ID: "csStudentCount",
			StateProjector: dcb.StateProjector{
				Query:        dcb.NewQuery(dcb.NewTags("major", "Computer Science"), "StudentRegistered"),
				InitialState: 0,
				TransitionFn: func(state any, event dcb.Event) any {
					return state.(int) + 1
				},
			},
		},
		{
			ID: "aGradeEnrollments",
			StateProjector: dcb.StateProjector{
				Query:        dcb.NewQuery(dcb.NewTags("grade", "A"), "StudentEnrolledInCourse"),
				InitialState: 0,
				TransitionFn: func(state any, event dcb.Event) any {
					return state.(int) + 1
				},
			},
		},
	}

	start := time.Now()
	states, _, err := channelStore.ProjectDecisionModel(ctx, projectors)
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

func benchmarkChannelProjection(ctx context.Context, channelStore dcb.EventStore) {
	fmt.Println("Channel Projection:")

	// DCB-focused projector: specific category instead of all courses
	projectors := []dcb.BatchProjector{
		{
			ID: "csCourseCount",
			StateProjector: dcb.StateProjector{
				Query:        dcb.NewQuery(dcb.NewTags("category", "Computer Science"), "CourseDefined"),
				InitialState: 0,
				TransitionFn: func(state any, event dcb.Event) any {
					return state.(int) + 1
				},
			},
		},
	}

	start := time.Now()
	resultChan, _, err := channelStore.ProjectDecisionModelChannel(ctx, projectors)
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
