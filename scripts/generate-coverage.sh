#!/bin/bash

# Script to generate comprehensive code coverage for go-crablet
# This script runs both internal and external tests and merges coverage using gocovmerge
# Usage: ./scripts/generate-coverage.sh [update-badge]

set -e

echo "🧪 Generating comprehensive code coverage for go-crablet..."

# Clean up any existing coverage files
rm -f coverage.out coverage_combined.out coverage_internal.out coverage_external.out

# Step 1: Run internal tests (pkg/dcb only)
echo "📦 Running internal tests (pkg/dcb)..."
go test -v -coverpkg=github.com/rodolfodpk/go-crablet/pkg/dcb -coverprofile=coverage_internal.out ./pkg/dcb || {
    echo "❌ Internal tests failed"
    exit 1
}

# Step 2: Run external tests (pkg/dcb/tests)
echo "🔗 Running external tests (pkg/dcb/tests)..."
go test -v -coverpkg=github.com/rodolfodpk/go-crablet/pkg/dcb -coverprofile=coverage_external.out ./pkg/dcb/tests || {
    echo "❌ External tests failed"
    exit 1
}

# Step 3: Merge coverage files using gocovmerge
echo "🔀 Merging coverage files with gocovmerge..."
GOCOVMERGE_BIN=$(command -v gocovmerge || true)
if [ -z "$GOCOVMERGE_BIN" ]; then
    if [ -x "$HOME/go/bin/gocovmerge" ]; then
        GOCOVMERGE_BIN="$HOME/go/bin/gocovmerge"
    elif [ -x "$GOPATH/bin/gocovmerge" ]; then
        GOCOVMERGE_BIN="$GOPATH/bin/gocovmerge"
    else
        echo "❌ gocovmerge not found. Please install it with: go install github.com/wadey/gocovmerge@latest"
        exit 1
    fi
fi
"$GOCOVMERGE_BIN" coverage_internal.out coverage_external.out > coverage_combined.out

# Step 4: Generate coverage report
echo "📊 Generating coverage report..."
COVERAGE_PERCENT=$(go tool cover -func=coverage_combined.out | grep total: | awk '{print $3}')
echo "📈 Total coverage: $COVERAGE_PERCENT"
go tool cover -html=coverage_combined.out -o coverage.html

echo "\n📋 Detailed coverage breakdown:"
go tool cover -func=coverage_combined.out

echo "\n📁 Generated files:"
echo "  - coverage_internal.out: Internal tests coverage"
echo "  - coverage_external.out: External tests coverage"
echo "  - coverage_combined.out: Combined coverage (main file)"
echo "  - coverage.html: HTML coverage report"

echo "\n🎉 Coverage generation completed successfully!"
echo "📊 Final coverage: $COVERAGE_PERCENT"

# Optionally update badge if requested
if [ "$1" == "update-badge" ]; then
    ./scripts/update-coverage-badge.sh "$COVERAGE_PERCENT"
fi 