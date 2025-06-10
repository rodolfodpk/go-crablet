// Package dcb provides domain-specific types and helpers for the tenant domain.
package dcb

import (
	"encoding/json"
)

// TenantState represents the state of a tenant
type TenantState struct {
	UserCount      int
	OrderCount     int
	AssignedOrders int
	EventTypes     []string
}

// TenantCreatedEvent represents when a tenant is created
type TenantCreatedEvent struct {
	Name string `json:"name"`
}

// OrderCreatedEvent represents when an order is created
type OrderCreatedEvent struct {
	Amount float64 `json:"amount"`
}

// OrderAssignedEvent represents when an order is assigned
type OrderAssignedEvent struct {
	Status string `json:"status"`
}

// NewTenantCreatedEvent creates a new tenant created event
func NewTenantCreatedEvent(tenantID string) InputEvent {
	data, _ := json.Marshal(TenantCreatedEvent{Name: "Test Tenant"})
	return NewInputEvent(
		"TenantCreated",
		NewTags("tenant_id", tenantID),
		data,
	)
}

// NewUserRegisteredEvent creates a new user registered event
func NewUserRegisteredEvent(tenantID, userID string) InputEvent {
	data, _ := json.Marshal(UserRegisteredEvent{Name: "Test User"})
	return NewInputEvent(
		"UserRegistered",
		NewTags("tenant_id", tenantID, "user_id", userID),
		data,
	)
}

// NewOrderCreatedEvent creates a new order created event
func NewOrderCreatedEvent(tenantID, orderID string) InputEvent {
	data, _ := json.Marshal(OrderCreatedEvent{Amount: 100})
	return NewInputEvent(
		"OrderCreated",
		NewTags("tenant_id", tenantID, "order_id", orderID),
		data,
	)
}

// NewOrderAssignedEvent creates a new order assigned event
func NewOrderAssignedEvent(tenantID, orderID, status string) InputEvent {
	data, _ := json.Marshal(OrderAssignedEvent{Status: status})
	return NewInputEvent(
		"OrderAssigned",
		NewTags("tenant_id", tenantID, "order_id", orderID),
		data,
	)
}

// TenantProjector creates a projector for tenant events
func TenantProjector(tenantID string) StateProjector {
	return StateProjector{
		Query:        NewLegacyQuery(NewTags("tenant_id", tenantID), []string{}),
		InitialState: &TenantState{},
		TransitionFn: func(state any, e Event) any {
			s := state.(*TenantState)
			s.EventTypes = append(s.EventTypes, e.Type)

			// Check for user_id tag
			for _, tag := range e.Tags {
				if tag.Key == "user_id" {
					s.UserCount++
				}
				if tag.Key == "order_id" {
					s.OrderCount++
				}
			}

			// Check for assigned orders
			if e.Type == "OrderAssigned" {
				s.AssignedOrders++
			}

			return s
		},
	}
}
