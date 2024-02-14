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

type SuccessTransactionResponse struct {
	CreditLimit int `json:"limite"`
	Balance     int `json:"saldo"`
}

type TransactionHistoryResponse struct {
	CreditLimit      int           `json:"limite"`
	Total            int           `json:"total"`
	LastTransactions []Transaction `json:"ultimas_transacoes"`
	Date             time.Time     `json:"data_extrato"`
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
