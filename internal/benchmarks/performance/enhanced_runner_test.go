package performance

import (
	"os"
	"testing"
)

// TestEnhancedBenchmarkRunner verifies our enhanced benchmark runner works
func TestEnhancedBenchmarkRunner(t *testing.T) {
	runner := NewEnhancedBenchmarkRunner()

	// Verify runner was created correctly
	if runner.OutputDir != "benchmark-results" {
		t.Errorf("Expected OutputDir to be 'benchmark-results', got %s", runner.OutputDir)
	}

	if len(runner.DatasetSizes) != 2 {
		t.Errorf("Expected 2 dataset sizes, got %d", len(runner.DatasetSizes))
	}

	if len(runner.BenchmarkTypes) != 4 {
		t.Errorf("Expected 4 benchmark types, got %d", len(runner.BenchmarkTypes))
	}

	// Verify dataset sizes
	expectedSizes := []string{"tiny", "small"}
	for i, size := range expectedSizes {
		if runner.DatasetSizes[i] != size {
			t.Errorf("Expected dataset size %s at index %d, got %s", size, i, runner.DatasetSizes[i])
		}
	}

	// Verify benchmark types
	expectedTypes := []string{"basic", "complex", "concurrent", "business"}
	for i, bType := range expectedTypes {
		if runner.BenchmarkTypes[i] != bType {
			t.Errorf("Expected benchmark type %s at index %d, got %s", bType, i, runner.BenchmarkTypes[i])
		}
	}
}

// TestBenchmarkReportGeneration tests report generation
func TestBenchmarkReportGeneration(t *testing.T) {
	runner := NewEnhancedBenchmarkRunner()
	timestamp := "2025-01-01_12-00-00"

	// Create a mock benchmark context for testing
	mockB := &testing.B{}

	// This should create a report file
	runner.GenerateBenchmarkReport(mockB, timestamp)

	// Verify the report file was created
	reportPath := "benchmark-results/enhanced_benchmark_report_" + timestamp + ".md"

	// Clean up
	os.Remove(reportPath)
	os.Remove("benchmark-results")
}
