package benchmarks

import "testing"

func BenchmarkProjection_Small(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "small")

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

	b.Run("MemoryProjection", func(b *testing.B) {
		BenchmarkMemoryUsage(b, benchCtx, "projection")
	})
}

func BenchmarkProjection_Tiny(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "tiny")

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

	b.Run("MemoryProjection", func(b *testing.B) {
		BenchmarkMemoryUsage(b, benchCtx, "projection")
	})
}

// Individual projection benchmarks for detailed analysis
func BenchmarkProjectDecisionModel1_Small(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "small")
	BenchmarkProjectDecisionModel(b, benchCtx, 1)
}

func BenchmarkProjectDecisionModel1_Tiny(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "tiny")
	BenchmarkProjectDecisionModel(b, benchCtx, 1)
}

func BenchmarkProjectDecisionModel5_Small(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "small")
	BenchmarkProjectDecisionModel(b, benchCtx, 5)
}

func BenchmarkProjectDecisionModel5_Tiny(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "tiny")
	BenchmarkProjectDecisionModel(b, benchCtx, 5)
}

func BenchmarkProjectDecisionModelChannel1_Small(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "small")
	BenchmarkProjectDecisionModelChannel(b, benchCtx, 1)
}

func BenchmarkProjectDecisionModelChannel1_Tiny(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "tiny")
	BenchmarkProjectDecisionModelChannel(b, benchCtx, 1)
}

func BenchmarkProjectDecisionModelChannel5_Small(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "small")
	BenchmarkProjectDecisionModelChannel(b, benchCtx, 5)
}

func BenchmarkProjectDecisionModelChannel5_Tiny(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "tiny")
	BenchmarkProjectDecisionModelChannel(b, benchCtx, 5)
}

// Memory usage benchmarks for projection operations
func BenchmarkMemoryProjection_Small(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "small")
	BenchmarkMemoryUsage(b, benchCtx, "projection")
}

func BenchmarkMemoryProjection_Tiny(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "tiny")
	BenchmarkMemoryUsage(b, benchCtx, "projection")
}
