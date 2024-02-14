package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/feralc/rinha-backend-2024/app"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	store, close := buildTransactionStore()
	defer close()

	clientManager := app.NewClientManager(store)

	r := gin.Default()

	r.POST("/clientes/:id/transacoes", func(c *gin.Context) {
		clientIDStr := c.Param("id")
		clientID, err := strconv.Atoi(clientIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid client ID"})
			return
		}

		var req app.TransactionRequest
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}

		actor := clientManager.Spawn(clientID, 100000)
		result := actor.Send(&app.ActorMessage{
			Type:    app.TransactionMessage,
			Payload: req,
		})

		if result.Error != nil {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": result.Error.Error()})
			return
		}

		c.JSON(http.StatusOK, result.Data)
	})

	r.GET("/clientes/:id/extrato", func(c *gin.Context) {
		clientIDStr := c.Param("id")
		clientID, err := strconv.Atoi(clientIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid client ID"})
			return
		}

		// @TODO consultar client no banco de dados para ler o limite
		actor := clientManager.Spawn(clientID, 100000)

		history := actor.CurrentState().GetTransactionHistory()

		c.JSON(http.StatusOK, history)
	})

	if err := r.Run(":8080"); err != nil {
		fmt.Printf("Failed to start server: %v\n", err)
	}
}

func buildTransactionStore() (store app.TransactionStore, close func()) {
	if inMemory, _ := strconv.ParseBool(os.Getenv("APP_IN_MEMORY")); inMemory {
		return app.NewInMemoryTransactionStore(), close
	}

	clientOptions := options.Client().ApplyURI("mongodb://db:27017")
	client, err := mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		panic(err)
	}

	return app.NewMongoDBTransactionStore(client), func() { client.Disconnect(context.Background()) }
}
