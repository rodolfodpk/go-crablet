package dcb

import (
	"context"
	"fmt"
	"log"
	"sort"
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
	OccurredAt    time.Time
}

// convertRowToEvent converts a database row to an Event
func convertRowToEvent(row rowEvent) Event {
	return Event{
		Type:          row.Type,
		Tags:          ParseTagsArray(row.Tags),
		Data:          row.Data,
		Position:      row.Position,
		TransactionID: row.TransactionID,
		OccurredAt:    row.OccurredAt,
	}
}

// buildReadQuerySQL builds the SQL query for reading events
func (es *eventStore) buildReadQuerySQL(query Query, after *Cursor, limit *int) (string, []interface{}, error) {
	// Pre-allocate slices with reasonable capacity
	conditions := make([]string, 0, 4) // Usually 1-4 conditions
	args := make([]interface{}, 0, 8)  // Usually 2-8 args
	argIndex := 1

	// Add query conditions
	if len(query.GetItems()) > 0 {
		orConditions := make([]string, 0, len(query.GetItems()))

		for _, item := range query.GetItems() {
			andConditions := make([]string, 0, 2) // Usually 1-2 conditions per item

			// Add event type conditions
			if len(item.GetEventTypes()) > 0 {
				andConditions = append(andConditions, fmt.Sprintf("type = ANY($%d::text[])", argIndex))
				args = append(args, item.GetEventTypes())
				argIndex++
			}

			// Add tag conditions - use contains operator for DCB semantics
			if len(item.GetTags()) > 0 {
				tagsArray := TagsToArray(item.GetTags())
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
	if after != nil {
		// Use the correct cursor logic from Oskar's article:
		// (transaction_id = after.TransactionID AND position > after.Position) OR (transaction_id > after.TransactionID)
		conditions = append(conditions, fmt.Sprintf("( (transaction_id = $%d AND position > $%d) OR (transaction_id > $%d) )", argIndex, argIndex+1, argIndex+2))
		args = append(args, after.TransactionID, after.Position, after.TransactionID)
		argIndex += 3
	}

	// Build final query efficiently
	var sqlQuery strings.Builder
	sqlQuery.WriteString("SELECT type, tags, data, transaction_id, position, occurred_at FROM events")

	if len(conditions) > 0 {
		sqlQuery.WriteString(" WHERE ")
		sqlQuery.WriteString(strings.Join(conditions, " AND "))
	}

	// Use transaction_id ordering for proper event ordering guarantees
	sqlQuery.WriteString(" ORDER BY transaction_id ASC, position ASC")

	// Add limit if specified
	if limit != nil {
		sqlQuery.WriteString(fmt.Sprintf(" LIMIT %d", *limit))
	}

	return sqlQuery.String(), args, nil
}

// CombineProjectorQueries optimizes by merging QueryItems with the same tags but different event types
// This is useful for consumers who want to optimize their projector queries
func CombineProjectorQueries(projectors []StateProjector) Query {
	// Use a map to group QueryItems by their tags (as a key)
	tagGroups := make(map[string]*queryItem)

	for _, bp := range projectors {
		for _, item := range bp.Query.GetItems() {
			// Create a key from tags for grouping
			tagKey := tagsToKey(item.GetTags())

			if existingItem, exists := tagGroups[tagKey]; exists {
				// Merge event types with existing item
				existingItem.EventTypes = append(existingItem.EventTypes, item.GetEventTypes()...)
			} else {
				// Create new item
				tagGroups[tagKey] = &queryItem{
					EventTypes: append([]string{}, item.GetEventTypes()...),
					Tags:       append([]Tag{}, item.GetTags()...),
				}
			}
		}
	}

	// Convert map back to slice
	var allItems []QueryItem
	for _, item := range tagGroups {
		allItems = append(allItems, item)
	}

	// Create a new query with all combined items
	return &query{Items: allItems}
}

// Optimizes by merging QueryItems with the same tags but different event types
func (es *eventStore) combineProjectorQueries(projectors []StateProjector) Query {
	return CombineProjectorQueries(projectors)
}

// tagsToKey creates a consistent key from tags for grouping
func tagsToKey(tags []Tag) string {
	if len(tags) == 0 {
		return ""
	}

	// Sort tags by key for consistent ordering
	tagPairs := make([]string, len(tags))
	for i, tag := range tags {
		tagPairs[i] = tag.GetKey() + ":" + tag.GetValue()
	}
	sort.Strings(tagPairs)

	return strings.Join(tagPairs, ",")
}

// EventMatchesProjector checks if an event matches a projector's query
// This is useful for consumers who want to do their own event filtering or validation
func EventMatchesProjector(event Event, projector StateProjector) bool {
	// If projector has no query items, it matches all events
	if len(projector.Query.GetItems()) == 0 {
		return true
	}

	// Check if event matches any of the projector's query items
	for _, item := range projector.Query.GetItems() {
		// Check event types if specified
		if len(item.GetEventTypes()) > 0 {
			eventTypeMatches := false
			for _, eventType := range item.GetEventTypes() {
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
		if len(item.GetTags()) > 0 {
			// Convert tags to map for easy lookup
			eventTags := make(map[string]string)
			for _, tag := range event.Tags {
				eventTags[tag.GetKey()] = tag.GetValue()
			}

			// Check if ALL required tags match
			allTagsMatch := true
			for _, requiredTag := range item.GetTags() {
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

// eventMatchesProjector checks if an event matches a projector's query
func (es *eventStore) eventMatchesProjector(event Event, projector StateProjector) bool {
	return EventMatchesProjector(event, projector)
}

// Project projects state from events matching projectors with optional cursor
// cursor == nil: project from beginning of stream
// cursor != nil: project from specified cursor position
// Returns final aggregated states and append condition for optimistic locking
func (es *eventStore) Project(ctx context.Context, projectors []StateProjector, after *Cursor) (map[string]any, AppendCondition, error) {
	// Validate projectors
	for _, bp := range projectors {
		if bp.ID == "" {
			return nil, nil, &ValidationError{
				EventStoreError: EventStoreError{
					Op:  "Project",
					Err: fmt.Errorf("projector ID cannot be empty"),
				},
				Field: "projector.id",
				Value: "empty",
			}
		}
		if bp.TransitionFn == nil {
			return nil, nil, &ValidationError{
				EventStoreError: EventStoreError{
					Op:  "Project",
					Err: fmt.Errorf("projector %s has nil transition function", bp.ID),
				},
				Field: "transitionFn",
				Value: "nil",
			}
		}
		if len(bp.Query.GetItems()) == 0 {
			return nil, nil, &ValidationError{
				EventStoreError: EventStoreError{
					Op:  "Project",
					Err: fmt.Errorf("projector %s has empty query", bp.ID),
				},
				Field: "query",
				Value: "empty",
			}
		}
	}

	// Combine all projector queries for the append condition
	combinedQuery := es.combineProjectorQueries(projectors)

	// Use cursor-based or full projection based on cursor parameter
	if after != nil {
		return es.projectDecisionModelWithQueryFromCursor(ctx, combinedQuery, projectors, after)
	}
	return es.projectDecisionModelWithQuery(ctx, combinedQuery, projectors)
}

// projectDecisionModelWithQuery uses query-based approach for all datasets
func (es *eventStore) projectDecisionModelWithQuery(ctx context.Context, query Query, projectors []StateProjector) (map[string]any, AppendCondition, error) {
	// Validate query
	if query == nil {
		return nil, nil, &ValidationError{
			EventStoreError: EventStoreError{
				Op:  "Project",
				Err: fmt.Errorf("query cannot be nil"),
			},
			Field: "query",
			Value: "nil",
		}
	}

	// Build SQL query
	sqlQuery, args, err := es.buildReadQuerySQL(query, nil, nil)
	if err != nil {
		return nil, nil, &ResourceError{
			EventStoreError: EventStoreError{
				Op:  "Project",
				Err: fmt.Errorf("failed to build query: %w", err),
			},
			Resource: "database",
		}
	}

	// Execute query
	rows, err := es.pool.Query(ctx, sqlQuery, args...)
	if err != nil {
		return nil, nil, &ResourceError{
			EventStoreError: EventStoreError{
				Op:  "Project",
				Err: fmt.Errorf("query failed: %w", err),
			},
			Resource: "database",
		}
	}
	defer rows.Close()

	// Initialize states with initial values
	states := make(map[string]any)
	for _, projector := range projectors {
		states[projector.ID] = projector.InitialState
	}

	// Track latest cursor for append condition
	var latestCursor *Cursor

	// Process events
	for rows.Next() {
		var row rowEvent
		err := rows.Scan(&row.Type, &row.Tags, &row.Data, &row.TransactionID, &row.Position, &row.OccurredAt)
		if err != nil {
			return nil, nil, &ResourceError{
				EventStoreError: EventStoreError{
					Op:  "Project",
					Err: fmt.Errorf("failed to scan row: %w", err),
				},
				Resource: "database",
			}
		}

		// Convert row to event
		event := convertRowToEvent(row)

		// Update latest cursor (events are ordered by transaction_id ASC, position ASC)
		if latestCursor == nil ||
			event.TransactionID > latestCursor.TransactionID ||
			(event.TransactionID == latestCursor.TransactionID && event.Position > latestCursor.Position) {
			latestCursor = &Cursor{
				TransactionID: event.TransactionID,
				Position:      event.Position,
			}
		}

		// Apply event to matching projectors
		for _, projector := range projectors {
			if es.eventMatchesProjector(event, projector) {
				states[projector.ID] = projector.TransitionFn(states[projector.ID], event)
			}
		}
	}

	// Check for row iteration errors
	if err := rows.Err(); err != nil {
		return nil, nil, &ResourceError{
			EventStoreError: EventStoreError{
				Op:  "Project",
				Err: fmt.Errorf("row iteration failed: %w", err),
			},
			Resource: "database",
		}
	}

	// Build append condition from projector queries for optimistic locking
	appendCondition := es.buildAppendConditionFromQuery(query)

	// Set cursor in append condition if we have events
	if latestCursor != nil {
		appendCondition.setAfterCursor(latestCursor)
	}

	return states, appendCondition, nil
}

// projectDecisionModelWithQueryFromCursor uses query-based approach for all datasets with cursor
func (es *eventStore) projectDecisionModelWithQueryFromCursor(ctx context.Context, query Query, projectors []StateProjector, after *Cursor) (map[string]any, AppendCondition, error) {
	// Validate query
	if query == nil {
		return nil, nil, &ValidationError{
			EventStoreError: EventStoreError{
				Op:  "ProjectFromCursor",
				Err: fmt.Errorf("query cannot be nil"),
			},
			Field: "query",
			Value: "nil",
		}
	}

	// Build SQL query
	sqlQuery, args, err := es.buildReadQuerySQL(query, after, nil)
	if err != nil {
		return nil, nil, &ResourceError{
			EventStoreError: EventStoreError{
				Op:  "ProjectFromCursor",
				Err: fmt.Errorf("failed to build query: %w", err),
			},
			Resource: "database",
		}
	}

	// Execute query
	rows, err := es.pool.Query(ctx, sqlQuery, args...)
	if err != nil {
		return nil, nil, &ResourceError{
			EventStoreError: EventStoreError{
				Op:  "ProjectFromCursor",
				Err: fmt.Errorf("query failed: %w", err),
			},
			Resource: "database",
		}
	}
	defer rows.Close()

	// Initialize states with initial values
	states := make(map[string]any)
	for _, projector := range projectors {
		states[projector.ID] = projector.InitialState
	}

	// Track latest cursor for append condition
	var latestCursor *Cursor

	// Process events
	for rows.Next() {
		var row rowEvent
		err := rows.Scan(&row.Type, &row.Tags, &row.Data, &row.TransactionID, &row.Position, &row.OccurredAt)
		if err != nil {
			return nil, nil, &ResourceError{
				EventStoreError: EventStoreError{
					Op:  "ProjectFromCursor",
					Err: fmt.Errorf("failed to scan row: %w", err),
				},
				Resource: "database",
			}
		}

		// Convert row to event
		event := convertRowToEvent(row)

		// Update latest cursor (events are ordered by transaction_id ASC, position ASC)
		if latestCursor == nil ||
			event.TransactionID > latestCursor.TransactionID ||
			(event.TransactionID == latestCursor.TransactionID && event.Position > latestCursor.Position) {
			latestCursor = &Cursor{
				TransactionID: event.TransactionID,
				Position:      event.Position,
			}
		}

		// Apply event to matching projectors
		for _, projector := range projectors {
			if es.eventMatchesProjector(event, projector) {
				states[projector.ID] = projector.TransitionFn(states[projector.ID], event)
			}
		}
	}

	// Check for row iteration errors
	if err := rows.Err(); err != nil {
		return nil, nil, &ResourceError{
			EventStoreError: EventStoreError{
				Op:  "ProjectFromCursor",
				Err: fmt.Errorf("row iteration failed: %w", err),
			},
			Resource: "database",
		}
	}

	// Build append condition from projector queries for optimistic locking
	appendCondition := es.buildAppendConditionFromQuery(query)

	// Set cursor in append condition if we have events
	if latestCursor != nil {
		appendCondition.setAfterCursor(latestCursor)
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

// ProjectStream projects multiple states using channel-based streaming with optional cursor
// cursor == nil: stream from beginning of stream
// cursor != nil: stream from specified cursor position
// This is optimized for large datasets and provides backpressure through channels
// for efficient memory usage and Go-idiomatic streaming
// Returns final aggregated states (same as batch version) via streaming
func (es *eventStore) ProjectStream(ctx context.Context, projectors []StateProjector, after *Cursor) (<-chan map[string]any, <-chan AppendCondition, error) {
	if len(projectors) == 0 {
		return nil, nil, fmt.Errorf("at least one projector is required")
	}

	// Validate projectors
	for _, bp := range projectors {
		if bp.TransitionFn == nil {
			return nil, nil, &ValidationError{
				EventStoreError: EventStoreError{
					Op:  "ProjectStream",
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
	if len(query.GetItems()) == 0 {
		return nil, nil, &ValidationError{
			EventStoreError: EventStoreError{
				Op:  "ProjectStream",
				Err: fmt.Errorf("query must contain at least one item"),
			},
			Field: "query",
			Value: "empty",
		}
	}

	// Validate query items
	if err := validateQueryTags(query); err != nil {
		return nil, nil, err
	}

	// Build the SQL query with cursor
	sqlQuery, args, err := es.buildReadQuerySQL(query, after, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build query: %w", err)
	}

	// Execute the query with hybrid timeout (respects caller timeout if set, otherwise uses default)
	queryCtx, cancel := es.withTimeout(ctx, es.config.QueryTimeout)
	// Don't defer cancel() here - let the goroutine handle it

	rows, err := es.pool.Query(queryCtx, sqlQuery, args...)
	if err != nil {
		return nil, nil, &ResourceError{
			EventStoreError: EventStoreError{
				Op:  "ProjectStream",
				Err: fmt.Errorf("query failed: %w", err),
			},
			Resource: "database",
		}
	}

	// Create result channel with configurable buffer
	resultChan := make(chan map[string]any, es.config.StreamBuffer)

	// Create channel for final AppendCondition
	appendConditionChan := make(chan AppendCondition, 1)

	// Start projection processing in a goroutine
	go func() {
		// Ensure rows are always closed, even if goroutine panics
		defer func() {
			if r := recover(); r != nil {
				log.Printf("ProjectStream panic recovered: %v", r)
			}
			rows.Close()
			close(resultChan)
			close(appendConditionChan)
			cancel() // Cancel the context when goroutine is done
		}()

		// Initialize projector states
		projectorStates := make(map[string]interface{})
		for _, projector := range projectors {
			projectorStates[projector.ID] = projector.InitialState
		}

		// Build AppendCondition from projector queries for optimistic locking (same as Project)
		appendCondition := es.buildAppendConditionFromQuery(query)

		// Track latest cursor (same as Project)
		var latestCursor *Cursor
		var hasEvents bool

		// Process events using the same context as the database query
		for rows.Next() {
			select {
			case <-queryCtx.Done():
				// Context cancelled - exit cleanly
				return
			default:
				var row rowEvent
				err := rows.Scan(
					&row.Type,
					&row.Tags,
					&row.Data,
					&row.TransactionID,
					&row.Position,
					&row.OccurredAt,
				)
				if err != nil {
					// Log error and exit
					log.Printf("Error scanning row in ProjectStream: %v", err)
					return
				}

				// Update latest cursor (events are ordered by transaction_id ASC, position ASC) - same as Project
				latestCursor = &Cursor{
					TransactionID: row.TransactionID,
					Position:      row.Position,
				}
				hasEvents = true

				event := convertRowToEvent(row)

				// Process event with each projector
				for _, projector := range projectors {
					// Check if projector should process this event
					if !es.eventMatchesProjector(event, projector) {
						continue
					}

					// Get current state for this projector
					currentState := projectorStates[projector.ID]

					// Project the event using the transition function
					newState := projector.TransitionFn(currentState, event)

					// Update state
					projectorStates[projector.ID] = newState
				}
			}
		}

		// Check for row iteration errors
		if err := rows.Err(); err != nil {
			log.Printf("Row iteration error in ProjectStream: %v", err)
			return
		}

		// Set cursor in AppendCondition (same logic as Project)
		if !hasEvents {
			appendCondition.setAfterCursor(nil)
		} else {
			appendCondition.setAfterCursor(latestCursor)
		}

		// Send final aggregated states (same as batch version)
		select {
		case resultChan <- projectorStates:
		case <-queryCtx.Done():
			// Context cancelled while trying to send final states
		}

		// Send complete AppendCondition with cursor
		select {
		case appendConditionChan <- appendCondition:
		case <-queryCtx.Done():
			// Context cancelled while trying to send AppendCondition
		}
	}()

	return resultChan, appendConditionChan, nil
}
