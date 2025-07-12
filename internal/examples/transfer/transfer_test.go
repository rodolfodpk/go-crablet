package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"time"

	"go-crablet/pkg/dcb"

	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

var (
	ctx       context.Context
	cancel    context.CancelFunc
	pool      *pgxpool.Pool
	container testcontainers.Container
	store     dcb.EventStore
)

var _ = BeforeSuite(func() {
	ctx, cancel = context.WithTimeout(context.Background(), 2*time.Minute)
	var err error
	pool, container, err = setupTestDatabase(ctx)
	Expect(err).NotTo(HaveOccurred())
	store, err = dcb.NewEventStore(ctx, pool)
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
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

func TestTransferExample(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Transfer Example Suite")
}

var _ = Describe("TransferExample", func() {
	cleanupEvents := func() {
		_, err := pool.Exec(ctx, "TRUNCATE TABLE events RESTART IDENTITY CASCADE")
		Expect(err).NotTo(HaveOccurred())
	}

	Describe("Account and Transfer Commands", func() {
		BeforeEach(func() {
			cleanupEvents()
		})

		It("should create account 1", func() {
			createAccount1Cmd := CreateAccountCommand{
				AccountID:      "test_acc1",
				InitialBalance: 1000,
			}
			err := handleCreateAccount(ctx, store, createAccount1Cmd)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should create account 2", func() {
			createAccount2Cmd := CreateAccountCommand{
				AccountID:      "test_acc2",
				InitialBalance: 500,
			}
			err := handleCreateAccount(ctx, store, createAccount2Cmd)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should transfer money", func() {
			// Create accounts first
			createAccount1Cmd := CreateAccountCommand{
				AccountID:      "test_acc1",
				InitialBalance: 1000,
			}
			err := handleCreateAccount(ctx, store, createAccount1Cmd)
			Expect(err).NotTo(HaveOccurred())

			createAccount2Cmd := CreateAccountCommand{
				AccountID:      "test_acc2",
				InitialBalance: 500,
			}
			err = handleCreateAccount(ctx, store, createAccount2Cmd)
			Expect(err).NotTo(HaveOccurred())

			transferCmd := TransferMoneyCommand{
				TransferID:    "test_transfer_1",
				FromAccountID: "test_acc1",
				ToAccountID:   "test_acc2",
				Amount:        300,
			}
			err = handleTransferMoney(ctx, store, transferCmd)
			Expect(err).NotTo(HaveOccurred())
		})

		Describe("Business Rules", func() {
			It("should prevent creating duplicate account", func() {
				createAccountCmd := CreateAccountCommand{
					AccountID:      "test_acc1",
					InitialBalance: 1000,
				}
				err := handleCreateAccount(ctx, store, createAccountCmd)
				Expect(err).NotTo(HaveOccurred())

				duplicateCmd := CreateAccountCommand{
					AccountID:      "test_acc1",
					InitialBalance: 2000,
				}
				err = handleCreateAccount(ctx, store, duplicateCmd)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("already exists"))
			})

			It("should prevent transferring more than available balance", func() {
				createAccount1Cmd := CreateAccountCommand{
					AccountID:      "test_acc1",
					InitialBalance: 1000,
				}
				err := handleCreateAccount(ctx, store, createAccount1Cmd)
				Expect(err).NotTo(HaveOccurred())

				createAccount2Cmd := CreateAccountCommand{
					AccountID:      "test_acc2",
					InitialBalance: 500,
				}
				err = handleCreateAccount(ctx, store, createAccount2Cmd)
				Expect(err).NotTo(HaveOccurred())

				insufficientFundsCmd := TransferMoneyCommand{
					TransferID:    "test_transfer_2",
					FromAccountID: "test_acc1",
					ToAccountID:   "test_acc2",
					Amount:        1001,
				}
				err = handleTransferMoney(ctx, store, insufficientFundsCmd)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("insufficient funds"))
			})

			It("should prevent transferring from non-existent account", func() {
				createAccount2Cmd := CreateAccountCommand{
					AccountID:      "test_acc2",
					InitialBalance: 500,
				}
				err := handleCreateAccount(ctx, store, createAccount2Cmd)
				Expect(err).NotTo(HaveOccurred())

				nonExistentFromCmd := TransferMoneyCommand{
					TransferID:    "test_transfer_3",
					FromAccountID: "non_existent_account",
					ToAccountID:   "test_acc2",
					Amount:        100,
				}
				err = handleTransferMoney(ctx, store, nonExistentFromCmd)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("insufficient funds"))
			})

			It("should allow transferring to non-existent account (creates it)", func() {
				createAccount1Cmd := CreateAccountCommand{
					AccountID:      "test_acc1",
					InitialBalance: 1000,
				}
				err := handleCreateAccount(ctx, store, createAccount1Cmd)
				Expect(err).NotTo(HaveOccurred())

				nonExistentToCmd := TransferMoneyCommand{
					TransferID:    "test_transfer_4",
					FromAccountID: "test_acc1",
					ToAccountID:   "non_existent_account",
					Amount:        100,
				}
				err = handleTransferMoney(ctx, store, nonExistentToCmd)
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	Describe("Sequential Transfers", func() {
		BeforeEach(func() {
			cleanupEvents()
		})

		It("should handle sequential transfers and detect insufficient funds", func() {
			createAccount3Cmd := CreateAccountCommand{
				AccountID:      "test_acc3",
				InitialBalance: 2000,
			}
			err := handleCreateAccount(ctx, store, createAccount3Cmd)
			Expect(err).NotTo(HaveOccurred())

			createAccount4Cmd := CreateAccountCommand{
				AccountID:      "test_acc4",
				InitialBalance: 0,
			}
			err = handleCreateAccount(ctx, store, createAccount4Cmd)
			Expect(err).NotTo(HaveOccurred())

			createAccount5Cmd := CreateAccountCommand{
				AccountID:      "test_acc5",
				InitialBalance: 0,
			}
			err = handleCreateAccount(ctx, store, createAccount5Cmd)
			Expect(err).NotTo(HaveOccurred())

			transfer1Cmd := TransferMoneyCommand{
				TransferID:    "test_transfer_5",
				FromAccountID: "test_acc3",
				ToAccountID:   "test_acc4",
				Amount:        1500,
			}
			err1 := handleTransferMoney(ctx, store, transfer1Cmd)
			Expect(err1).NotTo(HaveOccurred())

			transfer2Cmd := TransferMoneyCommand{
				TransferID:    "test_transfer_6",
				FromAccountID: "test_acc3",
				ToAccountID:   "test_acc5",
				Amount:        1000,
			}
			err2 := handleTransferMoney(ctx, store, transfer2Cmd)
			Expect(err2).To(HaveOccurred())
			Expect(err2.Error()).To(ContainSubstring("insufficient funds"))
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
