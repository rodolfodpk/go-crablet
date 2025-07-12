package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"go-crablet/pkg/dcb"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

var _ = Describe("DecisionModelExample", func() {
	var (
		ctx       context.Context
		cancel    context.CancelFunc
		pool      *pgxpool.Pool
		container testcontainers.Container
		store     dcb.EventStore
		testID    string
	)

	BeforeSuite(func() {
		ctx, cancel = context.WithTimeout(context.Background(), 2*time.Minute)
		var err error
		pool, container, err = setupTestDatabase(ctx)
		Expect(err).NotTo(HaveOccurred())
		store, err = dcb.NewEventStore(ctx, pool)
		Expect(err).NotTo(HaveOccurred())
		testID = fmt.Sprintf("test_%d", time.Now().UnixNano())
	})

	AfterSuite(func() {
		if pool != nil {
			pool.Close()
		}
		if container != nil {
			container.Terminate(ctx)
		}
		if cancel != nil {
			cancel()
		}
	})

	Describe("Account and Transaction Commands", func() {
		It("should open an account", func() {
			openAccountCmd := OpenAccountCommand{
				AccountID:      fmt.Sprintf("test_acc_decision_%s", testID),
				InitialBalance: 1000,
			}
			err := handleOpenAccount(ctx, store, openAccountCmd)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should process a transaction", func() {
			processTransactionCmd := ProcessTransactionCommand{
				AccountID: fmt.Sprintf("test_acc_decision_%s", testID),
				Amount:    500,
			}
			err := handleProcessTransaction(ctx, store, processTransactionCmd)
			Expect(err).NotTo(HaveOccurred())
		})

		Describe("Business Rules", func() {
			It("should prevent opening duplicate account", func() {
				duplicateCmd := OpenAccountCommand{
					AccountID:      fmt.Sprintf("test_acc_decision_%s", testID),
					InitialBalance: 2000,
				}
				err := handleOpenAccount(ctx, store, duplicateCmd)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("already exists"))
			})

			It("should prevent processing transaction for non-existent account", func() {
				nonExistentAccountCmd := ProcessTransactionCommand{
					AccountID: fmt.Sprintf("non_existent_account_%s", testID),
					Amount:    100,
				}
				err := handleProcessTransaction(ctx, store, nonExistentAccountCmd)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("does not exist"))
			})
		})

		It("should project decision model state and support optimistic locking", func() {
			accountID := fmt.Sprintf("test_acc_decision_%s", testID)

			accountProjector := dcb.StateProjector{
				Query: dcb.NewQuery(
					dcb.NewTags("account_id", accountID),
					"AccountOpened", "AccountBalanceChanged",
				),
				InitialState: &AccountState{ID: accountID, Balance: 0},
				TransitionFn: func(state any, event dcb.Event) any {
					account := state.(*AccountState)
					switch event.Type {
					case "AccountOpened":
						var data AccountOpenedData
						json.Unmarshal(event.Data, &data)
						account.Balance = data.InitialBalance
					case "AccountBalanceChanged":
						var data AccountBalanceChangedData
						json.Unmarshal(event.Data, &data)
						account.Balance = data.NewBalance
					}
					return account
				},
			}

			transactionProjector := dcb.StateProjector{
				Query: dcb.NewQuery(
					dcb.NewTags("account_id", accountID),
					"TransactionProcessed",
				),
				InitialState: &TransactionState{Count: 0, TotalAmount: 0},
				TransitionFn: func(state any, event dcb.Event) any {
					transactions := state.(*TransactionState)
					if event.Type == "TransactionProcessed" {
						var data TransactionProcessedData
						json.Unmarshal(event.Data, &data)
						transactions.Count++
						transactions.TotalAmount += data.Amount
					}
					return transactions
				},
			}

			projectors := []dcb.StateProjector{
				{
					ID:           "account",
					Query:        accountProjector.Query,
					InitialState: accountProjector.InitialState,
					TransitionFn: accountProjector.TransitionFn,
				},
				{
					ID:           "transactions",
					Query:        transactionProjector.Query,
					InitialState: transactionProjector.InitialState,
					TransitionFn: transactionProjector.TransitionFn,
				},
			}

			states, _, err := store.Project(ctx, projectors, nil)
			Expect(err).NotTo(HaveOccurred())

			// Verify account state
			account, ok := states["account"].(*AccountState)
			Expect(ok).To(BeTrue())
			Expect(account.ID).To(Equal(accountID))
			Expect(account.Balance).To(Equal(1000))

			// Verify transaction state
			transactions, ok := states["transactions"].(*TransactionState)
			Expect(ok).To(BeTrue())
			Expect(transactions.Count).To(Equal(1))
			Expect(transactions.TotalAmount).To(Equal(500))

			// Test optimistic locking
			By("supporting optimistic locking", func() {
				_, appendCondition, err := store.Project(ctx, []dcb.StateProjector{{
					ID:           "account",
					Query:        accountProjector.Query,
					InitialState: accountProjector.InitialState,
					TransitionFn: accountProjector.TransitionFn,
				}}, nil)
				Expect(err).NotTo(HaveOccurred())

				optimisticCmd := ProcessTransactionCommand{
					AccountID: accountID,
					Amount:    200,
				}
				err = handleProcessTransactionWithCondition(ctx, store, optimisticCmd, appendCondition)
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})
})

// setupTestDatabase creates a test database using testcontainers
func setupTestDatabase(ctx context.Context) (*pgxpool.Pool, testcontainers.Container, error) {
	password, err := generateRandomPassword(16)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate password: %w", err)
	}

	req := testcontainers.ContainerRequest{
		Image:        "postgres:17.5-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_PASSWORD": password,
		},
		WaitingFor: wait.ForListeningPort("5432/tcp"),
	}

	postgresC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, nil, err
	}

	host, err := postgresC.Host(ctx)
	if err != nil {
		return nil, nil, err
	}

	port, err := postgresC.MappedPort(ctx, "5432")
	if err != nil {
		return nil, nil, err
	}

	dsn := fmt.Sprintf("postgres://postgres:%s@%s:%s/postgres?sslmode=disable", password, host, port.Port())
	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, nil, err
	}

	poolConfig.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeCacheDescribe
	poolConfig.ConnConfig.StatementCacheCapacity = 100

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, nil, err
	}

	schemaSQL, err := os.ReadFile("../../../docker-entrypoint-initdb.d/schema.sql")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read schema: %w", err)
	}

	filteredSQL := filterPsqlCommands(string(schemaSQL))
	_, err = pool.Exec(ctx, filteredSQL)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to execute schema: %w", err)
	}

	return pool, postgresC, nil
}

func generateRandomPassword(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes)[:length], nil
}

func filterPsqlCommands(sql string) string {
	lines := strings.Split(sql, "\n")
	var filteredLines []string

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if strings.HasPrefix(trimmedLine, "\\") {
			continue
		}
		if strings.Contains(trimmedLine, "\\gexec") {
			continue
		}
		if strings.Contains(trimmedLine, "SELECT 'CREATE DATABASE") {
			continue
		}
		if trimmedLine == "" {
			continue
		}
		filteredLines = append(filteredLines, trimmedLine)
	}
	return strings.Join(filteredLines, "\n")
}
