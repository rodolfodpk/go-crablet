#!/bin/bash

# Script to generate comprehensive code coverage for go-crablet
# This script runs external tests and generates coverage (no internal tests exist)
# Usage: ./scripts/generate-coverage.sh [update-badge]

set -e

echo "🧪 Generating comprehensive code coverage for go-crablet..."

# Clean up any existing coverage files
rm -f coverage.out coverage_combined.out coverage_internal.out coverage_external.out

# Note: No internal tests exist in pkg/dcb/ - all tests are in pkg/dcb/tests/
echo "📦 Running external tests (pkg/dcb/tests)..."
go test -v -coverpkg=github.com/rodolfodpk/go-crablet/pkg/dcb -coverprofile=coverage_external.out ./pkg/dcb/tests || {
    echo "❌ External tests failed"
    exit 1
}

# Copy external coverage as combined coverage (since no internal tests exist)
echo "📊 Using external tests coverage as combined coverage..."
cp coverage_external.out coverage_combined.out

# Generate coverage report
echo "📊 Generating coverage report..."
COVERAGE_PERCENT=$(go tool cover -func=coverage_combined.out | grep total: | awk '{print $3}')
echo "📈 Total coverage: $COVERAGE_PERCENT"
go tool cover -html=coverage_combined.out -o coverage.html

echo "\n📋 Detailed coverage breakdown:"
go tool cover -func=coverage_combined.out

echo "\n📁 Generated files:"
echo "  - coverage_external.out: External tests coverage"
echo "  - coverage_combined.out: Combined coverage (main file)"
echo "  - coverage.html: HTML coverage report"

echo "\n🎉 Coverage generation completed successfully!"
echo "📊 Final coverage: $COVERAGE_PERCENT"

# Optionally update badge if requested
if [ "$1" == "update-badge" ]; then
    ./scripts/update-coverage-badge.sh "$COVERAGE_PERCENT"
fi 