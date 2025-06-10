package dcb

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// eventIterator implements EventIterator for streaming events
type eventIterator struct {
	store        *eventStore
	query        Query
	options      *ReadOptions
	lastPosition int64
	batchSize    int
	currentBatch []Event
	currentIndex int
	closed       bool
	hasMore      bool
	initialized  bool
	ctx          context.Context
	totalFetched int
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
			BatchSize:    1000, // Default batch size
		}
	}

	// Set default batch size if not specified
	if options.BatchSize <= 0 {
		options.BatchSize = 1000
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

	// Initialize iterator with streaming configuration
	iterator := &eventIterator{
		store:        es,
		query:        query,
		options:      options,
		batchSize:    options.BatchSize, // Use configurable batch size
		hasMore:      true,
		ctx:          ctx,
		totalFetched: 0,
	}

	// Set initial position based on options
	if options.FromPosition > 0 {
		iterator.lastPosition = options.FromPosition - 1 // Start from the position before FromPosition
	} else {
		iterator.lastPosition = 0
	}

	return iterator, nil
}

// Next returns the next event in the stream
func (ei *eventIterator) Next() (*Event, error) {
	if ei.closed {
		return nil, fmt.Errorf("iterator is closed")
	}

	// Check if we need to fetch the first batch
	if !ei.initialized {
		if err := ei.fetchNextBatch(); err != nil {
			return nil, err
		}
		ei.initialized = true
	}

	// Check if we need more events
	if ei.currentIndex >= len(ei.currentBatch) {
		if !ei.hasMore {
			return nil, nil // No more events
		}
		if err := ei.fetchNextBatch(); err != nil {
			return nil, err
		}
		if len(ei.currentBatch) == 0 {
			ei.hasMore = false
			return nil, nil
		}
	}

	// Return next event from current batch
	event := ei.currentBatch[ei.currentIndex]
	ei.currentIndex++
	ei.lastPosition = event.Position
	return &event, nil
}

// fetchNextBatch loads the next batch of events using keyset pagination
func (ei *eventIterator) fetchNextBatch() error {
	// Build SQL query with keyset pagination
	var sqlQuery string
	var args []interface{}
	argIndex := 1

	// Handle empty query (matches all events)
	if len(ei.query.Items) == 0 {
		sqlQuery = `
			SELECT id, type, tags, data, position, causation_id, correlation_id 
			FROM events 
		`
	} else {
		// Build conditions for each query item (OR logic)
		var conditions []string
		for i, item := range ei.query.Items {
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
				return &EventStoreError{
					Op:  "EventIterator.fetchNextBatch",
					Err: fmt.Errorf("failed to marshal query item %d tags: %w", i, err),
				}
			}

			// Build condition for this item
			itemCondition := fmt.Sprintf("(tags @> $%d::jsonb", argIndex)
			args = append(args, itemTagsJSON)
			argIndex++

			// Add event type filtering if specified
			if len(item.EventTypes) > 0 {
				placeholders := make([]string, len(item.EventTypes))
				for j := range item.EventTypes {
					placeholders[j] = fmt.Sprintf("$%d", argIndex)
					args = append(args, item.EventTypes[j])
					argIndex++
				}
				itemCondition += fmt.Sprintf(" AND type IN (%s)", strings.Join(placeholders, ", "))
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

	// Add keyset pagination condition
	if ei.lastPosition > 0 {
		if ei.options.OrderBy == "asc" {
			sqlQuery += fmt.Sprintf(" AND position > $%d", argIndex)
		} else {
			sqlQuery += fmt.Sprintf(" AND position < $%d", argIndex)
		}
		args = append(args, ei.lastPosition)
		argIndex++
	} else if ei.options.FromPosition > 0 {
		// Handle initial position filtering
		sqlQuery += fmt.Sprintf(" AND position >= $%d", argIndex)
		args = append(args, ei.options.FromPosition)
		argIndex++
	}

	// Add ordering
	sqlQuery += fmt.Sprintf(" ORDER BY position %s", ei.options.OrderBy)

	// Add batch size limit
	sqlQuery += fmt.Sprintf(" LIMIT $%d", argIndex)
	args = append(args, ei.batchSize)
	argIndex++

	// Execute query
	rows, err := ei.store.pool.Query(ei.ctx, sqlQuery, args...)
	if err != nil {
		return &ResourceError{
			EventStoreError: EventStoreError{
				Op:  "EventIterator.fetchNextBatch",
				Err: fmt.Errorf("failed to execute query: %w", err),
			},
			Resource: "database",
		}
	}
	defer rows.Close()

	// Scan all rows in this batch
	var batch []Event
	for rows.Next() {
		var row rowEvent
		if err := rows.Scan(&row.ID, &row.Type, &row.Tags, &row.Data, &row.Position, &row.CausationID, &row.CorrelationID); err != nil {
			return &ResourceError{
				EventStoreError: EventStoreError{
					Op:  "EventIterator.fetchNextBatch",
					Err: fmt.Errorf("failed to scan event row: %w", err),
				},
				Resource: "database",
			}
		}

		event := convertRowToEvent(row)
		batch = append(batch, event)
	}

	// Check for errors from iterating over rows
	if err := rows.Err(); err != nil {
		return &ResourceError{
			EventStoreError: EventStoreError{
				Op:  "EventIterator.fetchNextBatch",
				Err: fmt.Errorf("error iterating over events: %w", err),
			},
			Resource: "database",
		}
	}

	// Update iterator state
	ei.currentBatch = batch
	ei.currentIndex = 0

	// Check if we have more events (if we got a full batch, there might be more)
	if len(batch) < ei.batchSize {
		ei.hasMore = false
	}

	// Apply global limit if specified
	if ei.options.Limit > 0 {
		totalFetched := ei.totalFetched + len(ei.currentBatch)
		if totalFetched >= ei.options.Limit {
			// Trim batch to respect limit
			remaining := ei.options.Limit - ei.totalFetched
			if remaining < len(ei.currentBatch) {
				ei.currentBatch = ei.currentBatch[:remaining]
			}
			ei.hasMore = false
		}
		ei.totalFetched = totalFetched
	}

	return nil
}

// Close closes the iterator and releases resources
func (ei *eventIterator) Close() error {
	if ei.closed {
		return nil
	}
	ei.closed = true
	ei.currentBatch = nil // Release memory
	return nil
}

// Position returns the position of the last event read
func (ei *eventIterator) Position() int64 {
	return ei.lastPosition
}
