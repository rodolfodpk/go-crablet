package dcb

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"
)

// rowEvent is a helper struct for scanning database rows.
type rowEvent struct {
	Type          string
	Tags          []string
	Data          []byte
	Position      int64
	TransactionID uint64
	CreatedAt     time.Time
}

// convertRowToEvent converts a database row to an Event
func convertRowToEvent(row rowEvent) Event {
	return Event{
		Type:          row.Type,
		Tags:          ParseTagsArray(row.Tags),
		Data:          row.Data,
		Position:      row.Position,
		TransactionID: row.TransactionID,
		CreatedAt:     row.CreatedAt,
	}
}

// buildReadQuerySQL builds the SQL query for reading events
func (es *eventStore) buildReadQuerySQL(query Query, options ReadOptions) (string, []interface{}, error) {
	// Pre-allocate slices with reasonable capacity
	conditions := make([]string, 0, 4) // Usually 1-4 conditions
	args := make([]interface{}, 0, 8)  // Usually 2-8 args
	argIndex := 1

	// Add query conditions
	if len(query.getItems()) > 0 {
		orConditions := make([]string, 0, len(query.getItems()))

		for _, item := range query.getItems() {
			andConditions := make([]string, 0, 2) // Usually 1-2 conditions per item

			// Add event type conditions
			if len(item.getEventTypes()) > 0 {
				andConditions = append(andConditions, fmt.Sprintf("type = ANY($%d::text[])", argIndex))
				args = append(args, item.getEventTypes())
				argIndex++
			}

			// Add tag conditions - use contains operator for DCB semantics
			if len(item.getTags()) > 0 {
				tagsArray := TagsToArray(item.getTags())
				andConditions = append(andConditions, fmt.Sprintf("tags @> $%d::text[]", argIndex))
				args = append(args, tagsArray)
				argIndex++
			}

			// Combine AND conditions for this item
			if len(andConditions) > 0 {
				orConditions = append(orConditions, "("+strings.Join(andConditions, " AND ")+")")
			}
		}

		// Combine OR conditions for all items
		if len(orConditions) > 0 {
			conditions = append(conditions, "("+strings.Join(orConditions, " OR ")+")")
		}
	}

	// Add cursor conditions (replaces FromPosition logic)
	if options.Cursor != nil {
		// Use the correct cursor logic from Oskar's article:
		// (transaction_id = cursor.TransactionID AND position > cursor.Position) OR (transaction_id > cursor.TransactionID)
		conditions = append(conditions, fmt.Sprintf("( (transaction_id = $%d AND position > $%d) OR (transaction_id > $%d) )", argIndex, argIndex+1, argIndex+2))
		args = append(args, options.Cursor.TransactionID, options.Cursor.Position, options.Cursor.TransactionID)
		argIndex += 3
	}

	// Build final query efficiently
	var sqlQuery strings.Builder
	sqlQuery.WriteString("SELECT type, tags, data, position, transaction_id, created_at FROM events")

	if len(conditions) > 0 {
		sqlQuery.WriteString(" WHERE ")
		sqlQuery.WriteString(strings.Join(conditions, " AND "))
	}

	// Use transaction_id ordering for proper event ordering guarantees
	sqlQuery.WriteString(" ORDER BY transaction_id ASC, position ASC")

	// Add limit if specified
	if options.Limit != nil {
		sqlQuery.WriteString(fmt.Sprintf(" LIMIT %d", *options.Limit))
	}

	return sqlQuery.String(), args, nil
}

// combineProjectorQueries combines queries from multiple projectors
func (es *eventStore) combineProjectorQueries(projectors []BatchProjector) Query {
	// Collect all query items from all projectors
	var allItems []QueryItem

	for _, bp := range projectors {
		// Add all items from this projector's query
		allItems = append(allItems, bp.StateProjector.Query.getItems()...)
	}

	// Create a new query with all combined items
	return &query{Items: allItems}
}

// eventMatchesProjector checks if an event matches a projector's query
func (es *eventStore) eventMatchesProjector(event Event, projector StateProjector) bool {
	// If projector has no query items, it matches all events
	if len(projector.Query.getItems()) == 0 {
		return true
	}

	// Check if event matches any of the projector's query items
	for _, item := range projector.Query.getItems() {
		// Check event types if specified
		if len(item.getEventTypes()) > 0 {
			eventTypeMatches := false
			for _, eventType := range item.getEventTypes() {
				if event.Type == eventType {
					eventTypeMatches = true
					break
				}
			}
			if !eventTypeMatches {
				continue // Event type doesn't match, try next item
			}
		}

		// Check tags if specified
		if len(item.getTags()) > 0 {
			// Convert tags to map for easy lookup
			eventTags := make(map[string]string)
			for _, tag := range event.Tags {
				eventTags[tag.GetKey()] = tag.GetValue()
			}

			// Check if ALL required tags match
			allTagsMatch := true
			for _, requiredTag := range item.getTags() {
				if eventTags[requiredTag.GetKey()] != requiredTag.GetValue() {
					allTagsMatch = false
					break
				}
			}
			if !allTagsMatch {
				continue // Tags don't match, try next item
			}
		}

		// If we get here, this item matches
		return true
	}

	// No items matched
	return false
}

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
	sqlQuery, args, err := es.buildReadQuerySQL(query, ReadOptions{})
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
	var latestCursor *Cursor
	var hasEvents bool
	for rows.Next() {
		var row rowEvent
		if err := rows.Scan(&row.Type, &row.Tags, &row.Data, &row.Position, &row.TransactionID, &row.CreatedAt); err != nil {
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
		hasEvents = true

		// Update latest cursor (events are ordered by transaction_id ASC, position ASC)
		latestCursor = &Cursor{
			TransactionID: row.TransactionID,
			Position:      row.Position,
		}

		// Apply projectors
		for _, bp := range projectors {
			if es.eventMatchesProjector(event, bp.StateProjector) {
				states[bp.ID] = bp.StateProjector.TransitionFn(states[bp.ID], event)
			}
		}
	}

	// Only set cursor if we actually processed events
	if !hasEvents {
		appendCondition.setAfterCursor(nil)
	} else {
		// Set the cursor in the append condition for proper optimistic locking
		appendCondition.setAfterCursor(latestCursor)
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

// BuildAppendConditionFromQuery builds an AppendCondition from a specific query
// This aligns with DCB specification: each append operation should use the same query
// that was used when building the Decision Model
func BuildAppendConditionFromQuery(query Query) AppendCondition {
	return NewAppendCondition(query)
}

// ProjectDecisionModelChannel projects multiple states using channel-based streaming
// This is optimized for small to medium datasets (< 500 events) and provides
// a more Go-idiomatic interface using channels for state projection
func (es *eventStore) ProjectDecisionModelChannel(ctx context.Context, projectors []BatchProjector) (<-chan ProjectionResult, Cursor, error) {
	if len(projectors) == 0 {
		return nil, Cursor{}, fmt.Errorf("at least one projector is required")
	}

	// Validate projectors
	for _, bp := range projectors {
		if bp.StateProjector.TransitionFn == nil {
			return nil, Cursor{}, &ValidationError{
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
		return nil, Cursor{}, &ValidationError{
			EventStoreError: EventStoreError{
				Op:  "ProjectDecisionModelChannel",
				Err: fmt.Errorf("query must contain at least one item"),
			},
			Field: "query",
			Value: "empty",
		}
	}

	// Validate query items
	if err := validateQueryTags(query); err != nil {
		return nil, Cursor{}, err
	}

	// Build the SQL query
	sqlQuery, args, err := es.buildReadQuerySQL(query, ReadOptions{})
	if err != nil {
		return nil, Cursor{}, fmt.Errorf("failed to build query: %w", err)
	}

	// Use the caller's context directly for streaming
	queryCtx := ctx

	// Execute the query with timeout (same as ReadWithOptions)
	rows, err := es.pool.Query(queryCtx, sqlQuery, args...)
	if err != nil {
		return nil, Cursor{}, &ResourceError{
			EventStoreError: EventStoreError{
				Op:  "ProjectDecisionModelChannel",
				Err: fmt.Errorf("query failed: %w", err),
			},
			Resource: "database",
		}
	}

	// Create result channel with configurable buffer
	resultChan := make(chan ProjectionResult, es.streamBuffer)

	// Track latest cursor
	var latestCursor Cursor

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
					// Log error and send error result
					log.Printf("Error scanning row in ProjectDecisionModelChannel: %v", err)
					select {
					case resultChan <- ProjectionResult{
						Error: fmt.Errorf("scan error: %w", err),
					}:
					case <-ctx.Done():
						return
					}
					continue
				}

				// Update latest cursor (events are ordered by transaction_id ASC, position ASC)
				latestCursor = Cursor{
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
