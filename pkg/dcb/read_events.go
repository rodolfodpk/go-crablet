package dcb

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// eventIterator implements EventIterator for streaming events
type eventIterator struct {
	rows     pgx.Rows
	ctx      context.Context
	position int64
	limit    int
	count    int
	orderBy  string
	closed   bool
}

// ReadEvents reads events matching the query with optional configuration.
// This is the core DCB method for reading events.
// Returns an EventIterator for streaming events efficiently.
func (es *eventStore) ReadEvents(ctx context.Context, query Query, options *ReadOptions) (EventIterator, error) {
	// Validate query
	if err := validateQueryTags(query); err != nil {
		return nil, err
	}

	// Set default options if nil
	if options == nil {
		options = &ReadOptions{
			FromPosition: 0,
			Limit:        0, // No limit
			OrderBy:      "asc",
		}
	}

	// Validate options
	if options.Limit < 0 {
		return nil, &ValidationError{
			EventStoreError: EventStoreError{
				Op:  "ReadEvents",
				Err: fmt.Errorf("limit cannot be negative"),
			},
			Field: "limit",
			Value: fmt.Sprintf("%d", options.Limit),
		}
	}

	if options.OrderBy != "asc" && options.OrderBy != "desc" {
		return nil, &ValidationError{
			EventStoreError: EventStoreError{
				Op:  "ReadEvents",
				Err: fmt.Errorf("orderBy must be 'asc' or 'desc'"),
			},
			Field: "orderBy",
			Value: options.OrderBy,
		}
	}

	// Build SQL query
	var sqlQuery string
	var args []interface{}
	argIndex := 1

	// Handle empty query (matches all events)
	if len(query.Items) == 0 {
		sqlQuery = `
			SELECT id, type, tags, data, position, causation_id, correlation_id 
			FROM events 
		`
	} else {
		// Build conditions for each query item (OR logic)
		var conditions []string
		for i, item := range query.Items {
			if i > 0 {
				conditions = append(conditions, "OR")
			}

			// Convert item tags to JSONB
			itemTagMap := make(map[string]string)
			for _, t := range item.Tags {
				itemTagMap[t.Key] = t.Value
			}
			itemTagsJSON, err := json.Marshal(itemTagMap)
			if err != nil {
				return nil, &EventStoreError{
					Op:  "ReadEvents",
					Err: fmt.Errorf("failed to marshal query item %d tags: %w", i, err),
				}
			}

			// Build condition for this item
			itemCondition := fmt.Sprintf("(tags @> $%d::jsonb", argIndex)
			args = append(args, itemTagsJSON)
			argIndex++

			// Add event type filtering if specified
			if len(item.Types) > 0 {
				itemCondition += fmt.Sprintf(" AND type = ANY($%d)", argIndex)
				args = append(args, item.Types)
				argIndex++
			}

			itemCondition += ")"
			conditions = append(conditions, itemCondition)
		}

		// Build the complete SQL query
		sqlQuery = fmt.Sprintf(`
			SELECT id, type, tags, data, position, causation_id, correlation_id 
			FROM events 
			WHERE %s
		`, conditions[0])

		// Add remaining conditions with OR
		for i := 1; i < len(conditions); i++ {
			if conditions[i] == "OR" {
				continue
			}
			sqlQuery += fmt.Sprintf(" OR %s", conditions[i])
		}
	}

	// Add position filtering if specified
	if options.FromPosition > 0 {
		sqlQuery += fmt.Sprintf(" AND position >= $%d", argIndex)
		args = append(args, options.FromPosition)
		argIndex++
	}

	// Add ordering
	sqlQuery += fmt.Sprintf(" ORDER BY position %s", options.OrderBy)

	// Add limit if specified
	if options.Limit > 0 {
		sqlQuery += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, options.Limit)
	}

	// Execute query
	rows, err := es.pool.Query(ctx, sqlQuery, args...)
	if err != nil {
		return nil, &ResourceError{
			EventStoreError: EventStoreError{
				Op:  "ReadEvents",
				Err: fmt.Errorf("failed to execute query: %w", err),
			},
			Resource: "database",
		}
	}

	return &eventIterator{
		rows:    rows,
		ctx:     ctx,
		limit:   options.Limit,
		orderBy: options.OrderBy,
	}, nil
}

// Next returns the next event in the stream
func (ei *eventIterator) Next() (*Event, error) {
	if ei.closed {
		return nil, fmt.Errorf("iterator is closed")
	}

	// Check limit
	if ei.limit > 0 && ei.count >= ei.limit {
		return nil, nil // No more events
	}

	// Get next row
	if !ei.rows.Next() {
		return nil, nil // No more events
	}

	// Scan row
	var row rowEvent
	if err := ei.rows.Scan(&row.ID, &row.Type, &row.Tags, &row.Data, &row.Position, &row.CausationID, &row.CorrelationID); err != nil {
		return nil, &ResourceError{
			EventStoreError: EventStoreError{
				Op:  "EventIterator.Next",
				Err: fmt.Errorf("failed to scan event row: %w", err),
			},
			Resource: "database",
		}
	}

	// Convert row to Event
	event := convertRowToEvent(row)
	ei.position = row.Position
	ei.count++

	return &event, nil
}

// Close closes the iterator and releases resources
func (ei *eventIterator) Close() error {
	if ei.closed {
		return nil
	}
	ei.closed = true
	ei.rows.Close()
	return nil
}

// Position returns the position of the last event read
func (ei *eventIterator) Position() int64 {
	return ei.position
}
