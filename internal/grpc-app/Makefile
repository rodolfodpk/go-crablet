# Makefile for gRPC app benchmark

.PHONY: benchmark quick-test full full-scan concurrency-test clean start-db stop-db run-server kill-server ensure-server

# Variables
DB_URL=postgres://postgres:postgres@localhost:5432/dcb_app?sslmode=disable
PORT=9090

benchmark: clean start-db ensure-server full
	@echo "\nBenchmark complete."

quick-test:
	@echo "Running quick gRPC test (10s)..."
	@echo "Ensuring database is running..."
	cd ../.. && docker-compose up -d postgres
	@echo "Waiting for database to be ready..."
	@sleep 5
	@echo "Ensuring gRPC server is running..."
	@make ensure-server
	@echo "Cleaning up database before test..."
	@curl -X POST http://localhost:9091/cleanup || echo "Cleanup failed, continuing anyway..."
	@echo "Running k6 quick test..."
	k6 run k6-quick-test.js
	@echo "Quick test completed."

full:
	@echo "Running gRPC full scenario test..."
	@echo "Running k6 full scenario test..."
	k6 run full.js
	@echo "Full scenario test completed."

full-scan:
	@echo "Running gRPC full-scan test (resource-intensive queries)..."
	@echo "Ensuring database is running..."
	cd ../.. && docker-compose up -d postgres
	@echo "Waiting for database to be ready..."
	@sleep 5
	@echo "Ensuring gRPC server is running..."
	@make ensure-server
	@echo "Cleaning up database before test..."
	@curl -X POST http://localhost:9091/cleanup || echo "Cleanup failed, continuing anyway..."
	@echo "Running k6 full-scan test..."
	k6 run full-scan.js
	@echo "Full-scan test completed."

concurrency-test:
	@echo "Running gRPC concurrency test (4m) - tests optimistic locking..."
	@echo "Ensuring database is running..."
	cd ../.. && docker-compose up -d postgres
	@echo "Waiting for database to be ready..."
	@sleep 5
	@echo "Ensuring gRPC server is running..."
	@make ensure-server
	@echo "Cleaning up database before test..."
	@curl -X POST http://localhost:9091/cleanup || echo "Cleanup failed, continuing anyway..."
	@echo "Running k6 concurrency test..."
	k6 run k6-grpc-concurrency-test.js
	@echo "Concurrency test completed."

start-db:
	@echo "Starting PostgreSQL via docker-compose..."
	cd ../.. && docker-compose up -d postgres
	@echo "Creating database if needed..."
	cd ../.. && docker exec postgres_db psql -U postgres -c "CREATE DATABASE dcb_app;" || true

stop-db:
	@echo "Stopping PostgreSQL..."
	cd ../.. && docker-compose stop postgres

run-server:
	@echo "Starting gRPC server on port $(PORT)..."
	PORT=$(PORT) DATABASE_URL=$(DB_URL) go run server/main.go

ensure-server:
	@echo "Checking if gRPC server is running on port $(PORT)..."
	@if ! lsof -i:$(PORT) > /dev/null 2>&1; then \
		echo "gRPC server not running. Starting it in background..."; \
		PORT=$(PORT) DATABASE_URL=$(DB_URL) go run server/main.go & \
		echo $$! > .server.pid; \
		sleep 3; \
		echo "gRPC server started."; \
	else \
		echo "gRPC server is already running on port $(PORT)."; \
	fi

kill-server:
	@echo "Stopping gRPC server..."
	@if [ -f .server.pid ]; then \
		kill $$(cat .server.pid) 2>/dev/null || true; \
		rm -f .server.pid; \
	fi
	@pkill -f "go run server/main.go" 2>/dev/null || true
	@pkill -f "main.go" 2>/dev/null || true

clean:
	@echo "\nWARNING: This will remove all PostgreSQL data (docker-compose down -v)!"
	@read -p "Are you sure you want to continue? [y/N] " ans; \
	if [ "$$ans" = "y" ] || [ "$$ans" = "Y" ]; then \
	  cd ../.. && docker-compose down -v; \
	  echo "PostgreSQL volumes removed."; \
	else \
	  echo "Aborted."; \
	fi 