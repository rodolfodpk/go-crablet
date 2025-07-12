.PHONY: help build run test clean k6-test setup-db start-db stop-db run-server kill-server quick-test full-scan concurrency-test ensure-server cleanup-db full append-quick append-full append-if-quick append-if-full append-if-isolated-quick append-if-isolated-full

# Default target
help:
	@echo "Available commands:"
	@echo ""
	@echo "🏗️  BUILD & RUN:"
	@echo "  build        - Build the Go binary"
	@echo "  run          - Run the application locally"
	@echo "  run-server   - Run the web application"
	@echo "  kill-server  - Kill the web application server"
	@echo "  ensure-server - Ensure server is running on port 8080"
	@echo ""
	@echo "🗄️  DATABASE:"
	@echo "  start-db     - Start PostgreSQL database (from root)"
	@echo "  stop-db      - Stop PostgreSQL database (from root)"
	@echo "  setup-db     - Setup database (manual instructions)"
	@echo "  cleanup-db   - Clean database using HTTP endpoint"
	@echo ""
	@echo "🏃‍♂️ QUICK TESTS (30s-2m):"
	@echo "  test-quick   - Run all quick validation tests"
	@echo "  quick-basic  - Basic functionality test"
	@echo "  quick-append - Quick append validation"
	@echo "  quick-isolation-levels - Consolidated isolation levels test"
	@echo "  quick-conditional-append - Consolidated conditional append test"
	@echo ""
	@echo "📊 BENCHMARKS (3-5m):"
	@echo "  test-benchmarks - Run all benchmark tests"
	@echo "  benchmark-append - Full append performance benchmark"
	@echo "  benchmark-append-if - Conditional append benchmark"
	@echo "  benchmark-append-isolated - Serializable isolation benchmark"
	@echo "  benchmark-isolation - Compare all isolation levels"
	@echo ""
	@echo "🧪 FUNCTIONAL TESTS (2-4m):"
	@echo "  test-functional - Run all functional tests"
	@echo "  test-concurrency - Basic concurrency testing"
	@echo "  test-advisory-locks - Advisory locks concurrency"
	@echo ""
	@echo "🔥 LOAD TESTS (5-10m):"
	@echo "  test-load     - Run all load tests"
	@echo "  load-full     - Complete system load test"
	@echo "  load-full-scan - Full scan performance under load"
	@echo ""
	@echo "🧹 UTILITY:"
	@echo "  clean        - Clean build artifacts"
	@echo "  test         - Run k6 performance tests (legacy - use load-full)"
	@echo "  clean-test   - Clean environment and run k6 tests (legacy - use load-full)"
	@echo "  benchmark    - Run all benchmarks (legacy - use test-benchmarks)"

# Build the Go binary
build:
	@echo "Building web application..."
	go build -o web-app main.go

# Run the application locally
run:
	@echo "Running web application..."
	go run main.go

# Ensure server is running on port 8080
ensure-server:
	@echo "Checking if web-app server is running on port 8080..."
	@if ! curl -s http://localhost:8080/health > /dev/null; then \
		echo "Server not running. Starting web-app server..."; \
		make run-server; \
		sleep 3; \
	else \
		echo "Server is already running on port 8080"; \
	fi

# Clean database using HTTP endpoint
cleanup-db:
	@echo "Cleaning database using HTTP endpoint..."
	@curl -X POST http://localhost:8080/cleanup -s

# Run k6 performance tests (legacy - use load-full instead)
test:
	@echo "Running k6 performance tests (legacy - use 'make load-full' instead)..."
	@make load-full

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -f web-app

# Start PostgreSQL database (from root)
start-db:
	@echo "Starting PostgreSQL database..."
	cd ../.. && docker-compose up -d postgres

# Stop PostgreSQL database (from root)
stop-db:
	@echo "Stopping PostgreSQL database..."
	cd ../.. && docker-compose stop postgres

# Setup database (manual setup)
setup-db:
	@echo "Setting up database..."
	@echo "Make sure PostgreSQL is running and create database 'dcb_app'"
	@echo "Example: createdb dcb_app"
	@echo "Or use: make start-db"

# Install k6 (macOS)
install-k6:
	@echo "Installing k6..."
	brew install k6

# Quick test with curl
curl-test:
	@echo "Running quick API test..."
	@echo "Testing append endpoint..."
	curl -X POST http://localhost:8080/append \
		-H "Content-Type: application/json" \
		-d '{"events": [{"type": "TestEvent", "data": "test", "tags": ["test:1"]}]}' \
		-s | jq .
	@echo ""
	@echo "Testing read endpoint..."
	curl -X POST http://localhost:8080/read \
		-H "Content-Type: application/json" \
		-d '{"query": {"items": [{"types": ["TestEvent"], "tags": ["test:1"]}]}}' \
		-s | jq .

# Full setup and run
setup-and-run:
	@echo "Setting up and running the complete stack..."
	@echo "1. Starting PostgreSQL database..."
	@cd ../.. && docker-compose up -d postgres
	@echo "2. Waiting for database to be ready..."
	@sleep 10
	@echo "3. Starting web application..."
	@make run-server
	@echo "4. Application is ready at http://localhost:8080"

# Clean test environment and run k6 tests (legacy - use load-full instead)
clean-test:
	@echo "Clean test environment (legacy - use 'make load-full' instead)..."
	@make load-full

# Run the web application
run-server:
	@echo "Starting web-app server on port $(PORT)..."
	PORT=$(PORT) DATABASE_URL=$(DB_URL) go run main.go & \
	echo $$! > .server.pid

# Kill the web application server
kill-server:
	@echo "Stopping web application server..."
	@if [ -f .server.pid ]; then \
		kill $$(cat .server.pid) 2>/dev/null || true; \
		rm -f .server.pid; \
	fi
	@pkill -f "go run main.go" 2>/dev/null || true
	@pkill -f "main.go" 2>/dev/null || true

# Variables
DB_URL=postgres://postgres:postgres@localhost:5432/dcb_app?sslmode=disable
PORT=8080

# Benchmark (legacy - use test-benchmarks instead)
benchmark:
	@echo "Running benchmarks (legacy - use 'make test-benchmarks' instead)..."
	@make test-benchmarks

# =============================================================================
# QUICK TESTS (30s-2m)
# =============================================================================

# Run all quick tests
test-quick:
	@echo "Running all quick validation tests..."
	@make ensure-server
	@make cleanup-db
	@echo "Running basic functionality test..."
	k6 run k6/quick/quick.js
	@echo "Running quick append validation..."
	k6 run k6/quick/append-quick.js
	@echo "Running consolidated isolation levels test..."
	k6 run k6/quick/isolation-levels-quick.js
	@echo "Running consolidated conditional append test..."
	k6 run k6/quick/conditional-append-quick.js
	@echo "All quick tests completed."

# Basic functionality test
quick-basic:
	@echo "Running basic functionality test..."
	@make ensure-server
	@make cleanup-db
	k6 run k6/quick/quick.js
	@echo "Basic functionality test completed."

# Quick append validation
quick-append:
	@echo "Running quick append validation..."
	@make ensure-server
	@make cleanup-db
	k6 run k6/quick/append-quick.js
	@echo "Quick append validation completed."

# Consolidated isolation levels test
quick-isolation-levels:
	@echo "Running consolidated isolation levels test..."
	@make ensure-server
	@make cleanup-db
	k6 run k6/quick/isolation-levels-quick.js
	@echo "Consolidated isolation levels test completed."

# Consolidated conditional append test
quick-conditional-append:
	@echo "Running consolidated conditional append test..."
	@make ensure-server
	@make cleanup-db
	k6 run k6/quick/conditional-append-quick.js
	@echo "Consolidated conditional append test completed."

# =============================================================================
# BENCHMARKS (3-5m)
# =============================================================================

# Run all benchmark tests
test-benchmarks:
	@echo "Running all benchmark tests..."
	@make ensure-server
	@make cleanup-db
	@echo "Running append performance benchmark..."
	k6 run k6/benchmarks/append-benchmark.js
	@echo "Running conditional append benchmark..."
	k6 run k6/benchmarks/append-if-benchmark.js
	@echo "Running isolated append benchmark..."
	k6 run k6/benchmarks/append-if-isolated-benchmark.js
	@echo "Running isolation level comparison..."
	k6 run k6/benchmarks/isolation-level-benchmark.js
	@echo "All benchmark tests completed."

# Full append performance benchmark
benchmark-append:
	@echo "Running full append performance benchmark..."
	@make ensure-server
	@make cleanup-db
	k6 run k6/benchmarks/append-benchmark.js
	@echo "Append performance benchmark completed."

# Conditional append benchmark
benchmark-append-if:
	@echo "Running conditional append benchmark..."
	@make ensure-server
	@make cleanup-db
	k6 run k6/benchmarks/append-if-benchmark.js
	@echo "Conditional append benchmark completed."

# Serializable isolation benchmark
benchmark-append-isolated:
	@echo "Running Serializable isolation benchmark..."
	@make ensure-server
	@make cleanup-db
	k6 run k6/benchmarks/append-if-isolated-benchmark.js
	@echo "Serializable isolation benchmark completed."

# Compare all isolation levels
benchmark-isolation:
	@echo "Running isolation level comparison..."
	@make ensure-server
	@make cleanup-db
	k6 run k6/benchmarks/isolation-level-benchmark.js
	@echo "Isolation level comparison completed."

# =============================================================================
# FUNCTIONAL TESTS (2-4m)
# =============================================================================

# Run all functional tests
test-functional:
	@echo "Running all functional tests..."
	@make ensure-server
	@make cleanup-db
	@echo "Running basic concurrency test..."
	k6 run k6/tests/k6-concurrency-test.js
	@echo "Running advisory locks concurrency test..."
	k6 run k6/tests/k6-advisory-locks-concurrency-test.js
	@echo "All functional tests completed."

# Basic concurrency testing
test-concurrency:
	@echo "Running basic concurrency test..."
	@make ensure-server
	@make cleanup-db
	k6 run k6/tests/k6-concurrency-test.js
	@echo "Basic concurrency test completed."

# Advisory locks concurrency
test-advisory-locks:
	@echo "Running advisory locks concurrency test..."
	@make ensure-server
	@make cleanup-db
	k6 run k6/tests/k6-advisory-locks-concurrency-test.js
	@echo "Advisory locks concurrency test completed."

# =============================================================================
# LOAD TESTS (5-10m)
# =============================================================================

# Run all load tests
test-load:
	@echo "Running all load tests..."
	@make ensure-server
	@make cleanup-db
	@echo "Running complete system load test..."
	k6 run k6/full.js
	@echo "Running full scan performance test..."
	k6 run k6/full-scan.js
	@echo "All load tests completed."

# Complete system load test
load-full:
	@echo "Running complete system load test..."
	@make ensure-server
	@make cleanup-db
	k6 run k6/full.js
	@echo "Complete system load test completed."

# Full scan performance under load
load-full-scan:
	@echo "Running full scan performance test..."
	@make ensure-server
	@make cleanup-db
	k6 run k6/full-scan.js
	@echo "Full scan performance test completed."

# =============================================================================
# LEGACY COMPATIBILITY
# =============================================================================

# Full scenario test (legacy)
full:
	@echo "Running full scenario test (5m, up to 100 VUs)..."
	@echo "Running k6 full scenario test..."
	k6 run k6/full.js
	@echo "Full scenario test completed."

# =============================================================================
# LEGACY COMPATIBILITY TARGETS
# =============================================================================

# Full scan test (resource-intensive queries) - legacy
full-scan:
	@echo "Running full-scan test (resource-intensive queries)..."
	@make ensure-server
	@make cleanup-db
	@echo "Running k6 full-scan test..."
	k6 run k6/full-scan.js
	@echo "Full-scan test completed."

# Quick test with k6 - legacy
quick-test:
	@echo "Running quick test (10s)..."
	@make ensure-server
	@make cleanup-db
	@echo "Running k6 quick test..."
	k6 run k6/quick/quick.js
	@echo "Quick test completed."

# Concurrency test (tests optimistic locking) - legacy
concurrency-test:
	@echo "Running concurrency test (4m) - tests optimistic locking..."
	@make ensure-server
	@make cleanup-db
	@echo "Running k6 concurrency test..."
	k6 run k6/tests/k6-concurrency-test.js
	@echo "Concurrency test completed."

# Quick append benchmark (30s) - legacy
append-quick:
	@echo "Running quick append benchmark (30s)..."
	@make ensure-server
	@make cleanup-db
	@echo "Running k6 append quick benchmark..."
	k6 run k6/quick/append-quick.js
	@echo "Quick append benchmark completed."

# Full append benchmark (6m, up to 200 VUs) - legacy
append-full:
	@echo "Running full append benchmark (6m, up to 200 VUs)..."
	@make ensure-server
	@make cleanup-db
	@echo "Running k6 append full benchmark..."
	k6 run k6/benchmarks/append-benchmark.js
	@echo "Full append benchmark completed."

# Quick appendIf benchmark (30s) - legacy
append-if-quick:
	@echo "Running quick appendIf benchmark (30s)..."
	@make ensure-server
	@make cleanup-db
	@echo "Running k6 appendIf quick benchmark..."
	k6 run k6/quick/append-if-quick.js
	@echo "Quick appendIf benchmark completed."

# Full appendIf benchmark (6m, up to 200 VUs) - legacy
append-if-full:
	@echo "Running full appendIf benchmark (6m, up to 200 VUs)..."
	@make ensure-server
	@make cleanup-db
	@echo "Running k6 appendIf full benchmark..."
	k6 run k6/benchmarks/append-if-benchmark.js
	@echo "Full appendIf benchmark completed."

# Quick appendIfIsolated benchmark (30s) - legacy
append-if-isolated-quick:
	@echo "Running quick appendIfIsolated benchmark (30s)..."
	@make ensure-server
	@make cleanup-db
	@echo "Running k6 appendIfIsolated quick benchmark..."
	k6 run k6/quick/append-if-isolated-quick.js
	@echo "Quick appendIfIsolated benchmark completed."

# Full appendIfIsolated benchmark (6m, up to 100 VUs) - legacy
append-if-isolated-full:
	@echo "Running full appendIfIsolated benchmark (6m, up to 100 VUs)..."
	@make ensure-server
	@make cleanup-db
	@echo "Running k6 appendIfIsolated full benchmark..."
	k6 run k6/benchmarks/append-if-isolated-benchmark.js
	@echo "Full appendIfIsolated benchmark completed." 