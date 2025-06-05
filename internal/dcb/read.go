package dcb

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx/v5/pgtype"
)

// rowEvent is a helper struct for scanning database rows.
type rowEvent struct {
	ID            pgtype.UUID
	Type          string
	Tags          []byte
	Data          []byte
	Position      int64
	CausationID   pgtype.UUID
	CorrelationID pgtype.UUID
}

// convertRowToEvent converts a database row to an Event
func convertRowToEvent(row rowEvent) Event {
	var e Event
	if !row.ID.Valid {
		panic(fmt.Sprintf("invalid UUID for id at position %d", row.Position))
	}
	e.ID = row.ID.String()
	e.Type = row.Type
	var tagMap map[string]string
	if err := json.Unmarshal(row.Tags, &tagMap); err != nil {
		panic(fmt.Sprintf("failed to unmarshal tags at position %d: %v", row.Position, err))
	}
	for k, v := range tagMap {
		e.Tags = append(e.Tags, Tag{Key: k, Value: v})
	}
	e.Data = row.Data
	e.Position = row.Position
	if row.CausationID.Valid {
		e.CausationID = row.CausationID.String()
	}
	if row.CorrelationID.Valid {
		e.CorrelationID = row.CorrelationID.String()
	}
	return e
}

// ReadState computes a state by streaming events matching the query, up to maxPosition.
func (es *eventStore) ProjectState(ctx context.Context, query Query, stateReducer StateReducer) (int64, any, error) {
	return es.ProjectStateUpTo(ctx, query, stateReducer, -1)
}

// ReadStateUpTo computes a state by streaming events matching the query, up to maxPosition.
func (es *eventStore) ProjectStateUpTo(ctx context.Context, query Query, stateReducer StateReducer, maxPosition int64) (int64, any, error) {
	if stateReducer.ReducerFn == nil {
		return 0, stateReducer.InitialState, &ValidationError{
			EventStoreError: EventStoreError{
				Op:  "ProjectStateUpTo",
				Err: fmt.Errorf("reducer function cannot be nil"),
			},
			Field: "reducer",
			Value: "nil",
		}
	}

	// Build JSONB query condition with proper error handling
	tagMap := make(map[string]string)
	for _, t := range query.Tags {
		tagMap[t.Key] = t.Value
	}
	queryTags, err := json.Marshal(tagMap)
	if err != nil {
		return 0, stateReducer.InitialState, &EventStoreError{
			Op:  "ProjectStateUpTo",
			Err: fmt.Errorf("failed to marshal query tags %v: %w", tagMap, err),
		}
	}

	// Construct SQL query with proper error context
	sqlQuery := "SELECT id, type, tags, data, position, causation_id, correlation_id FROM events WHERE tags @> $1"
	args := []interface{}{queryTags}

	// Add event type filtering if specified
	if len(query.EventTypes) > 0 {
		sqlQuery += fmt.Sprintf(" AND type = ANY($%d)", len(args)+1)
		args = append(args, query.EventTypes)
	}

	// Add position filtering if maxPosition is specified
	// If maxPosition is -1, it means no limit
	// If maxPosition is 0, it means no events should be returned
	// If maxPosition is greater than 0, we filter events up to that position
	if maxPosition >= 0 {
		sqlQuery += fmt.Sprintf(" AND position <= $%d", len(args)+1)
		args = append(args, maxPosition)
	}

	// Query and stream rows with proper error handling
	rows, err := es.pool.Query(ctx, sqlQuery, args...)
	if err != nil {
		return 0, stateReducer.InitialState, &ResourceError{
			EventStoreError: EventStoreError{
				Op:  "ProjectStateUpTo",
				Err: fmt.Errorf("failed to execute query for tags %v: %w", tagMap, err),
			},
			Resource: "database",
		}
	}
	defer rows.Close()

	// Initialize state
	state := stateReducer.InitialState
	position := int64(0)

	// Process events with proper error handling
	for rows.Next() {
		var row rowEvent
		if err := rows.Scan(&row.ID, &row.Type, &row.Tags, &row.Data, &row.Position, &row.CausationID, &row.CorrelationID); err != nil {
			return 0, stateReducer.InitialState, &ResourceError{
				EventStoreError: EventStoreError{
					Op:  "ProjectStateUpTo",
					Err: fmt.Errorf("failed to scan event row at position %d: %w", position, err),
				},
				Resource: "database",
			}
		}

		// Convert row to Event with panic recovery
		var event Event
		func() {
			defer func() {
				if r := recover(); r != nil {
					err = &EventStoreError{
						Op:  "ProjectStateUpTo",
						Err: fmt.Errorf("panic converting row to event at position %d: %v", row.Position, r),
					}
				}
			}()
			event = convertRowToEvent(row)
		}()
		if err != nil {
			return 0, stateReducer.InitialState, err
		}

		// Apply reducer with panic recovery
		func() {
			defer func() {
				if r := recover(); r != nil {
					err = &EventStoreError{
						Op:  "ProjectStateUpTo",
						Err: fmt.Errorf("panic in reducer for event type %s at position %d: %v", event.Type, event.Position, r),
					}
				}
			}()
			state = stateReducer.ReducerFn(state, event)
		}()
		if err != nil {
			return 0, stateReducer.InitialState, err
		}

		position = row.Position
	}

	// Check for errors from iterating over rows
	if err := rows.Err(); err != nil {
		return 0, stateReducer.InitialState, &ResourceError{
			EventStoreError: EventStoreError{
				Op:  "ProjectStateUpTo",
				Err: fmt.Errorf("error iterating over events: %w", err),
			},
			Resource: "database",
		}
	}

	return position, state, nil
}
