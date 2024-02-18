package app

import (
	"fmt"
	"time"
)

type TransactionType string

const (
	CreditTransaction TransactionType = "c"
	DebitTransaction  TransactionType = "d"
)

type TransactionRequest struct {
	HttpRequest
	Amount    int             `json:"valor" binding:"required"`
	Type      TransactionType `json:"tipo"`
	Descricao string          `json:"descricao" binding:"required,min=1,max=10"`
}

func (r TransactionRequest) Validate() error {
	if r.Amount <= 0 {
		return fmt.Errorf("o valor deve ser maior que zero")
	}

	if r.Type != CreditTransaction && r.Type != DebitTransaction {
		return fmt.Errorf("tipo de transacao invalida")
	}

	if r.Descricao == "" || len(r.Descricao) > 10 {
		return fmt.Errorf("descricao deve ter entre 1 e 10 caracteres")
	}

	return nil
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
