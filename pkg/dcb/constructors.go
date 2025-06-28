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
	return newEventStore(ctx, pool)
}

// NewEventStoreFromPool creates a new EventStore from an existing pool without connection testing.
// This is used for tests that share a PostgreSQL container.
func NewEventStoreFromPool(pool *pgxpool.Pool) EventStore {
	return &eventStore{
		pool:         pool,
		maxBatchSize: 1000, // Default maximum batch size
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
// This is a backward-compatible function that creates a single QueryItem.
func NewQuery(tags []Tag, eventTypes ...string) Query {
	return &query{
		Items: []QueryItem{
			NewQueryItem(eventTypes, tags),
		},
	}
}

// NewQuerySimple creates a new Query with the given tags and event types.
// This is a convenience function that combines NewTags and NewQuery.
// Validation is performed when the query is used in EventStore operations.
func NewQuerySimple(tags []Tag, eventTypes ...string) Query {
	// Remove validation from constructor - validation will happen in EventStore operations
	return NewQuery(tags, eventTypes...)
}

// NewQuerySimpleUnsafe creates a new Query without validation.
// Use this only when you're certain the data is valid and you need maximum performance.
func NewQuerySimpleUnsafe(tags []Tag, eventTypes ...string) Query {
	return NewQuery(tags, eventTypes...)
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

// NewQItem creates a new QueryItem with a single event type and tags.
// This simplifies the common case of querying for one event type.
func NewQItem(eventType string, tags []Tag) QueryItem {
	return NewQueryItem([]string{eventType}, tags)
}

// NewQItemKV creates a new QueryItem with a single event type and key-value tags.
// This is the most concise way to create a QueryItem for a single event type.
func NewQItemKV(eventType string, kv ...string) QueryItem {
	return NewQueryItem([]string{eventType}, NewTags(kv...))
}

// =============================================================================
// AppendCondition Constructors
// =============================================================================

// NewAppendCondition creates a new AppendCondition with the given fail condition.
func NewAppendCondition(failIfEventsMatch Query) AppendCondition {
	var q *query
	if failIfEventsMatch != nil {
		q = failIfEventsMatch.(*query)
	}
	return &appendCondition{
		FailIfEventsMatch: q,
	}
}

// NewAppendConditionWithAfter creates a new AppendCondition with both fail condition and after position.
func NewAppendConditionWithAfter(failIfEventsMatch Query, after *int64) AppendCondition {
	var q *query
	if failIfEventsMatch != nil {
		q = failIfEventsMatch.(*query)
	}
	return &appendCondition{
		FailIfEventsMatch: q,
		After:             after,
	}
}

// NewAppendConditionAfter creates a new AppendCondition with only the after position.
func NewAppendConditionAfter(after *int64) AppendCondition {
	return &appendCondition{
		After: after,
	}
}
