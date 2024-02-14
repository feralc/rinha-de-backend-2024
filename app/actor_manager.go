package app

import (
	"sync"
)

type ActorManager struct {
	clients map[int]*ClientActor
	mutex   sync.Mutex
	store   TransactionStore
}

func NewActorManager(store TransactionStore) *ActorManager {
	return &ActorManager{
		clients: make(map[int]*ClientActor),
		store:   store,
	}
}

func (cm *ActorManager) Spawn(clientID, limit int) *ClientActor {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	actor, ok := cm.clients[clientID]
	if !ok {
		actor = NewClientActor(clientID, limit)
		cm.clients[clientID] = actor
		defer actor.Send(&ActorMessage{Type: RefreshMessage})
	}

	ctx := &ActorContext{
		store: cm.store,
	}

	go actor.Start(ctx)

	return actor
}
