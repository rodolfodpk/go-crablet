// Package setup provides dataset and benchmark data management for benchmarks.
// This file ensures the package is properly built and initialized.
package setup

import (
	_ "github.com/mattn/go-sqlite3"
)

// Ensure package is built and dependencies are resolved
var _ = struct{}{}

// Package initialization
func init() {
	// This ensures the package is properly initialized
}
