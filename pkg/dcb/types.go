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

	// AppendCondition is used to enforce consistency when appending events.
	// It ensures that between the time of building the Decision Model and appending the events,
	// no new events were stored by another client that match the same query.
	AppendCondition struct {
		FailIfEventsMatch Query  // Query that must not match any events for the append to succeed
		After             *int64 // Optional sequence position to ignore events before this position
	}

	// StateProjector defines how to project state from events.
	StateProjector struct {
		Query        Query // Query to filter events for this projector
		InitialState any
		TransitionFn func(any, Event) any // TODO should this receive only the JSON-encoded data?
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

	// EventStore provides methods to append and read events in a PostgreSQL database.
	EventStore interface {
		// AppendEvents appends events to the store, ensuring consistency with the given query and position.
		AppendEvents(ctx context.Context, events []InputEvent, query Query, latestPosition int64) (int64, error)

		// AppendEventsIf appends events only if no events match the append condition.
		// It uses the condition to enforce consistency by failing if any events match the query
		// after the specified position (if any).
		AppendEventsIf(ctx context.Context, events []InputEvent, condition AppendCondition) (int64, error)

		// GetCurrentPosition returns the current position for the given query.
		GetCurrentPosition(ctx context.Context, query Query) (int64, error)

		// ProjectState computes a state by streaming events matching the projector's query.
		ProjectState(ctx context.Context, projector StateProjector) (int64, any, error)

		// ProjectStateUpTo computes a state by streaming events matching the projector's query, up to maxPosition.
		ProjectStateUpTo(ctx context.Context, projector StateProjector, maxPosition int64) (int64, any, error)
	}
)
