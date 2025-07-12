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
			cmdData, err := json.Marshal(createAccount1Cmd)
			Expect(err).NotTo(HaveOccurred())

			command := dcb.NewCommand(CommandTypeCreateAccount, cmdData, nil)
			handler := dcb.CommandHandlerFunc(handleCommand)
			commandExecutor := dcb.NewCommandExecutor(store)

			err = commandExecutor.ExecuteCommand(ctx, command, handler, nil)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should create account 2", func() {
			createAccount2Cmd := CreateAccountCommand{
				AccountID:      "test_acc2",
				InitialBalance: 500,
			}
			cmdData, err := json.Marshal(createAccount2Cmd)
			Expect(err).NotTo(HaveOccurred())

			command := dcb.NewCommand(CommandTypeCreateAccount, cmdData, nil)
			handler := dcb.CommandHandlerFunc(handleCommand)
			commandExecutor := dcb.NewCommandExecutor(store)

			err = commandExecutor.ExecuteCommand(ctx, command, handler, nil)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should transfer money", func() {
			// Create accounts first
			createAccount1Cmd := CreateAccountCommand{
				AccountID:      "test_acc1",
				InitialBalance: 1000,
			}
			cmdData1, err := json.Marshal(createAccount1Cmd)
			Expect(err).NotTo(HaveOccurred())

			command1 := dcb.NewCommand(CommandTypeCreateAccount, cmdData1, nil)
			handler := dcb.CommandHandlerFunc(handleCommand)
			commandExecutor := dcb.NewCommandExecutor(store)

			err = commandExecutor.ExecuteCommand(ctx, command1, handler, nil)
			Expect(err).NotTo(HaveOccurred())

			createAccount2Cmd := CreateAccountCommand{
				AccountID:      "test_acc2",
				InitialBalance: 500,
			}
			cmdData2, err := json.Marshal(createAccount2Cmd)
			Expect(err).NotTo(HaveOccurred())

			command2 := dcb.NewCommand(CommandTypeCreateAccount, cmdData2, nil)
			err = commandExecutor.ExecuteCommand(ctx, command2, handler, nil)
			Expect(err).NotTo(HaveOccurred())

			transferCmd := TransferMoneyCommand{
				TransferID:    "test_transfer_1",
				FromAccountID: "test_acc1",
				ToAccountID:   "test_acc2",
				Amount:        300,
			}
			transferData, err := json.Marshal(transferCmd)
			Expect(err).NotTo(HaveOccurred())

			transferCommand := dcb.NewCommand(CommandTypeTransferMoney, transferData, nil)
			err = commandExecutor.ExecuteCommand(ctx, transferCommand, handler, nil)
			Expect(err).NotTo(HaveOccurred())
		})

		Describe("Business Rules", func() {
			It("should prevent creating duplicate account", func() {
				createAccountCmd := CreateAccountCommand{
					AccountID:      "test_acc1",
					InitialBalance: 1000,
				}
				cmdData, err := json.Marshal(createAccountCmd)
				Expect(err).NotTo(HaveOccurred())

				command := dcb.NewCommand(CommandTypeCreateAccount, cmdData, nil)
				handler := dcb.CommandHandlerFunc(handleCommand)
				commandExecutor := dcb.NewCommandExecutor(store)

				err = commandExecutor.ExecuteCommand(ctx, command, handler, nil)
				Expect(err).NotTo(HaveOccurred())

				duplicateCmd := CreateAccountCommand{
					AccountID:      "test_acc1",
					InitialBalance: 2000,
				}
				duplicateData, err := json.Marshal(duplicateCmd)
				Expect(err).NotTo(HaveOccurred())

				duplicateCommand := dcb.NewCommand(CommandTypeCreateAccount, duplicateData, nil)
				err = commandExecutor.ExecuteCommand(ctx, duplicateCommand, handler, nil)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("already exists"))
			})

			It("should prevent transferring more than available balance", func() {
				createAccount1Cmd := CreateAccountCommand{
					AccountID:      "test_acc1",
					InitialBalance: 1000,
				}
				cmdData1, err := json.Marshal(createAccount1Cmd)
				Expect(err).NotTo(HaveOccurred())

				command1 := dcb.NewCommand(CommandTypeCreateAccount, cmdData1, nil)
				handler := dcb.CommandHandlerFunc(handleCommand)
				commandExecutor := dcb.NewCommandExecutor(store)

				err = commandExecutor.ExecuteCommand(ctx, command1, handler, nil)
				Expect(err).NotTo(HaveOccurred())

				createAccount2Cmd := CreateAccountCommand{
					AccountID:      "test_acc2",
					InitialBalance: 500,
				}
				cmdData2, err := json.Marshal(createAccount2Cmd)
				Expect(err).NotTo(HaveOccurred())

				command2 := dcb.NewCommand(CommandTypeCreateAccount, cmdData2, nil)
				err = commandExecutor.ExecuteCommand(ctx, command2, handler, nil)
				Expect(err).NotTo(HaveOccurred())

				insufficientFundsCmd := TransferMoneyCommand{
					TransferID:    "test_transfer_2",
					FromAccountID: "test_acc1",
					ToAccountID:   "test_acc2",
					Amount:        1001,
				}
				insufficientData, err := json.Marshal(insufficientFundsCmd)
				Expect(err).NotTo(HaveOccurred())

				insufficientCommand := dcb.NewCommand(CommandTypeTransferMoney, insufficientData, nil)
				err = commandExecutor.ExecuteCommand(ctx, insufficientCommand, handler, nil)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("insufficient funds"))
			})

			It("should prevent transferring from non-existent account", func() {
				createAccount2Cmd := CreateAccountCommand{
					AccountID:      "test_acc2",
					InitialBalance: 500,
				}
				cmdData, err := json.Marshal(createAccount2Cmd)
				Expect(err).NotTo(HaveOccurred())

				command := dcb.NewCommand(CommandTypeCreateAccount, cmdData, nil)
				handler := dcb.CommandHandlerFunc(handleCommand)
				commandExecutor := dcb.NewCommandExecutor(store)

				err = commandExecutor.ExecuteCommand(ctx, command, handler, nil)
				Expect(err).NotTo(HaveOccurred())

				nonExistentFromCmd := TransferMoneyCommand{
					TransferID:    "test_transfer_3",
					FromAccountID: "non_existent_account",
					ToAccountID:   "test_acc2",
					Amount:        100,
				}
				nonExistentData, err := json.Marshal(nonExistentFromCmd)
				Expect(err).NotTo(HaveOccurred())

				nonExistentCommand := dcb.NewCommand(CommandTypeTransferMoney, nonExistentData, nil)
				err = commandExecutor.ExecuteCommand(ctx, nonExistentCommand, handler, nil)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("insufficient funds"))
			})

			It("should allow transferring to non-existent account (creates it)", func() {
				createAccount1Cmd := CreateAccountCommand{
					AccountID:      "test_acc1",
					InitialBalance: 1000,
				}
				cmdData, err := json.Marshal(createAccount1Cmd)
				Expect(err).NotTo(HaveOccurred())

				command := dcb.NewCommand(CommandTypeCreateAccount, cmdData, nil)
				handler := dcb.CommandHandlerFunc(handleCommand)
				commandExecutor := dcb.NewCommandExecutor(store)

				err = commandExecutor.ExecuteCommand(ctx, command, handler, nil)
				Expect(err).NotTo(HaveOccurred())

				nonExistentToCmd := TransferMoneyCommand{
					TransferID:    "test_transfer_4",
					FromAccountID: "test_acc1",
					ToAccountID:   "non_existent_account",
					Amount:        100,
				}
				nonExistentData, err := json.Marshal(nonExistentToCmd)
				Expect(err).NotTo(HaveOccurred())

				nonExistentCommand := dcb.NewCommand(CommandTypeTransferMoney, nonExistentData, nil)
				err = commandExecutor.ExecuteCommand(ctx, nonExistentCommand, handler, nil)
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	Describe("Sequential Transfers", func() {
		BeforeEach(func() {
			cleanupEvents()
		})

		It("should handle sequential transfers and detect insufficient funds", func() {
			handler := dcb.CommandHandlerFunc(handleCommand)
			commandExecutor := dcb.NewCommandExecutor(store)

			createAccount3Cmd := CreateAccountCommand{
				AccountID:      "test_acc3",
				InitialBalance: 2000,
			}
			cmdData3, err := json.Marshal(createAccount3Cmd)
			Expect(err).NotTo(HaveOccurred())

			command3 := dcb.NewCommand(CommandTypeCreateAccount, cmdData3, nil)
			err = commandExecutor.ExecuteCommand(ctx, command3, handler, nil)
			Expect(err).NotTo(HaveOccurred())

			createAccount4Cmd := CreateAccountCommand{
				AccountID:      "test_acc4",
				InitialBalance: 0,
			}
			cmdData4, err := json.Marshal(createAccount4Cmd)
			Expect(err).NotTo(HaveOccurred())

			command4 := dcb.NewCommand(CommandTypeCreateAccount, cmdData4, nil)
			err = commandExecutor.ExecuteCommand(ctx, command4, handler, nil)
			Expect(err).NotTo(HaveOccurred())

			createAccount5Cmd := CreateAccountCommand{
				AccountID:      "test_acc5",
				InitialBalance: 0,
			}
			cmdData5, err := json.Marshal(createAccount5Cmd)
			Expect(err).NotTo(HaveOccurred())

			command5 := dcb.NewCommand(CommandTypeCreateAccount, cmdData5, nil)
			err = commandExecutor.ExecuteCommand(ctx, command5, handler, nil)
			Expect(err).NotTo(HaveOccurred())

			transfer1Cmd := TransferMoneyCommand{
				TransferID:    "test_transfer_5",
				FromAccountID: "test_acc3",
				ToAccountID:   "test_acc4",
				Amount:        1500,
			}
			transferData1, err := json.Marshal(transfer1Cmd)
			Expect(err).NotTo(HaveOccurred())

			transferCommand1 := dcb.NewCommand(CommandTypeTransferMoney, transferData1, nil)
			err = commandExecutor.ExecuteCommand(ctx, transferCommand1, handler, nil)
			Expect(err).NotTo(HaveOccurred())

			transfer2Cmd := TransferMoneyCommand{
				TransferID:    "test_transfer_6",
				FromAccountID: "test_acc3",
				ToAccountID:   "test_acc5",
				Amount:        1000,
			}
			transferData2, err := json.Marshal(transfer2Cmd)
			Expect(err).NotTo(HaveOccurred())

			transferCommand2 := dcb.NewCommand(CommandTypeTransferMoney, transferData2, nil)
			err = commandExecutor.ExecuteCommand(ctx, transferCommand2, handler, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("insufficient funds"))
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
