#!/bin/bash

# Fix all BatchProjector and nested StateProjector issues in examples and benchmarks

echo "Fixing examples and benchmarks..."

# Function to fix a file
fix_file() {
    local file="$1"
    echo "Fixing $file"
    
    # Replace BatchProjector with StateProjector
    sed -i '' 's/\[\]dcb\.BatchProjector/[]dcb.StateProjector/g' "$file"
    
    # Fix nested StateProjector constructions
    # Pattern: {ID: "name", StateProjector: dcb.StateProjector{...}}
    # Replace with: {ID: "name", ...}
    sed -i '' 's/{ID: "\([^"]*\)", StateProjector: dcb\.StateProjector{/{ID: "\1", {/g' "$file"
    
    # Remove trailing }, from the end of StateProjector structs
    sed -i '' 's/},$//g' "$file"
    
    # Fix dcb.BatchProjector{ to dcb.StateProjector{
    sed -i '' 's/dcb\.BatchProjector{/dcb.StateProjector{/g' "$file"
}

# Fix all example files
fix_file "internal/examples/batch/main.go"
fix_file "internal/examples/decision_model/main.go"
fix_file "internal/examples/enrollment/main.go"
fix_file "internal/examples/transfer/main.go"
fix_file "internal/benchmarks/main.go"
fix_file "internal/benchmarks/setup/projectors.go"

echo "Done fixing examples and benchmarks." 