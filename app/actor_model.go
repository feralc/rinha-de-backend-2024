package app

import (
	"context"
	"fmt"
	"log"
	"time"
)

type MessageType string

const (
	RefreshMessage     MessageType = "Refresh"
	TransactionMessage MessageType = "Transaction"
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
	totalCreditLimit int
	balance          int
	lastTransactions []Transaction
}

func (s ClientState) LastTransactions() []Transaction {
	return s.lastTransactions
}

func NewClientActor(clientID, totalCreditLimit int) *ClientActor {
	actor := &ClientActor{
		clientID: clientID,
		state: &ClientState{
			totalCreditLimit: totalCreditLimit,
			balance:          0,
			lastTransactions: make([]Transaction, 0),
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
	transactions, err := ctx.store.GetTransactionHistory(context.Background(), a.clientID)
	if err != nil {
		return ActorResult{
			Error: fmt.Errorf("error fetching transactions for client id %d: %w", a.clientID, err),
		}
	}
	a.rebuildState(transactions)

	return ActorResult{}
}

func (a *ClientActor) processTransaction(ctx *ActorContext, req TransactionRequest) ActorResult {
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

	transaction := Transaction{
		ClientID:    a.clientID,
		Amount:      req.Amount,
		Type:        req.Type,
		Description: req.Descricao,
		Timestamp:   time.Now(),
	}

	// @TODO quando chegar no indice 10 precisamos ir ciclando o array pra manter somente as 10 ultimas transactions
	// lembrar de usar rwmutex tanto aqui quando na alteração do saldo
	a.state.lastTransactions = append(a.state.lastTransactions, transaction)

	go func() {
		if err := ctx.store.Add(context.Background(), transaction); err != nil {
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

func (a *ClientActor) rebuildState(transactions []Transaction) {
	a.state.balance = 0

	for _, t := range transactions {
		if t.Type == CreditTransaction {
			a.state.balance += t.Amount
		} else if t.Type == DebitTransaction {
			a.state.balance -= t.Amount
		}
	}
}
