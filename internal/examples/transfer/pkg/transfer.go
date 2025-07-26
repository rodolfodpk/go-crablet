// Package transfer provides types and functions for the transfer example
package transfer

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rodolfodpk/go-crablet/pkg/dcb"
)

// AccountState holds the state for an account
type AccountState struct {
	AccountID string
	Owner     string
	Balance   int
	CreatedAt time.Time
	UpdatedAt time.Time
}

// AccountOpened represents when an account is opened
type AccountOpened struct {
	AccountID      string    `json:"account_id"`
	Owner          string    `json:"owner"`
	InitialBalance int       `json:"initial_balance"`
	OpenedAt       time.Time `json:"opened_at"`
}

// MoneyTransferred represents a money transfer between accounts
type MoneyTransferred struct {
	TransferID    string    `json:"transfer_id"`
	FromAccountID string    `json:"from_account_id"`
	ToAccountID   string    `json:"to_account_id"`
	Amount        int       `json:"amount"`
	FromBalance   int       `json:"from_balance"` // Balance after transfer
	ToBalance     int       `json:"to_balance"`   // Balance after transfer
	TransferredAt time.Time `json:"transferred_at"`
	Description   string    `json:"description,omitempty"`
}

// Command types
const (
	CommandTypeCreateAccount = "create_account"
	CommandTypeTransferMoney = "transfer_money"
)

// CreateAccountCommand represents a command to create an account
type CreateAccountCommand struct {
	AccountID      string `json:"account_id"`
	Owner          string `json:"owner"`
	InitialBalance int    `json:"initial_balance"`
}

// TransferMoneyCommand represents a command to transfer money between accounts
type TransferMoneyCommand struct {
	TransferID    string `json:"transfer_id"`
	FromAccountID string `json:"from_account_id"`
	ToAccountID   string `json:"to_account_id"`
	Amount        int    `json:"amount"`
	Description   string `json:"description,omitempty"`
}

// HandleCreateAccount handles the creation of an account
func HandleCreateAccount(ctx context.Context, store dcb.EventStore, command dcb.Command) ([]dcb.InputEvent, *dcb.AppendCondition, error) {
	var cmd CreateAccountCommand
	if err := json.Unmarshal(command.GetData(), &cmd); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal create account command: %w", err)
	}

	// Check for duplicate account
	query := dcb.NewQuery(dcb.NewTags("account_id", cmd.AccountID), "AccountOpened")
	events, err := store.Query(ctx, query, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query for existing account: %w", err)
	}
	if len(events) > 0 {
		return nil, nil, fmt.Errorf("account %s already exists", cmd.AccountID)
	}

	// Create the event
	event := AccountOpened{
		AccountID:      cmd.AccountID,
		Owner:          cmd.Owner,
		InitialBalance: cmd.InitialBalance,
		OpenedAt:       time.Now(),
	}
	eventData, err := json.Marshal(event)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal account opened event: %w", err)
	}

	return []dcb.InputEvent{
		dcb.NewInputEvent("AccountOpened", []dcb.Tag{
			dcb.NewTag("account_id", cmd.AccountID),
		}, eventData),
	}, nil, nil
}

// HandleTransferMoney handles money transfers between accounts
func HandleTransferMoney(ctx context.Context, store dcb.EventStore, command dcb.Command) ([]dcb.InputEvent, *dcb.AppendCondition, error) {
	var cmd TransferMoneyCommand
	if err := json.Unmarshal(command.GetData(), &cmd); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal transfer money command: %w", err)
	}

	// Define projectors for account states (DCB pattern)
	projectors := []dcb.StateProjector{
		{
			ID:           "fromAccount",
			Query:        dcb.NewQuery(dcb.NewTags("account_id", cmd.FromAccountID), "AccountOpened"),
			InitialState: &AccountState{AccountID: cmd.FromAccountID},
			TransitionFn: func(state any, event dcb.Event) any {
				acc := state.(*AccountState)
				switch event.Type {
				case "AccountOpened":
					var data AccountOpened
					if err := json.Unmarshal(event.Data, &data); err == nil {
						acc.Owner = data.Owner
						acc.Balance = data.InitialBalance
						acc.CreatedAt = data.OpenedAt
						acc.UpdatedAt = data.OpenedAt
					}
				}
				return acc
			},
		},
		{
			ID:           "toAccount",
			Query:        dcb.NewQuery(dcb.NewTags("account_id", cmd.ToAccountID), "AccountOpened"),
			InitialState: &AccountState{AccountID: cmd.ToAccountID},
			TransitionFn: func(state any, event dcb.Event) any {
				acc := state.(*AccountState)
				switch event.Type {
				case "AccountOpened":
					var data AccountOpened
					if err := json.Unmarshal(event.Data, &data); err == nil {
						acc.Owner = data.Owner
						acc.Balance = data.InitialBalance
						acc.CreatedAt = data.OpenedAt
						acc.UpdatedAt = data.OpenedAt
					}
				}
				return acc
			},
		},
		{
			ID:           "allTransfers",
			Query:        dcb.NewQuery(nil, "MoneyTransferred"),
			InitialState: []MoneyTransferred{},
			TransitionFn: func(state any, event dcb.Event) any {
				transfers := state.([]MoneyTransferred)
				if event.Type == "MoneyTransferred" {
					var data MoneyTransferred
					if err := json.Unmarshal(event.Data, &data); err == nil {
						transfers = append(transfers, data)
					}
				}
				return transfers
			},
		},
	}

	// Project the account states
	states, _, err := store.Project(ctx, projectors, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to project account states: %w", err)
	}

	fromAccount, fromOk := states["fromAccount"].(*AccountState)
	toAccount, toOk := states["toAccount"].(*AccountState)
	allTransfers, transfersOk := states["allTransfers"].([]MoneyTransferred)

	if !fromOk || !toOk || !transfersOk {
		return nil, nil, fmt.Errorf("failed to get account states from projection")
	}

	// Apply transfer history to calculate current balances
	for _, transfer := range allTransfers {
		if transfer.FromAccountID == cmd.FromAccountID {
			fromAccount.Balance = transfer.FromBalance
			fromAccount.UpdatedAt = transfer.TransferredAt
		} else if transfer.ToAccountID == cmd.FromAccountID {
			fromAccount.Balance = transfer.ToBalance
			fromAccount.UpdatedAt = transfer.TransferredAt
		}

		if transfer.FromAccountID == cmd.ToAccountID {
			toAccount.Balance = transfer.FromBalance
			toAccount.UpdatedAt = transfer.TransferredAt
		} else if transfer.ToAccountID == cmd.ToAccountID {
			toAccount.Balance = transfer.ToBalance
			toAccount.UpdatedAt = transfer.TransferredAt
		}
	}

	// Validate that the FROM account exists (required for transfer)
	if fromAccount.Owner == "" {
		return nil, nil, fmt.Errorf("source account %s does not exist", cmd.FromAccountID)
	}
	// Note: TO account can be non-existent - this allows for instant account creation
	// during transfers, which is a valid business scenario in some banking systems
	// The transfer will create the destination account with the transferred amount as initial balance

	// Validate transfer
	if fromAccount.Balance < cmd.Amount {
		return nil, nil, fmt.Errorf("insufficient funds: account %s has %d, needs %d", cmd.FromAccountID, fromAccount.Balance, cmd.Amount)
	}

	// Calculate new balances
	newFromBalance := fromAccount.Balance - cmd.Amount
	newToBalance := toAccount.Balance + cmd.Amount

	// Create transfer event
	transferEvent := MoneyTransferred{
		TransferID:    cmd.TransferID,
		FromAccountID: cmd.FromAccountID,
		ToAccountID:   cmd.ToAccountID,
		Amount:        cmd.Amount,
		FromBalance:   newFromBalance,
		ToBalance:     newToBalance,
		TransferredAt: time.Now(),
		Description:   cmd.Description,
	}
	eventData, err := json.Marshal(transferEvent)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal money transferred event: %w", err)
	}

	// Create AppendCondition to ensure the FROM account still exists when we append
	// This prevents race conditions where the source account could be deleted between projection and append
	item1 := dcb.NewQueryItem([]string{"AccountOpened"}, []dcb.Tag{dcb.NewTag("account_id", cmd.FromAccountID)})
	query := dcb.NewQueryFromItems(item1)
	appendCondition := dcb.NewAppendCondition(query)

	return []dcb.InputEvent{
		dcb.NewInputEvent("MoneyTransferred", []dcb.Tag{
			dcb.NewTag("transfer_id", cmd.TransferID),
			dcb.NewTag("from_account_id", cmd.FromAccountID),
			dcb.NewTag("to_account_id", cmd.ToAccountID),
		}, eventData),
	}, &appendCondition, nil
}

// HandleCommand is a unified command handler function
func HandleCommand(ctx context.Context, store dcb.EventStore, command dcb.Command) ([]dcb.InputEvent, *dcb.AppendCondition, error) {
	switch command.GetType() {
	case CommandTypeCreateAccount:
		return HandleCreateAccount(ctx, store, command)
	case CommandTypeTransferMoney:
		return HandleTransferMoney(ctx, store, command)
	default:
		return nil, nil, fmt.Errorf("unknown command type: %s", command.GetType())
	}
}
