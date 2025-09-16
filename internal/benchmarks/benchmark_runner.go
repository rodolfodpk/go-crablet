package benchmarks

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/rodolfodpk/go-crablet/internal/benchmarks/setup"
	"github.com/rodolfodpk/go-crablet/pkg/dcb"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Global shared pool for all benchmarks
var (
	globalPool     *pgxpool.Pool
	globalPoolOnce sync.Once
	globalPoolMu   sync.RWMutex
)

// Global progress tracking for all benchmarks
var (
	globalProgressMu          sync.Mutex
	globalCompletedBenchmarks int = 0
	globalStartTime           time.Time
	globalInitialized         bool = false
)

// ProgressTracker provides progress information for benchmark execution
type ProgressTracker struct {
	mu             sync.Mutex
	total          int
	completed      int
	startTime      time.Time
	lastUpdateTime time.Time
	isGlobal       bool
}

// NewProgressTracker creates a new progress tracker
func NewProgressTracker(total int) *ProgressTracker {
	return &ProgressTracker{
		total:          total,
		completed:      0,
		startTime:      time.Now(),
		lastUpdateTime: time.Now(),
		isGlobal:       false,
	}
}

// NewGlobalProgressTracker creates a global progress tracker for all datasets
func NewGlobalProgressTracker() *ProgressTracker {
	globalProgressMu.Lock()
	defer globalProgressMu.Unlock()

	if !globalInitialized {
		globalStartTime = time.Now()
		globalCompletedBenchmarks = 0
		globalInitialized = true
		fmt.Printf("[START] Running all benchmarks\n")
	}

	return &ProgressTracker{
		total:          0, // Don't use total for global tracker
		completed:      0,
		startTime:      globalStartTime,
		lastUpdateTime: time.Now(),
		isGlobal:       true,
	}
}

// Update increments the completed count and prints progress
func (pt *ProgressTracker) Update(benchmarkName string) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	pt.completed++

	if pt.isGlobal {
		globalProgressMu.Lock()
		globalCompletedBenchmarks++
		elapsed := time.Since(globalStartTime)
		globalProgressMu.Unlock()

		// Always print progress after each benchmark execution (without percentage)
		fmt.Printf("[PROGRESS] %s completed (%d total) - Elapsed: %s\n",
			benchmarkName, globalCompletedBenchmarks, elapsed.Round(time.Second))
		pt.lastUpdateTime = time.Now()
	} else {
		percentage := float64(pt.completed) / float64(pt.total) * 100
		elapsed := time.Since(pt.startTime)

		// Only print progress every 30 seconds or when percentage changes significantly
		if time.Since(pt.lastUpdateTime) > 30*time.Second || pt.completed == pt.total {
			fmt.Printf("[PROGRESS] %s - %d/%d (%.1f%%) - Elapsed: %s\n",
				benchmarkName, pt.completed, pt.total, percentage, elapsed.Round(time.Second))
			pt.lastUpdateTime = time.Now()
		}
	}
}

// Complete prints final progress information
func (pt *ProgressTracker) Complete() {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	if pt.isGlobal {
		globalProgressMu.Lock()
		if globalCompletedBenchmarks > 0 {
			totalTime := time.Since(globalStartTime)
			fmt.Printf("[COMPLETE] All benchmarks finished - Total time: %s\n", totalTime.Round(time.Second))
			// Reset for next run
			globalInitialized = false
			globalCompletedBenchmarks = 0
		}
		globalProgressMu.Unlock()
	} else {
		totalTime := time.Since(pt.startTime)
		fmt.Printf("[COMPLETE] All benchmarks finished - Total time: %s\n", totalTime.Round(time.Second))
	}
}

// BenchmarkContext holds the context for running benchmarks
type BenchmarkContext struct {
	Store        dcb.EventStore
	ChannelStore dcb.EventStore
	HasChannel   bool
	Dataset      *setup.Dataset
}

// getOrCreateGlobalPool returns the shared global pool, creating it if necessary
func getOrCreateGlobalPool() (*pgxpool.Pool, error) {
	globalPoolMu.RLock()
	if globalPool != nil {
		defer globalPoolMu.RUnlock()
		return globalPool, nil
	}
	globalPoolMu.RUnlock()

	globalPoolMu.Lock()
	defer globalPoolMu.Unlock()

	// Double-check after acquiring write lock
	if globalPool != nil {
		return globalPool, nil
	}

	// Get database URL from environment or use default
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://crablet:crablet@localhost:5432/crablet"
	}

	// Create new pool with conservative settings
	poolConfig, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse connection string: %v", err)
	}

	// Configure pool for performance benchmarking (matching web app)
	poolConfig.MaxConns = 50                      // Match web app performance
	poolConfig.MinConns = 10                      // Keep connections ready
	poolConfig.MaxConnLifetime = 10 * time.Minute // Longer connection life
	poolConfig.MaxConnIdleTime = 5 * time.Minute  // Better connection reuse

	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create global pool: %v", err)
	}

	globalPool = pool
	return globalPool, nil
}

// SetupProjectionBenchmarkContext creates a clean context for projection benchmarks
// This ensures projection benchmarks run against only the events created by Append benchmarks
func SetupProjectionBenchmarkContext(b *testing.B, datasetSize string) *BenchmarkContext {
	ctx := context.Background()

	// Use the shared global pool
	pool, err := getOrCreateGlobalPool()
	if err != nil {
		b.Fatalf("Failed to get global pool: %v", err)
	}

	// Create event stores with different configurations
	readCommittedConfig := dcb.EventStoreConfig{
		MaxAppendBatchSize:     1000,
		StreamBuffer:           1000,
		DefaultAppendIsolation: dcb.IsolationLevelReadCommitted,
		QueryTimeout:           15000,
		AppendTimeout:          15000,
	}

	repeatableReadConfig := dcb.EventStoreConfig{
		MaxAppendBatchSize:     1000,
		StreamBuffer:           1000,
		DefaultAppendIsolation: dcb.IsolationLevelRepeatableRead,
		QueryTimeout:           15000,
		AppendTimeout:          15000,
	}

	store, err := dcb.NewEventStoreWithConfig(ctx, pool, readCommittedConfig)
	if err != nil {
		b.Fatalf("Failed to create event store: %v", err)
	}

	channelStore, err := dcb.NewEventStoreWithConfig(ctx, pool, repeatableReadConfig)
	if err != nil {
		b.Fatalf("Failed to create channel event store: %v", err)
	}

	return &BenchmarkContext{
		Store:        store,
		ChannelStore: channelStore,
		HasChannel:   true,
		Dataset:      nil, // No dataset for projection benchmarks
	}
}

// SetupBenchmarkContext creates a new benchmark context with the specified dataset size
// pastEventCount specifies how many past events to create for AppendIf testing (1, 10, 100, etc.)
func SetupBenchmarkContext(b *testing.B, datasetSize string, pastEventCount int) *BenchmarkContext {
	ctx := context.Background()

	// Use the shared global pool
	pool, err := getOrCreateGlobalPool()
	if err != nil {
		b.Fatalf("Failed to get global pool: %v", err)
	}

	// Create event stores with different configurations
	readCommittedConfig := dcb.EventStoreConfig{
		MaxAppendBatchSize:     1000,
		StreamBuffer:           1000,
		DefaultAppendIsolation: dcb.IsolationLevelReadCommitted,
		QueryTimeout:           15000,
		AppendTimeout:          15000,
	}

	repeatableReadConfig := dcb.EventStoreConfig{
		MaxAppendBatchSize:     1000,
		StreamBuffer:           1000,
		DefaultAppendIsolation: dcb.IsolationLevelRepeatableRead,
		QueryTimeout:           15000,
		AppendTimeout:          15000,
	}

	store, err := dcb.NewEventStoreWithConfig(ctx, pool, readCommittedConfig)
	if err != nil {
		b.Fatalf("Failed to create event store: %v", err)
	}

	channelStore, err := dcb.NewEventStoreWithConfig(ctx, pool, repeatableReadConfig)
	if err != nil {
		b.Fatalf("Failed to create channel event store: %v", err)
	}

	// Get dataset configuration
	config, exists := setup.DatasetSizes[datasetSize]
	if !exists {
		b.Fatalf("Unknown dataset size: %s", datasetSize)
	}

	// Generate dataset for benchmarking
	dataset := setup.GenerateDataset(config)

	// Load dataset into PostgreSQL for realistic benchmarking
	if err := setup.LoadDatasetIntoStore(ctx, store, dataset); err != nil {
		b.Fatalf("Failed to load dataset into store: %v", err)
	}

	// Create past events for AppendIf testing (business rule validation context)
	if pastEventCount > 0 {
		if err := createPastEventsForAppendIf(ctx, store, pastEventCount); err != nil {
			b.Fatalf("Failed to create past events for AppendIf testing: %v", err)
		}
	}

	return &BenchmarkContext{
		Store:        store,
		ChannelStore: channelStore,
		HasChannel:   true,
		Dataset:      dataset,
	}
}

// createPastEventsForAppendIf creates a controlled number of past events for AppendIf testing
// This ensures consistent business rule validation context across benchmark runs
func createPastEventsForAppendIf(ctx context.Context, store dcb.EventStore, count int) error {
	events := make([]dcb.InputEvent, count)

	for i := 0; i < count; i++ {
		eventID := fmt.Sprintf("past_event_%d", i)
		events[i] = dcb.NewInputEvent("PastEvent",
			dcb.NewTags("test", "past", "event_id", eventID),
			[]byte(fmt.Sprintf(`{"value": "past", "event_id": "%s", "index": %d}`, eventID, i)))
	}

	return store.Append(ctx, events)
}

// BenchmarkAppendIfConcurrent benchmarks concurrent AppendIf operations
func BenchmarkAppendIfConcurrent(b *testing.B, benchCtx *BenchmarkContext, concurrencyLevel int, eventCount int, conflictScenario bool) {
	// Create context with timeout for each benchmark iteration using Go 1.25 WithTimeoutCause
	ctx, cancel := context.WithTimeoutCause(context.Background(), 2*time.Minute,
		fmt.Errorf("benchmark timeout after 2 minutes"))
	defer cancel()

	// Truncate events table BEFORE timing starts (but after dataset is loaded)
	pool, err := getOrCreateGlobalPool()
	if err != nil {
		b.Fatalf("Failed to get global pool: %v", err)
	}

	_, err = pool.Exec(ctx, "TRUNCATE TABLE events RESTART IDENTITY CASCADE")
	if err != nil {
		b.Fatalf("Failed to truncate events table: %v", err)
	}

	// WARM-UP PHASE: Run the exact same logic we'll benchmark without timing
	warmupBenchmark(func() {
		// For conflict scenarios, create exactly 1 conflicting event
		if conflictScenario {
			uniqueID := fmt.Sprintf("warmup_conflict_%d", time.Now().UnixNano())
			conflictEvent := dcb.NewInputEvent("WarmupConflictingEvent",
				dcb.NewTags("test", "warmup", "unique_id", uniqueID),
				[]byte(fmt.Sprintf(`{"value": "warmup", "unique_id": "%s"}`, uniqueID)))

			err := benchCtx.Store.Append(ctx, []dcb.InputEvent{conflictEvent})
			if err != nil {
				// Log but don't fail during warm-up
				return
			}
		}

		var wg sync.WaitGroup
		results := make(chan error, concurrencyLevel)

		// Launch concurrent AppendIf operations using WaitGroup.Go()
		for j := 0; j < concurrencyLevel; j++ {
			goroutineID := j // Capture loop variable
			wg.Go(func() {
				// Create unique ID for this goroutine
				uniqueID := fmt.Sprintf("warmup_%d_%d", time.Now().UnixNano(), goroutineID)

				// Create events to append
				events := make([]dcb.InputEvent, eventCount)
				for k := 0; k < eventCount; k++ {
					eventID := fmt.Sprintf("warmup_%s_%d", uniqueID, k)
					events[k] = dcb.NewInputEvent("WarmupEvent",
						dcb.NewTags("test", "warmup", "unique_id", eventID),
						[]byte(fmt.Sprintf(`{"value": "warmup", "unique_id": "%s"}`, eventID)))
				}

				// Create condition
				var condition dcb.AppendCondition
				if conflictScenario {
					// Condition that should fail (conflicting event exists)
					condition = dcb.NewAppendCondition(
						dcb.NewQuery(dcb.NewTags("test", "conflict")),
					)
				} else {
					// Condition that should pass (no conflicting event)
					condition = dcb.NewAppendCondition(
						dcb.NewQuery(dcb.NewTags("test", "noconflict", "unique_id", uniqueID)),
					)
				}

				// Execute AppendIf
				err := benchCtx.Store.AppendIf(ctx, events, condition)
				if err != nil {
					results <- fmt.Errorf("warmup appendif failed: %v", err)
					return
				}

				results <- nil
			})
		}

		wg.Wait()
		close(results)

		// Check for errors but don't fail on warm-up
		for err := range results {
			if err != nil {
				// Log but don't fail during warm-up
				continue
			}
		}
	})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// For conflict scenarios, create exactly 1 conflicting event
		if conflictScenario {
			uniqueID := fmt.Sprintf("conflict_%d_%d", time.Now().UnixNano(), i)
			conflictEvent := dcb.NewInputEvent("ConflictingEvent",
				dcb.NewTags("test", "conflict", "unique_id", uniqueID),
				[]byte(fmt.Sprintf(`{"value": "test", "unique_id": "%s"}`, uniqueID)))

			err := benchCtx.Store.Append(ctx, []dcb.InputEvent{conflictEvent})
			if err != nil {
				b.Fatalf("Failed to append conflict event: %v", err)
			}
		}

		// Use Go 1.25 WaitGroup.Go() for concurrent operations
		var wg sync.WaitGroup
		results := make(chan error, concurrencyLevel)

		// Launch concurrent AppendIf operations using WaitGroup.Go()
		for j := 0; j < concurrencyLevel; j++ {
			goroutineID := j // Capture loop variable
			wg.Go(func() {
				// Create unique ID for this goroutine
				uniqueID := fmt.Sprintf("concurrent_%d_%d_%d", time.Now().UnixNano(), i, goroutineID)

				// Create events to append
				events := make([]dcb.InputEvent, eventCount)
				for k := 0; k < eventCount; k++ {
					eventID := fmt.Sprintf("appendif_%s_%d", uniqueID, k)
					events[k] = dcb.NewInputEvent("TestEvent",
						dcb.NewTags("test", "appendif", "unique_id", eventID),
						[]byte(fmt.Sprintf(`{"value": "test", "unique_id": "%s"}`, eventID)))
				}

				// Create condition
				var condition dcb.AppendCondition
				if conflictScenario {
					// Condition that should fail (conflicting event exists)
					condition = dcb.NewAppendCondition(
						dcb.NewQuery(dcb.NewTags("test", "conflict")),
					)
				} else {
					// Condition that should pass (no conflicting event)
					condition = dcb.NewAppendCondition(
						dcb.NewQuery(dcb.NewTags("test", "noconflict", "unique_id", uniqueID)),
					)
				}

				// Execute AppendIf
				err := benchCtx.Store.AppendIf(ctx, events, condition)
				if conflictScenario && err == nil {
					results <- fmt.Errorf("AppendIf should have failed due to conflict")
					return
				} else if !conflictScenario && err != nil {
					results <- fmt.Errorf("AppendIf should have succeeded: %v", err)
					return
				}

				results <- nil
			})
		}

		// Wait for all goroutines to complete
		wg.Wait()
		close(results)

		// Check for any errors
		for err := range results {
			if err != nil {
				b.Fatalf("Concurrent AppendIf failed: %v", err)
			}
		}
	}
}

// BenchmarkAppendIfConcurrentRealistic benchmarks concurrent AppendIf operations using realistic dataset events
// Uses CourseOffered, StudentRegistered, and EnrollmentCompleted events with realistic business conditions
func BenchmarkAppendIfConcurrentRealistic(b *testing.B, benchCtx *BenchmarkContext, concurrencyLevel int, eventCount int, conflictScenario bool) {
	// Create context with timeout for each benchmark iteration using Go 1.25 WithTimeoutCause
	ctx, cancel := context.WithTimeoutCause(context.Background(), 2*time.Minute,
		fmt.Errorf("benchmark timeout after 2 minutes"))
	defer cancel()

	// Truncate events table BEFORE timing starts (but after dataset is loaded)
	pool, err := getOrCreateGlobalPool()
	if err != nil {
		b.Fatalf("Failed to get global pool: %v", err)
	}

	_, err = pool.Exec(ctx, "TRUNCATE TABLE events RESTART IDENTITY CASCADE")
	if err != nil {
		b.Fatalf("Failed to truncate events table: %v", err)
	}

	// WARM-UP PHASE: Run the exact same logic we'll benchmark without timing
	warmupBenchmark(func() {
		// For conflict scenarios, create exactly 1 conflicting event
		if conflictScenario {
			uniqueID := fmt.Sprintf("warmup_conflict_%d", time.Now().UnixNano())
			conflictEvent := dcb.NewInputEvent("CourseOffered",
				dcb.NewTags("course_id", uniqueID, "category", "Computer Science"),
				[]byte(fmt.Sprintf(`{
					"id": "%s",
					"name": "Conflicting Course",
					"instructor": "Dr. Conflict",
					"capacity": 50,
					"category": "Computer Science",
					"popularity": 0.5
				}`, uniqueID)))

			err := benchCtx.Store.Append(ctx, []dcb.InputEvent{conflictEvent})
			if err != nil {
				// Log but don't fail during warm-up
				return
			}
		}

		var wg sync.WaitGroup
		results := make(chan error, concurrencyLevel)

		// Launch concurrent AppendIf operations using WaitGroup.Go()
		for j := 0; j < concurrencyLevel; j++ {
			goroutineID := j // Capture loop variable
			wg.Go(func() {
				// Create unique ID for this goroutine
				uniqueID := fmt.Sprintf("warmup_%d_%d", time.Now().UnixNano(), goroutineID)

				// Create realistic events to append based on eventCount
				events := make([]dcb.InputEvent, eventCount)
				for k := 0; k < eventCount; k++ {
					eventID := fmt.Sprintf("warmup_%s_%d", uniqueID, k)

					// Alternate between different realistic event types
					switch k % 3 {
					case 0: // CourseOffered
						events[k] = dcb.NewInputEvent("CourseOffered",
							dcb.NewTags("course_id", eventID, "category", "Computer Science"),
							[]byte(fmt.Sprintf(`{
								"id": "%s",
								"name": "Advanced Programming",
								"instructor": "Dr. Smith",
								"capacity": 50,
								"category": "Computer Science",
								"popularity": 0.8
							}`, eventID)))
					case 1: // StudentRegistered
						events[k] = dcb.NewInputEvent("StudentRegistered",
							dcb.NewTags("student_id", eventID, "major", "Computer Science"),
							[]byte(fmt.Sprintf(`{
								"id": "%s",
								"name": "John Doe",
								"email": "john.doe@university.edu",
								"major": "Computer Science",
								"year": 3,
								"maxCourses": 5
							}`, eventID)))
					case 2: // EnrollmentCompleted
						events[k] = dcb.NewInputEvent("EnrollmentCompleted",
							dcb.NewTags("student_id", eventID, "course_id", fmt.Sprintf("course_%d", k)),
							[]byte(fmt.Sprintf(`{
								"studentId": "%s",
								"courseId": "course_%d",
								"enrolledAt": "%s",
								"grade": ""
							}`, eventID, k, time.Now().Format(time.RFC3339))))
					}
				}

				// Create realistic condition based on scenario
				var condition dcb.AppendCondition
				if conflictScenario {
					// Condition that should fail (conflicting course exists)
					condition = dcb.NewAppendCondition(
						dcb.NewQuery(dcb.NewTags("course_id", uniqueID, "category", "Computer Science")),
					)
				} else {
					// Condition that should pass (no conflicting course)
					condition = dcb.NewAppendCondition(
						dcb.NewQuery(dcb.NewTags("course_id", fmt.Sprintf("noconflict_%s", uniqueID), "category", "Computer Science")),
					)
				}

				// Execute AppendIf
				err := benchCtx.Store.AppendIf(ctx, events, condition)
				if err != nil {
					results <- fmt.Errorf("warmup realistic appendif failed: %v", err)
					return
				}

				results <- nil
			})
		}

		wg.Wait()
		close(results)

		// Check for errors but don't fail on warm-up
		for err := range results {
			if err != nil {
				// Log but don't fail during warm-up
				continue
			}
		}
	})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// For conflict scenarios, create exactly 1 conflicting event
		if conflictScenario {
			uniqueID := fmt.Sprintf("conflict_%d_%d", time.Now().UnixNano(), i)
			conflictEvent := dcb.NewInputEvent("CourseOffered",
				dcb.NewTags("course_id", uniqueID, "category", "Computer Science"),
				[]byte(fmt.Sprintf(`{
					"id": "%s",
					"name": "Conflicting Course",
					"instructor": "Dr. Conflict",
					"capacity": 50,
					"category": "Computer Science",
					"popularity": 0.5
				}`, uniqueID)))

			err := benchCtx.Store.Append(ctx, []dcb.InputEvent{conflictEvent})
			if err != nil {
				b.Fatalf("Failed to create conflicting event: %v", err)
			}
		}

		var wg sync.WaitGroup
		results := make(chan error, concurrencyLevel)

		// Launch concurrent AppendIf operations using WaitGroup.Go()
		for j := 0; j < concurrencyLevel; j++ {
			goroutineID := j // Capture loop variable
			wg.Go(func() {
				// Create unique ID for this goroutine
				uniqueID := fmt.Sprintf("realistic_%d_%d", time.Now().UnixNano(), goroutineID)

				// Create realistic events to append based on eventCount
				events := make([]dcb.InputEvent, eventCount)
				for k := 0; k < eventCount; k++ {
					eventID := fmt.Sprintf("realistic_%s_%d", uniqueID, k)

					// Alternate between different realistic event types
					switch k % 3 {
					case 0: // CourseOffered
						events[k] = dcb.NewInputEvent("CourseOffered",
							dcb.NewTags("course_id", eventID, "category", "Computer Science"),
							[]byte(fmt.Sprintf(`{
								"id": "%s",
								"name": "Advanced Programming",
								"instructor": "Dr. Smith",
								"capacity": 50,
								"category": "Computer Science",
								"popularity": 0.8
							}`, eventID)))
					case 1: // StudentRegistered
						events[k] = dcb.NewInputEvent("StudentRegistered",
							dcb.NewTags("student_id", eventID, "major", "Computer Science"),
							[]byte(fmt.Sprintf(`{
								"id": "%s",
								"name": "John Doe",
								"email": "john.doe@university.edu",
								"major": "Computer Science",
								"year": 3,
								"maxCourses": 5
							}`, eventID)))
					case 2: // EnrollmentCompleted
						events[k] = dcb.NewInputEvent("EnrollmentCompleted",
							dcb.NewTags("student_id", eventID, "course_id", fmt.Sprintf("course_%d", k)),
							[]byte(fmt.Sprintf(`{
								"studentId": "%s",
								"courseId": "course_%d",
								"enrolledAt": "%s",
								"grade": ""
							}`, eventID, k, time.Now().Format(time.RFC3339))))
					}
				}

				// Create realistic condition based on scenario
				var condition dcb.AppendCondition
				if conflictScenario {
					// Condition that should fail (conflicting course exists)
					condition = dcb.NewAppendCondition(
						dcb.NewQuery(dcb.NewTags("course_id", uniqueID, "category", "Computer Science")),
					)
				} else {
					// Condition that should pass (no conflicting course)
					condition = dcb.NewAppendCondition(
						dcb.NewQuery(dcb.NewTags("course_id", fmt.Sprintf("noconflict_%s", uniqueID), "category", "Computer Science")),
					)
				}

				// Execute AppendIf
				err := benchCtx.Store.AppendIf(ctx, events, condition)
				if conflictScenario && err != nil {
					// Expected to fail in conflict scenario
					results <- nil
					return
				} else if !conflictScenario && err != nil {
					results <- fmt.Errorf("AppendIf should have succeeded: %v", err)
					return
				}

				results <- nil
			})
		}

		// Wait for all goroutines to complete
		wg.Wait()
		close(results)

		// Check for any errors
		for err := range results {
			if err != nil {
				b.Fatalf("Concurrent realistic AppendIf failed: %v", err)
			}
		}
	}
}

// warmupBenchmark runs warm-up iterations without timing to ensure consistent performance
func warmupBenchmark(warmupFunc func()) {
	// Run warm-up iterations to warm up JIT compiler, CPU caches, memory allocators, and database connections
	warmupIterations := 3
	for i := 0; i < warmupIterations; i++ {
		warmupFunc()
	}
}

// warmupDatabaseQueries warms up PostgreSQL query plans and connection pool
func warmupDatabaseQueries(ctx context.Context, store dcb.EventStore) {
	// Warm up Append operations
	for i := 0; i < 3; i++ {
		events := []dcb.InputEvent{
			dcb.NewInputEvent("WarmupEvent", dcb.NewTags("warmup", "true"), []byte(`{"warmup": true}`)),
		}
		store.Append(ctx, events) // This builds query plans for INSERT
	}

	// Warm up Query operations
	for i := 0; i < 3; i++ {
		query := dcb.NewQuery(dcb.NewTags("warmup", "true"), "WarmupEvent")
		store.Query(ctx, query, nil) // This builds query plans for SELECT
	}
}
func BenchmarkAppendConcurrent(b *testing.B, benchCtx *BenchmarkContext, concurrencyLevel int, eventCount int) {
	// Create context with timeout for each benchmark iteration using Go 1.25 WithTimeoutCause
	ctx, cancel := context.WithTimeoutCause(context.Background(), 2*time.Minute,
		fmt.Errorf("benchmark timeout after 2 minutes"))
	defer cancel()

	// Truncate events table BEFORE timing starts (but after dataset is loaded)
	pool, err := getOrCreateGlobalPool()
	if err != nil {
		b.Fatalf("Failed to get global pool: %v", err)
	}

	_, err = pool.Exec(ctx, "TRUNCATE TABLE events RESTART IDENTITY CASCADE")
	if err != nil {
		b.Fatalf("Failed to truncate events table: %v", err)
	}

	// WARM-UP PHASE: Run the exact same logic we'll benchmark without timing
	warmupBenchmark(func() {
		var wg sync.WaitGroup
		results := make(chan error, concurrencyLevel)

		// Launch concurrent Append operations using WaitGroup.Go()
		for j := 0; j < concurrencyLevel; j++ {
			goroutineID := j // Capture loop variable
			wg.Go(func() {
				// Create unique ID for this goroutine
				uniqueID := fmt.Sprintf("warmup_%d_%d", time.Now().UnixNano(), goroutineID)

				// Create multiple events to append
				events := make([]dcb.InputEvent, eventCount)
				for k := 0; k < eventCount; k++ {
					eventID := fmt.Sprintf("warmup_%s_%d", uniqueID, k)
					events[k] = dcb.NewInputEvent("WarmupEvent",
						dcb.NewTags("test", "warmup", "unique_id", eventID),
						[]byte(fmt.Sprintf(`{"value": "warmup", "unique_id": "%s"}`, eventID)))
				}

				// Execute Append
				err := benchCtx.Store.Append(ctx, events)
				if err != nil {
					results <- fmt.Errorf("warmup append failed: %v", err)
					return
				}

				results <- nil
			})
		}

		wg.Wait()
		close(results)

		// Check for errors but don't fail on warm-up
		for err := range results {
			if err != nil {
				// Log but don't fail during warm-up
				continue
			}
		}
	})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Use Go 1.25 WaitGroup.Go() for concurrent operations
		var wg sync.WaitGroup
		results := make(chan error, concurrencyLevel)

		// Launch concurrent Append operations using WaitGroup.Go()
		for j := 0; j < concurrencyLevel; j++ {
			goroutineID := j // Capture loop variable
			wg.Go(func() {
				// Create unique ID for this goroutine
				uniqueID := fmt.Sprintf("concurrent_%d_%d_%d", time.Now().UnixNano(), i, goroutineID)

				// Create multiple events to append
				events := make([]dcb.InputEvent, eventCount)
				for k := 0; k < eventCount; k++ {
					eventID := fmt.Sprintf("append_%s_%d", uniqueID, k)
					events[k] = dcb.NewInputEvent("TestEvent",
						dcb.NewTags("test", "concurrent", "unique_id", eventID),
						[]byte(fmt.Sprintf(`{"value": "test", "unique_id": "%s"}`, eventID)))
				}

				// Execute Append
				err := benchCtx.Store.Append(ctx, events)
				if err != nil {
					results <- fmt.Errorf("concurrent append failed: %v", err)
					return
				}

				results <- nil
			})
		}

		// Wait for all goroutines to complete
		wg.Wait()
		close(results)

		// Check for any errors
		for err := range results {
			if err != nil {
				b.Fatalf("Concurrent Append failed: %v", err)
			}
		}
	}
}

// BenchmarkAppendConcurrentRealistic benchmarks concurrent append operations using realistic dataset events
// Uses CourseOffered, StudentRegistered, and EnrollmentCompleted events from the loaded dataset
func BenchmarkAppendConcurrentRealistic(b *testing.B, benchCtx *BenchmarkContext, concurrencyLevel int, eventCount int) {
	// Create context with timeout for each benchmark iteration using Go 1.25 WithTimeoutCause
	ctx, cancel := context.WithTimeoutCause(context.Background(), 2*time.Minute,
		fmt.Errorf("benchmark timeout after 2 minutes"))
	defer cancel()

	// Truncate events table BEFORE timing starts (but after dataset is loaded)
	pool, err := getOrCreateGlobalPool()
	if err != nil {
		b.Fatalf("Failed to get global pool: %v", err)
	}

	_, err = pool.Exec(ctx, "TRUNCATE TABLE events RESTART IDENTITY CASCADE")
	if err != nil {
		b.Fatalf("Failed to truncate events table: %v", err)
	}

	// WARM-UP PHASE: Run the exact same logic we'll benchmark without timing
	warmupBenchmark(func() {
		var wg sync.WaitGroup
		results := make(chan error, concurrencyLevel)

		// Launch concurrent Append operations using WaitGroup.Go()
		for j := 0; j < concurrencyLevel; j++ {
			goroutineID := j // Capture loop variable
			wg.Go(func() {
				// Create unique ID for this goroutine
				uniqueID := fmt.Sprintf("warmup_%d_%d", time.Now().UnixNano(), goroutineID)

				// Create realistic events to append based on eventCount
				events := make([]dcb.InputEvent, eventCount)
				for k := 0; k < eventCount; k++ {
					eventID := fmt.Sprintf("warmup_%s_%d", uniqueID, k)

					// Alternate between different realistic event types
					switch k % 3 {
					case 0: // CourseOffered
						events[k] = dcb.NewInputEvent("CourseOffered",
							dcb.NewTags("course_id", eventID, "category", "Computer Science"),
							[]byte(fmt.Sprintf(`{
								"id": "%s",
								"name": "Advanced Programming",
								"instructor": "Dr. Smith",
								"capacity": 50,
								"category": "Computer Science",
								"popularity": 0.8
							}`, eventID)))
					case 1: // StudentRegistered
						events[k] = dcb.NewInputEvent("StudentRegistered",
							dcb.NewTags("student_id", eventID, "major", "Computer Science"),
							[]byte(fmt.Sprintf(`{
								"id": "%s",
								"name": "John Doe",
								"email": "john.doe@university.edu",
								"major": "Computer Science",
								"year": 3,
								"maxCourses": 5
							}`, eventID)))
					case 2: // EnrollmentCompleted
						events[k] = dcb.NewInputEvent("EnrollmentCompleted",
							dcb.NewTags("student_id", eventID, "course_id", fmt.Sprintf("course_%d", k)),
							[]byte(fmt.Sprintf(`{
								"studentId": "%s",
								"courseId": "course_%d",
								"enrolledAt": "%s",
								"grade": ""
							}`, eventID, k, time.Now().Format(time.RFC3339))))
					}
				}

				// Execute Append
				err := benchCtx.Store.Append(ctx, events)
				if err != nil {
					results <- fmt.Errorf("warmup realistic append failed: %v", err)
					return
				}

				results <- nil
			})
		}

		wg.Wait()
		close(results)

		// Check for any errors
		for err := range results {
			if err != nil {
				b.Fatalf("Warmup realistic Append failed: %v", err)
			}
		}
	})

	// Reset timer after warm-up
	b.ResetTimer()

	// BENCHMARK PHASE: Now measure the actual performance
	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		results := make(chan error, concurrencyLevel)

		// Launch concurrent Append operations using WaitGroup.Go()
		for j := 0; j < concurrencyLevel; j++ {
			goroutineID := j // Capture loop variable
			wg.Go(func() {
				// Create unique ID for this goroutine
				uniqueID := fmt.Sprintf("realistic_%d_%d", time.Now().UnixNano(), goroutineID)

				// Create realistic events to append based on eventCount
				events := make([]dcb.InputEvent, eventCount)
				for k := 0; k < eventCount; k++ {
					eventID := fmt.Sprintf("realistic_%s_%d", uniqueID, k)

					// Alternate between different realistic event types
					switch k % 3 {
					case 0: // CourseOffered
						events[k] = dcb.NewInputEvent("CourseOffered",
							dcb.NewTags("course_id", eventID, "category", "Computer Science"),
							[]byte(fmt.Sprintf(`{
								"id": "%s",
								"name": "Advanced Programming",
								"instructor": "Dr. Smith",
								"capacity": 50,
								"category": "Computer Science",
								"popularity": 0.8
							}`, eventID)))
					case 1: // StudentRegistered
						events[k] = dcb.NewInputEvent("StudentRegistered",
							dcb.NewTags("student_id", eventID, "major", "Computer Science"),
							[]byte(fmt.Sprintf(`{
								"id": "%s",
								"name": "John Doe",
								"email": "john.doe@university.edu",
								"major": "Computer Science",
								"year": 3,
								"maxCourses": 5
							}`, eventID)))
					case 2: // EnrollmentCompleted
						events[k] = dcb.NewInputEvent("EnrollmentCompleted",
							dcb.NewTags("student_id", eventID, "course_id", fmt.Sprintf("course_%d", k)),
							[]byte(fmt.Sprintf(`{
								"studentId": "%s",
								"courseId": "course_%d",
								"enrolledAt": "%s",
								"grade": ""
							}`, eventID, k, time.Now().Format(time.RFC3339))))
					}
				}

				// Execute Append
				err := benchCtx.Store.Append(ctx, events)
				if err != nil {
					results <- fmt.Errorf("realistic append failed: %v", err)
					return
				}

				results <- nil
			})
		}

		// Wait for all goroutines to complete
		wg.Wait()
		close(results)

		// Check for any errors
		for err := range results {
			if err != nil {
				b.Fatalf("Concurrent realistic Append failed: %v", err)
			}
		}
	}
}

// BenchmarkProjectConcurrent benchmarks concurrent projection operations
// Uses core API's built-in goroutine limits and Go 1.25 concurrency features
func BenchmarkProjectConcurrent(b *testing.B, benchCtx *BenchmarkContext, goroutines int) {
	// Create context with timeout using Go 1.25 WithTimeoutCause
	ctx, cancel := context.WithTimeoutCause(context.Background(), 2*time.Minute,
		fmt.Errorf("projection benchmark timeout after 2 minutes"))
	defer cancel()

	// Truncate events table and create test data BEFORE timing starts
	pool, err := getOrCreateGlobalPool()
	if err != nil {
		b.Fatalf("Failed to get global pool: %v", err)
	}

	_, err = pool.Exec(ctx, "TRUNCATE TABLE events RESTART IDENTITY CASCADE")
	if err != nil {
		b.Fatalf("Failed to truncate events table: %v", err)
	}

	// Create test events for projection benchmarks
	testEvents := make([]dcb.InputEvent, 100)
	for i := 0; i < 100; i++ {
		eventID := fmt.Sprintf("test_event_%d", i)
		testEvents[i] = dcb.NewInputEvent("TestEvent",
			dcb.NewTags("test", "concurrent", "unique_id", eventID),
			[]byte(fmt.Sprintf(`{"value": "test", "unique_id": "%s"}`, eventID)))
	}

	// Append test events to the store
	if err := benchCtx.Store.Append(ctx, testEvents); err != nil {
		b.Fatalf("Failed to append test events: %v", err)
	}

	// Create a simple projector for testing
	projector := dcb.StateProjector{
		ID:           "test_concurrent_projection",
		Query:        dcb.NewQueryBuilder().WithType("TestEvent").WithTag("test", "concurrent").Build(),
		InitialState: 0,
		TransitionFn: func(state any, event dcb.Event) any {
			return state.(int) + 1
		},
	}

	// WARM-UP PHASE: Run the exact same logic we'll benchmark without timing
	warmupBenchmark(func() {
		var wg sync.WaitGroup
		results := make(chan error, goroutines)

		// Start concurrent projections using core API's built-in limits
		for j := 0; j < goroutines; j++ {
			wg.Go(func() {
				// Let the core API handle goroutine limits
				_, _, err := benchCtx.Store.Project(ctx, []dcb.StateProjector{projector}, nil)
				if err != nil {
					results <- fmt.Errorf("warmup projection failed: %v", err)
					return
				}

				results <- nil
			})
		}

		wg.Wait()
		close(results)

		// Check for errors but don't fail on warm-up
		for err := range results {
			if err != nil {
				// Log but don't fail during warm-up
				continue
			}
		}
	})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Use Go 1.25 WaitGroup.Go() for concurrent operations
		var wg sync.WaitGroup
		results := make(chan error, goroutines)

		// Start concurrent projections using core API's built-in limits
		for j := 0; j < goroutines; j++ {
			wg.Go(func() {
				// Let the core API handle goroutine limits
				_, _, err := benchCtx.Store.Project(ctx, []dcb.StateProjector{projector}, nil)
				if err != nil {
					results <- fmt.Errorf("concurrent projection failed: %v", err)
					return
				}

				results <- nil
			})
		}

		// Wait for all goroutines to complete using Go 1.25 WaitGroup
		wg.Wait()
		close(results)

		// Check for any errors
		for err := range results {
			if err != nil {
				b.Fatalf("Concurrent Project failed: %v", err)
			}
		}
	}
}

// BenchmarkProjectConcurrentRealistic benchmarks concurrent projection operations using realistic dataset events
// Uses CourseOffered, StudentRegistered, and EnrollmentCompleted events with realistic projectors
func BenchmarkProjectConcurrentRealistic(b *testing.B, benchCtx *BenchmarkContext, goroutines int) {
	// Create context with timeout using Go 1.25 WithTimeoutCause
	ctx, cancel := context.WithTimeoutCause(context.Background(), 2*time.Minute,
		fmt.Errorf("realistic projection benchmark timeout after 2 minutes"))
	defer cancel()

	// Truncate events table and create realistic test data BEFORE timing starts
	pool, err := getOrCreateGlobalPool()
	if err != nil {
		b.Fatalf("Failed to get global pool: %v", err)
	}

	_, err = pool.Exec(ctx, "TRUNCATE TABLE events RESTART IDENTITY CASCADE")
	if err != nil {
		b.Fatalf("Failed to truncate events table: %v", err)
	}

	// Create realistic test events for projection benchmarks
	testEvents := make([]dcb.InputEvent, 100)
	for i := 0; i < 100; i++ {
		eventID := fmt.Sprintf("realistic_event_%d", i)

		// Alternate between different realistic event types
		switch i % 3 {
		case 0: // CourseOffered
			testEvents[i] = dcb.NewInputEvent("CourseOffered",
				dcb.NewTags("course_id", eventID, "category", "Computer Science"),
				[]byte(fmt.Sprintf(`{
					"id": "%s",
					"name": "Advanced Programming",
					"instructor": "Dr. Smith",
					"capacity": 50,
					"category": "Computer Science",
					"popularity": 0.8
				}`, eventID)))
		case 1: // StudentRegistered
			testEvents[i] = dcb.NewInputEvent("StudentRegistered",
				dcb.NewTags("student_id", eventID, "major", "Computer Science"),
				[]byte(fmt.Sprintf(`{
					"id": "%s",
					"name": "John Doe",
					"email": "john.doe@university.edu",
					"major": "Computer Science",
					"year": 3,
					"maxCourses": 5
				}`, eventID)))
		case 2: // EnrollmentCompleted
			testEvents[i] = dcb.NewInputEvent("EnrollmentCompleted",
				dcb.NewTags("student_id", eventID, "course_id", fmt.Sprintf("course_%d", i)),
				[]byte(fmt.Sprintf(`{
					"studentId": "%s",
					"courseId": "course_%d",
					"enrolledAt": "%s",
					"grade": ""
				}`, eventID, i, time.Now().Format(time.RFC3339))))
		}
	}

	// Append realistic test events to the store
	if err := benchCtx.Store.Append(ctx, testEvents); err != nil {
		b.Fatalf("Failed to append realistic test events: %v", err)
	}

	// Create realistic projectors for testing
	courseProjector := dcb.StateProjector{
		ID:           "course_count_projection",
		Query:        dcb.NewQueryBuilder().WithType("CourseOffered").WithTag("category", "Computer Science").Build(),
		InitialState: 0,
		TransitionFn: func(state any, event dcb.Event) any {
			return state.(int) + 1
		},
	}

	studentProjector := dcb.StateProjector{
		ID:           "student_count_projection",
		Query:        dcb.NewQueryBuilder().WithType("StudentRegistered").WithTag("major", "Computer Science").Build(),
		InitialState: 0,
		TransitionFn: func(state any, event dcb.Event) any {
			return state.(int) + 1
		},
	}

	enrollmentProjector := dcb.StateProjector{
		ID:           "enrollment_count_projection",
		Query:        dcb.NewQueryBuilder().WithType("EnrollmentCompleted").Build(),
		InitialState: 0,
		TransitionFn: func(state any, event dcb.Event) any {
			return state.(int) + 1
		},
	}

	// Combine all realistic projectors
	realisticProjectors := []dcb.StateProjector{courseProjector, studentProjector, enrollmentProjector}

	// WARM-UP PHASE: Run the exact same logic we'll benchmark without timing
	warmupBenchmark(func() {
		var wg sync.WaitGroup
		results := make(chan error, goroutines)

		// Start concurrent realistic projections using core API's built-in limits
		for j := 0; j < goroutines; j++ {
			wg.Go(func() {
				// Let the core API handle goroutine limits
				_, _, err := benchCtx.Store.Project(ctx, realisticProjectors, nil)
				if err != nil {
					results <- fmt.Errorf("warmup realistic projection failed: %v", err)
					return
				}

				results <- nil
			})
		}

		// Wait for all goroutines to complete using Go 1.25 WaitGroup
		wg.Wait()
		close(results)

		// Check for any errors
		for err := range results {
			if err != nil {
				// Log but don't fail during warm-up
				continue
			}
		}
	})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Use Go 1.25 WaitGroup.Go() for concurrent operations
		var wg sync.WaitGroup
		results := make(chan error, goroutines)

		// Start concurrent realistic projections using core API's built-in limits
		for j := 0; j < goroutines; j++ {
			wg.Go(func() {
				// Let the core API handle goroutine limits
				_, _, err := benchCtx.Store.Project(ctx, realisticProjectors, nil)
				if err != nil {
					results <- fmt.Errorf("concurrent realistic projection failed: %v", err)
					return
				}

				results <- nil
			})
		}

		// Wait for all goroutines to complete using Go 1.25 WaitGroup
		wg.Wait()
		close(results)

		// Check for any errors
		for err := range results {
			if err != nil {
				b.Fatalf("Concurrent realistic Project failed: %v", err)
			}
		}
	}
}

// BenchmarkProjectStreamConcurrent benchmarks concurrent streaming projection operations
// Uses core API's built-in goroutine limits and Go 1.25 concurrency features
func BenchmarkProjectStreamConcurrent(b *testing.B, benchCtx *BenchmarkContext, goroutines int) {
	ctx := context.Background()

	// Truncate events table and create test data BEFORE timing starts
	pool, err := getOrCreateGlobalPool()
	if err != nil {
		b.Fatalf("Failed to get global pool: %v", err)
	}

	_, err = pool.Exec(ctx, "TRUNCATE TABLE events RESTART IDENTITY CASCADE")
	if err != nil {
		b.Fatalf("Failed to truncate events table: %v", err)
	}

	// Create test events for projection benchmarks
	testEvents := make([]dcb.InputEvent, 100)
	for i := 0; i < 100; i++ {
		eventID := fmt.Sprintf("test_event_%d", i)
		testEvents[i] = dcb.NewInputEvent("TestEvent",
			dcb.NewTags("test", "concurrent", "unique_id", eventID),
			[]byte(fmt.Sprintf(`{"value": "test", "unique_id": "%s"}`, eventID)))
	}

	// Append test events to the store
	if err := benchCtx.Store.Append(ctx, testEvents); err != nil {
		b.Fatalf("Failed to append test events: %v", err)
	}

	// Create a simple projector for testing
	projector := dcb.StateProjector{
		ID:           "test_concurrent_stream_projection",
		Query:        dcb.NewQueryBuilder().WithType("TestEvent").WithTag("test", "concurrent").Build(),
		InitialState: 0,
		TransitionFn: func(state any, event dcb.Event) any {
			return state.(int) + 1
		},
	}

	// WARM-UP PHASE: Run the exact same logic we'll benchmark without timing
	warmupBenchmark(func() {
		var wg sync.WaitGroup
		results := make(chan error, goroutines)

		// Start concurrent streaming projections using core API's built-in limits
		for j := 0; j < goroutines; j++ {
			wg.Go(func() {
				// Let the core API handle goroutine limits and streaming
				stateChan, conditionChan, err := benchCtx.Store.ProjectStream(ctx, []dcb.StateProjector{projector}, nil)
				if err != nil {
					results <- fmt.Errorf("warmup ProjectStream failed: %v", err)
					return
				}

				// Consume from channels (API handles concurrency internally)
				select {
				case state := <-stateChan:
					_ = state // Use state to prevent optimization
				case <-time.After(5 * time.Second):
					results <- fmt.Errorf("warmup ProjectStream timeout")
					return
				}

				select {
				case condition := <-conditionChan:
					_ = condition // Use condition to prevent optimization
				case <-time.After(5 * time.Second):
					results <- fmt.Errorf("warmup ProjectStream condition timeout")
					return
				}

				results <- nil
			})
		}

		wg.Wait()
		close(results)

		// Check for errors but don't fail on warm-up
		for err := range results {
			if err != nil {
				// Log but don't fail during warm-up
				continue
			}
		}
	})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Use Go 1.25 WaitGroup.Go() for concurrent operations
		var wg sync.WaitGroup
		results := make(chan error, goroutines)

		// Start concurrent streaming projections using core API's built-in limits
		for j := 0; j < goroutines; j++ {
			wg.Go(func() {
				// Let the core API handle goroutine limits and streaming
				stateChan, conditionChan, err := benchCtx.Store.ProjectStream(ctx, []dcb.StateProjector{projector}, nil)
				if err != nil {
					results <- fmt.Errorf("concurrent ProjectStream failed: %v", err)
					return
				}

				// Consume from channels (API handles concurrency internally)
				select {
				case state := <-stateChan:
					_ = state // Use state to prevent optimization
				case <-time.After(5 * time.Second):
					results <- fmt.Errorf("ProjectStream timeout")
					return
				}

				select {
				case condition := <-conditionChan:
					_ = condition // Use condition to prevent optimization
				case <-time.After(5 * time.Second):
					results <- fmt.Errorf("ProjectStream condition timeout")
					return
				}

				results <- nil
			})
		}

		// Wait for all goroutines to complete using Go 1.25 WaitGroup
		wg.Wait()
		close(results)

		// Check for any errors
		for err := range results {
			if err != nil {
				b.Fatalf("Concurrent ProjectStream failed: %v", err)
			}
		}
	}
}

// BenchmarkProjectStreamConcurrentRealistic benchmarks concurrent streaming projection operations using realistic dataset events
// Uses CourseOffered, StudentRegistered, and EnrollmentCompleted events with realistic projectors
func BenchmarkProjectStreamConcurrentRealistic(b *testing.B, benchCtx *BenchmarkContext, goroutines int) {
	ctx := context.Background()

	// Truncate events table and create realistic test data BEFORE timing starts
	pool, err := getOrCreateGlobalPool()
	if err != nil {
		b.Fatalf("Failed to get global pool: %v", err)
	}

	_, err = pool.Exec(ctx, "TRUNCATE TABLE events RESTART IDENTITY CASCADE")
	if err != nil {
		b.Fatalf("Failed to truncate events table: %v", err)
	}

	// Create realistic test events for projection benchmarks
	testEvents := make([]dcb.InputEvent, 100)
	for i := 0; i < 100; i++ {
		eventID := fmt.Sprintf("realistic_stream_event_%d", i)

		// Alternate between different realistic event types
		switch i % 3 {
		case 0: // CourseOffered
			testEvents[i] = dcb.NewInputEvent("CourseOffered",
				dcb.NewTags("course_id", eventID, "category", "Computer Science"),
				[]byte(fmt.Sprintf(`{
					"id": "%s",
					"name": "Advanced Programming",
					"instructor": "Dr. Smith",
					"capacity": 50,
					"category": "Computer Science",
					"popularity": 0.8
				}`, eventID)))
		case 1: // StudentRegistered
			testEvents[i] = dcb.NewInputEvent("StudentRegistered",
				dcb.NewTags("student_id", eventID, "major", "Computer Science"),
				[]byte(fmt.Sprintf(`{
					"id": "%s",
					"name": "John Doe",
					"email": "john.doe@university.edu",
					"major": "Computer Science",
					"year": 3,
					"maxCourses": 5
				}`, eventID)))
		case 2: // EnrollmentCompleted
			testEvents[i] = dcb.NewInputEvent("EnrollmentCompleted",
				dcb.NewTags("student_id", eventID, "course_id", fmt.Sprintf("course_%d", i)),
				[]byte(fmt.Sprintf(`{
					"studentId": "%s",
					"courseId": "course_%d",
					"enrolledAt": "%s",
					"grade": ""
				}`, eventID, i, time.Now().Format(time.RFC3339))))
		}
	}

	// Append realistic test events to the store
	if err := benchCtx.Store.Append(ctx, testEvents); err != nil {
		b.Fatalf("Failed to append realistic test events: %v", err)
	}

	// Create realistic projectors for testing
	courseProjector := dcb.StateProjector{
		ID:           "course_count_stream_projection",
		Query:        dcb.NewQueryBuilder().WithType("CourseOffered").WithTag("category", "Computer Science").Build(),
		InitialState: 0,
		TransitionFn: func(state any, event dcb.Event) any {
			return state.(int) + 1
		},
	}

	studentProjector := dcb.StateProjector{
		ID:           "student_count_stream_projection",
		Query:        dcb.NewQueryBuilder().WithType("StudentRegistered").WithTag("major", "Computer Science").Build(),
		InitialState: 0,
		TransitionFn: func(state any, event dcb.Event) any {
			return state.(int) + 1
		},
	}

	enrollmentProjector := dcb.StateProjector{
		ID:           "enrollment_count_stream_projection",
		Query:        dcb.NewQueryBuilder().WithType("EnrollmentCompleted").Build(),
		InitialState: 0,
		TransitionFn: func(state any, event dcb.Event) any {
			return state.(int) + 1
		},
	}

	// Combine all realistic projectors
	realisticProjectors := []dcb.StateProjector{courseProjector, studentProjector, enrollmentProjector}

	// WARM-UP PHASE: Run the exact same logic we'll benchmark without timing
	warmupBenchmark(func() {
		var wg sync.WaitGroup
		results := make(chan error, goroutines)

		// Start concurrent realistic streaming projections using core API's built-in limits
		for j := 0; j < goroutines; j++ {
			wg.Go(func() {
				// Let the core API handle goroutine limits and streaming
				stateChan, conditionChan, err := benchCtx.Store.ProjectStream(ctx, realisticProjectors, nil)
				if err != nil {
					results <- fmt.Errorf("warmup realistic ProjectStream failed: %v", err)
					return
				}

				// Consume from channels (API handles concurrency internally)
				select {
				case state := <-stateChan:
					_ = state // Use state to prevent optimization
				case <-time.After(5 * time.Second):
					results <- fmt.Errorf("warmup realistic ProjectStream timeout")
					return
				}

				select {
				case condition := <-conditionChan:
					_ = condition // Use condition to prevent optimization
				case <-time.After(5 * time.Second):
					results <- fmt.Errorf("warmup realistic ProjectStream condition timeout")
					return
				}

				results <- nil
			})
		}

		wg.Wait()
		close(results)

		// Check for errors but don't fail on warm-up
		for err := range results {
			if err != nil {
				// Log but don't fail during warm-up
				continue
			}
		}
	})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Use Go 1.25 WaitGroup.Go() for concurrent operations
		var wg sync.WaitGroup
		results := make(chan error, goroutines)

		// Start concurrent realistic streaming projections using core API's built-in limits
		for j := 0; j < goroutines; j++ {
			wg.Go(func() {
				// Let the core API handle goroutine limits and streaming
				stateChan, conditionChan, err := benchCtx.Store.ProjectStream(ctx, realisticProjectors, nil)
				if err != nil {
					results <- fmt.Errorf("concurrent realistic ProjectStream failed: %v", err)
					return
				}

				// Consume from channels (API handles concurrency internally)
				select {
				case state := <-stateChan:
					_ = state // Use state to prevent optimization
				case <-time.After(5 * time.Second):
					results <- fmt.Errorf("realistic ProjectStream timeout")
					return
				}

				select {
				case condition := <-conditionChan:
					_ = condition // Use condition to prevent optimization
				case <-time.After(5 * time.Second):
					results <- fmt.Errorf("realistic ProjectStream condition timeout")
					return
				}

				results <- nil
			})
		}

		// Wait for all goroutines to complete using Go 1.25 WaitGroup
		wg.Wait()
		close(results)

		// Check for any errors
		for err := range results {
			if err != nil {
				b.Fatalf("Concurrent realistic ProjectStream failed: %v", err)
			}
		}
	}
}

// BenchmarkProjectionLimits benchmarks projection limits with low limits
func BenchmarkProjectionLimits(b *testing.B, benchCtx *BenchmarkContext, goroutines int) {
	// Create EventStore with low limit for testing
	config := dcb.EventStoreConfig{
		MaxConcurrentProjections: 5, // Low limit for testing
		MaxProjectionGoroutines:  10,
		StreamBuffer:             100,
		QueryTimeout:             5000,
		AppendTimeout:            3000,
	}

	pool, err := getOrCreateGlobalPool()
	if err != nil {
		b.Fatalf("Failed to get global pool: %v", err)
	}

	store, err := dcb.NewEventStoreWithConfig(context.Background(), pool, config)
	if err != nil {
		b.Fatalf("Failed to create store with config: %v", err)
	}

	// Create test data
	testEvents := make([]dcb.InputEvent, 100)
	for i := 0; i < 100; i++ {
		testEvents[i] = dcb.NewInputEvent("TestEvent",
			dcb.NewTags("test", "limits", "id", fmt.Sprintf("event_%d", i)),
			dcb.ToJSON(map[string]string{"value": fmt.Sprintf("test_%d", i)}))
	}

	err = store.Append(context.Background(), testEvents)
	if err != nil {
		b.Fatalf("Failed to append test events: %v", err)
	}

	// Create projector
	projector := dcb.StateProjector{
		ID:           "test_limit_projection",
		Query:        dcb.NewQuery(dcb.NewTags("test", "limits"), "TestEvent"),
		InitialState: 0,
		TransitionFn: func(state any, event dcb.Event) any {
			return state.(int) + 1
		},
	}

	// WARM-UP PHASE: Run the exact same logic we'll benchmark without timing
	warmupBenchmark(func() {
		// Launch concurrent projections (may exceed limit of 5)
		var wg sync.WaitGroup
		results := make(chan error, goroutines)

		for j := 0; j < goroutines; j++ {
			wg.Go(func() {
				_, _, err := store.Project(context.Background(), []dcb.StateProjector{projector}, nil)
				results <- err
			})
		}

		wg.Wait()
		close(results)

		// Check for errors but don't fail on warm-up
		for err := range results {
			if err != nil {
				// Log but don't fail during warm-up
				continue
			}
		}
	})

	b.ResetTimer()
	b.ReportAllocs()

	successCount := 0
	limitExceededCount := 0

	for i := 0; i < b.N; i++ {
		// Launch concurrent projections (may exceed limit of 5)
		var wg sync.WaitGroup
		results := make(chan error, goroutines)

		for j := 0; j < goroutines; j++ {
			wg.Go(func() {
				_, _, err := store.Project(context.Background(), []dcb.StateProjector{projector}, nil)
				results <- err
			})
		}

		wg.Wait()
		close(results)

		// Count results
		for err := range results {
			if err == nil {
				successCount++
			} else if _, ok := err.(*dcb.TooManyProjectionsError); ok {
				limitExceededCount++
			}
		}
	}

	// Report metrics
	totalOperations := successCount + limitExceededCount
	if totalOperations > 0 {
		b.ReportMetric(float64(successCount)/float64(totalOperations), "success_rate")
		b.ReportMetric(float64(limitExceededCount)/float64(totalOperations), "limit_exceeded_rate")
	}
}

// BenchmarkProjectionLimitsRealistic benchmarks projection limits with realistic dataset events
// Uses CourseOffered, StudentRegistered, and EnrollmentCompleted events with realistic projectors
func BenchmarkProjectionLimitsRealistic(b *testing.B, benchCtx *BenchmarkContext, goroutines int) {
	// Create EventStore with low limit for testing
	config := dcb.EventStoreConfig{
		MaxConcurrentProjections: 5, // Low limit for testing
		MaxProjectionGoroutines:  10,
		StreamBuffer:             100,
		QueryTimeout:             5000,
		AppendTimeout:            3000,
	}

	pool, err := getOrCreateGlobalPool()
	if err != nil {
		b.Fatalf("Failed to get global pool: %v", err)
	}

	store, err := dcb.NewEventStoreWithConfig(context.Background(), pool, config)
	if err != nil {
		b.Fatalf("Failed to create store with config: %v", err)
	}

	// Create realistic test data
	testEvents := make([]dcb.InputEvent, 100)
	for i := 0; i < 100; i++ {
		eventID := fmt.Sprintf("realistic_limits_event_%d", i)

		// Alternate between different realistic event types
		switch i % 3 {
		case 0: // CourseOffered
			testEvents[i] = dcb.NewInputEvent("CourseOffered",
				dcb.NewTags("course_id", eventID, "category", "Computer Science"),
				[]byte(fmt.Sprintf(`{
					"id": "%s",
					"name": "Advanced Programming",
					"instructor": "Dr. Smith",
					"capacity": 50,
					"category": "Computer Science",
					"popularity": 0.8
				}`, eventID)))
		case 1: // StudentRegistered
			testEvents[i] = dcb.NewInputEvent("StudentRegistered",
				dcb.NewTags("student_id", eventID, "major", "Computer Science"),
				[]byte(fmt.Sprintf(`{
					"id": "%s",
					"name": "John Doe",
					"email": "john.doe@university.edu",
					"major": "Computer Science",
					"year": 3,
					"maxCourses": 5
				}`, eventID)))
		case 2: // EnrollmentCompleted
			testEvents[i] = dcb.NewInputEvent("EnrollmentCompleted",
				dcb.NewTags("student_id", eventID, "course_id", fmt.Sprintf("course_%d", i)),
				[]byte(fmt.Sprintf(`{
					"studentId": "%s",
					"courseId": "course_%d",
					"enrolledAt": "%s",
					"grade": ""
				}`, eventID, i, time.Now().Format(time.RFC3339))))
		}
	}

	err = store.Append(context.Background(), testEvents)
	if err != nil {
		b.Fatalf("Failed to append realistic test events: %v", err)
	}

	// Create realistic projector
	projector := dcb.StateProjector{
		ID:           "realistic_limit_projection",
		Query:        dcb.NewQuery(dcb.NewTags("course_id", "realistic_limits_event_0", "category", "Computer Science"), "CourseOffered"),
		InitialState: 0,
		TransitionFn: func(state any, event dcb.Event) any {
			return state.(int) + 1
		},
	}

	// WARM-UP PHASE: Run the exact same logic we'll benchmark without timing
	warmupBenchmark(func() {
		// Launch concurrent realistic projections (may exceed limit of 5)
		var wg sync.WaitGroup
		results := make(chan error, goroutines)

		for j := 0; j < goroutines; j++ {
			wg.Go(func() {
				_, _, err := store.Project(context.Background(), []dcb.StateProjector{projector}, nil)
				results <- err
			})
		}

		wg.Wait()
		close(results)

		// Check for errors but don't fail on warm-up
		for err := range results {
			if err != nil {
				// Log but don't fail during warm-up
				continue
			}
		}
	})

	b.ResetTimer()
	b.ReportAllocs()

	successCount := 0
	limitExceededCount := 0

	for i := 0; i < b.N; i++ {
		// Launch concurrent realistic projections (may exceed limit of 5)
		var wg sync.WaitGroup
		results := make(chan error, goroutines)

		for j := 0; j < goroutines; j++ {
			wg.Go(func() {
				_, _, err := store.Project(context.Background(), []dcb.StateProjector{projector}, nil)
				results <- err
			})
		}

		wg.Wait()
		close(results)

		// Count results
		for err := range results {
			if err == nil {
				successCount++
			} else if _, ok := err.(*dcb.TooManyProjectionsError); ok {
				limitExceededCount++
			}
		}
	}

	// Report metrics
	totalOperations := successCount + limitExceededCount
	if totalOperations > 0 {
		b.ReportMetric(float64(successCount)/float64(totalOperations), "success_rate")
		b.ReportMetric(float64(limitExceededCount)/float64(totalOperations), "limit_exceeded_rate")
	}
}

// RunAllBenchmarks runs all benchmarks with the specified dataset size
func RunAllBenchmarks(b *testing.B, datasetSize string) {
	// Use 100 past events for realistic AppendIf testing (business rule validation context)
	benchCtx := SetupBenchmarkContext(b, datasetSize, 100)

	// Append benchmarks (concurrent only - standardized to 1 or 100 events)

	// Concurrent Append benchmarks (1 event per user)
	b.Run("Append_Concurrent_1User_1Event", func(b *testing.B) {
		BenchmarkAppendConcurrent(b, benchCtx, 1, 1)
	})

	b.Run("Append_Concurrent_100Users_1Event", func(b *testing.B) {
		BenchmarkAppendConcurrent(b, benchCtx, 100, 1)
	})

	// Concurrent Append benchmarks (10 events per user)
	b.Run("Append_Concurrent_1User_10Events", func(b *testing.B) {
		BenchmarkAppendConcurrent(b, benchCtx, 1, 10)
	})

	b.Run("Append_Concurrent_100Users_10Events", func(b *testing.B) {
		BenchmarkAppendConcurrent(b, benchCtx, 100, 10)
	})

	// Concurrent AppendIf benchmarks - NO CONFLICT (1 event)
	b.Run("AppendIf_NoConflict_Concurrent_1User_1Event", func(b *testing.B) {
		BenchmarkAppendIfConcurrent(b, benchCtx, 1, 1, false)
	})

	b.Run("AppendIf_NoConflict_Concurrent_100Users_1Event", func(b *testing.B) {
		BenchmarkAppendIfConcurrent(b, benchCtx, 100, 1, false)
	})

	// Concurrent AppendIf benchmarks - NO CONFLICT (10 events)
	b.Run("AppendIf_NoConflict_Concurrent_1User_10Events", func(b *testing.B) {
		BenchmarkAppendIfConcurrent(b, benchCtx, 1, 10, false)
	})

	b.Run("AppendIf_NoConflict_Concurrent_100Users_10Events", func(b *testing.B) {
		BenchmarkAppendIfConcurrent(b, benchCtx, 100, 10, false)
	})

	// Concurrent AppendIf benchmarks - WITH CONFLICT (1 event)
	b.Run("AppendIf_WithConflict_Concurrent_1User_1Event", func(b *testing.B) {
		BenchmarkAppendIfConcurrent(b, benchCtx, 1, 1, true)
	})

	b.Run("AppendIf_WithConflict_Concurrent_100Users_1Event", func(b *testing.B) {
		BenchmarkAppendIfConcurrent(b, benchCtx, 100, 1, true)
	})

	// Concurrent AppendIf benchmarks - WITH CONFLICT (10 events)
	b.Run("AppendIf_WithConflict_Concurrent_1User_10Events", func(b *testing.B) {
		BenchmarkAppendIfConcurrent(b, benchCtx, 1, 10, true)
	})

	b.Run("AppendIf_WithConflict_Concurrent_100Users_10Events", func(b *testing.B) {
		BenchmarkAppendIfConcurrent(b, benchCtx, 100, 10, true)
	})

	// Projection benchmarks
	// Use separate setup for projection benchmarks to avoid scanning large datasets
	projectionCtx := SetupProjectionBenchmarkContext(b, datasetSize)

	// Concurrent projection benchmarks (replaces individual Project_1/2 and ProjectStream_1/2)
	b.Run("Project_Concurrent_1User", func(b *testing.B) {
		BenchmarkProjectConcurrent(b, projectionCtx, 1)
	})

	b.Run("Project_Concurrent_100Users", func(b *testing.B) {
		BenchmarkProjectConcurrent(b, projectionCtx, 100)
	})

	b.Run("ProjectStream_Concurrent_1User", func(b *testing.B) {
		BenchmarkProjectStreamConcurrent(b, projectionCtx, 1)
	})

	b.Run("ProjectStream_Concurrent_100Users", func(b *testing.B) {
		BenchmarkProjectStreamConcurrent(b, projectionCtx, 100)
	})

	// Projection limits benchmarks
	b.Run("ProjectionLimits_5Users", func(b *testing.B) {
		BenchmarkProjectionLimits(b, projectionCtx, 5)
	})

	b.Run("ProjectionLimits_8Users", func(b *testing.B) {
		BenchmarkProjectionLimits(b, projectionCtx, 8)
	})

	b.Run("ProjectionLimits_10Users", func(b *testing.B) {
		BenchmarkProjectionLimits(b, projectionCtx, 10)
	})

}

// BenchmarkQueryConcurrentRealistic benchmarks realistic Query operations with business events
func BenchmarkQueryConcurrentRealistic(b *testing.B, benchCtx *BenchmarkContext, goroutines int) {
	ctx := context.Background()

	// Warm-up: Run a few iterations without timing to stabilize JIT and caches
	for i := 0; i < 3; i++ {
		query := dcb.NewQuery(dcb.NewTags("category", "Computer Science"), "CourseOffered")
		_, err := benchCtx.Store.Query(ctx, query, nil)
		if err != nil {
			b.Fatalf("Query warm-up failed: %v", err)
		}
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		results := make(chan error, goroutines)

		for j := 0; j < goroutines; j++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				// Realistic query: Find all Computer Science course offerings
				query := dcb.NewQuery(dcb.NewTags("category", "Computer Science"), "CourseOffered")
				_, err := benchCtx.Store.Query(ctx, query, nil)
				if err != nil {
					results <- fmt.Errorf("concurrent Query failed: %v", err)
					return
				}

				results <- nil
			}()
		}

		wg.Wait()
		close(results)

		// Check for errors
		for err := range results {
			if err != nil {
				b.Fatalf("Query benchmark failed: %v", err)
			}
		}
	}
}

// BenchmarkQueryStreamConcurrentRealistic benchmarks realistic QueryStream operations with business events
func BenchmarkQueryStreamConcurrentRealistic(b *testing.B, benchCtx *BenchmarkContext, goroutines int) {
	ctx := context.Background()

	// Warm-up: Run a few iterations without timing to stabilize JIT and caches
	for i := 0; i < 3; i++ {
		query := dcb.NewQuery(dcb.NewTags("major", "Computer Science"), "StudentRegistered")
		eventChan, err := benchCtx.Store.QueryStream(ctx, query, nil)
		if err != nil {
			b.Fatalf("QueryStream warm-up failed: %v", err)
		}

		// Consume from channels to prevent blocking
		select {
		case <-eventChan:
		case <-time.After(1 * time.Second):
		}
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		results := make(chan error, goroutines)

		for j := 0; j < goroutines; j++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				// Realistic query: Stream all Computer Science student registrations
				query := dcb.NewQuery(dcb.NewTags("major", "Computer Science"), "StudentRegistered")
				eventChan, err := benchCtx.Store.QueryStream(ctx, query, nil)
				if err != nil {
					results <- fmt.Errorf("concurrent QueryStream failed: %v", err)
					return
				}

				// Consume from channels (API handles concurrency internally)
				select {
				case <-eventChan:
					// Use event to prevent optimization
				case <-time.After(5 * time.Second):
					results <- fmt.Errorf("QueryStream timeout")
					return
				}

				results <- nil
			}()
		}

		wg.Wait()
		close(results)

		// Check for errors
		for err := range results {
			if err != nil {
				b.Fatalf("QueryStream benchmark failed: %v", err)
			}
		}
	}
}

// RunAllBenchmarksRealistic runs all realistic benchmarks using actual dataset events
// Uses CourseOffered, StudentRegistered, and EnrollmentCompleted events
func RunAllBenchmarksRealistic(b *testing.B, datasetSize string) {
	// Use 100 past events for realistic AppendIf testing (business rule validation context)
	benchCtx := SetupBenchmarkContext(b, datasetSize, 100)

	// Realistic Append benchmarks (concurrent only - standardized to 1 or 10 events)

	// Concurrent Realistic Append benchmarks (1 event per user)
	b.Run("Append_Concurrent_1User_1Event", func(b *testing.B) {
		BenchmarkAppendConcurrentRealistic(b, benchCtx, 1, 1)
	})

	b.Run("Append_Concurrent_100Users_1Event", func(b *testing.B) {
		BenchmarkAppendConcurrentRealistic(b, benchCtx, 100, 1)
	})

	// Concurrent Realistic Append benchmarks (10 events per user)
	b.Run("Append_Concurrent_1User_10Events", func(b *testing.B) {
		BenchmarkAppendConcurrentRealistic(b, benchCtx, 1, 10)
	})

	b.Run("Append_Concurrent_100Users_10Events", func(b *testing.B) {
		BenchmarkAppendConcurrentRealistic(b, benchCtx, 100, 10)
	})

	// Concurrent Realistic AppendIf benchmarks - NO CONFLICT (1 event)
	b.Run("AppendIf_NoConflict_Concurrent_1User_1Event", func(b *testing.B) {
		BenchmarkAppendIfConcurrentRealistic(b, benchCtx, 1, 1, false)
	})

	b.Run("AppendIf_NoConflict_Concurrent_100Users_1Event", func(b *testing.B) {
		BenchmarkAppendIfConcurrentRealistic(b, benchCtx, 100, 1, false)
	})

	// Concurrent Realistic AppendIf benchmarks - NO CONFLICT (10 events)
	b.Run("AppendIf_NoConflict_Concurrent_1User_10Events", func(b *testing.B) {
		BenchmarkAppendIfConcurrentRealistic(b, benchCtx, 1, 10, false)
	})

	b.Run("AppendIf_NoConflict_Concurrent_100Users_10Events", func(b *testing.B) {
		BenchmarkAppendIfConcurrentRealistic(b, benchCtx, 100, 10, false)
	})

	// Concurrent Realistic AppendIf benchmarks - WITH CONFLICT (1 event)
	b.Run("AppendIf_WithConflict_Concurrent_1User_1Event", func(b *testing.B) {
		BenchmarkAppendIfConcurrentRealistic(b, benchCtx, 1, 1, true)
	})

	b.Run("AppendIf_WithConflict_Concurrent_100Users_1Event", func(b *testing.B) {
		BenchmarkAppendIfConcurrentRealistic(b, benchCtx, 100, 1, true)
	})

	// Concurrent Realistic AppendIf benchmarks - WITH CONFLICT (10 events)
	b.Run("AppendIf_WithConflict_Concurrent_1User_10Events", func(b *testing.B) {
		BenchmarkAppendIfConcurrentRealistic(b, benchCtx, 1, 10, true)
	})

	b.Run("AppendIf_WithConflict_Concurrent_100Users_10Events", func(b *testing.B) {
		BenchmarkAppendIfConcurrentRealistic(b, benchCtx, 100, 10, true)
	})

	// Concurrent Realistic Project benchmarks
	b.Run("Project_Concurrent_1User", func(b *testing.B) {
		BenchmarkProjectConcurrentRealistic(b, benchCtx, 1)
	})

	b.Run("Project_Concurrent_100Users", func(b *testing.B) {
		BenchmarkProjectConcurrentRealistic(b, benchCtx, 100)
	})

	// Concurrent Realistic ProjectStream benchmarks
	b.Run("ProjectStream_Concurrent_1User", func(b *testing.B) {
		BenchmarkProjectStreamConcurrentRealistic(b, benchCtx, 1)
	})

	b.Run("ProjectStream_Concurrent_100Users", func(b *testing.B) {
		BenchmarkProjectStreamConcurrentRealistic(b, benchCtx, 100)
	})

	// Realistic Read benchmarks (Query operations)
	b.Run("Query_Concurrent_1User", func(b *testing.B) {
		BenchmarkQueryConcurrentRealistic(b, benchCtx, 1)
	})

	b.Run("Query_Concurrent_100Users", func(b *testing.B) {
		BenchmarkQueryConcurrentRealistic(b, benchCtx, 100)
	})

	// Realistic Read Stream benchmarks (QueryStream operations)
	b.Run("QueryStream_Concurrent_1User", func(b *testing.B) {
		BenchmarkQueryStreamConcurrentRealistic(b, benchCtx, 1)
	})

	b.Run("QueryStream_Concurrent_100Users", func(b *testing.B) {
		BenchmarkQueryStreamConcurrentRealistic(b, benchCtx, 100)
	})

	// Realistic ProjectionLimits benchmarks
	b.Run("ProjectionLimits_5Users", func(b *testing.B) {
		BenchmarkProjectionLimitsRealistic(b, benchCtx, 5)
	})

	b.Run("ProjectionLimits_8Users", func(b *testing.B) {
		BenchmarkProjectionLimitsRealistic(b, benchCtx, 8)
	})

	b.Run("ProjectionLimits_10Users", func(b *testing.B) {
		BenchmarkProjectionLimitsRealistic(b, benchCtx, 10)
	})
}

// TestMain sets up and tears down the shared global pool for all benchmarks
func TestMain(m *testing.M) {
	// Initialize the shared global pool before running any benchmarks
	pool, err := getOrCreateGlobalPool()
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize global pool: %v", err))
	}

	// Warm up the pool with a few test queries
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Execute a few warm-up queries to ensure connections are ready
	for i := 0; i < 3; i++ {
		_, err := pool.Exec(ctx, "SELECT 1")
		if err != nil {
			panic(fmt.Sprintf("Failed to warm up pool: %v", err))
		}
	}

	// Run all benchmarks
	exitCode := m.Run()

	// Clean up the global pool
	globalPoolMu.Lock()
	if globalPool != nil {
		globalPool.Close()
		globalPool = nil
	}
	globalPoolMu.Unlock()

	// Exit with the same code as the tests
	os.Exit(exitCode)
}
