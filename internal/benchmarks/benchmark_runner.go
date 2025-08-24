package benchmarks

import (
	"context"
	"fmt"
	"os"
	"runtime"
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

// BenchmarkContext holds the context for running benchmarks
type BenchmarkContext struct {
	Store        dcb.EventStore
	ChannelStore dcb.EventStore
	HasChannel   bool
	Dataset      *setup.Dataset
	Queries      []dcb.Query
	Projectors   []dcb.StateProjector
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

	// Create new pool with conservative settings
	poolConfig, err := pgxpool.ParseConfig("postgres://crablet:crablet@localhost:5432/crablet")
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

// SetupBenchmarkContext creates a new benchmark context with the specified dataset size
func SetupBenchmarkContext(b *testing.B, datasetSize string) *BenchmarkContext {
	ctx := context.Background()

	// Use the shared global pool
	pool, err := getOrCreateGlobalPool()
	if err != nil {
		b.Fatalf("Failed to get global pool: %v", err)
	}

	// Truncate events table before running benchmarks
	_, err = pool.Exec(ctx, "TRUNCATE TABLE events RESTART IDENTITY CASCADE")
	if err != nil {
		b.Fatalf("Failed to truncate events table: %v", err)
	}

	// Create event stores with different configurations
	readCommittedConfig := dcb.EventStoreConfig{
		MaxBatchSize:           1000,
		StreamBuffer:           1000,
		DefaultAppendIsolation: dcb.IsolationLevelReadCommitted,
		QueryTimeout:           15000,
		AppendTimeout:          15000,
	}

	repeatableReadConfig := dcb.EventStoreConfig{
		MaxBatchSize:           1000,
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

	// Create queries for benchmarking
	queries := []dcb.Query{
		dcb.NewQuery(dcb.NewTags("test", "single")),
		dcb.NewQuery(dcb.NewTags("test", "batch")),
		dcb.NewQuery(dcb.NewTags("test", "appendif")),
		dcb.NewQuery(dcb.NewTags("test", "conflict")),
		dcb.NewQuery(dcb.NewTags("test", "appendifconflict")),
	}

	// Create projectors for benchmarking
	projectors := []dcb.StateProjector{
		{
			ID:           "count",
			Query:        dcb.NewQuery(dcb.NewTags("test", "single")),
			InitialState: 0,
			TransitionFn: func(state any, event dcb.Event) any {
				return state.(int) + 1
			},
		},
		{
			ID:           "sum",
			Query:        dcb.NewQuery(dcb.NewTags("test", "batch")),
			InitialState: 0,
			TransitionFn: func(state any, event dcb.Event) any {
				return state.(int) + 1
			},
		},
	}

	return &BenchmarkContext{
		Store:        store,
		ChannelStore: channelStore,
		HasChannel:   true,
		Dataset:      dataset,
		Queries:      queries,
		Projectors:   projectors,
	}
}

// BenchmarkAppendSingle benchmarks single event append
func BenchmarkAppendSingle(b *testing.B, benchCtx *BenchmarkContext) {
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		uniqueID := fmt.Sprintf("single_%d_%d", time.Now().UnixNano(), i)
		event := dcb.NewInputEvent("TestEvent",
			dcb.NewTags("test", "single", "unique_id", uniqueID),
			[]byte(fmt.Sprintf(`{"value": "test", "unique_id": "%s"}`, uniqueID)))

		err := benchCtx.Store.Append(ctx, []dcb.InputEvent{event})
		if err != nil {
			b.Fatalf("Append failed: %v", err)
		}
	}
}



// BenchmarkAppendRealistic benchmarks realistic batch sizes (1-12 events) for real-world usage
func BenchmarkAppendRealistic(b *testing.B, benchCtx *BenchmarkContext) {
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	// Generate realistic batch sizes at runtime
	realisticSizes := []int{1, 2, 3, 5, 8, 12}
	
	for i := 0; i < b.N; i++ {
		batchSize := realisticSizes[i%len(realisticSizes)]
		events := make([]dcb.InputEvent, batchSize)
		uniqueID := fmt.Sprintf("realistic_%d_%d", time.Now().UnixNano(), i)

		for j := 0; j < batchSize; j++ {
			eventID := fmt.Sprintf("%s_%d", uniqueID, j)
			events[j] = dcb.NewInputEvent("TestEvent",
				dcb.NewTags("test", "realistic", "batch_size", fmt.Sprintf("%d", batchSize), "unique_id", eventID),
				[]byte(fmt.Sprintf(`{"batch_size": %d, "unique_id": "%s"}`, batchSize, eventID)))
		}

		err := benchCtx.Store.Append(ctx, events)
		if err != nil {
			b.Fatalf("Realistic batch append failed: %v", err)
		}
	}
}

// BenchmarkAppendIf benchmarks conditional append with RepeatableRead isolation
func BenchmarkAppendIf(b *testing.B, benchCtx *BenchmarkContext, batchSize int) {
	// Create context with timeout for each benchmark iteration
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		events := make([]dcb.InputEvent, batchSize)
		uniqueID := fmt.Sprintf("appendif_%d_%d", time.Now().UnixNano(), i)

		for j := 0; j < batchSize; j++ {
			eventID := fmt.Sprintf("%s_%d", uniqueID, j)
			events[j] = dcb.NewInputEvent("TestEvent",
				dcb.NewTags("test", "appendif", "unique_id", eventID),
				[]byte(fmt.Sprintf(`{"value": "test", "unique_id": "%s"}`, eventID)))
		}

		// Create a simple condition that should pass (no conflicting events)
		condition := dcb.NewAppendCondition(
			dcb.NewQuery(dcb.NewTags("test", "conflict"), "ConflictingEvent"),
		)

		err := benchCtx.Store.AppendIf(ctx, events, condition)
		if err != nil {
			b.Fatalf("AppendIf failed: %v", err)
		}
	}
}

// BenchmarkAppendIfWithCondition benchmarks conditional append with configurable isolation
// NOTE: The isolation level is configured in the EventStore config.
func BenchmarkAppendIfWithCondition(b *testing.B, benchCtx *BenchmarkContext, batchSize int) {
	// Create context with timeout for each benchmark iteration
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		events := make([]dcb.InputEvent, batchSize)
		uniqueID := fmt.Sprintf("appendif_%d_%d", time.Now().UnixNano(), i)

		for j := 0; j < batchSize; j++ {
			eventID := fmt.Sprintf("%s_%d", uniqueID, j)
			events[j] = dcb.NewInputEvent("TestEvent",
				dcb.NewTags("test", "appendif", "unique_id", eventID),
				[]byte(fmt.Sprintf(`{"value": "test", "unique_id": "%s"}`, eventID)))
		}

		// Create a simple condition that should pass (no conflicting events)
		condition := dcb.NewAppendCondition(
			dcb.NewQuery(dcb.NewTags("test", "conflict"), "ConflictingEvent"),
		)

		err := benchCtx.Store.AppendIf(ctx, events, condition)
		if err != nil {
			b.Fatalf("AppendIf failed: %v", err)
		}
	}
}

// BenchmarkAppendIfWithConflict benchmarks AppendIf with a condition that should fail
func BenchmarkAppendIfWithConflict(b *testing.B, benchCtx *BenchmarkContext, batchSize int) {
	// Create context with timeout for each benchmark iteration
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Create a conflicting event with unique ID for this iteration
		uniqueID := fmt.Sprintf("conflict_%d_%d", time.Now().UnixNano(), i)
		conflictEvent := dcb.NewInputEvent("ConflictingEvent",
			dcb.NewTags("test", "conflict", "unique_id", uniqueID),
			[]byte(fmt.Sprintf(`{"value": "test", "unique_id": "%s"}`, uniqueID)))

		// Append the conflicting event first
		err := benchCtx.Store.Append(ctx, []dcb.InputEvent{conflictEvent})
		if err != nil {
			b.Fatalf("Failed to append conflict event: %v", err)
		}

		// Now try to append with a condition that should fail
		events := make([]dcb.InputEvent, batchSize)
		for j := 0; j < batchSize; j++ {
			eventID := fmt.Sprintf("appendif_%s_%d", uniqueID, j)
			events[j] = dcb.NewInputEvent("TestEvent",
				dcb.NewTags("test", "appendif", "unique_id", eventID),
				[]byte(fmt.Sprintf(`{"value": "test", "unique_id": "%s"}`, eventID)))
		}

		// Create a condition that should fail (conflicting event exists)
		condition := dcb.NewAppendCondition(
			dcb.NewQuery(dcb.NewTags("test", "conflict", "unique_id", uniqueID), "ConflictingEvent"),
		)

		// This should fail due to the conflicting event
		err = benchCtx.Store.AppendIf(ctx, events, condition)
		if err == nil {
			b.Fatalf("AppendIf should have failed due to conflict")
		}
	}
}



// BenchmarkAppendMixedEventTypes benchmarks append with mixed event types (matching web-app scenarios)
func BenchmarkAppendMixedEventTypes(b *testing.B, benchCtx *BenchmarkContext, batchSize int) {
	// Create context with timeout for each benchmark iteration
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	b.ResetTimer()
	b.ReportAllocs()

	eventTypes := []string{"UserCreated", "AccountOpened", "TransactionInitiated", "NotificationSent", "AuditLog"}

	for i := 0; i < b.N; i++ {
		events := make([]dcb.InputEvent, batchSize)
		uniqueID := fmt.Sprintf("mixed_%d_%d", time.Now().UnixNano(), i)

		for j := 0; j < batchSize; j++ {
			eventID := fmt.Sprintf("%s_%d", uniqueID, j)
			eventType := eventTypes[j%len(eventTypes)]
			events[j] = dcb.NewInputEvent(eventType,
				dcb.NewTags("test", "mixed", "unique_id", eventID),
				[]byte(fmt.Sprintf(`{"value": "test", "unique_id": "%s", "type": "%s"}`, eventID, eventType)))
		}

		err := benchCtx.Store.Append(ctx, events)
		if err != nil {
			b.Fatalf("Mixed event types append failed: %v", err)
		}
	}
}

// BenchmarkAppendHighFrequency benchmarks high-frequency event append (matching web-app scenarios)
func BenchmarkAppendHighFrequency(b *testing.B, benchCtx *BenchmarkContext, batchSize int) {
	// Create context with timeout for each benchmark iteration
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		events := make([]dcb.InputEvent, batchSize)
		uniqueID := fmt.Sprintf("highfreq_%d_%d", time.Now().UnixNano(), i)

		for j := 0; j < batchSize; j++ {
			eventID := fmt.Sprintf("%s_%d", uniqueID, j)
			events[j] = dcb.NewInputEvent("SensorReading",
				dcb.NewTags("sensor", fmt.Sprintf("sensor_%d", j), "location", "data_center", "type", "temperature"),
				[]byte(fmt.Sprintf(`{"value": %d, "timestamp": "%d", "sensor_id": "%s"}`, j, time.Now().UnixNano(), eventID)))
		}

		err := benchCtx.Store.Append(ctx, events)
		if err != nil {
			b.Fatalf("High frequency append failed: %v", err)
		}
	}
}

// BenchmarkRead benchmarks event reading
func BenchmarkRead(b *testing.B, benchCtx *BenchmarkContext, queryIndex int) {
	// Create context with timeout for each benchmark iteration
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if queryIndex >= len(benchCtx.Queries) {
		b.Fatalf("Query index out of range: %d", queryIndex)
	}

	query := benchCtx.Queries[queryIndex]

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		events, err := benchCtx.Store.Query(ctx, query, nil)
		if err != nil {
			b.Fatalf("Read failed: %v", err)
		}
		_ = events // Prevent compiler optimization
	}
}

// BenchmarkReadChannel benchmarks channel-based event reading
func BenchmarkReadChannel(b *testing.B, benchCtx *BenchmarkContext, queryIndex int) {
	// Create context with timeout for each benchmark iteration
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if queryIndex >= len(benchCtx.Queries) {
		b.Fatalf("Query index out of range: %d", queryIndex)
	}

	query := benchCtx.Queries[queryIndex]

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		eventChan, err := benchCtx.ChannelStore.QueryStream(ctx, query, nil)
		if err != nil {
			b.Fatalf("ReadStream failed: %v", err)
		}

		count := 0
		for range eventChan {
			count++
		}
		_ = count // Prevent compiler optimization
	}
}

// BenchmarkProject benchmarks state projection
func BenchmarkProject(b *testing.B, benchCtx *BenchmarkContext, projectorCount int) {
	// Create context with timeout for each benchmark iteration
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if projectorCount > len(benchCtx.Projectors) {
		b.Fatalf("Projector count out of range: %d", projectorCount)
	}

	projectors := benchCtx.Projectors[:projectorCount]

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		states, _, err := benchCtx.Store.Project(ctx, projectors, nil)
		if err != nil {
			b.Fatalf("Project failed: %v", err)
		}
		_ = states // Prevent compiler optimization
	}
}

// BenchmarkProjectStream benchmarks channel-based state projection
func BenchmarkProjectStream(b *testing.B, benchCtx *BenchmarkContext, projectorCount int) {
	// Create context with timeout for each benchmark iteration
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if projectorCount > len(benchCtx.Projectors) {
		b.Fatalf("Projector count out of range: %d", projectorCount)
	}

	projectors := benchCtx.Projectors[:projectorCount]

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		stateChan, _, err := benchCtx.ChannelStore.ProjectStream(ctx, projectors, nil)
		if err != nil {
			b.Fatalf("ProjectStream failed: %v", err)
		}

		count := 0
		for range stateChan {
			count++
		}
		_ = count // Prevent compiler optimization
	}
}

// BenchmarkMemoryUsage benchmarks memory usage for different operations
func BenchmarkMemoryUsage(b *testing.B, benchCtx *BenchmarkContext, operation string) {
	// Create context with timeout for each benchmark iteration
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	initialAlloc := m.Alloc

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		switch operation {
		case "read":
			events, err := benchCtx.Store.Query(ctx, benchCtx.Queries[0], nil)
			if err != nil {
				b.Fatalf("Read failed: %v", err)
			}
			_ = events
		case "read_stream", "stream":
			eventChan, err := benchCtx.ChannelStore.QueryStream(ctx, benchCtx.Queries[0], nil)
			if err != nil {
				b.Fatalf("ReadStream failed: %v", err)
			}
			count := 0
			for range eventChan {
				count++
			}
			_ = count
		case "project":
			states, _, err := benchCtx.Store.Project(ctx, benchCtx.Projectors, nil)
			if err != nil {
				b.Fatalf("Project failed: %v", err)
			}
			_ = states
		case "project_stream":
			stateChan, _, err := benchCtx.ChannelStore.ProjectStream(ctx, benchCtx.Projectors, nil)
			if err != nil {
				b.Fatalf("ProjectStream failed: %v", err)
			}
			count := 0
			for range stateChan {
				count++
			}
			_ = count
		default:
			b.Fatalf("Unknown operation: %s", operation)
		}
	}

	runtime.ReadMemStats(&m)
	finalAlloc := m.Alloc
	b.ReportMetric(float64(finalAlloc-initialAlloc), "bytes/op")
}

// RunAllBenchmarks runs all benchmarks with the specified dataset size
func RunAllBenchmarks(b *testing.B, datasetSize string) {
	benchCtx := SetupBenchmarkContext(b, datasetSize)

	// Append benchmarks
	b.Run("AppendSingle", func(b *testing.B) {
		BenchmarkAppendSingle(b, benchCtx)
	})

	// Realistic append benchmarks (1-12 events, most common real-world scenarios)
	b.Run("AppendRealistic", func(b *testing.B) {
		BenchmarkAppendRealistic(b, benchCtx)
	})

	// Conditional append benchmarks (realistic batch sizes)
	b.Run("AppendIf_1", func(b *testing.B) {
		BenchmarkAppendIf(b, benchCtx, 1)
	})

	b.Run("AppendIf_5", func(b *testing.B) {
		BenchmarkAppendIf(b, benchCtx, 5)
	})

	b.Run("AppendIf_12", func(b *testing.B) {
		BenchmarkAppendIf(b, benchCtx, 12)
	})

	// Conflict benchmarks (realistic batch sizes)
	b.Run("AppendIfWithConflict_1", func(b *testing.B) {
		BenchmarkAppendIfWithConflict(b, benchCtx, 1)
	})

	b.Run("AppendIfWithConflict_5", func(b *testing.B) {
		BenchmarkAppendIfWithConflict(b, benchCtx, 5)
	})

	// Read benchmarks
	b.Run("Read_Single", func(b *testing.B) {
		BenchmarkRead(b, benchCtx, 0)
	})

	b.Run("Read_Batch", func(b *testing.B) {
		BenchmarkRead(b, benchCtx, 1)
	})

	b.Run("Read_AppendIf", func(b *testing.B) {
		BenchmarkRead(b, benchCtx, 2)
	})

	// Channel read benchmarks
	b.Run("ReadChannel_Single", func(b *testing.B) {
		BenchmarkReadChannel(b, benchCtx, 0)
	})

	b.Run("ReadChannel_Batch", func(b *testing.B) {
		BenchmarkReadChannel(b, benchCtx, 1)
	})

	// Projection benchmarks
	b.Run("Project_1", func(b *testing.B) {
		BenchmarkProject(b, benchCtx, 1)
	})

	b.Run("Project_2", func(b *testing.B) {
		BenchmarkProject(b, benchCtx, 2)
	})

	// Channel projection benchmarks
	b.Run("ProjectStream_1", func(b *testing.B) {
		BenchmarkProjectStream(b, benchCtx, 1)
	})

	b.Run("ProjectStream_2", func(b *testing.B) {
		BenchmarkProjectStream(b, benchCtx, 2)
	})

	// Memory usage benchmarks
	b.Run("MemoryUsage_Read", func(b *testing.B) {
		BenchmarkMemoryUsage(b, benchCtx, "read")
	})

	b.Run("MemoryUsage_ReadStream", func(b *testing.B) {
		BenchmarkMemoryUsage(b, benchCtx, "read_stream")
	})

	b.Run("MemoryUsage_Project", func(b *testing.B) {
		BenchmarkMemoryUsage(b, benchCtx, "project")
	})

	b.Run("MemoryUsage_ProjectStream", func(b *testing.B) {
		BenchmarkMemoryUsage(b, benchCtx, "project_stream")
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
