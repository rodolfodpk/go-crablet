package dcb

import (
	"context"
	"fmt"
	"log"
)

// ReadStreamChannel creates a channel-based stream of events matching a query
// This is optimized for small to medium datasets (< 500 events) and provides
// a more Go-idiomatic interface using channels
func (es *eventStore) ReadStreamChannel(ctx context.Context, query Query) (<-chan Event, *Cursor, error) {
	// Validate that the query is not empty (same validation as Read method)
	if len(query.getItems()) == 0 {
		return nil, nil, &ValidationError{
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
		return nil, nil, fmt.Errorf("failed to build query: %w", err)
	}

	// Execute the query
	rows, err := es.pool.Query(ctx, sqlQuery, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("query failed: %w", err)
	}

	// Create result channel
	resultChan := make(chan Event, 100)

	// Track latest cursor
	var latestCursor *Cursor

	// Start streaming events in a goroutine
	go func() {
		// Ensure rows are always closed, even if goroutine panics
		defer func() {
			if r := recover(); r != nil {
				log.Printf("ReadStreamChannel panic recovered: %v", r)
			}
			rows.Close()
			close(resultChan)
		}()

		for rows.Next() {
			select {
			case <-ctx.Done():
				// Context cancelled - exit cleanly
				return
			case resultChan <- func() Event {
				var row rowEvent
				err := rows.Scan(
					&row.Type,
					&row.Tags,
					&row.Data,
					&row.Position,
					&row.TransactionID,
					&row.CreatedAt,
				)
				if err != nil {
					// Log error but continue processing
					log.Printf("Error scanning row in ReadStreamChannel: %v", err)
					return Event{} // Return empty event, will be filtered out
				}

				// Update latest cursor (events are ordered by transaction_id ASC, position ASC)
				latestCursor = &Cursor{
					TransactionID: row.TransactionID,
					Position:      row.Position,
				}

				return convertRowToEvent(row)
			}():
				// Event sent successfully
			}
		}

		// Check for row iteration errors
		if err := rows.Err(); err != nil {
			log.Printf("Row iteration error in ReadStreamChannel: %v", err)
		}
	}()

	return resultChan, latestCursor, nil
}

// ProjectDecisionModelChannel projects multiple states using channel-based streaming
// This is optimized for small to medium datasets (< 500 events) and provides
// a more Go-idiomatic interface using channels for state projection
func (es *eventStore) ProjectDecisionModelChannel(ctx context.Context, projectors []BatchProjector) (<-chan ProjectionResult, *Cursor, error) {
	if len(projectors) == 0 {
		return nil, nil, fmt.Errorf("at least one projector is required")
	}

	// Validate projectors
	for _, bp := range projectors {
		if bp.StateProjector.TransitionFn == nil {
			return nil, nil, &ValidationError{
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
	if len(query.getItems()) == 0 {
		return nil, nil, &ValidationError{
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
		return nil, nil, fmt.Errorf("failed to build query: %w", err)
	}

	// Execute the query
	rows, err := es.pool.Query(ctx, sqlQuery, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("query failed: %w", err)
	}

	// Create result channel
	resultChan := make(chan ProjectionResult, 100)

	// Track latest cursor
	var latestCursor *Cursor

	// Start projection processing in a goroutine
	go func() {
		// Ensure rows are always closed, even if goroutine panics
		defer func() {
			if r := recover(); r != nil {
				log.Printf("ProjectDecisionModelChannel panic recovered: %v", r)
			}
			rows.Close()
			close(resultChan)
		}()

		// Initialize projector states
		projectorStates := make(map[string]interface{})
		for _, projector := range projectors {
			projectorStates[projector.ID] = projector.StateProjector.InitialState
		}

		// Process events
		for rows.Next() {
			select {
			case <-ctx.Done():
				// Context cancelled - send error and exit cleanly
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
					&row.TransactionID,
					&row.CreatedAt,
				)
				if err != nil {
					// Log error but continue processing
					log.Printf("Error scanning row in ProjectDecisionModelChannel: %v", err)
					continue
				}

				// Update latest cursor (events are ordered by transaction_id ASC, position ASC)
				latestCursor = &Cursor{
					TransactionID: row.TransactionID,
					Position:      row.Position,
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

					// Send result (non-blocking to avoid deadlocks)
					select {
					case resultChan <- ProjectionResult{
						ProjectorID: projector.ID,
						State:       newState,
					}:
						// Result sent successfully
					case <-ctx.Done():
						// Context cancelled while trying to send
						return
					}
				}
			}
		}

		// Check for row iteration errors
		if err := rows.Err(); err != nil {
			log.Printf("Row iteration error in ProjectDecisionModelChannel: %v", err)
			select {
			case resultChan <- ProjectionResult{
				Error: fmt.Errorf("row iteration failed: %w", err),
			}:
			case <-ctx.Done():
				// Context cancelled while trying to send error
			}
		}
	}()

	return resultChan, latestCursor, nil
}

// Append now returns only error, not position.
