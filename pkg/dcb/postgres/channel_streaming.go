package postgres

import (
	"context"
	"fmt"

	"go-crablet/pkg/dcb"

	"github.com/jackc/pgx/v5/pgxpool"
)

// channelEventStore extends eventStore with channel-based streaming capabilities
type channelEventStore struct {
	*eventStore
}

// NewChannelEventStore creates a new channel event store instance
func NewChannelEventStore(ctx context.Context, pool *pgxpool.Pool) (dcb.ChannelEventStore, error) {
	baseStore, err := NewEventStore(ctx, pool)
	if err != nil {
		return nil, err
	}

	es, ok := baseStore.(*eventStore)
	if !ok {
		return nil, fmt.Errorf("failed to cast to eventStore")
	}

	return &channelEventStore{
		eventStore: es,
	}, nil
}

// ReadStreamChannel creates a channel-based stream of events matching a query
// This is optimized for small to medium datasets (< 500 events) and provides
// a more Go-idiomatic interface using channels
func (ces *channelEventStore) ReadStreamChannel(ctx context.Context, query dcb.Query) (<-chan dcb.Event, error) {
	// Validate query
	if err := ces.validateQueryTags(query); err != nil {
		return nil, &dcb.EventStoreError{
			Op:  "readStreamChannel",
			Err: fmt.Errorf("invalid query: %w", err),
		}
	}

	// Create channel
	eventChan := make(chan dcb.Event, 100) // Buffered channel for better performance

	// Start goroutine to stream events
	go func() {
		defer close(eventChan)

		// Build the SQL query
		sqlQuery, args, err := ces.buildReadQuerySQL(query, nil)
		if err != nil {
			// We can't return error through channel, so we'll just return
			return
		}

		// Execute the query
		rows, err := ces.pool.Query(ctx, sqlQuery, args...)
		if err != nil {
			return
		}
		defer rows.Close()

		// Stream results
		for rows.Next() {
			var eventType string
			var tagsArray []string
			var data []byte
			var position int64

			err := rows.Scan(&eventType, &tagsArray, &data, &position)
			if err != nil {
				return
			}

			event := dcb.Event{
				Type:     eventType,
				Tags:     dcb.ParseTagsArray(tagsArray),
				Data:     data,
				Position: position,
			}

			// Send event to channel
			select {
			case eventChan <- event:
			case <-ctx.Done():
				return
			}
		}

		if err := rows.Err(); err != nil {
			return
		}
	}()

	return eventChan, nil
}

// ProjectDecisionModelChannel projects multiple states using channel-based streaming
// This is optimized for small to medium datasets (< 500 events) and provides
// a more Go-idiomatic interface using channels for state projection
func (ces *channelEventStore) ProjectDecisionModelChannel(ctx context.Context, projectors []dcb.BatchProjector) (<-chan dcb.ProjectionResult, error) {
	if len(projectors) == 0 {
		return nil, &dcb.ValidationError{
			EventStoreError: dcb.EventStoreError{
				Op:  "projectDecisionModelChannel",
				Err: fmt.Errorf("projectors must not be empty"),
			},
			Field: "projectors",
			Value: "empty",
		}
	}

	// Validate projectors
	for i, projector := range projectors {
		if projector.ID == "" {
			return nil, &dcb.ValidationError{
				EventStoreError: dcb.EventStoreError{
					Op:  "projectDecisionModelChannel",
					Err: fmt.Errorf("projector at index %d has empty ID", i),
				},
				Field: "projector.id",
				Value: "empty",
			}
		}

		if projector.StateProjector.TransitionFn == nil {
			return nil, &dcb.ValidationError{
				EventStoreError: dcb.EventStoreError{
					Op:  "projectDecisionModelChannel",
					Err: fmt.Errorf("projector at index %d has nil transition function", i),
				},
				Field: "projector.transitionFn",
				Value: "nil",
			}
		}

		// Validate query
		if err := ces.validateQueryTags(projector.StateProjector.Query); err != nil {
			return nil, &dcb.EventStoreError{
				Op:  "projectDecisionModelChannel",
				Err: fmt.Errorf("invalid query in projector %s: %w", projector.ID, err),
			}
		}
	}

	// Create channel
	resultChan := make(chan dcb.ProjectionResult, 100) // Buffered channel for better performance

	// Start goroutine to stream projection results
	go func() {
		defer close(resultChan)

		// Process each projector
		for _, projector := range projectors {
			query := projector.StateProjector.Query
			state := projector.StateProjector.InitialState

			// Get event stream for this projector
			eventChan, err := ces.ReadStreamChannel(ctx, query)
			if err != nil {
				// Send error result
				select {
				case resultChan <- dcb.ProjectionResult{
					ProjectorID: projector.ID,
					State:       state,
					Event:       dcb.Event{},
					Position:    0,
					Error:       err,
				}:
				case <-ctx.Done():
					return
				}
				continue
			}

			// Process events
			for event := range eventChan {
				// Apply transition function
				newState := projector.StateProjector.TransitionFn(state, event)
				state = newState

				// Send result
				select {
				case resultChan <- dcb.ProjectionResult{
					ProjectorID: projector.ID,
					State:       state,
					Event:       event,
					Position:    event.Position,
					Error:       nil,
				}:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return resultChan, nil
}
