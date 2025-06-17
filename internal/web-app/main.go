package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"go-crablet/pkg/dcb"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Server struct {
	store dcb.EventStore
}

func main() {
	// Get database connection string from environment
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/dcb_app?sslmode=disable"
	}

	// Connect to database
	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	// Create event store
	store, err := dcb.NewEventStore(context.Background(), pool)
	if err != nil {
		log.Fatalf("Failed to create event store: %v", err)
	}

	server := &Server{store: store}

	// Setup routes
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})
	http.HandleFunc("/read", server.handleRead)
	http.HandleFunc("/append", server.handleAppend)

	// Get port from environment
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting DCB Bench server on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

// OpenAPI Schema Types
type EventType string
type EventTypes []EventType
type Tag string
type Tags []Tag

type QueryItem struct {
	Types EventTypes `json:"types"`
	Tags  Tags       `json:"tags"`
}

type QueryItems []QueryItem

type Query struct {
	Items QueryItems `json:"items"`
}

type EventId string

type ReadOptions struct {
	From      *EventId `json:"from,omitempty"`
	Backwards *bool    `json:"backwards,omitempty"`
}

type ReadRequest struct {
	Query   Query        `json:"query"`
	Options *ReadOptions `json:"options,omitempty"`
}

type ReadResponse struct {
	DurationInMicroseconds int64    `json:"durationInMicroseconds"`
	NumberOfMatchingEvents int      `json:"numberOfMatchingEvents"`
	CheckpointEventId      *EventId `json:"checkpointEventId,omitempty"`
}

type Event struct {
	Id   EventId   `json:"id"`
	Type EventType `json:"type"`
	Data string    `json:"data"`
	Tags Tags      `json:"tags"`
}

type Events []Event

type AppendCondition struct {
	FailIfEventsMatch Query    `json:"failIfEventsMatch"`
	After             *EventId `json:"after,omitempty"`
}

type AppendRequest struct {
	Events    json.RawMessage  `json:"events"` // Can be Event or []Event
	Condition *AppendCondition `json:"condition,omitempty"`
}

type AppendResponse struct {
	DurationInMicroseconds int64 `json:"durationInMicroseconds"`
	AppendConditionFailed  bool  `json:"appendConditionFailed"`
}

// Convert OpenAPI types to DCB types
func convertTags(tags Tags) []dcb.Tag {
	result := make([]dcb.Tag, len(tags))
	for i, tag := range tags {
		// Parse tag in format "key:value"
		key, value := parseTag(string(tag))
		result[i] = dcb.Tag{Key: key, Value: value}
	}
	return result
}

func parseTag(tag string) (string, string) {
	// Simple parsing for "key:value" format
	for i := 0; i < len(tag); i++ {
		if tag[i] == ':' {
			return tag[:i], tag[i+1:]
		}
	}
	return tag, "" // If no colon found, use entire string as key
}

func convertQuery(query Query) dcb.Query {
	items := make([]dcb.QueryItem, len(query.Items))
	for i, item := range query.Items {
		eventTypes := make([]string, len(item.Types))
		for j, eventType := range item.Types {
			eventTypes[j] = string(eventType)
		}
		items[i] = dcb.QueryItem{
			EventTypes: eventTypes,
			Tags:       convertTags(item.Tags),
		}
	}
	return dcb.Query{Items: items}
}

func convertReadOptions(options *ReadOptions) *dcb.ReadOptions {
	if options == nil {
		return nil
	}

	var fromPosition *int64
	if options.From != nil {
		// In a real implementation, you'd need to convert EventId to position
		// For now, we'll use a simple approach
		pos := int64(0) // This should be looked up from the EventId
		fromPosition = &pos
	}

	return &dcb.ReadOptions{
		FromPosition: fromPosition,
	}
}

func convertAppendCondition(condition *AppendCondition) *dcb.AppendCondition {
	if condition == nil {
		return nil
	}

	var after *int64
	if condition.After != nil {
		// In a real implementation, you'd need to convert EventId to position
		pos := int64(0) // This should be looked up from the EventId
		after = &pos
	}

	query := convertQuery(condition.FailIfEventsMatch)
	return &dcb.AppendCondition{
		FailIfEventsMatch: &query,
		After:             after,
	}
}

func convertInputEvent(event Event) dcb.InputEvent {
	return dcb.InputEvent{
		Type: string(event.Type),
		Tags: convertTags(event.Tags),
		Data: []byte(event.Data),
	}
}

func convertInputEvents(events interface{}) ([]dcb.InputEvent, error) {
	// Debug logging
	log.Printf("convertInputEvents called with type: %T, value: %+v\n", events, events)

	// Handle the case where events is already unmarshaled as our custom types
	switch v := events.(type) {
	case Event:
		return []dcb.InputEvent{convertInputEvent(v)}, nil
	case Events:
		result := make([]dcb.InputEvent, len(v))
		for i, event := range v {
			result[i] = convertInputEvent(event)
		}
		return result, nil
	}

	// Handle raw JSON unmarshaling
	switch v := events.(type) {
	case map[string]interface{}:
		// Single event
		eventType, ok := v["type"].(string)
		if !ok {
			return nil, fmt.Errorf("missing or invalid type field")
		}

		data, ok := v["data"].(string)
		if !ok {
			return nil, fmt.Errorf("missing or invalid data field")
		}

		var tags []string
		if tagsRaw, ok := v["tags"].([]interface{}); ok {
			tags = make([]string, len(tagsRaw))
			for i, tag := range tagsRaw {
				if tagStr, ok := tag.(string); ok {
					tags[i] = tagStr
				} else {
					return nil, fmt.Errorf("invalid tag at index %d", i)
				}
			}
		}

		// Convert string slice to Tags
		tagsSlice := make(Tags, len(tags))
		for i, tag := range tags {
			tagsSlice[i] = Tag(tag)
		}

		return []dcb.InputEvent{{
			Type: eventType,
			Tags: convertTags(tagsSlice),
			Data: []byte(data),
		}}, nil

	case []interface{}:
		// Array of events
		result := make([]dcb.InputEvent, len(v))
		for i, eventRaw := range v {
			eventMap, ok := eventRaw.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("invalid event at index %d", i)
			}

			eventType, ok := eventMap["type"].(string)
			if !ok {
				return nil, fmt.Errorf("missing or invalid type field at index %d", i)
			}

			data, ok := eventMap["data"].(string)
			if !ok {
				return nil, fmt.Errorf("missing or invalid data field at index %d", i)
			}

			var tags []string
			if tagsRaw, ok := eventMap["tags"].([]interface{}); ok {
				tags = make([]string, len(tagsRaw))
				for j, tag := range tagsRaw {
					if tagStr, ok := tag.(string); ok {
						tags[j] = tagStr
					} else {
						return nil, fmt.Errorf("invalid tag at index %d in event %d", j, i)
					}
				}
			}

			// Convert string slice to Tags
			tagsSlice := make(Tags, len(tags))
			for j, tag := range tags {
				tagsSlice[j] = Tag(tag)
			}

			result[i] = dcb.InputEvent{
				Type: eventType,
				Tags: convertTags(tagsSlice),
				Data: []byte(data),
			}
		}
		return result, nil

	default:
		return nil, fmt.Errorf("invalid events type: %T", events)
	}
}

// HTTP Handlers
func (s *Server) handleRead(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ReadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	start := time.Now()

	// Convert to DCB types
	query := convertQuery(req.Query)
	options := convertReadOptions(req.Options)

	// Execute read
	ctx := context.Background()
	sequencedEvents, err := s.store.Read(ctx, query, options)
	if err != nil {
		http.Error(w, fmt.Sprintf("Read failed: %v", err), http.StatusBadRequest)
		return
	}

	duration := time.Since(start)
	durationMicroseconds := duration.Microseconds()

	// Build response
	response := ReadResponse{
		DurationInMicroseconds: durationMicroseconds,
		NumberOfMatchingEvents: len(sequencedEvents.Events),
	}

	// Set checkpoint event ID if there are events
	if len(sequencedEvents.Events) > 0 {
		lastEvent := sequencedEvents.Events[len(sequencedEvents.Events)-1]
		checkpointId := EventId(lastEvent.ID)
		response.CheckpointEventId = &checkpointId
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleAppend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req AppendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Robustly decode events with debug logging
	var inputEvents []dcb.InputEvent
	{
		var single Event
		singleErr := json.Unmarshal(req.Events, &single)
		if singleErr == nil && single.Type != "" {
			inputEvents = []dcb.InputEvent{convertInputEvent(single)}
		} else {
			var many Events
			manyErr := json.Unmarshal(req.Events, &many)
			if manyErr == nil && len(many) > 0 {
				inputEvents = make([]dcb.InputEvent, len(many))
				for i, ev := range many {
					inputEvents[i] = convertInputEvent(ev)
				}
			} else {
				log.Printf("Append decode error: singleErr=%v, manyErr=%v, raw=%s", singleErr, manyErr, string(req.Events))
				http.Error(w, "Invalid events: must be a single event or array of events", http.StatusBadRequest)
				return
			}
		}
	}

	start := time.Now()
	condition := convertAppendCondition(req.Condition)

	// Execute append
	ctx := context.Background()
	_, err := s.store.Append(ctx, inputEvents, condition)

	duration := time.Since(start)
	durationMicroseconds := duration.Microseconds()

	// Check if it was a concurrency error
	appendConditionFailed := false
	if err != nil {
		if _, ok := err.(*dcb.ConcurrencyError); ok {
			appendConditionFailed = true
		} else {
			http.Error(w, fmt.Sprintf("Append failed: %v", err), http.StatusBadRequest)
			return
		}
	}

	response := AppendResponse{
		DurationInMicroseconds: durationMicroseconds,
		AppendConditionFailed:  appendConditionFailed,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
