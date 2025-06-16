#!/bin/bash

# Script to update the coverage badge in README.md
# Usage: ./scripts/update-coverage-badge.sh [coverage_percentage]

set -e

# Default coverage if not provided
COVERAGE=${1:-86.7}

echo "Updating coverage badge to: ${COVERAGE}%"

# Convert to integer for comparison (multiply by 10 to handle decimals)
COVERAGE_INT=$(echo "$COVERAGE * 10" | bc | cut -d. -f1)

# Determine color based on coverage
if [ "$COVERAGE_INT" -ge 900 ]; then
  COLOR="brightgreen"
elif [ "$COVERAGE_INT" -ge 800 ]; then
  COLOR="green"
elif [ "$COVERAGE_INT" -ge 700 ]; then
  COLOR="yellowgreen"
elif [ "$COVERAGE_INT" -ge 600 ]; then
  COLOR="yellow"
elif [ "$COVERAGE_INT" -ge 500 ]; then
  COLOR="orange"
else
  COLOR="red"
fi

# Create the new badge line
NEW_BADGE="[![Code Coverage](https://img.shields.io/badge/code%20coverage-${COVERAGE}%25-${COLOR}?logo=go)](https://github.com/rodolfodpk/go-crablet/actions/workflows/coverage.yml)"

echo "Color: $COLOR"
echo "New badge: $NEW_BADGE"

# Replace the existing codecov badge with our custom one
sed -i '' "s|\[!\[codecov\](https://codecov.io/gh/rodolfodpk/go-crablet/branch/main/graph/badge.svg)\](https://codecov.io/gh/rodolfodpk/go-crablet)|${NEW_BADGE}|" README.md

echo "✅ Successfully updated README.md with new coverage badge"
echo "📊 Coverage: ${COVERAGE}% (${COLOR})" 