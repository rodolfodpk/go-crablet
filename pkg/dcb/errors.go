package dcb

import (
	"fmt"
)

type (

	// EventStoreError represents a base error type for event store operations
	EventStoreError struct {
		Op  string // Operation that failed
		Err error  // The underlying error
	}

	// ValidationError represents an error in event or query validation
	ValidationError struct {
		EventStoreError
		Field string // The field that failed validation
		Value string // The invalid value
	}

	// ConcurrencyError represents a concurrency conflict
	ConcurrencyError struct {
		EventStoreError
		ExpectedPosition int64
		ActualPosition   int64
	}

	// ResourceError represents an error related to resource management
	ResourceError struct {
		EventStoreError
		Resource string // The resource that caused the error
	}
)

// Error implements the error interface
func (e EventStoreError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Op, e.Err)
	}
	return e.Op
}

// Unwrap returns the underlying error
func (e EventStoreError) Unwrap() error {
	return e.Err
}
