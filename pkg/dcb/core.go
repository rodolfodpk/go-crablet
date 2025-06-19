package dcb

import (
	"context"
)

// EventStore defines the core interface for event sourcing operations
type EventStore interface {
	// Read reads events matching a query, optionally starting from a specified sequence position
	Read(ctx context.Context, query Query, options *ReadOptions) (SequencedEvents, error)

	// Append atomically persists one or more events, optionally with an append condition
	Append(ctx context.Context, events []InputEvent, condition *AppendCondition) (int64, error)
}

// ProjectionResult represents a single projection result from channel-based projection
type ProjectionResult struct {
	// ProjectorID identifies which projector produced this result
	ProjectorID string

	// State is the projected state for this projector
	State interface{}

	// Event is the event that was processed to produce this state
	Event Event

	// Position is the sequence position of the event
	Position int64

	// Error is set if there was an error processing this event
	Error error
}

// CrabletEventStore extends EventStore with channel-based streaming capabilities
// This provides an alternative Go-idiomatic interface for event streaming
type CrabletEventStore interface {
	EventStore

	// ReadStreamChannel creates a channel-based stream of events matching a query
	// This is optimized for small to medium datasets (< 500 events) and provides
	// a more Go-idiomatic interface using channels
	ReadStreamChannel(ctx context.Context, query Query) (<-chan Event, error)

	// ProjectDecisionModel projects multiple states using multiple projectors and returns final states with append condition
	ProjectDecisionModel(ctx context.Context, projectors []BatchProjector) (map[string]any, AppendCondition, error)

	// ProjectDecisionModelChannel projects multiple states using channel-based streaming
	// This is optimized for small to medium datasets (< 500 events) and provides
	// a more Go-idiomatic interface using channels for state projection
	ProjectDecisionModelChannel(ctx context.Context, projectors []BatchProjector) (<-chan ProjectionResult, error)
}

// EventIterator provides a streaming interface for reading events
type EventIterator interface {
	// Next advances to the next event, returning false if no more events
	Next() bool

	// Event returns the current event
	Event() Event

	// Err returns any error that occurred during iteration
	Err() error

	// Close closes the iterator and releases resources
	Close() error
}

// Event represents a single event in the event store
type Event struct {
	Type     string `json:"type"`
	Tags     []Tag  `json:"tags"`
	Data     []byte `json:"data"`
	Position int64  `json:"position"`
}

// InputEvent represents an event to be appended to the store
type InputEvent struct {
	Type string `json:"type"`
	Tags []Tag  `json:"tags"`
	Data []byte `json:"data"`
}

// Tag represents a key-value pair for event categorization
type Tag struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// Query represents a composite query with multiple conditions combined with OR logic
type Query struct {
	Items []QueryItem `json:"items"`
}

// QueryItem represents a single atomic query condition
type QueryItem struct {
	EventTypes []string `json:"event_types"`
	Tags       []Tag    `json:"tags"`
}

// ReadOptions provides options for reading events
type ReadOptions struct {
	FromPosition *int64 `json:"from_position"`
	Limit        *int   `json:"limit"`
	BatchSize    *int   `json:"batch_size"` // Batch size for cursor-based streaming
}

// SequencedEvents represents a collection of events with their final position
type SequencedEvents struct {
	Events   []Event `json:"events"`
	Position int64   `json:"position"`
}

// AppendCondition represents conditions for optimistic locking during append operations
type AppendCondition struct {
	FailIfEventsMatch *Query `json:"fail_if_events_match"`
	After             *int64 `json:"after"`
}

// StateProjector defines how to project a state from events
type StateProjector struct {
	Query        Query                            `json:"query"`
	InitialState any                              `json:"initial_state"`
	TransitionFn func(state any, event Event) any `json:"-"`
}

// BatchProjector combines a state projector with an identifier
type BatchProjector struct {
	ID             string         `json:"id"`
	StateProjector StateProjector `json:"state_projector"`
}

// StreamingProjectionResult represents the result of streaming projection
type StreamingProjectionResult struct {
	States          map[string]any  `json:"states"`
	AppendCondition AppendCondition `json:"append_condition"`
	Position        int64           `json:"position"`
	ProcessedCount  int             `json:"processed_count"`
}
