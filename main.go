package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/feralc/rinha-backend-2024/app"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	ctx := context.Background()

	clientOptions := options.Client().ApplyURI("mongodb://127.0.0.1:27017")
	mongoClient, err := mongo.Connect(ctx, clientOptions)

	if err != nil {
		panic(err)
	}

	defer mongoClient.Disconnect(context.Background())

	db := mongoClient.Database(app.DatabaseName)

	if err := db.Drop(ctx); err != nil {
		panic(err)
	}

	db.Collection(app.TransactionsCollectionName).Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.M{"client_id": 1},
			Options: options.Index(),
		},
		{
			Keys:    bson.M{"revision": 1},
			Options: options.Index(),
		},
	})

	db.Collection(app.SnapshotsCollectionName).Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.M{"created_at": 1},
		Options: options.Index(),
	})

	transactionStore := app.NewMongoDBTransactionStore(mongoClient)
	clientsStore := app.NewMongoDBClientStore(mongoClient)

	initialClients := []app.Client{
		{ID: 1, CreditLimit: 100000, Balance: 0},
		{ID: 2, CreditLimit: 80000, Balance: 0},
		{ID: 3, CreditLimit: 1000000, Balance: 0},
		{ID: 4, CreditLimit: 10000000, Balance: 0},
		{ID: 5, CreditLimit: 500000, Balance: 0},
	}

	for _, c := range initialClients {
		err := clientsStore.Add(ctx, c)
		if err != nil {
			panic(err)
		}
	}

	actorManager := app.NewActorManager(clientsStore, transactionStore)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	r.POST("/clientes/:id/transacoes", app.MakeTransactionsHandler(actorManager))
	r.GET("/clientes/:id/extrato", app.MakeTransactionHistoryHandler(actorManager))

	port := os.Getenv("APP_PORT")

	log.Printf("server listening on :%s\n", port)
	if err := r.Run(fmt.Sprintf(":%s", port)); err != nil {
		fmt.Printf("failed to start server: %v\n", err)
	}
}
