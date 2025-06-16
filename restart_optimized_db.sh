#!/bin/bash

# Script to restart PostgreSQL with optimized settings and run benchmarks
# This script will stop the current database, apply optimizations, and restart it

set -e

echo "🚀 Optimizing PostgreSQL for go-crablet benchmarks..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    print_error "Docker is not running. Please start Docker and try again."
    exit 1
fi

# Stop existing containers
print_status "Stopping existing PostgreSQL container..."
docker-compose down

# Remove existing volume to start fresh with optimized settings
print_status "Removing existing database volume for fresh start..."
docker volume rm go-crablet_postgres_data 2>/dev/null || true

# Start PostgreSQL with optimized settings
print_status "Starting PostgreSQL with optimized configuration..."
docker-compose up -d

# Wait for PostgreSQL to be ready
print_status "Waiting for PostgreSQL to be ready..."
sleep 10

# Check if PostgreSQL is running
if docker-compose ps | grep -q "postgres.*Up"; then
    print_success "PostgreSQL is running with optimized settings"
else
    print_error "PostgreSQL failed to start. Check logs with: docker-compose logs postgres"
    exit 1
fi

# Display resource allocation
print_status "PostgreSQL resource allocation:"
echo "  - CPU: 4 cores (2 reserved, 4 max)"
echo "  - Memory: 4GB (2GB reserved, 4GB max)"
echo "  - Shared buffers: 256MB"
echo "  - Effective cache size: 1GB"
echo "  - Work memory: 16MB"
echo "  - Parallel workers: 8 max"

# Run a quick connectivity test
print_status "Testing database connectivity..."
if docker-compose exec -T postgres pg_isready -U postgres; then
    print_success "Database connectivity test passed"
else
    print_error "Database connectivity test failed"
    exit 1
fi

print_success "PostgreSQL optimization complete!"
echo ""
print_status "You can now run benchmarks with improved performance:"
echo "  cd internal/benchmarks"
echo "  ./run_benchmarks.sh quick"
echo ""
print_status "Or run specific benchmarks:"
echo "  cd internal/benchmarks/benchmarks"
echo "  go test -bench=. -benchmem -benchtime=10s"
echo ""
print_warning "Note: The database has been reset. Any existing data has been removed." 