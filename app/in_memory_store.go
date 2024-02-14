package app

import (
	"context"
	"sync"
)

type inMemoryTransactionStore struct {
	transactions map[int][]Transaction
	mu           sync.Mutex
}

func NewInMemoryTransactionStore() TransactionStore {
	return &inMemoryTransactionStore{
		transactions: make(map[int][]Transaction),
	}
}

func (s *inMemoryTransactionStore) Add(ctx context.Context, transaction Transaction) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.transactions[transaction.ClientID] = append(s.transactions[transaction.ClientID], transaction)

	return nil
}

func (s *inMemoryTransactionStore) GetTransactionHistory(ctx context.Context, clientID int) ([]Transaction, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	transactions := make([]Transaction, 0)

	if len(s.transactions[clientID]) > 0 {
		transactions = s.transactions[clientID]
	}

	return transactions, nil
}
