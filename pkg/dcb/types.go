package dcb

import (
	"context"
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
type EventStoreConfig struct {
	MaxBatchSize           int            `json:"max_batch_size"`
	StreamBuffer           int            `json:"stream_buffer"`            // Channel buffer size for streaming operations
	DefaultAppendIsolation IsolationLevel `json:"default_append_isolation"` // Default isolation level for Append operations
	QueryTimeout           int            `json:"query_timeout"`            // Query timeout in milliseconds (defensive against hanging queries)
	AppendTimeout          int            `json:"append_timeout"`           // Append timeout in milliseconds (defensive against hanging appends)
	// TargetEventsTable removed - always use 'events' table for maximum performance
}

// =============================================================================
// OPTIONAL CONVENIENCE INTERFACES (For user convenience, not used by core)
// =============================================================================

// CommandExecutor executes commands and generates events
// This is an optional convenience API for command-driven event generation
type CommandExecutor interface {
	ExecuteCommand(ctx context.Context, command Command, handler CommandHandler, condition *AppendCondition) ([]InputEvent, error)
	ExecuteCommandWithLocks(ctx context.Context, command Command, handler CommandHandler, locks []string, condition *AppendCondition) ([]InputEvent, error)
}

// CommandHandler handles command execution and generates events
// This is an optional convenience API for users - not used by core abstractions
type CommandHandler interface {
	Handle(ctx context.Context, store EventStore, command Command) ([]InputEvent, error)
}

// CommandHandlerFunc allows using functions as CommandHandler implementations
type CommandHandlerFunc func(ctx context.Context, store EventStore, command Command) ([]InputEvent, error)

func (f CommandHandlerFunc) Handle(ctx context.Context, store EventStore, command Command) ([]InputEvent, error) {
	return f(ctx, store, command)
}

// Command represents a command that triggers event generation
type Command interface {
	GetType() string
	GetData() []byte
	GetMetadata() map[string]interface{}
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

type command struct {
	commandType string
	data        []byte
	metadata    map[string]interface{}
}

func (c *command) GetType() string                     { return c.commandType }
func (c *command) GetData() []byte                     { return c.data }
func (c *command) GetMetadata() map[string]interface{} { return c.metadata }

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
