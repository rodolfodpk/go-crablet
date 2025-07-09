package dcb

import (
	"context"
	"fmt"
	"time"
)

// ReadWithOptions reads events matching the query with additional options
func (es *eventStore) ReadWithOptions(ctx context.Context, query Query, options *ReadOptions) ([]Event, error) {
	if len(query.getItems()) == 0 {
		return nil, &ValidationError{
			EventStoreError: EventStoreError{
				Op:  "read",
				Err: fmt.Errorf("query must contain at least one item"),
			},
			Field: "query",
			Value: "empty",
		}
	}

	// Validate query items
	if err := validateQueryTags(query); err != nil {
		return nil, err
	}

	// Build SQL query based on query items
	sqlQuery, args, err := es.buildReadQuerySQL(query, options)
	if err != nil {
		return nil, err
	}

	// Execute the query with timeout
	queryCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Execute the query
	rows, err := es.pool.Query(queryCtx, sqlQuery, args...)
	if err != nil {
		return nil, &ResourceError{
			EventStoreError: EventStoreError{
				Op:  "read",
				Err: fmt.Errorf("failed to execute read query: %w", err),
			},
			Resource: "database",
		}
	}
	defer rows.Close()

	// Collect events
	var events []Event

	for rows.Next() {
		var row rowEvent

		if err := rows.Scan(&row.Type, &row.Tags, &row.Data, &row.Position, &row.TransactionID, &row.CreatedAt); err != nil {
			return nil, &ResourceError{
				EventStoreError: EventStoreError{
					Op:  "read",
					Err: fmt.Errorf("failed to scan event row: %w", err),
				},
				Resource: "database",
			}
		}

		// Convert row to Event using the helper function
		event := convertRowToEvent(row)
		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, &ResourceError{
			EventStoreError: EventStoreError{
				Op:  "read",
				Err: fmt.Errorf("error iterating over events: %w", err),
			},
			Resource: "database",
		}
	}

	return events, nil
}

// Read reads events matching the query (no options)
func (es *eventStore) Read(ctx context.Context, query Query) ([]Event, error) {
	return es.ReadWithOptions(ctx, query, nil)
}
