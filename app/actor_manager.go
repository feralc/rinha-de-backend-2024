package app

import (
	"sync"
)

type ClientManager struct {
	clients   map[int]*ClientActor
	mutex     sync.Mutex
	responses chan *TransactionResponse
	store     TransactionStore
}

func NewClientManager(store TransactionStore) *ClientManager {
	return &ClientManager{
		clients:   make(map[int]*ClientActor),
		responses: make(chan *TransactionResponse),
		store:     store,
	}
}

func (cm *ClientManager) Spawn(clientID, limit int) *ClientActor {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	actor, ok := cm.clients[clientID]
	if !ok {
		actor = NewClientActor(clientID, limit, cm.responses)
		cm.clients[clientID] = actor
		defer actor.Send(&ActorMessage{Type: RefreshMessage})
	}

	ctx := &ActorContext{
		store: cm.store,
	}

	go actor.Start(ctx)

	return actor
}
