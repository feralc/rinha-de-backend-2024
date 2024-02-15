package app

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func MakeTransactionsHandler(actorManager *ActorManager) func(c *gin.Context) {
	return enhanceClientActorHandler[TransactionRequest](actorManager, func(c *gin.Context, req TransactionRequest, actor *ClientActor) ActorResult {
		return actor.Send(ActorMessage{
			Type:    TransactionMessage,
			Payload: req,
		})
	})
}

func MakeTransactionHistoryHandler(actorManager *ActorManager) func(c *gin.Context) {
	return enhanceClientActorHandler(actorManager, func(c *gin.Context, req any, actor *ClientActor) ActorResult {
		return actor.Send(ActorMessage{
			Type:    QueryHistoryMessage,
			Payload: req,
		})
	})
}

func enhanceClientActorHandler[R any](actorManager *ActorManager, handler func(c *gin.Context, req R, actor *ClientActor) ActorResult) func(c *gin.Context) {
	return func(c *gin.Context) {
		clientIDStr := c.Param("id")
		clientID, err := strconv.Atoi(clientIDStr)
		if err != nil {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "invalid client ID"})
			return
		}

		var req R
		if err := c.ShouldBind(&req); err != nil {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "invalid request body"})
			return
		}

		actor, err := actorManager.Spawn(clientID)

		if err != nil {
			if err == ErrNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "client not found"})
				return
			}
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
			return
		}

		result := handler(c, req, actor)

		if result.Error != nil {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": result.Error.Error()})
			return
		}

		c.JSON(http.StatusOK, result.Data)
	}
}
