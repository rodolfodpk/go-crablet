package dcb

// NewTags creates a slice of Tag from alternating key-value string pairs.
// Example: NewTags("course_id", "C1", "user_id", "U1")
// Validation will be performed when tags are used in AppendEvents
func NewTags(kv ...string) []Tag {
	if len(kv)%2 != 0 || len(kv) == 0 {
		// Return empty slice instead of error
		return []Tag{}
	}

	tags := make([]Tag, 0, len(kv)/2)
	for i := 0; i < len(kv); i += 2 {
		tags = append(tags, Tag{Key: kv[i], Value: kv[i+1]})
	}
	return tags
}

// NewQuery creates a Query from tags and optional event types.
// If eventTypes is nil or empty, the query will match any event type.
func NewQuery(tags []Tag, eventTypes ...string) Query {
	return Query{
		Tags:       tags,
		EventTypes: eventTypes,
	}
}

// NewInputEvent creates a new InputEvent without validation
// Validation will be performed when events are appended in AppendEvents
func NewInputEvent(eventType string, tags []Tag, data []byte) InputEvent {
	return InputEvent{
		Type: eventType,
		Tags: tags,
		Data: data,
	}
}
