package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"go-crablet/pkg/dcb"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Set log level to only show WARN and ERROR
func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	// Only show WARN and ERROR level logs
	log.SetOutput(os.Stderr)
}

type Server struct {
	store dcb.EventStore
}

func main() {
	// Get database connection string from environment
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/dcb_app?sslmode=disable"
	}

	// Configure connection pool for performance
	config, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		log.Fatalf("Failed to parse database URL: %v", err)
	}

	// Optimize connection pool for high throughput with adaptive sizing
	// Base configuration on available system resources
	maxConns := 50 // Reduced from 300 to prevent exhaustion
	minConns := 10 // Reduced from 100 to prevent exhaustion

	// Adaptive sizing based on environment or system resources
	if maxConnsEnv := os.Getenv("DB_MAX_CONNS"); maxConnsEnv != "" {
		if parsed, err := strconv.Atoi(maxConnsEnv); err == nil && parsed > 0 {
			maxConns = parsed
		}
	}

	if minConnsEnv := os.Getenv("DB_MIN_CONNS"); minConnsEnv != "" {
		if parsed, err := strconv.Atoi(minConnsEnv); err == nil && parsed > 0 && parsed <= maxConns {
			minConns = parsed
		}
	}

	config.MaxConns = int32(maxConns)
	config.MinConns = int32(minConns)
	config.MaxConnLifetime = 10 * time.Minute // Reduced from 15 minutes
	config.MaxConnIdleTime = 5 * time.Minute  // Reduced from 10 minutes
	config.HealthCheckPeriod = 30 * time.Second

	// Connect to database with retry logic
	var pool *pgxpool.Pool
	maxRetries := 30
	retryDelay := 2 * time.Second

	for i := 0; i < maxRetries; i++ {
		// Only log on error, not on success
		pool, err = pgxpool.NewWithConfig(context.Background(), config)
		if err == nil {
			break
		}

		// Log error and retry info
		log.Printf("Failed to connect to database: %v", err)
		if i < maxRetries-1 {
			log.Printf("Retrying in %v...", retryDelay)
			time.Sleep(retryDelay)
		}
	}

	if err != nil {
		log.Fatalf("Failed to connect to database after %d attempts: %v", maxRetries, err)
	}
	defer pool.Close()

	// Create event store
	store, err := dcb.NewEventStore(context.Background(), pool)
	if err != nil {
		log.Fatalf("Failed to create event store: %v", err)
	}

	server := &Server{store: store}

	// Setup routes with optimized handlers
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-cache")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	http.HandleFunc("/cleanup", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Truncate the events table and reset the position sequence
		_, err := pool.Exec(context.Background(), `
			TRUNCATE TABLE events RESTART IDENTITY CASCADE;
		`)

		if err != nil {
			log.Printf("Failed to cleanup database: %v", err)
			http.Error(w, fmt.Sprintf("Failed to cleanup database: %v", err), http.StatusInternalServerError)
			return
		}

		response := map[string]interface{}{
			"success": true,
			"message": "Database cleaned up successfully",
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-cache")
		json.NewEncoder(w).Encode(response)
		log.Printf("Database cleaned up successfully")
	})

	http.HandleFunc("/read", server.handleRead)
	http.HandleFunc("/append", server.handleAppend)

	// Get port from environment
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Log startup information
	log.Printf("Starting go-crablet web-app server on port %s", port)
	log.Printf("Database connected successfully (pool: %d-%d connections)", minConns, maxConns)

	// Configure HTTP server for high performance
	httpServer := &http.Server{
		Addr:           ":" + port,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		IdleTimeout:    120 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1MB
	}

	// Start server with startup log
	log.Printf("Server listening on http://localhost:%s", port)
	log.Printf("Health check available at http://localhost:%s/health", port)
	log.Fatal(httpServer.ListenAndServe())
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
	if len(query.Items) == 0 {
		return dcb.NewQueryEmpty()
	}
	items := make([]dcb.QueryItem, 0, len(query.Items))
	for _, item := range query.Items {
		// Convert EventTypes ([]EventType) to []string
		types := make([]string, len(item.Types))
		for i, eventType := range item.Types {
			types[i] = string(eventType)
		}
		items = append(items, dcb.NewQueryItem(types, convertTags(item.Tags)))
	}
	return dcb.NewQueryFromItems(items...)
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

func convertAppendCondition(condition *AppendCondition) dcb.AppendCondition {
	if condition == nil {
		return nil
	}

	var after *int64
	if condition.After != nil {
		// In a real implementation, you'd need to convert EventId to position
		pos := int64(0) // This should be looked up from the EventId
		after = &pos
	}

	// Check if the original query is empty before converting
	isQueryEmpty := len(condition.FailIfEventsMatch.Items) == 0

	switch {
	case !isQueryEmpty && after != nil:
		query := convertQuery(condition.FailIfEventsMatch)
		queryPtr := &query
		return dcb.NewAppendConditionWithAfter(queryPtr, after)
	case !isQueryEmpty:
		query := convertQuery(condition.FailIfEventsMatch)
		queryPtr := &query
		return dcb.NewAppendCondition(queryPtr)
	case after != nil:
		return dcb.NewAppendConditionAfter(after)
	default:
		return nil
	}
}

func convertInputEvent(event Event) dcb.InputEvent {
	return dcb.NewInputEvent(string(event.Type), convertTags(event.Tags), []byte(event.Data))
}

func convertInputEvents(events interface{}) ([]dcb.InputEvent, error) {
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

		return []dcb.InputEvent{dcb.NewInputEvent(eventType, convertTags(tagsSlice), []byte(data))}, nil

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

			result[i] = dcb.NewInputEvent(eventType, convertTags(tagsSlice), []byte(data))
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
	query := convertQuery(req.Query)
	options := convertReadOptions(req.Options)

	// Execute read
	ctx := context.Background()
	result, err := s.store.Read(ctx, query, options)

	duration := time.Since(start)
	durationMicroseconds := duration.Microseconds()

	if err != nil {
		// Provide more specific error responses
		if _, ok := err.(*dcb.ValidationError); ok {
			http.Error(w, fmt.Sprintf("Validation error: %v", err), http.StatusBadRequest)
		} else if _, ok := err.(*dcb.ResourceError); ok {
			http.Error(w, fmt.Sprintf("Resource error: %v", err), http.StatusInternalServerError)
		} else {
			http.Error(w, fmt.Sprintf("Read failed: %v", err), http.StatusInternalServerError)
		}
		return
	}

	response := ReadResponse{
		DurationInMicroseconds: durationMicroseconds,
		NumberOfMatchingEvents: len(result.Events),
	}

	// Only set checkpoint if we have events
	if len(result.Events) > 0 {
		lastPosition := result.Events[len(result.Events)-1].Position
		lastEventID := EventId(fmt.Sprintf("%d", lastPosition))
		response.CheckpointEventId = &lastEventID
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

	// Robustly decode events with better error handling
	var inputEvents []dcb.InputEvent
	{
		var single Event
		if singleErr := json.Unmarshal(req.Events, &single); singleErr == nil && single.Type != "" {
			inputEvents = []dcb.InputEvent{convertInputEvent(single)}
		} else {
			var many Events
			if manyErr := json.Unmarshal(req.Events, &many); manyErr == nil && len(many) > 0 {
				inputEvents = make([]dcb.InputEvent, len(many))
				for i, ev := range many {
					inputEvents[i] = convertInputEvent(ev)
				}
			} else {
				http.Error(w, "Invalid events: must be a single event or array of events", http.StatusBadRequest)
				return
			}
		}
	}

	start := time.Now()
	condition := convertAppendCondition(req.Condition)

	// Execute append
	ctx := context.Background()
	err := s.store.AppendIf(ctx, inputEvents, condition)

	duration := time.Since(start)
	durationMicroseconds := duration.Microseconds()

	// Check if it was a concurrency error
	appendConditionFailed := false
	if err != nil {
		if _, ok := err.(*dcb.ConcurrencyError); ok {
			appendConditionFailed = true
		} else {
			// Provide more specific error responses
			if _, ok := err.(*dcb.ValidationError); ok {
				http.Error(w, fmt.Sprintf("Validation error: %v", err), http.StatusBadRequest)
			} else if _, ok := err.(*dcb.ResourceError); ok {
				http.Error(w, fmt.Sprintf("Resource error: %v", err), http.StatusInternalServerError)
			} else {
				http.Error(w, fmt.Sprintf("Append failed: %v", err), http.StatusInternalServerError)
			}
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
