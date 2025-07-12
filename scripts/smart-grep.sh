#!/bin/bash

# Smart grep script that excludes large files and binaries for better performance
# Usage: ./scripts/smart-grep.sh "pattern" [additional-grep-options]

if [ $# -eq 0 ]; then
    echo "Usage: $0 <pattern> [grep-options...]"
    echo "Example: $0 'EventStore' --include='*.go'"
    exit 1
fi

PATTERN="$1"
shift

# Exclude patterns for large files and binaries
EXCLUDE_PATTERNS=(
    "-path" "./.git" "-prune" "-o"
    "-path" "./benchmark-results" "-prune" "-o"
    "-path" "./.idea" "-prune" "-o"
    "-path" "./vendor" "-prune" "-o"
    "-path" "./node_modules" "-prune" "-o"
    "-name" "*.exe" "-o"
    "-name" "*.dll" "-o"
    "-name" "*.so" "-o"
    "-name" "*.dylib" "-o"
    "-name" "*.test" "-o"
    "-name" "*.db" "-o"
    "-name" "*.sqlite" "-o"
    "-name" "*.sqlite3" "-o"
    "-name" "*.out" "-o"
    "-name" "coverage.*" "-o"
    "-name" "*.coverprofile" "-o"
    "-name" "profile.cov" "-o"
    "-name" "*.prof" "-o"
    "-name" "*.json" "-size" "+1M" "-o"
    "-name" "batch" "-o"
    "-name" "benchmark" "-o"
    "-name" "channel_projection" "-o"
    "-name" "decision_model" "-o"
    "-name" "enrollment" "-o"
    "-name" "transfer" "-o"
    "-name" "web-app" "-o"
    "-name" "streaming" "-o"
    "-name" "server" "-o"
    "-name" "benchmarks" "-o"
    "-name" "command_execution" "-o"
    "-type" "f" "-print"
)

find . \( "${EXCLUDE_PATTERNS[@]}" \) | xargs grep "$@" "$PATTERN" 2>/dev/null 