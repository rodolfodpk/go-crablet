package benchmarks

import (
	"context"
	"fmt"
	"runtime"
	"testing"
	"time"

	"go-crablet/pkg/dcb"

	"go-crablet/internal/benchmarks/setup"

	"github.com/jackc/pgx/v5/pgxpool"
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

// SetupBenchmarkContext creates a benchmark context with test data
func SetupBenchmarkContext(b *testing.B, datasetSize string) *BenchmarkContext {
	// Create context with timeout for benchmark setup
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

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
			b.Fatalf("Failed to connect to database after %d retries. Make sure docker-compose is running: docker-compose up -d", maxRetries)
		}

		time.Sleep(1 * time.Second)
	}

	// Connect to database
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		b.Fatalf("Failed to connect to database: %v", err)
	}

	// Create event store
	store, err := dcb.NewEventStore(ctx, pool)
	if err != nil {
		b.Fatalf("Failed to create event store: %v", err)
	}

	// Check if EventStore is available
	store, hasChannel := store.(dcb.EventStore)

	// Get dataset configuration
	config, exists := setup.DatasetSizes[datasetSize]
	if !exists {
		b.Fatalf("Unknown dataset size: %s", datasetSize)
	}

	// Get dataset from cache (or generate and cache it)
	dataset, err := setup.GetCachedDataset(config)
	if err != nil {
		b.Fatalf("Failed to get cached dataset: %v", err)
	}

	// Load dataset into store
	if err := setup.LoadDatasetIntoStore(ctx, store, dataset); err != nil {
		b.Fatalf("Failed to load dataset: %v", err)
	}

	// Generate queries and projectors
	queries := setup.GenerateRandomQueries(dataset, 100)
	projectors := setup.CreateBenchmarkProjectors(dataset)

	benchCtx := &BenchmarkContext{
		Store:        store,
		ChannelStore: store,
		HasChannel:   hasChannel,
		Dataset:      dataset,
		Queries:      queries,
		Projectors:   projectors,
	}

	// Cleanup function
	b.Cleanup(func() {
		pool.Close()
	})

	return benchCtx
}

// BenchmarkAppendSingle benchmarks single event append
func BenchmarkAppendSingle(b *testing.B, benchCtx *BenchmarkContext) {
	// Create context with timeout for each benchmark iteration
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Use unique data to avoid collisions
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

// BenchmarkAppendBatch benchmarks batch event append
func BenchmarkAppendBatch(b *testing.B, benchCtx *BenchmarkContext, batchSize int) {
	// Create context with timeout for each benchmark iteration
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		events := make([]dcb.InputEvent, batchSize)
		uniqueID := fmt.Sprintf("batch_%d_%d", time.Now().UnixNano(), i)

		for j := 0; j < batchSize; j++ {
			eventID := fmt.Sprintf("%s_%d", uniqueID, j)
			events[j] = dcb.NewInputEvent("TestEvent",
				dcb.NewTags("test", "batch", "unique_id", eventID),
				[]byte(fmt.Sprintf(`{"value": "test", "unique_id": "%s"}`, eventID)))
		}

		err := benchCtx.Store.Append(ctx, events)
		if err != nil {
			b.Fatalf("Batch append failed: %v", err)
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
			[]byte(fmt.Sprintf(`{"value": "conflict", "unique_id": "%s"}`, uniqueID)))

		err := benchCtx.Store.Append(ctx, []dcb.InputEvent{conflictEvent})
		if err != nil {
			b.Fatalf("Failed to create conflict event: %v", err)
		}

		events := make([]dcb.InputEvent, batchSize)
		for j := 0; j < batchSize; j++ {
			eventID := fmt.Sprintf("%s_%d", uniqueID, j)
			events[j] = dcb.NewInputEvent("TestEvent",
				dcb.NewTags("test", "appendifconflict", "unique_id", eventID),
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

// BenchmarkAppendIfWithConflictCondition benchmarks AppendIf with a condition that should fail
// NOTE: The isolation level is configured in the EventStore config.
func BenchmarkAppendIfWithConflictCondition(b *testing.B, benchCtx *BenchmarkContext, batchSize int) {
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
			[]byte(fmt.Sprintf(`{"value": "conflict", "unique_id": "%s"}`, uniqueID)))

		err := benchCtx.Store.Append(ctx, []dcb.InputEvent{conflictEvent})
		if err != nil {
			b.Fatalf("Failed to create conflict event: %v", err)
		}

		events := make([]dcb.InputEvent, batchSize)
		for j := 0; j < batchSize; j++ {
			eventID := fmt.Sprintf("%s_%d", uniqueID, j)
			events[j] = dcb.NewInputEvent("TestEvent",
				dcb.NewTags("test", "appendifconflict", "unique_id", eventID),
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
		_, err := benchCtx.Store.Read(ctx, query)
		if err != nil {
			b.Fatalf("Read failed: %v", err)
		}
	}
}

// BenchmarkReadStream has been removed - use Read instead for batch reading
// ReadStream was replaced with ReadChannel for streaming operations

// BenchmarkReadChannel benchmarks event streaming with channels
func BenchmarkReadChannel(b *testing.B, benchCtx *BenchmarkContext, queryIndex int) {
	if !benchCtx.HasChannel {
		b.Skip("Channel streaming not available")
	}

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
		eventChan, err := benchCtx.ChannelStore.ReadStream(ctx, query)
		if err != nil {
			b.Fatalf("ReadStream failed: %v", err)
		}

		count := 0
		for range eventChan {
			count++
		}

		if count == 0 && i == 0 {
			b.Logf("Warning: No events found for query")
		}
	}
}

// BenchmarkProject benchmarks decision model projection
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
		_, _, err := benchCtx.Store.Project(ctx, projectors)
		if err != nil {
			b.Fatalf("Project failed: %v", err)
		}
	}
}

// BenchmarkProjectStream benchmarks channel-based decision model projection
func BenchmarkProjectStream(b *testing.B, benchCtx *BenchmarkContext, projectorCount int) {
	if !benchCtx.HasChannel {
		b.Skip("Channel streaming not available")
	}

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
		resultChan, _, err := benchCtx.ChannelStore.ProjectStream(ctx, projectors)
		if err != nil {
			b.Fatalf("ProjectStream failed: %v", err)
		}

		finalStates := <-resultChan
		if len(finalStates) == 0 && i == 0 {
			b.Logf("Warning: No projection results found")
		}
	}
}

// BenchmarkMemoryUsage benchmarks memory usage for different operations
func BenchmarkMemoryUsage(b *testing.B, benchCtx *BenchmarkContext, operation string) {
	// Create context with timeout for each benchmark iteration
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		switch operation {
		case "read":
			query := dcb.NewQuery(dcb.NewTags(), "StudentEnrolledInCourse")
			_, err := benchCtx.Store.Read(ctx, query)
			if err != nil {
				b.Fatalf("Read failed: %v", err)
			}
		case "stream":
			query := dcb.NewQuery(dcb.NewTags(), "StudentEnrolledInCourse")
			eventChan, err := benchCtx.ChannelStore.ReadStream(ctx, query)
			if err != nil {
				b.Fatalf("ReadStream failed: %v", err)
			}
			for range eventChan {
				// Just iterate through events
			}
		case "projection":
			_, _, err := benchCtx.ChannelStore.Project(ctx, benchCtx.Projectors)
			if err != nil {
				b.Fatalf("Project failed: %v", err)
			}
		default:
			b.Fatalf("Unknown operation: %s", operation)
		}
	}

	runtime.ReadMemStats(&m2)
	b.ReportMetric(float64(m2.Alloc-m1.Alloc)/float64(b.N), "bytes/op")
}

// RunAllBenchmarks runs a comprehensive set of benchmarks
func RunAllBenchmarks(b *testing.B, datasetSize string) {
	benchCtx := SetupBenchmarkContext(b, datasetSize)

	b.Run("AppendSingle", func(b *testing.B) {
		BenchmarkAppendSingle(b, benchCtx)
	})

	b.Run("AppendBatch10", func(b *testing.B) {
		BenchmarkAppendBatch(b, benchCtx, 10)
	})

	b.Run("AppendBatch100", func(b *testing.B) {
		BenchmarkAppendBatch(b, benchCtx, 100)
	})

	b.Run("AppendBatch1000", func(b *testing.B) {
		BenchmarkAppendBatch(b, benchCtx, 1000)
	})

	b.Run("ReadSimple", func(b *testing.B) {
		BenchmarkRead(b, benchCtx, 0)
	})

	b.Run("ReadComplex", func(b *testing.B) {
		BenchmarkRead(b, benchCtx, 1)
	})

	if benchCtx.HasChannel {
		b.Run("ReadChannel", func(b *testing.B) {
			BenchmarkReadChannel(b, benchCtx, 0)
		})
	}

	b.Run("Project1", func(b *testing.B) {
		BenchmarkProject(b, benchCtx, 1)
	})

	b.Run("Project5", func(b *testing.B) {
		BenchmarkProject(b, benchCtx, 5)
	})

	if benchCtx.HasChannel {
		b.Run("ProjectStream1", func(b *testing.B) {
			BenchmarkProjectStream(b, benchCtx, 1)
		})

		b.Run("ProjectStream5", func(b *testing.B) {
			BenchmarkProjectStream(b, benchCtx, 5)
		})
	}

	b.Run("MemoryRead", func(b *testing.B) {
		BenchmarkMemoryUsage(b, benchCtx, "read")
	})

	b.Run("MemoryStream", func(b *testing.B) {
		BenchmarkMemoryUsage(b, benchCtx, "stream")
	})

	b.Run("MemoryProjection", func(b *testing.B) {
		BenchmarkMemoryUsage(b, benchCtx, "projection")
	})
}
