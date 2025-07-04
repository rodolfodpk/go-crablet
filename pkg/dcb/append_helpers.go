package dcb

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

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
