package benchmarks

import (
	"testing"
)

// Quick benchmarks for essential operations only
// These provide fast feedback on core performance

func BenchmarkQuickAppend(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "tiny")
	BenchmarkAppendSingle(b, benchCtx)
}

func BenchmarkQuickRead(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "tiny")
	BenchmarkRead(b, benchCtx, 0)
}

func BenchmarkQuickProjection(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "tiny")
	BenchmarkProject(b, benchCtx, 1)
}
