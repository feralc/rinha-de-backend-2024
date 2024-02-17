package app

import (
	"context"
	"errors"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	DatabaseName               = "rinha_backend"
	TransactionsCollectionName = "transactions"
	SnapshotsCollectionName    = "snapshots"
	SnapshotSize               = 50
)

type mongoDBTransactionStore struct {
	client       *mongo.Client
	transactions *mongo.Collection
	snapshots    *mongo.Collection
}

func NewMongoDBTransactionStore(client *mongo.Client) TransactionStore {
	db := client.Database(DatabaseName)
	return &mongoDBTransactionStore{
		client:       client,
		transactions: db.Collection(TransactionsCollectionName),
		snapshots:    db.Collection(SnapshotsCollectionName),
	}
}

func (s *mongoDBTransactionStore) Add(ctx context.Context, currentBalance int, transaction Transaction) error {
	_, err := s.transactions.InsertOne(ctx, transaction)
	if err != nil {
		return err
	}

	if transaction.Revision%SnapshotSize == 0 {
		err = s.takeSnapshot(ctx, transaction.ClientID, transaction.Revision, currentBalance)
		if err != nil {
			log.Printf("error taking snapshot for client %d\n", transaction.ClientID)
		}
	}
	return nil
}

func (s *mongoDBTransactionStore) GetTransactionHistory(ctx context.Context, clientID int) (lastSnapshot Snapshot, transactions []Transaction, err error) {
	lastSnapshot, err = s.getLastSnapshot(ctx, clientID)

	if !lastSnapshot.ID.IsZero() {
		log.Printf("loading state of client %d from snapshot revision %d:\n", clientID, lastSnapshot.Revision)
	}

	if err != nil {
		return lastSnapshot, nil, err
	}

	filter := bson.M{
		"client_id": clientID,
		"revision":  bson.M{"$gte": lastSnapshot.Revision - HistorySize},
	}
	cursor, err := s.transactions.Find(ctx, filter)
	if err != nil {
		return lastSnapshot, nil, err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var t Transaction
		if err := cursor.Decode(&t); err != nil {
			return lastSnapshot, nil, err
		}
		transactions = append(transactions, t)
	}
	if err := cursor.Err(); err != nil {
		return lastSnapshot, nil, err
	}

	return lastSnapshot, transactions, nil
}

func (s *mongoDBTransactionStore) getLastSnapshot(ctx context.Context, clientID int) (lastSnapshot Snapshot, err error) {
	var snapshot Snapshot
	opts := options.FindOne().SetSort(bson.D{{Key: "created_at", Value: -1}})
	err = s.snapshots.FindOne(ctx, bson.D{{Key: "client_id", Value: clientID}}, opts).Decode(&snapshot)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return lastSnapshot, nil
		}

		return lastSnapshot, err
	}
	return snapshot, nil
}

func (s *mongoDBTransactionStore) takeSnapshot(ctx context.Context, clientID int, revision int, currentBalance int) error {
	snapshot := Snapshot{
		ClientID:  clientID,
		Revision:  revision,
		Balance:   currentBalance,
		CreatedAt: time.Now(),
	}
	_, err := s.snapshots.InsertOne(ctx, snapshot)
	return err
}
