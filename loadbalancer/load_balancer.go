package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/feralc/rinha-backend-2024/app"
	"github.com/feralc/rinha-backend-2024/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

var (
	targetBackends []proto.TransactionServiceClient
)

func main() {
	addresses := strings.Split(os.Getenv("APP_BACKENDS"), ",")
	port, _ := strconv.Atoi(os.Getenv("APP_PORT"))

	defer connectBackends(addresses)()

	http.HandleFunc("/clientes/{id}/transacoes", loadBalance(handleTransaction))
	http.HandleFunc("/clientes/{id}/extrato", loadBalance(handleHistory))

	fmt.Printf("load balancer listening on port %d...\n", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}

func connectBackends(addresses []string) func() {
	targetBackends = make([]proto.TransactionServiceClient, len(addresses))
	conns := make([]*grpc.ClientConn, len(addresses))

	for i, address := range addresses {
		conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			log.Fatalf("failed to connect to gRPC backend: %v", err)
		}

		conns[i] = conn
		grpcClient := proto.NewTransactionServiceClient(conn)
		targetBackends[i] = grpcClient
	}

	return func() {
		for _, c := range conns {
			c.Close()
		}
	}
}

func loadBalance(handler func(clientID int, backend proto.TransactionServiceClient) func(w http.ResponseWriter, r *http.Request)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		clientIDStr := r.PathValue("id")
		clientID, err := strconv.Atoi(clientIDStr)
		if err != nil {
			http.Error(w, "invalid client id", http.StatusUnprocessableEntity)
			return
		}

		targetIndex := clientID % len(targetBackends)
		targetBackend := targetBackends[targetIndex]

		handler(clientID, targetBackend)(w, r)
	}
}

func handleTransaction(clientID int, backend proto.TransactionServiceClient) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var req app.TransactionRequest

		if r.Method == http.MethodPost {
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "invalid request body", http.StatusUnprocessableEntity)
				return
			}

			if err := req.Validate(); err != nil {
				http.Error(w, err.Error(), http.StatusUnprocessableEntity)
				return
			}
		}

		var txType proto.TransactionType

		switch req.Type {
		case app.CreditTransaction:
			txType = proto.TransactionType_CREDIT_TRANSACTION
		case app.DebitTransaction:
			txType = proto.TransactionType_DEBIT_TRANSACTION
		}

		result, err := backend.DoTransaction(context.Background(), &proto.TransactionRequest{
			ClientID:    int32(clientID),
			Amount:      int32(req.Amount),
			Type:        txType,
			Description: req.Description,
		})

		if err != nil {
			if status.Code(err) == codes.NotFound {
				http.Error(w, "client not found", http.StatusNotFound)
				return
			}
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(app.SuccessTransactionResult{
			CreditLimit: int(result.CreditLimit),
			Balance:     int(result.Balance),
		})
	}
}

func handleHistory(clientID int, backend proto.TransactionServiceClient) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		result, err := backend.GetHistory(context.Background(), &proto.HistoryRequest{
			ClientID: int32(clientID),
		})

		if err != nil {
			if status.Code(err) == codes.NotFound {
				http.Error(w, "client not found", http.StatusNotFound)
				return
			}

			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}

		lastTransactions := make([]app.TransactionSummary, len(result.LastTransactions))

		for i, t := range result.LastTransactions {
			lastTransactions[i] = app.TransactionSummary{
				Amount:      int(t.Amount),
				Type:        app.TransactionType(t.Type),
				Description: t.Description,
				Timestamp:   time.Unix(t.Timestamp, 0),
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(app.TransactionHistory{
			Balance: app.TransactionHistoryBalance{
				CreditLimit: int(result.Balance.CreditLimit),
				Total:       int(result.Balance.Total),
				Date:        time.Unix(result.Balance.Date, 0),
			},
			LastTransactions: lastTransactions,
		})
	}
}
