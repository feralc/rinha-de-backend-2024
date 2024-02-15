package app

import (
	"context"
	"sync"
)

type ActorManager struct {
	clients          map[int]*ClientActor
	mutex            sync.Mutex
	transactionStore TransactionStore
	clientStore      ClientStore
}

func NewActorManager(clientStore ClientStore, transactionStore TransactionStore) *ActorManager {
	return &ActorManager{
		clients:          make(map[int]*ClientActor),
		clientStore:      clientStore,
		transactionStore: transactionStore,
	}
}

func (m *ActorManager) Spawn(clientID int) (*ClientActor, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if actor, ok := m.clients[clientID]; ok {
		return actor, nil
	}

	client, err := m.clientStore.GetOne(context.Background(), clientID)
	if err != nil {
		return nil, err
	}

	actor := NewClientActor(&client)
	m.clients[clientID] = actor

	defer actor.Send(ActorMessage{Type: RefreshMessage})

	ctx := &ActorContext{
		store: m.transactionStore,
	}

	go actor.Start(ctx)

	return actor, nil
}
