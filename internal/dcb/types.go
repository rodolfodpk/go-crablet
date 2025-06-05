package dcb

import "context"

type (

	// Tag is a key-value pair for querying events.
	Tag struct {
		Key   string
		Value string
	}

	// Query defines criteria for selecting events.
	Query struct {
		Tags       []Tag    // Events must match all these tags (empty means match any tag)
		EventTypes []string // Events must match one of these types (empty means match any type)
	}

	StateProjector struct {
		InitialState any
		TransitionFn func(any, Event) any
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
		CausationID   string // UUID of the event that caused this event (optional)
		CorrelationID string // UUID linking to the root event or process (optional)
	}

	// EventStore provides methods to append and read events in a PostgreSQL database.
	EventStore interface {
		AppendEvents(ctx context.Context, events []InputEvent, query Query, latestKnownPosition int64) (int64, error)
		AppendEventsIfNotExists(ctx context.Context, events []InputEvent, query Query, latestKnownPosition int64, stateProjector StateProjector) (int64, error)
		ProjectState(ctx context.Context, query Query, stateProjector StateProjector) (int64, any, error)
		ProjectStateUpTo(ctx context.Context, query Query, stateProjector StateProjector, maxPosition int64) (int64, any, error)
		Close()
	}
)
