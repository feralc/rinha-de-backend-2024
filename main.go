package main

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// TransactionRequest represents a transaction request.
type TransactionRequest struct {
	Amount    int    `json:"valor"`
	Type      string `json:"tipo"`
	Descricao string `json:"descricao"`
}

// TransactionResponse represents a transaction response.
type TransactionResponse struct {
	Limite           int           `json:"limite"`
	Saldo            int           `json:"saldo"`
	LastTransactions []Transaction `json:"ultimas_transacoes"`
}

// Transaction represents a transaction record.
type Transaction struct {
	Amount      int       `json:"valor"`
	Type        string    `json:"tipo"`
	Description string    `json:"descricao"`
	Timestamp   time.Time `json:"realizada_em"`
}

// ClientActor represents an actor responsible for processing transactions for a specific client.
type ClientActor struct {
	clientID  int
	balance   int
	limit     int
	inbox     chan *TransactionRequest
	responses chan *TransactionResponse
	coll      *mongo.Collection
}

// NewClientActor creates a new actor for a specific client.
func NewClientActor(clientID, limit int, responses chan *TransactionResponse, coll *mongo.Collection) *ClientActor {
	actor := &ClientActor{
		clientID:  clientID,
		balance:   0,
		limit:     limit,
		inbox:     make(chan *TransactionRequest),
		responses: responses,
		coll:      coll,
	}

	go actor.start()

	return actor
}

// start starts processing messages in the actor's inbox.
func (a *ClientActor) start() {
	for msg := range a.inbox {
		// Process transaction
		response := a.processTransaction(msg)

		// Send response
		a.responses <- response
	}
}

// processTransaction processes a transaction request.
func (a *ClientActor) processTransaction(req *TransactionRequest) *TransactionResponse {
	// Check transaction type and amount
	switch req.Type {
	case "credit":
		a.balance += req.Amount
	case "debit":
		if a.balance-req.Amount < -a.limit {
			return &TransactionResponse{
				Limite: a.limit,
				Saldo:  a.balance,
			}
		}
		a.balance -= req.Amount
	default:
		fmt.Println("Invalid transaction type")
	}

	// Insert transaction into MongoDB
	err := InsertTransaction(a.coll, a.clientID, req.Amount, req.Type, req.Descricao)
	if err != nil {
		panic(err)
	}

	// Return transaction response
	return &TransactionResponse{
		Limite: a.limit,
		Saldo:  a.balance,
	}
}

// Send sends a transaction request to the actor.
func (a *ClientActor) Send(req *TransactionRequest) *TransactionResponse {
	a.inbox <- req
	return <-a.responses
}

// ClientManager manages actors for different clients.
type ClientManager struct {
	clients   map[int]*ClientActor
	mutex     sync.Mutex
	responses chan *TransactionResponse
}

// NewClientManager creates a new client manager.
func NewClientManager(responses chan *TransactionResponse) *ClientManager {
	return &ClientManager{
		clients:   make(map[int]*ClientActor),
		responses: responses,
	}
}

// GetOrCreateClientActor returns the actor for the specified client ID, creating a new one if it doesn't exist.
func (cm *ClientManager) GetOrCreateClientActor(clientID, limit int, coll *mongo.Collection) *ClientActor {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	actor, ok := cm.clients[clientID]
	if !ok {
		actor = NewClientActor(clientID, limit, cm.responses, coll)
		cm.clients[clientID] = actor
	}
	return actor
}

func main() {
	// MongoDB connection URI
	clientOptions := options.Client().ApplyURI("mongodb://db:27017")
	client, err := mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		panic(err)
	}
	defer client.Disconnect(context.Background())

	// Select the database and collection
	db := client.Database("mydatabase")
	transactionsColl := db.Collection("transactions")

	clientManager := NewClientManager(make(chan *TransactionResponse))

	// Create a Gin router
	r := gin.Default()

	// Define route handler for transaction requests
	r.POST("/clientes/:id/transacoes", func(c *gin.Context) {
		// Parse client ID from URL
		clientIDStr := c.Param("id")
		clientID, err := strconv.Atoi(clientIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid client ID"})
			return
		}

		// Parse transaction request from request body
		var req TransactionRequest
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
			return
		}

		// Send transaction request to the client's actor
		actor := clientManager.GetOrCreateClientActor(clientID, 100000, transactionsColl)
		response := actor.Send(&req)

		// Send response
		c.JSON(http.StatusOK, response)
	})

	// Define route handler for transaction history requests
	r.GET("/clientes/:id/extrato", func(c *gin.Context) {
		// Parse client ID from URL
		clientIDStr := c.Param("id")
		clientID, err := strconv.Atoi(clientIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid client ID"})
			return
		}

		// Fetch transaction history from MongoDB
		transactions, err := GetTransactionHistory(transactionsColl, clientID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch transaction history"})
			return
		}

		// Calculate total balance
		totalBalance := calculateTotalBalance(transactions)

		// Prepare response
		response := TransactionResponse{
			Limite:           100000, // Assuming this is the client's limit
			Saldo:            totalBalance,
			LastTransactions: transactions,
		}

		// Send response
		c.JSON(http.StatusOK, response)
	})

	// Start HTTP server
	if err := r.Run(":8080"); err != nil {
		fmt.Printf("Failed to start server: %v\n", err)
	}
}

// InsertTransaction inserts a transaction record into the MongoDB collection.
func InsertTransaction(coll *mongo.Collection, clientID, amount int, transactionType, description string) error {
	_, err := coll.InsertOne(context.Background(), bson.M{
		"client_id":   clientID,
		"amount":      amount,
		"type":        transactionType,
		"description": description,
		"created_at":  time.Now(),
	})
	return err
}

// GetTransactionHistory fetches the transaction history for a specific client from the MongoDB collection.
func GetTransactionHistory(coll *mongo.Collection, clientID int) ([]Transaction, error) {
	findOptions := options.Find()
	findOptions.SetSort(bson.D{{"created_at", -1}})
	filter := bson.D{{"client_id", clientID}}
	cur, err := coll.Find(context.Background(), filter, findOptions)
	if err != nil {
		return nil, err
	}
	defer cur.Close(context.Background())

	var transactions []Transaction
	for cur.Next(context.Background()) {
		var t Transaction
		err := cur.Decode(&t)
		if err != nil {
			return nil, err
		}
		transactions = append(transactions, t)
	}
	if err := cur.Err(); err != nil {
		return nil, err
	}
	return transactions, nil
}

func calculateTotalBalance(transactions []Transaction) int {
	var totalBalance int
	for _, t := range transactions {
		if t.Type == "credit" {
			totalBalance += t.Amount
		} else if t.Type == "debit" {
			totalBalance -= t.Amount
		}
	}
	return totalBalance
}
