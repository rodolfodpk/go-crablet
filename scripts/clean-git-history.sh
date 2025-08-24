#!/bin/bash

# Clean Git History Script
# This script removes large files and binary artifacts from git history

set -e

echo "🧹 Starting Git History Cleanup..."

# Create a backup of the current state
echo "📦 Creating backup branch..."
git checkout -b backup-before-cleanup
git push origin backup-before-cleanup

# Switch back to main
git checkout main

# Remove large files from git history
echo "🗑️  Removing large files from git history..."

# Remove files larger than 1MB from git history
git filter-repo \
    --path-glob '*.exe' \
    --path-glob '*.dll' \
    --path-glob '*.so' \
    --path-glob '*.dylib' \
    --path-glob '*.test' \
    --path-glob '*.out' \
    --path-glob '*.db' \
    --path-glob '*.sqlite' \
    --path-glob 'web-app' \
    --path-glob 'main' \
    --path-glob 'enrollment' \
    --path-glob 'ticket_booking' \
    --path-glob 'performance.test' \
    --path-glob 'benchmark-results/' \
    --path-glob 'internal/web-app/' \
    --path-glob 'internal/benchmarks/cache/' \
    --path-glob 'internal/benchmarks/tools/cache/' \
    --invert-paths \
    --force

echo "✅ Git history cleaned successfully!"

# Show repository size improvement
echo "📊 Repository size after cleanup:"
du -sh .git

echo "🚀 Cleanup complete! You can now push the cleaned history:"
echo "   git push origin main --force"
echo ""
echo "⚠️  Note: This rewrites history. All collaborators will need to re-clone."
echo "📦 Backup branches created: backup-with-binaries, backup-before-cleanup"
