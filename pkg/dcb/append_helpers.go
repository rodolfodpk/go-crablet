package dcb

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"
)

// prepareEventBatch prepares arrays for batch insert from events
func prepareEventBatch(events []InputEvent) ([]string, [][]string, [][]byte, error) {
	types := make([]string, len(events))
	tags := make([][]string, len(events))
	data := make([][]byte, len(events))

	for i, e := range events {
		types[i] = e.GetType()
		data[i] = e.GetData()

		// Convert tags to TEXT[] format
		tagStrings := TagsToString(e.GetTags())
		tags[i] = tagStrings

		// Log event details
		log.Printf("Appending event %d: Type=%s", i, e.GetType())
	}

	return types, tags, data, nil
}

// executeBatchInsert executes the batch insert and returns positions
func executeBatchInsert(ctx context.Context, tx pgx.Tx, events []InputEvent, types []string, tags [][]string, data [][]byte) ([]int64, error) {
	// Pre-allocate batch and results for better performance
	batch := &pgx.Batch{}
	positions := make([]int64, len(events))

	// Pre-allocate batch with known size
	batch.Queue("BEGIN")

	// Add insert statements to batch efficiently
	for i := range events {
		batch.Queue(`
			INSERT INTO events (type, tags, data)
			VALUES ($1, $2, $3)
			RETURNING position
		`, types[i], tags[i], data[i])
	}

	batch.Queue("COMMIT")

	// Execute batch
	br := tx.SendBatch(ctx, batch)
	defer br.Close()

	// Get results efficiently
	for i := range events {
		err := br.QueryRow().Scan(&positions[i])
		if err != nil {
			return nil, &EventStoreError{
				Op:  "executeBatchInsert",
				Err: fmt.Errorf("failed to insert event %d: %w", i, err),
			}
		}
	}

	return positions, nil
}

// checkForConflictingEvents checks for conflicting events in optimistic locking
func checkForConflictingEvents(ctx context.Context, tx pgx.Tx, query Query, latestPosition int64) error {
	if len(query.getItems()) == 0 {
		return nil // No query items, no conflict check needed
	}

	// For optimistic locking, we only check the first query item
	// This maintains backward compatibility while supporting the new structure
	item := query.getItems()[0]

	// Convert item tags to TEXT[] format
	itemTagsArray := TagsToArray(item.getTags())

	var exists bool
	checkQuery := `
		SELECT EXISTS(
			SELECT 1 FROM events 
			WHERE position > $1 
			  AND tags @> $2::text[]
			  AND ($3::text[] IS NULL OR
				   array_length($3::text[], 1) = 0 OR
				   type = ANY($3::text[]))
		)
	`
	err := tx.QueryRow(ctx, checkQuery, latestPosition, itemTagsArray, item.getEventTypes()).Scan(&exists)
	if err != nil {
		return &EventStoreError{
			Op:  "checkForConflictingEvents",
			Err: fmt.Errorf("failed to check for conflicting events: %w", err),
		}
	}

	if exists {
		return &ConcurrencyError{
			EventStoreError: EventStoreError{
				Op:  "checkForConflictingEvents",
				Err: fmt.Errorf("conflicting events found after position %d", latestPosition),
			},
			ExpectedPosition: latestPosition,
			ActualPosition:   latestPosition + 1, // Since we found a conflicting event, it must be at the next position
		}
	}

	return nil
}

// checkForMatchingEvents checks if any events match the given condition
func checkForMatchingEvents(ctx context.Context, tx pgx.Tx, condition AppendCondition) error {
	failIfEventsMatch := condition.getFailIfEventsMatch()
	if failIfEventsMatch == nil || len((*failIfEventsMatch).getItems()) == 0 {
		return nil // No query items, no check needed
	}

	// Use the SQL function for proper cursor-based condition checking
	afterCursor := condition.getAfterCursor()

	// Convert condition to JSON for SQL function
	conditionJSON, err := buildConditionJSON(failIfEventsMatch, afterCursor)
	if err != nil {
		return &EventStoreError{
			Op:  "checkForMatchingEvents",
			Err: fmt.Errorf("failed to build condition JSON: %w", err),
		}
	}

	// Call the SQL function to check conditions
	_, err = tx.Exec(ctx, "SELECT check_append_condition($1, $2)",
		conditionJSON, afterCursor)
	if err != nil {
		// Check if it's a concurrency error (raised by the SQL function)
		if isConcurrencyError(err) {
			return &ConcurrencyError{
				EventStoreError: EventStoreError{
					Op:  "checkForMatchingEvents",
					Err: fmt.Errorf("events matching condition found: %w", err),
				},
				ExpectedPosition: 0,
				ActualPosition:   0,
			}
		}
		return &EventStoreError{
			Op:  "checkForMatchingEvents",
			Err: fmt.Errorf("failed to check for matching events: %w", err),
		}
	}

	return nil
}

// buildConditionJSON builds JSON for the SQL function from the condition components
func buildConditionJSON(failIfEventsMatch *Query, afterCursor *Cursor) (interface{}, error) {
	if failIfEventsMatch == nil {
		return nil, nil
	}

	// Build the condition structure
	condition := map[string]interface{}{
		"fail_if_events_match": failIfEventsMatch,
	}

	if afterCursor != nil {
		condition["after_cursor"] = afterCursor
	}

	return condition, nil
}

// isConcurrencyError checks if the error is a concurrency error raised by SQL
func isConcurrencyError(err error) bool {
	// PostgreSQL raises specific error messages for concurrency violations
	// The SQL function raises: 'append condition violated: % matching events found'
	return err != nil &&
		(err.Error() == "append condition violated: 1 matching events found" ||
			err.Error() == "append condition violated: 2 matching events found" ||
			err.Error() == "append condition violated: 3 matching events found" ||
			err.Error() == "append condition violated: 4 matching events found" ||
			err.Error() == "append condition violated: 5 matching events found")
}
