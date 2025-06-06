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

// NewProductDefinedEvent creates a new ProductDefined event
func NewProductDefinedEvent(productID string, price float64, minutesAgo int) InputEvent {
	data, _ := json.Marshal(ProductDefinedEvent{
		ProductID:  productID,
		Price:      price,
		MinutesAgo: minutesAgo,
	})
	return InputEvent{
		Type: "ProductDefined",
		Tags: []Tag{{Key: "product", Value: productID}},
		Data: data,
	}
}

// NewProductPriceChangedEvent creates a new ProductPriceChanged event
func NewProductPriceChangedEvent(productID string, newPrice float64, minutesAgo int) InputEvent {
	data, _ := json.Marshal(ProductPriceChangedEvent{
		ProductID:  productID,
		NewPrice:   newPrice,
		MinutesAgo: minutesAgo,
	})
	return InputEvent{
		Type: "ProductPriceChanged",
		Tags: []Tag{{Key: "product", Value: productID}},
		Data: data,
	}
}

// NewProductOrderedEvent creates a new ProductOrdered event
func NewProductOrderedEvent(productID string, price float64) InputEvent {
	data, _ := json.Marshal(ProductOrderedEvent{
		ProductID: productID,
		Price:     price,
	})
	return InputEvent{
		Type: "ProductOrdered",
		Tags: []Tag{{Key: "product", Value: productID}},
		Data: data,
	}
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

// NewProductAPI creates a new ProductAPI instance
func NewProductAPI(store EventStore) *ProductAPI {
	return &ProductAPI{store: store}
}

// OrderProductCommand represents the command to order a product
type OrderProductCommand struct {
	ProductID      string  `json:"productId"`
	DisplayedPrice float64 `json:"displayedPrice"`
}

// OrderProduct handles the order product command
func (api *ProductAPI) OrderProduct(ctx context.Context, cmd OrderProductCommand) error {
	const productPriceGracePeriod = 10 // minutes

	// Project current product price state
	projector := StateProjector{
		Query: Query{
			Tags:       []Tag{{Key: "product", Value: cmd.ProductID}},
			EventTypes: []string{"ProductDefined", "ProductPriceChanged"},
		},
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
					return ProductPriceState{
						LastValidOldPrice: currentState.LastValidOldPrice,
						ValidNewPrices:    append(currentState.ValidNewPrices, e.Price),
					}
				}
				return ProductPriceState{
					LastValidOldPrice: e.Price,
					ValidNewPrices:    currentState.ValidNewPrices,
				}

			case "ProductPriceChanged":
				var e ProductPriceChangedEvent
				if err := json.Unmarshal(event.Data, &e); err != nil {
					return currentState
				}
				if e.MinutesAgo <= productPriceGracePeriod {
					return ProductPriceState{
						LastValidOldPrice: currentState.LastValidOldPrice,
						ValidNewPrices:    append(currentState.ValidNewPrices, e.NewPrice),
					}
				}
				return ProductPriceState{
					LastValidOldPrice: e.NewPrice,
					ValidNewPrices:    currentState.ValidNewPrices,
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
	if priceState.LastValidOldPrice != cmd.DisplayedPrice && !contains(priceState.ValidNewPrices, cmd.DisplayedPrice) {
		return fmt.Errorf("invalid price for product \"%s\"", cmd.ProductID)
	}

	// Create the order event
	event := NewProductOrderedEvent(cmd.ProductID, cmd.DisplayedPrice)

	// Create a query that filters by product tag and event type
	appendQuery := Query{
		Tags:       []Tag{{Key: "product", Value: cmd.ProductID}},
		EventTypes: []string{"ProductOrdered"},
	}

	// Append the event using appendQuery
	_, err = api.store.AppendEvents(ctx, []InputEvent{event}, appendQuery, 0)
	if err != nil {
		return fmt.Errorf("failed to append order event: %w", err)
	}

	return nil
}

func contains(slice []float64, value float64) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}

var _ = Describe("Product Events", func() {
	var api *ProductAPI

	BeforeEach(func() {
		// Truncate the events table before each test
		err := truncateEventsTable(ctx, pool)
		Expect(err).NotTo(HaveOccurred())

		api = NewProductAPI(store)
	})

	Describe("OrderProduct", func() {
		Context("when product is defined", func() {
			It("fails with invalid displayed price", func() {
				// Given
				event := NewProductDefinedEvent("p1", 123, 0)
				query := Query{
					Tags:       []Tag{{Key: "product", Value: "p1"}},
					EventTypes: []string{"ProductDefined"},
				}
				_, err := store.AppendEvents(ctx, []InputEvent{event}, query, 0)
				Expect(err).NotTo(HaveOccurred())

				// When
				err = api.OrderProduct(ctx, OrderProductCommand{
					ProductID:      "p1",
					DisplayedPrice: 100,
				})

				// Then
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid price"))
			})

			It("succeeds with valid displayed price", func() {
				// Given
				event := NewProductDefinedEvent("p1", 123, 0)
				query := Query{
					Tags:       []Tag{{Key: "product", Value: "p1"}},
					EventTypes: []string{"ProductDefined"},
				}
				_, err := store.AppendEvents(ctx, []InputEvent{event}, query, 0)
				Expect(err).NotTo(HaveOccurred())

				// When
				err = api.OrderProduct(ctx, OrderProductCommand{
					ProductID:      "p1",
					DisplayedPrice: 123,
				})

				// Then
				Expect(err).NotTo(HaveOccurred())
			})

			It("fails with a price that was never valid", func() {
				// Given
				event := NewProductDefinedEvent("p1", 123, 20)
				query := Query{
					Tags:       []Tag{{Key: "product", Value: "p1"}},
					EventTypes: []string{"ProductDefined"},
				}
				_, err := store.AppendEvents(ctx, []InputEvent{event}, query, 0)
				Expect(err).NotTo(HaveOccurred())

				// When
				err = api.OrderProduct(ctx, OrderProductCommand{
					ProductID:      "p1",
					DisplayedPrice: 100,
				})

				// Then
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid price"))
			})

			It("fails with a price that was changed more than 10 minutes ago", func() {
				// Given
				events := []InputEvent{
					NewProductDefinedEvent("p1", 123, 20),
					NewProductPriceChangedEvent("p1", 134, 20),
				}
				query := Query{
					Tags:       []Tag{{Key: "product", Value: "p1"}},
					EventTypes: []string{"ProductDefined", "ProductPriceChanged"},
				}
				_, err := store.AppendEvents(ctx, events, query, 0)
				Expect(err).NotTo(HaveOccurred())

				// When
				err = api.OrderProduct(ctx, OrderProductCommand{
					ProductID:      "p1",
					DisplayedPrice: 123,
				})

				// Then
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid price"))
			})

			It("succeeds with initial valid price", func() {
				// Given
				event := NewProductDefinedEvent("p1", 123, 20)
				query := Query{
					Tags:       []Tag{{Key: "product", Value: "p1"}},
					EventTypes: []string{"ProductDefined"},
				}
				_, err := store.AppendEvents(ctx, []InputEvent{event}, query, 0)
				Expect(err).NotTo(HaveOccurred())

				// When
				err = api.OrderProduct(ctx, OrderProductCommand{
					ProductID:      "p1",
					DisplayedPrice: 123,
				})

				// Then
				Expect(err).NotTo(HaveOccurred())
			})

			It("succeeds with a price that was changed less than 10 minutes ago", func() {
				// Given
				events := []InputEvent{
					NewProductDefinedEvent("p1", 123, 20),
					NewProductPriceChangedEvent("p1", 134, 9),
				}
				query := Query{
					Tags:       []Tag{{Key: "product", Value: "p1"}},
					EventTypes: []string{"ProductDefined", "ProductPriceChanged"},
				}
				_, err := store.AppendEvents(ctx, events, query, 0)
				Expect(err).NotTo(HaveOccurred())

				// When
				err = api.OrderProduct(ctx, OrderProductCommand{
					ProductID:      "p1",
					DisplayedPrice: 123,
				})

				// Then
				Expect(err).NotTo(HaveOccurred())
			})

			It("succeeds with valid new price", func() {
				// Given
				events := []InputEvent{
					NewProductDefinedEvent("p1", 123, 20),
					NewProductPriceChangedEvent("p1", 134, 9),
				}
				query := Query{
					Tags:       []Tag{{Key: "product", Value: "p1"}},
					EventTypes: []string{"ProductDefined", "ProductPriceChanged"},
				}
				_, err := store.AppendEvents(ctx, events, query, 0)
				Expect(err).NotTo(HaveOccurred())

				// When
				err = api.OrderProduct(ctx, OrderProductCommand{
					ProductID:      "p1",
					DisplayedPrice: 134,
				})

				// Then
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})
})
