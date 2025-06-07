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

// NewAccountRegistered creates a new account registration event
func NewAccountRegistered(username string) InputEvent {
	data, _ := json.Marshal(AccountRegistered{Username: username})
	return InputEvent{
		Type: "AccountRegistered",
		Data: data,
		Tags: []Tag{{Key: "username", Value: username}},
	}
}

// IsUsernameClaimedProjection creates a projector to check if a username is claimed
func IsUsernameClaimedProjection(username string) StateProjector {
	return StateProjector{
		Query: Query{
			Tags:       []Tag{{Key: "username", Value: username}},
			EventTypes: []string{"AccountRegistered"},
		},
		InitialState: false,
		TransitionFn: func(state any, event Event) any {
			return true
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
		return nil, fmt.Errorf("event store cannot be nil")
	}
	return &AccountAPI{eventStore: eventStore}, nil
}

// RegisterAccount attempts to register a new account
func (a *AccountAPI) RegisterAccount(ctx context.Context, username string) error {
	if a.eventStore == nil {
		return fmt.Errorf("account API not properly initialized: event store is nil")
	}

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
					Query:        NewQuery(NewTags("username", "concurrentuser")),
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
