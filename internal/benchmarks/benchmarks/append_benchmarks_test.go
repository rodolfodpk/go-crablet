package benchmarks

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
