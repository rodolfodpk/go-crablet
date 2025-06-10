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
				Types: eventTypes,
				Tags:  tags,
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
				Types: []string{},
				Tags:  []Tag{},
			},
		},
	}
}

// NewQueryItem creates a new QueryItem with the given types and tags.
func NewQueryItem(types []string, tags []Tag) QueryItem {
	return QueryItem{
		Types: types,
		Tags:  tags,
	}
}

// ToLegacyQuery converts a Query to LegacyQuery for backward compatibility.
func (q Query) ToLegacyQuery() LegacyQuery {
	if len(q.Items) == 0 {
		return LegacyQuery{}
	}

	// For backward compatibility, we use the first item
	item := q.Items[0]
	return LegacyQuery{
		Tags:       item.Tags,
		EventTypes: item.Types,
	}
}

// FromLegacyQuery converts a LegacyQuery to Query.
func FromLegacyQuery(lq LegacyQuery) Query {
	return Query{
		Items: []QueryItem{
			{
				Types: lq.EventTypes,
				Tags:  lq.Tags,
			},
		},
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

// NewLegacyQuery creates a Query from the old-style Tags and EventTypes fields.
// This is a convenience function for backward compatibility in tests.
func NewLegacyQuery(tags []Tag, eventTypes []string) Query {
	return Query{
		Items: []QueryItem{
			{
				Types: eventTypes,
				Tags:  tags,
			},
		},
	}
}
