package app

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

const (
	DatabaseName               = "rinha_backend"
	TransactionsCollectionName = "transactions"
)

type mongoDBTransactionStore struct {
	client       *mongo.Client
	transactions *mongo.Collection
}

func NewMongoDBTransactionStore(client *mongo.Client) TransactionStore {
	db := client.Database(DatabaseName)
	return &mongoDBTransactionStore{client, db.Collection(TransactionsCollectionName)}
}

func (s *mongoDBTransactionStore) Add(ctx context.Context, transaction Transaction) error {
	_, err := s.transactions.InsertOne(ctx, bson.M{
		"client_id":   transaction.ClientID,
		"amount":      transaction.Amount,
		"type":        transaction.Type,
		"description": transaction.Description,
		"created_at":  time.Now(),
	})
	return err
}

func (s *mongoDBTransactionStore) GetTransactionHistory(ctx context.Context, clientID int) ([]Transaction, error) {
	// @TODO implementar logica de snapshot para ler as ultimas transactions a partir de uma revision do snapshot
	// @TODO handle errors
	filter := bson.M{"client_id": clientID}
	cursor, err := s.transactions.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var transactions []Transaction
	for cursor.Next(ctx) {
		var t Transaction
		if err := cursor.Decode(&t); err != nil {
			return nil, err
		}
		transactions = append(transactions, t)
	}
	if err := cursor.Err(); err != nil {
		return nil, err
	}
	return transactions, nil
}
