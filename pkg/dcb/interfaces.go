package dcb

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// =============================================================================
// CORE ABSTRACTIONS (High-level, most relevant)
// =============================================================================

// EventStore is the core interface for appending and reading events
// This is the primary abstraction that users interact with
type EventStore interface {
	// Query reads events matching the query with optional cursor
	// after == nil: query from beginning of stream
	// after != nil: query from specified cursor position
	Query(ctx context.Context, query Query, after *Cursor) ([]Event, error)

	// QueryStream creates a channel-based stream of events matching a query with optional cursor
	// after == nil: stream from beginning of stream
	// after != nil: stream from specified cursor position
	// This is optimized for large datasets and provides backpressure through channels
	// for efficient memory usage and Go-idiomatic streaming
	QueryStream(ctx context.Context, query Query, after *Cursor) (<-chan Event, error)

	// Append appends events to the store with optional condition
	// condition == nil: unconditional append
	// condition != nil: conditional append with optimistic locking
	Append(ctx context.Context, events []InputEvent, condition *AppendCondition) error

	// Project projects state from events matching projectors with optional cursor
	// after == nil: project from beginning of stream
	// after != nil: project from specified cursor position
	// Returns final aggregated states and append condition for optimistic locking
	Project(ctx context.Context, projectors []StateProjector, after *Cursor) (map[string]any, AppendCondition, error)

	// ProjectStream creates a channel-based stream of projected states with optional cursor
	// after == nil: stream from beginning of stream
	// after != nil: stream from specified cursor position
	// Returns intermediate states and append conditions via channels for streaming projections
	ProjectStream(ctx context.Context, projectors []StateProjector, after *Cursor) (<-chan map[string]any, <-chan AppendCondition, error)

	// GetConfig returns the current EventStore configuration
	GetConfig() EventStoreConfig

	// GetPool exposes the underlying PostgreSQL connection pool (pgxpool.Pool).
	// This is intended for advanced/internal use cases such as custom transaction management,
	// integration testing, or infrastructure extensions. Regular application logic should NOT
	// use this method, as it bypasses the event store's consistency and abstraction guarantees.
	GetPool() *pgxpool.Pool
}

// CommandExecutor executes commands and generates events
// This is the high-level interface for command-driven event generation
type CommandExecutor interface {
	ExecuteCommand(ctx context.Context, command Command, handler CommandHandler, condition *AppendCondition) ([]InputEvent, error)
}

// =============================================================================
// SUPPORTING INTERFACES (Used by core abstractions)
// =============================================================================

// CommandHandler handles command execution and generates events
type CommandHandler interface {
	Handle(ctx context.Context, store EventStore, command Command) ([]InputEvent, error)
}

// CommandHandlerFunc allows using functions as CommandHandler implementations
type CommandHandlerFunc func(ctx context.Context, store EventStore, command Command) ([]InputEvent, error)

func (f CommandHandlerFunc) Handle(ctx context.Context, store EventStore, command Command) ([]InputEvent, error) {
	return f(ctx, store, command)
}

// Query represents a composite query with multiple conditions combined with OR logic
// This is opaque to consumers - they can only construct it via helper functions
// Now exposes GetItems for public access
type Query interface {
	// isQuery is a marker method to make this interface unexported
	isQuery()
	// getItems returns the internal query items (used by event store)
	GetItems() []QueryItem
}

// AppendCondition represents conditions for optimistic locking during append operations
// This is opaque to consumers - they can only construct it via helper functions
type AppendCondition interface {
	// isAppendCondition is a marker method to make this interface unexported
	isAppendCondition()
	// setAfterCursor sets the after cursor for proper (transaction_id, position) tracking
	setAfterCursor(after *Cursor)
	// getFailIfEventsMatch returns the internal query (used by event store)
	getFailIfEventsMatch() *Query
	// getAfterCursor returns the internal after cursor (used by event store)
	getAfterCursor() *Cursor
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

// Command represents a command that triggers event generation
type Command interface {
	GetType() string
	GetData() []byte
	GetMetadata() map[string]interface{}
}

// Tag represents a key-value pair for event categorization
// This is now an opaque type: construct only via NewTag
// and access fields only via methods
type Tag interface {
	isTag()
	GetKey() string
	GetValue() string
}

// QueryItem represents a single atomic query condition
// This is opaque to consumers - they can only construct it via helper functions
// Now exposes GetEventTypes and GetTags for public access
type QueryItem interface {
	// isQueryItem is a marker method to make this interface unexported
	isQueryItem()
	// getEventTypes returns the internal event types (used by event store)
	GetEventTypes() []string
	// getTags returns the internal tags (used by event store)
	GetTags() []Tag
}

// =============================================================================
// CONCRETE TYPES AND STRUCTS
// =============================================================================

// Event represents a single event in the store
type Event struct {
	Type          string    `json:"type"`
	Tags          []Tag     `json:"tags"`
	Data          []byte    `json:"data"`
	TransactionID uint64    `json:"transaction_id"`
	Position      int64     `json:"position"`
	OccurredAt    time.Time `json:"occurred_at"`
}

// Cursor represents a position in the event stream
// When used in Read/Project operations, events are returned EXCLUSIVE of this position
// (i.e., events after this cursor, not including the cursor position itself)
type Cursor struct {
	TransactionID uint64 `json:"transaction_id"`
	Position      int64  `json:"position"`
}

// StateProjector defines how to project a state from events
type StateProjector struct {
	ID           string                           `json:"id"`
	Query        Query                            `json:"query"`
	InitialState any                              `json:"initial_state"`
	TransitionFn func(state any, event Event) any `json:"-"`
}

// =============================================================================
// CONFIGURATION TYPES
// =============================================================================

// IsolationLevel represents PostgreSQL transaction isolation levels as a type-safe enum
// Only valid values can be constructed via constants or ParseIsolationLevel
type IsolationLevel int

const (
	IsolationLevelReadCommitted IsolationLevel = iota
	IsolationLevelRepeatableRead
	IsolationLevelSerializable
)

func (l IsolationLevel) String() string {
	switch l {
	case IsolationLevelReadCommitted:
		return "READ_COMMITTED"
	case IsolationLevelRepeatableRead:
		return "REPEATABLE_READ"
	case IsolationLevelSerializable:
		return "SERIALIZABLE"
	default:
		return "UNKNOWN"
	}
}

func ParseIsolationLevel(s string) (IsolationLevel, error) {
	switch s {
	case "READ_COMMITTED":
		return IsolationLevelReadCommitted, nil
	case "REPEATABLE_READ":
		return IsolationLevelRepeatableRead, nil
	case "SERIALIZABLE":
		return IsolationLevelSerializable, nil
	default:
		return IsolationLevelReadCommitted, fmt.Errorf("invalid isolation level: %s", s)
	}
}

// EventStoreConfig contains configuration for EventStore behavior
type EventStoreConfig struct {
	MaxBatchSize           int            `json:"max_batch_size"`
	LockTimeout            int            `json:"lock_timeout"`             // Lock timeout in milliseconds for advisory locks (optional feature, currently unused)
	StreamBuffer           int            `json:"stream_buffer"`            // Channel buffer size for streaming operations
	DefaultAppendIsolation IsolationLevel `json:"default_append_isolation"` // Default isolation level for Append operations
	QueryTimeout           int            `json:"query_timeout"`            // Query timeout in milliseconds (defensive against hanging queries)
	AppendTimeout          int            `json:"append_timeout"`           // Append timeout in milliseconds (defensive against hanging appends)
	// TargetEventsTable removed - always use 'events' table for maximum performance
}

// =============================================================================
// INTERNAL IMPLEMENTATIONS (Private)
// =============================================================================

type inputEvent struct {
	eventType string
	tags      []Tag
	data      []byte
}

func (e *inputEvent) isInputEvent()   {}
func (e *inputEvent) GetType() string { return e.eventType }
func (e *inputEvent) GetTags() []Tag  { return e.tags }
func (e *inputEvent) GetData() []byte { return e.data }

type tag struct {
	key   string
	value string
}

func (t *tag) isTag()           {}
func (t *tag) GetKey() string   { return t.key }
func (t *tag) GetValue() string { return t.value }

type command struct {
	commandType string
	data        []byte
	metadata    map[string]interface{}
}

func (c *command) GetType() string                     { return c.commandType }
func (c *command) GetData() []byte                     { return c.data }
func (c *command) GetMetadata() map[string]interface{} { return c.metadata }

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

// query is the internal implementation
type query struct {
	Items []QueryItem `json:"items"`
}

// isQuery implements Query
func (q *query) isQuery() {}

// getItems returns the internal query items (used by event store)
func (q *query) GetItems() []QueryItem {
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
func (qi *queryItem) GetEventTypes() []string {
	return qi.EventTypes
}

// getTags returns the internal tags (used by event store)
func (qi *queryItem) GetTags() []Tag {
	return qi.Tags
}

// appendCondition is the internal implementation
type appendCondition struct {
	FailIfEventsMatch *query  `json:"fail_if_events_match"`
	AfterCursor       *Cursor `json:"after_cursor"`
}

// isAppendCondition implements AppendCondition
func (ac *appendCondition) isAppendCondition() {}

// setAfterCursor sets the after cursor for proper (transaction_id, position) tracking
func (ac *appendCondition) setAfterCursor(after *Cursor) {
	ac.AfterCursor = after
}

// getFailIfEventsMatch returns the internal query (used by event store)
func (ac *appendCondition) getFailIfEventsMatch() *Query {
	if ac.FailIfEventsMatch == nil {
		return nil
	}
	var q Query = ac.FailIfEventsMatch
	return &q
}

// getAfterCursor returns the internal after cursor (used by event store)
func (ac *appendCondition) getAfterCursor() *Cursor {
	return ac.AfterCursor
}
