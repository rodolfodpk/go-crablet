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
func NewQuery(tags []Tag, eventTypes ...string) Query {
	return Query{
		Tags:       tags,
		EventTypes: eventTypes,
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
