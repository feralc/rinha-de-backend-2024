package app

import (
	"context"
	"sync"
)

type inMemoryTransactionStore struct {
	transactions   map[int][]Transaction
	mu             sync.Mutex
	currentBalance int
}

func NewInMemoryTransactionStore() TransactionStore {
	return &inMemoryTransactionStore{
		transactions:   make(map[int][]Transaction),
		currentBalance: 0,
	}
}

func (s *inMemoryTransactionStore) Add(ctx context.Context, currentBalance int, transaction Transaction) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.transactions[transaction.ClientID] = append(s.transactions[transaction.ClientID], transaction)
	s.currentBalance = currentBalance

	return nil
}

func (s *inMemoryTransactionStore) GetTransactionHistory(ctx context.Context, clientID int) (Snapshot, []Transaction, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	transactions := make([]Transaction, 0)

	if len(s.transactions[clientID]) > 0 {
		transactions = s.transactions[clientID]
	}

	return Snapshot{}, transactions, nil
}
