package dcb

import (
	"context"
)

type (
	// SequencedEvents represents a collection of events with their sequence positions
	// This matches the DCB specification for the return type of read operations
	SequencedEvents struct {
		Events   []Event
		Position int64 // Position of the last event in the stream
	}

	// Tag is a key-value pair for querying events.
	Tag struct {
		Key   string
		Value string
	}

	// QueryItem represents a single query condition
	QueryItem struct {
		EventTypes []string
		Tags       []Tag
	}

	// Query represents a query for events with multiple query items
	Query struct {
		Items []QueryItem
	}

	// AppendCondition is used to enforce consistency when appending events.
	// This matches the DCB specification exactly
	AppendCondition struct {
		FailIfEventsMatch Query  // Query that must not match any events for the append to succeed
		After             *int64 // Optional sequence position to ignore events before this position
	}

	// StateProjector defines how to project state from events.
	StateProjector struct {
		Query        Query
		InitialState any
		TransitionFn func(state any, e Event) any
	}

	// BatchProjectionResult represents the result of projecting multiple states
	BatchProjectionResult struct {
		Position int64
		States   map[string]any // Key is projector identifier
	}

	// StreamingProjectionResult represents a streaming result for multiple projectors
	StreamingProjectionResult struct {
		Position        int64           // Position of the last event processed
		States          map[string]any  // Key is projector identifier
		Iterator        EventIterator   // For streaming access to processed events
		AppendCondition AppendCondition // For optimistic locking when appending new events
	}

	// BatchProjector defines a projector with an identifier for batch operations
	BatchProjector struct {
		ID             string // Unique identifier for this projector
		StateProjector StateProjector
	}

	// InputEvent represents an event to be appended to the store.
	InputEvent struct {
		Type string // Event type (e.g., "Subscription")
		Tags []Tag  // Tags for querying (e.g., {"course_id": "C1"})
		Data []byte // JSON-encoded event payload
	}

	// Event represents a persisted event in the system.
	Event struct {
		ID            string // Unique event identifier (UUID)
		Type          string // Event type (e.g., "Subscription")
		Tags          []Tag  // Tags for querying (e.g., {"course_id": "C1"})
		Data          []byte // Event payload
		Position      int64  // Position in the event stream
		CausationID   string // UUID of the event that caused this event
		CorrelationID string // UUID linking to the root event or process
	}

	// EventStore provides methods to read and append events in a PostgreSQL database.
	// This interface matches the DCB specification with Read() and Append() methods
	EventStore interface {
		// Read reads events matching a query, optionally starting from a specified sequence position
		// This matches the DCB specification exactly
		Read(ctx context.Context, query Query, options *ReadOptions) (SequencedEvents, error)

		// ReadStream returns a pure event iterator for streaming events from PostgreSQL
		ReadStream(ctx context.Context, query Query, options *ReadOptions) (EventIterator, error)

		// ProjectDecisionModel projects multiple states using projectors and returns final states and append condition
		ProjectDecisionModel(ctx context.Context, query Query, options *ReadOptions, projectors []BatchProjector) (map[string]any, AppendCondition, error)

		// Append atomically persists one or more events, optionally with an append condition
		// This matches the DCB specification exactly
		Append(ctx context.Context, events []InputEvent, condition *AppendCondition) (int64, error)

		// ProjectBatch projects multiple states using multiple projectors in a single database query.
		// This is more efficient than calling ProjectState multiple times as it uses one combined query
		// and streams events once, routing them to the appropriate projectors.
		// Returns the latest position processed and a map of projector results keyed by projector ID.
		ProjectBatch(ctx context.Context, projectors []BatchProjector) (BatchProjectionResult, error)

		// ProjectBatchUpTo projects multiple states up to a specific position using multiple projectors.
		// Similar to ProjectBatch but limits the events processed to those up to maxPosition.
		ProjectBatchUpTo(ctx context.Context, projectors []BatchProjector, maxPosition int64) (BatchProjectionResult, error)
	}

	// ReadOptions represents options for reading events
	// This matches the DCB specification for read options
	ReadOptions struct {
		FromPosition *int64 // Optional starting sequence position
		Limit        *int   // Optional limit on number of events to read
	}

	// EventIterator allows streaming access to events from the store
	EventIterator interface {
		Next() bool
		Event() Event
		Err() error
		Close() error
	}
)
