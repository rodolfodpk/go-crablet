package dcb

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// =============================================================================
// EventStore Constructor Helpers
// =============================================================================

func newEventStore(pool *pgxpool.Pool, cfg EventStoreConfig) *eventStore {
	// Validate configuration
	if cfg.MaxBatchSize <= 0 {
		cfg.MaxBatchSize = 1000 // Default batch size
	}
	if cfg.StreamBuffer <= 0 {
		cfg.StreamBuffer = 1000 // Default stream buffer size
	}
	if cfg.QueryTimeout <= 0 {
		cfg.QueryTimeout = 15000 // 15 seconds default
	}
	if cfg.AppendTimeout <= 0 {
		cfg.AppendTimeout = 10000 // 10 seconds default
	}
	// TargetEventsTable removed - always use 'events' table for maximum performance

	return &eventStore{
		pool:   pool,
		config: cfg,
	}
}

// =============================================================================
// EventStore Constructors
// =============================================================================

// NewEventStore creates a new EventStore instance with default configuration
func NewEventStore(ctx context.Context, pool *pgxpool.Pool) (EventStore, error) {
	// Test the connection
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Validate that the events table exists with correct structure
	if err := validateEventsTableExists(ctx, pool); err != nil {
		return nil, fmt.Errorf("failed to validate events table: %w", err)
	}

	// Optionally validate commands table (if it exists)
	if err := validateCommandsTableExists(ctx, pool); err != nil {
		return nil, fmt.Errorf("failed to validate commands table: %w", err)
	}

	config := EventStoreConfig{
		MaxBatchSize:           1000,
		StreamBuffer:           1000,
		DefaultAppendIsolation: IsolationLevelReadCommitted,
		QueryTimeout:           15000, // 15 seconds default
		AppendTimeout:          10000, // 10 seconds default
		// TargetEventsTable removed - always use 'events' table for maximum performance
	}
	return newEventStore(pool, config), nil
}

// NewEventStoreWithConfig creates a new EventStore instance with custom configuration
func NewEventStoreWithConfig(ctx context.Context, pool *pgxpool.Pool, config EventStoreConfig) (EventStore, error) {
	// Test the connection
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Validate that the events table exists with correct structure
	if err := validateEventsTableExists(ctx, pool); err != nil {
		return nil, fmt.Errorf("failed to validate events table: %w", err)
	}

	// Optionally validate commands table (if it exists)
	if err := validateCommandsTableExists(ctx, pool); err != nil {
		return nil, fmt.Errorf("failed to validate commands table: %w", err)
	}

	return newEventStore(pool, config), nil
}

// =============================================================================
// Event Constructors
// =============================================================================

// NewInputEvent creates a new InputEvent with the given type, tags, and data.
// Validation is performed when the event is used in EventStore operations.
func NewInputEvent(eventType string, tags []Tag, data []byte) InputEvent {
	return &inputEvent{
		eventType: eventType,
		tags:      tags,
		data:      data,
	}
}

// NewEventBatch creates a slice of events from the given InputEvents.
// This is a convenience function for creating event batches, particularly useful
// when appending multiple related events in a single operation.
func NewEventBatch(events ...InputEvent) []InputEvent {
	return events
}

// =============================================================================
// Tag Constructors
// =============================================================================

// NewTag creates a single tag from key-value pair.
func NewTag(key, value string) Tag {
	return &tag{
		key:   key,
		value: value,
	}
}

// NewTags creates a slice of tags from key-value pairs.
// Validation is performed when the tags are used in EventStore operations.
func NewTags(kv ...string) []Tag {
	if len(kv)%2 != 0 {
		// Return empty tags instead of panicking - validation will happen in EventStore operations
		return []Tag{}
	}
	tags := make([]Tag, len(kv)/2)
	for i := 0; i < len(kv); i += 2 {
		tags[i/2] = NewTag(kv[i], kv[i+1])
	}
	return tags
}

// =============================================================================
// Query Constructors
// =============================================================================

// NewQuery creates a new Query with the given tags and event types.
// This creates a single QueryItem with the specified tags and event types.
func NewQuery(tags []Tag, eventTypes ...string) Query {
	return &query{
		Items: []QueryItem{
			NewQueryItem(eventTypes, tags),
		},
	}
}

// NewQueryEmpty creates a new empty query
func NewQueryEmpty() Query {
	return &query{Items: []QueryItem{}}
}

// NewQueryFromItems creates a new query from a list of query items
func NewQueryFromItems(items ...QueryItem) Query {
	return &query{Items: items}
}

// NewQueryAll creates a query that matches all events.
func NewQueryAll() Query {
	return &query{
		Items: []QueryItem{
			NewQueryItem([]string{}, []Tag{}),
		},
	}
}

// NewQueryItem creates a new QueryItem with the given types and tags.
func NewQueryItem(types []string, tags []Tag) QueryItem {
	return &queryItem{
		EventTypes: types,
		Tags:       tags,
	}
}

// =============================================================================
// AppendCondition Constructors
// =============================================================================

// NewAppendCondition creates a new AppendCondition with the given fail condition.
// This is the primary constructor for DCB concurrency control conditions.
func NewAppendCondition(failIfEventsMatch Query) AppendCondition {
	if failIfEventsMatch == nil {
		return &appendCondition{}
	}
	return &appendCondition{
		FailIfEventsMatch: failIfEventsMatch.(*query),
	}
}

// ToJSON marshals a value to JSON bytes, panicking on error (for convenience in tests and examples).
func ToJSON(v any) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal to JSON: %v", err))
	}
	return data
}

// =============================================================================
// Command Constructors
// =============================================================================

// NewCommand creates a new Command with type, data, and metadata
func NewCommand(commandType string, data []byte, metadata map[string]interface{}) Command {
	return &command{
		commandType: commandType,
		data:        data,
		metadata:    metadata,
	}
}

// =============================================================================
// Query Builder Pattern (Additive - for better developer experience)
// =============================================================================

// QueryBuilder provides a fluent interface for building queries
// DCB compliant: QueryItems are combined with OR, conditions within QueryItem are AND
type QueryBuilder struct {
	items       []QueryItem
	currentItem *queryItemBuilder
}

// queryItemBuilder builds a single QueryItem with AND conditions
type queryItemBuilder struct {
	eventTypes []string
	tags       []Tag
}

// NewQueryBuilder creates a new QueryBuilder instance
func NewQueryBuilder() *QueryBuilder {
	return &QueryBuilder{
		items:       make([]QueryItem, 0),
		currentItem: &queryItemBuilder{},
	}
}

// AddItem starts a new QueryItem for OR conditions
// This creates a new QueryItem that will be combined with OR
func (qb *QueryBuilder) AddItem() *QueryBuilder {
	// Finalize current item if it has content
	if len(qb.currentItem.eventTypes) > 0 || len(qb.currentItem.tags) > 0 {
		item := NewQueryItem(qb.currentItem.eventTypes, qb.currentItem.tags)
		qb.items = append(qb.items, item)
	}

	// Start new item
	qb.currentItem = &queryItemBuilder{}
	return qb
}

// WithTag adds a single tag condition to the current QueryItem (AND)
func (qb *QueryBuilder) WithTag(key, value string) *QueryBuilder {
	qb.currentItem.tags = append(qb.currentItem.tags, NewTag(key, value))
	return qb
}

// WithTags adds multiple tag conditions to the current QueryItem (AND)
func (qb *QueryBuilder) WithTags(kv ...string) *QueryBuilder {
	if len(kv)%2 != 0 {
		// Invalid key-value pairs, return builder unchanged
		return qb
	}

	for i := 0; i < len(kv); i += 2 {
		qb.currentItem.tags = append(qb.currentItem.tags, NewTag(kv[i], kv[i+1]))
	}
	return qb
}

// WithType adds a single event type condition to the current QueryItem (OR with existing types)
func (qb *QueryBuilder) WithType(eventType string) *QueryBuilder {
	qb.currentItem.eventTypes = append(qb.currentItem.eventTypes, eventType)
	return qb
}

// WithTypes adds multiple event type conditions to the current QueryItem (OR with existing types)
func (qb *QueryBuilder) WithTypes(eventTypes ...string) *QueryBuilder {
	qb.currentItem.eventTypes = append(qb.currentItem.eventTypes, eventTypes...)
	return qb
}

// WithTagAndType adds both tag and event type conditions to the current QueryItem
func (qb *QueryBuilder) WithTagAndType(key, value, eventType string) *QueryBuilder {
	qb.WithTag(key, value)
	qb.WithType(eventType)
	return qb
}

// WithTagsAndTypes method removed - use WithTypes() and WithTags() separately for clarity
// Example: qb.WithTypes("Type1", "Type2").WithTags("key1", "value1", "key2", "value2")

// Build creates the final Query from the builder
func (qb *QueryBuilder) Build() Query {
	// Finalize current item if it has content
	if len(qb.currentItem.eventTypes) > 0 || len(qb.currentItem.tags) > 0 {
		item := NewQueryItem(qb.currentItem.eventTypes, qb.currentItem.tags)
		qb.items = append(qb.items, item)
	}

	if len(qb.items) == 0 {
		return NewQueryEmpty()
	}

	return NewQueryFromItems(qb.items...)
}

// =============================================================================
// Simplified AppendCondition Constructors (Additive)
// =============================================================================

// FailIfExists creates an AppendCondition that fails if any events match the given tag
func FailIfExists(key, value string) AppendCondition {
	query := NewQueryBuilder().WithTag(key, value).Build()
	return NewAppendCondition(query)
}

// FailIfEventType creates an AppendCondition that fails if events of the given type exist with the specified tag
func FailIfEventType(eventType, key, value string) AppendCondition {
	query := NewQueryBuilder().WithTagAndType(key, value, eventType).Build()
	return NewAppendCondition(query)
}

// FailIfEventTypes creates an AppendCondition that fails if events of any of the given types exist with the specified tag
func FailIfEventTypes(eventTypes []string, key, value string) AppendCondition {
	query := NewQueryBuilder().WithTypes(eventTypes...).WithTag(key, value).Build()
	return NewAppendCondition(query)
}

// =============================================================================
// Simplified Tag Construction (Additive)
// =============================================================================

// Tags is a map-based tag constructor for better readability
type Tags map[string]string

// ToTags converts a Tags map to a slice of Tag interfaces
func (t Tags) ToTags() []Tag {
	tags := make([]Tag, 0, len(t))
	for key, value := range t {
		tags = append(tags, NewTag(key, value))
	}
	return tags
}

// =============================================================================
// Projection Helpers (Additive - for common patterns)
// =============================================================================

// ProjectCounter creates a projector that counts events
func ProjectCounter(id string, eventType string, key, value string) StateProjector {
	return StateProjector{
		ID:           id,
		Query:        NewQueryBuilder().WithTagAndType(key, value, eventType).Build(),
		InitialState: 0,
		TransitionFn: func(state any, event Event) any {
			return state.(int) + 1
		},
	}
}

// ProjectBoolean creates a projector that tracks if events exist
func ProjectBoolean(id string, eventType string, key, value string) StateProjector {
	return StateProjector{
		ID:           id,
		Query:        NewQueryBuilder().WithTagAndType(key, value, eventType).Build(),
		InitialState: false,
		TransitionFn: func(state any, event Event) any {
			return true
		},
	}
}

// ProjectState creates a projector with custom initial state and transition function
func ProjectState(id string, eventType string, key, value string, initialState any, transitionFn func(any, Event) any) StateProjector {
	return StateProjector{
		ID:           id,
		Query:        NewQueryBuilder().WithTagAndType(key, value, eventType).Build(),
		InitialState: initialState,
		TransitionFn: transitionFn,
	}
}

// ProjectStateWithTypes creates a projector for multiple event types
func ProjectStateWithTypes(id string, eventTypes []string, key, value string, initialState any, transitionFn func(any, Event) any) StateProjector {
	return StateProjector{
		ID:           id,
		Query:        NewQueryBuilder().WithTypes(eventTypes...).WithTag(key, value).Build(),
		InitialState: initialState,
		TransitionFn: transitionFn,
	}
}

// ProjectStateWithTags creates a projector with multiple tag conditions
func ProjectStateWithTags(id string, eventType string, tags Tags, initialState any, transitionFn func(any, Event) any) StateProjector {
	builder := NewQueryBuilder().WithType(eventType)
	for key, value := range tags {
		builder.WithTag(key, value)
	}
	return StateProjector{
		ID:           id,
		Query:        builder.Build(),
		InitialState: initialState,
		TransitionFn: transitionFn,
	}
}

// =============================================================================
// Event Builder Pattern (Additive - for better developer experience)
// =============================================================================

// EventBuilder provides a fluent interface for building events
type EventBuilder struct {
	eventType string
	tags      map[string]string
	data      any
}

// NewEvent creates a new EventBuilder for fluent event construction
func NewEvent(eventType string) *EventBuilder {
	return &EventBuilder{
		eventType: eventType,
		tags:      make(map[string]string),
	}
}

// WithTag adds a single tag to the event
func (eb *EventBuilder) WithTag(key, value string) *EventBuilder {
	eb.tags[key] = value
	return eb
}

// WithTags adds multiple tags to the event
func (eb *EventBuilder) WithTags(tags map[string]string) *EventBuilder {
	for key, value := range tags {
		eb.tags[key] = value
	}
	return eb
}

// WithData sets the event data (will be JSON marshaled)
func (eb *EventBuilder) WithData(data any) *EventBuilder {
	eb.data = data
	return eb
}

// Build creates the final InputEvent
func (eb *EventBuilder) Build() InputEvent {
	tags := make([]Tag, 0, len(eb.tags))
	for key, value := range eb.tags {
		tags = append(tags, NewTag(key, value))
	}

	var data []byte
	if eb.data != nil {
		data = ToJSON(eb.data)
	}

	return NewInputEvent(eb.eventType, tags, data)
}

// =============================================================================
// Batch Builder Pattern (Additive - for better developer experience)
// =============================================================================

// BatchBuilder provides a fluent interface for building event batches
type BatchBuilder struct {
	events []InputEvent
}

// NewBatch creates a new BatchBuilder for fluent batch construction
func NewBatch() *BatchBuilder {
	return &BatchBuilder{
		events: make([]InputEvent, 0),
	}
}

// AddEvent adds a single event to the batch
func (bb *BatchBuilder) AddEvent(event InputEvent) *BatchBuilder {
	bb.events = append(bb.events, event)
	return bb
}

// AddEvents adds multiple events to the batch
func (bb *BatchBuilder) AddEvents(events ...InputEvent) *BatchBuilder {
	bb.events = append(bb.events, events...)
	return bb
}

// AddEventFromBuilder adds an event from an EventBuilder to the batch
func (bb *BatchBuilder) AddEventFromBuilder(builder *EventBuilder) *BatchBuilder {
	bb.events = append(bb.events, builder.Build())
	return bb
}

// Build creates the final event batch
func (bb *BatchBuilder) Build() []InputEvent {
	return bb.events
}
