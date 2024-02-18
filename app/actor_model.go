package app

import (
	"context"
	"fmt"
	"log"
)

type MessageType rune

const (
	RefreshMessage      MessageType = 'R'
	TransactionMessage  MessageType = 'T'
	QueryHistoryMessage MessageType = 'Q'
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
	client *Client
	inbox  chan ActorMessage
	outbox chan ActorResult
}

func NewClientActor(client *Client) *ClientActor {
	actor := &ClientActor{
		client: client,
		inbox:  make(chan ActorMessage),
		outbox: make(chan ActorResult),
	}

	return actor
}

func (a *ClientActor) Send(msg ActorMessage) ActorResult {
	a.inbox <- msg
	return <-a.outbox
}

func (a *ClientActor) Start(ctx *ActorContext) {
	for msg := range a.inbox {
		switch msg.Type {
		case RefreshMessage:
			a.outbox <- a.handleRefreshMessage(ctx)
		case TransactionMessage:
			a.outbox <- a.handleTransactionMessage(ctx, msg)
		case QueryHistoryMessage:
			a.outbox <- ActorResult{
				Data: a.client.GetTransactionHistory(),
			}
		}
	}
}

func (a *ClientActor) handleRefreshMessage(ctx *ActorContext) ActorResult {
	snapshot, transactions, err := ctx.store.GetTransactionHistory(context.Background(), a.client.ID)
	if err != nil {
		return ActorResult{
			Error: fmt.Errorf("error fetching transactions for client id %d: %w", a.client.ID, err),
		}
	}

	a.client.RebuildStateFromHistory(snapshot, transactions)

	return ActorResult{}
}

func (a *ClientActor) handleTransactionMessage(ctx *ActorContext, msg ActorMessage) ActorResult {
	req, ok := msg.Payload.(TransactionRequest)
	if !ok {
		return ActorResult{
			Error: fmt.Errorf("invalid payload for actor message type %c", msg.Type),
		}
	}

	transaction, err := a.client.ProcessTransaction(req)

	if err != nil {
		return ActorResult{
			Error: err,
		}
	}

	go func() {
		if err := ctx.store.Add(context.Background(), a.client.Balance, transaction); err != nil {
			log.Println(fmt.Errorf("error adding transaction to store for client id %d: %s", a.client.ID, err.Error()))
		}
	}()

	return ActorResult{
		Data: SuccessTransactionResult{
			CreditLimit: a.client.CreditLimit,
			Balance:     a.client.Balance,
		},
	}
}
