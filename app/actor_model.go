package app

import (
	"context"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"
)

type MessageType string

const (
	RefreshMessage     MessageType = "Refresh"
	TransactionMessage MessageType = "Transaction"
	HistorySize                    = 10
)

type ActorMessage struct {
	Type    MessageType
	Payload any
}

type ActorResult struct {
	Error error
	Data  any
}

type ActorContext struct {
	store TransactionStore
}

type ClientActor struct {
	clientID int
	state    *ClientState
	inbox    chan *ActorMessage
	outbox   chan ActorResult
}

type ClientState struct {
	totalCreditLimit        int
	balance                 int
	lastTransactions        []Transaction
	lastTransactionRevision int
	mu                      *sync.RWMutex
}

func (s ClientState) GetTransactionHistory() TransactionHistory {
	s.mu.RLock()
	defer s.mu.RUnlock()

	lastTransactions := make([]Transaction, len(s.lastTransactions))
	copy(lastTransactions, s.lastTransactions)

	sort.Slice(lastTransactions, func(i, j int) bool {
		return lastTransactions[i].Timestamp.After(lastTransactions[j].Timestamp)
	})

	transactionsResponse := make([]TransactionResponse, len(s.lastTransactions))
	for i, t := range lastTransactions {
		transactionsResponse[i] = TransactionResponse{
			Amount:      t.Amount,
			Type:        t.Type,
			Description: t.Description,
			Timestamp:   t.Timestamp,
		}
	}

	return TransactionHistory{
		CreditLimit:      s.totalCreditLimit,
		Total:            s.balance,
		LastTransactions: transactionsResponse,
		Date:             time.Now(),
	}
}

func NewClientActor(clientID, totalCreditLimit int) *ClientActor {
	actor := &ClientActor{
		clientID: clientID,
		state: &ClientState{
			totalCreditLimit: totalCreditLimit,
			balance:          0,
			lastTransactions: make([]Transaction, 0),
			mu:               &sync.RWMutex{},
		},
		inbox:  make(chan *ActorMessage, 1000),
		outbox: make(chan ActorResult),
	}

	return actor
}

func (a *ClientActor) Start(ctx *ActorContext) {
	for msg := range a.inbox {
		switch msg.Type {
		case RefreshMessage:
			a.outbox <- a.rebuildFromHistory(ctx)
		case TransactionMessage:
			req, ok := msg.Payload.(TransactionRequest)

			if !ok {
				a.outbox <- ActorResult{
					Error: fmt.Errorf("invalid payload for actor message type %s", msg.Type),
				}
				continue
			}
			a.outbox <- a.processTransaction(ctx, req)
		}
	}
}

func (a *ClientActor) Send(msg *ActorMessage) ActorResult {
	a.inbox <- msg
	return <-a.outbox
}

func (a *ClientActor) CurrentState() ClientState {
	return *a.state
}

func (a *ClientActor) rebuildFromHistory(ctx *ActorContext) ActorResult {
	log.Printf("rebuilding actor state from history for client id %d\n", a.clientID)
	currentBalance, transactions, err := ctx.store.GetTransactionHistory(context.Background(), a.clientID)
	if err != nil {
		return ActorResult{
			Error: fmt.Errorf("error fetching transactions for client id %d: %w", a.clientID, err),
		}
	}
	a.rebuildState(currentBalance, transactions)

	return ActorResult{}
}

func (a *ClientActor) processTransaction(ctx *ActorContext, req TransactionRequest) ActorResult {
	a.state.mu.Lock()
	defer a.state.mu.Unlock()

	if req.Amount <= 0 {
		return ActorResult{
			Error: fmt.Errorf("o valor não pode ser menor que zero"),
		}
	}

	switch req.Type {
	case CreditTransaction:
		a.state.balance += req.Amount
	case DebitTransaction:
		newBalance := a.state.balance - req.Amount
		if newBalance < -a.state.totalCreditLimit {
			return ActorResult{
				Error: fmt.Errorf("sem limite para realizar a transacao"),
			}
		}
		a.state.balance = newBalance
	default:
		return ActorResult{
			Error: fmt.Errorf("tipo de transacao invalida, opcoes disponiveis c => credito, d => debito"),
		}
	}

	a.state.lastTransactionRevision++

	transaction := Transaction{
		ClientID:    a.clientID,
		Amount:      req.Amount,
		Type:        req.Type,
		Description: req.Descricao,
		Timestamp:   time.Now(),
		Revision:    a.state.lastTransactionRevision,
	}

	a.state.lastTransactions = append(a.state.lastTransactions, transaction)

	if len(a.state.lastTransactions) > HistorySize {
		a.state.lastTransactions = a.state.lastTransactions[1:]
	}

	go func() {
		if err := ctx.store.Add(context.Background(), a.state.balance, transaction); err != nil {
			log.Printf("error adding transaction to store for client id %d: %s", a.clientID, err.Error())
		}
	}()

	return ActorResult{
		Data: SuccessTransactionResponse{
			CreditLimit: a.state.totalCreditLimit,
			Balance:     a.state.balance,
		},
	}
}

func (a *ClientActor) rebuildState(lastSnapshot Snapshot, transactions []Transaction) {
	a.state.mu.Lock()
	defer a.state.mu.Unlock()

	a.state.balance = lastSnapshot.Balance
	a.state.lastTransactionRevision = lastSnapshot.Revision

	lastTenTransactions := make([]Transaction, 0, 10)

	for _, t := range transactions {
		if t.Revision > a.state.lastTransactionRevision {
			if t.Type == CreditTransaction {
				a.state.balance += t.Amount
			} else if t.Type == DebitTransaction {
				a.state.balance -= t.Amount
			}

			a.state.lastTransactionRevision = t.Revision
		}

		lastTenTransactions = append(lastTenTransactions, t)
		if len(lastTenTransactions) > HistorySize {
			lastTenTransactions = lastTenTransactions[1:]
		}
	}

	a.state.lastTransactions = lastTenTransactions
}
