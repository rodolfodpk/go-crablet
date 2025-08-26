package benchmarks

import "testing"

// Core projection benchmark suite
func BenchmarkProjection_Small(b *testing.B) {
	// Use 100 past events for realistic AppendIf testing
	benchCtx := SetupBenchmarkContext(b, "medium", 100)

	b.Run("Project1", func(b *testing.B) {
		BenchmarkProject(b, benchCtx, 1)
	})

	b.Run("Project2", func(b *testing.B) {
		BenchmarkProject(b, benchCtx, 2)
	})

	b.Run("Project5", func(b *testing.B) {
		BenchmarkProject(b, benchCtx, 5)
	})

	b.Run("Project10", func(b *testing.B) {
		BenchmarkProject(b, benchCtx, 10)
	})

	b.Run("Project20", func(b *testing.B) {
		BenchmarkProject(b, benchCtx, 20)
	})

	b.Run("Project50", func(b *testing.B) {
		BenchmarkProject(b, benchCtx, 50)
	})

	b.Run("Project100", func(b *testing.B) {
		BenchmarkProject(b, benchCtx, 100)
	})

	b.Run("Project120", func(b *testing.B) {
		BenchmarkProject(b, benchCtx, 120)
	})

	if benchCtx.HasChannel {
		b.Run("ProjectStream1", func(b *testing.B) {
			BenchmarkProjectStream(b, benchCtx, 1)
		})

		b.Run("ProjectStream2", func(b *testing.B) {
			BenchmarkProjectStream(b, benchCtx, 2)
		})

		b.Run("ProjectStream5", func(b *testing.B) {
			BenchmarkProjectStream(b, benchCtx, 5)
		})

		b.Run("ProjectStream10", func(b *testing.B) {
			BenchmarkProjectStream(b, benchCtx, 10)
		})

		b.Run("ProjectStream20", func(b *testing.B) {
			BenchmarkProjectStream(b, benchCtx, 20)
		})

		b.Run("ProjectStream50", func(b *testing.B) {
			BenchmarkProjectStream(b, benchCtx, 50)
		})

		b.Run("ProjectStream100", func(b *testing.B) {
			BenchmarkProjectStream(b, benchCtx, 100)
		})

		b.Run("ProjectStream120", func(b *testing.B) {
			BenchmarkProjectStream(b, benchCtx, 120)
		})
	}

	b.Run("MemoryProjection", func(b *testing.B) {
		BenchmarkMemoryUsage(b, benchCtx, "project")
	})
}

func BenchmarkProjection_Tiny(b *testing.B) {
	// Use 10 past events for tiny dataset testing
	benchCtx := SetupBenchmarkContext(b, "tiny", 10)

	b.Run("Project1", func(b *testing.B) {
		BenchmarkProject(b, benchCtx, 1)
	})

	b.Run("Project2", func(b *testing.B) {
		BenchmarkProject(b, benchCtx, 2)
	})

	b.Run("Project5", func(b *testing.B) {
		BenchmarkProject(b, benchCtx, 5)
	})

	b.Run("Project10", func(b *testing.B) {
		BenchmarkProject(b, benchCtx, 10)
	})

	b.Run("Project20", func(b *testing.B) {
		BenchmarkProject(b, benchCtx, 20)
	})

	if benchCtx.HasChannel {
		b.Run("ProjectStream1", func(b *testing.B) {
			BenchmarkProjectStream(b, benchCtx, 1)
		})

		b.Run("ProjectStream2", func(b *testing.B) {
			BenchmarkProjectStream(b, benchCtx, 2)
		})

		b.Run("ProjectStream5", func(b *testing.B) {
			BenchmarkProjectStream(b, benchCtx, 5)
		})

		b.Run("ProjectStream10", func(b *testing.B) {
			BenchmarkProjectStream(b, benchCtx, 10)
		})

		b.Run("ProjectStream20", func(b *testing.B) {
			BenchmarkProjectStream(b, benchCtx, 20)
		})
	}

	b.Run("MemoryProjection", func(b *testing.B) {
		BenchmarkMemoryUsage(b, benchCtx, "project")
	})
}
