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

# Run tests with coverage
test-coverage:
	$(GO) test -v -coverprofile=coverage.out ./pkg/...
	$(GO) tool cover -html=coverage.out

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f coverage.out
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

benchmark-all: benchmark-quick benchmark-append benchmark-isolation benchmark-concurrency benchmark-results

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
	@echo "  test-coverage  - Run tests with coverage report"
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
	@echo "  benchmark-results - Show benchmark results"
	@echo "  help           - Show this help message" 