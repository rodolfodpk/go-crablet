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

	Describe("ProjectDecisionModel with cursor streaming", func() {
		It("should use cursor-based streaming when BatchSize is specified", func() {
			// Create test events
			events := []InputEvent{
				{Type: "CourseCreated", Tags: []Tag{{Key: "course_id", Value: "course-1"}}, Data: []byte(`{"name":"Math 101"}`)},
				{Type: "StudentEnrolled", Tags: []Tag{{Key: "course_id", Value: "course-1"}, {Key: "student_id", Value: "student-1"}}, Data: []byte(`{"student_name":"Alice"}`)},
				{Type: "StudentEnrolled", Tags: []Tag{{Key: "course_id", Value: "course-1"}, {Key: "student_id", Value: "student-2"}}, Data: []byte(`{"student_name":"Bob"}`)},
			}

			// Append events
			_, err := store.Append(ctx, events, nil)
			Expect(err).To(BeNil())

			// Define projectors
			projectors := []BatchProjector{
				{
					ID: "course_enrollment",
					StateProjector: StateProjector{
						Query: Query{
							Items: []QueryItem{
								{
									EventTypes: []string{"CourseCreated", "StudentEnrolled"},
									Tags:       []Tag{{Key: "course_id", Value: "course-1"}},
								},
							},
						},
						InitialState: 0,
						TransitionFn: func(state any, event Event) any {
							count := state.(int)
							if event.Type == "StudentEnrolled" {
								return count + 1
							}
							return count
						},
					},
				},
			}

			// Query for events
			query := Query{
				Items: []QueryItem{
					{
						EventTypes: []string{"CourseCreated", "StudentEnrolled"},
						Tags:       []Tag{{Key: "course_id", Value: "course-1"}},
					},
				},
			}

			// Use cursor-based streaming with small batch size
			batchSize := 2
			options := &ReadOptions{BatchSize: &batchSize}

			// Test ProjectDecisionModel with cursor streaming
			states, appendCondition, err := store.ProjectDecisionModel(ctx, query, options, projectors)
			Expect(err).To(BeNil())
			Expect(states).To(HaveKey("course_enrollment"))
			Expect(states["course_enrollment"]).To(Equal(2)) // 2 StudentEnrolled events
			Expect(appendCondition.After).To(Not(BeNil()))
			Expect(*appendCondition.After).To(Equal(int64(3))) // Last event position
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
