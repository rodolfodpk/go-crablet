// Package dcb consolidates production helper functions for the dcb package.
package dcb

import "go-crablet/pkg/dcb"

// NewTags creates a slice of Tag from alternating key-value pairs.
// It panics if an odd number of strings is provided.
func NewTags(kv ...string) []dcb.Tag {
	if len(kv)%2 != 0 {
		panic("NewTags: odd number of strings provided")
	}
	tags := make([]dcb.Tag, 0, len(kv)/2)
	for i := 0; i < len(kv); i += 2 {
		tags = append(tags, dcb.Tag{Key: kv[i], Value: kv[i+1]})
	}
	return tags
}

// NewQuery returns a Query with the given tags and (optional) event types.
// If eventTypes is nil or empty, the query will match any event type.
func NewQuery(tags []dcb.Tag, eventTypes ...string) dcb.Query {
	return dcb.Query{Tags: tags, EventTypes: eventTypes}
}

// NewInputEvent returns an InputEvent with the given event type, tags, and payload (data).
func NewInputEvent(eventType string, tags []dcb.Tag, data []byte) dcb.InputEvent {
	return dcb.InputEvent{Type: eventType, Tags: tags, Data: data}
}
