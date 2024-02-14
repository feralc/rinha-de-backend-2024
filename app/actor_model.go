package app

import (
	"context"
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

type ActorContext struct {
	store TransactionStore
}

type ClientActor struct {
	clientID  int
	state     *ClientState
	inbox     chan *ActorMessage
	responses chan *TransactionResponse
}

type ClientState struct {
	balance     int
	creditLimit int
}

func NewClientActor(clientID, creditLimit int, responses chan *TransactionResponse) *ClientActor {
	actor := &ClientActor{
		clientID:  clientID,
		state:     &ClientState{0, creditLimit},
		inbox:     make(chan *ActorMessage),
		responses: responses,
	}

	return actor
}

func (a *ClientActor) Start(ctx *ActorContext) {
	for msg := range a.inbox {
		switch msg.Type {
		case RefreshMessage:
			log.Printf("refreshing actor state for actor id %d\n", a.clientID)
			transactions, err := ctx.store.GetTransactionHistory(context.Background(), a.clientID)
			if err != nil {
				log.Println("error fetching transactions:", err)
				continue
			}
			a.RebuildState(transactions)
		case TransactionMessage:
			// @TODO handle cast error
			req := msg.Payload.(TransactionRequest)
			a.responses <- a.processTransaction(ctx, req)
		}
	}
}

func (a *ClientActor) processTransaction(ctx *ActorContext, req TransactionRequest) *TransactionResponse {
	switch req.Type {
	case CreditTransaction:
		a.state.balance += req.Amount
	case DebitTransaction:
		if a.state.balance-req.Amount < -a.state.creditLimit {
			return &TransactionResponse{
				CreditLimit: a.state.creditLimit,
				Balance:     a.state.balance,
			}
		}
		a.state.balance -= req.Amount
	default:
		log.Println("Invalid transaction type")
	}

	err := ctx.store.Add(context.Background(), Transaction{
		ClientID:    a.clientID,
		Amount:      req.Amount,
		Type:        req.Type,
		Description: req.Descricao,
		Timestamp:   time.Now(),
	})

	//@TODO handle response proper (maybe use ctx??)
	if err != nil {
		log.Println("error adding transaction to store:", err)
	}

	return &TransactionResponse{
		CreditLimit: a.state.creditLimit,
		Balance:     a.state.balance,
	}
}

func (a *ClientActor) Send(msg *ActorMessage) *TransactionResponse {
	a.inbox <- msg
	return <-a.responses
}

func (a *ClientActor) RebuildState(transactions []Transaction) {
	a.state.balance = 0

	for _, t := range transactions {
		if t.Type == CreditTransaction {
			a.state.balance += t.Amount
		} else if t.Type == DebitTransaction {
			a.state.balance -= t.Amount
		}
	}
}
