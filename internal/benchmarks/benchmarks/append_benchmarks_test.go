package benchmarks

import "testing"

func BenchmarkAppend_Small(b *testing.B) {
	RunAllBenchmarks(b, "small")
}

func BenchmarkAppend_Medium(b *testing.B) {
	RunAllBenchmarks(b, "medium")
}

func BenchmarkAppend_Large(b *testing.B) {
	RunAllBenchmarks(b, "large")
}

func BenchmarkAppend_XLarge(b *testing.B) {
	RunAllBenchmarks(b, "xlarge")
}

// Individual append benchmarks for detailed analysis
func BenchmarkAppendSingle_Small(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "small")
	BenchmarkAppendSingle(b, benchCtx)
}

func BenchmarkAppendSingle_Medium(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "medium")
	BenchmarkAppendSingle(b, benchCtx)
}

func BenchmarkAppendSingle_Large(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "large")
	BenchmarkAppendSingle(b, benchCtx)
}

func BenchmarkAppendBatch10_Small(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "small")
	BenchmarkAppendBatch(b, benchCtx, 10)
}

func BenchmarkAppendBatch10_Medium(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "medium")
	BenchmarkAppendBatch(b, benchCtx, 10)
}

func BenchmarkAppendBatch10_Large(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "large")
	BenchmarkAppendBatch(b, benchCtx, 10)
}

func BenchmarkAppendBatch100_Small(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "small")
	BenchmarkAppendBatch(b, benchCtx, 100)
}

func BenchmarkAppendBatch100_Medium(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "medium")
	BenchmarkAppendBatch(b, benchCtx, 100)
}

func BenchmarkAppendBatch100_Large(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "large")
	BenchmarkAppendBatch(b, benchCtx, 100)
}

func BenchmarkAppendBatch1000_Small(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "small")
	BenchmarkAppendBatch(b, benchCtx, 1000)
}

func BenchmarkAppendBatch1000_Medium(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "medium")
	BenchmarkAppendBatch(b, benchCtx, 1000)
}

func BenchmarkAppendBatch1000_Large(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "large")
	BenchmarkAppendBatch(b, benchCtx, 1000)
}
