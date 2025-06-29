.PHONY: help build run test clean k6-test setup-db start-db stop-db run-server kill-server quick-test full-scan concurrency-test ensure-server cleanup-db full append-quick append-full append-if-quick append-if-full append-if-isolated-quick append-if-isolated-full

# Default target
help:
	@echo "Available commands:"
	@echo "  build        - Build the Go binary"
	@echo "  run          - Run the application locally"
	@echo "  test         - Run k6 performance tests (with clean start)"
	@echo "  clean-test   - Clean environment and run k6 tests"
	@echo "  clean        - Clean build artifacts"
	@echo "  start-db     - Start PostgreSQL database (from root)"
	@echo "  setup-db     - Setup database (manual instructions)"
	@echo "  stop-db      - Stop PostgreSQL database (from root)"
	@echo "  run-server   - Run the web application"
	@echo "  kill-server  - Kill the web application server"
	@echo "  ensure-server - Ensure server is running on port 8080"
	@echo "  cleanup-db   - Clean database using HTTP endpoint"
	@echo "  quick-test   - Quick k6 test (10s)"
	@echo "  full - Full scenario k6 test (5m, up to 100 VUs)"
	@echo "  full-scan    - Full scan k6 test (resource-intensive)"
	@echo "  concurrency-test - Concurrency test (4m)"
	@echo "  append-quick - Quick append benchmark (30s)"
	@echo "  append-full  - Full append benchmark (6m, up to 200 VUs)"
	@echo "  append-if-quick - Quick appendIf benchmark (30s)"
	@echo "  append-if-full  - Full appendIf benchmark (6m, up to 200 VUs)"
	@echo "  append-if-isolated-quick - Quick appendIfIsolated benchmark (30s)"
	@echo "  append-if-isolated-full  - Full appendIfIsolated benchmark (6m, up to 100 VUs)"

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

# Run k6 performance tests
test:
	@echo "Running k6 performance tests..."
	@echo "Cleaning up previous data..."
	cd ../.. && docker-compose down -v
	@echo "Starting fresh database..."
	cd ../.. && docker-compose up -d postgres
	@echo "Waiting for database to be ready..."
	@sleep 10
	@echo "Starting web application..."
	@make run-server &
	@sleep 3
	@echo "Running k6 load test..."
	k6 run full.js
	@echo "Cleaning up after test..."
	@make kill-server
	@make stop-db
	@echo "Test completed."

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

# Clean test environment and run k6 tests
clean-test:
	@echo "Cleaning test environment and running k6 tests..."
	@echo "Stopping and removing all containers and volumes..."
	cd ../.. && docker-compose down -v
	@echo "Starting fresh database..."
	cd ../.. && docker-compose up -d postgres
	@echo "Waiting for database to be ready..."
	@sleep 10
	@echo "Starting web application..."
	@make run-server &
	@sleep 3
	@echo "Running k6 load test..."
	k6 run full.js
	@echo "Cleaning up after test..."
	@make kill-server
	@make stop-db
	@echo "Clean test completed."

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

# Benchmark
benchmark: clean start-db run-server full
	@echo "\nBenchmark complete."

# Full scenario test
full:
	@echo "Running full scenario test (5m, up to 100 VUs)..."
	@echo "Running k6 full scenario test..."
	k6 run full.js
	@echo "Full scenario test completed."

# Full scan test (resource-intensive queries)
full-scan:
	@echo "Running full-scan test (resource-intensive queries)..."
	@make ensure-server
	@make cleanup-db
	@echo "Running k6 full-scan test..."
	k6 run full-scan.js
	@echo "Full-scan test completed."

# Quick test with k6
quick-test:
	@echo "Running quick test (10s)..."
	@make ensure-server
	@make cleanup-db
	@echo "Running k6 quick test..."
	k6 run quick.js
	@echo "Quick test completed."

# Concurrency test (tests optimistic locking)
concurrency-test:
	@echo "Running concurrency test (4m) - tests optimistic locking..."
	@make ensure-server
	@make cleanup-db
	@echo "Running k6 concurrency test..."
	k6 run k6-concurrency-test.js
	@echo "Concurrency test completed."

# Quick append benchmark (30s)
append-quick:
	@echo "Running quick append benchmark (30s)..."
	@make ensure-server
	@make cleanup-db
	@echo "Running k6 append quick benchmark..."
	k6 run append-quick.js
	@echo "Quick append benchmark completed."

# Full append benchmark (6m, up to 200 VUs)
append-full:
	@echo "Running full append benchmark (6m, up to 200 VUs)..."
	@make ensure-server
	@make cleanup-db
	@echo "Running k6 append full benchmark..."
	k6 run append-benchmark.js
	@echo "Full append benchmark completed."

# Quick appendIf benchmark (30s)
append-if-quick:
	@echo "Running quick appendIf benchmark (30s)..."
	@make ensure-server
	@make cleanup-db
	@echo "Running k6 appendIf quick benchmark..."
	k6 run append-if-quick.js
	@echo "Quick appendIf benchmark completed."

# Full appendIf benchmark (6m, up to 200 VUs)
append-if-full:
	@echo "Running full appendIf benchmark (6m, up to 200 VUs)..."
	@make ensure-server
	@make cleanup-db
	@echo "Running k6 appendIf full benchmark..."
	k6 run append-if-benchmark.js
	@echo "Full appendIf benchmark completed."

# Quick appendIfIsolated benchmark (30s)
append-if-isolated-quick:
	@echo "Running quick appendIfIsolated benchmark (30s)..."
	@make ensure-server
	@make cleanup-db
	@echo "Running k6 appendIfIsolated quick benchmark..."
	k6 run append-if-isolated-quick.js
	@echo "Quick appendIfIsolated benchmark completed."

# Full appendIfIsolated benchmark (6m, up to 100 VUs)
append-if-isolated-full:
	@echo "Running full appendIfIsolated benchmark (6m, up to 100 VUs)..."
	@make ensure-server
	@make cleanup-db
	@echo "Running k6 appendIfIsolated full benchmark..."
	k6 run append-if-isolated-benchmark.js
	@echo "Full appendIfIsolated benchmark completed." 