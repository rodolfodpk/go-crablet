package dcb

import (
	"context"
	"encoding/json"
)

// EventStore is the core interface for appending and reading events
type EventStore interface {
	// Append appends events to the store (always succeeds if no validation errors)
	// Uses PostgreSQL Read Committed isolation level (pgx.ReadCommitted)
	Append(ctx context.Context, events []InputEvent) error

	// AppendIf appends events to the store only if the condition is met
	// Uses PostgreSQL Repeatable Read isolation level (pgx.RepeatableRead)
	AppendIf(ctx context.Context, events []InputEvent, condition AppendCondition) error

	// AppendIfIsolated appends events to the store only if the condition is met
	// Uses PostgreSQL Serializable isolation level (pgx.Serializable)
	AppendIfIsolated(ctx context.Context, events []InputEvent, condition AppendCondition) error

	// Read reads events matching the query (no options)
	Read(ctx context.Context, query Query) ([]Event, error)

	// ReadWithOptions reads events matching the query with additional options
	ReadWithOptions(ctx context.Context, query Query, options *ReadOptions) ([]Event, error)

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

	// Error is set if there was an error processing events
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
// This is now an opaque type: construct only via NewInputEvent
// and access fields only via methods

type InputEvent interface {
	isInputEvent()
	GetType() string
	GetTags() []Tag
	GetData() []byte
}

type inputEvent struct {
	eventType string
	tags      []Tag
	data      []byte
}

func (e *inputEvent) isInputEvent()   {}
func (e *inputEvent) GetType() string { return e.eventType }
func (e *inputEvent) GetTags() []Tag  { return e.tags }
func (e *inputEvent) GetData() []byte { return e.data }

// Tag represents a key-value pair for event categorization
// This is now an opaque type: construct only via NewTag
// and access fields only via methods
type Tag interface {
	isTag()
	GetKey() string
	GetValue() string
}

type tag struct {
	key   string
	value string
}

func (t *tag) isTag()           {}
func (t *tag) GetKey() string   { return t.key }
func (t *tag) GetValue() string { return t.value }

// MarshalJSON ensures Tag is marshaled as {"key":..., "value":...}
func (t *tag) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}{
		Key:   t.key,
		Value: t.value,
	})
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
	FailIfEventsMatch *query `json:"fail_if_events_match"`
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
	if ac.FailIfEventsMatch == nil {
		return nil
	}
	var q Query = ac.FailIfEventsMatch
	return &q
}

// getAfter returns the internal after position (used by event store)
func (ac *appendCondition) getAfter() *int64 {
	return ac.After
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

// IsolationLevel is a public alias for isolation levels
// Valid values: "READ UNCOMMITTED", "READ COMMITTED", "REPEATABLE READ", "SERIALIZABLE"
type IsolationLevel string
