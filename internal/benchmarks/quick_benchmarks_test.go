package benchmarks

import (
	"testing"
)

// Quick benchmarks for essential operations only
// These provide fast feedback on core performance

func BenchmarkQuickAppend(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "tiny", 10)
	BenchmarkAppendSingle(b, benchCtx)
}

func BenchmarkQuickRead(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "tiny", 10)
	BenchmarkRead(b, benchCtx, 0)
}

func BenchmarkQuickProjection(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "tiny", 10)
	BenchmarkProject(b, benchCtx, 1)
}
