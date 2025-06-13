package dcb

import (
	"context"
	"fmt"
)

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

	// Check if we should use cursor-based streaming
	if options != nil && options.BatchSize != nil && *options.BatchSize > 0 {
		return es.projectDecisionModelWithCursor(ctx, query, options, projectors)
	}

	// Use the original approach for small datasets
	return es.projectDecisionModelWithQuery(ctx, query, options, projectors)
}

// projectDecisionModelWithCursor uses cursor-based streaming for large datasets
func (es *eventStore) projectDecisionModelWithCursor(ctx context.Context, query Query, options *ReadOptions, projectors []BatchProjector) (map[string]any, AppendCondition, error) {
	// Initialize states for all projectors
	states := make(map[string]any)
	for _, bp := range projectors {
		states[bp.ID] = bp.StateProjector.InitialState
	}

	// Build AppendCondition from projector queries for optimistic locking
	appendCondition := es.buildAppendConditionFromProjectors(projectors)

	// Use ReadStream for cursor-based processing
	iterator, err := es.ReadStream(ctx, query, options)
	if err != nil {
		return nil, AppendCondition{}, err
	}
	defer iterator.Close()

	// Process events using the streaming iterator
	var lastPosition int64
	for iterator.Next() {
		event := iterator.Event()
		lastPosition = event.Position

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
	if err := iterator.Err(); err != nil {
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

// projectDecisionModelWithQuery uses the original query-based approach for small datasets
func (es *eventStore) projectDecisionModelWithQuery(ctx context.Context, query Query, options *ReadOptions, projectors []BatchProjector) (map[string]any, AppendCondition, error) {
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

// buildAppendConditionFromProjectors builds an AppendCondition from projector queries
// This ensures that when appending new events, we check that no conflicting events
// have been added since we read the current state
func (es *eventStore) buildAppendConditionFromProjectors(projectors []BatchProjector) AppendCondition {
	// Combine all projector queries into a single OR query
	combinedQuery := es.combineProjectorQueries(projectors)

	// The AppendCondition should fail if any events match the combined projector queries
	// after the current position (which will be updated as events are processed)
	return AppendCondition{
		FailIfEventsMatch: &combinedQuery,
		After:             nil, // Will be set to current position when processing completes
	}
}
