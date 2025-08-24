package performance

import "testing"

func BenchmarkRead_Small(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "small")

	b.Run("ReadSimple", func(b *testing.B) {
		BenchmarkRead(b, benchCtx, 0)
	})

	b.Run("ReadComplex", func(b *testing.B) {
		BenchmarkRead(b, benchCtx, 1)
	})

	if benchCtx.HasChannel {
		b.Run("ReadStreamChannel", func(b *testing.B) {
			BenchmarkReadChannel(b, benchCtx, 0)
		})
	}
}

func BenchmarkRead_Tiny(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "tiny")

	b.Run("ReadSimple", func(b *testing.B) {
		BenchmarkRead(b, benchCtx, 0)
	})

	b.Run("ReadComplex", func(b *testing.B) {
		BenchmarkRead(b, benchCtx, 1)
	})

	if benchCtx.HasChannel {
		b.Run("ReadStreamChannel", func(b *testing.B) {
			BenchmarkReadChannel(b, benchCtx, 0)
		})
	}
}

// Individual read benchmarks for detailed analysis
func BenchmarkReadSimple_Small(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "small")
	BenchmarkRead(b, benchCtx, 0)
}

func BenchmarkReadSimple_Tiny(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "tiny")
	BenchmarkRead(b, benchCtx, 0)
}

func BenchmarkReadComplex_Small(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "small")
	BenchmarkRead(b, benchCtx, 1)
}

func BenchmarkReadComplex_Tiny(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "tiny")
	BenchmarkRead(b, benchCtx, 1)
}

func BenchmarkReadChannel_Small(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "small")
	BenchmarkReadChannel(b, benchCtx, 0)
}

func BenchmarkReadChannel_Tiny(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "tiny")
	BenchmarkReadChannel(b, benchCtx, 0)
}

// Memory usage benchmarks for read operations
func BenchmarkMemoryRead_Small(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "small")
	BenchmarkMemoryUsage(b, benchCtx, "read")
}

func BenchmarkMemoryRead_Tiny(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "tiny")
	BenchmarkMemoryUsage(b, benchCtx, "read")
}

func BenchmarkMemoryStream_Small(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "small")
	BenchmarkMemoryUsage(b, benchCtx, "stream")
}

func BenchmarkMemoryStream_Tiny(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "tiny")
	BenchmarkMemoryUsage(b, benchCtx, "stream")
}
