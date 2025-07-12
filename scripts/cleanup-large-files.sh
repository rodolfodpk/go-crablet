#!/bin/bash

# Script to clean up large binary files from git history
# This will rewrite git history to remove large files

echo "Cleaning up large binary files from git history..."

# Create a backup branch first
git branch backup-before-cleanup

# Use git filter-repo to remove large files from history
# This will rewrite the entire git history
git filter-repo \
    --path benchmarks \
    --path server \
    --path web-app \
    --path command_execution \
    --path streaming \
    --path transfer \
    --path internal/web-app/web-app \
    --path internal/examples/*/enrollment \
    --path internal/examples/*/streaming \
    --path internal/examples/*/transfer \
    --path batch \
    --path channel_projection \
    --path decision_model \
    --path enrollment \
    --invert-paths \
    --force

echo "Large files cleaned up from git history!"
echo "Note: This rewrote git history. You may need to force push:"
echo "git push --force-with-lease origin refactor/stateprojector-id" 