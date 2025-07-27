package dcb

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	dcb "github.com/rodolfodpk/go-crablet/pkg/dcb"
)

var _ = Describe("Error Handling Helpers", func() {
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()
		// Clean DB state before each test
		Expect(truncateEventsTable(ctx, pool)).To(Succeed())
	})

	It("detects and extracts ValidationError", func() {
		invalidEvent := dcb.NewInputEvent("", dcb.NewTags("user_id", "123"), dcb.ToJSON(map[string]string{"name": "John"}))
		events := []dcb.InputEvent{invalidEvent}
		err := store.Append(ctx, events)

		Expect(dcb.IsValidationError(err)).To(BeTrue())
		validationErr, ok := dcb.GetValidationError(err)
		Expect(ok).To(BeTrue())
		Expect(validationErr.Field).To(Equal("type"))
	})

	It("detects and extracts ConcurrencyError", func() {
		event := dcb.NewInputEvent("UserRegistered", dcb.NewTags("user_id", "456"), dcb.ToJSON(map[string]string{"name": "Jane"}))
		events := []dcb.InputEvent{event}
		// First append should succeed
		err := store.Append(ctx, events)
		Expect(err).NotTo(HaveOccurred())

		// Second append with a condition that will fail
		query := dcb.NewQuery(dcb.NewTags("user_id", "456"), "UserRegistered")
		condition := dcb.NewAppendCondition(query)
		err = store.AppendIf(ctx, events, condition)

		Expect(dcb.IsConcurrencyError(err)).To(BeTrue())
		concurrencyErr, ok := dcb.GetConcurrencyError(err)
		Expect(ok).To(BeTrue())
		Expect(concurrencyErr).NotTo(BeNil())
	})

	It("detects and extracts ResourceError (simulated)", func() {
		// Simulate a resource error by creating a ResourceError directly
		err := &dcb.ResourceError{
			EventStoreError: dcb.EventStoreError{
				Op:  "connect",
				Err: fmt.Errorf("connection failed"),
			},
			Resource: "database",
		}
		Expect(dcb.IsResourceError(err)).To(BeTrue())
		resourceErr, ok := dcb.GetResourceError(err)
		Expect(ok).To(BeTrue())
		Expect(resourceErr.Resource).To(Equal("database"))
	})

	It("uses error type assertion helpers", func() {
		validationErr := &dcb.ValidationError{
			EventStoreError: dcb.EventStoreError{
				Op:  "validate",
				Err: fmt.Errorf("invalid value"),
			},
			Field: "testField",
			Value: "testValue",
		}
		extracted, ok := dcb.AsValidationError(validationErr)
		Expect(ok).To(BeTrue())
		Expect(extracted).To(Equal(validationErr))
	})
})
