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

	"github.com/rodolfodpk/go-crablet/internal/benchmarks/setup"
	"github.com/rodolfodpk/go-crablet/pkg/dcb"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Set log level to only show WARN and ERROR
func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	// Only show WARN and ERROR level logs
	log.SetOutput(os.Stderr)
}

// Configuration for concurrency error logging
var (
	// Set to true to log concurrency errors (expected behavior)
	// Set to false to suppress them (they're normal in high-concurrency scenarios)
	logConcurrencyErrors = os.Getenv("LOG_CONCURRENCY_ERRORS") == "true"
)

type Server struct {
	storeReadCommitted  dcb.EventStore
	storeRepeatableRead dcb.EventStore
	storeSerializable   dcb.EventStore
	pool                *pgxpool.Pool
}

func main() {
	// Create context for the entire application (no timeout for server lifetime)
	ctx := context.Background()

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
		dbUser = "crablet"
	}
	dbPassword := os.Getenv("DB_PASSWORD")
	if dbPassword == "" {
		dbPassword = "crablet"
	}
	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		dbName = "crablet"
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

	// Create event stores for each isolation level
	storeReadCommitted, err := dcb.NewEventStoreWithConfig(ctx, pool, dcb.EventStoreConfig{
		MaxBatchSize:           1000,
		LockTimeout:            5000,
		StreamBuffer:           1000,
		DefaultAppendIsolation: dcb.IsolationLevelReadCommitted,
	})
	if err != nil {
		log.Fatalf("Failed to create ReadCommitted store: %v", err)
	}

	storeRepeatableRead, err := dcb.NewEventStoreWithConfig(ctx, pool, dcb.EventStoreConfig{
		MaxBatchSize:           1000,
		LockTimeout:            5000,
		StreamBuffer:           1000,
		DefaultAppendIsolation: dcb.IsolationLevelRepeatableRead,
	})
	if err != nil {
		log.Fatalf("Failed to create RepeatableRead store: %v", err)
	}

	storeSerializable, err := dcb.NewEventStoreWithConfig(ctx, pool, dcb.EventStoreConfig{
		MaxBatchSize:           1000,
		LockTimeout:            5000,
		StreamBuffer:           1000,
		DefaultAppendIsolation: dcb.IsolationLevelSerializable,
	})
	if err != nil {
		log.Fatalf("Failed to create Serializable store: %v", err)
	}

	server := &Server{
		storeReadCommitted:  storeReadCommitted,
		storeRepeatableRead: storeRepeatableRead,
		storeSerializable:   storeSerializable,
		pool:                pool,
	}

	// Setup routes with optimized handlers
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		// Add timeout to health check context
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		r = r.WithContext(ctx)

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-cache")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	http.HandleFunc("/cleanup", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Add timeout to cleanup context
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		// Truncate the events table and reset the position sequence
		_, err := pool.Exec(ctx, `
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

	http.HandleFunc("/load-test-data", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Add timeout to load test data context
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()

		// Get dataset size from query parameter
		datasetSize := r.URL.Query().Get("size")
		if datasetSize == "" {
			datasetSize = "tiny" // Default to tiny dataset
		}

		// Validate dataset size
		config, exists := setup.DatasetSizes[datasetSize]
		if !exists {
			http.Error(w, fmt.Sprintf("Invalid dataset size: %s. Available: tiny, small", datasetSize), http.StatusBadRequest)
			return
		}

		// Initialize SQLite cache
		if err := setup.InitGlobalCache(); err != nil {
			log.Printf("Failed to initialize cache: %v", err)
			http.Error(w, fmt.Sprintf("Failed to initialize cache: %v", err), http.StatusInternalServerError)
			return
		}

		// Get dataset from cache (or generate if not cached)
		dataset, err := setup.GetCachedDataset(config)
		if err != nil {
			log.Printf("Failed to get cached dataset: %v", err)
			http.Error(w, fmt.Sprintf("Failed to get cached dataset: %v", err), http.StatusInternalServerError)
			return
		}

		// Load dataset into PostgreSQL
		if err := setup.LoadDatasetIntoStore(ctx, server.storeReadCommitted, dataset); err != nil {
			log.Printf("Failed to load dataset into store: %v", err)
			http.Error(w, fmt.Sprintf("Failed to load dataset into store: %v", err), http.StatusInternalServerError)
			return
		}

		response := map[string]interface{}{
			"success": true,
			"message": fmt.Sprintf("Test data loaded successfully: %d courses, %d students, %d enrollments",
				len(dataset.Courses), len(dataset.Students), len(dataset.Enrollments)),
			"dataset_size": datasetSize,
			"courses":      len(dataset.Courses),
			"students":     len(dataset.Students),
			"enrollments":  len(dataset.Enrollments),
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-cache")
		json.NewEncoder(w).Encode(response)
		log.Printf("Test data loaded successfully: %s dataset", datasetSize)
	})

	http.HandleFunc("/read", server.handleRead)
	http.HandleFunc("/append", server.handleAppend)
	http.HandleFunc("/project", server.handleProject)

	// Get port from environment
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Log startup information
	log.Printf("Starting github.com/rodolfodpk/go-crablet web-app server on port %s", port)
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

// ReadOptions struct removed - ReadWithOptions functionality is now handled
// by ReadChannel for streaming cases or Read for simple cases

type ReadRequest struct {
	Query Query `json:"query"`
	// Options field removed - ReadWithOptions functionality is now handled
	// by ReadChannel for streaming cases or Read for simple cases
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

type StateProjector struct {
	ID                 string      `json:"id"`
	Query              Query       `json:"query"`
	InitialState       interface{} `json:"initialState"`
	TransitionFunction string      `json:"transitionFunction"`
}

type StateProjectors []StateProjector

type ProjectRequest struct {
	Projectors StateProjectors `json:"projectors"`
	After      *EventId        `json:"after,omitempty"`
}

type ProjectResponse struct {
	DurationInMicroseconds int64                  `json:"durationInMicroseconds"`
	States                 map[string]interface{} `json:"states"`
	AppendCondition        *AppendCondition       `json:"appendCondition,omitempty"`
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

// convertReadOptions has been removed - ReadWithOptions functionality is now handled
// by ReadChannel for streaming cases or Read for simple cases

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

	// Execute read with timeout
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	result, err := s.storeReadCommitted.Query(ctx, query, nil)

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
		// Debug logging removed for performance
		http.Error(w, "Invalid events", http.StatusBadRequest)
		return
	}

	inputEvents, err := convertInputEvents(eventsAny)
	if err != nil {
		// Debug logging removed for performance
		http.Error(w, "Invalid events", http.StatusBadRequest)
		return
	}
	condition := convertAppendCondition(req.Condition)

	// Check if any events have lock: tags to determine if we should use advisory locks
	useAdvisoryLocks := hasLockTags(inputEvents)

	// Execute append with timeout
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	// Determine append method based on lock tags
	var appendErr error
	if useAdvisoryLocks {
		// Use advisory lock function when lock: tags are present
		appendErr = s.appendWithAdvisoryLocks(ctx, inputEvents, condition)
	} else {
		// Use standard append methods
		isolation := r.Header.Get("X-Append-If-Isolation")
		var store dcb.EventStore
		switch isolation {
		case "SERIALIZABLE":
			store = s.storeSerializable
		case "REPEATABLE READ":
			store = s.storeRepeatableRead
		default:
			store = s.storeReadCommitted
		}

		if condition != nil {
			// Use AppendIf with conditions
			appendErr = store.AppendIf(ctx, inputEvents, condition)
		} else {
			// Simple append without conditions
			appendErr = store.Append(ctx, inputEvents)
		}
	}
	duration := time.Since(start)

	resp := AppendResponse{
		DurationInMicroseconds: duration.Microseconds(),
		AppendConditionFailed:  false,
	}

	if appendErr != nil {
		if _, ok := appendErr.(*dcb.ConcurrencyError); ok {
			// Log concurrency errors only if configured to do so
			if logConcurrencyErrors {
				log.Printf("Concurrency condition failed (expected): %v", appendErr)
			}
			resp.AppendConditionFailed = true
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(resp)
			return
		}
		// Debug logging removed for performance
		http.Error(w, appendErr.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleProject(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	start := time.Now()

	// Convert projectors to DCB format
	projectors := make([]dcb.StateProjector, len(req.Projectors))
	for i, p := range req.Projectors {

		// Create transition function
		transitionFn := func(state any, event dcb.Event) any {
			// For now, implement a simple counter transition
			// In a real implementation, you'd parse and execute the JavaScript function
			if current, ok := state.(int); ok {
				return current + 1
			}
			return 1
		}

		projectors[i] = dcb.StateProjector{
			ID:           p.ID,
			Query:        convertQuery(p.Query),
			InitialState: p.InitialState,
			TransitionFn: transitionFn,
		}
	}

	// Execute projection with timeout
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	var cursor *dcb.Cursor
	if req.After != nil {
		position, err := strconv.ParseInt(string(*req.After), 10, 64)
		if err != nil {
			http.Error(w, "Invalid cursor format", http.StatusBadRequest)
			return
		}
		cursor = &dcb.Cursor{
			Position: position,
		}
	}

	states, appendCondition, err := s.storeReadCommitted.Project(ctx, projectors, cursor)

	duration := time.Since(start)
	durationMicroseconds := duration.Microseconds()

	if err != nil {
		// Provide more specific error responses
		if _, ok := err.(*dcb.ValidationError); ok {
			http.Error(w, fmt.Sprintf("Validation error: %v", err), http.StatusBadRequest)
		} else if _, ok := err.(*dcb.ResourceError); ok {
			http.Error(w, fmt.Sprintf("Resource error: %v", err), http.StatusInternalServerError)
		} else {
			http.Error(w, fmt.Sprintf("Projection failed: %v", err), http.StatusInternalServerError)
		}
		return
	}

	// Convert append condition back to API format
	var apiAppendCondition *AppendCondition
	if appendCondition != nil {
		// For now, create a simple append condition
		// In a real implementation, you'd convert the DCB append condition
		apiAppendCondition = &AppendCondition{
			FailIfEventsMatch: Query{Items: []QueryItem{}},
		}
	}

	response := ProjectResponse{
		DurationInMicroseconds: durationMicroseconds,
		States:                 states,
		AppendCondition:        apiAppendCondition,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
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

	// Get lock timeout from EventStore config
	lockTimeout := s.storeReadCommitted.GetConfig().LockTimeout // Assuming a default or common lock timeout

	// Call the advisory lock function directly with timeout
	var result []byte
	err = s.pool.QueryRow(ctx, `
		SELECT append_events_with_advisory_locks($1, $2, $3, $4, $5)
	`, types, tags, data, conditionJSON, lockTimeout).Scan(&result)

	if err != nil {
		return fmt.Errorf("failed to append events with advisory locks: %w", err)
	}

	// Check result for conditional append
	if condition != nil && len(result) > 0 {
		var resultMap map[string]interface{}
		if err := json.Unmarshal(result, &resultMap); err != nil {
			return fmt.Errorf("failed to parse advisory lock result: %w", err)
		}

		// Check if the operation was successful
		if success, ok := resultMap["success"].(bool); !ok || !success {
			// This is a concurrency violation
			return &dcb.ConcurrencyError{
				EventStoreError: dcb.EventStoreError{
					Op:  "append",
					Err: fmt.Errorf("append condition violated: %v", resultMap["message"]),
				},
			}
		}
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
