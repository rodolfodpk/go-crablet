.PHONY: all build test clean docker-up docker-down lint help benchmark benchmark-all benchmark-quick benchmark-append benchmark-isolation benchmark-concurrency benchmark-results

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
	@cd internal/benchmarks/benchmarks && $(GO) test -bench=. -benchmem -benchtime=2s -timeout=5m . > ../../../$(BENCHMARK_RESULTS_DIR)/go_benchmarks_$(TIMESTAMP).txt 2>&1 || true
	@echo "✅ Go benchmarks completed"

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
	@echo "  docs           - Generate and serve documentation"
	@echo "  benchmark      - Run all benchmarks (alias for benchmark-all)"
	@echo "  benchmark-all  - Run all benchmark tests"
	@echo "  benchmark-quick - Run quick functionality test"
	@echo "  benchmark-append - Run append performance benchmark"
	@echo "  benchmark-isolation - Run isolation level benchmark"
	@echo "  benchmark-concurrency - Run concurrency test"
	@echo "  benchmark-go - Run Go library benchmarks"
	@echo "  benchmark-web-app - Run web-app benchmarks with SQLite test data"
	@echo "  benchmark-web-app-appendif - Run web-app appendIf benchmarks with SQLite test data"
	@echo "  benchmark-results - Show benchmark results"
	@echo "  help           - Show this help message" 