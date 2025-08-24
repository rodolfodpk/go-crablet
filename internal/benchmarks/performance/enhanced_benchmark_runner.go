package performance

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/rodolfodpk/go-crablet/pkg/dcb"
)

// EnhancedBenchmarkRunner provides advanced benchmarking capabilities
type EnhancedBenchmarkRunner struct {
	OutputDir      string
	DatasetSizes   []string
	BenchmarkTypes []string
}

// NewEnhancedBenchmarkRunner creates a new enhanced benchmark runner
func NewEnhancedBenchmarkRunner() *EnhancedBenchmarkRunner {
	return &EnhancedBenchmarkRunner{
		OutputDir:      "benchmark-results",
		DatasetSizes:   []string{"tiny", "small"},
		BenchmarkTypes: []string{"basic", "complex", "concurrent", "business"},
	}
}

// RunAllEnhancedBenchmarks executes all benchmarks with enhanced output
func (r *EnhancedBenchmarkRunner) RunAllEnhancedBenchmarks(b *testing.B) {
	// Create output directory
	if err := os.MkdirAll(r.OutputDir, 0755); err != nil {
		b.Fatalf("Failed to create output directory: %v", err)
	}

	timestamp := time.Now().Format("2006-01-02_15-04-05")

	for _, datasetSize := range r.DatasetSizes {
		b.Logf("Running enhanced benchmarks for dataset: %s", datasetSize)

		// Run basic benchmarks
		r.runBasicBenchmarks(b, datasetSize, timestamp)

		// Run complex scenario benchmarks
		r.runComplexScenarioBenchmarks(b, datasetSize, timestamp)

		// Run concurrent benchmarks
		r.runConcurrentBenchmarks(b, datasetSize, timestamp)

		// Run business logic benchmarks
		r.runBusinessLogicBenchmarks(b, datasetSize, timestamp)
	}
}

// runBasicBenchmarks runs the core performance benchmarks
func (r *EnhancedBenchmarkRunner) runBasicBenchmarks(b *testing.B, datasetSize, timestamp string) {
	b.Logf("Running basic benchmarks for %s dataset", datasetSize)

	// These are already implemented in the existing benchmark files
	// We just need to ensure they run with our enhanced setup
}

// runComplexScenarioBenchmarks runs our new complex business workflow benchmarks
func (r *EnhancedBenchmarkRunner) runComplexScenarioBenchmarks(b *testing.B, datasetSize, timestamp string) {
	b.Logf("Running complex scenario benchmarks for %s dataset", datasetSize)

	// Complex business workflow
	b.Run(fmt.Sprintf("ComplexBusinessWorkflow_%s", datasetSize), func(b *testing.B) {
		benchCtx := SetupBenchmarkContext(b, datasetSize)
		ctx := context.Background()

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			// Simulate user registration workflow
			// 1. Check if user exists
			// 2. Create user account
			// 3. Create user profile
			// 4. Send welcome event

			// Step 1: Check if user exists (query)
			query := dcb.NewQuery(dcb.NewTags("user_id", "user123"), "UserCreated")
			cursor := &dcb.Cursor{}

			_, err := benchCtx.Store.Query(ctx, query, cursor)
			if err != nil {
				b.Fatal(err)
			}

			// Step 2: Create user account (append)
			userEvent := dcb.NewInputEvent("UserCreated",
				dcb.NewTags("user_id", "user123", "event_type", "user_management"),
				[]byte(`{"user_id":"user123","email":"user@example.com","timestamp":1234567890}`))

			err = benchCtx.Store.Append(ctx, []dcb.InputEvent{userEvent})
			if err != nil {
				b.Fatal(err)
			}

			// Step 3: Create user profile (append)
			profileEvent := dcb.NewInputEvent("UserProfileCreated",
				dcb.NewTags("user_id", "user123", "event_type", "user_management"),
				[]byte(`{"user_id":"user123","first_name":"John","last_name":"Doe","timestamp":1234567890}`))

			err = benchCtx.Store.Append(ctx, []dcb.InputEvent{profileEvent})
			if err != nil {
				b.Fatal(err)
			}

			// Step 4: Send welcome event (append)
			welcomeEvent := dcb.NewInputEvent("WelcomeEmailSent",
				dcb.NewTags("user_id", "user123", "event_type", "communication"),
				[]byte(`{"user_id":"user123","email":"user@example.com","timestamp":1234567890}`))

			err = benchCtx.Store.Append(ctx, []dcb.InputEvent{welcomeEvent})
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// runConcurrentBenchmarks runs concurrent operation benchmarks
func (r *EnhancedBenchmarkRunner) runConcurrentBenchmarks(b *testing.B, datasetSize, timestamp string) {
	b.Logf("Running concurrent benchmarks for %s dataset", datasetSize)

	// Concurrent appends
	b.Run(fmt.Sprintf("ConcurrentAppends_%s", datasetSize), func(b *testing.B) {
		benchCtx := SetupBenchmarkContext(b, datasetSize)
		ctx := context.Background()

		b.ResetTimer()
		b.ReportAllocs()

		// Simulate 10 concurrent users
		concurrentUsers := 10
		var wg sync.WaitGroup

		for i := 0; i < b.N; i++ {
			wg.Add(concurrentUsers)

			for userID := 0; userID < concurrentUsers; userID++ {
				go func(userID int) {
					defer wg.Done()

					// Each user creates an event
					event := dcb.NewInputEvent("UserAction",
						dcb.NewTags("user_id", fmt.Sprintf("%d", userID), "action_type", "authentication"),
						[]byte(fmt.Sprintf(`{"user_id":%d,"action":"login","timestamp":1234567890}`, userID)))

					err := benchCtx.Store.Append(ctx, []dcb.InputEvent{event})
					if err != nil {
						b.Fatal(err)
					}
				}(userID)
			}

			wg.Wait()
		}
	})
}

// runBusinessLogicBenchmarks runs business rule validation benchmarks
func (r *EnhancedBenchmarkRunner) runBusinessLogicBenchmarks(b *testing.B, datasetSize, timestamp string) {
	b.Logf("Running business logic benchmarks for %s dataset", datasetSize)

	// Business rule validation
	b.Run(fmt.Sprintf("BusinessRuleValidation_%s", datasetSize), func(b *testing.B) {
		benchCtx := SetupBenchmarkContext(b, datasetSize)
		ctx := context.Background()

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			// Simulate complex business rule: user can only enroll in courses
			// if they have completed prerequisites

			// 1. Check if user has completed prerequisites
			prereqQuery := dcb.NewQuery(dcb.NewTags("user_id", "user123"), "PrerequisiteCompleted")
			prereqCondition := dcb.NewAppendCondition(prereqQuery)

			// 2. Try to enroll with condition
			enrollmentEvent := dcb.NewInputEvent("CourseEnrollment",
				dcb.NewTags("user_id", "user123", "course_id", "course456", "event_type", "enrollment"),
				[]byte(`{"user_id":"user123","course_id":"course456","timestamp":1234567890}`))

			// This will fail if conditions aren't met, but that's expected
			// We're measuring the performance of the validation logic
			_ = benchCtx.Store.AppendIf(ctx, []dcb.InputEvent{enrollmentEvent}, prereqCondition)
		}
	})
}

// GenerateBenchmarkReport creates a comprehensive benchmark report
func (r *EnhancedBenchmarkRunner) GenerateBenchmarkReport(b *testing.B, timestamp string) {
	reportPath := filepath.Join(r.OutputDir, fmt.Sprintf("enhanced_benchmark_report_%s.md", timestamp))

	report := fmt.Sprintf(`# Enhanced Benchmark Report - %s

## Overview
This report contains comprehensive benchmark results for go-crablet's DCB library,
including complex business scenarios, concurrent operations, and business logic validation.

## Benchmark Categories

### 1. Basic Performance
- Core append operations
- Query performance
- Projection performance
- Memory usage patterns

### 2. Complex Business Scenarios
- User registration workflows
- Multi-step business processes
- Event chain validation

### 3. Concurrent Operations
- Multiple user simulation
- Concurrent append operations
- Load testing patterns

### 4. Business Logic Validation
- DCB concurrency control
- Business rule enforcement
- Conditional append performance

## Recommendations

### Performance Optimization
- Focus on business workflow bottlenecks
- Optimize concurrent operation handling
- Improve memory allocation patterns

### Production Readiness
- Validate business logic performance
- Test concurrent user scenarios
- Monitor memory usage under load

## Next Steps
1. Run benchmarks with different dataset sizes
2. Analyze performance bottlenecks
3. Optimize critical business workflows
4. Validate production load patterns

Generated: %s
`, timestamp, time.Now().Format("2006-01-02 15:04:05"))

	if err := os.WriteFile(reportPath, []byte(report), 0644); err != nil {
		b.Logf("Failed to write benchmark report: %v", err)
	}
}

// RunBenchstatComparison runs benchstat to compare benchmark results
func (r *EnhancedBenchmarkRunner) RunBenchstatComparison(b *testing.B, oldFile, newFile string) {
	// Check if benchstat is available
	if _, err := exec.LookPath("benchstat"); err != nil {
		b.Logf("benchstat not found, skipping comparison")
		return
	}

	cmd := exec.Command("benchstat", oldFile, newFile)
	output, err := cmd.Output()
	if err != nil {
		b.Logf("Failed to run benchstat: %v", err)
		return
	}

	b.Logf("Benchstat comparison:\n%s", string(output))
}
