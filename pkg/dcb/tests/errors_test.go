package dcb

import (
	"errors"
	"fmt"

	"github.com/rodolfodpk/go-crablet/pkg/dcb"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Event Store: Error Types", func() {
	Describe("EventStoreError", func() {
		It("implements error interface", func() {
			var err error = &dcb.EventStoreError{
				Op:  "test",
				Err: errors.New("test error"),
			}
			Expect(err.Error()).To(Equal("test: test error"))
		})

		It("handles nil underlying error", func() {
			var err error = &dcb.EventStoreError{
				Op: "test",
			}
			Expect(err.Error()).To(Equal("test"))
		})

		It("implements error unwrapping", func() {
			underlying := errors.New("underlying error")
			err := &dcb.EventStoreError{
				Op:  "test",
				Err: underlying,
			}
			Expect(errors.Unwrap(err)).To(Equal(underlying))
		})
	})

	Describe("ValidationError", func() {
		It("includes field and value in error message", func() {
			err := &dcb.ValidationError{
				EventStoreError: dcb.EventStoreError{
					Op:  "validate",
					Err: fmt.Errorf("invalid value"),
				},
				Field: "testField",
				Value: "testValue",
			}
			Expect(err.Error()).To(ContainSubstring("validate"))
			Expect(err.Error()).To(ContainSubstring("invalid value"))
			Expect(err.Field).To(Equal("testField"))
			Expect(err.Value).To(Equal("testValue"))
		})

		It("implements error unwrapping", func() {
			underlying := errors.New("underlying error")
			err := &dcb.ValidationError{
				EventStoreError: dcb.EventStoreError{
					Op:  "validate",
					Err: underlying,
				},
				Field: "testField",
				Value: "testValue",
			}
			Expect(errors.Unwrap(err)).To(Equal(underlying))
		})
	})

	Describe("ConcurrencyError", func() {
		It("includes expected and actual positions in error message", func() {
			err := &dcb.ConcurrencyError{
				EventStoreError: dcb.EventStoreError{
					Op:  "append",
					Err: fmt.Errorf("concurrent modification"),
				},
				ExpectedPosition: 1,
				ActualPosition:   2,
			}
			Expect(err.Error()).To(ContainSubstring("append"))
			Expect(err.Error()).To(ContainSubstring("concurrent modification"))
			Expect(err.ExpectedPosition).To(Equal(int64(1)))
			Expect(err.ActualPosition).To(Equal(int64(2)))
		})

		It("implements error unwrapping", func() {
			underlying := errors.New("underlying error")
			err := &dcb.ConcurrencyError{
				EventStoreError: dcb.EventStoreError{
					Op:  "append",
					Err: underlying,
				},
				ExpectedPosition: 1,
				ActualPosition:   2,
			}
			Expect(errors.Unwrap(err)).To(Equal(underlying))
		})
	})

	Describe("ResourceError", func() {
		It("includes resource name in error message", func() {
			err := &dcb.ResourceError{
				EventStoreError: dcb.EventStoreError{
					Op:  "connect",
					Err: fmt.Errorf("connection failed"),
				},
				Resource: "database",
			}
			Expect(err.Error()).To(ContainSubstring("connect"))
			Expect(err.Error()).To(ContainSubstring("connection failed"))
			Expect(err.Resource).To(Equal("database"))
		})

		It("implements error unwrapping", func() {
			underlying := errors.New("underlying error")
			err := &dcb.ResourceError{
				EventStoreError: dcb.EventStoreError{
					Op:  "connect",
					Err: underlying,
				},
				Resource: "database",
			}
			Expect(errors.Unwrap(err)).To(Equal(underlying))
		})
	})

	Describe("Error Type Assertions", func() {
		It("allows type assertions for specific error types", func() {
			// Create a validation error
			validationErr := &dcb.ValidationError{
				EventStoreError: dcb.EventStoreError{
					Op:  "validate",
					Err: fmt.Errorf("invalid value"),
				},
				Field: "testField",
				Value: "testValue",
			}

			// Test type assertion
			var err error = validationErr
			_, ok := err.(*dcb.ValidationError)
			Expect(ok).To(BeTrue())
		})
	})
})
