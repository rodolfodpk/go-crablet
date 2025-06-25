package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go-crablet/pkg/dcb"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// eventStore implements the EventStore interface
type eventStore struct {
	pool         *pgxpool.Pool // Database connection pool
	maxBatchSize int           // Maximum number of events in a single batch operation
}

// NewEventStore creates a new event store instance
func NewEventStore(ctx context.Context, pool *pgxpool.Pool) (dcb.EventStore, error) {
	if pool == nil {
		return nil, &dcb.ValidationError{
			EventStoreError: dcb.EventStoreError{
				Op:  "NewEventStore",
				Err: fmt.Errorf("pool cannot be nil"),
			},
			Field: "pool",
			Value: "nil",
		}
	}

	return &eventStore{
		pool:         pool,
		maxBatchSize: 1000, // Default batch size
	}, nil
}

// Read reads events matching a query, optionally starting from a specified sequence position
// This matches the DCB specification exactly
func (es *eventStore) Read(ctx context.Context, query dcb.Query, options *dcb.ReadOptions) (dcb.SequencedEvents, error) {
	// Build the SQL query
	sqlQuery, args, err := es.buildReadQuerySQL(query, options)
	if err != nil {
		return dcb.SequencedEvents{}, &dcb.EventStoreError{
			Op:  "read",
			Err: fmt.Errorf("failed to build query: %w", err),
		}
	}

	// Execute the query
	rows, err := es.pool.Query(ctx, sqlQuery, args...)
	if err != nil {
		return dcb.SequencedEvents{}, &dcb.EventStoreError{
			Op:  "read",
			Err: fmt.Errorf("failed to execute query: %w", err),
		}
	}
	defer rows.Close()

	// Scan results
	var events []dcb.Event
	var lastPosition int64

	for rows.Next() {
		var eventType string
		var tagsArray []string
		var data []byte
		var position int64

		err := rows.Scan(&eventType, &tagsArray, &data, &position)
		if err != nil {
			return dcb.SequencedEvents{}, &dcb.EventStoreError{
				Op:  "read",
				Err: fmt.Errorf("failed to scan row: %w", err),
			}
		}

		event := dcb.Event{
			Type:     eventType,
			Tags:     dcb.ParseTagsArray(tagsArray),
			Data:     data,
			Position: position,
		}

		events = append(events, event)
		lastPosition = position
	}

	if err := rows.Err(); err != nil {
		return dcb.SequencedEvents{}, &dcb.EventStoreError{
			Op:  "read",
			Err: fmt.Errorf("error iterating rows: %w", err),
		}
	}

	return dcb.SequencedEvents{
		Events:   events,
		Position: lastPosition,
	}, nil
}

// Append atomically persists one or more events, optionally with an append condition
// This matches the DCB specification exactly
func (es *eventStore) Append(ctx context.Context, events []dcb.InputEvent, condition dcb.AppendCondition) error {
	if len(events) == 0 {
		return &dcb.ValidationError{
			EventStoreError: dcb.EventStoreError{
				Op:  "append",
				Err: fmt.Errorf("events must not be empty"),
			},
			Field: "events",
			Value: "empty",
		}
	}

	if len(events) > es.maxBatchSize {
		return &dcb.ValidationError{
			EventStoreError: dcb.EventStoreError{
				Op:  "append",
				Err: fmt.Errorf("batch size %d exceeds maximum of %d", len(events), es.maxBatchSize),
			},
			Field: "events",
			Value: fmt.Sprintf("count:%d", len(events)),
		}
	}

	// Validate events
	for i, event := range events {
		if event.GetType() == "" {
			return &dcb.ValidationError{
				EventStoreError: dcb.EventStoreError{
					Op:  "append",
					Err: fmt.Errorf("event at index %d has empty type", i),
				},
				Field: "type",
				Value: "empty",
			}
		}

		// Validate tags
		tagKeys := make(map[string]bool)
		for _, tag := range event.GetTags() {
			if tag.Key == "" {
				return &dcb.ValidationError{
					EventStoreError: dcb.EventStoreError{
						Op:  "append",
						Err: fmt.Errorf("event at index %d has tag with empty key", i),
					},
					Field: "tag.key",
					Value: "empty",
				}
			}
			if tag.Value == "" {
				return &dcb.ValidationError{
					EventStoreError: dcb.EventStoreError{
						Op:  "append",
						Err: fmt.Errorf("event at index %d has tag with empty value", i),
					},
					Field: "tag.value",
					Value: "empty",
				}
			}
			if tagKeys[tag.Key] {
				return &dcb.ValidationError{
					EventStoreError: dcb.EventStoreError{
						Op:  "append",
						Err: fmt.Errorf("event at index %d has duplicate tag key: %s", i, tag.Key),
					},
					Field: "tag.key",
					Value: tag.Key,
				}
			}
			tagKeys[tag.Key] = true
		}
	}

	// Validate query if provided
	if condition != nil {
		if failQuery := es.getFailIfEventsMatch(condition); failQuery != nil {
			if err := es.validateQueryTags(*failQuery); err != nil {
				return &dcb.EventStoreError{
					Op:  "append",
					Err: fmt.Errorf("invalid append condition query: %w", err),
				}
			}
		}
	}

	// Begin transaction with SERIALIZABLE isolation level
	tx, err := es.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.Serializable,
	})
	if err != nil {
		return &dcb.EventStoreError{
			Op:  "append",
			Err: fmt.Errorf("failed to begin transaction: %w", err),
		}
	}
	defer tx.Rollback(ctx)

	// Check append condition if provided
	if condition != nil {
		if err := es.checkAppendCondition(ctx, tx, condition); err != nil {
			return err
		}
	}

	// Get current position for optimistic locking
	var currentPosition int64
	err = tx.QueryRow(ctx, "SELECT COALESCE(MAX(position), 0) FROM events").Scan(&currentPosition)
	if err != nil {
		return &dcb.EventStoreError{
			Op:  "append",
			Err: fmt.Errorf("failed to get current position: %w", err),
		}
	}

	// Check optimistic locking condition
	if condition != nil {
		if after := es.getAfter(condition); after != nil {
			expectedPosition := *after
			if currentPosition != expectedPosition {
				return &dcb.ConcurrencyError{
					EventStoreError: dcb.EventStoreError{
						Op:  "append",
						Err: fmt.Errorf("optimistic locking failed: expected position %d, got %d", expectedPosition, currentPosition),
					},
					ExpectedPosition: expectedPosition,
					ActualPosition:   currentPosition,
				}
			}
		}
	}

	// Prepare batch insert
	batch := &pgx.Batch{}
	for i, event := range events {
		position := currentPosition + int64(i+1)
		tagsArray := dcb.TagsToArray(event.GetTags())

		query := `
			INSERT INTO events (type, tags, data, position, created_at)
			VALUES ($1, $2, $3, $4, $5)
		`
		batch.Queue(query, event.GetType(), tagsArray, event.GetData(), position, time.Now())
	}

	// Execute batch
	br := tx.SendBatch(ctx, batch)
	defer br.Close()

	// Check for errors
	for i := 0; i < len(events); i++ {
		_, err := br.Exec()
		if err != nil {
			return &dcb.EventStoreError{
				Op:  "append",
				Err: fmt.Errorf("failed to insert event %d: %w", i, err),
			}
		}
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return &dcb.EventStoreError{
			Op:  "append",
			Err: fmt.Errorf("failed to commit transaction: %w", err),
		}
	}

	return nil
}

// checkAppendCondition checks if the append condition is satisfied
func (es *eventStore) checkAppendCondition(ctx context.Context, tx pgx.Tx, condition dcb.AppendCondition) error {
	failQuery := es.getFailIfEventsMatch(condition)
	if failQuery == nil {
		return nil
	}

	// Build query to check for existing events
	sqlQuery, args, err := es.buildReadQuerySQL(*failQuery, nil)
	if err != nil {
		return &dcb.EventStoreError{
			Op:  "checkAppendCondition",
			Err: fmt.Errorf("failed to build condition query: %w", err),
		}
	}

	// Add LIMIT 1 to optimize the query
	sqlQuery += " LIMIT 1"

	// Execute the query
	var exists bool
	err = tx.QueryRow(ctx, sqlQuery, args...).Scan(&exists)
	if err != nil && err != pgx.ErrNoRows {
		return &dcb.EventStoreError{
			Op:  "checkAppendCondition",
			Err: fmt.Errorf("failed to check append condition: %w", err),
		}
	}

	// If we found matching events, the condition fails
	if err == nil {
		return &dcb.ConcurrencyError{
			EventStoreError: dcb.EventStoreError{
				Op:  "append",
				Err: fmt.Errorf("append condition failed: found matching events"),
			},
			ExpectedPosition: 0,
			ActualPosition:   0,
		}
	}

	return nil
}

// ProjectDecisionModel projects multiple states using projectors and returns final states and append condition
func (es *eventStore) ProjectDecisionModel(ctx context.Context, projectors []dcb.BatchProjector) (map[string]any, dcb.AppendCondition, error) {
	if len(projectors) == 0 {
		return nil, nil, &dcb.ValidationError{
			EventStoreError: dcb.EventStoreError{
				Op:  "projectDecisionModel",
				Err: fmt.Errorf("projectors must not be empty"),
			},
			Field: "projectors",
			Value: "empty",
		}
	}

	// Validate projectors
	for i, projector := range projectors {
		if projector.ID == "" {
			return nil, nil, &dcb.ValidationError{
				EventStoreError: dcb.EventStoreError{
					Op:  "projectDecisionModel",
					Err: fmt.Errorf("projector at index %d has empty ID", i),
				},
				Field: "projector.id",
				Value: "empty",
			}
		}

		if projector.StateProjector.TransitionFn == nil {
			return nil, nil, &dcb.ValidationError{
				EventStoreError: dcb.EventStoreError{
					Op:  "projectDecisionModel",
					Err: fmt.Errorf("projector at index %d has nil transition function", i),
				},
				Field: "projector.transitionFn",
				Value: "nil",
			}
		}

		// Validate query
		if err := es.validateQueryTags(projector.StateProjector.Query); err != nil {
			return nil, nil, &dcb.EventStoreError{
				Op:  "projectDecisionModel",
				Err: fmt.Errorf("invalid query in projector %s: %w", projector.ID, err),
			}
		}
	}

	// Read events for all projectors
	allEvents := make(map[string][]dcb.Event)
	var maxPosition int64

	for _, projector := range projectors {
		query := projector.StateProjector.Query
		sequencedEvents, err := es.Read(ctx, query, nil)
		if err != nil {
			return nil, nil, &dcb.EventStoreError{
				Op:  "projectDecisionModel",
				Err: fmt.Errorf("failed to read events for projector %s: %w", projector.ID, err),
			}
		}

		allEvents[projector.ID] = sequencedEvents.Events
		if sequencedEvents.Position > maxPosition {
			maxPosition = sequencedEvents.Position
		}
	}

	// Project states
	states := make(map[string]any)
	for _, projector := range projectors {
		events := allEvents[projector.ID]
		state := projector.StateProjector.InitialState

		for _, event := range events {
			state = projector.StateProjector.TransitionFn(state, event)
		}

		states[projector.ID] = state
	}

	// Create append condition based on max position
	var appendCondition dcb.AppendCondition
	if maxPosition > 0 {
		appendCondition = dcb.NewAppendConditionAfter(&maxPosition)
	}

	return states, appendCondition, nil
}

// buildReadQuerySQL builds the SQL query for reading events
func (es *eventStore) buildReadQuerySQL(query dcb.Query, options *dcb.ReadOptions) (string, []interface{}, error) {
	// Pre-allocate slices with reasonable capacity
	conditions := make([]string, 0, 4) // Usually 1-4 conditions
	args := make([]interface{}, 0, 8)  // Usually 2-8 args
	argIndex := 1

	// Add query conditions
	items := es.getQueryItems(query)
	if len(items) > 0 {
		orConditions := make([]string, 0, len(items))

		for _, item := range items {
			andConditions := make([]string, 0, 2) // Usually 1-2 conditions per item

			// Add event type conditions
			eventTypes := es.getEventTypes(item)
			if len(eventTypes) > 0 {
				andConditions = append(andConditions, fmt.Sprintf("type = ANY($%d::text[])", argIndex))
				args = append(args, eventTypes)
				argIndex++
			}

			// Add tag conditions - use contains operator for DCB semantics
			tags := es.getTags(item)
			if len(tags) > 0 {
				tagsArray := dcb.TagsToArray(tags)
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

	// Build the base query
	sqlQuery := "SELECT type, tags, data, position FROM events"

	// Add WHERE clause if we have conditions
	if len(conditions) > 0 {
		sqlQuery += " WHERE " + strings.Join(conditions, " AND ")
	}

	// Add ORDER BY
	sqlQuery += " ORDER BY position ASC"

	// Add LIMIT if specified
	if options != nil && options.Limit != nil {
		sqlQuery += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, *options.Limit)
	}

	return sqlQuery, args, nil
}

// CheckConnectionPoolHealth checks the health of a connection pool
func (es *eventStore) CheckConnectionPoolHealth() dcb.ConnectionPoolHealth {
	stats := es.pool.Stat()
	return dcb.ConnectionPoolHealth{
		TotalConns:        stats.TotalConns(),
		IdleConns:         stats.IdleConns(),
		AcquiredConns:     stats.AcquiredConns(),
		ConstructingConns: stats.ConstructingConns(),
		Healthy:           stats.TotalConns() > 0,
		Message:           fmt.Sprintf("Pool has %d total connections", stats.TotalConns()),
	}
}

// Helper methods to access unexported methods from the core package
func (es *eventStore) getQueryItems(query dcb.Query) []dcb.QueryItem {
	// Use type assertion to access the unexported method
	if q, ok := query.(interface{ getItems() []dcb.QueryItem }); ok {
		return q.getItems()
	}
	return nil
}

func (es *eventStore) getEventTypes(item dcb.QueryItem) []string {
	// Use type assertion to access the unexported method
	if qi, ok := item.(interface{ getEventTypes() []string }); ok {
		return qi.getEventTypes()
	}
	return nil
}

func (es *eventStore) getTags(item dcb.QueryItem) []dcb.Tag {
	// Use type assertion to access the unexported method
	if qi, ok := item.(interface{ getTags() []dcb.Tag }); ok {
		return qi.getTags()
	}
	return nil
}

func (es *eventStore) getFailIfEventsMatch(condition dcb.AppendCondition) *dcb.Query {
	// Use type assertion to access the unexported method
	if ac, ok := condition.(interface{ getFailIfEventsMatch() *dcb.Query }); ok {
		return ac.getFailIfEventsMatch()
	}
	return nil
}

func (es *eventStore) getAfter(condition dcb.AppendCondition) *int64 {
	// Use type assertion to access the unexported method
	if ac, ok := condition.(interface{ getAfter() *int64 }); ok {
		return ac.getAfter()
	}
	return nil
}

// validateQueryTags validates the query tags and returns a ValidationError if invalid
func (es *eventStore) validateQueryTags(query dcb.Query) error {
	// Handle empty query (matches all events)
	items := es.getQueryItems(query)
	if len(items) == 0 {
		return nil
	}

	// Validate each query item
	for itemIndex, item := range items {
		// Validate individual tags if present
		tags := es.getTags(item)
		for i, t := range tags {
			if t.Key == "" {
				return &dcb.ValidationError{
					EventStoreError: dcb.EventStoreError{
						Op:  "validateQueryTags",
						Err: fmt.Errorf("empty key in tag %d of item %d", i, itemIndex),
					},
					Field: fmt.Sprintf("item[%d].tag[%d].key", itemIndex, i),
					Value: fmt.Sprintf("tag[%d]", i),
				}
			}
			if t.Value == "" {
				return &dcb.ValidationError{
					EventStoreError: dcb.EventStoreError{
						Op:  "validateQueryTags",
						Err: fmt.Errorf("empty value for key %s in tag %d of item %d", t.Key, i, itemIndex),
					},
					Field: fmt.Sprintf("item[%d].tag[%d].value", itemIndex, i),
					Value: t.Key,
				}
			}
		}

		// Validate event types if present
		eventTypes := es.getEventTypes(item)
		for i, eventType := range eventTypes {
			if eventType == "" {
				return &dcb.ValidationError{
					EventStoreError: dcb.EventStoreError{
						Op:  "validateQueryTags",
						Err: fmt.Errorf("empty event type at index %d of item %d", i, itemIndex),
					},
					Field: fmt.Sprintf("item[%d].eventTypes[%d]", itemIndex, i),
					Value: fmt.Sprintf("index[%d]", i),
				}
			}
		}
	}

	return nil
}
