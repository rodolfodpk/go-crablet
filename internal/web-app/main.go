package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
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
	pool  *pgxpool.Pool
}

func main() {
	// Create context with timeout for the entire application
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get database configuration from environment
	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		dbHost = "localhost"
	}
	dbPort := os.Getenv("DB_PORT")
	if dbPort == "" {
		dbPort = "5432"
	}
	dbUser := os.Getenv("DB_USER")
	if dbUser == "" {
		dbUser = "postgres"
	}
	dbPassword := os.Getenv("DB_PASSWORD")
	if dbPassword == "" {
		dbPassword = "postgres"
	}
	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		dbName = "dcb_app"
	}

	// Get connection pool configuration from environment
	maxConns := 20
	if maxConnsStr := os.Getenv("DB_MAX_CONNS"); maxConnsStr != "" {
		if parsed, err := strconv.Atoi(maxConnsStr); err == nil {
			maxConns = parsed
		}
	}

	minConns := 5
	if minConnsStr := os.Getenv("DB_MIN_CONNS"); minConnsStr != "" {
		if parsed, err := strconv.Atoi(minConnsStr); err == nil {
			minConns = parsed
		}
	}

	// Build connection string
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", dbUser, dbPassword, dbHost, dbPort, dbName)

	// Parse connection string and configure pool
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		log.Fatalf("Failed to parse database config: %v", err)
	}

	// Configure connection pool settings
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
		pool, err = pgxpool.NewWithConfig(ctx, config)
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
	store, err := dcb.NewEventStore(ctx, pool)
	if err != nil {
		log.Fatalf("Failed to create event store: %v", err)
	}

	server := &Server{store: store, pool: pool}

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
		_, err := pool.Exec(r.Context(), `
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
		result[i] = dcb.NewTag(key, value)
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

	var cursor *dcb.Cursor
	if options.From != nil {
		// In a real implementation, you'd need to convert EventId to cursor
		// For now, we'll use a default cursor
		cursor = &dcb.Cursor{
			TransactionID: 0,
			Position:      0,
		}
	}

	return &dcb.ReadOptions{
		Cursor: cursor,
	}
}

func convertAppendCondition(condition *AppendCondition) dcb.AppendCondition {
	if condition == nil {
		return nil
	}

	// Check if the original query is empty before converting
	isQueryEmpty := len(condition.FailIfEventsMatch.Items) == 0

	if !isQueryEmpty {
		query := convertQuery(condition.FailIfEventsMatch)
		return dcb.NewAppendCondition(query)
	}

	return nil
}

func convertInputEvent(event Event) dcb.InputEvent {
	return dcb.NewInputEvent(string(event.Type), convertTags(event.Tags), []byte(event.Data))
}

func convertInputEvents(events interface{}) ([]dcb.InputEvent, error) {
	// If events is nil, return error
	if events == nil {
		return nil, fmt.Errorf("events is nil")
	}

	// If events is already a slice, use it
	switch v := events.(type) {
	case []interface{}:
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
			tagsSlice := make(Tags, len(tags))
			for j, tag := range tags {
				tagsSlice[j] = Tag(tag)
			}
			result[i] = dcb.NewInputEvent(eventType, convertTags(tagsSlice), []byte(data))
		}
		return result, nil
	case map[string]interface{}:
		// Single event, wrap in a slice
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
		tagsSlice := make(Tags, len(tags))
		for i, tag := range tags {
			tagsSlice[i] = Tag(tag)
		}
		return []dcb.InputEvent{dcb.NewInputEvent(eventType, convertTags(tagsSlice), []byte(data))}, nil
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
	ctx := r.Context()
	result, err := s.store.ReadWithOptions(ctx, query, options)

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
		NumberOfMatchingEvents: len(result),
	}

	// Only set checkpoint if we have events
	if len(result) > 0 {
		lastPosition := result[len(result)-1].Position
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

	start := time.Now()
	var req AppendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Unmarshal req.Events (json.RawMessage) into interface{}
	var eventsAny interface{}
	if err := json.Unmarshal(req.Events, &eventsAny); err != nil {
		log.Printf("Failed to unmarshal events: %v", err)
		http.Error(w, "Invalid events", http.StatusBadRequest)
		return
	}

	inputEvents, err := convertInputEvents(eventsAny)
	if err != nil {
		log.Printf("convertInputEvents error: %v, events type: %T", err, eventsAny)
		http.Error(w, "Invalid events", http.StatusBadRequest)
		return
	}
	condition := convertAppendCondition(req.Condition)

	// Check if any events have lock: tags to determine if we should use advisory locks
	useAdvisoryLocks := hasLockTags(inputEvents)

	// Determine append method based on headers and lock tags
	var appendErr error
	if useAdvisoryLocks {
		// Use advisory lock function when lock: tags are present
		appendErr = s.appendWithAdvisoryLocks(r.Context(), inputEvents, condition)
	} else if condition != nil {
		// Check for isolation level header for conditional appends
		isoHeader := r.Header.Get("X-Append-If-Isolation")
		if strings.ToLower(isoHeader) == "serializable" {
			appendErr = s.store.AppendIfIsolated(r.Context(), inputEvents, condition)
		} else {
			appendErr = s.store.AppendIf(r.Context(), inputEvents, condition)
		}
	} else {
		// Simple append without conditions
		appendErr = s.store.Append(r.Context(), inputEvents)
	}
	duration := time.Since(start)

	resp := AppendResponse{
		DurationInMicroseconds: duration.Microseconds(),
		AppendConditionFailed:  false,
	}

	if appendErr != nil {
		if _, ok := appendErr.(*dcb.ConcurrencyError); ok {
			resp.AppendConditionFailed = true
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(resp)
			return
		}
		http.Error(w, appendErr.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// hasLockTags checks if any events have tags starting with "lock:"
func hasLockTags(events []dcb.InputEvent) bool {
	for _, event := range events {
		for _, tag := range event.GetTags() {
			if strings.HasPrefix(tag.GetKey(), "lock:") {
				return true
			}
		}
	}
	return false
}

// appendWithAdvisoryLocks calls the PostgreSQL advisory lock function directly
func (s *Server) appendWithAdvisoryLocks(ctx context.Context, events []dcb.InputEvent, condition dcb.AppendCondition) error {
	// Prepare data for the function
	types := make([]string, len(events))
	tags := make([]string, len(events))
	data := make([][]byte, len(events))

	for i, event := range events {
		types[i] = event.GetType()
		tags[i] = encodeTagsArrayLiteral(event.GetTags())
		data[i] = event.GetData()
	}

	// Convert condition to JSON
	var conditionJSON []byte
	var err error
	if condition != nil {
		conditionJSON, err = json.Marshal(condition)
		if err != nil {
			return fmt.Errorf("failed to marshal condition: %w", err)
		}
	}

	// Call the advisory lock function directly
	_, err = s.pool.Exec(ctx, `
		SELECT append_events_with_advisory_locks($1, $2, $3, $4)
	`, types, tags, data, conditionJSON)

	if err != nil {
		// Check if it's a condition violation error
		if strings.Contains(err.Error(), "append condition violated") {
			return &dcb.ConcurrencyError{
				EventStoreError: dcb.EventStoreError{
					Op:  "append",
					Err: fmt.Errorf("append condition violated: %w", err),
				},
			}
		}

		return fmt.Errorf("failed to append events with advisory locks: %w", err)
	}

	return nil
}

// encodeTagsArrayLiteral converts tags to PostgreSQL array literal format
func encodeTagsArrayLiteral(tags []dcb.Tag) string {
	result := "{"
	for i, tag := range tags {
		if i > 0 {
			result += ","
		}
		result += "\"" + tag.GetKey() + ":" + tag.GetValue() + "\""
	}
	result += "}"
	return result
}
