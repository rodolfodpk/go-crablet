package dcb

import (
	"encoding/json"
	"fmt"
	"time"
)

// =============================================================================
// SUPPORTING INTERFACES AND TYPES
// =============================================================================

// Tag represents a key-value pair for event categorization
// This is now an opaque type: construct only via NewTag
// and access fields only via methods
type Tag interface {
	isTag()
	GetKey() string
	GetValue() string
}

// =============================================================================
// CONCRETE TYPES AND STRUCTS
// =============================================================================

// Event represents a single event in the store
type Event struct {
	Type          string    `json:"type"`
	Tags          []Tag     `json:"tags"`
	Data          []byte    `json:"data"`
	TransactionID uint64    `json:"transaction_id"`
	Position      int64     `json:"position"`
	OccurredAt    time.Time `json:"occurred_at"`
}

// Cursor represents a position in the event stream
// When used in Read/Project operations, events are returned EXCLUSIVE of this position
// (i.e., events after this cursor, not including the cursor position itself)
type Cursor struct {
	TransactionID uint64 `json:"transaction_id"`
	Position      int64  `json:"position"`
}

// =============================================================================
// CONFIGURATION TYPES
// =============================================================================

// IsolationLevel represents PostgreSQL transaction isolation levels as a type-safe enum
// Only valid values can be constructed via constants or ParseIsolationLevel
type IsolationLevel int

const (
	IsolationLevelReadCommitted IsolationLevel = iota
	IsolationLevelRepeatableRead
	IsolationLevelSerializable
)

func (l IsolationLevel) String() string {
	switch l {
	case IsolationLevelReadCommitted:
		return "READ_COMMITTED"
	case IsolationLevelRepeatableRead:
		return "REPEATABLE_READ"
	case IsolationLevelSerializable:
		return "SERIALIZABLE"
	default:
		return "UNKNOWN"
	}
}

func ParseIsolationLevel(s string) (IsolationLevel, error) {
	switch s {
	case "READ_COMMITTED":
		return IsolationLevelReadCommitted, nil
	case "REPEATABLE_READ":
		return IsolationLevelRepeatableRead, nil
	case "SERIALIZABLE":
		return IsolationLevelSerializable, nil
	default:
		return IsolationLevelReadCommitted, fmt.Errorf("invalid isolation level: %s", s)
	}
}

// EventStoreConfig contains configuration for EventStore behavior
// Organized into logical groups for append and query operations
type EventStoreConfig struct {
	// =============================================================================
	// APPEND OPERATIONS CONFIGURATION
	// =============================================================================
	
	// MaxBatchSize controls the maximum number of events that can be appended in a single batch
	// Larger batches improve performance but increase memory usage and transaction duration
	MaxBatchSize int `json:"max_batch_size"`
	
	// DefaultAppendIsolation sets the PostgreSQL transaction isolation level for append operations
	// Higher isolation levels provide stronger consistency guarantees but may impact performance
	DefaultAppendIsolation IsolationLevel `json:"default_append_isolation"`
	
	// AppendTimeout sets the maximum time (in milliseconds) for append operations to complete
	// This is a defensive timeout to prevent hanging appends
	AppendTimeout int `json:"append_timeout"`
	
	// =============================================================================
	// QUERY OPERATIONS CONFIGURATION  
	// =============================================================================
	
	// QueryTimeout sets the maximum time (in milliseconds) for query operations to complete
	// This is a defensive timeout to prevent hanging queries
	QueryTimeout int `json:"query_timeout"`
	
	// StreamBuffer sets the channel buffer size for streaming operations (QueryStream, ProjectStream)
	// Larger buffers improve throughput but increase memory usage
	StreamBuffer int `json:"stream_buffer"`
	
	// =============================================================================
	// FUTURE EXTENSIBILITY (Optional fields)
	// =============================================================================
	
	// QueryCacheSize sets the size of the query result cache (0 = disabled)
	// Caching can improve performance for repeated queries
	QueryCacheSize int `json:"query_cache_size,omitempty"`
	
	// AppendRetryAttempts sets the number of retry attempts for append conflicts (0 = no retry)
	// Useful for handling concurrent modification scenarios
	AppendRetryAttempts int `json:"append_retry_attempts,omitempty"`
	
	// MaxConcurrentQueries limits the number of concurrent query operations (0 = unlimited)
	// Helps prevent overwhelming the database with too many simultaneous queries
	MaxConcurrentQueries int `json:"max_concurrent_queries,omitempty"`
}

// =============================================================================
// INTERNAL IMPLEMENTATIONS (Private)
// =============================================================================

type tag struct {
	key   string
	value string
}

func (t *tag) isTag()           {}
func (t *tag) GetKey() string   { return t.key }
func (t *tag) GetValue() string { return t.value }

// MarshalJSON ensures Tag is marshaled as {"key":..., "value":...}
func (t *tag) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}{
		Key:   t.key,
		Value: t.value,
	})
}
