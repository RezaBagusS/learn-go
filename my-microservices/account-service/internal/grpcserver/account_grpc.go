package grpcserver

import (
	"context"
	"errors"

	"go.uber.org/zap"

	"my-microservices/account-service/internal/domain"
	"my-microservices/account-service/internal/repository"
	pb "my-microservices/shared/pb/account"
)

type accountGRPCServer struct {
	pb.UnimplementedAccountGRPCServiceServer
	repo repository.AccountRepository
	log  *zap.Logger
}

func NewAccountGRPCServer(repo repository.AccountRepository, log *zap.Logger) pb.AccountGRPCServiceServer {
	return &accountGRPCServer{repo: repo, log: log}
}

func (s *accountGRPCServer) ExecuteTransferMutation(ctx context.Context, req *pb.TransferMutationRequest) (*pb.TransferMutationResponse, error) {
	err := s.repo.ProcessTransferMutation(ctx, req.SourceAccount, req.BeneficiaryAccount, req.Amount)

	if err != nil {
		s.log.Error("Transfer mutation failed", zap.Error(err))

		// Mapping error dari repository ke response gRPC
		if errors.Is(err, domain.ErrLogicBalanceTrx) {
			return &pb.TransferMutationResponse{
				Success:      false,
				ErrorCode:    "ERR_INSUFFICIENT_FUNDS",
				ErrorMessage: "Saldo pengirim tidak mencukupi",
			}, nil
		}

		if errors.Is(err, domain.ErrIdNotFound) {
			return &pb.TransferMutationResponse{
				Success:      false,
				ErrorCode:    "ERR_INVALID_ACCOUNT",
				ErrorMessage: "Akun pengirim atau penerima tidak valid",
			}, nil
		}

		// Error sistem/database lainnya
		return &pb.TransferMutationResponse{
			Success:      false,
			ErrorCode:    "ERR_SYSTEM",
			ErrorMessage: "Terjadi kesalahan internal pada sistem",
		}, nil
	}

	// Sukses
	return &pb.TransferMutationResponse{
		Success: true,
	}, nil
}
