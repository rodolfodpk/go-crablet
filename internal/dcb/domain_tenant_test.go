// Package dcb provides domain-specific types and helpers for the tenant domain.
package dcb

import (
	"encoding/json"
	"go-crablet/pkg/dcb"
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
func NewTenantCreatedEvent(name string, tags []dcb.Tag) dcb.InputEvent {
	data, _ := json.Marshal(TenantCreatedEvent{Name: name})
	return dcb.InputEvent{
		Type: "TenantCreated",
		Tags: tags,
		Data: data,
	}
}

// NewOrderCreatedEvent creates a new order created event
func NewOrderCreatedEvent(amount float64, tags []dcb.Tag) dcb.InputEvent {
	data, _ := json.Marshal(OrderCreatedEvent{Amount: amount})
	return dcb.InputEvent{
		Type: "OrderCreated",
		Tags: tags,
		Data: data,
	}
}

// NewOrderAssignedEvent creates a new order assigned event
func NewOrderAssignedEvent(status string, tags []dcb.Tag) dcb.InputEvent {
	data, _ := json.Marshal(OrderAssignedEvent{Status: status})
	return dcb.InputEvent{
		Type: "OrderAssigned",
		Tags: tags,
		Data: data,
	}
}

// TenantProjector creates a projector for tenant events
func TenantProjector(tenantID string) dcb.StateProjector {
	return dcb.StateProjector{
		Query:        NewQuery(NewTags("tenant_id", tenantID)),
		InitialState: &TenantState{},
		TransitionFn: func(state any, e dcb.Event) any {
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
