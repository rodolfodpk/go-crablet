package main

import (
	"fmt"
	"log"
	"time"

	"github.com/rodolfodpk/go-crablet/internal/benchmarks/setup"
)

func main() {
	fmt.Println("=== Pre-generating datasets for benchmarks ===")
	start := time.Now()

	// Initialize the cache
	if err := setup.InitGlobalCache(); err != nil {
		log.Fatalf("Failed to initialize cache: %v", err)
	}

	// Pre-generate all dataset sizes
	datasetSizes := []string{"tiny", "small"}

	for _, size := range datasetSizes {
		fmt.Printf("Generating %s dataset... ", size)
		startGen := time.Now()

		config, exists := setup.DatasetSizes[size]
		if !exists {
			log.Fatalf("Unknown dataset size: %s", size)
		}

		// This will generate and cache the dataset
		dataset, err := setup.GetCachedDataset(config)
		if err != nil {
			log.Fatalf("Failed to generate %s dataset: %v", size, err)
		}

		duration := time.Since(startGen)
		fmt.Printf("done in %v\n", duration)
		fmt.Printf("  - Courses: %d\n", len(dataset.Courses))
		fmt.Printf("  - Students: %d\n", len(dataset.Students))
		fmt.Printf("  - Enrollments: %d\n", len(dataset.Enrollments))
	}

	// Show cache info
	fmt.Println("\n=== Cache Information ===")
	cacheInfo, err := setup.GetGlobalCache().GetCacheInfo()
	if err != nil {
		log.Printf("Failed to get cache info: %v", err)
	} else {
		for id, createdAt := range cacheInfo {
			fmt.Printf("Dataset %s: cached at %v\n", id, createdAt)
		}
	}

	totalDuration := time.Since(start)
	fmt.Printf("\n=== All datasets prepared in %v ===\n", totalDuration)
	fmt.Println("Benchmarks will now run much faster!")
}
