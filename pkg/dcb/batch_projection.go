package dcb

import (
	"fmt"
	"strings"
)

// rowEvent is a helper struct for scanning database rows.
type rowEvent struct {
	Type     string
	Tags     []string
	Data     []byte
	Position int64
}

// convertRowToEvent converts a database row to an Event
func convertRowToEvent(row rowEvent) Event {
	return Event{
		Type:     row.Type,
		Tags:     ParseTagsArray(row.Tags),
		Data:     row.Data,
		Position: row.Position,
	}
}

// buildReadQuerySQL builds the SQL query for reading events
func (es *eventStore) buildReadQuerySQL(query Query, options *ReadOptions) (string, []interface{}, error) {
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

	// Add position conditions
	if options != nil && options.FromPosition != nil {
		conditions = append(conditions, fmt.Sprintf("position > $%d", argIndex))
		args = append(args, *options.FromPosition)
		argIndex++
	}

	// Build final query efficiently
	var sqlQuery strings.Builder
	sqlQuery.WriteString("SELECT type, tags, data, position FROM events")

	if len(conditions) > 0 {
		sqlQuery.WriteString(" WHERE ")
		sqlQuery.WriteString(strings.Join(conditions, " AND "))
	}

	sqlQuery.WriteString(" ORDER BY position ASC")

	// Add limit if specified
	if options != nil && options.Limit != nil {
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
