package main

import (
	"context"
	"log"
	"net"
	"os"
	"strconv"

	"github.com/feralc/rinha-backend-2024/app"
	"github.com/feralc/rinha-backend-2024/proto"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"
)

func main() {
	ctx := context.Background()

	mongoClient := setupMongoDB(ctx)
	defer mongoClient.Disconnect(context.Background())

	initializeMongoDB(ctx, mongoClient)

	transactionStore := app.NewMongoDBTransactionStore(mongoClient)
	clientsStore := app.NewMongoDBClientStore(mongoClient)

	addInitialClients(ctx, clientsStore)

	actorManager := app.NewActorManager(clientsStore, transactionStore)

	grpcServer := grpc.NewServer()

	proto.RegisterTransactionServiceServer(grpcServer, app.NewTransactionService(actorManager))

	port := os.Getenv("APP_PORT")
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	log.Printf("gRPC server listening on :%s\n", port)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func setupMongoDB(ctx context.Context) *mongo.Client {
	clientOptions := options.Client().
		ApplyURI("mongodb://127.0.0.1:27017").
		SetMinPoolSize(25).
		SetMaxPoolSize(100)

	mongoClient, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatalf("failed to connect to MongoDB: %v\n", err)
	}
	return mongoClient
}

func initializeMongoDB(ctx context.Context, client *mongo.Client) {
	db := client.Database(app.DatabaseName)

	if shouldDropDB, _ := strconv.ParseBool(os.Getenv("DROP_DB_ON_START")); shouldDropDB {
		log.Printf("dropping database %s\n", app.DatabaseName)
		if err := db.Drop(ctx); err != nil {
			log.Fatalf("failed to drop database: %v\n", err)
		}
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

	db.Collection(app.SnapshotsCollectionName).Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.M{"created_at": 1},
			Options: options.Index(),
		},
		{
			Keys:    bson.M{"client_id": 1},
			Options: options.Index(),
		},
	})
}

func addInitialClients(ctx context.Context, clientsStore app.ClientStore) {
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
			log.Fatalf("failed to add initial client: %v\n", err)
		}
	}
}
