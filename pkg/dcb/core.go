package dcb

import (
	"context"
)

// EventStore is the core interface for appending and reading events
// Append now returns only error
type EventStore interface {
	// Append appends events to the store with optional append condition
	Append(ctx context.Context, events []InputEvent, condition AppendCondition) error

	// Read reads events matching the query with optional read options
	Read(ctx context.Context, query Query, options *ReadOptions) (SequencedEvents, error)

	// ProjectDecisionModel projects multiple states using projectors and returns final states and append condition
	// This is a go-crablet feature for building decision models in command handlers
	ProjectDecisionModel(ctx context.Context, projectors []BatchProjector) (map[string]any, AppendCondition, error)
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

// ChannelEventStore extends EventStore with channel-based streaming capabilities
// This provides an alternative Go-idiomatic interface for event streaming
type ChannelEventStore interface {
	EventStore

	// ReadStreamChannel creates a channel-based stream of events matching a query
	// This is optimized for small to medium datasets (< 500 events) and provides
	// a more Go-idiomatic interface using channels
	ReadStreamChannel(ctx context.Context, query Query) (<-chan Event, error)

	// ProjectDecisionModelChannel projects multiple states using channel-based streaming
	// This is optimized for small to medium datasets (< 500 events) and provides
	// a more Go-idiomatic interface using channels for state projection
	ProjectDecisionModelChannel(ctx context.Context, projectors []BatchProjector) (<-chan ProjectionResult, error)
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
// This is opaque to consumers - they can only construct it via helper functions
type Query interface {
	// isQuery is a marker method to make this interface unexported
	isQuery()
	// getItems returns the internal query items (used by event store)
	getItems() []QueryItem
}

// QueryItem represents a single atomic query condition
// This is opaque to consumers - they can only construct it via helper functions
type QueryItem interface {
	// isQueryItem is a marker method to make this interface unexported
	isQueryItem()
	// getEventTypes returns the internal event types (used by event store)
	getEventTypes() []string
	// getTags returns the internal tags (used by event store)
	getTags() []Tag
}

// query is the internal implementation
type query struct {
	Items []QueryItem `json:"items"`
}

// isQuery implements Query
func (q *query) isQuery() {}

// getItems returns the internal query items (used by event store)
func (q *query) getItems() []QueryItem {
	return q.Items
}

// queryItem is the internal implementation
type queryItem struct {
	EventTypes []string `json:"event_types"`
	Tags       []Tag    `json:"tags"`
}

// isQueryItem implements QueryItem
func (qi *queryItem) isQueryItem() {}

// getEventTypes returns the internal event types (used by event store)
func (qi *queryItem) getEventTypes() []string {
	return qi.EventTypes
}

// getTags returns the internal tags (used by event store)
func (qi *queryItem) getTags() []Tag {
	return qi.Tags
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
// This is opaque to consumers - they can only construct it via helper functions
type AppendCondition interface {
	// isAppendCondition is a marker method to make this interface unexported
	isAppendCondition()
	// setAfterPosition sets the after position (used internally by event store)
	setAfterPosition(after *int64)
	// getFailIfEventsMatch returns the internal query (used by event store)
	getFailIfEventsMatch() *Query
	// getAfter returns the internal after position (used by event store)
	getAfter() *int64
}

// appendCondition is the internal implementation
type appendCondition struct {
	FailIfEventsMatch *Query `json:"fail_if_events_match"`
	After             *int64 `json:"after"`
}

// isAppendCondition implements AppendCondition
func (ac *appendCondition) isAppendCondition() {}

// setAfterPosition sets the after position (used internally by event store)
func (ac *appendCondition) setAfterPosition(after *int64) {
	ac.After = after
}

// getFailIfEventsMatch returns the internal query (used by event store)
func (ac *appendCondition) getFailIfEventsMatch() *Query {
	return ac.FailIfEventsMatch
}

// getAfter returns the internal after position (used by event store)
func (ac *appendCondition) getAfter() *int64 {
	return ac.After
}

// NewAppendCondition creates a new AppendCondition with the specified query
// This is the DCB-compliant way to construct append conditions
func NewAppendCondition(failIfEventsMatch *Query) AppendCondition {
	return &appendCondition{
		FailIfEventsMatch: failIfEventsMatch,
		After:             nil, // Will be set during processing
	}
}

// NewAppendConditionWithAfter creates a new AppendCondition with both query and after position
// This is used internally by the event store for optimistic locking
func NewAppendConditionWithAfter(failIfEventsMatch *Query, after *int64) AppendCondition {
	return &appendCondition{
		FailIfEventsMatch: failIfEventsMatch,
		After:             after,
	}
}

// NewAppendConditionAfter creates a new AppendCondition with only after position
// This is used for optimistic locking based on position
func NewAppendConditionAfter(after *int64) AppendCondition {
	return &appendCondition{
		FailIfEventsMatch: nil,
		After:             after,
	}
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
