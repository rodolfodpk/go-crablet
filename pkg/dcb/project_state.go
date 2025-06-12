package dcb

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// rowEvent is a helper struct for scanning database rows.
type rowEvent struct {
	ID            string
	Type          string
	Tags          []byte
	Data          []byte
	Position      int64
	CausationID   string
	CorrelationID string
}

// convertRowToEvent converts a database row to an Event
func convertRowToEvent(row rowEvent) Event {
	var e Event
	e.ID = row.ID
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
	e.CausationID = row.CausationID
	e.CorrelationID = row.CorrelationID
	return e
}

// ProjectBatch projects multiple states using multiple projectors in a single database query.
func (es *eventStore) ProjectBatch(ctx context.Context, projectors []BatchProjector) (BatchProjectionResult, error) {
	return es.ProjectBatchUpTo(ctx, projectors, -1)
}

// ProjectBatchUpTo projects multiple states up to a specific position using multiple projectors.
func (es *eventStore) ProjectBatchUpTo(ctx context.Context, projectors []BatchProjector, maxPosition int64) (BatchProjectionResult, error) {
	if len(projectors) == 0 {
		return BatchProjectionResult{Position: 0, States: make(map[string]any)}, nil
	}

	// Validate all projectors have transition functions
	for _, bp := range projectors {
		if bp.StateProjector.TransitionFn == nil {
			return BatchProjectionResult{}, &ValidationError{
				EventStoreError: EventStoreError{
					Op:  "ProjectBatchUpTo",
					Err: fmt.Errorf("projector %s has nil transition function", bp.ID),
				},
				Field: "projector",
				Value: bp.ID,
			}
		}
	}

	// Combine all projector queries into a single OR query
	combinedQuery := es.combineProjectorQueries(projectors)

	// Build the combined SQL query
	sqlQuery, args, err := es.buildCombinedQuerySQL(combinedQuery, maxPosition)
	if err != nil {
		return BatchProjectionResult{}, err
	}

	// Execute the combined query
	rows, err := es.pool.Query(ctx, sqlQuery, args...)
	if err != nil {
		return BatchProjectionResult{}, &ResourceError{
			EventStoreError: EventStoreError{
				Op:  "ProjectBatchUpTo",
				Err: fmt.Errorf("failed to execute combined query: %w", err),
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

	position := int64(0)

	// Process events and route to appropriate projectors
	for rows.Next() {
		var row rowEvent
		if err := rows.Scan(&row.ID, &row.Type, &row.Tags, &row.Data, &row.Position, &row.CausationID, &row.CorrelationID); err != nil {
			return BatchProjectionResult{}, &ResourceError{
				EventStoreError: EventStoreError{
					Op:  "ProjectBatchUpTo",
					Err: fmt.Errorf("failed to scan event row at position %d: %w", position, err),
				},
				Resource: "database",
			}
		}

		// Convert row to Event
		var event Event
		func() {
			defer func() {
				if r := recover(); r != nil {
					err = &EventStoreError{
						Op:  "ProjectBatchUpTo",
						Err: fmt.Errorf("panic converting row to event at position %d: %v", row.Position, r),
					}
				}
			}()
			event = convertRowToEvent(row)
		}()
		if err != nil {
			return BatchProjectionResult{}, err
		}

		// Route event to matching projectors
		for _, bp := range projectors {
			if es.eventMatchesProjector(event, bp.StateProjector) {
				// Apply projector with panic recovery
				func() {
					defer func() {
						if r := recover(); r != nil {
							err = &EventStoreError{
								Op:  "ProjectBatchUpTo",
								Err: fmt.Errorf("panic in projector %s for event type %s at position %d: %v", bp.ID, event.Type, event.Position, r),
							}
						}
					}()
					states[bp.ID] = bp.StateProjector.TransitionFn(states[bp.ID], event)
				}()
				if err != nil {
					return BatchProjectionResult{}, err
				}
			}
		}

		position = row.Position
	}

	// Check for errors from iterating over rows
	if err := rows.Err(); err != nil {
		return BatchProjectionResult{}, &ResourceError{
			EventStoreError: EventStoreError{
				Op:  "ProjectBatchUpTo",
				Err: fmt.Errorf("error iterating over events: %w", err),
			},
			Resource: "database",
		}
	}

	return BatchProjectionResult{Position: position, States: states}, nil
}

// combineProjectorQueries combines multiple projector queries into a single OR query
func (es *eventStore) combineProjectorQueries(projectors []BatchProjector) Query {
	var combinedItems []QueryItem

	for _, bp := range projectors {
		// Add all items from this projector's query
		for _, item := range bp.StateProjector.Query.Items {
			combinedItems = append(combinedItems, item)
		}
	}

	return Query{Items: combinedItems}
}

// buildCombinedQuerySQL builds the SQL query for the combined projector queries
func (es *eventStore) buildCombinedQuerySQL(query Query, maxPosition int64) (string, []interface{}, error) {
	if len(query.Items) == 0 {
		// Empty query matches all events
		sqlQuery := "SELECT id, type, tags, data, position, causation_id, correlation_id FROM events"
		args := []interface{}{}

		if maxPosition >= 0 {
			sqlQuery += " WHERE position <= $1"
			args = append(args, maxPosition)
		}

		sqlQuery += " ORDER BY position ASC"
		return sqlQuery, args, nil
	}

	// Build OR conditions for each query item
	var conditions []string
	var args []interface{}

	for i, item := range query.Items {
		var condition string
		argIndex := len(args) + 1

		// Build tag condition
		tagMap := make(map[string]string)
		for _, t := range item.Tags {
			tagMap[t.Key] = t.Value
		}
		queryTags, err := json.Marshal(tagMap)
		if err != nil {
			return "", nil, &EventStoreError{
				Op:  "buildCombinedQuerySQL",
				Err: fmt.Errorf("failed to marshal query tags for item %d: %w", i, err),
			}
		}

		condition = fmt.Sprintf("tags @> $%d", argIndex)
		args = append(args, queryTags)

		// Add event type filtering if specified
		if len(item.EventTypes) > 0 {
			argIndex = len(args) + 1
			condition += fmt.Sprintf(" AND type = ANY($%d)", argIndex)
			args = append(args, item.EventTypes)
		}

		conditions = append(conditions, condition)
	}

	// Combine conditions with OR
	sqlQuery := fmt.Sprintf("SELECT id, type, tags, data, position, causation_id, correlation_id FROM events WHERE (%s)", strings.Join(conditions, " OR "))

	// Add position filtering if specified
	if maxPosition >= 0 {
		argIndex := len(args) + 1
		sqlQuery += fmt.Sprintf(" AND position <= $%d", argIndex)
		args = append(args, maxPosition)
	}

	sqlQuery += " ORDER BY position ASC"
	return sqlQuery, args, nil
}

// eventMatchesProjector checks if an event matches a projector's query criteria
func (es *eventStore) eventMatchesProjector(event Event, projector StateProjector) bool {
	if len(projector.Query.Items) == 0 {
		return true // Empty query matches all events
	}

	// Check if event matches any of the query items (OR logic)
	for _, item := range projector.Query.Items {
		if es.eventMatchesQueryItem(event, item) {
			return true
		}
	}

	return false
}

// eventMatchesQueryItem checks if an event matches a specific query item
func (es *eventStore) eventMatchesQueryItem(event Event, item QueryItem) bool {
	// Check event type filtering
	if len(item.EventTypes) > 0 {
		typeMatches := false
		for _, eventType := range item.EventTypes {
			if event.Type == eventType {
				typeMatches = true
				break
			}
		}
		if !typeMatches {
			return false
		}
	}

	// Check tag filtering
	if len(item.Tags) > 0 {
		// Convert event tags to map for efficient lookup
		eventTagMap := make(map[string]string)
		for _, tag := range event.Tags {
			eventTagMap[tag.Key] = tag.Value
		}

		// Check if all required tags are present
		for _, requiredTag := range item.Tags {
			if eventTagMap[requiredTag.Key] != requiredTag.Value {
				return false
			}
		}
	}

	return true
}

// CombineProjectorQueries combines multiple projector queries into a single OR query.
// This is useful for creating AppendCondition queries that ensure consistency
// across all projectors used in a decision model.
func CombineProjectorQueries(projectors []BatchProjector) Query {
	var combinedItems []QueryItem

	for _, bp := range projectors {
		// Add all items from this projector's query
		for _, item := range bp.StateProjector.Query.Items {
			combinedItems = append(combinedItems, item)
		}
	}

	return Query{Items: combinedItems}
}

// SingleProjector creates a BatchProjector for single projector usage.
// This provides a clean API for the common case of projecting a single state.
func SingleProjector(id string, projector StateProjector) BatchProjector {
	return BatchProjector{
		ID:             id,
		StateProjector: projector,
	}
}
