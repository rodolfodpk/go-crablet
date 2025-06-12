package dcb

import (
	"context"
	"fmt"
	"sync"

	"github.com/jackc/pgx/v5"
)

// StreamingProjectionIterator implements EventIterator for streaming projection results
type StreamingProjectionIterator struct {
	rows            pgx.Rows
	err             error
	event           Event
	projectors      []BatchProjector
	states          map[string]any
	position        int64
	mu              sync.RWMutex
	processedCount  int
	result          *StreamingProjectionResult // Reference to update AppendCondition
	appendCondition AppendCondition
}

// SimpleEventIterator implements EventIterator for streaming events from PostgreSQL
type SimpleEventIterator struct {
	rows  pgx.Rows
	err   error
	event Event
}

// ReadStream returns a pure event iterator for streaming events from PostgreSQL
func (es *eventStore) ReadStream(ctx context.Context, query Query, options *ReadOptions) (EventIterator, error) {
	if len(query.Items) == 0 {
		return nil, &ValidationError{
			EventStoreError: EventStoreError{
				Op:  "ReadStream",
				Err: fmt.Errorf("query must contain at least one item"),
			},
			Field: "query",
			Value: "empty",
		}
	}

	// Build SQL query based on query items
	sqlQuery, args, err := es.buildReadQuerySQL(query, options)
	if err != nil {
		return nil, err
	}

	// Execute the query - this starts the streaming
	rows, err := es.pool.Query(ctx, sqlQuery, args...)
	if err != nil {
		return nil, &ResourceError{
			EventStoreError: EventStoreError{
				Op:  "ReadStream",
				Err: fmt.Errorf("failed to execute read query: %w", err),
			},
			Resource: "database",
		}
	}

	return &SimpleEventIterator{rows: rows}, nil
}

// Next processes the next event
func (it *SimpleEventIterator) Next() bool {
	if !it.rows.Next() {
		return false
	}

	var row rowEvent
	if err := it.rows.Scan(&row.ID, &row.Type, &row.Tags, &row.Data, &row.Position, &row.CausationID, &row.CorrelationID); err != nil {
		it.err = &ResourceError{
			EventStoreError: EventStoreError{
				Op:  "SimpleEventIterator.Next",
				Err: fmt.Errorf("failed to scan event row: %w", err),
			},
			Resource: "database",
		}
		return false
	}

	// Convert row to Event
	it.event = convertRowToEvent(row)
	return true
}

// Event returns the current event
func (it *SimpleEventIterator) Event() Event {
	return it.event
}

// Err returns any error that occurred during iteration
func (it *SimpleEventIterator) Err() error {
	if it.err != nil {
		return it.err
	}
	return it.rows.Err()
}

// Close closes the iterator and releases resources
func (it *SimpleEventIterator) Close() error {
	it.rows.Close()
	return nil
}

// ProjectDecisionModel projects multiple states using projectors and returns final states and append condition
func (es *eventStore) ProjectDecisionModel(ctx context.Context, query Query, options *ReadOptions, projectors []BatchProjector) (map[string]any, AppendCondition, error) {
	if len(query.Items) == 0 {
		return nil, AppendCondition{}, &ValidationError{
			EventStoreError: EventStoreError{
				Op:  "ProjectDecisionModel",
				Err: fmt.Errorf("query must contain at least one item"),
			},
			Field: "query",
			Value: "empty",
		}
	}

	// Validate projectors
	for _, bp := range projectors {
		if bp.StateProjector.TransitionFn == nil {
			return nil, AppendCondition{}, &ValidationError{
				EventStoreError: EventStoreError{
					Op:  "ProjectDecisionModel",
					Err: fmt.Errorf("projector %s has nil transition function", bp.ID),
				},
				Field: "transitionFn",
				Value: "nil",
			}
		}
	}

	// Build SQL query based on query items
	sqlQuery, args, err := es.buildReadQuerySQL(query, options)
	if err != nil {
		return nil, AppendCondition{}, err
	}

	// Execute the query
	rows, err := es.pool.Query(ctx, sqlQuery, args...)
	if err != nil {
		return nil, AppendCondition{}, &ResourceError{
			EventStoreError: EventStoreError{
				Op:  "ProjectDecisionModel",
				Err: fmt.Errorf("failed to execute read query: %w", err),
			},
			Resource: "database",
		}
	}
	defer rows.Close()

	// Initialize states for all projectors
	states := make(map[string]any)
	for _, bp := range projectors {
		states[bp.ID] = bp.StateProjector.InitialState
	}

	// Build AppendCondition from projector queries for optimistic locking
	appendCondition := es.buildAppendConditionFromProjectors(projectors)

	// Process all events to build final states
	var lastPosition int64
	for rows.Next() {
		var row rowEvent
		if err := rows.Scan(&row.ID, &row.Type, &row.Tags, &row.Data, &row.Position, &row.CausationID, &row.CorrelationID); err != nil {
			return nil, AppendCondition{}, &ResourceError{
				EventStoreError: EventStoreError{
					Op:  "ProjectDecisionModel",
					Err: fmt.Errorf("failed to scan event row: %w", err),
				},
				Resource: "database",
			}
		}

		// Convert row to Event
		event := convertRowToEvent(row)
		lastPosition = row.Position

		// Update AppendCondition.After field with current position
		appendCondition.After = &lastPosition

		// Apply projectors
		for _, bp := range projectors {
			if es.eventMatchesProjector(event, bp.StateProjector) {
				states[bp.ID] = bp.StateProjector.TransitionFn(states[bp.ID], event)
			}
		}
	}

	// Check for iteration errors
	if err := rows.Err(); err != nil {
		return nil, AppendCondition{}, &ResourceError{
			EventStoreError: EventStoreError{
				Op:  "ProjectDecisionModel",
				Err: fmt.Errorf("error iterating over events: %w", err),
			},
			Resource: "database",
		}
	}

	return states, appendCondition, nil
}

// Next processes the next event and applies it to all matching projectors
func (it *StreamingProjectionIterator) Next() bool {
	if !it.rows.Next() {
		return false
	}

	var row rowEvent
	if err := it.rows.Scan(&row.ID, &row.Type, &row.Tags, &row.Data, &row.Position, &row.CausationID, &row.CorrelationID); err != nil {
		it.err = &ResourceError{
			EventStoreError: EventStoreError{
				Op:  "streamingProjectionIterator.Next",
				Err: fmt.Errorf("failed to scan event row at position %d: %w", row.Position, err),
			},
			Resource: "database",
		}
		return false
	}

	// Convert row to Event
	event := convertRowToEvent(row)
	it.event = event
	it.position = row.Position

	// Update AppendCondition.After field with current position
	it.appendCondition.After = &row.Position

	// Route event to matching projectors
	for _, bp := range it.projectors {
		if it.eventMatchesProjector(event, bp.StateProjector) {
			// Apply projector with panic recovery
			func() {
				defer func() {
					if r := recover(); r != nil {
						it.err = &EventStoreError{
							Op:  "streamingProjectionIterator.Next",
							Err: fmt.Errorf("panic in projector %s for event type %s at position %d: %v", bp.ID, event.Type, event.Position, r),
						}
					}
				}()
				it.states[bp.ID] = bp.StateProjector.TransitionFn(it.states[bp.ID], event)
			}()
			if it.err != nil {
				return false
			}
		}
	}

	it.processedCount++
	return true
}

// Event returns the current event
func (it *StreamingProjectionIterator) Event() Event {
	return it.event
}

// Err returns any error that occurred during iteration
func (it *StreamingProjectionIterator) Err() error {
	if it.err != nil {
		return it.err
	}
	return it.rows.Err()
}

// Close closes the iterator and releases resources
func (it *StreamingProjectionIterator) Close() error {
	it.rows.Close()
	return nil
}

// GetStates returns a copy of the current states
func (it *StreamingProjectionIterator) GetStates() map[string]any {
	it.mu.RLock()
	defer it.mu.RUnlock()

	// Create a copy of states to avoid race conditions
	statesCopy := make(map[string]any)
	for k, v := range it.states {
		statesCopy[k] = v
	}
	return statesCopy
}

// States returns the current states (alias for GetStates for interface compatibility)
func (it *StreamingProjectionIterator) States() map[string]any {
	return it.GetStates()
}

// AppendCondition returns the current append condition
func (it *StreamingProjectionIterator) AppendCondition() AppendCondition {
	it.mu.RLock()
	defer it.mu.RUnlock()
	return it.appendCondition
}

// GetPosition returns the current position
func (it *StreamingProjectionIterator) GetPosition() int64 {
	it.mu.RLock()
	defer it.mu.RUnlock()
	return it.position
}

// GetProcessedCount returns the number of events processed
func (it *StreamingProjectionIterator) GetProcessedCount() int {
	it.mu.RLock()
	defer it.mu.RUnlock()
	return it.processedCount
}

// eventMatchesProjector checks if an event matches a projector's query criteria
func (it *StreamingProjectionIterator) eventMatchesProjector(event Event, projector StateProjector) bool {
	if len(projector.Query.Items) == 0 {
		return true // Empty query matches all events
	}

	for _, item := range projector.Query.Items {
		if it.eventMatchesQueryItem(event, item) {
			return true
		}
	}
	return false
}

// eventMatchesQueryItem checks if an event matches a specific query item
func (it *StreamingProjectionIterator) eventMatchesQueryItem(event Event, item QueryItem) bool {
	// Check event types
	if len(item.EventTypes) > 0 {
		typeMatch := false
		for _, eventType := range item.EventTypes {
			if event.Type == eventType {
				typeMatch = true
				break
			}
		}
		if !typeMatch {
			return false
		}
	}

	// Check tags
	if len(item.Tags) > 0 {
		for _, queryTag := range item.Tags {
			tagMatch := false
			for _, eventTag := range event.Tags {
				if eventTag.Key == queryTag.Key && eventTag.Value == queryTag.Value {
					tagMatch = true
					break
				}
			}
			if !tagMatch {
				return false
			}
		}
	}

	return true
}

// buildAppendConditionFromProjectors builds an AppendCondition from projector queries
// This ensures that when appending new events, we check that no conflicting events
// have been added since we read the current state
func (es *eventStore) buildAppendConditionFromProjectors(projectors []BatchProjector) AppendCondition {
	// Combine all projector queries into a single OR query
	combinedQuery := es.combineProjectorQueries(projectors)

	// The AppendCondition should fail if any events match the combined projector queries
	// after the current position (which will be updated as events are processed)
	return AppendCondition{
		FailIfEventsMatch: combinedQuery,
		After:             nil, // Will be set to current position when processing completes
	}
}
