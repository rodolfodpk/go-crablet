package benchmarks

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/rodolfodpk/go-crablet/pkg/dcb"
)

// BenchmarkAppendBatch_1 tests append with batch size 1
func BenchmarkAppendBatch_1(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "small", 100)
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		events := make([]dcb.InputEvent, 1)
		uniqueID := fmt.Sprintf("batch1_%d_%d", time.Now().UnixNano(), i)

		events[0] = dcb.NewInputEvent("TestEvent",
			dcb.NewTags("test", "batch1", "unique_id", uniqueID),
			[]byte(fmt.Sprintf(`{"value": "test", "unique_id": "%s"}`, uniqueID)))

		err := benchCtx.Store.Append(ctx, events)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkAppendBatch_5 tests append with batch size 5
func BenchmarkAppendBatch_5(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "small", 100)
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		events := make([]dcb.InputEvent, 5)
		uniqueID := fmt.Sprintf("batch5_%d_%d", time.Now().UnixNano(), i)

		for j := 0; j < 5; j++ {
			eventID := fmt.Sprintf("%s_%d", uniqueID, j)
			events[j] = dcb.NewInputEvent("TestEvent",
				dcb.NewTags("test", "batch5", "unique_id", eventID),
				[]byte(fmt.Sprintf(`{"value": "test", "unique_id": "%s"}`, eventID)))
		}

		err := benchCtx.Store.Append(ctx, events)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkAppendBatch_12 tests append with batch size 12
func BenchmarkAppendBatch_12(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "small", 100)
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		events := make([]dcb.InputEvent, 12)
		uniqueID := fmt.Sprintf("batch12_%d_%d", time.Now().UnixNano(), i)

		for j := 0; j < 12; j++ {
			eventID := fmt.Sprintf("%s_%d", uniqueID, j)
			events[j] = dcb.NewInputEvent("TestEvent",
				dcb.NewTags("test", "batch12", "unique_id", eventID),
				[]byte(fmt.Sprintf(`{"value": "test", "unique_id": "%s"}`, eventID)))
		}

		err := benchCtx.Store.Append(ctx, events)
		if err != nil {
			b.Fatal(err)
		}
	}
}
