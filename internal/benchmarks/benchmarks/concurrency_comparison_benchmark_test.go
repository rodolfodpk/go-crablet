package benchmarks

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rodolfodpk/go-crablet/pkg/dcb"
)

// ConcurrencyComparisonBenchmark compares DCB concurrency control vs advisory locks approach for go-crablet.
// This benchmark tests both approaches under high concurrency scenarios

type ConcurrencyComparisonBenchmark struct {
	store           dcb.EventStore
	commandExecutor dcb.CommandExecutor
	pool            *pgxpool.Pool
	ctx             context.Context
}

func NewConcurrencyComparisonBenchmark(ctx context.Context, pool *pgxpool.Pool) *ConcurrencyComparisonBenchmark {
	store, err := dcb.NewEventStore(ctx, pool)
	if err != nil {
		panic(fmt.Sprintf("failed to create event store: %v", err))
	}

	commandExecutor := dcb.NewCommandExecutor(store)

	return &ConcurrencyComparisonBenchmark{
		store:           store,
		commandExecutor: commandExecutor,
		pool:            pool,
		ctx:             ctx,
	}
}

func (b *ConcurrencyComparisonBenchmark) Close() {
	// EventStore doesn't have a Close method - connection pool is managed externally
}

// Test data structures
type AccountState struct {
	AccountID string  `json:"account_id"`
	Balance   float64 `json:"balance"`
}

type TransferCommand struct {
	FromAccount string  `json:"from_account"`
	ToAccount   string  `json:"to_account"`
	Amount      float64 `json:"amount"`
	TransferID  string  `json:"transfer_id"`
}

// BenchmarkDCBConcurrencyControl tests DCB concurrency control approach
func BenchmarkDCBConcurrencyControl(b *testing.B) {
	ctx := context.Background()
	pool := getTestPool(ctx)
	defer pool.Close()

	benchmark := NewConcurrencyComparisonBenchmark(ctx, pool)
	defer benchmark.Close()

	// Setup test accounts
	benchmark.setupTestAccounts(ctx)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			benchmark.runDCBTransfer(ctx)
		}
	})
}

// BenchmarkAdvisoryLocks tests advisory locks approach
func BenchmarkAdvisoryLocks(b *testing.B) {
	ctx := context.Background()
	pool := getTestPool(ctx)
	defer pool.Close()

	benchmark := NewConcurrencyComparisonBenchmark(ctx, pool)
	defer benchmark.Close()

	// Setup test accounts
	benchmark.setupTestAccounts(ctx)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			benchmark.runAdvisoryLockTransfer(ctx)
		}
	})
}

// BenchmarkMixedApproach tests mixed approach (DCB + Advisory Locks)
func BenchmarkMixedApproach(b *testing.B) {
	ctx := context.Background()
	pool := getTestPool(ctx)
	defer pool.Close()

	benchmark := NewConcurrencyComparisonBenchmark(ctx, pool)
	defer benchmark.Close()

	// Setup test accounts
	benchmark.setupTestAccounts(ctx)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			benchmark.runMixedTransfer(ctx)
		}
	})
}

// BenchmarkConcurrencyStressTest tests high concurrency scenarios
func BenchmarkConcurrencyStressTest(b *testing.B) {
	ctx := context.Background()
	pool := getTestPool(ctx)
	defer pool.Close()

	benchmark := NewConcurrencyComparisonBenchmark(ctx, pool)
	defer benchmark.Close()

	// Setup test accounts
	benchmark.setupTestAccounts(ctx)

	// Test different concurrency levels
	concurrencyLevels := []int{1, 5, 10, 20, 50}

	for _, concurrency := range concurrencyLevels {
		b.Run(fmt.Sprintf("DCB_%d_goroutines", concurrency), func(b *testing.B) {
			benchmark.benchmarkConcurrencyLevel(ctx, b, concurrency, "dcb")
		})

		b.Run(fmt.Sprintf("AdvisoryLocks_%d_goroutines", concurrency), func(b *testing.B) {
			benchmark.benchmarkConcurrencyLevel(ctx, b, concurrency, "advisory")
		})

		b.Run(fmt.Sprintf("Mixed_%d_goroutines", concurrency), func(b *testing.B) {
			benchmark.benchmarkConcurrencyLevel(ctx, b, concurrency, "mixed")
		})
	}
}

// Helper methods

func (b *ConcurrencyComparisonBenchmark) setupTestAccounts(ctx context.Context) {
	// Create test accounts with initial balances
	accounts := []string{"acc-001", "acc-002", "acc-003", "acc-004", "acc-005"}

	for _, accountID := range accounts {
		// Create account event
		accountEvent := dcb.NewEvent("AccountCreated").
			WithTag("account_id", accountID).
			WithData(map[string]interface{}{
				"account_id": accountID,
				"balance":    1000.0,
				"created_at": time.Now(),
			}).
			Build()

		err := b.store.Append(ctx, []dcb.InputEvent{accountEvent})
		if err != nil {
			panic(fmt.Sprintf("failed to create account %s: %v", accountID, err))
		}
	}
}

func (b *ConcurrencyComparisonBenchmark) runDCBTransfer(ctx context.Context) error {
	// Use DCB concurrency control only
	transferID := fmt.Sprintf("txn-%d", time.Now().UnixNano())

	// Project current account states
	projectors := []dcb.StateProjector{
		{
			ID: "fromAccount",
			Query: dcb.NewQuery(
				dcb.NewTags("account_id", "acc-001"),
			),
			InitialState: AccountState{},
			TransitionFn: func(state any, event dcb.Event) any {
				account := state.(AccountState)
				if event.Type == "AccountCreated" {
					var data map[string]interface{}
					json.Unmarshal(event.Data, &data)
					account.AccountID = data["account_id"].(string)
					account.Balance = data["balance"].(float64)
				} else if event.Type == "AccountDebited" {
					var data map[string]interface{}
					json.Unmarshal(event.Data, &data)
					account.Balance -= data["amount"].(float64)
				} else if event.Type == "AccountCredited" {
					var data map[string]interface{}
					json.Unmarshal(event.Data, &data)
					account.Balance += data["amount"].(float64)
				}
				return account
			},
		},
		{
			ID: "toAccount",
			Query: dcb.NewQuery(
				dcb.NewTags("account_id", "acc-002"),
			),
			InitialState: AccountState{},
			TransitionFn: func(state any, event dcb.Event) any {
				account := state.(AccountState)
				if event.Type == "AccountCreated" {
					var data map[string]interface{}
					json.Unmarshal(event.Data, &data)
					account.AccountID = data["account_id"].(string)
					account.Balance = data["balance"].(float64)
				} else if event.Type == "AccountDebited" {
					var data map[string]interface{}
					json.Unmarshal(event.Data, &data)
					account.Balance -= data["amount"].(float64)
				} else if event.Type == "AccountCredited" {
					var data map[string]interface{}
					json.Unmarshal(event.Data, &data)
					account.Balance += data["amount"].(float64)
				}
				return account
			},
		},
	}

	states, appendCondition, err := b.store.Project(ctx, projectors, nil)
	if err != nil {
		return fmt.Errorf("failed to project account states: %w", err)
	}

	fromAccount := states["fromAccount"].(AccountState)
	toAccount := states["toAccount"].(AccountState)

	// Check if transfer is possible
	if fromAccount.Balance < 100 {
		return fmt.Errorf("insufficient funds in account %s", fromAccount.AccountID)
	}

	// Create transfer events
	events := []dcb.InputEvent{
		dcb.NewEvent("AccountDebited").
			WithTag("account_id", fromAccount.AccountID).
			WithTag("transfer_id", transferID).
			WithData(map[string]interface{}{
				"amount":        100.0,
				"balance_after": fromAccount.Balance - 100.0,
				"transfer_id":   transferID,
			}).
			Build(),
		dcb.NewEvent("AccountCredited").
			WithTag("account_id", toAccount.AccountID).
			WithTag("transfer_id", transferID).
			WithData(map[string]interface{}{
				"amount":        100.0,
				"balance_after": toAccount.Balance + 100.0,
				"transfer_id":   transferID,
			}).
			Build(),
	}

	// Append with DCB concurrency control
	return b.store.AppendIf(ctx, events, appendCondition)
}

func (b *ConcurrencyComparisonBenchmark) runAdvisoryLockTransfer(ctx context.Context) error {
	// Use advisory locks only (no DCB conditions)
	transferID := fmt.Sprintf("txn-%d", time.Now().UnixNano())

	// Create transfer events with advisory lock tags
	events := []dcb.InputEvent{
		dcb.NewEvent("AccountDebited").
			WithTag("account_id", "acc-001").
			WithTag("transfer_id", transferID).
			WithTag("lock:account", "acc-001"). // Advisory lock
			WithData(map[string]interface{}{
				"amount":      100.0,
				"transfer_id": transferID,
			}).
			Build(),
		dcb.NewEvent("AccountCredited").
			WithTag("account_id", "acc-002").
			WithTag("transfer_id", transferID).
			WithTag("lock:account", "acc-002"). // Advisory lock
			WithData(map[string]interface{}{
				"amount":      100.0,
				"transfer_id": transferID,
			}).
			Build(),
	}

	// Append with advisory locks (automatic when lock: tags present)
	return b.store.Append(ctx, events)
}

func (b *ConcurrencyComparisonBenchmark) runMixedTransfer(ctx context.Context) error {
	// Use both DCB concurrency control AND advisory locks
	transferID := fmt.Sprintf("txn-%d", time.Now().UnixNano())

	// Project current account states (DCB approach)
	projectors := []dcb.StateProjector{
		{
			ID: "fromAccount",
			Query: dcb.NewQuery(
				dcb.NewTags("account_id", "acc-001"),
			),
			InitialState: AccountState{},
			TransitionFn: func(state any, event dcb.Event) any {
				account := state.(AccountState)
				if event.Type == "AccountCreated" {
					var data map[string]interface{}
					json.Unmarshal(event.Data, &data)
					account.AccountID = data["account_id"].(string)
					account.Balance = data["balance"].(float64)
				} else if event.Type == "AccountDebited" {
					var data map[string]interface{}
					json.Unmarshal(event.Data, &data)
					account.Balance -= data["amount"].(float64)
				} else if event.Type == "AccountCredited" {
					var data map[string]interface{}
					json.Unmarshal(event.Data, &data)
					account.Balance += data["amount"].(float64)
				}
				return account
			},
		},
	}

	states, appendCondition, err := b.store.Project(ctx, projectors, nil)
	if err != nil {
		return fmt.Errorf("failed to project account states: %w", err)
	}

	fromAccount := states["fromAccount"].(AccountState)

	// Check if transfer is possible (DCB business rule)
	if fromAccount.Balance < 100 {
		return fmt.Errorf("insufficient funds in account %s", fromAccount.AccountID)
	}

	// Create transfer events with both DCB conditions AND advisory locks
	events := []dcb.InputEvent{
		dcb.NewEvent("AccountDebited").
			WithTag("account_id", fromAccount.AccountID).
			WithTag("transfer_id", transferID).
			WithTag("lock:account", fromAccount.AccountID). // Advisory lock
			WithData(map[string]interface{}{
				"amount":        100.0,
				"balance_after": fromAccount.Balance - 100.0,
				"transfer_id":   transferID,
			}).
			Build(),
		dcb.NewEvent("AccountCredited").
			WithTag("account_id", "acc-002").
			WithTag("transfer_id", transferID).
			WithTag("lock:account", "acc-002"). // Advisory lock
			WithData(map[string]interface{}{
				"amount":      100.0,
				"transfer_id": transferID,
			}).
			Build(),
	}

	// Append with both DCB conditions AND advisory locks
	return b.store.AppendIf(ctx, events, appendCondition)
}

func (benchmark *ConcurrencyComparisonBenchmark) benchmarkConcurrencyLevel(ctx context.Context, b *testing.B, concurrency int, approach string) {
	var wg sync.WaitGroup
	results := make(chan error, concurrency)
	start := make(chan struct{})

	// Start goroutines
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start

			var err error
			switch approach {
			case "dcb":
				err = benchmark.runDCBTransfer(ctx)
			case "advisory":
				err = benchmark.runAdvisoryLockTransfer(ctx)
			case "mixed":
				err = benchmark.runMixedTransfer(ctx)
			}

			results <- err
		}()
	}

	// Start all goroutines simultaneously
	close(start)

	// Wait for completion
	wg.Wait()
	close(results)

	// Count results
	successCount := 0
	errorCount := 0
	for err := range results {
		if err != nil {
			errorCount++
		} else {
			successCount++
		}
	}

	// Report metrics
	b.ReportMetric(float64(successCount), "successes")
	b.ReportMetric(float64(errorCount), "errors")
	b.ReportMetric(float64(successCount)/float64(concurrency)*100, "success_rate_percent")
}

// Helper function to get test database pool
func getTestPool(ctx context.Context) *pgxpool.Pool {
	pool, err := pgxpool.New(ctx, "postgres://crablet:crablet@localhost:5432/crablet")
	if err != nil {
		panic(fmt.Sprintf("failed to create connection pool: %v", err))
	}
	return pool
}

// Benchmark comparison summary
func BenchmarkConcurrencyComparisonSummary(b *testing.B) {
	ctx := context.Background()
	pool := getTestPool(ctx)
	defer pool.Close()

	benchmark := NewConcurrencyComparisonBenchmark(ctx, pool)
	defer benchmark.Close()

	// Setup test accounts
	benchmark.setupTestAccounts(ctx)

	// Run all approaches and compare
	b.Run("DCB_Only", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				benchmark.runDCBTransfer(ctx)
			}
		})
	})

	b.Run("Advisory_Locks_Only", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				benchmark.runAdvisoryLockTransfer(ctx)
			}
		})
	})

	b.Run("Mixed_Approach", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				benchmark.runMixedTransfer(ctx)
			}
		})
	})
}
