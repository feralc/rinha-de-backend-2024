package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
)

// TransactionRequest represents a transaction request.
type TransactionRequest struct {
	Amount    int    `json:"valor"`
	Type      string `json:"tipo"`
	Descricao string `json:"descricao"`
}

// TransactionResponse represents a transaction response.
type TransactionResponse struct {
	Limite int `json:"limite"`
	Saldo  int `json:"saldo"`
}

// ClientActor represents an actor responsible for processing transactions for a specific client.
type ClientActor struct {
	clientID  int
	balance   int
	limit     int
	inbox     chan *TransactionRequest
	responses chan *TransactionResponse
	db        *sql.DB
}

// NewClientActor creates a new actor for a specific client.
func NewClientActor(clientID, limit int, responses chan *TransactionResponse, db *sql.DB) *ClientActor {
	actor := &ClientActor{
		clientID:  clientID,
		balance:   0,
		limit:     limit,
		inbox:     make(chan *TransactionRequest),
		responses: responses,
		db:        db,
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

	err := InsertTransaction(a.db, a.clientID, req.Amount, req.Type, req.Descricao)
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
func NewClientManager() *ClientManager {
	return &ClientManager{
		clients:   make(map[int]*ClientActor),
		responses: make(chan *TransactionResponse),
	}
}

// GetOrCreateClientActor returns the actor for the specified client ID, creating a new one if it doesn't exist.
func (cm *ClientManager) GetOrCreateClientActor(clientID, limit int, db *sql.DB) *ClientActor {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	actor, ok := cm.clients[clientID]
	if !ok {
		actor = NewClientActor(clientID, limit, cm.responses, db)
		cm.clients[clientID] = actor
	}
	return actor
}

func main() {

	// Open SQLite database
	db, err := sql.Open("sqlite3", "transactions.db")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// Create transactions table
	_, err = db.Exec(`
		 CREATE TABLE IF NOT EXISTS transactions (
			 id INTEGER PRIMARY KEY AUTOINCREMENT,
			 client_id INTEGER,
			 amount INTEGER,
			 type TEXT,
			 description TEXT,
			 created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		 )
	 `)
	if err != nil {
		panic(err)
	}

	clientManager := NewClientManager()

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
		actor := clientManager.GetOrCreateClientActor(clientID, 100000, db)
		response := actor.Send(&req)

		// Send response
		c.JSON(http.StatusOK, response)
	})

	r.GET("/clientes/:id/extrato", func(c *gin.Context) {
		// Parse client ID from URL
		clientIDStr := c.Param("id")
		clientID, err := strconv.Atoi(clientIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid client ID"})
			return
		}

		// Fetch transaction history from database
		transactions, err := GetTransactionHistory(db, clientID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch transaction history"})
			return
		}

		// Calculate total balance
		totalBalance := calculateTotalBalance(transactions)

		// Prepare response
		response := gin.H{
			"saldo": gin.H{
				"total":        totalBalance,
				"data_extrato": time.Now().Format(time.RFC3339),
				"limite":       100000, // Assuming this is the client's limit
			},
			"ultimas_transacoes": transactions,
		}

		// Send response
		c.JSON(http.StatusOK, response)
	})

	// Start HTTP server
	if err := r.Run(":8080"); err != nil {
		fmt.Printf("Failed to start server: %v\n", err)
	}
}

// Transaction represents a transaction record.
type Transaction struct {
	Amount      int    `json:"valor"`
	Type        string `json:"tipo"`
	Description string `json:"descricao"`
	Timestamp   string `json:"realizada_em"`
}

// InsertTransaction inserts a transaction record into the database.
func InsertTransaction(db *sql.DB, clientID, amount int, transactionType, description string) error {
	_, err := db.Exec(`
        INSERT INTO transactions (client_id, amount, type, description)
        VALUES (?, ?, ?, ?)
    `, clientID, amount, transactionType, description)
	return err
}

// GetTransactionHistory fetches the transaction history for a specific client from the database.
func GetTransactionHistory(db *sql.DB, clientID int) ([]Transaction, error) {
	rows, err := db.Query(`
        SELECT amount, type, description, created_at
        FROM transactions
        WHERE client_id = ?
    `, clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transactions []Transaction
	for rows.Next() {
		var t Transaction
		err := rows.Scan(&t.Amount, &t.Type, &t.Description, &t.Timestamp)
		if err != nil {
			return nil, err
		}
		transactions = append(transactions, t)
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
