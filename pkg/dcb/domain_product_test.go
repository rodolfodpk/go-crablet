// Package dcb provides domain-specific types and helpers for the product domain.
package dcb

import (
	"context"
	"encoding/json"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// ProductDefinedEvent represents when a product is defined with its price
type ProductDefinedEvent struct {
	ProductID  string  `json:"productId"`
	Price      float64 `json:"price"`
	MinutesAgo int     `json:"minutesAgo,omitempty"`
}

// ProductPriceChangedEvent represents when a product's price is changed
type ProductPriceChangedEvent struct {
	ProductID  string  `json:"productId"`
	NewPrice   float64 `json:"newPrice"`
	MinutesAgo int     `json:"minutesAgo,omitempty"`
}

// ProductOrderedEvent represents when a product is ordered
type ProductOrderedEvent struct {
	ProductID string  `json:"productId"`
	Price     float64 `json:"price"`
}

// EventMetadata contains metadata about when an event occurred
type EventMetadata struct {
	MinutesAgo int `json:"minutesAgo"`
}

// ProductPriceState represents the state of product prices
type ProductPriceState struct {
	LastValidOldPrice float64   `json:"lastValidOldPrice"`
	ValidNewPrices    []float64 `json:"validNewPrices"`
}

// ProductAPI provides methods for product-related operations
type ProductAPI struct {
	store EventStore
}

// OrderProductCommand represents the command to order a product
type OrderProductCommand struct {
	ProductID      string  `json:"productId"`
	DisplayedPrice float64 `json:"displayedPrice"`
}

// NewProductDefinedEvent creates a new ProductDefined event
func NewProductDefinedEvent(productID string, price float64, minutesAgo int) InputEvent {
	data, _ := json.Marshal(ProductDefinedEvent{
		ProductID:  productID,
		Price:      price,
		MinutesAgo: minutesAgo,
	})
	return NewInputEvent(
		"ProductDefined",
		NewTags("product", productID),
		data,
	)
}

// NewProductPriceChangedEvent creates a new ProductPriceChanged event
func NewProductPriceChangedEvent(productID string, newPrice float64, minutesAgo int) InputEvent {
	data, _ := json.Marshal(ProductPriceChangedEvent{
		ProductID:  productID,
		NewPrice:   newPrice,
		MinutesAgo: minutesAgo,
	})
	return NewInputEvent(
		"ProductPriceChanged",
		NewTags("product", productID),
		data,
	)
}

// NewProductOrderedEvent creates a new ProductOrdered event
func NewProductOrderedEvent(productID string, price float64) InputEvent {
	data, _ := json.Marshal(ProductOrderedEvent{
		ProductID: productID,
		Price:     price,
	})
	return NewInputEvent(
		"ProductOrdered",
		NewTags("product", productID),
		data,
	)
}

// NewProductAPI creates a new ProductAPI instance
func NewProductAPI(store EventStore) *ProductAPI {
	return &ProductAPI{store: store}
}

// OrderProduct handles the order product command
func (api *ProductAPI) OrderProduct(ctx context.Context, cmd OrderProductCommand) error {
	const productPriceGracePeriod = 15 // minutes

	// Project current product price state
	projector := StateProjector{
		Query: NewQuery(NewTags("product", cmd.ProductID), "ProductDefined", "ProductPriceChanged"),
		InitialState: ProductPriceState{
			LastValidOldPrice: 0,
			ValidNewPrices:    make([]float64, 0),
		},
		TransitionFn: func(state any, event Event) any {
			currentState := state.(ProductPriceState)

			switch event.Type {
			case "ProductDefined":
				var e ProductDefinedEvent
				if err := json.Unmarshal(event.Data, &e); err != nil {
					return currentState
				}
				if e.MinutesAgo <= productPriceGracePeriod {
					// Add to valid prices if within grace period, but avoid duplicates
					if !contains(currentState.ValidNewPrices, e.Price) {
						currentState.ValidNewPrices = append(currentState.ValidNewPrices, e.Price)
					}
					return currentState
				}
				// If it's old, it becomes the last valid old price and we reset valid prices
				return ProductPriceState{
					LastValidOldPrice: e.Price,
					ValidNewPrices:    make([]float64, 0),
				}

			case "ProductPriceChanged":
				var e ProductPriceChangedEvent
				if err := json.Unmarshal(event.Data, &e); err != nil {
					return currentState
				}
				if e.MinutesAgo <= productPriceGracePeriod {
					// Add to valid prices if within grace period, but avoid duplicates
					if !contains(currentState.ValidNewPrices, e.NewPrice) {
						currentState.ValidNewPrices = append(currentState.ValidNewPrices, e.NewPrice)
					}
					return currentState
				}
				// If the price change is old, update the last valid old price and reset valid prices
				return ProductPriceState{
					LastValidOldPrice: e.NewPrice,
					ValidNewPrices:    make([]float64, 0),
				}
			}
			return currentState
		},
	}

	_, state, err := api.store.ProjectState(ctx, projector)
	if err != nil {
		return fmt.Errorf("failed to project product price state: %w", err)
	}

	priceState := state.(ProductPriceState)

	// Check if the displayed price matches any recent price
	// We only consider recent prices (within grace period) as valid
	if !contains(priceState.ValidNewPrices, cmd.DisplayedPrice) {
		return fmt.Errorf("invalid price for product \"%s\"", cmd.ProductID)
	}

	// Create the order event
	event := NewProductOrderedEvent(cmd.ProductID, cmd.DisplayedPrice)

	// Create a query that filters by product tag and event type
	appendQuery := NewQuery(
		NewTags("product", cmd.ProductID),
		"ProductOrdered",
	)

	// Get the current position for this product's events
	pos, _, err := api.store.ProjectState(ctx, StateProjector{
		Query:        appendQuery,
		InitialState: []Event{},
		TransitionFn: func(state any, event Event) any {
			events := state.([]Event)
			return append(events, event)
		},
	})
	if err != nil {
		return fmt.Errorf("failed to query events: %w", err)
	}

	// Append the event using the current position
	_, err = api.store.AppendEvents(ctx, []InputEvent{event}, appendQuery, pos)
	if err != nil {
		return fmt.Errorf("failed to append order event: %w", err)
	}

	return nil
}

// contains checks if a slice contains a value
func contains(slice []float64, value float64) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}

var _ = Describe("Product Events", func() {
	var (
		api       *ProductAPI
		productID string
	)

	BeforeEach(func() {
		// Truncate the events table before each test
		err := truncateEventsTable(ctx, pool)
		Expect(err).NotTo(HaveOccurred())

		api = NewProductAPI(store)
		Expect(api).NotTo(BeNil(), "ProductAPI should not be nil")

		productID = "test-product"
	})

	Context("Given a product with a defined price", func() {
		BeforeEach(func() {
			// Given
			event := NewProductDefinedEvent(productID, 100.0, 0)
			query := NewQuery(NewTags("product", productID), "ProductDefined")
			_, err := store.AppendEvents(ctx, []InputEvent{event}, query, 0)
			Expect(err).NotTo(HaveOccurred())
		})

		When("ordering the product at the defined price", func() {
			It("should succeed and create a ProductOrdered event", func() {
				// When
				err := api.OrderProduct(ctx, OrderProductCommand{
					ProductID:      productID,
					DisplayedPrice: 100.0,
				})

				// Then
				Expect(err).NotTo(HaveOccurred())

				// Verify the event was created
				_, eventsResult, err := store.ProjectState(ctx, StateProjector{
					Query: NewQuery(
						NewTags("product", productID),
						"ProductOrdered",
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
				Expect(events[0].Type).To(Equal("ProductOrdered"))

				var eventData ProductOrderedEvent
				err = json.Unmarshal(events[0].Data, &eventData)
				Expect(err).NotTo(HaveOccurred())
				Expect(eventData.ProductID).To(Equal(productID))
				Expect(eventData.Price).To(Equal(100.0))
			})
		})

		When("ordering the product at a different price", func() {
			It("should fail with invalid price error", func() {
				// When
				err := api.OrderProduct(ctx, OrderProductCommand{
					ProductID:      productID,
					DisplayedPrice: 200.0,
				})

				// Then
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid price for product"))

				// Verify no order event was created
				_, eventsResult, err := store.ProjectState(ctx, StateProjector{
					Query: NewQuery(
						NewTags("product", productID),
						"ProductOrdered",
					),
					InitialState: []Event{},
					TransitionFn: func(state any, event Event) any {
						events := state.([]Event)
						return append(events, event)
					},
				})
				Expect(err).NotTo(HaveOccurred())
				events := eventsResult.([]Event)
				Expect(events).To(BeEmpty())
			})
		})
	})

	Context("Given a product with a recent price change", func() {
		BeforeEach(func() {
			// Given
			defineEvent := NewProductDefinedEvent(productID, 100.0, 20)     // 20 minutes ago
			changeEvent := NewProductPriceChangedEvent(productID, 200.0, 5) // 5 minutes ago
			query := NewQuery(NewTags("product", productID), "ProductDefined", "ProductPriceChanged")
			_, err := store.AppendEvents(ctx, []InputEvent{defineEvent, changeEvent}, query, 0)
			Expect(err).NotTo(HaveOccurred())
		})

		When("ordering the product at the new price", func() {
			It("should succeed and create a ProductOrdered event", func() {
				// When
				err := api.OrderProduct(ctx, OrderProductCommand{
					ProductID:      productID,
					DisplayedPrice: 200.0,
				})

				// Then
				Expect(err).NotTo(HaveOccurred())

				// Verify the event was created
				_, eventsResult, err := store.ProjectState(ctx, StateProjector{
					Query: NewQuery(
						NewTags("product", productID),
						"ProductOrdered",
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
				Expect(events[0].Type).To(Equal("ProductOrdered"))

				var eventData ProductOrderedEvent
				err = json.Unmarshal(events[0].Data, &eventData)
				Expect(err).NotTo(HaveOccurred())
				Expect(eventData.ProductID).To(Equal(productID))
				Expect(eventData.Price).To(Equal(200.0))
			})
		})

		When("ordering the product at the old price", func() {
			It("should fail with invalid price error", func() {
				// When
				err := api.OrderProduct(ctx, OrderProductCommand{
					ProductID:      productID,
					DisplayedPrice: 100.0,
				})

				// Then
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid price for product"))

				// Verify no order event was created
				_, eventsResult, err := store.ProjectState(ctx, StateProjector{
					Query: NewQuery(
						NewTags("product", productID),
						"ProductOrdered",
					),
					InitialState: []Event{},
					TransitionFn: func(state any, event Event) any {
						events := state.([]Event)
						return append(events, event)
					},
				})
				Expect(err).NotTo(HaveOccurred())
				events := eventsResult.([]Event)
				Expect(events).To(BeEmpty())
			})
		})
	})

	Context("Given a product with multiple recent price changes", func() {
		BeforeEach(func() {
			// Given
			defineEvent := NewProductDefinedEvent(productID, 100.0, 20)       // 20 minutes ago
			changeEvent1 := NewProductPriceChangedEvent(productID, 200.0, 15) // 15 minutes ago
			changeEvent2 := NewProductPriceChangedEvent(productID, 300.0, 5)  // 5 minutes ago
			query := NewQuery(NewTags("product", productID), "ProductDefined", "ProductPriceChanged")
			_, err := store.AppendEvents(ctx, []InputEvent{defineEvent, changeEvent1, changeEvent2}, query, 0)
			Expect(err).NotTo(HaveOccurred())
		})

		When("ordering the product at any recent price", func() {
			It("should succeed for all recent prices", func() {
				// Test ordering at the most recent price
				err := api.OrderProduct(ctx, OrderProductCommand{
					ProductID:      productID,
					DisplayedPrice: 300.0,
				})
				Expect(err).NotTo(HaveOccurred())

				// Test ordering at the second most recent price
				err = api.OrderProduct(ctx, OrderProductCommand{
					ProductID:      productID,
					DisplayedPrice: 200.0,
				})
				Expect(err).NotTo(HaveOccurred())

				// Test ordering at the original price (should fail)
				err = api.OrderProduct(ctx, OrderProductCommand{
					ProductID:      productID,
					DisplayedPrice: 100.0,
				})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid price for product"))

				// Verify two order events were created
				_, eventsResult, err := store.ProjectState(ctx, StateProjector{
					Query: NewQuery(
						NewTags("product", productID),
						"ProductOrdered",
					),
					InitialState: []Event{},
					TransitionFn: func(state any, event Event) any {
						events := state.([]Event)
						return append(events, event)
					},
				})
				Expect(err).NotTo(HaveOccurred())
				events := eventsResult.([]Event)
				Expect(events).To(HaveLen(2))

				// Verify first order
				var eventData1 ProductOrderedEvent
				err = json.Unmarshal(events[0].Data, &eventData1)
				Expect(err).NotTo(HaveOccurred())
				Expect(eventData1.ProductID).To(Equal(productID))
				Expect(eventData1.Price).To(Equal(300.0))

				// Verify second order
				var eventData2 ProductOrderedEvent
				err = json.Unmarshal(events[1].Data, &eventData2)
				Expect(err).NotTo(HaveOccurred())
				Expect(eventData2.ProductID).To(Equal(productID))
				Expect(eventData2.Price).To(Equal(200.0))
			})
		})
	})
})
