package main

import (
	"fmt"
	"log"

	"github.com/feralc/rinha-backend-2024/app"
	"github.com/gin-gonic/gin"
)

func main() {
	store, close := app.BuildTransactionStore()
	defer close()

	actorManager := app.NewActorManager(store)

	gin.SetMode(gin.ReleaseMode)

	r := gin.New()
	r.Use(gin.Recovery())

	r.POST("/clientes/:id/transacoes", app.MakeTransactionsHandler(actorManager))
	r.GET("/clientes/:id/extrato", app.MakeTransactionHistoryHandler(actorManager))

	log.Println("server listening on :8080")
	if err := r.Run(":8080"); err != nil {
		fmt.Printf("failed to start server: %v\n", err)
	}
}
