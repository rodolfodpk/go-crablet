package dcb

import (
	"context"
)

// EventStore defines the core interface for event sourcing operations
type EventStore interface {
	// Read reads events matching a query, optionally starting from a specified sequence position
	Read(ctx context.Context, query Query, options *ReadOptions) (SequencedEvents, error)

	// ReadStream returns a streaming iterator for events matching a query
	ReadStream(ctx context.Context, query Query, options *ReadOptions) (EventIterator, error)

	// Append atomically persists one or more events, optionally with an append condition
	Append(ctx context.Context, events []InputEvent, condition *AppendCondition) (int64, error)

	// ProjectDecisionModel projects multiple states using projectors and returns final states and append condition
	// This is the primary DCB API for building decision models in command handlers
	ProjectDecisionModel(ctx context.Context, query Query, options *ReadOptions, projectors []BatchProjector) (map[string]any, AppendCondition, error)
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
	ID            string `json:"id"`
	Type          string `json:"type"`
	Tags          []Tag  `json:"tags"`
	Data          []byte `json:"data"`
	Position      int64  `json:"position"`
	CausationID   string `json:"causation_id"`
	CorrelationID string `json:"correlation_id"`
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
