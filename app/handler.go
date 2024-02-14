package app

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func MakeTransactionsHandler(actorManager *ActorManager) func(c *gin.Context) {
	return func(c *gin.Context) {
		clientIDStr := c.Param("id")
		clientID, err := strconv.Atoi(clientIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid client ID"})
			return
		}

		var req TransactionRequest
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}

		actor := actorManager.Spawn(clientID, 100000)
		result := actor.Send(&ActorMessage{
			Type:    TransactionMessage,
			Payload: req,
		})

		if result.Error != nil {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": result.Error.Error()})
			return
		}

		c.JSON(http.StatusOK, result.Data)
	}
}

func MakeTransactionHistoryHandler(actorManager *ActorManager) func(c *gin.Context) {
	return func(c *gin.Context) {
		clientIDStr := c.Param("id")
		clientID, err := strconv.Atoi(clientIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid client ID"})
			return
		}

		// @TODO consultar client no banco de dados para ler o limite
		actor := actorManager.Spawn(clientID, 100000)

		history := actor.CurrentState().GetTransactionHistory()

		c.JSON(http.StatusOK, history)
	}
}
