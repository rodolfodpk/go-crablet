package dcb

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// =============================================================================
// EventStore Interface
// =============================================================================

// EventStore is the core interface for appending and reading events
// This is the primary abstraction that users interact with
type EventStore interface {
	// Query reads events matching the query with optional cursor
	// after == nil: query from beginning of stream
	// after != nil: query from specified cursor position
	Query(ctx context.Context, query Query, after *Cursor) ([]Event, error)

	// QueryStream creates a channel-based stream of events matching a query with optional cursor
	// after == nil: stream from beginning of stream
	// after != nil: stream from specified cursor position
	// This is optimized for large datasets and provides backpressure through channels
	// for efficient memory usage and Go-idiomatic streaming
	QueryStream(ctx context.Context, query Query, after *Cursor) (<-chan Event, error)

	// Append appends events to the store without any consistency/concurrency checks
	// Use this only when there are no business rules or consistency requirements
	// For operations that require DCB concurrency control, use AppendIf instead
	Append(ctx context.Context, events []InputEvent) error

	// AppendIf appends events to the store with explicit DCB concurrency control
	// This method makes it clear when consistency/concurrency checks are required
	// Use this for operations that need to ensure data hasn't changed since projection
	// Note: DCB uses its own concurrency control mechanism via AppendCondition
	AppendIf(ctx context.Context, events []InputEvent, condition AppendCondition) error

	// Project projects state from events matching projectors with optional cursor
	// after == nil: project from beginning of stream
	// after != nil: project from specified cursor position
	// Returns final aggregated states and append condition for DCB concurrency control
	Project(ctx context.Context, projectors []StateProjector, after *Cursor) (map[string]any, AppendCondition, error)

	// ProjectStream creates a channel-based stream of projected states with optional cursor
	// after == nil: stream from beginning of stream
	// after != nil: stream from specified cursor position
	// Returns intermediate states and append conditions via channels for streaming projections
	ProjectStream(ctx context.Context, projectors []StateProjector, after *Cursor) (<-chan map[string]any, <-chan AppendCondition, error)

	// GetConfig returns the current EventStore configuration
	GetConfig() EventStoreConfig

	// GetPool exposes the underlying PostgreSQL connection pool (pgxpool.Pool).
	// This is intended for advanced/internal use cases such as custom transaction management,
	// integration testing, or infrastructure extensions. Regular application logic should NOT
	// use this method, as it bypasses the event store's consistency and abstraction guarantees.
	GetPool() *pgxpool.Pool
}

// =============================================================================
// EventStore Implementation
// =============================================================================

// eventStore implements the EventStore interface using PostgreSQL
type eventStore struct {
	pool   *pgxpool.Pool
	config EventStoreConfig

	// projectionSemaphore limits concurrent projection operations
	projectionSemaphore chan struct{}
}

func (es *eventStore) isEventStore() {}

// GetConfig returns the current EventStore configuration
func (es *eventStore) GetConfig() EventStoreConfig {
	return es.config
}

// GetPool returns the underlying database pool
func (es *eventStore) GetPool() *pgxpool.Pool {
	return es.pool
}

// executeReadInTx executes a read operation within a transaction using the configured read isolation level
// This is an internal helper method that wraps read operations in transactions for consistency
func (es *eventStore) executeReadInTx(ctx context.Context, operation func(tx pgx.Tx) error) error {
	tx, err := es.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: toPgxIsoLevel(es.config.DefaultReadIsolation),
	})
	if err != nil {
		return &ResourceError{
			EventStoreError: EventStoreError{
				Op:  "read_transaction",
				Err: fmt.Errorf("failed to begin read transaction: %w", err),
			},
			Resource: "database",
		}
	}
	defer tx.Rollback(ctx)

	return operation(tx)
}
