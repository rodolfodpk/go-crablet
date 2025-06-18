package benchmarks

import (
	"context"
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/rodolfodpk/go-crablet/internal/benchmarks/setup"
	"github.com/rodolfodpk/go-crablet/pkg/dcb"

	"github.com/jackc/pgx/v5/pgxpool"
)

// BenchmarkContext holds the context for running benchmarks
type BenchmarkContext struct {
	Store        dcb.EventStore
	ChannelStore dcb.ChannelEventStore
	HasChannel   bool
	Dataset      *setup.Dataset
	Queries      []dcb.Query
	Projectors   []dcb.BatchProjector
}

// SetupBenchmarkContext creates a benchmark context with test data
func SetupBenchmarkContext(b *testing.B, datasetSize string) *BenchmarkContext {
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

	// Check if ChannelEventStore is available
	channelStore, hasChannel := store.(dcb.ChannelEventStore)

	// Generate dataset
	config, exists := setup.DatasetSizes[datasetSize]
	if !exists {
		b.Fatalf("Unknown dataset size: %s", datasetSize)
	}

	dataset := setup.GenerateDataset(config)

	// Load dataset into store
	if err := setup.LoadDatasetIntoStore(ctx, store, dataset); err != nil {
		b.Fatalf("Failed to load dataset: %v", err)
	}

	// Generate queries and projectors
	queries := setup.GenerateRandomQueries(dataset, 100)
	projectors := setup.CreateBenchmarkProjectors(dataset)

	benchCtx := &BenchmarkContext{
		Store:        store,
		ChannelStore: channelStore,
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
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Create a batch with a single event to demonstrate batch append
		event := dcb.NewInputEvent("TestEvent",
			dcb.NewTags("test", "single", "iteration", fmt.Sprintf("%d", i)),
			[]byte(`{"value": "test"}`))

		_, err := benchCtx.Store.Append(ctx, []dcb.InputEvent{event}, nil)
		if err != nil {
			b.Fatalf("Append failed: %v", err)
		}
	}
}

// BenchmarkAppendBatch benchmarks batch event append
func BenchmarkAppendBatch(b *testing.B, benchCtx *BenchmarkContext, batchSize int) {
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		events := make([]dcb.InputEvent, batchSize)
		for j := 0; j < batchSize; j++ {
			events[j] = dcb.NewInputEvent("TestEvent",
				dcb.NewTags("test", "batch", "iteration", fmt.Sprintf("%d", i), "index", fmt.Sprintf("%d", j)),
				[]byte(`{"value": "test"}`))
		}

		_, err := benchCtx.Store.Append(ctx, events, nil)
		if err != nil {
			b.Fatalf("Batch append failed: %v", err)
		}
	}
}

// BenchmarkRead benchmarks event reading
func BenchmarkRead(b *testing.B, benchCtx *BenchmarkContext, queryIndex int) {
	ctx := context.Background()

	if queryIndex >= len(benchCtx.Queries) {
		b.Fatalf("Query index out of range: %d", queryIndex)
	}

	query := benchCtx.Queries[queryIndex]

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := benchCtx.Store.Read(ctx, query, nil)
		if err != nil {
			b.Fatalf("Read failed: %v", err)
		}
	}
}

// BenchmarkReadStream benchmarks event streaming with iterator
func BenchmarkReadStream(b *testing.B, benchCtx *BenchmarkContext, queryIndex int) {
	ctx := context.Background()

	if queryIndex >= len(benchCtx.Queries) {
		b.Fatalf("Query index out of range: %d", queryIndex)
	}

	query := benchCtx.Queries[queryIndex]

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		iterator, err := benchCtx.Store.ReadStream(ctx, query, nil)
		if err != nil {
			b.Fatalf("ReadStream failed: %v", err)
		}

		count := 0
		for iterator.Next() {
			count++
		}
		iterator.Close()

		if count == 0 && i == 0 {
			b.Logf("Warning: No events found for query")
		}
	}
}

// BenchmarkReadStreamChannel benchmarks event streaming with channels
func BenchmarkReadStreamChannel(b *testing.B, benchCtx *BenchmarkContext, queryIndex int) {
	if !benchCtx.HasChannel {
		b.Skip("Channel streaming not available")
	}

	ctx := context.Background()

	if queryIndex >= len(benchCtx.Queries) {
		b.Fatalf("Query index out of range: %d", queryIndex)
	}

	query := benchCtx.Queries[queryIndex]

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		eventChan, err := benchCtx.ChannelStore.ReadStreamChannel(ctx, query, nil)
		if err != nil {
			b.Fatalf("ReadStreamChannel failed: %v", err)
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

// BenchmarkProjectDecisionModel benchmarks decision model projection
func BenchmarkProjectDecisionModel(b *testing.B, benchCtx *BenchmarkContext, projectorCount int) {
	ctx := context.Background()

	if projectorCount > len(benchCtx.Projectors) {
		b.Fatalf("Projector count out of range: %d", projectorCount)
	}

	projectors := benchCtx.Projectors[:projectorCount]

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _, err := benchCtx.Store.ProjectDecisionModel(ctx, projectors, nil)
		if err != nil {
			b.Fatalf("ProjectDecisionModel failed: %v", err)
		}
	}
}

// BenchmarkProjectDecisionModelChannel benchmarks channel-based decision model projection
func BenchmarkProjectDecisionModelChannel(b *testing.B, benchCtx *BenchmarkContext, projectorCount int) {
	if !benchCtx.HasChannel {
		b.Skip("Channel streaming not available")
	}

	ctx := context.Background()

	if projectorCount > len(benchCtx.Projectors) {
		b.Fatalf("Projector count out of range: %d", projectorCount)
	}

	projectors := benchCtx.Projectors[:projectorCount]

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		resultChan, err := benchCtx.ChannelStore.ProjectDecisionModelChannel(ctx, projectors, nil)
		if err != nil {
			b.Fatalf("ProjectDecisionModelChannel failed: %v", err)
		}

		count := 0
		for range resultChan {
			count++
		}

		if count == 0 && i == 0 {
			b.Logf("Warning: No projection results found")
		}
	}
}

// BenchmarkMemoryUsage benchmarks memory usage for different operations
func BenchmarkMemoryUsage(b *testing.B, benchCtx *BenchmarkContext, operation string) {
	ctx := context.Background()

	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		switch operation {
		case "read":
			query := dcb.NewQuery(dcb.NewTags(), "StudentEnrolledInCourse")
			_, err := benchCtx.Store.Read(ctx, query, nil)
			if err != nil {
				b.Fatalf("Read failed: %v", err)
			}
		case "stream":
			query := dcb.NewQuery(dcb.NewTags(), "StudentEnrolledInCourse")
			iterator, err := benchCtx.Store.ReadStream(ctx, query, nil)
			if err != nil {
				b.Fatalf("ReadStream failed: %v", err)
			}
			for iterator.Next() {
				// Just iterate through events
			}
			iterator.Close()
		case "projection":
			_, _, err := benchCtx.Store.ProjectDecisionModel(ctx, benchCtx.Projectors, nil)
			if err != nil {
				b.Fatalf("ProjectDecisionModel failed: %v", err)
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

	b.Run("ReadStream", func(b *testing.B) {
		BenchmarkReadStream(b, benchCtx, 0)
	})

	if benchCtx.HasChannel {
		b.Run("ReadStreamChannel", func(b *testing.B) {
			BenchmarkReadStreamChannel(b, benchCtx, 0)
		})
	}

	b.Run("ProjectDecisionModel1", func(b *testing.B) {
		BenchmarkProjectDecisionModel(b, benchCtx, 1)
	})

	b.Run("ProjectDecisionModel5", func(b *testing.B) {
		BenchmarkProjectDecisionModel(b, benchCtx, 5)
	})

	if benchCtx.HasChannel {
		b.Run("ProjectDecisionModelChannel1", func(b *testing.B) {
			BenchmarkProjectDecisionModelChannel(b, benchCtx, 1)
		})

		b.Run("ProjectDecisionModelChannel5", func(b *testing.B) {
			BenchmarkProjectDecisionModelChannel(b, benchCtx, 5)
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
