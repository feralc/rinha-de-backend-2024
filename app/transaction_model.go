package app

import "time"

type TransactionType string

const (
	CreditTransaction TransactionType = "c"
	DebitTransaction  TransactionType = "d"
)

type TransactionRequest struct {
	Amount    int             `json:"valor" binding:"required"`
	Type      TransactionType `json:"tipo"`
	Descricao string          `json:"descricao" binding:"required,min=1,max=10"`
}

type SuccessTransactionResult struct {
	CreditLimit int `json:"limite"`
	Balance     int `json:"saldo"`
}

type Transaction struct {
	ClientID    int             `json:"client_id,omitempty" bson:"client_id,omitempty"`
	Amount      int             `json:"valor" bson:"amount"`
	Type        TransactionType `json:"tipo" bson:"type"`
	Description string          `json:"descricao" bson:"description"`
	Timestamp   time.Time       `json:"realizada_em" bson:"created_at"`
	Revision    int             `json:"revision,omitempty" bson:"revision"`
}
