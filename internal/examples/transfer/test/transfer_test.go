package transferexample_test

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/rodolfodpk/go-crablet/pkg/dcb"

	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	transferexample "github.com/rodolfodpk/go-crablet/internal/examples/transfer/pkg"
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
			createAccount1Cmd := transferexample.CreateAccountCommand{
				AccountID:      "test_acc1",
				Owner:          "test_user1",
				InitialBalance: 1000,
			}
			cmdData, err := json.Marshal(createAccount1Cmd)
			Expect(err).NotTo(HaveOccurred())

			command := dcb.NewCommand(transferexample.CommandTypeCreateAccount, cmdData, nil)
			// For account creation
			handlerCreate := dcb.CommandHandlerFunc(func(ctx context.Context, store dcb.EventStore, command dcb.Command) ([]dcb.InputEvent, error) {
				events, _, err := transferexample.HandleCreateAccount(ctx, store, command)
				return events, err
			})
			commandExecutor := dcb.NewCommandExecutor(store)

			_, err = commandExecutor.ExecuteCommand(ctx, command, handlerCreate, nil)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should create account 2", func() {
			createAccount2Cmd := transferexample.CreateAccountCommand{
				AccountID:      "test_acc2",
				Owner:          "test_user2",
				InitialBalance: 500,
			}
			cmdData, err := json.Marshal(createAccount2Cmd)
			Expect(err).NotTo(HaveOccurred())

			command := dcb.NewCommand(transferexample.CommandTypeCreateAccount, cmdData, nil)
			// For account creation
			handlerCreate := dcb.CommandHandlerFunc(func(ctx context.Context, store dcb.EventStore, command dcb.Command) ([]dcb.InputEvent, error) {
				events, _, err := transferexample.HandleCreateAccount(ctx, store, command)
				return events, err
			})
			commandExecutor := dcb.NewCommandExecutor(store)

			_, err = commandExecutor.ExecuteCommand(ctx, command, handlerCreate, nil)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should transfer money", func() {
			// Create accounts first
			createAccount1Cmd := transferexample.CreateAccountCommand{
				AccountID:      "test_acc1",
				Owner:          "test_user1",
				InitialBalance: 1000,
			}
			cmdData1, err := json.Marshal(createAccount1Cmd)
			Expect(err).NotTo(HaveOccurred())

			command1 := dcb.NewCommand(transferexample.CommandTypeCreateAccount, cmdData1, nil)
			// For account creation
			handlerCreate := dcb.CommandHandlerFunc(func(ctx context.Context, store dcb.EventStore, command dcb.Command) ([]dcb.InputEvent, error) {
				events, _, err := transferexample.HandleCreateAccount(ctx, store, command)
				return events, err
			})
			commandExecutor := dcb.NewCommandExecutor(store)

			_, err = commandExecutor.ExecuteCommand(ctx, command1, handlerCreate, nil)
			Expect(err).NotTo(HaveOccurred())

			createAccount2Cmd := transferexample.CreateAccountCommand{
				AccountID:      "test_acc2",
				Owner:          "test_user2",
				InitialBalance: 500,
			}
			cmdData2, err := json.Marshal(createAccount2Cmd)
			Expect(err).NotTo(HaveOccurred())

			command2 := dcb.NewCommand(transferexample.CommandTypeCreateAccount, cmdData2, nil)
			_, err = commandExecutor.ExecuteCommand(ctx, command2, handlerCreate, nil)
			Expect(err).NotTo(HaveOccurred())

			transferCmd := transferexample.TransferMoneyCommand{
				TransferID:    "test_transfer_1",
				FromAccountID: "test_acc1",
				ToAccountID:   "test_acc2",
				Amount:        300,
			}
			transferData, err := json.Marshal(transferCmd)
			Expect(err).NotTo(HaveOccurred())

			transferCommand := dcb.NewCommand(transferexample.CommandTypeTransferMoney, transferData, nil)
			// For transfer
			handlerTransfer := dcb.CommandHandlerFunc(func(ctx context.Context, store dcb.EventStore, command dcb.Command) ([]dcb.InputEvent, error) {
				events, _, err := transferexample.HandleTransferMoney(ctx, store, command)
				return events, err
			})
			_, err = commandExecutor.ExecuteCommand(ctx, transferCommand, handlerTransfer, nil)
			Expect(err).NotTo(HaveOccurred())
		})

		Describe("Business Rules", func() {
			It("should prevent creating duplicate account", func() {
				createAccountCmd := transferexample.CreateAccountCommand{
					AccountID:      "test_acc1",
					Owner:          "test_user1",
					InitialBalance: 1000,
				}
				cmdData, err := json.Marshal(createAccountCmd)
				Expect(err).NotTo(HaveOccurred())

				command := dcb.NewCommand(transferexample.CommandTypeCreateAccount, cmdData, nil)
				// For account creation
				handlerCreate := dcb.CommandHandlerFunc(func(ctx context.Context, store dcb.EventStore, command dcb.Command) ([]dcb.InputEvent, error) {
					events, _, err := transferexample.HandleCreateAccount(ctx, store, command)
					return events, err
				})
				commandExecutor := dcb.NewCommandExecutor(store)

				_, err = commandExecutor.ExecuteCommand(ctx, command, handlerCreate, nil)
				Expect(err).NotTo(HaveOccurred())

				duplicateCmd := transferexample.CreateAccountCommand{
					AccountID:      "test_acc1",
					Owner:          "test_user1",
					InitialBalance: 2000,
				}
				duplicateData, err := json.Marshal(duplicateCmd)
				Expect(err).NotTo(HaveOccurred())

				duplicateCommand := dcb.NewCommand(transferexample.CommandTypeCreateAccount, duplicateData, nil)
				_, err = commandExecutor.ExecuteCommand(ctx, duplicateCommand, handlerCreate, nil)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("already exists"))
			})

			It("should prevent transferring more than available balance", func() {
				createAccount1Cmd := transferexample.CreateAccountCommand{
					AccountID:      "test_acc1",
					Owner:          "test_user1",
					InitialBalance: 1000,
				}
				cmdData1, err := json.Marshal(createAccount1Cmd)
				Expect(err).NotTo(HaveOccurred())

				command1 := dcb.NewCommand(transferexample.CommandTypeCreateAccount, cmdData1, nil)
				// For account creation
				handlerCreate := dcb.CommandHandlerFunc(func(ctx context.Context, store dcb.EventStore, command dcb.Command) ([]dcb.InputEvent, error) {
					events, _, err := transferexample.HandleCreateAccount(ctx, store, command)
					return events, err
				})
				commandExecutor := dcb.NewCommandExecutor(store)

				_, err = commandExecutor.ExecuteCommand(ctx, command1, handlerCreate, nil)
				Expect(err).NotTo(HaveOccurred())

				createAccount2Cmd := transferexample.CreateAccountCommand{
					AccountID:      "test_acc2",
					Owner:          "test_user2",
					InitialBalance: 500,
				}
				cmdData2, err := json.Marshal(createAccount2Cmd)
				Expect(err).NotTo(HaveOccurred())

				command2 := dcb.NewCommand(transferexample.CommandTypeCreateAccount, cmdData2, nil)
				_, err = commandExecutor.ExecuteCommand(ctx, command2, handlerCreate, nil)
				Expect(err).NotTo(HaveOccurred())

				insufficientFundsCmd := transferexample.TransferMoneyCommand{
					TransferID:    "test_transfer_2",
					FromAccountID: "test_acc1",
					ToAccountID:   "test_acc2",
					Amount:        1001,
				}
				insufficientData, err := json.Marshal(insufficientFundsCmd)
				Expect(err).NotTo(HaveOccurred())

				insufficientCommand := dcb.NewCommand(transferexample.CommandTypeTransferMoney, insufficientData, nil)
				// For transfer commands, get the AppendCondition from the handler
				// Note: HandleCommand will return an error for insufficient funds, which is expected
				_, _, err = transferexample.HandleCommand(ctx, store, insufficientCommand)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("insufficient funds"))
			})

			It("should prevent transferring from non-existent account", func() {
				createAccount2Cmd := transferexample.CreateAccountCommand{
					AccountID:      "test_acc2",
					Owner:          "test_user2",
					InitialBalance: 500,
				}
				cmdData, err := json.Marshal(createAccount2Cmd)
				Expect(err).NotTo(HaveOccurred())

				command := dcb.NewCommand(transferexample.CommandTypeCreateAccount, cmdData, nil)
				// For account creation
				handlerCreate := dcb.CommandHandlerFunc(func(ctx context.Context, store dcb.EventStore, command dcb.Command) ([]dcb.InputEvent, error) {
					events, _, err := transferexample.HandleCreateAccount(ctx, store, command)
					return events, err
				})
				commandExecutor := dcb.NewCommandExecutor(store)

				_, err = commandExecutor.ExecuteCommand(ctx, command, handlerCreate, nil)
				Expect(err).NotTo(HaveOccurred())

				nonExistentFromCmd := transferexample.TransferMoneyCommand{
					TransferID:    "test_transfer_3",
					FromAccountID: "non_existent_account",
					ToAccountID:   "test_acc2",
					Amount:        100,
				}
				nonExistentData, err := json.Marshal(nonExistentFromCmd)
				Expect(err).NotTo(HaveOccurred())

				nonExistentCommand := dcb.NewCommand(transferexample.CommandTypeTransferMoney, nonExistentData, nil)
				// For transfer commands, get the AppendCondition from the handler
				// Note: HandleCommand will return an error for insufficient funds, which is expected
				_, _, err = transferexample.HandleCommand(ctx, store, nonExistentCommand)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("source account"))
			})

			It("should allow transferring to non-existent account (creates it)", func() {
				createAccount1Cmd := transferexample.CreateAccountCommand{
					AccountID:      "test_acc1",
					Owner:          "test_user1",
					InitialBalance: 1000,
				}
				cmdData, err := json.Marshal(createAccount1Cmd)
				Expect(err).NotTo(HaveOccurred())

				command := dcb.NewCommand(transferexample.CommandTypeCreateAccount, cmdData, nil)
				// For account creation
				handlerCreate := dcb.CommandHandlerFunc(func(ctx context.Context, store dcb.EventStore, command dcb.Command) ([]dcb.InputEvent, error) {
					events, _, err := transferexample.HandleCreateAccount(ctx, store, command)
					return events, err
				})
				commandExecutor := dcb.NewCommandExecutor(store)

				_, err = commandExecutor.ExecuteCommand(ctx, command, handlerCreate, nil)
				Expect(err).NotTo(HaveOccurred())

				nonExistentToCmd := transferexample.TransferMoneyCommand{
					TransferID:    "test_transfer_4",
					FromAccountID: "test_acc1",
					ToAccountID:   "non_existent_account",
					Amount:        100,
				}
				nonExistentData, err := json.Marshal(nonExistentToCmd)
				Expect(err).NotTo(HaveOccurred())

				nonExistentCommand := dcb.NewCommand(transferexample.CommandTypeTransferMoney, nonExistentData, nil)
				// For transfer commands, get the AppendCondition from the handler
				_, appendCondition, err := transferexample.HandleCommand(ctx, store, nonExistentCommand)
				Expect(err).NotTo(HaveOccurred())
				// Use the transfer handler, not the create handler
				handlerTransfer := dcb.CommandHandlerFunc(func(ctx context.Context, store dcb.EventStore, command dcb.Command) ([]dcb.InputEvent, error) {
					events, _, err := transferexample.HandleTransferMoney(ctx, store, command)
					return events, err
				})
				_, err = commandExecutor.ExecuteCommand(ctx, nonExistentCommand, handlerTransfer, appendCondition)
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	Describe("Sequential Transfers", func() {
		BeforeEach(func() {
			cleanupEvents()
		})

		It("should handle sequential transfers and detect insufficient funds", func() {
			// For account creation
			handlerCreate := dcb.CommandHandlerFunc(func(ctx context.Context, store dcb.EventStore, command dcb.Command) ([]dcb.InputEvent, error) {
				events, _, err := transferexample.HandleCreateAccount(ctx, store, command)
				return events, err
			})
			commandExecutor := dcb.NewCommandExecutor(store)

			createAccount3Cmd := transferexample.CreateAccountCommand{
				AccountID:      "test_acc3",
				Owner:          "test_user3",
				InitialBalance: 2000,
			}
			cmdData3, err := json.Marshal(createAccount3Cmd)
			Expect(err).NotTo(HaveOccurred())

			command3 := dcb.NewCommand(transferexample.CommandTypeCreateAccount, cmdData3, nil)
			_, err = commandExecutor.ExecuteCommand(ctx, command3, handlerCreate, nil)
			Expect(err).NotTo(HaveOccurred())

			createAccount4Cmd := transferexample.CreateAccountCommand{
				AccountID:      "test_acc4",
				Owner:          "test_user4",
				InitialBalance: 0,
			}
			cmdData4, err := json.Marshal(createAccount4Cmd)
			Expect(err).NotTo(HaveOccurred())

			command4 := dcb.NewCommand(transferexample.CommandTypeCreateAccount, cmdData4, nil)
			_, err = commandExecutor.ExecuteCommand(ctx, command4, handlerCreate, nil)
			Expect(err).NotTo(HaveOccurred())

			createAccount5Cmd := transferexample.CreateAccountCommand{
				AccountID:      "test_acc5",
				Owner:          "test_user5",
				InitialBalance: 0,
			}
			cmdData5, err := json.Marshal(createAccount5Cmd)
			Expect(err).NotTo(HaveOccurred())

			command5 := dcb.NewCommand(transferexample.CommandTypeCreateAccount, cmdData5, nil)
			_, err = commandExecutor.ExecuteCommand(ctx, command5, handlerCreate, nil)
			Expect(err).NotTo(HaveOccurred())

			transfer1Cmd := transferexample.TransferMoneyCommand{
				TransferID:    "test_transfer_5",
				FromAccountID: "test_acc3",
				ToAccountID:   "test_acc4",
				Amount:        1500,
			}
			transferData1, err := json.Marshal(transfer1Cmd)
			Expect(err).NotTo(HaveOccurred())

			transferCommand1 := dcb.NewCommand(transferexample.CommandTypeTransferMoney, transferData1, nil)
			// For transfer commands, get the AppendCondition from the handler
			_, appendCondition1, err := transferexample.HandleCommand(ctx, store, transferCommand1)
			Expect(err).NotTo(HaveOccurred())
			// Use the transfer handler, not the create handler
			handlerTransfer := dcb.CommandHandlerFunc(func(ctx context.Context, store dcb.EventStore, command dcb.Command) ([]dcb.InputEvent, error) {
				events, _, err := transferexample.HandleTransferMoney(ctx, store, command)
				return events, err
			})
			_, err = commandExecutor.ExecuteCommand(ctx, transferCommand1, handlerTransfer, appendCondition1)
			Expect(err).NotTo(HaveOccurred())

			transfer2Cmd := transferexample.TransferMoneyCommand{
				TransferID:    "test_transfer_6",
				FromAccountID: "test_acc3",
				ToAccountID:   "test_acc5",
				Amount:        1000,
			}
			transferData2, err := json.Marshal(transfer2Cmd)
			Expect(err).NotTo(HaveOccurred())

			transferCommand2 := dcb.NewCommand(transferexample.CommandTypeTransferMoney, transferData2, nil)
			// For transfer commands, get the AppendCondition from the handler
			// Note: HandleCommand will return an error for insufficient funds, which is expected
			_, _, err = transferexample.HandleCommand(ctx, store, transferCommand2)
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

	schemaSQL, err := os.ReadFile("../../../../docker-entrypoint-initdb.d/schema.sql")
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
