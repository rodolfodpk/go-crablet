package dcb

import (
	"encoding/json"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Streaming Projection", func() {
	BeforeEach(func() {
		err := truncateEventsTable(ctx, pool)
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("ProjectDecisionModel", func() {
		It("should project multiple states and return append condition", func() {
			// Setup test data
			events := []InputEvent{
				{Type: "AccountCreated", Tags: []Tag{{Key: "account_id", Value: "acc123"}}, Data: []byte(`{"balance": 1000}`)},
				{Type: "TransactionCompleted", Tags: []Tag{{Key: "account_id", Value: "acc123"}}, Data: []byte(`{"amount": 500}`)},
				{Type: "TransactionCompleted", Tags: []Tag{{Key: "account_id", Value: "acc123"}}, Data: []byte(`{"amount": 300}`)},
			}

			// Append events
			_, err := store.Append(ctx, events, nil)
			Expect(err).NotTo(HaveOccurred())

			// Define projectors
			projectors := []BatchProjector{
				{
					ID: "account",
					StateProjector: StateProjector{
						Query:        Query{Items: []QueryItem{{EventTypes: []string{"AccountCreated", "TransactionCompleted"}}}},
						InitialState: &AccountState{Balance: 0},
						TransitionFn: func(state any, event Event) any {
							account := state.(*AccountState)
							switch event.Type {
							case "AccountCreated":
								var data AccountCreatedData
								json.Unmarshal(event.Data, &data)
								account.Balance = int64(data.Balance)
							case "TransactionCompleted":
								var data TransactionCompletedData
								json.Unmarshal(event.Data, &data)
								account.Balance += int64(data.Amount)
							}
							return account
						},
					},
				},
			}

			// Test ProjectDecisionModel
			query := Query{Items: []QueryItem{{EventTypes: []string{"AccountCreated", "TransactionCompleted"}}}}
			states, appendCondition, err := store.ProjectDecisionModel(ctx, query, nil, projectors)
			Expect(err).NotTo(HaveOccurred())

			// Verify results
			Expect(states).NotTo(BeNil())
			Expect(appendCondition.After).NotTo(BeNil())
			Expect(*appendCondition.After).To(Equal(int64(3)))

			account, ok := states["account"].(*AccountState)
			Expect(ok).To(BeTrue())
			Expect(account.Balance).To(Equal(int64(1800))) // 1000 + 500 + 300
		})
	})

	Describe("ReadStream", func() {
		It("should return a pure event iterator", func() {
			// Setup test data
			events := []InputEvent{
				{Type: "AccountCreated", Tags: []Tag{{Key: "account_id", Value: "acc123"}}, Data: []byte(`{"balance": 1000}`)},
				{Type: "TransactionCompleted", Tags: []Tag{{Key: "account_id", Value: "acc123"}}, Data: []byte(`{"amount": 500}`)},
			}

			// Append events
			_, err := store.Append(ctx, events, nil)
			Expect(err).NotTo(HaveOccurred())

			// Test ReadStream (pure event iterator)
			query := Query{Items: []QueryItem{{EventTypes: []string{"AccountCreated", "TransactionCompleted"}}}}
			iterator, err := store.ReadStream(ctx, query, nil)
			Expect(err).NotTo(HaveOccurred())
			defer iterator.Close()

			// Process events
			var processedEvents []Event
			for iterator.Next() {
				event := iterator.Event()
				processedEvents = append(processedEvents, event)
			}

			// Verify results
			Expect(iterator.Err()).NotTo(HaveOccurred())
			Expect(processedEvents).To(HaveLen(2))
			Expect(processedEvents[0].Type).To(Equal("AccountCreated"))
			Expect(processedEvents[1].Type).To(Equal("TransactionCompleted"))
		})
	})
})

// Helper types for testing
type AccountState struct {
	ID      string
	Balance int64
}

type TransactionState struct {
	Count       int
	TotalAmount int
}

type BalanceState struct {
	CurrentBalance int
	LastUpdated    time.Time
}

type AccountCreatedData struct {
	Balance int `json:"balance"`
}

type AccountUpdatedData struct {
	NewBalance int `json:"new_balance"`
}

type TransactionCompletedData struct {
	Amount int `json:"amount"`
}

func mustMarshal(v any) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}
