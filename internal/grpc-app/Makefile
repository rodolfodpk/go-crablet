# Makefile for gRPC app benchmark

.PHONY: benchmark quick-test full-benchmark clean start-db stop-db run-server

# Variables
DB_URL=postgres://postgres:postgres@localhost:5432/dcb_app?sslmode=disable
PORT=9090

benchmark: clean start-db run-server full-benchmark
	@echo "\nBenchmark complete."

quick-test:
	@echo "Running quick gRPC test (10s)..."
	k6 run k6-quick-test.js

full-benchmark:
	@echo "Running full gRPC benchmark..."
	k6 run k6-grpc-test.js

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
	@echo "(Press Ctrl+C to stop the server after the benchmark)"
	PORT=$(PORT) DATABASE_URL=$(DB_URL) go run server/main.go & \
	sleep 3

clean:
	@echo "\nWARNING: This will remove all PostgreSQL data (docker-compose down -v)!"
	@read -p "Are you sure you want to continue? [y/N] " ans; \
	if [ "$$ans" = "y" ] || [ "$$ans" = "Y" ]; then \
	  cd ../.. && docker-compose down -v; \
	  echo "PostgreSQL volumes removed."; \
	else \
	  echo "Aborted."; \
	fi 