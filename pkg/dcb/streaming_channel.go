package dcb

import (
	"context"
	"fmt"
)

// ReadStreamChannel creates a channel-based stream of events matching a query.
// This method is part of the CrabletEventStore interface.
// It's optimized for small to medium datasets (< 500 events).
func (es *eventStore) ReadStreamChannel(ctx context.Context, query Query) (<-chan Event, error) {
	// Validate that the query is not empty (same validation as Read method)
	if len(query.Items) == 0 {
		return nil, &ValidationError{
			EventStoreError: EventStoreError{
				Op:  "ReadStreamChannel",
				Err: fmt.Errorf("query must contain at least one item"),
			},
			Field: "query",
			Value: "empty",
		}
	}

	// Build the SQL query
	sqlQuery, args, err := es.buildReadQuerySQL(query, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	// Execute the query
	rows, err := es.pool.Query(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	// Create result channel
	resultChan := make(chan Event, 100)

	// Start streaming events in a goroutine
	go func() {
		defer rows.Close()
		defer close(resultChan)

		for rows.Next() {
			select {
			case <-ctx.Done():
				return
			default:
				var row rowEvent
				err := rows.Scan(
					&row.Type,
					&row.Tags,
					&row.Data,
					&row.Position,
				)
				if err != nil {
					// Log error but continue processing
					continue
				}

				event := convertRowToEvent(row)
				resultChan <- event
			}
		}

		if err := rows.Err(); err != nil {
			// Log error but don't send it through channel
			// The channel will be closed normally
		}
	}()

	return resultChan, nil
}

// Ensure eventStore implements CrabletEventStore interface
var _ CrabletEventStore = (*eventStore)(nil)

// ProjectDecisionModelChannel projects multiple states using channel-based streaming
// This is optimized for small to medium datasets (< 500 events) and provides
// a more Go-idiomatic interface using channels for state projection
func (es *eventStore) ProjectDecisionModelChannel(ctx context.Context, projectors []BatchProjector) (<-chan ProjectionResult, error) {
	if len(projectors) == 0 {
		return nil, fmt.Errorf("at least one projector is required")
	}

	// Validate projectors
	for _, bp := range projectors {
		if bp.StateProjector.TransitionFn == nil {
			return nil, &ValidationError{
				EventStoreError: EventStoreError{
					Op:  "ProjectDecisionModelChannel",
					Err: fmt.Errorf("projector %s has nil transition function", bp.ID),
				},
				Field: "transitionFn",
				Value: "nil",
			}
		}
	}

	// Build combined query from all projectors
	query := es.combineProjectorQueries(projectors)

	// Validate that the combined query is not empty (same validation as Read method)
	if len(query.Items) == 0 {
		return nil, &ValidationError{
			EventStoreError: EventStoreError{
				Op:  "ProjectDecisionModelChannel",
				Err: fmt.Errorf("query must contain at least one item"),
			},
			Field: "query",
			Value: "empty",
		}
	}

	// Build the SQL query
	sqlQuery, args, err := es.buildReadQuerySQL(query, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	// Execute the query
	rows, err := es.pool.Query(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	// Create result channel
	resultChan := make(chan ProjectionResult, 100)

	// Start projection processing in a goroutine
	go func() {
		defer rows.Close()
		defer close(resultChan)

		// Initialize projector states
		projectorStates := make(map[string]interface{})
		for _, projector := range projectors {
			projectorStates[projector.ID] = projector.StateProjector.InitialState
		}

		// Process events
		for rows.Next() {
			select {
			case <-ctx.Done():
				resultChan <- ProjectionResult{
					Error: ctx.Err(),
				}
				return
			default:
				var row rowEvent
				err := rows.Scan(
					&row.Type,
					&row.Tags,
					&row.Data,
					&row.Position,
				)
				if err != nil {
					// Log error but continue processing
					continue
				}

				event := convertRowToEvent(row)

				// Process event with each projector
				for _, projector := range projectors {
					// Check if projector should process this event
					if !es.eventMatchesProjector(event, projector.StateProjector) {
						continue
					}

					// Get current state for this projector
					currentState := projectorStates[projector.ID]

					// Project the event using the transition function
					newState := projector.StateProjector.TransitionFn(currentState, event)

					// Update state
					projectorStates[projector.ID] = newState

					// Send result
					resultChan <- ProjectionResult{
						ProjectorID: projector.ID,
						State:       newState,
						Event:       event,
						Position:    event.Position,
					}
				}
			}
		}

		// Check for row iteration errors
		if err := rows.Err(); err != nil {
			resultChan <- ProjectionResult{
				Error: fmt.Errorf("row iteration failed: %w", err),
			}
		}
	}()

	return resultChan, nil
}
