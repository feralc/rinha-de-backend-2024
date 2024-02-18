package app

import (
	"encoding/json"
	"net/http"
	"strconv"
)

type HttpRequest interface {
	Validate() error
}

func MakeTransactionsHandler(actorManager *ActorManager) http.HandlerFunc {
	return enhanceClientActorHandler[TransactionRequest](actorManager, func(w http.ResponseWriter, r *http.Request, req TransactionRequest, actor *ClientActor) ActorResult {
		return actor.Send(ActorMessage{
			Type:    TransactionMessage,
			Payload: req,
		})
	})
}

func MakeTransactionHistoryHandler(actorManager *ActorManager) http.HandlerFunc {
	return enhanceClientActorHandler(actorManager, func(w http.ResponseWriter, r *http.Request, req HttpRequest, actor *ClientActor) ActorResult {
		return actor.Send(ActorMessage{
			Type:    QueryHistoryMessage,
			Payload: req,
		})
	})
}

func enhanceClientActorHandler[R HttpRequest](actorManager *ActorManager, handler func(w http.ResponseWriter, r *http.Request, req R, actor *ClientActor) ActorResult) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		clientIDStr := r.PathValue("id")
		clientID, err := strconv.Atoi(clientIDStr)
		if err != nil {
			http.Error(w, "invalid client id", http.StatusUnprocessableEntity)
			return
		}

		var req R

		if r.Method == http.MethodPost {
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "invalid request body", http.StatusUnprocessableEntity)
				return
			}

			if err := req.Validate(); err != nil {
				http.Error(w, err.Error(), http.StatusUnprocessableEntity)
				return
			}
		}

		actor, err := actorManager.Spawn(clientID)
		if err != nil {
			if err == ErrNotFound {
				http.Error(w, "client not found", http.StatusNotFound)
				return
			}
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}

		result := handler(w, r, req, actor)

		if result.Error != nil {
			http.Error(w, result.Error.Error(), http.StatusUnprocessableEntity)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result.Data)
	}
}
