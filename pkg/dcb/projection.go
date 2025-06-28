package dcb

import (
	"context"
	"fmt"
)

// ProjectDecisionModel projects multiple states using projectors and returns final states and append condition
// This is a go-crablet feature for building decision models in command handlers
// The function internally computes the combined query from all projectors for the append condition
func (es *eventStore) ProjectDecisionModel(ctx context.Context, projectors []BatchProjector) (map[string]any, AppendCondition, error) {
	// Validate projectors
	for _, bp := range projectors {
		if bp.ID == "" {
			return nil, nil, &ValidationError{
				EventStoreError: EventStoreError{
					Op:  "ProjectDecisionModel",
					Err: fmt.Errorf("projector ID cannot be empty"),
				},
				Field: "projector.id",
				Value: "empty",
			}
		}
		if bp.StateProjector.TransitionFn == nil {
			return nil, nil, &ValidationError{
				EventStoreError: EventStoreError{
					Op:  "ProjectDecisionModel",
					Err: fmt.Errorf("projector %s has nil transition function", bp.ID),
				},
				Field: "transitionFn",
				Value: "nil",
			}
		}
		if len(bp.StateProjector.Query.getItems()) == 0 {
			return nil, nil, &ValidationError{
				EventStoreError: EventStoreError{
					Op:  "ProjectDecisionModel",
					Err: fmt.Errorf("projector %s has empty query", bp.ID),
				},
				Field: "query",
				Value: "empty",
			}
		}
	}

	// Build combined query from all projectors
	query := es.combineProjectorQueries(projectors)

	// Use query-based approach for all datasets
	return es.projectDecisionModelWithQuery(ctx, query, projectors)
}

// projectDecisionModelWithQuery uses query-based approach for all datasets
func (es *eventStore) projectDecisionModelWithQuery(ctx context.Context, query Query, projectors []BatchProjector) (map[string]any, AppendCondition, error) {
	// Build SQL query based on query items
	sqlQuery, args, err := es.buildReadQuerySQL(query, nil)
	if err != nil {
		return nil, nil, err
	}

	// Execute the query
	rows, err := es.pool.Query(ctx, sqlQuery, args...)
	if err != nil {
		return nil, nil, &ResourceError{
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
	appendCondition := es.buildAppendConditionFromQuery(query)

	// Process all events to build final states
	var lastPosition int64
	var hasEvents bool
	for rows.Next() {
		var row rowEvent
		if err := rows.Scan(&row.Type, &row.Tags, &row.Data, &row.Position); err != nil {
			return nil, nil, &ResourceError{
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
		hasEvents = true

		// Update AppendCondition.After field with current position
		appendCondition.setAfterPosition(&lastPosition)

		// Apply projectors
		for _, bp := range projectors {
			if es.eventMatchesProjector(event, bp.StateProjector) {
				states[bp.ID] = bp.StateProjector.TransitionFn(states[bp.ID], event)
			}
		}
	}

	// Only set After field if we actually processed events
	if !hasEvents {
		appendCondition.setAfterPosition(nil)
	}

	// Check for iteration errors
	if err := rows.Err(); err != nil {
		return nil, nil, &ResourceError{
			EventStoreError: EventStoreError{
				Op:  "ProjectDecisionModel",
				Err: fmt.Errorf("error iterating over events: %w", err),
			},
			Resource: "database",
		}
	}

	return states, appendCondition, nil
}

// buildAppendConditionFromQuery builds an AppendCondition from a specific query
// This aligns with DCB specification: each append operation should use the same query
// that was used when building the Decision Model
func (es *eventStore) buildAppendConditionFromQuery(query Query) AppendCondition {
	return NewAppendCondition(query)
}
