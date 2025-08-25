package benchmarks

import "testing"

// Core read benchmark suite
func BenchmarkRead_Small(b *testing.B) {
	// Use 100 past events for realistic AppendIf testing
	benchCtx := SetupBenchmarkContext(b, "small", 100)

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
	// Use 10 past events for tiny dataset testing
	benchCtx := SetupBenchmarkContext(b, "tiny", 10)

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
