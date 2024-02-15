package app

import "time"

const HistorySize = 10

type TransactionHistory struct {
	Balance          TransactionHistoryBalance `json:"saldo"`
	LastTransactions []TransactionSummary      `json:"ultimas_transacoes"`
}

type TransactionHistoryBalance struct {
	CreditLimit int       `json:"limite"`
	Total       int       `json:"total"`
	Date        time.Time `json:"data_extrato"`
}

type TransactionSummary struct {
	Amount      int             `json:"valor"`
	Type        TransactionType `json:"tipo"`
	Description string          `json:"descricao"`
	Timestamp   time.Time       `json:"realizada_em"`
}

func (h *TransactionHistory) RegisterTransaction(t Transaction) {
	h.LastTransactions = append(h.LastTransactions, TransactionSummary{
		Amount:      t.Amount,
		Type:        t.Type,
		Description: t.Description,
		Timestamp:   t.Timestamp,
	})

	n := len(h.LastTransactions)
	for i := n - 1; i > 0 && h.LastTransactions[i].Timestamp.After(h.LastTransactions[i-1].Timestamp); i-- {
		h.LastTransactions[i], h.LastTransactions[i-1] = h.LastTransactions[i-1], h.LastTransactions[i]
	}

	if n > HistorySize {
		h.LastTransactions = h.LastTransactions[:HistorySize]
	}
}

func (h *TransactionHistory) Clear() {
	h.Balance.Total = 0
	h.LastTransactions = []TransactionSummary{}
}
