package benchmarks

import "testing"

// Core benchmark suite - runs all realistic benchmarks
func BenchmarkAppend_Small(b *testing.B) {
	RunAllBenchmarks(b, "small")
}

func BenchmarkAppend_Tiny(b *testing.B) {
	RunAllBenchmarks(b, "tiny")
}

func BenchmarkAppend_Medium(b *testing.B) {
	RunAllBenchmarks(b, "medium")
}
