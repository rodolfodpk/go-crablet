package dcb

import (
	"encoding/json"
	"fmt"
)

// validateQueryTags validates the query tags and returns a ValidationError if invalid
func validateQueryTags(query Query) error {
	// Handle empty query (matches all events)
	if len(query.GetItems()) == 0 {
		return nil
	}

	// Validate each query item
	for itemIndex, item := range query.GetItems() {
		// Validate individual tags if present
		for i, t := range item.GetTags() {
			if t.GetKey() == "" {
				return &ValidationError{
					EventStoreError: EventStoreError{
						Op:  "validateQueryTags",
						Err: fmt.Errorf("empty tag key in item %d", itemIndex),
					},
					Field: fmt.Sprintf("item[%d].tag[%d].key", itemIndex, i),
				}
			}
			if t.GetValue() == "" {
				return &ValidationError{
					EventStoreError: EventStoreError{
						Op:  "validateQueryTags",
						Err: fmt.Errorf("empty value for key %s in tag %d of item %d", t.GetKey(), i, itemIndex),
					},
					Field: fmt.Sprintf("item[%d].tag[%d].value", itemIndex, i),
					Value: t.GetKey(),
				}
			}
		}

		// Validate event types if present
		for i, eventType := range item.GetEventTypes() {
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
	if e.GetType() == "" {
		return &ValidationError{
			EventStoreError: EventStoreError{
				Op:  "validateEvent",
				Err: fmt.Errorf("empty type in event %d", index),
			},
			Field: "type",
			Value: fmt.Sprintf("event[%d]", index),
		}
	}

	if len(e.GetTags()) == 0 {
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
	for j, t := range e.GetTags() {
		if t.GetKey() == "" {
			return &ValidationError{
				EventStoreError: EventStoreError{
					Op:  "validateEvent",
					Err: fmt.Errorf("empty tag key in event %d", index),
				},
				Field: fmt.Sprintf("event[%d].tag[%d].key", index, j),
			}
		}
		if t.GetValue() == "" {
			return &ValidationError{
				EventStoreError: EventStoreError{
					Op:  "validateEvent",
					Err: fmt.Errorf("empty value for key %s in tag %d of event %d", t.GetKey(), j, index),
				},
				Field: fmt.Sprintf("event[%d].tag[%d].value", index, j),
				Value: t.GetKey(),
			}
		}
	}

	// Validate Data as JSON
	if !json.Valid(e.GetData()) {
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
	if len(events) > es.config.MaxBatchSize {
		return &ValidationError{
			EventStoreError: EventStoreError{
				Op:  operation,
				Err: fmt.Errorf("batch size %d exceeds maximum %d", len(events), es.config.MaxBatchSize),
			},
			Field: "batchSize",
			Value: fmt.Sprintf("%d", len(events)),
		}
	}
	return nil
}

// validateEvents validates all events in a batch
func validateEvents(events []InputEvent) error {
	// Check batch size limit (default 1000)
	const maxBatchSize = 1000
	if len(events) > maxBatchSize {
		return &ValidationError{
			EventStoreError: EventStoreError{
				Op:  "validateEvents",
				Err: fmt.Errorf("batch size %d exceeds maximum %d", len(events), maxBatchSize),
			},
			Field: "batchSize",
			Value: fmt.Sprintf("%d", len(events)),
		}
	}

	for i, event := range events {
		if err := validateEvent(event, i); err != nil {
			return err
		}
	}
	return nil
}
