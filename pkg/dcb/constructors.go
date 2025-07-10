package dcb

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// =============================================================================
// EventStore Constructors
// =============================================================================

// NewEventStore creates a new EventStore instance with the given PostgreSQL connection pool.
// This is the main constructor for creating an event store.
func NewEventStore(ctx context.Context, pool *pgxpool.Pool) (EventStore, error) {
	return newEventStore(ctx, pool, nil)
}

// NewEventStoreWithConfig creates a new EventStore instance with custom configuration.
func NewEventStoreWithConfig(ctx context.Context, pool *pgxpool.Pool, config EventStoreConfig) (EventStore, error) {
	return newEventStore(ctx, pool, &config)
}

// NewEventStoreFromPool creates a new EventStore from an existing pool without connection testing.
// This is used for tests that share a PostgreSQL container.
func NewEventStoreFromPool(pool *pgxpool.Pool) EventStore {
	cfg := EventStoreConfig{
		MaxBatchSize:           1000,
		LockTimeout:            5000,
		StreamBuffer:           1000,
		DefaultAppendIsolation: IsolationLevelReadCommitted,
	}
	return &eventStore{
		pool:   pool,
		config: cfg,
	}
}

// NewEventStoreFromPoolWithConfig creates a new EventStore from an existing pool with custom configuration.
func NewEventStoreFromPoolWithConfig(pool *pgxpool.Pool, config EventStoreConfig) EventStore {
	return &eventStore{
		pool:   pool,
		config: config,
	}
}

// =============================================================================
// Event Constructors
// =============================================================================

// NewInputEvent creates a new InputEvent with the given type, tags, and data.
// Validation is performed when the event is used in EventStore operations.
func NewInputEvent(eventType string, tags []Tag, data []byte) InputEvent {
	return &inputEvent{
		eventType: eventType,
		tags:      tags,
		data:      data,
	}
}

// NewInputEventUnsafe creates a new InputEvent without validation.
// Use this only when you're certain the data is valid and you need maximum performance.
func NewInputEventUnsafe(eventType string, tags []Tag, data []byte) InputEvent {
	return &inputEvent{
		eventType: eventType,
		tags:      tags,
		data:      data,
	}
}

// NewEventBatch creates a slice of events from the given InputEvents.
// This is a convenience function for creating event batches, particularly useful
// when appending multiple related events in a single operation.
func NewEventBatch(events ...InputEvent) []InputEvent {
	return events
}

// =============================================================================
// Tag Constructors
// =============================================================================

// NewTag creates a single tag from key-value pair.
func NewTag(key, value string) Tag {
	return &tag{
		key:   key,
		value: value,
	}
}

// NewTags creates a slice of tags from key-value pairs.
// Validation is performed when the tags are used in EventStore operations.
func NewTags(kv ...string) []Tag {
	if len(kv)%2 != 0 {
		// Return empty tags instead of panicking - validation will happen in EventStore operations
		return []Tag{}
	}
	tags := make([]Tag, len(kv)/2)
	for i := 0; i < len(kv); i += 2 {
		tags[i/2] = NewTag(kv[i], kv[i+1])
	}
	return tags
}

// =============================================================================
// Query Constructors
// =============================================================================

// NewQuery creates a new Query with the given tags and event types.
// This creates a single QueryItem with the specified tags and event types.
func NewQuery(tags []Tag, eventTypes ...string) Query {
	return &query{
		Items: []QueryItem{
			NewQueryItem(eventTypes, tags),
		},
	}
}

// NewQueryEmpty creates a new empty query
func NewQueryEmpty() Query {
	return &query{Items: []QueryItem{}}
}

// NewQueryFromItems creates a new query from a list of query items
func NewQueryFromItems(items ...QueryItem) Query {
	return &query{Items: items}
}

// NewQueryAll creates a query that matches all events.
func NewQueryAll() Query {
	return &query{
		Items: []QueryItem{
			NewQueryItem([]string{}, []Tag{}),
		},
	}
}

// NewQueryItem creates a new QueryItem with the given types and tags.
func NewQueryItem(types []string, tags []Tag) QueryItem {
	return &queryItem{
		EventTypes: types,
		Tags:       tags,
	}
}

// =============================================================================
// AppendCondition Constructors
// =============================================================================

// NewAppendCondition creates a new AppendCondition with the given fail condition.
// This is the primary constructor for optimistic locking conditions.
func NewAppendCondition(failIfEventsMatch Query) AppendCondition {
	if failIfEventsMatch == nil {
		return &appendCondition{}
	}
	return &appendCondition{
		FailIfEventsMatch: failIfEventsMatch.(*query),
	}
}
