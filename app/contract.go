package app

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

var ErrNotFound = fmt.Errorf("not found")

type Snapshot struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	ClientID  int                `bson:"client_id"`
	Revision  int                `bson:"revision"`
	Balance   int                `bson:"balance"`
	CreatedAt time.Time          `bson:"created_at"`
}

type TransactionStore interface {
	Add(ctx context.Context, currentBalance int, transaction Transaction) error
	GetTransactionHistory(ctx context.Context, clientID int) (lastSnapshot Snapshot, transactions []Transaction, err error)
}

type ClientStore interface {
	Add(ctx context.Context, client Client) error
	GetOne(ctx context.Context, clientId int) (client Client, err error)
}
