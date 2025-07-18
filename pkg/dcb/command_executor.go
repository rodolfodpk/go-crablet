package dcb

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

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

	// Start transaction with EventStore's isolation level
	executeCtx, cancel := ce.withTimeout(ctx, config.AppendTimeout)
	defer cancel()

	// Get pool and start transaction
	pool := ce.eventStore.GetPool()
	tx, err := pool.BeginTx(executeCtx, pgx.TxOptions{
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

// Helper methods
func (ce *commandExecutor) withTimeout(ctx context.Context, defaultTimeoutMs int) (context.Context, context.CancelFunc) {
	if deadline, ok := ctx.Deadline(); ok {
		return context.WithDeadline(context.Background(), deadline)
	}
	return context.WithTimeout(context.Background(), time.Duration(defaultTimeoutMs)*time.Millisecond)
}
