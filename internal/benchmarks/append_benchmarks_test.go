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

// Realistic benchmark suite - runs benchmarks using actual dataset events
func BenchmarkAppend_Small_Realistic(b *testing.B) {
	RunAllBenchmarksRealistic(b, "small")
}

func BenchmarkAppend_Tiny_Realistic(b *testing.B) {
	RunAllBenchmarksRealistic(b, "tiny")
}

func BenchmarkAppend_Medium_Realistic(b *testing.B) {
	RunAllBenchmarksRealistic(b, "medium")
}
