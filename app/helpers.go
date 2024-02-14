package app

import (
	"context"
	"os"
	"strconv"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func BuildTransactionStore() (store TransactionStore, close func()) {
	if inMemory, _ := strconv.ParseBool(os.Getenv("APP_IN_MEMORY")); inMemory {
		return NewInMemoryTransactionStore(), close
	}

	clientOptions := options.Client().ApplyURI("mongodb://db:27017")
	client, err := mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		panic(err)
	}

	return NewMongoDBTransactionStore(client), func() { client.Disconnect(context.Background()) }
}
