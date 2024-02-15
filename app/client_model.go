package app

import (
	"fmt"
	"time"
)

type Client struct {
	ID                      int `bson:"client_id"`
	CreditLimit             int `bson:"limit"`
	Balance                 int `bson:"balance"`
	history                 TransactionHistory
	lastTransactionRevision int
}

func (c *Client) ProcessTransaction(req TransactionRequest) (result Transaction, err error) {
	if req.Amount <= 0 {
		return result, fmt.Errorf("o valor não pode ser menor que zero")
	}

	if req.Type != CreditTransaction && req.Type != DebitTransaction {
		return result, fmt.Errorf("tipo de transacao invalida")
	}

	switch req.Type {
	case CreditTransaction:
		c.Balance += req.Amount
	case DebitTransaction:
		newBalance := c.Balance - req.Amount
		if newBalance < -c.CreditLimit {
			return result, fmt.Errorf("sem limite para realizar a transacao")
		}
		c.Balance = newBalance
	}

	c.lastTransactionRevision++

	transaction := Transaction{
		ClientID:    c.ID,
		Amount:      req.Amount,
		Type:        req.Type,
		Description: req.Descricao,
		Timestamp:   time.Now(),
		Revision:    c.lastTransactionRevision,
	}

	c.history.RegisterTransaction(transaction)

	return transaction, nil
}

func (c *Client) RebuildStateFromHistory(lastSnapshot Snapshot, transactions []Transaction) {
	c.Balance = lastSnapshot.Balance
	c.lastTransactionRevision = lastSnapshot.Revision
	c.history.Clear()

	for _, t := range transactions {
		if t.Revision > c.lastTransactionRevision {
			if t.Type == CreditTransaction {
				c.Balance += t.Amount
			} else if t.Type == DebitTransaction {
				c.Balance -= t.Amount
			}

			c.lastTransactionRevision = t.Revision
		}

		c.history.RegisterTransaction(t)
	}
}

func (c *Client) GetTransactionHistory() *TransactionHistory {
	h := c.history
	h.Balance.Date = time.Now()
	h.Balance.Total = c.Balance
	h.Balance.CreditLimit = c.CreditLimit
	return &h
}
