package dcb

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"log"
	"sync"
)

// Tag is a key-value pair for querying events.
type Tag struct {
	Key   string
	Value string
}

// Query defines criteria for selecting events.
type Query struct {
	Tags       []Tag    // Events must match all these tags (empty means match any tag)
	EventTypes []string // Events must match one of these types (empty means match any type)
}

type StateReducer struct {
	InitialState any
	ReducerFn    func(any, Event) any
}

// InputEvent represents an event to be appended to the store.
type InputEvent struct {
	Type string // Event type (e.g., "Subscription")
	Tags []Tag  // Tags for querying (e.g., {"course_id": "C1"})
	Data []byte // JSON-encoded event payload
}

// Event represents a persisted event in the system.
type Event struct {
	ID            string // Unique event identifier (UUID)
	Type          string // Event type (e.g., "Subscription")
	Tags          []Tag  // Tags for querying (e.g., {"course_id": "C1"})
	Data          []byte // Event payload
	Position      int64  // Position in the event stream
	CausationID   string // UUID of the event that caused this event (optional)
	CorrelationID string // UUID linking to the root event or process (optional)
}

// NewTags creates a slice of Tag from alternating key-value string pairs.
// Example: NewTags("course_id", "C1", "user_id", "U1")
// Validation will be performed when tags are used in AppendEvents
func NewTags(kv ...string) []Tag {
	if len(kv)%2 != 0 || len(kv) == 0 {
		// Return empty slice instead of error
		return []Tag{}
	}

	tags := make([]Tag, 0, len(kv)/2)
	for i := 0; i < len(kv); i += 2 {
		tags = append(tags, Tag{Key: kv[i], Value: kv[i+1]})
	}
	return tags
}

// NewQuery creates a Query from tags and optional event types.
// If eventTypes is nil or empty, the query will match any event type.
func NewQuery(tags []Tag, eventTypes ...string) Query {
	return Query{
		Tags:       tags,
		EventTypes: eventTypes,
	}
}

// NewInputEvent creates a new InputEvent without validation
// Validation will be performed when events are appended in AppendEvents
func NewInputEvent(eventType string, tags []Tag, data []byte) InputEvent {
	return InputEvent{
		Type: eventType,
		Tags: tags,
		Data: data,
	}
}

// EventStore provides methods to append and read events in a PostgreSQL database.
type EventStore interface {
	AppendEvents(ctx context.Context, events []InputEvent, query Query, latestKnownPosition int64) (int64, error)
	AppendEventsIfNotExists(ctx context.Context, events []InputEvent, query Query, latestKnownPosition int64, reducer StateReducer) (int64, error)
	ReadState(ctx context.Context, query Query, stateReducer StateReducer) (int64, any, error)
	ReadStateUpTo(ctx context.Context, query Query, stateReducer StateReducer, maxPosition int64) (int64, any, error)
	Close()
}

// eventStore implements EventStore.
type eventStore struct {
	pool         *pgxpool.Pool // Database connection pool
	mu           sync.Mutex    // Protects concurrent access to shared fields
	closed       bool          // Indicates if the store has been closed
	maxBatchSize int           // Maximum number of events in a single batch operation
}

// NewEventStore creates a new EventStore using the provided PostgreSQL connection pool.
// It uses a default maximum batch size of 1000 events.
func NewEventStore(ctx context.Context, pool *pgxpool.Pool) (EventStore, error) {
	// Test the connection
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("unable to connect to database: %w", err)
	}

	return &eventStore{
		pool:         pool,
		maxBatchSize: 1000, // Default maximum batch size
	}, nil
}

// Close closes the appender’s connection pool.
func (es *eventStore) Close() {
	es.mu.Lock()
	defer es.mu.Unlock()
	if !es.closed {
		es.closed = true
		es.pool.Close()
	}
}

func (es *eventStore) AppendEventsIfNotExists(ctx context.Context, events []InputEvent, query Query, latestPosition int64, reducer StateReducer) (int64, error) {
	position, state, err := es.ReadStateUpTo(ctx, query, reducer, latestPosition)
	if err == nil && state != nil {
		log.Printf("Events already exist for query: %v", query)
		return position, nil
	}
	return es.AppendEvents(ctx, events, query, latestPosition)
}

// validateQueryTags validates the query tags and returns an error if invalid
func validateQueryTags(query Query) error {
	// Empty Tags or EventTypes are allowed

	// Validate individual tags if present
	for i, t := range query.Tags {
		if t.Key == "" {
			log.Printf("Query tag %d has empty key", i)
			return fmt.Errorf("query tag %d has empty key", i)
		}
		if t.Value == "" {
			log.Printf("Query tag %d has empty value for key %s", i, t.Key)
			return fmt.Errorf("query tag %d has empty value for key %s", i, t.Key)
		}
	}

	// Validate event types (optional)
	for i, eventType := range query.EventTypes {
		if eventType == "" {
			log.Printf("Query event type %d is empty", i)
			return fmt.Errorf("query event type %d is empty", i)
		}
	}

	return nil
}

// validateEvent validates a single event and returns an error if invalid
func validateEvent(e InputEvent, index int) error {
	// Validate event type
	if e.Type == "" {
		log.Printf("Event %d has empty type", index)
		return fmt.Errorf("event %d has empty type", index)
	}

	// Validate event tags
	if len(e.Tags) == 0 {
		log.Printf("Event %d has empty tags", index)
		return fmt.Errorf("event %d has empty tags", index)
	}
	for j, t := range e.Tags {
		if t.Key == "" {
			log.Printf("Event %d tag %d has empty key", index, j)
			return fmt.Errorf("event %d tag %d has empty key", index, j)
		}
		if t.Value == "" {
			log.Printf("Event %d tag %d has empty value for key %s", index, j, t.Key)
			return fmt.Errorf("event %d tag %d has empty value for key %s", index, j, t.Key)
		}
	}

	// Validate Data as JSON
	if !json.Valid(e.Data) {
		log.Printf("Event %d has invalid JSON data", index)
		return fmt.Errorf("event %d has invalid JSON data", index)
	}

	return nil
}

// AppendEvents adds multiple events to the stream and returns the latest position.
func (es *eventStore) AppendEvents(ctx context.Context, events []InputEvent, query Query, latestPosition int64) (int64, error) {
	es.mu.Lock()
	if es.closed {
		es.mu.Unlock()
		return 0, fmt.Errorf("eventStore is closed")
	}
	es.mu.Unlock()
	if len(events) > es.maxBatchSize {
		return 0, fmt.Errorf("batch size exceeds %d events", es.maxBatchSize)
	}
	if len(events) == 0 {
		return latestPosition, nil
	}

	// Validate query tags
	if err := validateQueryTags(query); err != nil {
		return 0, err
	}

	// Prepare arrays for PL/pgSQL
	ids := make([]pgtype.UUID, len(events))
	types := make([]string, len(events))
	tagsJSON := make([][]byte, len(events)) // Changed to [][]byte for JSONB
	data := make([][]byte, len(events))     // Changed to [][]byte for JSONB
	causationIDs := make([]pgtype.UUID, len(events))
	correlationIDs := make([]pgtype.UUID, len(events))

	for i, e := range events {
		// Validate each event
		if err := validateEvent(e, i); err != nil {
			return 0, err
		}

		// Generate UUID for event (UUIDv7)
		uuidVal, err := uuid.NewV7()
		if err != nil {
			log.Printf("Failed to generate UUID for event %d: %v", i, err)
			return 0, fmt.Errorf("failed to generate UUID for event %d: %w", i, err)
		}
		pgUUID := pgtype.UUID{}
		err = pgUUID.Scan(uuidVal.String())
		if err != nil {
			log.Printf("Failed to parse UUID for event %d: %v", i, err)
			return 0, fmt.Errorf("failed to parse UUID for event %d: %w", i, err)
		}
		ids[i] = pgUUID

		types[i] = e.Type
		data[i] = e.Data // Store as []byte for JSONB

		// Convert tags to JSONB
		tagMap := make(map[string]string)
		for _, t := range e.Tags {
			tagMap[t.Key] = t.Value
		}
		jsonBytes, err := json.Marshal(tagMap)
		if err != nil {
			log.Printf("Failed to marshal tags for event %d: %v", i, err)
			return 0, fmt.Errorf("failed to marshal tags for event %d: %w", i, err)
		}
		tagsJSON[i] = jsonBytes // Store as []byte for JSONB

		// Set causation_id
		if i > 0 {
			causationIDs[i] = ids[i-1] // Previous event's ID
		} else {
			// For first event, set causation_id to its own ID (self-caused)
			causationIDs[i] = pgUUID
		}

		// Set correlation_id
		if i == 0 {
			// For first event, set correlation_id to its own ID
			correlationIDs[i] = pgUUID
		} else {
			// For subsequent events, use the correlation_id of the first event
			correlationIDs[i] = correlationIDs[0]
		}

		// Log event relationships
		causationIDStr := causationIDs[i].String()
		correlationIDStr := correlationIDs[i].String()
		log.Printf("Appending event %d: ID=%s, CausationID=%s, CorrelationID=%s", i, uuidVal.String(), causationIDStr, correlationIDStr)
	}

	// Convert query tags to JSONB
	queryTagMap := make(map[string]string)
	for _, t := range query.Tags {
		queryTagMap[t.Key] = t.Value
	}
	queryTagsJSON, err := json.Marshal(queryTagMap)
	if err != nil {
		log.Printf("Failed to marshal query tags: %v", err)
		return 0, fmt.Errorf("failed to marshal query tags: %w", err)
	}

	// Append new events
	var pgPositions pgtype.Array[int64]
	err = es.pool.QueryRow(ctx, "SELECT append_events_batch($1, $2, $3::jsonb[], $4::jsonb[], $5::jsonb, $6, $7, $8, $9)",
		ids, types, tagsJSON, data, queryTagsJSON, latestPosition, causationIDs, correlationIDs, query.EventTypes,
	).Scan(&pgPositions)
	if err != nil {
		log.Printf("Failed to append events: %v", err)
		if err.Error() == "ERROR: Foreign key violation: invalid causation_id or correlation_id in batch (SQLSTATE P0001)" {
			return 0, fmt.Errorf("foreign key violation: one or more causation_id or correlation_id values are invalid")
		}
		return 0, fmt.Errorf("append_events failed: %w", err)
	}

	// Extract positions from pgtype.Array[int64]
	positions := pgPositions.Elements
	// Log successful append
	log.Printf("Appended %d events, positions: %v", len(events), positions)

	// Return the latest position
	if len(positions) > 0 {
		return positions[len(positions)-1], nil
	}
	return latestPosition, nil // Fallback, though unlikely
}

// rowEvent is a helper struct for scanning database rows.
type rowEvent struct {
	ID            pgtype.UUID
	Type          string
	Tags          []byte
	Data          []byte
	Position      int64
	CausationID   pgtype.UUID
	CorrelationID pgtype.UUID
}

// ReadState computes a state by streaming events matching the query, up to maxPosition.
func (es *eventStore) ReadState(ctx context.Context, query Query, stateReducer StateReducer) (int64, any, error) {
	return es.ReadStateUpTo(ctx, query, stateReducer, -1)
}

// ReadStateUpTo computes a state by streaming events matching the query, up to maxPosition.
func (es *eventStore) ReadStateUpTo(ctx context.Context, query Query, stateReducer StateReducer, maxPosition int64) (int64, any, error) {
	// We still need a reducer function
	// We still need a reducer function
	if stateReducer.ReducerFn == nil {
		return 0, stateReducer.InitialState, fmt.Errorf("reducer cannot be nil")
	}

	// Build JSONB query condition
	tagMap := make(map[string]string)
	for _, t := range query.Tags {
		tagMap[t.Key] = t.Value
	}
	queryTags, err := json.Marshal(tagMap)
	if err != nil {
		return 0, stateReducer.InitialState, fmt.Errorf("failed to marshal query tags: %w", err)
	}

	// Construct SQL query
	sqlQuery := "SELECT id, type, tags, data, position, causation_id, correlation_id FROM events WHERE tags @> $1"
	args := []interface{}{queryTags}

	// Add event type filtering if specified
	if len(query.EventTypes) > 0 {
		sqlQuery += fmt.Sprintf(" AND type = ANY($%d)", len(args)+1)
		args = append(args, query.EventTypes)
	}

	if maxPosition > 0 {
		sqlQuery += fmt.Sprintf(" AND position <= $%d", len(args)+1)
		args = append(args, maxPosition)
	}

	// Query and stream rows
	rows, err := es.pool.Query(ctx, sqlQuery, args...)
	if err != nil {
		return 0, stateReducer.InitialState, fmt.Errorf("query failed for tags %v: %w", tagMap, err)
	}
	defer rows.Close()

	// Initialize state to initialState; remains unchanged if no events are processed
	state := stateReducer.InitialState
	// Initialize position to 0; will be updated as events are processed
	position := int64(0)

	for rows.Next() {
		var row rowEvent
		err := rows.Scan(&row.ID, &row.Type, &row.Tags, &row.Data, &row.Position, &row.CausationID, &row.CorrelationID)
		if err != nil {
			return 0, stateReducer.InitialState, fmt.Errorf("failed to scan row: %w", err)
		}

		var e Event
		if !row.ID.Valid {
			return 0, stateReducer.InitialState, fmt.Errorf("invalid UUID for id at position %d", row.Position)
		}
		e.ID = row.ID.String()
		e.Type = row.Type
		var tagMap map[string]string
		if err := json.Unmarshal(row.Tags, &tagMap); err != nil {
			return 0, stateReducer.InitialState, fmt.Errorf("failed to unmarshal tags at position %d: %w", row.Position, err)
		}
		for k, v := range tagMap {
			e.Tags = append(e.Tags, Tag{Key: k, Value: v})
		}
		e.Data = row.Data
		e.Position = row.Position
		if row.CausationID.Valid {
			e.CausationID = row.CausationID.String()
		}
		if row.CorrelationID.Valid {
			e.CorrelationID = row.CorrelationID.String()
		}

		state = stateReducer.ReducerFn(state, e)
		position = row.Position
	}

	if err := rows.Err(); err != nil {
		return 0, stateReducer.InitialState, fmt.Errorf("error iterating rows: %w", err)
	}
	if err != nil {
		return 0, stateReducer.InitialState, fmt.Errorf("failed to stream events: %w", err)
	}

	return position, state, nil
}
