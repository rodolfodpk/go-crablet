package dcb

import (
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
	var orConditions []string
	var args []interface{}

	for _, item := range query.Items {
		var andConditions []string
		argIndex := len(args) + 1

		// Add event type filtering if specified
		if len(item.EventTypes) > 0 {
			andConditions = append(andConditions, fmt.Sprintf("type = ANY($%d)", argIndex))
			args = append(args, item.EventTypes)
			argIndex++
		}

		// Add tag conditions - use contains operator for DCB semantics
		if len(item.Tags) > 0 {
			tagMap := make(map[string]string)
			for _, tag := range item.Tags {
				tagMap[tag.Key] = tag.Value
			}
			queryTags, err := json.Marshal(tagMap)
			if err != nil {
				return "", nil, &EventStoreError{
					Op:  "buildCombinedQuerySQL",
					Err: fmt.Errorf("failed to marshal query tags: %w", err),
				}
			}
			andConditions = append(andConditions, fmt.Sprintf("tags @> $%d", argIndex))
			args = append(args, queryTags)
			argIndex++
		}

		// Combine AND conditions for this item
		if len(andConditions) > 0 {
			orConditions = append(orConditions, "("+strings.Join(andConditions, " AND ")+")")
		}
	}

	// Combine OR conditions for all items
	sqlQuery := fmt.Sprintf("SELECT id, type, tags, data, position, causation_id, correlation_id FROM events WHERE (%s)", strings.Join(orConditions, " OR "))

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
