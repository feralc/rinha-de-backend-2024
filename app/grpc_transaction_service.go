package app

import (
	"context"
	"errors"

	"github.com/feralc/rinha-backend-2024/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type TransactionService struct {
	*proto.UnimplementedTransactionServiceServer
	actorManager *ActorManager
}

func NewTransactionService(actorManager *ActorManager) *TransactionService {
	return &TransactionService{actorManager: actorManager}
}

func (s *TransactionService) DoTransaction(ctx context.Context, req *proto.TransactionRequest) (*proto.TransactionResult, error) {
	actor, err := s.actorManager.Spawn(int(req.ClientID))

	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, err
	}

	var txType TransactionType

	switch req.Type {
	case proto.TransactionType_CREDIT_TRANSACTION:
		txType = CreditTransaction
	case proto.TransactionType_DEBIT_TRANSACTION:
		txType = DebitTransaction
	}

	result := actor.Send(ActorMessage{
		Type: TransactionMessage,
		Payload: TransactionRequest{
			Amount:      int(req.Amount),
			Type:        txType,
			Description: req.Description,
		},
	})

	if result.Error != nil {
		return nil, result.Error
	}

	data := result.Data.(SuccessTransactionResult)

	return &proto.TransactionResult{
		CreditLimit: int32(data.CreditLimit),
		Balance:     int32(data.Balance),
	}, nil
}

func (s *TransactionService) GetHistory(ctx context.Context, req *proto.HistoryRequest) (*proto.AccountStatement, error) {
	actor, err := s.actorManager.Spawn(int(req.ClientID))

	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, err
	}

	result := actor.Send(ActorMessage{
		Type: QueryHistoryMessage,
	})

	if result.Error != nil {
		return nil, result.Error
	}

	data := result.Data.(*TransactionHistory)

	lastTransactions := make([]*proto.Transaction, len(data.LastTransactions))

	for i, t := range data.LastTransactions {
		lastTransactions[i] = &proto.Transaction{
			Amount:      int32(t.Amount),
			Type:        string(t.Type),
			Description: t.Description,
			Timestamp:   t.Timestamp.Unix(),
		}
	}

	return &proto.AccountStatement{
		Balance: &proto.Balance{
			CreditLimit: int32(data.Balance.CreditLimit),
			Total:       int32(data.Balance.Total),
			Date:        data.Balance.Date.Unix(),
		},
		LastTransactions: lastTransactions,
	}, nil
}
