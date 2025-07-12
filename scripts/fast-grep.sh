#!/bin/bash

# Fast grep that excludes large files and binaries
# Usage: ./scripts/fast-grep.sh "pattern" [grep-options]

if [ $# -eq 0 ]; then
    echo "Usage: $0 <pattern> [grep-options...]"
    echo "Example: $0 'EventStore' --include='*.go'"
    exit 1
fi

PATTERN="$1"
shift

# Use grep with exclude patterns for common large files
grep -r "$PATTERN" . \
    --exclude="*.exe" \
    --exclude="*.dll" \
    --exclude="*.so" \
    --exclude="*.dylib" \
    --exclude="*.test" \
    --exclude="*.db" \
    --exclude="*.sqlite" \
    --exclude="*.sqlite3" \
    --exclude="*.out" \
    --exclude="coverage.*" \
    --exclude="*.coverprofile" \
    --exclude="profile.cov" \
    --exclude="*.prof" \
    --exclude-dir=".git" \
    --exclude-dir="benchmark-results" \
    --exclude-dir=".idea" \
    --exclude-dir="vendor" \
    --exclude-dir="node_modules" \
    --exclude-dir="cache" \
    "$@" 