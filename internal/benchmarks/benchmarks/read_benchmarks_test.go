package benchmarks

import "testing"

func BenchmarkRead_Small(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "small")

	b.Run("ReadSimple", func(b *testing.B) {
		BenchmarkRead(b, benchCtx, 0)
	})

	b.Run("ReadComplex", func(b *testing.B) {
		BenchmarkRead(b, benchCtx, 1)
	})

	b.Run("ReadStream", func(b *testing.B) {
		BenchmarkReadStream(b, benchCtx, 0)
	})

	if benchCtx.HasChannel {
		b.Run("ReadStreamChannel", func(b *testing.B) {
			BenchmarkReadStreamChannel(b, benchCtx, 0)
		})
	}
}

func BenchmarkRead_Medium(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "medium")

	b.Run("ReadSimple", func(b *testing.B) {
		BenchmarkRead(b, benchCtx, 0)
	})

	b.Run("ReadComplex", func(b *testing.B) {
		BenchmarkRead(b, benchCtx, 1)
	})

	b.Run("ReadStream", func(b *testing.B) {
		BenchmarkReadStream(b, benchCtx, 0)
	})

	if benchCtx.HasChannel {
		b.Run("ReadStreamChannel", func(b *testing.B) {
			BenchmarkReadStreamChannel(b, benchCtx, 0)
		})
	}
}

func BenchmarkRead_Large(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "large")

	b.Run("ReadSimple", func(b *testing.B) {
		BenchmarkRead(b, benchCtx, 0)
	})

	b.Run("ReadComplex", func(b *testing.B) {
		BenchmarkRead(b, benchCtx, 1)
	})

	b.Run("ReadStream", func(b *testing.B) {
		BenchmarkReadStream(b, benchCtx, 0)
	})

	if benchCtx.HasChannel {
		b.Run("ReadStreamChannel", func(b *testing.B) {
			BenchmarkReadStreamChannel(b, benchCtx, 0)
		})
	}
}

// Individual read benchmarks for detailed analysis
func BenchmarkReadSimple_Small(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "small")
	BenchmarkRead(b, benchCtx, 0)
}

func BenchmarkReadSimple_Medium(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "medium")
	BenchmarkRead(b, benchCtx, 0)
}

func BenchmarkReadSimple_Large(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "large")
	BenchmarkRead(b, benchCtx, 0)
}

func BenchmarkReadComplex_Small(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "small")
	BenchmarkRead(b, benchCtx, 1)
}

func BenchmarkReadComplex_Medium(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "medium")
	BenchmarkRead(b, benchCtx, 1)
}

func BenchmarkReadComplex_Large(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "large")
	BenchmarkRead(b, benchCtx, 1)
}

func BenchmarkReadStream_Small(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "small")
	BenchmarkReadStream(b, benchCtx, 0)
}

func BenchmarkReadStream_Medium(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "medium")
	BenchmarkReadStream(b, benchCtx, 0)
}

func BenchmarkReadStream_Large(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "large")
	BenchmarkReadStream(b, benchCtx, 0)
}

func BenchmarkReadStreamChannel_Small(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "small")
	BenchmarkReadStreamChannel(b, benchCtx, 0)
}

func BenchmarkReadStreamChannel_Medium(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "medium")
	BenchmarkReadStreamChannel(b, benchCtx, 0)
}

func BenchmarkReadStreamChannel_Large(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "large")
	BenchmarkReadStreamChannel(b, benchCtx, 0)
}

// Memory usage benchmarks for read operations
func BenchmarkMemoryRead_Small(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "small")
	BenchmarkMemoryUsage(b, benchCtx, "read")
}

func BenchmarkMemoryRead_Medium(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "medium")
	BenchmarkMemoryUsage(b, benchCtx, "read")
}

func BenchmarkMemoryRead_Large(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "large")
	BenchmarkMemoryUsage(b, benchCtx, "read")
}

func BenchmarkMemoryStream_Small(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "small")
	BenchmarkMemoryUsage(b, benchCtx, "stream")
}

func BenchmarkMemoryStream_Medium(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "medium")
	BenchmarkMemoryUsage(b, benchCtx, "stream")
}

func BenchmarkMemoryStream_Large(b *testing.B) {
	benchCtx := SetupBenchmarkContext(b, "large")
	BenchmarkMemoryUsage(b, benchCtx, "stream")
}
