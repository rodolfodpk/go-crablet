package dcb

import (
	"encoding/json"
	"fmt"
)

// validateQueryTags validates the query tags and returns a ValidationError if invalid
func validateQueryTags(query Query) error {
	// Handle empty query (matches all events)
	if len(query.Items) == 0 {
		return nil
	}

	// Validate each query item
	for itemIndex, item := range query.Items {
		// Validate individual tags if present
		for i, t := range item.Tags {
			if t.Key == "" {
				return &ValidationError{
					EventStoreError: EventStoreError{
						Op:  "validateQueryTags",
						Err: fmt.Errorf("empty key in tag %d of item %d", i, itemIndex),
					},
					Field: fmt.Sprintf("item[%d].tag[%d].key", itemIndex, i),
					Value: fmt.Sprintf("tag[%d]", i),
				}
			}
			if t.Value == "" {
				return &ValidationError{
					EventStoreError: EventStoreError{
						Op:  "validateQueryTags",
						Err: fmt.Errorf("empty value for key %s in tag %d of item %d", t.Key, i, itemIndex),
					},
					Field: fmt.Sprintf("item[%d].tag[%d].value", itemIndex, i),
					Value: t.Key,
				}
			}
		}

		// Validate event types if present
		for i, eventType := range item.EventTypes {
			if eventType == "" {
				return &ValidationError{
					EventStoreError: EventStoreError{
						Op:  "validateQueryTags",
						Err: fmt.Errorf("empty event type at index %d of item %d", i, itemIndex),
					},
					Field: fmt.Sprintf("item[%d].eventTypes[%d]", itemIndex, i),
					Value: fmt.Sprintf("index[%d]", i),
				}
			}
		}
	}

	return nil
}

// validateEvent validates a single event and returns a ValidationError if invalid
func validateEvent(e InputEvent, index int) error {
	// Early validation checks with early returns
	if e.Type == "" {
		return &ValidationError{
			EventStoreError: EventStoreError{
				Op:  "validateEvent",
				Err: fmt.Errorf("empty type in event %d", index),
			},
			Field: "type",
			Value: fmt.Sprintf("event[%d]", index),
		}
	}

	if len(e.Tags) == 0 {
		return &ValidationError{
			EventStoreError: EventStoreError{
				Op:  "validateEvent",
				Err: fmt.Errorf("empty tags in event %d", index),
			},
			Field: "tags",
			Value: fmt.Sprintf("event[%d]", index),
		}
	}

	// Validate tags efficiently
	for j, t := range e.Tags {
		if t.Key == "" {
			return &ValidationError{
				EventStoreError: EventStoreError{
					Op:  "validateEvent",
					Err: fmt.Errorf("empty key in tag %d of event %d", j, index),
				},
				Field: fmt.Sprintf("event[%d].tag[%d].key", index, j),
				Value: fmt.Sprintf("tag[%d]", j),
			}
		}
		if t.Value == "" {
			return &ValidationError{
				EventStoreError: EventStoreError{
					Op:  "validateEvent",
					Err: fmt.Errorf("empty value for key %s in tag %d of event %d", t.Key, j, index),
				},
				Field: fmt.Sprintf("event[%d].tag[%d].value", index, j),
				Value: t.Key,
			}
		}
	}

	// Validate Data as JSON
	if !json.Valid(e.Data) {
		return &ValidationError{
			EventStoreError: EventStoreError{
				Op:  "validateEvent",
				Err: fmt.Errorf("invalid JSON data in event %d", index),
			},
			Field: "data",
			Value: fmt.Sprintf("event[%d]", index),
		}
	}

	return nil
}

// validateBatchSize validates that the batch size is within limits
func (es *eventStore) validateBatchSize(events []InputEvent, operation string) error {
	if len(events) > es.maxBatchSize {
		return &ValidationError{
			EventStoreError: EventStoreError{
				Op:  operation,
				Err: fmt.Errorf("batch size %d exceeds maximum %d", len(events), es.maxBatchSize),
			},
			Field: "batchSize",
			Value: fmt.Sprintf("%d", len(events)),
		}
	}
	return nil
}

// validateEvents validates all events in a batch
func validateEvents(events []InputEvent) error {
	for i, event := range events {
		if err := validateEvent(event, i); err != nil {
			return err
		}
	}
	return nil
}
