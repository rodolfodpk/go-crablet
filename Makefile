.PHONY: all build test clean docker-up docker-down lint help benchmark benchmark-all benchmark-quick benchmark-append benchmark-isolation benchmark-concurrency benchmark-results deps fmt examples generate-datasets web-app-start web-app-stop

# Variables
BINARY_NAME=crablet
GO=go
DOCKER_COMPOSE=docker-compose
BENCHMARK_RESULTS_DIR=benchmark-results
TIMESTAMP=$(shell date +%Y%m%d_%H%M%S)

# Default target
all: build

# Build all packages (library approach)
build:
	$(GO) build ./...

# Run tests
test:
	$(GO) test -v ./pkg/...

# Run tests with coverage (internal tests only)
test-coverage:
	$(GO) test -v -coverprofile=coverage.out ./pkg/...
	$(GO) tool cover -html=coverage.out

# Run comprehensive coverage (internal + external tests)
coverage:
	./scripts/generate-coverage.sh

# Run comprehensive coverage and update badge
coverage-badge:
	./scripts/generate-coverage.sh update-badge

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f coverage.out coverage_combined.out coverage_internal.out coverage_external.out coverage.html
	rm -rf $(BENCHMARK_RESULTS_DIR)

# Start Docker containers
docker-up:
	$(DOCKER_COMPOSE) up -d

# Stop Docker containers
docker-down:
	$(DOCKER_COMPOSE) down

# Run linter
lint:
	golangci-lint run

# Format code
fmt:
	$(GO) fmt ./...

# Download dependencies
deps:
	$(GO) mod download
	$(GO) mod tidy

# Generate SQLite test datasets
generate-datasets:
	@echo "🔧 Generating SQLite test datasets..."
	@cd internal/benchmarks/tools && $(GO) run prepare_datasets_main.go
	@echo "✅ Test datasets generated in cache/"

# Generate benchmark data for fast access
generate-benchmark-data:
	@echo "🔧 Generating benchmark data for fast access..."
	@cd internal/benchmarks/tools/benchmark-data && $(GO) run prepare_benchmark_data_main.go
	@echo "✅ Benchmark data generated and cached in SQLite"

# Generate all data (datasets + benchmark data)
generate-all-data: generate-datasets generate-benchmark-data
	@echo "🎯 All data generated and cached!"

# Start web app server
web-app-start:
	@echo "🚀 Starting web app server..."
	@cd internal/web-app && ./web-app &
	@echo "✅ Web app started on http://localhost:8080"

# Stop web app server
web-app-stop:
	@echo "🛑 Stopping web app server..."
	@pkill -f "web-app" || true
	@echo "✅ Web app stopped"

# Generate documentation
docs:
	godoc -http=:6060

# Benchmark targets
benchmark: benchmark-all

benchmark-all: benchmark-quick benchmark-append benchmark-isolation benchmark-concurrency benchmark-go benchmark-web-app benchmark-web-app-appendif benchmark-results

benchmark-quick:
	@echo "🚀 Running Quick Test Benchmark..."
	@mkdir -p $(BENCHMARK_RESULTS_DIR)
	@cd internal/web-app && k6 run k6/quick/quick.js > ../../$(BENCHMARK_RESULTS_DIR)/quick_test_$(TIMESTAMP).txt 2>&1 || true
	@echo "✅ Quick test completed"

benchmark-append:
	@echo "🚀 Running Append Performance Benchmark..."
	@cd internal/web-app && k6 run k6/benchmarks/append-benchmark.js > ../../$(BENCHMARK_RESULTS_DIR)/append_benchmark_$(TIMESTAMP).txt 2>&1 || true
	@echo "✅ Append benchmark completed"

benchmark-isolation:
	@echo "🚀 Running Isolation Level Benchmark..."
	@cd internal/web-app && k6 run k6/benchmarks/isolation-level-benchmark.js > ../../$(BENCHMARK_RESULTS_DIR)/isolation_benchmark_$(TIMESTAMP).txt 2>&1 || true
	@echo "✅ Isolation benchmark completed"

benchmark-concurrency:
	@echo "🚀 Running Concurrency Test..."
	@cd internal/web-app && k6 run k6/tests/k6-concurrency-test.js > ../../$(BENCHMARK_RESULTS_DIR)/concurrency_test_$(TIMESTAMP).txt 2>&1 || true
	@echo "✅ Concurrency test completed"

benchmark-go:
	@echo "🚀 Running Go Library Benchmarks..."
	@mkdir -p $(BENCHMARK_RESULTS_DIR)
	@cd internal/benchmarks && $(GO) test -bench="BenchmarkAppend_.*_Realistic" -benchmem -benchtime=2s -timeout=10m . > ../../$(BENCHMARK_RESULTS_DIR)/go_benchmarks_$(TIMESTAMP).txt 2>&1 || true
	@echo "✅ Go benchmarks completed"

benchmark-go-quick:
	@echo "🚀 Running Quick Go Benchmarks..."
	@mkdir -p $(BENCHMARK_RESULTS_DIR)
	@cd internal/benchmarks && $(GO) test -bench=BenchmarkQuick -benchmem -benchtime=1s -timeout=2m . > ../../$(BENCHMARK_RESULTS_DIR)/go_quick_benchmarks_$(TIMESTAMP).txt 2>&1 || true
	@echo "✅ Quick Go benchmarks completed"



# Core Operations (Single-threaded, Tiny dataset)
benchmark-go-append:
	@echo "🚀 Running Append Operation Benchmarks (Tiny dataset, single-threaded)..."
	@mkdir -p $(BENCHMARK_RESULTS_DIR)
	@cd internal/benchmarks && $(GO) test -bench="BenchmarkAppend_Tiny_Realistic" -benchmem -benchtime=1s -timeout=3m . > ../../$(BENCHMARK_RESULTS_DIR)/go_append_benchmarks_$(TIMESTAMP).txt 2>&1 || true
	@echo "✅ Append benchmarks completed"

benchmark-go-appendif:
	@echo "🚀 Running AppendIf Operation Benchmarks (Tiny dataset, single-threaded)..."
	@mkdir -p $(BENCHMARK_RESULTS_DIR)
	@cd internal/benchmarks && $(GO) test -bench="BenchmarkAppend_Tiny_Realistic.*AppendIf" -benchmem -benchtime=1s -timeout=3m . > ../../$(BENCHMARK_RESULTS_DIR)/go_appendif_benchmarks_$(TIMESTAMP).txt 2>&1 || true
	@echo "✅ AppendIf benchmarks completed"

benchmark-go-read:
	@echo "🚀 Running Read Operation Benchmarks (Tiny dataset, single-threaded)..."
	@mkdir -p $(BENCHMARK_RESULTS_DIR)
	@cd internal/benchmarks && $(GO) test -bench="BenchmarkRead_Tiny" -benchmem -benchtime=1s -timeout=3m . > ../../$(BENCHMARK_RESULTS_DIR)/go_read_benchmarks_$(TIMESTAMP).txt 2>&1 || true
	@echo "✅ Read benchmarks completed"

benchmark-go-projection:
	@echo "🚀 Running Projection Benchmarks (Tiny dataset, single-threaded)..."
	@mkdir -p $(BENCHMARK_RESULTS_DIR)
	@cd internal/benchmarks && $(GO) test -bench="BenchmarkProjection_Tiny" -benchmem -benchtime=1s -timeout=3m . > ../../$(BENCHMARK_RESULTS_DIR)/go_projection_benchmarks_$(TIMESTAMP).txt 2>&1 || true
	@echo "✅ Projection benchmarks completed"

benchmark-go-batch:
	@echo "🚀 Running Batch Operation Benchmarks (Tiny dataset, single-threaded)..."
	@mkdir -p $(BENCHMARK_RESULTS_DIR)
	@cd internal/benchmarks && $(GO) test -bench="BenchmarkAppendBatch" -benchmem -benchtime=1s -timeout=2m . > ../../$(BENCHMARK_RESULTS_DIR)/go_batch_benchmarks_$(TIMESTAMP).txt 2>&1 || true
	@echo "✅ Batch operation benchmarks completed"

# Concurrency + Operations (Concurrent, different datasets)
benchmark-go-append-concurrent:
	@echo "🚀 Running Append Operation Benchmarks with Concurrency (Small dataset)..."
	@mkdir -p $(BENCHMARK_RESULTS_DIR)
	@cd internal/benchmarks && $(GO) test -bench="BenchmarkConcurrentAppends" -benchmem -benchtime=1s -timeout=3m . > ../../$(BENCHMARK_RESULTS_DIR)/go_append_concurrent_benchmarks_$(TIMESTAMP).txt 2>&1 || true
	@echo "✅ Append concurrency benchmarks completed"

# Concurrency levels for read operations
benchmark-go-read-concurrent-1:
	@echo "🚀 Running Read Operation Benchmarks with 1 User (Small dataset, fast)..."
	@mkdir -p $(BENCHMARK_RESULTS_DIR)
	@cd internal/benchmarks && $(GO) test -bench="BenchmarkConcurrentRead_1User" -benchmem -benchtime=1s -timeout=2m . > ../../$(BENCHMARK_RESULTS_DIR)/go_read_concurrent_1user_$(TIMESTAMP).txt 2>&1 || true
	@echo "✅ Read 1-user concurrency benchmarks completed"

benchmark-go-read-concurrent-10:
	@echo "🚀 Running Read Operation Benchmarks with 10 Users (Small dataset, medium)..."
	@mkdir -p $(BENCHMARK_RESULTS_DIR)
	@cd internal/benchmarks && $(GO) test -bench="BenchmarkConcurrentRead_10Users" -benchmem -benchtime=1s -timeout=3m . > ../../$(BENCHMARK_RESULTS_DIR)/go_read_concurrent_10users_$(TIMESTAMP).txt 2>&1 || true
	@echo "✅ Read 10-user concurrency benchmarks completed"

benchmark-go-read-concurrent-100:
	@echo "🚀 Running Read Operation Benchmarks with 100 Users (Medium dataset, slow)..."
	@mkdir -p $(BENCHMARK_RESULTS_DIR)
	@cd internal/benchmarks && $(GO) test -bench="BenchmarkConcurrentRead_100Users" -benchmem -benchtime=1s -timeout=5m . > ../../$(BENCHMARK_RESULTS_DIR)/go_read_concurrent_100users_$(TIMESTAMP).txt 2>&1 || true
	@echo "✅ Read 100-user concurrency benchmarks completed"

# Concurrency levels for projection operations
benchmark-go-projection-concurrent-1:
	@echo "🚀 Running Projection Benchmarks with 1 Goroutine (Small dataset, fast)..."
	@mkdir -p $(BENCHMARK_RESULTS_DIR)
	@cd internal/benchmarks && $(GO) test -bench="BenchmarkConcurrentProjection_1Goroutine" -benchmem -benchtime=1s -timeout=2m . > ../../$(BENCHMARK_RESULTS_DIR)/go_projection_concurrent_1goroutine_$(TIMESTAMP).txt 2>&1 || true
	@echo "✅ Projection 1-goroutine concurrency benchmarks completed"

benchmark-go-projection-concurrent-10:
	@echo "🚀 Running Projection Benchmarks with 10 Goroutines (Small dataset, medium)..."
	@mkdir -p $(BENCHMARK_RESULTS_DIR)
	@cd internal/benchmarks && $(GO) test -bench="BenchmarkConcurrentProjection_10Goroutines" -benchmem -benchtime=1s -timeout=3m . > ../../$(BENCHMARK_RESULTS_DIR)/go_projection_concurrent_10goroutines_$(TIMESTAMP).txt 2>&1 || true
	@echo "✅ Projection 10-goroutine concurrency benchmarks completed"

benchmark-go-projection-concurrent-100:
	@echo "🚀 Running Projection Benchmarks with 100 Goroutines (Small dataset, slow)..."
	@mkdir -p $(BENCHMARK_RESULTS_DIR)
	@cd internal/benchmarks && $(GO) test -bench="BenchmarkConcurrentProjection_100Goroutines" -benchmem -benchtime=1s -timeout=5m . > ../../$(BENCHMARK_RESULTS_DIR)/go_projection_concurrent_100goroutines_$(TIMESTAMP).txt 2>&1 || true
	@echo "✅ Projection 100-goroutine concurrency benchmarks completed"

# Dataset-specific targets (all operations with specific dataset size)
benchmark-go-tiny:
	@echo "🚀 Running All Operations with Tiny Dataset (5 courses, 10 students, 20 enrollments)..."
	@mkdir -p $(BENCHMARK_RESULTS_DIR)
	@cd internal/benchmarks && $(GO) test -bench="BenchmarkAppend_Tiny_Realistic" -benchmem -benchtime=1s -timeout=3m . > ../../$(BENCHMARK_RESULTS_DIR)/go_tiny_benchmarks_$(TIMESTAMP).txt 2>&1 || true
	@echo "✅ Tiny dataset benchmarks completed"

benchmark-go-small:
	@echo "🚀 Running All Operations with Small Dataset (500 courses, 5K students, 25K enrollments)..."
	@mkdir -p $(BENCHMARK_RESULTS_DIR)
	@cd internal/benchmarks && $(GO) test -bench="BenchmarkAppend_Small_Realistic" -benchmem -benchtime=2s -timeout=5m . > ../../$(BENCHMARK_RESULTS_DIR)/go_small_benchmarks_$(TIMESTAMP).txt 2>&1 || true
	@echo "✅ Small dataset benchmarks completed"

benchmark-go-medium:
	@echo "🚀 Running All Operations with Medium Dataset (1K courses, 10K students, 50K enrollments)..."
	@mkdir -p $(BENCHMARK_RESULTS_DIR)
	@cd internal/benchmarks && $(GO) test -bench="BenchmarkAppend_Medium_Realistic" -benchmem -benchtime=5s -timeout=10m . > ../../$(BENCHMARK_RESULTS_DIR)/go_medium_benchmarks_$(TIMESTAMP).txt 2>&1 || true
	@echo "✅ Medium dataset benchmarks completed"



# Business Scenarios (Complex workflows)
benchmark-go-business:
	@echo "🚀 Running Business Logic Benchmarks..."
	@mkdir -p $(BENCHMARK_RESULTS_DIR)
	@cd internal/benchmarks && $(GO) test -bench="BenchmarkComplex|BenchmarkBusiness|BenchmarkMixed|BenchmarkRequest|BenchmarkSustained" -benchmem -benchtime=1s -timeout=5m . > ../../$(BENCHMARK_RESULTS_DIR)/go_business_benchmarks_$(TIMESTAMP).txt 2>&1 || true
	@echo "✅ Business logic benchmarks completed"

benchmark-go-stress:
	@echo "🚀 Running Stress and Load Benchmarks..."
	@mkdir -p $(BENCHMARK_RESULTS_DIR)
	@cd internal/benchmarks && $(GO) test -bench="BenchmarkRequestBurst|BenchmarkSustainedLoad" -benchmem -benchtime=1s -timeout=5m . > ../../$(BENCHMARK_RESULTS_DIR)/go_stress_benchmarks_$(TIMESTAMP).txt 2>&1 || true
	@echo "✅ Stress benchmarks completed"

benchmark-go-enhanced:
	@echo "🚀 Running Enhanced Go Benchmarks with Complex Business Scenarios..."
	@mkdir -p $(BENCHMARK_RESULTS_DIR)
	@cd internal/benchmarks && $(GO) test -bench=BenchmarkComplex -benchmem -benchtime=5s -count=3 -timeout=10m . > ../../$(BENCHMARK_RESULTS_DIR)/go_enhanced_benchmarks_$(TIMESTAMP).txt 2>&1 || true
	@echo "✅ Enhanced Go benchmarks completed"

benchmark-go-all:
	@echo "🚀 Running All Go Benchmarks (Basic + Enhanced)..."
	@mkdir -p $(BENCHMARK_RESULTS_DIR)
	@cd internal/benchmarks && $(GO) test -bench=. -benchmem -benchtime=2s -count=3 -timeout=10m . > ../../$(BENCHMARK_RESULTS_DIR)/go_all_benchmarks_$(TIMESTAMP).txt 2>&1 || true
	@echo "✅ All Go benchmarks completed"

benchmark-web-app:
	@echo "🚀 Running Web-App Benchmarks with SQLite Test Data..."
	@mkdir -p $(BENCHMARK_RESULTS_DIR)
	@cd internal/web-app/k6/benchmarks && k6 run --out json=../../../../$(BENCHMARK_RESULTS_DIR)/web_app_benchmarks_$(TIMESTAMP).json append-benchmark.js > ../../../../$(BENCHMARK_RESULTS_DIR)/web_app_benchmarks_$(TIMESTAMP).txt 2>&1 || true
	@echo "✅ Web-app benchmarks completed"

benchmark-web-app-appendif:
	@echo "🚀 Running Web-App AppendIf Benchmarks with SQLite Test Data..."
	@mkdir -p $(BENCHMARK_RESULTS_DIR)
	@cd internal/web-app/k6/benchmarks && k6 run --out json=../../../../$(BENCHMARK_RESULTS_DIR)/web_app_appendif_$(TIMESTAMP).json append-if-benchmark.js > ../../../../$(BENCHMARK_RESULTS_DIR)/web_app_appendif_$(TIMESTAMP).txt 2>&1 || true
	@echo "✅ Web-app appendIf benchmarks completed"



benchmark-results:
	@echo "📊 Collecting benchmark results..."
	@echo "Results saved in: $(BENCHMARK_RESULTS_DIR)/"
	@ls -la $(BENCHMARK_RESULTS_DIR)/*_$(TIMESTAMP).txt 2>/dev/null || echo "No results files found"

benchmark-summary:
	@echo "📊 Benchmark Summary - Three-Dimensional Organization..."
	@echo ""
	@echo "🔧 Core Operations (Single-threaded, Tiny dataset - 5 courses, 10 students, 20 enrollments):"
	@echo "  make benchmark-go-append     - Basic append operations (2-3 minutes)"
	@echo "  make benchmark-go-appendif   - AppendIf operations (2-3 minutes)"
	@echo "  make benchmark-go-read       - Read operations (2-3 minutes)"
	@echo "  make benchmark-go-projection - Projection operations (2-3 minutes)"
	@echo "  make benchmark-go-batch      - Batch operations (1-2 minutes)"
	@echo ""
	@echo "⚡ Concurrency + Operations (Concurrent, different datasets):"
	@echo "  make benchmark-go-append-concurrent     - Append with concurrency (3-5 minutes)"
	@echo "  make benchmark-go-read-concurrent-1     - Read with 1 user (2-3 minutes, small dataset)"
	@echo "  make benchmark-go-read-concurrent-10    - Read with 10 users (3-5 minutes, small dataset)"
	@echo "  make benchmark-go-read-concurrent-100   - Read with 100 users (5-10 minutes, medium dataset)"
	@echo "  make benchmark-go-projection-concurrent-1  - Projection with 1 goroutine (2-3 minutes, small dataset)"
	@echo "  make benchmark-go-projection-concurrent-10 - Projection with 10 goroutines (3-5 minutes, small dataset)"
	@echo "  make benchmark-go-projection-concurrent-100- Projection with 100 goroutines (5-10 minutes, small dataset)"
	@echo ""
	@echo "📊 Dataset Scaling (Single-threaded, different sizes):"
	@echo "  make benchmark-go-tiny       - All operations, tiny dataset (1-2 minutes)"
	@echo "  make benchmark-go-small      - All operations, small dataset (15-20 minutes)"
	@echo "  make benchmark-go-medium     - All operations, medium dataset (30-60 minutes)"
	@echo ""
	@echo "🏢 Business Scenarios:"
	@echo "  make benchmark-go-business   - Complex business workflows (3-5 minutes)"
	@echo "  make benchmark-go-stress     - Stress and load tests (3-5 minutes)"
	@echo ""
	@echo "🚀 Quick Tests:"
	@echo "  make benchmark-go-quick      - Quick tests (4-5 seconds)"
	@echo ""
	@echo "🎯 Full Suite:"
	@echo "  make benchmark-go            - Complete benchmark suite (very long - hours)"

# Run examples
examples:
	@echo "Available examples:"
	@echo "  make example-decision-model  - Run decision model example"
	@echo "  make example-enrollment      - Run course enrollment example"
	@echo "  make example-transfer        - Run money transfer example"
	@echo "  make example-streaming       - Run event streaming example"
	@echo "  make example-batch           - Run batch events example"
	@echo "  make example-concurrency     - Run concurrency comparison example"
	@echo "  make example-utils           - Run utility functions example"

example-decision-model:
	@echo "🚀 Running decision model example..."
	@$(GO) run internal/examples/decision_model/main.go

example-enrollment:
	@echo "🚀 Running course enrollment example..."
	@$(GO) run internal/examples/enrollment/main.go

example-transfer:
	@echo "🚀 Running money transfer example..."
	@$(GO) run internal/examples/transfer/main.go

example-streaming:
	@echo "🚀 Running event streaming example..."
	@$(GO) run internal/examples/streaming/main.go

example-batch:
	@echo "🚀 Running batch events example..."
	@$(GO) run internal/examples/batch/main.go

example-concurrency:
	@echo "🚀 Running ticket booking example..."
	@$(GO) run internal/examples/ticket_booking/main.go

example-utils:
	@echo "🚀 Running utility functions example..."
	@$(GO) run internal/examples/utils/main.go

# Show help
help:
	@echo "Available targets:"
	@echo "  all            - Build all packages (default)"
	@echo "  build          - Build all packages"
	@echo "  test           - Run tests"
	@echo "  test-coverage  - Run tests with coverage report (internal tests only)"
	@echo "  coverage       - Run comprehensive coverage (internal + external tests)"
	@echo "  coverage-badge - Run comprehensive coverage and update badge"
	@echo "  clean          - Remove build artifacts"
	@echo "  docker-up      - Start Docker containers"
	@echo "  docker-down    - Stop Docker containers"
	@echo "  lint           - Run linter"
	@echo "  fmt            - Format code"
	@echo "  deps           - Download and tidy dependencies"
	@echo "  docs           - Generate and serve documentation"
	@echo "  generate-datasets - Generate SQLite test datasets"
	@echo "  generate-benchmark-data - Generate benchmark data for fast access"
	@echo "  generate-all-data - Generate all data (datasets + benchmark data)"
	@echo "  web-app-start  - Start web app server"
	@echo "  web-app-stop   - Stop web app server"
	@echo "  examples       - Show available examples"
	@echo "  example-*      - Run specific example (see 'make examples' for list)"
	@echo "  benchmark      - Run all benchmarks (alias for benchmark-all)"
	@echo "  benchmark-all  - Run all benchmark tests"
	@echo "  benchmark-quick - Run quick functionality test"
	@echo "  benchmark-append - Run append performance benchmark"
	@echo "  benchmark-isolation - Run isolation level benchmark"
	@echo "  benchmark-concurrency - Run concurrency test"
	@echo "  benchmark-go   - Run basic Go library benchmarks"
	@echo "  benchmark-go-enhanced - Run enhanced Go benchmarks with complex scenarios"
	@echo "  benchmark-go-all - Run all Go benchmarks (basic + enhanced)"
	@echo "  benchmark-go - Run Go library benchmarks"
	@echo "  benchmark-web-app - Run web-app benchmarks with SQLite test data"
	@echo "  benchmark-web-app-appendif - Run web-app appendIf benchmarks with SQLite test data"
	@echo "  benchmark-results - Show benchmark results"
	@echo "  help           - Show this help message" 