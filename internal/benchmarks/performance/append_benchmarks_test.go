package performance

import "testing"

func BenchmarkAppend_Small(b *testing.B) {
	RunAllBenchmarks(b, "small")
}

func BenchmarkAppend_Tiny(b *testing.B) {
	RunAllBenchmarks(b, "tiny")
}

// Individual append benchmarks for detailed analysis
func BenchmarkAppendSingle_Small(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "small")
	BenchmarkAppendSingle(b, benchCtx)
}

func BenchmarkAppendSingle_Tiny(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "tiny")
	BenchmarkAppendSingle(b, benchCtx)
}

func BenchmarkAppendBatch10_Small(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "small")
	BenchmarkAppendBatch(b, benchCtx, 10)
}

func BenchmarkAppendBatch10_Tiny(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "tiny")
	BenchmarkAppendBatch(b, benchCtx, 10)
}

func BenchmarkAppendBatch100_Small(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "small")
	BenchmarkAppendBatch(b, benchCtx, 100)
}

func BenchmarkAppendBatch100_Tiny(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "tiny")
	BenchmarkAppendBatch(b, benchCtx, 100)
}

func BenchmarkAppendBatch1000_Small(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "small")
	BenchmarkAppendBatch(b, benchCtx, 1000)
}

func BenchmarkAppendBatch1000_Tiny(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "tiny")
	BenchmarkAppendBatch(b, benchCtx, 1000)
}

// AppendIf benchmarks (RepeatableRead isolation)
func BenchmarkAppendIf_Small(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "small")
	BenchmarkAppendIf(b, benchCtx, 1)
}

func BenchmarkAppendIf_Tiny(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "tiny")
	BenchmarkAppendIf(b, benchCtx, 1)
}

func BenchmarkAppendIfBatch10_Small(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "small")
	BenchmarkAppendIf(b, benchCtx, 10)
}

func BenchmarkAppendIfBatch10_Tiny(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "tiny")
	BenchmarkAppendIf(b, benchCtx, 10)
}

func BenchmarkAppendIfBatch100_Small(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "small")
	BenchmarkAppendIf(b, benchCtx, 100)
}

func BenchmarkAppendIfBatch100_Tiny(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "tiny")
	BenchmarkAppendIf(b, benchCtx, 100)
}

// Update all benchmark functions and comments: SERIALIZABLE isolation must be set in the config before running. All calls use AppendIf.
// Conflict scenario benchmarks
func BenchmarkAppendIfWithConflict_Small(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "small")
	BenchmarkAppendIfWithConflict(b, benchCtx, 1)
}

func BenchmarkAppendIfWithConflict_Tiny(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "tiny")
	BenchmarkAppendIfWithConflict(b, benchCtx, 1)
}

// Mixed event types benchmarks (matching web-app scenarios)
func BenchmarkAppendMixedEventTypes_Small(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "small")
	BenchmarkAppendMixedEventTypes(b, benchCtx, 5)
}

func BenchmarkAppendMixedEventTypes_Tiny(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "tiny")
	BenchmarkAppendMixedEventTypes(b, benchCtx, 5)
}

// High frequency event benchmarks (matching web-app scenarios)
func BenchmarkAppendHighFrequency_Small(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "small")
	BenchmarkAppendHighFrequency(b, benchCtx, 50)
}

func BenchmarkAppendHighFrequency_Tiny(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "tiny")
	BenchmarkAppendHighFrequency(b, benchCtx, 50)
}

// Realistic batch size benchmarks (most common real-world scenarios)
func BenchmarkAppendRealistic_Small(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "small")
	BenchmarkAppendRealistic(b, benchCtx)
}

func BenchmarkAppendRealistic_Tiny(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "tiny")
	BenchmarkAppendRealistic(b, benchCtx)
}
