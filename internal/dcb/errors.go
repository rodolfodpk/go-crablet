package dcb

import (
	"errors"
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

// IsValidationError checks if an error is a ValidationError
func IsValidationError(err error) bool {
	var ve *ValidationError
	return errors.As(err, &ve)
}

// IsConcurrencyError checks if an error is a ConcurrencyError
func IsConcurrencyError(err error) bool {
	var ce *ConcurrencyError
	return errors.As(err, &ce)
}

// IsResourceError checks if an error is a ResourceError
func IsResourceError(err error) bool {
	var re *ResourceError
	return errors.As(err, &re)
}
