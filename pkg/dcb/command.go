package dcb

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
)

// =============================================================================
// COMMAND-RELATED TYPES
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

// command is the internal implementation
type command struct {
	commandType string
	data        []byte
	metadata    map[string]interface{}
}

func (c *command) GetType() string                     { return c.commandType }
func (c *command) GetData() []byte                     { return c.data }
func (c *command) GetMetadata() map[string]interface{} { return c.metadata }

type commandExecutor struct {
	eventStore EventStore
}

func NewCommandExecutor(eventStore EventStore) CommandExecutor {
	return &commandExecutor{
		eventStore: eventStore,
	}
}

func (ce *commandExecutor) ExecuteCommand(ctx context.Context, command Command, handler CommandHandler, condition *AppendCondition) ([]InputEvent, error) {
	// Validate inputs
	if command == nil {
		return nil, &ValidationError{
			EventStoreError: EventStoreError{
				Op:  "ExecuteCommand",
				Err: fmt.Errorf("command cannot be nil"),
			},
			Field: "command",
			Value: "nil",
		}
	}

	if handler == nil {
		return nil, &ValidationError{
			EventStoreError: EventStoreError{
				Op:  "ExecuteCommand",
				Err: fmt.Errorf("handler cannot be nil"),
			},
			Field: "handler",
			Value: "nil",
		}
	}

	// Validate and prepare command data FIRST (fail early)
	var commandMetadata []byte
	if command.GetMetadata() != nil {
		var err error
		commandMetadata, err = json.Marshal(command.GetMetadata())
		if err != nil {
			return nil, &ResourceError{
				EventStoreError: EventStoreError{
					Op:  "ExecuteCommand",
					Err: fmt.Errorf("failed to marshal command metadata: %w", err),
				},
				Resource: "json",
			}
		}
	}

	// Get config from EventStore
	config := ce.eventStore.GetConfig()

	// Get pool and start transaction
	pool := ce.eventStore.GetPool()
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: toPgxIsoLevel(config.DefaultAppendIsolation),
	})
	if err != nil {
		return nil, &ResourceError{
			EventStoreError: EventStoreError{
				Op:  "ExecuteCommand",
				Err: fmt.Errorf("failed to begin transaction: %w", err),
			},
			Resource: "database",
		}
	}
	defer tx.Rollback(ctx)

	// 1. Generate events using the handler with access to EventStore
	events, handlerErr := handler.Handle(ctx, ce.eventStore, command)
	if handlerErr != nil {
		return nil, &ValidationError{
			EventStoreError: EventStoreError{
				Op:  "ExecuteCommand",
				Err: handlerErr,
			},
			Field: "handler",
			Value: "error",
		}
	}

	// 3. Validate generated events
	if len(events) == 0 {
		return nil, &ValidationError{
			EventStoreError: EventStoreError{
				Op:  "ExecuteCommand",
				Err: fmt.Errorf("handler generated no events"),
			},
			Field: "events",
			Value: "empty",
		}
	}

	// Validate individual events
	for i, event := range events {
		if event.GetType() == "" {
			return nil, &ValidationError{
				EventStoreError: EventStoreError{
					Op:  "ExecuteCommand",
					Err: fmt.Errorf("event at index %d has empty type", i),
				},
				Field: "type",
				Value: "empty",
			}
		}

		// Validate tags (reuse existing validation logic)
		tagKeys := make(map[string]bool)
		for j, tag := range event.GetTags() {
			if tag.GetKey() == "" {
				return nil, &ValidationError{
					EventStoreError: EventStoreError{
						Op:  "ExecuteCommand",
						Err: fmt.Errorf("empty tag key at index %d", j),
					},
					Field: "tag.key",
					Value: "empty",
				}
			}
			if tag.GetValue() == "" {
				return nil, &ValidationError{
					EventStoreError: EventStoreError{
						Op:  "ExecuteCommand",
						Err: fmt.Errorf("empty tag value for key %s", tag.GetKey()),
					},
					Field: "tag.value",
					Value: "empty",
				}
			}
			if tagKeys[tag.GetKey()] {
				return nil, &ValidationError{
					EventStoreError: EventStoreError{
						Op:  "ExecuteCommand",
						Err: fmt.Errorf("event at index %d has duplicate tag key: %s", i, tag.GetKey()),
					},
					Field: "tag.key",
					Value: tag.GetKey(),
				}
			}
			tagKeys[tag.GetKey()] = true
		}
	}

	// 4. Append events FIRST (primary data)
	// Use type assertion to access internal appendInTx method
	es := ce.eventStore.(*eventStore)
	if condition != nil {
		err = es.appendInTx(ctx, tx, events, *condition, nil)
	} else {
		err = es.appendInTx(ctx, tx, events, nil, nil)
	}
	if err != nil {
		return nil, err // If events fail, don't store command
	}

	// 5. Store command AFTER events (metadata) - now using pre-marshaled data
	_, err = tx.Exec(ctx, `
		INSERT INTO commands (transaction_id, type, data, metadata)
		VALUES (pg_current_xact_id(), $1, $2, $3)
	`, command.GetType(), command.GetData(), commandMetadata)
	if err != nil {
		return nil, &ResourceError{
			EventStoreError: EventStoreError{
				Op:  "ExecuteCommand",
				Err: fmt.Errorf("failed to store command: %w", err),
			},
			Resource: "database",
		}
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, &ResourceError{
			EventStoreError: EventStoreError{
				Op:  "ExecuteCommand",
				Err: fmt.Errorf("failed to commit transaction: %w", err),
			},
			Resource: "database",
		}
	}

	return events, nil
}

func (ce *commandExecutor) ExecuteCommandWithLocks(ctx context.Context, command Command, handler CommandHandler, locks []string, condition *AppendCondition) ([]InputEvent, error) {
	// Validate inputs
	if command == nil {
		return nil, &ValidationError{
			EventStoreError: EventStoreError{
				Op:  "ExecuteCommandWithLocks",
				Err: fmt.Errorf("command cannot be nil"),
			},
			Field: "command",
			Value: "nil",
		}
	}

	if handler == nil {
		return nil, &ValidationError{
			EventStoreError: EventStoreError{
				Op:  "ExecuteCommandWithLocks",
				Err: fmt.Errorf("handler cannot be nil"),
			},
			Field: "handler",
			Value: "nil",
		}
	}

	// Validate locks
	if len(locks) == 0 {
		return nil, &ValidationError{
			EventStoreError: EventStoreError{
				Op:  "ExecuteCommandWithLocks",
				Err: fmt.Errorf("locks slice cannot be empty"),
			},
			Field: "locks",
			Value: "empty",
		}
	}

	// Validate and prepare command data FIRST (fail early)
	var commandMetadata []byte
	if command.GetMetadata() != nil {
		var err error
		commandMetadata, err = json.Marshal(command.GetMetadata())
		if err != nil {
			return nil, &ResourceError{
				EventStoreError: EventStoreError{
					Op:  "ExecuteCommandWithLocks",
					Err: fmt.Errorf("failed to marshal command metadata: %w", err),
				},
				Resource: "json",
			}
		}
	}

	// Get config from EventStore
	config := ce.eventStore.GetConfig()

	// Get pool and start transaction
	pool := ce.eventStore.GetPool()
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: toPgxIsoLevel(config.DefaultAppendIsolation),
	})
	if err != nil {
		return nil, &ResourceError{
			EventStoreError: EventStoreError{
				Op:  "ExecuteCommandWithLocks",
				Err: fmt.Errorf("failed to begin transaction: %w", err),
			},
			Resource: "database",
		}
	}
	defer tx.Rollback(ctx)

	// 1. Acquire advisory locks FIRST (before any other operations)
	// Sort locks to prevent deadlocks
	sortedLocks := make([]string, len(locks))
	copy(sortedLocks, locks)
	sort.Strings(sortedLocks)

	for _, lockKey := range sortedLocks {
		// Use pg_advisory_xact_lock for transaction-scoped locks
		// Convert string to hash for advisory lock
		_, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtext($1))", lockKey)
		if err != nil {
			return nil, &ResourceError{
				EventStoreError: EventStoreError{
					Op:  "ExecuteCommandWithLocks",
					Err: fmt.Errorf("failed to acquire advisory lock for key '%s': %w", lockKey, err),
				},
				Resource: "database",
			}
		}
	}

	// 2. Generate events using the handler with access to EventStore
	events, handlerErr := handler.Handle(ctx, ce.eventStore, command)
	if handlerErr != nil {
		return nil, &ValidationError{
			EventStoreError: EventStoreError{
				Op:  "ExecuteCommandWithLocks",
				Err: handlerErr,
			},
			Field: "handler",
			Value: "error",
		}
	}

	// 3. Validate generated events
	if len(events) == 0 {
		return nil, &ValidationError{
			EventStoreError: EventStoreError{
				Op:  "ExecuteCommandWithLocks",
				Err: fmt.Errorf("handler generated no events"),
			},
			Field: "events",
			Value: "empty",
		}
	}

	// 4. Validate that events do NOT contain lock: tags (since locks are handled at command level)
	for i, event := range events {
		for j, tag := range event.GetTags() {
			if strings.HasPrefix(tag.GetKey(), "lock:") {
				return nil, &ValidationError{
					EventStoreError: EventStoreError{
						Op:  "ExecuteCommandWithLocks",
						Err: fmt.Errorf("event at index %d contains lock tag '%s' at position %d - lock tags are not allowed when using ExecuteCommandWithLocks", i, tag.GetKey(), j),
					},
					Field: "event.tags",
					Value: tag.GetKey(),
				}
			}
		}
	}

	// Validate individual events (reuse existing validation logic)
	for i, event := range events {
		if event.GetType() == "" {
			return nil, &ValidationError{
				EventStoreError: EventStoreError{
					Op:  "ExecuteCommandWithLocks",
					Err: fmt.Errorf("event at index %d has empty type", i),
				},
				Field: "type",
				Value: "empty",
			}
		}

		// Validate tags (reuse existing validation logic)
		tagKeys := make(map[string]bool)
		for j, tag := range event.GetTags() {
			if tag.GetKey() == "" {
				return nil, &ValidationError{
					EventStoreError: EventStoreError{
						Op:  "ExecuteCommandWithLocks",
						Err: fmt.Errorf("empty tag key at index %d", j),
					},
					Field: "tag.key",
					Value: "empty",
				}
			}
			if tag.GetValue() == "" {
				return nil, &ValidationError{
					EventStoreError: EventStoreError{
						Op:  "ExecuteCommandWithLocks",
						Err: fmt.Errorf("empty tag value for key %s", tag.GetKey()),
					},
					Field: "tag.value",
					Value: "empty",
				}
			}
			if tagKeys[tag.GetKey()] {
				return nil, &ValidationError{
					EventStoreError: EventStoreError{
						Op:  "ExecuteCommandWithLocks",
						Err: fmt.Errorf("event at index %d has duplicate tag key: %s", i, tag.GetKey()),
					},
					Field: "tag.key",
					Value: tag.GetKey(),
				}
			}
			tagKeys[tag.GetKey()] = true
		}
	}

	// 5. Append events (primary data)
	// Use type assertion to access internal appendInTx method
	es := ce.eventStore.(*eventStore)
	if condition != nil {
		err = es.appendInTx(ctx, tx, events, *condition, nil)
	} else {
		err = es.appendInTx(ctx, tx, events, nil, nil)
	}
	if err != nil {
		return nil, err // If events fail, don't store command
	}

	// 6. Store command AFTER events (metadata) - now using pre-marshaled data
	_, err = tx.Exec(ctx, `
		INSERT INTO commands (transaction_id, type, data, metadata)
		VALUES (pg_current_xact_id(), $1, $2, $3)
	`, command.GetType(), command.GetData(), commandMetadata)
	if err != nil {
		return nil, &ResourceError{
			EventStoreError: EventStoreError{
				Op:  "ExecuteCommandWithLocks",
				Err: fmt.Errorf("failed to store command: %w", err),
			},
			Resource: "database",
		}
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, &ResourceError{
			EventStoreError: EventStoreError{
				Op:  "ExecuteCommandWithLocks",
				Err: fmt.Errorf("failed to commit transaction: %w", err),
			},
			Resource: "database",
		}
	}

	return events, nil
}


