#!/bin/bash

# DCB Benchmark Runner Script
# This script provides easy commands to run different benchmark scenarios

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
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

# Function to check prerequisites
check_prerequisites() {
    print_status "Checking prerequisites..."
    
    # Check if Docker is running
    if ! docker info > /dev/null 2>&1; then
        print_error "Docker is not running. Please start Docker and try again."
        exit 1
    fi
    
    # Check if docker-compose is available
    if ! command -v docker-compose > /dev/null 2>&1; then
        print_error "docker-compose is not installed. Please install docker-compose and try again."
        exit 1
    fi
    
    # Check if Go is installed
    if ! command -v go > /dev/null 2>&1; then
        print_error "Go is not installed. Please install Go 1.21+ and try again."
        exit 1
    fi
    
    # Check Go version
    GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
    REQUIRED_VERSION="1.21"
    
    if [ "$(printf '%s\n' "$REQUIRED_VERSION" "$GO_VERSION" | sort -V | head -n1)" != "$REQUIRED_VERSION" ]; then
        print_warning "Go version $GO_VERSION detected. Go 1.21+ is recommended."
    fi
    
    # Check if docker-compose is running
    if ! docker-compose ps | grep -q "postgres.*Up"; then
        print_warning "Docker Compose is not running. Starting it now..."
        cd ../..
        if docker-compose up -d; then
            print_success "Docker Compose started successfully"
            # Wait a bit for the database to be ready
            sleep 5
        else
            print_error "Failed to start Docker Compose. Please run 'docker-compose up -d' manually and try again."
            exit 1
        fi
        cd internal/benchmarks
    else
        print_success "Docker Compose is already running"
    fi
    
    print_success "Prerequisites check passed"
}

# Function to reset database
reset_database() {
    print_status "Resetting database for clean benchmark run..."
    cd ../..
    if docker-compose down -v; then
        print_success "Database reset successfully"
        if docker-compose up -d; then
            print_success "Database started successfully"
            # Wait for database to be ready
            print_status "Waiting for database to be ready..."
            sleep 15
        else
            print_error "Failed to start database"
            exit 1
        fi
    else
        print_error "Failed to reset database"
        exit 1
    fi
    cd internal/benchmarks
}

# Function to run benchmarks
run_benchmarks() {
    local pattern=$1
    local dataset_size=$2
    local bench_time=${3:-10s}
    local count=${4:-1}
    
    print_status "Running benchmarks: $pattern with $dataset_size dataset"
    print_status "Benchmark time: $bench_time, Count: $count"
    
    # Reset database before each benchmark run
    reset_database
    
    cd benchmarks
    
    local cmd="go test -bench=$pattern -benchmem -benchtime=$bench_time -count=$count"
    
    if [ "$VERBOSE" = "true" ]; then
        cmd="$cmd -v"
    fi
    
    print_status "Executing: $cmd"
    echo
    
    if eval $cmd; then
        print_success "Benchmarks completed successfully"
    else
        print_error "Benchmarks failed"
        exit 1
    fi
    
    cd ..
}

# Function to run all benchmarks
run_all_benchmarks() {
    local dataset_size=$1
    local bench_time=${2:-10s}
    
    print_status "Running all benchmarks with $dataset_size dataset"
    
    # Run append benchmarks
    run_benchmarks "BenchmarkAppend_$dataset_size" "$dataset_size" "$bench_time"
    
    # Run read benchmarks
    run_benchmarks "BenchmarkRead_$dataset_size" "$dataset_size" "$bench_time"
    
    # Run projection benchmarks
    run_benchmarks "BenchmarkProjection_$dataset_size" "$dataset_size" "$bench_time"
    
    print_success "All benchmarks completed"
}

# Function to run quick benchmarks
run_quick_benchmarks() {
    print_status "Running quick benchmarks (tiny dataset, 5s each)"
    run_all_benchmarks "Tiny" "5s"
}

# Function to run comprehensive benchmarks
run_comprehensive_benchmarks() {
    print_status "Running comprehensive benchmarks (tiny and small datasets, 30s each)"
    
    for size in "Tiny" "Small"; do
        print_status "Running $size dataset benchmarks..."
        run_all_benchmarks "$size" "30s"
    done
    
    print_success "Comprehensive benchmarks completed"
}

# Function to run memory benchmarks
run_memory_benchmarks() {
    local dataset_size=$1
    local bench_time=${2:-10s}
    
    print_status "Running memory usage benchmarks with $dataset_size dataset"
    run_benchmarks "BenchmarkMemory" "$dataset_size" "$bench_time"
}

# Function to run profiling
run_profiling() {
    local benchmark=$1
    local dataset_size=$2
    
    print_status "Running profiling for $benchmark with $dataset_size dataset"
    
    cd benchmarks
    
    # CPU profiling
    print_status "Generating CPU profile..."
    go test -bench="$benchmark" -benchmem -cpuprofile=cpu.prof -benchtime=30s
    
    # Memory profiling
    print_status "Generating memory profile..."
    go test -bench="$benchmark" -benchmem -memprofile=mem.prof -benchtime=30s
    
    print_success "Profiles generated: cpu.prof, mem.prof"
    print_status "To analyze profiles:"
    print_status "  go tool pprof cpu.prof"
    print_status "  go tool pprof mem.prof"
    
    cd ..
}

# Function to show help
show_help() {
    echo "DCB Benchmark Runner"
    echo
    echo "Usage: $0 [COMMAND] [OPTIONS]"
    echo
    echo "Commands:"
    echo "  prepare                  Prepare and cache datasets (run once per machine)"
    echo "  quick                    Run quick benchmarks (tiny dataset, 5s each)"
    echo "  comprehensive           Run comprehensive benchmarks (all datasets, 30s each)"
    echo "  append [SIZE]           Run append benchmarks"
    echo "  read [SIZE]             Run read benchmarks"
    echo "  projection [SIZE]       Run projection benchmarks"
    echo "  memory [SIZE]           Run memory usage benchmarks"
    echo "  all [SIZE]              Run all benchmarks for a dataset size"
    echo "  profile [BENCHMARK] [SIZE] Run profiling for specific benchmark"
    echo "  help                    Show this help message"
    echo
    echo "Dataset Sizes:"
    echo "  Tiny                   5 courses, 10 students, 20 enrollments (quick validation)"
    echo "  Small                  1,000 courses, 10,000 students, 50,000 enrollments (performance testing)"
    echo
    echo "Options:"
    echo "  -t, --time TIME         Benchmark time (default: 10s)"
    echo "  -c, --count COUNT       Number of benchmark runs (default: 1)"
    echo "  -v, --verbose           Verbose output"
    echo
    echo "Examples:"
    echo "  $0 prepare                                 # Prepare datasets (run once)"
    echo "  $0 quick                                    # Quick benchmarks"
    echo "  $0 append Small                             # Append benchmarks with small dataset"
    echo "  $0 all Large -t 30s                         # All benchmarks with large dataset, 30s each"
    echo "  $0 profile BenchmarkAppendSingle_Small Small # Profile specific benchmark"
    echo
}

# Function to prepare datasets
prepare_datasets() {
    print_status "Preparing and caching datasets..."
    
    if go run tools/prepare_datasets_main.go; then
        print_success "Datasets prepared and cached successfully"
    else
        print_error "Failed to prepare datasets"
        exit 1
    fi
}

# Parse command line arguments
VERBOSE=false
BENCH_TIME="10s"
COUNT="1"

while [[ $# -gt 0 ]]; do
    case $1 in
        -t|--time)
            BENCH_TIME="$2"
            shift 2
            ;;
        -c|--count)
            COUNT="$2"
            shift 2
            ;;
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        help)
            show_help
            exit 0
            ;;
        prepare)
            check_prerequisites
            prepare_datasets
            exit 0
            ;;
        quick)
            check_prerequisites
            run_quick_benchmarks
            exit 0
            ;;
        comprehensive)
            check_prerequisites
            run_comprehensive_benchmarks
            exit 0
            ;;
        append)
            check_prerequisites
            run_benchmarks "BenchmarkAppend_${2:-Small}" "${2:-Small}" "$BENCH_TIME" "$COUNT"
            exit 0
            ;;
        read)
            check_prerequisites
            run_benchmarks "BenchmarkRead_${2:-Small}" "${2:-Small}" "$BENCH_TIME" "$COUNT"
            exit 0
            ;;
        projection)
            check_prerequisites
            run_benchmarks "BenchmarkProjection_${2:-Small}" "${2:-Small}" "$BENCH_TIME" "$COUNT"
            exit 0
            ;;
        memory)
            check_prerequisites
            run_memory_benchmarks "${2:-Small}" "$BENCH_TIME"
            exit 0
            ;;
        all)
            check_prerequisites
            run_all_benchmarks "${2:-Small}" "$BENCH_TIME"
            exit 0
            ;;
        profile)
            check_prerequisites
            run_profiling "$2" "${3:-Small}"
            exit 0
            ;;
        *)
            print_error "Unknown command: $1"
            echo
            show_help
            exit 1
            ;;
    esac
done

# If no command provided, show help
show_help 