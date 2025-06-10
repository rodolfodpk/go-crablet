// Package dcb provides domain-specific types and helpers for the account domain.
package dcb

import (
	"context"
	"encoding/json"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// AccountRegistered represents the event data for account registration
type AccountRegistered struct {
	Username string `json:"username"`
}

// AccountBalance represents the event data for account balance changes
type AccountBalance struct {
	Amount float64 `json:"amount"`
}

// MoneyTransferred represents the event data for money transfers between accounts
type MoneyTransferred struct {
	FromUsername string  `json:"fromUsername"`
	ToUsername   string  `json:"toUsername"`
	Amount       float64 `json:"amount"`
	FromBalance  float64 `json:"fromBalance"` // New sender balance after transfer
	ToBalance    float64 `json:"toBalance"`   // New receiver balance after transfer
}

// NewAccountRegistered creates a new account registration event
func NewAccountRegistered(username string) InputEvent {
	data, _ := json.Marshal(AccountRegistered{Username: username})
	return NewInputEvent(
		"AccountRegistered",
		NewTags("username", username),
		data,
	)
}

// NewAccountBalanceEvent creates a new account balance event
func NewAccountBalanceEvent(username string, amount float64) InputEvent {
	data, _ := json.Marshal(AccountBalance{Amount: amount})
	return NewInputEvent(
		"AccountBalance",
		NewTags("username", username),
		data,
	)
}

// NewMoneyTransferredEvent creates a new money transfer event
func NewMoneyTransferredEvent(fromUsername, toUsername string, amount, fromBalance, toBalance float64) InputEvent {
	data, _ := json.Marshal(MoneyTransferred{
		FromUsername: fromUsername,
		ToUsername:   toUsername,
		Amount:       amount,
		FromBalance:  fromBalance,
		ToBalance:    toBalance,
	})
	return NewInputEvent(
		"MoneyTransferred",
		NewTags("username", fromUsername, "username", toUsername),
		data,
	)
}

// IsUsernameClaimedProjection creates a projector to check if a username is claimed
func IsUsernameClaimedProjection(username string) StateProjector {
	return StateProjector{
		Query: NewLegacyQuery(
			NewTags("username", username),
			[]string{"AccountRegistered"},
		),
		InitialState: false,
		TransitionFn: func(state any, event Event) any {
			return true
		},
	}
}

// AccountBalanceProjection creates a projector for account balance
func AccountBalanceProjection(username string) StateProjector {
	return StateProjector{
		Query: NewLegacyQuery(
			NewTags("username", username),
			[]string{"AccountBalance", "MoneyTransferred"},
		),
		InitialState: 0.0,
		TransitionFn: func(state any, event Event) any {
			balance := state.(float64)
			switch event.Type {
			case "AccountBalance":
				var data AccountBalance
				if err := json.Unmarshal(event.Data, &data); err != nil {
					return balance
				}
				return data.Amount
			case "MoneyTransferred":
				var data MoneyTransferred
				if err := json.Unmarshal(event.Data, &data); err != nil {
					return balance
				}
				if data.FromUsername == username {
					return data.FromBalance
				}
				if data.ToUsername == username {
					return data.ToBalance
				}
			}
			return balance
		},
	}
}

// AccountAPI handles account registration commands
type AccountAPI struct {
	eventStore EventStore
}

// NewAccountAPI creates a new account API instance
func NewAccountAPI(eventStore EventStore) (*AccountAPI, error) {
	if eventStore == nil {
		return nil, fmt.Errorf("event store must not be nil")
	}
	return &AccountAPI{eventStore: eventStore}, nil
}

// RegisterAccount attempts to register a new account
func (a *AccountAPI) RegisterAccount(ctx context.Context, username string) error {
	projector := IsUsernameClaimedProjection(username)
	_, state, err := a.eventStore.ProjectState(ctx, projector)
	if err != nil {
		return fmt.Errorf("failed to check username: %w", err)
	}

	if state.(bool) {
		return fmt.Errorf("username %q is claimed", username)
	}

	event := NewAccountRegistered(username)
	_, err = a.eventStore.AppendEvents(ctx, []InputEvent{event}, projector.Query, 0)
	if err != nil {
		return fmt.Errorf("failed to register account: %w", err)
	}

	return nil
}

// TransferMoneyCommand represents the command to transfer money between accounts
type TransferMoneyCommand struct {
	FromUsername string  `json:"fromUsername"`
	ToUsername   string  `json:"toUsername"`
	Amount       float64 `json:"amount"`
}

// TransferMoney attempts to transfer money between accounts
func (a *AccountAPI) TransferMoney(ctx context.Context, cmd TransferMoneyCommand) error {
	// Check if both accounts exist
	fromProjector := IsUsernameClaimedProjection(cmd.FromUsername)
	toProjector := IsUsernameClaimedProjection(cmd.ToUsername)

	_, fromState, err := a.eventStore.ProjectState(ctx, fromProjector)
	if err != nil {
		return fmt.Errorf("failed to check from account: %w", err)
	}
	if !fromState.(bool) {
		return fmt.Errorf("from account %q does not exist", cmd.FromUsername)
	}

	_, toState, err := a.eventStore.ProjectState(ctx, toProjector)
	if err != nil {
		return fmt.Errorf("failed to check to account: %w", err)
	}
	if !toState.(bool) {
		return fmt.Errorf("to account %q does not exist", cmd.ToUsername)
	}

	// Check if from account has sufficient balance
	balanceProjector := AccountBalanceProjection(cmd.FromUsername)
	_, balance, err := a.eventStore.ProjectState(ctx, balanceProjector)
	if err != nil {
		return fmt.Errorf("failed to check account balance: %w", err)
	}

	if balance.(float64) < cmd.Amount {
		return fmt.Errorf("insufficient balance in account %q", cmd.FromUsername)
	}

	// Get current balances for both accounts
	fromBalanceProjector := AccountBalanceProjection(cmd.FromUsername)
	toBalanceProjector := AccountBalanceProjection(cmd.ToUsername)

	_, fromBalance, err := a.eventStore.ProjectState(ctx, fromBalanceProjector)
	if err != nil {
		return fmt.Errorf("failed to get from account balance: %w", err)
	}

	_, toBalance, err := a.eventStore.ProjectState(ctx, toBalanceProjector)
	if err != nil {
		return fmt.Errorf("failed to get to account balance: %w", err)
	}

	// Calculate new balances
	fromNewBalance := fromBalance.(float64) - cmd.Amount
	toNewBalance := toBalance.(float64) + cmd.Amount

	// Create balance update events for both accounts
	fromEvent := NewAccountBalanceEvent(cmd.FromUsername, fromNewBalance)
	toEvent := NewAccountBalanceEvent(cmd.ToUsername, toNewBalance)

	// Create the transfer event with both accounts' balance information
	transferEvent := NewMoneyTransferredEvent(cmd.FromUsername, cmd.ToUsername, cmd.Amount, fromNewBalance, toNewBalance)

	// Get current positions for both accounts' streams
	combinedQuery := NewLegacyQuery(
		[]Tag{{Key: "username", Value: cmd.FromUsername}, {Key: "username", Value: cmd.ToUsername}},
		[]string{"AccountBalance", "MoneyTransferred"},
	)

	// Get position for the combined stream
	pos, _, err := a.eventStore.ProjectState(ctx, StateProjector{
		Query:        combinedQuery,
		InitialState: []Event{},
		TransitionFn: func(state any, event Event) any {
			events := state.([]Event)
			return append(events, event)
		},
	})
	if err != nil {
		return fmt.Errorf("failed to get stream position: %w", err)
	}

	// Append all events in a single transaction using the combined query
	_, err = a.eventStore.AppendEvents(ctx, []InputEvent{fromEvent, toEvent, transferEvent}, combinedQuery, pos)
	if err != nil {
		return fmt.Errorf("failed to append transfer events: %w", err)
	}

	return nil
}

// SetAccountBalance sets the initial balance for an account
func (a *AccountAPI) SetAccountBalance(ctx context.Context, username string, amount float64) error {
	// Check if account exists
	projector := IsUsernameClaimedProjection(username)
	_, state, err := a.eventStore.ProjectState(ctx, projector)
	if err != nil {
		return fmt.Errorf("failed to check account: %w", err)
	}
	if !state.(bool) {
		return fmt.Errorf("account %q does not exist", username)
	}

	// Get current stream position
	query := NewQuery(NewTags("username", username))
	pos, _, err := a.eventStore.ProjectState(ctx, StateProjector{
		Query:        query,
		InitialState: []Event{},
		TransitionFn: func(state any, event Event) any {
			events := state.([]Event)
			return append(events, event)
		},
	})
	if err != nil {
		return fmt.Errorf("failed to get stream position: %w", err)
	}

	// Create and append the balance event
	event := NewAccountBalanceEvent(username, amount)
	_, err = a.eventStore.AppendEvents(ctx, []InputEvent{event}, query, pos)
	if err != nil {
		return fmt.Errorf("failed to set account balance: %w", err)
	}

	return nil
}

var _ = Describe("Account Registration", func() {
	var (
		api       *AccountAPI
		username  string
		query     Query
		projector StateProjector
	)

	BeforeEach(func() {
		// Truncate the events table before each test
		err := truncateEventsTable(ctx, pool)
		Expect(err).NotTo(HaveOccurred())

		var apiErr error
		api, apiErr = NewAccountAPI(store)
		Expect(apiErr).NotTo(HaveOccurred(), "Failed to create AccountAPI")
		Expect(api).NotTo(BeNil(), "AccountAPI should not be nil")

		username = "testuser"
		query = NewQuery(NewTags("username", username))
		projector = IsUsernameClaimedProjection(username)
	})

	Context("Given no existing accounts", func() {
		When("registering a new account", func() {
			It("should succeed and create an AccountRegistered event", func() {
				// When
				err := api.RegisterAccount(ctx, username)

				// Then
				Expect(err).NotTo(HaveOccurred())

				// Verify the event was created
				_, state, err := store.ProjectState(ctx, projector)
				Expect(err).NotTo(HaveOccurred())
				Expect(state).To(BeTrue())

				// Verify event details
				_, eventsResult, err := store.ProjectState(ctx, StateProjector{
					Query:        query,
					InitialState: []Event{},
					TransitionFn: func(state any, event Event) any {
						events := state.([]Event)
						return append(events, event)
					},
				})
				Expect(err).NotTo(HaveOccurred())
				events := eventsResult.([]Event)
				Expect(events).To(HaveLen(1))
				Expect(events[0].Type).To(Equal("AccountRegistered"))

				var eventData AccountRegistered
				err = json.Unmarshal(events[0].Data, &eventData)
				Expect(err).NotTo(HaveOccurred())
				Expect(eventData.Username).To(Equal(username))
			})
		})
	})

	Context("Given an existing account", func() {
		BeforeEach(func() {
			// Given
			event := NewAccountRegistered(username)
			_, err := store.AppendEvents(ctx, []InputEvent{event}, query, 0)
			Expect(err).NotTo(HaveOccurred())
		})

		When("attempting to register the same username", func() {
			It("should fail with username claimed error", func() {
				// When
				err := api.RegisterAccount(ctx, username)

				// Then
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("username \"testuser\" is claimed"))

				// Verify no new events were created
				_, eventsResult, err := store.ProjectState(ctx, StateProjector{
					Query:        query,
					InitialState: []Event{},
					TransitionFn: func(state any, event Event) any {
						events := state.([]Event)
						return append(events, event)
					},
				})
				Expect(err).NotTo(HaveOccurred())
				events := eventsResult.([]Event)
				Expect(events).To(HaveLen(1))
			})
		})

		When("registering a different username", func() {
			It("should succeed", func() {
				// When
				newUsername := "differentuser"
				err := api.RegisterAccount(ctx, newUsername)

				// Then
				Expect(err).NotTo(HaveOccurred())

				// Verify both usernames are claimed
				_, state1, err := store.ProjectState(ctx, IsUsernameClaimedProjection(username))
				Expect(err).NotTo(HaveOccurred())
				Expect(state1).To(BeTrue())

				_, state2, err := store.ProjectState(ctx, IsUsernameClaimedProjection(newUsername))
				Expect(err).NotTo(HaveOccurred())
				Expect(state2).To(BeTrue())
			})
		})
	})

	Context("Given multiple registration attempts", func() {
		When("registering the same username concurrently", func() {
			It("should ensure only one registration succeeds", func() {
				// Given
				username1 := "concurrentuser"
				username2 := "concurrentuser" // Same username

				// When
				err1 := api.RegisterAccount(ctx, username1)
				err2 := api.RegisterAccount(ctx, username2)

				// Then
				Expect(err1 == nil || err2 == nil).To(BeTrue(), "exactly one registration should succeed")
				Expect(err1 == nil && err2 == nil).To(BeFalse(), "not both registrations should succeed")

				// Verify only one event exists
				_, eventsResult, err := store.ProjectState(ctx, StateProjector{
					Query: NewLegacyQuery(
						[]Tag{{Key: "username", Value: "concurrentuser"}},
						[]string{"AccountRegistered"},
					),
					InitialState: []Event{},
					TransitionFn: func(state any, event Event) any {
						events := state.([]Event)
						return append(events, event)
					},
				})
				Expect(err).NotTo(HaveOccurred())
				events := eventsResult.([]Event)
				Expect(events).To(HaveLen(1))
			})
		})
	})
})

var _ = Describe("Account Money Transfer", func() {
	var (
		api           *AccountAPI
		fromUsername  string
		toUsername    string
		fromProjector StateProjector
		toProjector   StateProjector
	)

	BeforeEach(func() {
		// Truncate the events table before each test
		err := truncateEventsTable(ctx, pool)
		Expect(err).NotTo(HaveOccurred())

		var apiErr error
		api, apiErr = NewAccountAPI(store)
		Expect(apiErr).NotTo(HaveOccurred(), "Failed to create AccountAPI")
		Expect(api).NotTo(BeNil(), "AccountAPI should not be nil")

		fromUsername = "sender"
		toUsername = "receiver"
		fromProjector = AccountBalanceProjection(fromUsername)
		toProjector = AccountBalanceProjection(toUsername)

		// Register both accounts
		err = api.RegisterAccount(ctx, fromUsername)
		Expect(err).NotTo(HaveOccurred())
		err = api.RegisterAccount(ctx, toUsername)
		Expect(err).NotTo(HaveOccurred())
	})

	Context("Given accounts with sufficient balance", func() {
		BeforeEach(func() {
			// Set initial balances
			err := api.SetAccountBalance(ctx, fromUsername, 100.0)
			Expect(err).NotTo(HaveOccurred())
			err = api.SetAccountBalance(ctx, toUsername, 50.0)
			Expect(err).NotTo(HaveOccurred())
		})

		When("transferring money between accounts", func() {
			It("should succeed and update both account balances", func() {
				// When
				err := api.TransferMoney(ctx, TransferMoneyCommand{
					FromUsername: fromUsername,
					ToUsername:   toUsername,
					Amount:       30.0,
				})

				// Then
				Expect(err).NotTo(HaveOccurred())

				// Verify sender's balance
				_, fromBalance, err := store.ProjectState(ctx, fromProjector)
				Expect(err).NotTo(HaveOccurred())
				Expect(fromBalance).To(Equal(70.0))

				// Verify receiver's balance
				_, toBalance, err := store.ProjectState(ctx, toProjector)
				Expect(err).NotTo(HaveOccurred())
				Expect(toBalance).To(Equal(80.0))

				// Verify transfer event was created
				_, eventsResult, err := store.ProjectState(ctx, StateProjector{
					Query: NewLegacyQuery(
						[]Tag{{Key: "username", Value: fromUsername}, {Key: "username", Value: toUsername}},
						[]string{"MoneyTransferred"},
					),
					InitialState: []Event{},
					TransitionFn: func(state any, event Event) any {
						events := state.([]Event)
						return append(events, event)
					},
				})
				Expect(err).NotTo(HaveOccurred())
				events := eventsResult.([]Event)
				Expect(events).To(HaveLen(1))
				Expect(events[0].Type).To(Equal("MoneyTransferred"))

				var eventData MoneyTransferred
				err = json.Unmarshal(events[0].Data, &eventData)
				Expect(err).NotTo(HaveOccurred())
				Expect(eventData.FromUsername).To(Equal(fromUsername))
				Expect(eventData.ToUsername).To(Equal(toUsername))
				Expect(eventData.Amount).To(Equal(30.0))
			})
		})

		When("attempting to transfer more than available balance", func() {
			It("should fail with insufficient balance error", func() {
				// When
				err := api.TransferMoney(ctx, TransferMoneyCommand{
					FromUsername: fromUsername,
					ToUsername:   toUsername,
					Amount:       150.0,
				})

				// Then
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("insufficient balance"))

				// Verify balances remain unchanged
				_, fromBalance, err := store.ProjectState(ctx, fromProjector)
				Expect(err).NotTo(HaveOccurred())
				Expect(fromBalance).To(Equal(100.0))

				_, toBalance, err := store.ProjectState(ctx, toProjector)
				Expect(err).NotTo(HaveOccurred())
				Expect(toBalance).To(Equal(50.0))
			})
		})
	})

	Context("Given non-existent accounts", func() {
		When("attempting to transfer from non-existent account", func() {
			It("should fail with account does not exist error", func() {
				// When
				err := api.TransferMoney(ctx, TransferMoneyCommand{
					FromUsername: "nonexistent",
					ToUsername:   toUsername,
					Amount:       30.0,
				})

				// Then
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("does not exist"))
			})
		})

		When("attempting to transfer to non-existent account", func() {
			It("should fail with account does not exist error", func() {
				// When
				err := api.TransferMoney(ctx, TransferMoneyCommand{
					FromUsername: fromUsername,
					ToUsername:   "nonexistent",
					Amount:       30.0,
				})

				// Then
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("does not exist"))
			})
		})
	})

	Context("Given multiple transfers", func() {
		BeforeEach(func() {
			// Set initial balances
			err := api.SetAccountBalance(ctx, fromUsername, 100.0)
			Expect(err).NotTo(HaveOccurred())
			err = api.SetAccountBalance(ctx, toUsername, 50.0)
			Expect(err).NotTo(HaveOccurred())
		})

		When("performing multiple transfers", func() {
			It("should maintain correct balances after multiple transfers", func() {
				// When
				err := api.TransferMoney(ctx, TransferMoneyCommand{
					FromUsername: fromUsername,
					ToUsername:   toUsername,
					Amount:       20.0,
				})
				Expect(err).NotTo(HaveOccurred())

				err = api.TransferMoney(ctx, TransferMoneyCommand{
					FromUsername: toUsername,
					ToUsername:   fromUsername,
					Amount:       10.0,
				})
				Expect(err).NotTo(HaveOccurred())

				err = api.TransferMoney(ctx, TransferMoneyCommand{
					FromUsername: fromUsername,
					ToUsername:   toUsername,
					Amount:       15.0,
				})
				Expect(err).NotTo(HaveOccurred())

				// Then
				// Verify final balances
				_, fromBalance, err := store.ProjectState(ctx, fromProjector)
				Expect(err).NotTo(HaveOccurred())
				Expect(fromBalance).To(Equal(75.0)) // 100 - 20 + 10 - 15

				_, toBalance, err := store.ProjectState(ctx, toProjector)
				Expect(err).NotTo(HaveOccurred())
				Expect(toBalance).To(Equal(75.0)) // 50 + 20 - 10 + 15

				// Verify all transfer events were created
				// Query for sender's events
				_, senderEventsResult, err := store.ProjectState(ctx, StateProjector{
					Query: NewLegacyQuery(
						[]Tag{{Key: "username", Value: fromUsername}},
						[]string{"MoneyTransferred"},
					),
					InitialState: []Event{},
					TransitionFn: func(state any, event Event) any {
						events := state.([]Event)
						return append(events, event)
					},
				})
				Expect(err).NotTo(HaveOccurred())
				senderEvents := senderEventsResult.([]Event)

				// Query for receiver's events
				_, receiverEventsResult, err := store.ProjectState(ctx, StateProjector{
					Query: NewLegacyQuery(
						[]Tag{{Key: "username", Value: toUsername}},
						[]string{"MoneyTransferred"},
					),
					InitialState: []Event{},
					TransitionFn: func(state any, event Event) any {
						events := state.([]Event)
						return append(events, event)
					},
				})
				Expect(err).NotTo(HaveOccurred())
				receiverEvents := receiverEventsResult.([]Event)

				// Combine and deduplicate events
				allEvents := make(map[string]Event)
				for _, event := range senderEvents {
					allEvents[event.ID] = event
				}
				for _, event := range receiverEvents {
					allEvents[event.ID] = event
				}

				// Convert map to slice
				events := make([]Event, 0, len(allEvents))
				for _, event := range allEvents {
					events = append(events, event)
				}

				Expect(events).To(HaveLen(3))

				// Verify event details
				for _, event := range events {
					var eventData MoneyTransferred
					err = json.Unmarshal(event.Data, &eventData)
					Expect(err).NotTo(HaveOccurred())
					Expect(eventData.Amount).To(BeElementOf(20.0, 10.0, 15.0))
				}
			})
		})
	})
})
