package dcb

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// EventStream provides a channel-based streaming interface for events.
// This is part of the ChannelEventStore extension interface.
type EventStream struct {
	rows pgx.Rows
	ch   chan Event
	err  error
	ctx  context.Context
}

// NewEventStream creates a new EventStream for the given query.
// This method is part of the ChannelEventStore interface.
func (es *eventStore) NewEventStream(ctx context.Context, query Query, options *ReadOptions) (*EventStream, error) {
	// Build the SQL query
	sqlQuery, args, err := es.buildReadQuerySQL(query, options)
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	// Execute the query
	rows, err := es.pool.Query(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	stream := &EventStream{
		rows: rows,
		ch:   make(chan Event, 100), // Buffer size of 100
		ctx:  ctx,
	}

	// Start streaming events in a goroutine
	go stream.streamEvents()

	return stream, nil
}

// Events returns the channel to receive streamed events.
func (s *EventStream) Events() <-chan Event {
	return s.ch
}

// Close closes the stream and underlying resources.
func (s *EventStream) Close() error {
	if s.rows != nil {
		s.rows.Close()
	}
	close(s.ch)
	return s.err
}

// streamEvents processes rows and sends them to the channel.
func (s *EventStream) streamEvents() {
	defer s.rows.Close()
	defer close(s.ch)

	for s.rows.Next() {
		select {
		case <-s.ctx.Done():
			s.err = s.ctx.Err()
			return
		default:
			var row rowEvent
			err := s.rows.Scan(
				&row.ID,
				&row.Type,
				&row.Tags,
				&row.Data,
				&row.Position,
				&row.CausationID,
				&row.CorrelationID,
			)
			if err != nil {
				s.err = fmt.Errorf("scan failed: %w", err)
				continue
			}

			event := convertRowToEvent(row)
			s.ch <- event
		}
	}

	if err := s.rows.Err(); err != nil {
		s.err = fmt.Errorf("row iteration failed: %w", err)
	}
}

// ReadStreamChannel creates a channel-based stream of events matching a query.
// This method is part of the ChannelEventStore interface.
// It's optimized for small to medium datasets (< 500 events).
func (es *eventStore) ReadStreamChannel(ctx context.Context, query Query, options *ReadOptions) (<-chan Event, error) {
	stream, err := es.NewEventStream(ctx, query, options)
	if err != nil {
		return nil, err
	}

	// Return the channel directly for simpler usage
	return stream.Events(), nil
}

// Ensure eventStore implements ChannelEventStore interface
var _ ChannelEventStore = (*eventStore)(nil)

// ProjectDecisionModelChannel projects multiple states using channel-based streaming
// This is optimized for small to medium datasets (< 500 events) and provides
// a more Go-idiomatic interface using channels for state projection
func (es *eventStore) ProjectDecisionModelChannel(ctx context.Context, projectors []BatchProjector, options *ReadOptions) (<-chan ProjectionResult, error) {
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

	// Create event stream
	eventStream, err := es.NewEventStream(ctx, query, options)
	if err != nil {
		return nil, fmt.Errorf("failed to create event stream: %w", err)
	}

	// Create result channel
	resultChan := make(chan ProjectionResult, 100)

	// Start projection processing in a goroutine
	go func() {
		defer eventStream.Close()
		defer close(resultChan)

		// Initialize projector states
		projectorStates := make(map[string]interface{})
		for _, projector := range projectors {
			projectorStates[projector.ID] = projector.StateProjector.InitialState
		}

		// Process events
		for event := range eventStream.Events() {
			select {
			case <-ctx.Done():
				resultChan <- ProjectionResult{
					Error: ctx.Err(),
				}
				return
			default:
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

		// Check for stream errors
		if eventStream.err != nil {
			resultChan <- ProjectionResult{
				Error: fmt.Errorf("event stream error: %w", eventStream.err),
			}
		}
	}()

	return resultChan, nil
}
