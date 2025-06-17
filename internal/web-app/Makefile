.PHONY: help build run test clean docker-build docker-run docker-stop docker-logs k6-test setup-db start-db

# Default target
help:
	@echo "Available commands:"
	@echo "  build        - Build the Go binary"
	@echo "  run          - Run the application locally"
	@echo "  test         - Run k6 performance tests (with clean start)"
	@echo "  clean-test   - Clean environment and run k6 tests"
	@echo "  clean        - Clean build artifacts"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-run   - Start web app with Docker Compose"
	@echo "  docker-stop  - Stop Docker Compose services"
	@echo "  docker-logs  - View application logs"
	@echo "  start-db     - Start PostgreSQL database"
	@echo "  setup-db     - Setup database (manual instructions)"

# Build the Go binary
build:
	@echo "Building web application..."
	go build -o web-app ./internal/web-app

# Run the application locally
run:
	@echo "Running web application..."
	go run ./internal/web-app

# Run k6 performance tests
test:
	@echo "Running k6 performance tests..."
	@echo "Cleaning up previous data..."
	docker-compose down -v
	@echo "Starting fresh stack..."
	docker-compose up -d
	@echo "Waiting for services to be ready..."
	@sleep 10
	@echo "Running k6 load test..."
	k6 run k6-test.js

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -f web-app

# Build Docker image
docker-build:
	@echo "Building Docker image..."
	docker build -f internal/web-app/Dockerfile -t dcb-bench .

# Start PostgreSQL database
start-db:
	@echo "Starting PostgreSQL database..."
	docker-compose up -d postgres

# Start web app with Docker Compose
docker-run:
	@echo "Starting web application with Docker Compose..."
	docker-compose up -d

# Stop Docker Compose services
docker-stop:
	@echo "Stopping Docker Compose services..."
	docker-compose down

# View application logs
docker-logs:
	@echo "Viewing application logs..."
	docker-compose logs -f web-app

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
quick-test:
	@echo "Running quick API test..."
	@echo "Testing append endpoint..."
	curl -X POST http://localhost:8080/append \
		-H "Content-Type: application/json" \
		-d '{"events": {"type": "TestEvent", "data": "test", "tags": ["test:1"]}}' \
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
	@docker-compose up -d
	@echo "4. Application is ready at http://localhost:8080"

# Clean test environment and run k6 tests
clean-test:
	@echo "Cleaning test environment and running k6 tests..."
	@echo "Stopping and removing all containers and volumes..."
	docker-compose down -v
	@echo "Starting fresh stack..."
	docker-compose up -d
	@echo "Waiting for services to be ready..."
	@sleep 10
	@echo "Running k6 load test..."
	k6 run k6-test.js 