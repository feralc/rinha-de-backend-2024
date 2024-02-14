package app

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

const (
	ClientsCollectionName = "clients"
)

type mongoDBClientStore struct {
	client    *mongo.Client
	clients   *mongo.Collection
	snapshots *mongo.Collection
}

func NewMongoDBClientStore(client *mongo.Client) ClientStore {
	db := client.Database(DatabaseName)
	return &mongoDBClientStore{
		client:    client,
		clients:   db.Collection(ClientsCollectionName),
		snapshots: db.Collection(SnapshotsCollectionName),
	}
}

func (s *mongoDBClientStore) Add(ctx context.Context, client Client) error {
	filter := bson.M{"ID": client.ID}
	count, err := s.clients.CountDocuments(ctx, filter)
	if err != nil {
		return err
	}
	if count == 0 {
		_, err := s.clients.InsertOne(ctx, client)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *mongoDBClientStore) GetOne(ctx context.Context, clientID int) (client Client, err error) {
	filter := bson.M{"client_id": clientID}
	err = s.clients.FindOne(ctx, filter).Decode(&client)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return client, NotFoundErr
		}

		return client, err
	}
	return client, nil
}
