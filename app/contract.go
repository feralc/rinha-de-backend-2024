package app

import (
	"context"
	"time"
)

type TransactionType string

const (
	CreditTransaction TransactionType = "c"
	DebitTransaction  TransactionType = "d"
)

type TransactionRequest struct {
	Amount    int             `json:"valor"`
	Type      TransactionType `json:"tipo"`
	Descricao string          `json:"descricao"`
}

type TransactionResponse struct {
	CreditLimit      int           `json:"limite"`
	Balance          int           `json:"saldo"`
	LastTransactions []Transaction `json:"ultimas_transacoes,omitempty"`
}

type Transaction struct {
	ClientID    int
	Amount      int             `json:"valor"`
	Type        TransactionType `json:"tipo"`
	Description string          `json:"descricao"`
	Timestamp   time.Time       `json:"realizada_em"`
}

type TransactionStore interface {
	Add(ctx context.Context, transaction Transaction) error
	GetTransactionHistory(ctx context.Context, clientID int) ([]Transaction, error)
}
