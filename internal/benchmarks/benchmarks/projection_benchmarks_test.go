package benchmarks

import "testing"

func BenchmarkProjection_Small(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "small")

	b.Run("Project1", func(b *testing.B) {
		BenchmarkProject(b, benchCtx, 1)
	})

	b.Run("Project2", func(b *testing.B) {
		BenchmarkProject(b, benchCtx, 2)
	})

	if benchCtx.HasChannel {
		b.Run("ProjectStream1", func(b *testing.B) {
			BenchmarkProjectStream(b, benchCtx, 1)
		})

		b.Run("ProjectStream2", func(b *testing.B) {
			BenchmarkProjectStream(b, benchCtx, 2)
		})
	}

	b.Run("MemoryProjection", func(b *testing.B) {
		BenchmarkMemoryUsage(b, benchCtx, "project")
	})
}

func BenchmarkProjection_Tiny(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "tiny")

	b.Run("Project1", func(b *testing.B) {
		BenchmarkProject(b, benchCtx, 1)
	})

	b.Run("Project2", func(b *testing.B) {
		BenchmarkProject(b, benchCtx, 2)
	})

	if benchCtx.HasChannel {
		b.Run("ProjectStream1", func(b *testing.B) {
			BenchmarkProjectStream(b, benchCtx, 1)
		})

		b.Run("ProjectStream2", func(b *testing.B) {
			BenchmarkProjectStream(b, benchCtx, 2)
		})
	}

	b.Run("MemoryProjection", func(b *testing.B) {
		BenchmarkMemoryUsage(b, benchCtx, "project")
	})
}

// Individual projection benchmarks for detailed analysis
func BenchmarkProject1_Small(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "small")
	BenchmarkProject(b, benchCtx, 1)
}

func BenchmarkProject1_Tiny(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "tiny")
	BenchmarkProject(b, benchCtx, 1)
}

func BenchmarkProject2_Small(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "small")
	BenchmarkProject(b, benchCtx, 2)
}

func BenchmarkProject2_Tiny(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "tiny")
	BenchmarkProject(b, benchCtx, 2)
}

func BenchmarkProjectStream1_Small(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "small")
	BenchmarkProjectStream(b, benchCtx, 1)
}

func BenchmarkProjectStream1_Tiny(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "tiny")
	BenchmarkProjectStream(b, benchCtx, 1)
}

func BenchmarkProjectStream2_Small(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "small")
	BenchmarkProjectStream(b, benchCtx, 2)
}

func BenchmarkProjectStream2_Tiny(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "tiny")
	BenchmarkProjectStream(b, benchCtx, 2)
}

// Memory usage benchmarks for projection operations
func BenchmarkMemoryProjection_Small(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "small")
	BenchmarkMemoryUsage(b, benchCtx, "project")
}

func BenchmarkMemoryProjection_Tiny(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "tiny")
	BenchmarkMemoryUsage(b, benchCtx, "project")
}
