.PHONY: all build test clean docker-up docker-down lint help

# Variables
BINARY_NAME=crablet
GO=go
DOCKER_COMPOSE=docker-compose

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
	@echo "  help           - Show this help message" 