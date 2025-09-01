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

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// For conflict scenarios, create all conflicting events first
		if conflictScenario {
			for j := 0; j < concurrencyLevel; j++ {
				uniqueID := fmt.Sprintf("concurrent_%d_%d_%d", time.Now().UnixNano(), i, j)
				conflictEvent := dcb.NewInputEvent("ConflictingEvent",
					dcb.NewTags("test", "conflict", "unique_id", uniqueID),
					[]byte(fmt.Sprintf(`{"value": "test", "unique_id": "%s"}`, uniqueID)))

				err := benchCtx.Store.Append(ctx, []dcb.InputEvent{conflictEvent})
				if err != nil {
					b.Fatalf("Failed to append conflict event: %v", err)
				}
			}
			// Ensure all conflicting events are committed (no artificial delay)
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

// BenchmarkAppendConcurrent benchmarks concurrent Append operations
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

// RunAllBenchmarks runs all benchmarks with the specified dataset size
func RunAllBenchmarks(b *testing.B, datasetSize string) {
	// Use 100 past events for realistic AppendIf testing (business rule validation context)
	benchCtx := SetupBenchmarkContext(b, datasetSize, 100)

	// Append benchmarks (concurrent only - standardized to 1 or 100 events)

	// Concurrent Append benchmarks (1 event per user)
	b.Run("Append_Concurrent_1User_1Event", func(b *testing.B) {
		BenchmarkAppendConcurrent(b, benchCtx, 1, 1)
	})

	b.Run("Append_Concurrent_10Users_1Event", func(b *testing.B) {
		BenchmarkAppendConcurrent(b, benchCtx, 10, 1)
	})

	b.Run("Append_Concurrent_100Users_1Event", func(b *testing.B) {
		BenchmarkAppendConcurrent(b, benchCtx, 100, 1)
	})

	// Concurrent Append benchmarks (100 events per user)
	b.Run("Append_Concurrent_1User_100Events", func(b *testing.B) {
		BenchmarkAppendConcurrent(b, benchCtx, 1, 100)
	})

	b.Run("Append_Concurrent_10Users_100Events", func(b *testing.B) {
		BenchmarkAppendConcurrent(b, benchCtx, 10, 100)
	})

	b.Run("Append_Concurrent_100Users_100Events", func(b *testing.B) {
		BenchmarkAppendConcurrent(b, benchCtx, 100, 100)
	})

	// Concurrent AppendIf benchmarks - NO CONFLICT (1 event)
	b.Run("AppendIf_NoConflict_Concurrent_1User_1Event", func(b *testing.B) {
		BenchmarkAppendIfConcurrent(b, benchCtx, 1, 1, false)
	})

	b.Run("AppendIf_NoConflict_Concurrent_10Users_1Event", func(b *testing.B) {
		BenchmarkAppendIfConcurrent(b, benchCtx, 10, 1, false)
	})

	b.Run("AppendIf_NoConflict_Concurrent_100Users_1Event", func(b *testing.B) {
		BenchmarkAppendIfConcurrent(b, benchCtx, 100, 1, false)
	})

	// Concurrent AppendIf benchmarks - NO CONFLICT (100 events)
	b.Run("AppendIf_NoConflict_Concurrent_1User_100Events", func(b *testing.B) {
		BenchmarkAppendIfConcurrent(b, benchCtx, 1, 100, false)
	})

	b.Run("AppendIf_NoConflict_Concurrent_10Users_100Events", func(b *testing.B) {
		BenchmarkAppendIfConcurrent(b, benchCtx, 10, 100, false)
	})

	b.Run("AppendIf_NoConflict_Concurrent_100Users_100Events", func(b *testing.B) {
		BenchmarkAppendIfConcurrent(b, benchCtx, 100, 100, false)
	})

	// Concurrent AppendIf benchmarks - WITH CONFLICT (1 event)
	b.Run("AppendIf_WithConflict_Concurrent_1User_1Event", func(b *testing.B) {
		BenchmarkAppendIfConcurrent(b, benchCtx, 1, 1, true)
	})

	b.Run("AppendIf_WithConflict_Concurrent_10Users_1Event", func(b *testing.B) {
		BenchmarkAppendIfConcurrent(b, benchCtx, 10, 1, true)
	})

	b.Run("AppendIf_WithConflict_Concurrent_100Users_1Event", func(b *testing.B) {
		BenchmarkAppendIfConcurrent(b, benchCtx, 100, 1, true)
	})

	// Concurrent AppendIf benchmarks - WITH CONFLICT (100 events)
	b.Run("AppendIf_WithConflict_Concurrent_1User_100Events", func(b *testing.B) {
		BenchmarkAppendIfConcurrent(b, benchCtx, 1, 100, true)
	})

	b.Run("AppendIf_WithConflict_Concurrent_10Users_100Events", func(b *testing.B) {
		BenchmarkAppendIfConcurrent(b, benchCtx, 10, 100, true)
	})

	b.Run("AppendIf_WithConflict_Concurrent_100Users_100Events", func(b *testing.B) {
		BenchmarkAppendIfConcurrent(b, benchCtx, 100, 100, true)
	})

	// Projection benchmarks
	// Use separate setup for projection benchmarks to avoid scanning large datasets
	projectionCtx := SetupProjectionBenchmarkContext(b, datasetSize)

	// Concurrent projection benchmarks (replaces individual Project_1/2 and ProjectStream_1/2)
	b.Run("Project_Concurrent_1User", func(b *testing.B) {
		BenchmarkProjectConcurrent(b, projectionCtx, 1)
	})

	b.Run("Project_Concurrent_10Users", func(b *testing.B) {
		BenchmarkProjectConcurrent(b, projectionCtx, 10)
	})

	b.Run("Project_Concurrent_25Users", func(b *testing.B) {
		BenchmarkProjectConcurrent(b, projectionCtx, 25)
	})

	b.Run("ProjectStream_Concurrent_1User", func(b *testing.B) {
		BenchmarkProjectStreamConcurrent(b, projectionCtx, 1)
	})

	b.Run("ProjectStream_Concurrent_10Users", func(b *testing.B) {
		BenchmarkProjectStreamConcurrent(b, projectionCtx, 10)
	})

	b.Run("ProjectStream_Concurrent_25Users", func(b *testing.B) {
		BenchmarkProjectStreamConcurrent(b, projectionCtx, 25)
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
