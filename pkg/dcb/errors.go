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

	// TableStructureError represents an error when a table has incorrect structure
	TableStructureError struct {
		EventStoreError
		TableName    string // The table that has incorrect structure
		ColumnName   string // The specific column with issues (if applicable)
		ExpectedType string // The expected data type (if applicable)
		ActualType   string // The actual data type found (if applicable)
		Issue        string // Description of the specific issue
	}

	// TooManyProjectionsError represents an error when too many projections are running concurrently
	TooManyProjectionsError struct {
		EventStoreError
		MaxConcurrent int // Maximum allowed concurrent projections
		CurrentCount  int // Current number of running projections
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

// =============================================================================
// Error Detection Helpers
// =============================================================================

// IsValidationError checks if the error is a ValidationError
func IsValidationError(err error) bool {
	var validationErr *ValidationError
	return errors.As(err, &validationErr)
}

// IsConcurrencyError checks if the error is a ConcurrencyError
func IsConcurrencyError(err error) bool {
	var concurrencyErr *ConcurrencyError
	return errors.As(err, &concurrencyErr)
}

// IsResourceError checks if the error is a ResourceError
func IsResourceError(err error) bool {
	var resourceErr *ResourceError
	return errors.As(err, &resourceErr)
}

// IsTableStructureError checks if the error is a TableStructureError
func IsTableStructureError(err error) bool {
	var tableStructureErr *TableStructureError
	return errors.As(err, &tableStructureErr)
}

// IsTooManyProjectionsError checks if the error is a TooManyProjectionsError
func IsTooManyProjectionsError(err error) bool {
	var tooManyProjectionsErr *TooManyProjectionsError
	return errors.As(err, &tooManyProjectionsErr)
}

// =============================================================================
// Error Extraction Helpers
// =============================================================================

// GetValidationError extracts a ValidationError from the error chain
func GetValidationError(err error) (*ValidationError, bool) {
	var validationErr *ValidationError
	if errors.As(err, &validationErr) {
		return validationErr, true
	}
	return nil, false
}

// GetConcurrencyError extracts a ConcurrencyError from the error chain
func GetConcurrencyError(err error) (*ConcurrencyError, bool) {
	var concurrencyErr *ConcurrencyError
	if errors.As(err, &concurrencyErr) {
		return concurrencyErr, true
	}
	return nil, false
}

// GetResourceError extracts a ResourceError from the error chain
func GetResourceError(err error) (*ResourceError, bool) {
	var resourceErr *ResourceError
	if errors.As(err, &resourceErr) {
		return resourceErr, true
	}
	return nil, false
}

// GetTableStructureError extracts a TableStructureError from the error chain
func GetTableStructureError(err error) (*TableStructureError, bool) {
	var tableStructureErr *TableStructureError
	if errors.As(err, &tableStructureErr) {
		return tableStructureErr, true
	}
	return nil, false
}

// GetTooManyProjectionsError extracts a TooManyProjectionsError from the error chain
func GetTooManyProjectionsError(err error) (*TooManyProjectionsError, bool) {
	var tooManyProjectionsErr *TooManyProjectionsError
	if errors.As(err, &tooManyProjectionsErr) {
		return tooManyProjectionsErr, true
	}
	return nil, false
}

// =============================================================================
// Error Type Assertion Helpers (Aliases for Get* functions)
// =============================================================================

// AsValidationError is an alias for GetValidationError
func AsValidationError(err error) (*ValidationError, bool) {
	return GetValidationError(err)
}

// AsConcurrencyError is an alias for GetConcurrencyError
func AsConcurrencyError(err error) (*ConcurrencyError, bool) {
	return GetConcurrencyError(err)
}

// AsResourceError is an alias for GetResourceError
func AsResourceError(err error) (*ResourceError, bool) {
	return GetResourceError(err)
}

// AsTableStructureError is an alias for GetTableStructureError
func AsTableStructureError(err error) (*TableStructureError, bool) {
	return GetTableStructureError(err)
}
