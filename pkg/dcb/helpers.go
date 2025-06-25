package dcb

import (
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewTags creates a slice of tags from key-value pairs.
// Validation is performed when the tags are used in EventStore operations.
func NewTags(kv ...string) []Tag {
	if len(kv)%2 != 0 {
		// Return empty tags instead of panicking - validation will happen in EventStore operations
		return []Tag{}
	}
	tags := make([]Tag, len(kv)/2)
	for i := 0; i < len(kv); i += 2 {
		tags[i/2] = Tag{Key: kv[i], Value: kv[i+1]}
	}
	return tags
}

// ToArray converts a slice of Tags to a PostgreSQL TEXT[] array
func TagsToArray(tags []Tag) []string {
	if len(tags) == 0 {
		return []string{}
	}

	result := make([]string, len(tags))
	for i, tag := range tags {
		result[i] = tag.Key + ":" + tag.Value
	}

	// Sort for consistent ordering
	sort.Strings(result)
	return result
}

// ParseTagsArray converts a PostgreSQL TEXT[] array back to a slice of Tags
func ParseTagsArray(arr []string) []Tag {
	if len(arr) == 0 {
		return []Tag{}
	}

	tags := make([]Tag, 0, len(arr))
	for _, item := range arr {
		if item == "" {
			continue
		}

		// Split on first ":" only to handle values with colons
		parts := strings.SplitN(item, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := parts[1] // Keep original value (including colons)
			if key != "" {
				tags = append(tags, Tag{Key: key, Value: value})
			}
		}
	}
	return tags
}

// NewQuery creates a new Query with the given tags and event types.
// This is a backward-compatible function that creates a single QueryItem.
func NewQuery(tags []Tag, eventTypes ...string) Query {
	return Query{
		Items: []QueryItem{
			{
				EventTypes: eventTypes,
				Tags:       tags,
			},
		},
	}
}

// NewQuerySimple creates a new Query with the given tags and event types.
// This is a convenience function that combines NewTags and NewQuery.
// Validation is performed when the query is used in EventStore operations.
func NewQuerySimple(tags []Tag, eventTypes ...string) Query {
	// Remove validation from constructor - validation will happen in EventStore operations
	return NewQuery(tags, eventTypes...)
}

// NewQuerySimpleUnsafe creates a new Query without validation.
// Use this only when you're certain the data is valid and you need maximum performance.
func NewQuerySimpleUnsafe(tags []Tag, eventTypes ...string) Query {
	return NewQuery(tags, eventTypes...)
}

// NewQueryEmpty creates a new empty query
func NewQueryEmpty() Query {
	return Query{Items: []QueryItem{}}
}

// NewQueryFromItems creates a new query from a list of query items
func NewQueryFromItems(items ...QueryItem) Query {
	return Query{Items: items}
}

// NewQueryAll creates a query that matches all events.
func NewQueryAll() Query {
	return Query{
		Items: []QueryItem{
			{
				EventTypes: []string{},
				Tags:       []Tag{},
			},
		},
	}
}

// NewQueryItem creates a new QueryItem with the given types and tags.
func NewQueryItem(types []string, tags []Tag) QueryItem {
	return QueryItem{
		EventTypes: types,
		Tags:       tags,
	}
}

// NewInputEvent creates a new InputEvent with the given type, tags, and data.
// Validation is performed when the event is used in EventStore operations.
func NewInputEvent(eventType string, tags []Tag, data []byte) InputEvent {
	return InputEvent{
		Type: eventType,
		Tags: tags,
		Data: data,
	}
}

// NewInputEventUnsafe creates a new InputEvent without validation.
// Use this only when you're certain the data is valid and you need maximum performance.
func NewInputEventUnsafe(eventType string, tags []Tag, data []byte) InputEvent {
	return InputEvent{
		Type: eventType,
		Tags: tags,
		Data: data,
	}
}

// NewEventBatch creates a slice of events from the given InputEvents.
// This is a convenience function for creating event batches, particularly useful
// when appending multiple related events in a single operation.
func NewEventBatch(events ...InputEvent) []InputEvent {
	return events
}

// NewQItem creates a new QueryItem with a single event type and tags.
// This simplifies the common case of querying for one event type.
func NewQItem(eventType string, tags []Tag) QueryItem {
	return QueryItem{
		EventTypes: []string{eventType},
		Tags:       tags,
	}
}

// NewQItemKV creates a new QueryItem with a single event type and key-value tags.
// This is the most concise way to create a QueryItem for a single event type.
func NewQItemKV(eventType string, kv ...string) QueryItem {
	return QueryItem{
		EventTypes: []string{eventType},
		Tags:       NewTags(kv...),
	}
}

// ConnectionPoolHealth represents the health status of a connection pool
type ConnectionPoolHealth struct {
	TotalConns        int32
	IdleConns         int32
	AcquiredConns     int32
	ConstructingConns int32
	Healthy           bool
	Message           string
}

// CheckConnectionPoolHealth checks the health of a connection pool
func CheckConnectionPoolHealth(pool *pgxpool.Pool) ConnectionPoolHealth {
	stats := pool.Stat()

	health := ConnectionPoolHealth{
		TotalConns:        stats.TotalConns(),
		IdleConns:         stats.IdleConns(),
		AcquiredConns:     stats.AcquiredConns(),
		ConstructingConns: stats.ConstructingConns(),
		Healthy:           true,
	}

	// Check for potential connection leaks (high acquired connections)
	if stats.AcquiredConns() > stats.TotalConns()*80/100 {
		health.Healthy = false
		health.Message = "High number of acquired connections - potential connection leak"
	}

	// Check for no idle connections available
	if stats.IdleConns() == 0 && stats.AcquiredConns() > 0 {
		health.Healthy = false
		health.Message = "No idle connections available - pool may be exhausted"
	}

	// Check for connections being constructed (normal during startup/load)
	if stats.ConstructingConns() > 0 {
		if stats.ConstructingConns() > 5 {
			health.Healthy = false
			health.Message = fmt.Sprintf("High number of connections being constructed: %d", stats.ConstructingConns())
		} else {
			health.Message = fmt.Sprintf("Building %d new connections", stats.ConstructingConns())
		}
	}

	// Check for pool exhaustion (all connections acquired)
	if stats.AcquiredConns() == stats.TotalConns() && stats.TotalConns() > 0 {
		health.Healthy = false
		health.Message = "Connection pool exhausted - all connections acquired"
	}

	return health
}

// LogConnectionPoolHealth logs the health status of a connection pool
func LogConnectionPoolHealth(pool *pgxpool.Pool, operation string) {
	health := CheckConnectionPoolHealth(pool)

	if health.Healthy {
		log.Printf("[INFO] [%s] Pool healthy - Total: %d, Idle: %d, Acquired: %d, Constructing: %d",
			operation, health.TotalConns, health.IdleConns, health.AcquiredConns, health.ConstructingConns)
	} else {
		log.Printf("[WARN] [%s] Pool unhealthy - %s - Total: %d, Idle: %d, Acquired: %d, Constructing: %d",
			operation, health.Message, health.TotalConns, health.IdleConns, health.AcquiredConns, health.ConstructingConns)
	}
}

// LogConnectionPoolHealthDebug logs detailed connection pool information at DEBUG level
func LogConnectionPoolHealthDebug(pool *pgxpool.Pool, operation string) {
	health := CheckConnectionPoolHealth(pool)

	log.Printf("[DEBUG] [%s] Pool details - Total: %d, Idle: %d, Acquired: %d, Constructing: %d, Healthy: %t",
		operation, health.TotalConns, health.IdleConns, health.AcquiredConns, health.ConstructingConns, health.Healthy)

	if !health.Healthy {
		log.Printf("[DEBUG] [%s] Health issue: %s", operation, health.Message)
	}
}

// LogConnectionPoolHealthWithLevel logs connection pool health with specified log level
func LogConnectionPoolHealthWithLevel(pool *pgxpool.Pool, operation string, level string) {
	health := CheckConnectionPoolHealth(pool)

	// Determine log level based on health status and specified level
	logLevel := level
	if !health.Healthy {
		if health.Message == "Connection pool exhausted - all connections acquired" {
			logLevel = "ERROR"
		} else if health.Message == "High number of acquired connections - potential connection leak" {
			logLevel = "WARN"
		} else {
			logLevel = "WARN"
		}
	}

	log.Printf("[%s] [%s] Pool status - %s - Total: %d, Idle: %d, Acquired: %d, Constructing: %d",
		logLevel, operation, health.Message, health.TotalConns, health.IdleConns, health.AcquiredConns, health.ConstructingConns)
}
