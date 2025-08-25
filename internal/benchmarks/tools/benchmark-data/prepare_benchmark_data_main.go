package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/rodolfodpk/go-crablet/internal/benchmarks/setup"
)

func main() {
	fmt.Println("🔧 Generating benchmark data for fast access...")

	// Get dataset size from command line or use default
	datasetSize := "small"
	if len(os.Args) > 1 {
		datasetSize = os.Args[1]
	}

	// Validate dataset size
	if _, exists := setup.BenchmarkDataSizes[datasetSize]; !exists {
		log.Fatalf("Invalid dataset size: %s. Available: tiny, small", datasetSize)
	}

	fmt.Printf("📊 Generating %s dataset...\n", datasetSize)

	// Get benchmark data configuration
	config := setup.BenchmarkDataSizes[datasetSize]
	fmt.Printf("  - Single events: %d\n", config.SingleEvents)
	fmt.Printf("  - Batch 10 events: %d\n", config.Batch10Events)
	fmt.Printf("  - Batch 100 events: %d\n", config.Batch100Events)
	fmt.Printf("  - Batch 1000 events: %d\n", config.Batch1000Events)
	fmt.Printf("  - AppendIf events: %d\n", config.AppendIfEvents)
	fmt.Printf("  - Mixed events: %d\n", config.MixedEvents)

	// Generate benchmark data
	fmt.Println("🔄 Generating benchmark events...")
	benchmarkData := setup.GenerateBenchmarkData(config)

	// Calculate total events
	totalEvents := 0
	for _, events := range benchmarkData {
		totalEvents += len(events)
	}
	fmt.Printf("✅ Generated %d total events\n", totalEvents)

	// Determine cache file path
	cacheDir := filepath.Join("..", "performance", "cache")
	cacheFile := filepath.Join(cacheDir, "benchmark_data.db")

	// Ensure cache directory exists
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		log.Fatalf("Failed to create cache directory: %v", err)
	}

	// Cache data in SQLite
	fmt.Printf("💾 Caching data in SQLite: %s\n", cacheFile)
	if err := setup.CacheBenchmarkData(cacheFile, benchmarkData); err != nil {
		log.Fatalf("Failed to cache benchmark data: %v", err)
	}

	// Verify cached data
	fmt.Println("🔍 Verifying cached data...")
	cachedData, err := setup.LoadBenchmarkDataFromCache(cacheFile)
	if err != nil {
		log.Fatalf("Failed to load cached data: %v", err)
	}

	// Verify counts
	for category, events := range cachedData {
		expected := len(benchmarkData[category])
		actual := len(events)
		if expected != actual {
			log.Printf("⚠️  Warning: %s category has %d events, expected %d", category, actual, expected)
		} else {
			fmt.Printf("  ✅ %s: %d events\n", category, actual)
		}
	}

	fmt.Println("🎯 Benchmark data generation complete!")
	fmt.Printf("📁 Cache file: %s\n", cacheFile)
	fmt.Printf("📊 Total cached events: %d\n", totalEvents)
	fmt.Println("\n💡 Now benchmarks can use cached data for faster execution!")
}
