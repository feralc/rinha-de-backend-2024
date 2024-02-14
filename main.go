package main

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/feralc/rinha-backend-2024/app"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {

	clientOptions := options.Client().ApplyURI("mongodb://db:27017")
	client, err := mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		panic(err)
	}
	defer client.Disconnect(context.Background())

	store := app.NewMongoDBTransactionStore(client)

	clientManager := app.NewClientManager(store)

	r := gin.Default()

	r.POST("/clientes/:id/transacoes", func(c *gin.Context) {
		clientIDStr := c.Param("id")
		clientID, err := strconv.Atoi(clientIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid client ID"})
			return
		}

		var req app.TransactionRequest
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
			return
		}

		actor := clientManager.Spawn(clientID, 100000)
		response := actor.Send(&app.ActorMessage{
			Type:    app.TransactionMessage,
			Payload: app.TransactionRequest{},
		})

		c.JSON(http.StatusOK, response)
	})

	r.GET("/clientes/:id/extrato", func(c *gin.Context) {
		clientIDStr := c.Param("id")
		clientID, err := strconv.Atoi(clientIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid client ID"})
			return
		}

		transactions, err := store.GetTransactionHistory(c.Request.Context(), clientID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch transaction history"})
			return
		}

		//@TODO move this logic to better place
		totalBalance := calculateTotalBalance(transactions)

		response := app.TransactionResponse{
			CreditLimit:      100000,
			Balance:          totalBalance,
			LastTransactions: transactions,
		}

		c.JSON(http.StatusOK, response)
	})

	if err := r.Run(":8080"); err != nil {
		fmt.Printf("Failed to start server: %v\n", err)
	}
}

func calculateTotalBalance(transactions []app.Transaction) int {
	var totalBalance int
	for _, t := range transactions {
		if t.Type == app.CreditTransaction {
			totalBalance += t.Amount
		} else if t.Type == app.DebitTransaction {
			totalBalance -= t.Amount
		}
	}
	return totalBalance
}
