package dcb

// NewTags creates a slice of tags from key-value pairs.
// It panics if the number of arguments is odd.
func NewTags(kv ...string) []Tag {
	if len(kv)%2 != 0 {
		panic("NewTags: odd number of arguments")
	}
	tags := make([]Tag, len(kv)/2)
	for i := 0; i < len(kv); i += 2 {
		tags[i/2] = Tag{Key: kv[i], Value: kv[i+1]}
	}
	return tags
}

// NewQuery creates a new Query with the given tags and event types.
// This is a backward-compatible function that creates a single QueryItem.
func NewQuery(tags []Tag, eventTypes ...string) Query {
	return Query{
		Items: []QueryItem{
			{
				EventTypes: eventTypes,
				Tags:       tags,
			},
		},
	}
}

// NewQueryFromItems creates a new Query from multiple QueryItems.
// This follows the DCB specification for complex queries.
func NewQueryFromItems(items ...QueryItem) Query {
	return Query{Items: items}
}

// NewQueryAll creates a query that matches all events.
func NewQueryAll() Query {
	return Query{
		Items: []QueryItem{
			{
				EventTypes: []string{},
				Tags:       []Tag{},
			},
		},
	}
}

// NewQueryItem creates a new QueryItem with the given types and tags.
func NewQueryItem(types []string, tags []Tag) QueryItem {
	return QueryItem{
		EventTypes: types,
		Tags:       tags,
	}
}

// NewInputEvent creates a new InputEvent with the given type, tags, and data.
func NewInputEvent(eventType string, tags []Tag, data []byte) InputEvent {
	return InputEvent{
		Type: eventType,
		Tags: tags,
		Data: data,
	}
}

// NewEventBatch creates a slice of events from the given InputEvents.
// This is a convenience function for creating event batches, particularly useful
// when appending multiple related events in a single operation.
func NewEventBatch(events ...InputEvent) []InputEvent {
	return events
}
