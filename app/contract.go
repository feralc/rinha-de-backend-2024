package app

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

var NotFoundErr = fmt.Errorf("not found")

type Client struct {
	ID             int `bson:"client_id"`
	CreditLimit    int `bson:"limit"`
	InitialBalance int `bson:"initial_balance"`
}

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

type TransactionHistory struct {
	Balance          TransactionHistoryBalance `json:"saldo"`
	LastTransactions []TransactionResponse     `json:"ultimas_transacoes"`
}

type TransactionHistoryBalance struct {
	CreditLimit int       `json:"limite"`
	Total       int       `json:"total"`
	Date        time.Time `json:"data_extrato"`
}

type Transaction struct {
	ClientID    int             `json:"client_id,omitempty" bson:"client_id,omitempty"`
	Amount      int             `json:"valor" bson:"amount"`
	Type        TransactionType `json:"tipo" bson:"type"`
	Description string          `json:"descricao" bson:"description"`
	Timestamp   time.Time       `json:"realizada_em" bson:"created_at"`
	Revision    int             `json:"revision,omitempty" bson:"revision"`
}

type TransactionResponse struct {
	Amount      int             `json:"valor"`
	Type        TransactionType `json:"tipo"`
	Description string          `json:"descricao"`
	Timestamp   time.Time       `json:"realizada_em"`
}

type Snapshot struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	Revision  int                `bson:"revision"`
	Balance   int                `bson:"balance"`
	CreatedAt time.Time          `bson:"created_at"`
}

type TransactionStore interface {
	Add(ctx context.Context, currentBalance int, transaction Transaction) error
	GetTransactionHistory(ctx context.Context, clientID int) (lastSnapshot Snapshot, transactions []Transaction, err error)
}

type ClientStore interface {
	Add(ctx context.Context, client Client) error
	GetOne(ctx context.Context, clientId int) (client Client, err error)
}
