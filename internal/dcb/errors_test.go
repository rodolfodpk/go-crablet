package dcb

import (
	"errors"
	"testing"
)

func TestEventStoreError(t *testing.T) {
	tests := []struct {
		name     string
		err      EventStoreError
		expected string
	}{
		{
			name: "with underlying error",
			err: EventStoreError{
				Op:  "read",
				Err: errors.New("connection failed"),
			},
			expected: "read: connection failed",
		},
		{
			name: "without underlying error",
			err: EventStoreError{
				Op: "write",
			},
			expected: "write",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.expected {
				t.Errorf("EventStoreError.Error() = %v, want %v", got, tt.expected)
			}

			if tt.err.Err != nil {
				if got := tt.err.Unwrap(); got != tt.err.Err {
					t.Errorf("EventStoreError.Unwrap() = %v, want %v", got, tt.err.Err)
				}
			}
		})
	}
}

func TestValidationError(t *testing.T) {
	err := ValidationError{
		EventStoreError: EventStoreError{
			Op:  "validate",
			Err: errors.New("invalid input"),
		},
		Field: "name",
		Value: "123",
	}

	expected := "validate: invalid input"
	if got := err.Error(); got != expected {
		t.Errorf("ValidationError.Error() = %v, want %v", got, expected)
	}

	// Test that ValidationError embeds EventStoreError correctly
	if err.Field != "name" {
		t.Errorf("ValidationError.Field = %v, want %v", err.Field, "name")
	}
	if err.Value != "123" {
		t.Errorf("ValidationError.Value = %v, want %v", err.Value, "123")
	}
}

func TestConcurrencyError(t *testing.T) {
	err := ConcurrencyError{
		EventStoreError: EventStoreError{
			Op:  "append",
			Err: errors.New("version mismatch"),
		},
		ExpectedPosition: 5,
		ActualPosition:   6,
	}

	expected := "append: version mismatch"
	if got := err.Error(); got != expected {
		t.Errorf("ConcurrencyError.Error() = %v, want %v", got, expected)
	}

	// Test that ConcurrencyError embeds EventStoreError correctly
	if err.ExpectedPosition != 5 {
		t.Errorf("ConcurrencyError.ExpectedPosition = %v, want %v", err.ExpectedPosition, 5)
	}
	if err.ActualPosition != 6 {
		t.Errorf("ConcurrencyError.ActualPosition = %v, want %v", err.ActualPosition, 6)
	}
}

func TestResourceError(t *testing.T) {
	err := ResourceError{
		EventStoreError: EventStoreError{
			Op:  "allocate",
			Err: errors.New("insufficient resources"),
		},
		Resource: "memory",
	}

	expected := "allocate: insufficient resources"
	if got := err.Error(); got != expected {
		t.Errorf("ResourceError.Error() = %v, want %v", got, expected)
	}

	// Test that ResourceError embeds EventStoreError correctly
	if err.Resource != "memory" {
		t.Errorf("ResourceError.Resource = %v, want %v", err.Resource, "memory")
	}
}

func TestErrorUnwrapping(t *testing.T) {
	baseErr := errors.New("base error")
	storeErr := EventStoreError{
		Op:  "operation",
		Err: baseErr,
	}

	// Test that errors.Is works correctly
	if !errors.Is(storeErr, baseErr) {
		t.Error("errors.Is(storeErr, baseErr) = false, want true")
	}

	// Test that errors.As works correctly
	var target EventStoreError
	if !errors.As(storeErr, &target) {
		t.Error("errors.As(storeErr, &target) = false, want true")
	}
}
