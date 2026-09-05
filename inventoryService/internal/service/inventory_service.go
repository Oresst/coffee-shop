package service

import (
	"context"
	"fmt"
	"github.com/yourusername/inventory-service/internal/domain"
	"github.com/yourusername/inventory-service/internal/repository/postgres"
	"github.com/yourusername/inventory-service/pkg/logger"
	"go.uber.org/zap"
)

type InventoryService struct {
	repo *postgres.InventoryRepository
}

func NewInventoryService(repo *postgres.InventoryRepository) *InventoryService {
	return &InventoryService{repo: repo}
}

func (s *InventoryService) CheckAvailability(ctx context.Context, items []domain.OrderItemRequest) (bool, error) {
	logger.Log.Debug("Checking availability",
		logger.WithTraceID(ctx),
		zap.Int("items_count", len(items)),
	)

	available, err := s.repo.CheckAvailability(ctx, items)
	if err != nil {
		logger.Log.Error("Availability check failed",
			logger.WithTraceID(ctx),
			zap.Error(err),
		)
		return false, err
	}

	if !available {
		logger.Log.Warn("Not enough inventory",
			logger.WithTraceID(ctx),
		)
		return false, nil
	}

	logger.Log.Info("Inventory available",
		logger.WithTraceID(ctx),
		zap.Int("items_count", len(items)),
	)
	return true, nil
}

func (s *InventoryService) ReserveItems(ctx context.Context, req *domain.ReserveRequest) (*domain.ReserveResponse, error) {
	place := "[InventoryService.ReserveItems]"

	logger.Log.Info(fmt.Sprintf("%s резервирование товаров", place),
		logger.WithTraceID(ctx),
		zap.String("request_id", req.RequestID),
		zap.Int64("user_id", req.UserId),
	)

	// Проверяем, не было ли уже резервации для этого request_id (идемпотентность)
	existingStatus, err := s.repo.GetReservationStatus(ctx, req.RequestID)
	if err != nil {
		return nil, err
	}
	if existingStatus == "confirmed" {
		logger.Log.Info(fmt.Sprintf("%s Резерв уже подтвержден", place),
			logger.WithTraceID(ctx),
			zap.String("request_id", req.RequestID),
			zap.Int64("user_id", req.UserId),
		)

		items, err := s.repo.GetReservedItems(ctx, req.RequestID)
		if err != nil {
			logger.Log.Error(fmt.Sprintf("%s Ошибка получения зарезервированных товаров", place),
				logger.WithTraceID(ctx),
				zap.Error(err),
				zap.Int64("user_id", req.UserId),
				zap.String("request_id", req.RequestID),
			)

			return nil, err
		}

		return &domain.ReserveResponse{
			Success:   true,
			RequestID: req.RequestID,
			OrderID:   req.OrderID,
			Message:   "already confirmed",
			Items:     items,
		}, nil
	}

	if existingStatus == "pending" {
		// Резервация уже существует, продолжаем
		logger.Log.Debug(fmt.Sprintf("%s Товары уже зарезервированы", place),
			logger.WithTraceID(ctx),
			zap.String("request_id", req.RequestID),
			zap.Int64("user_id", req.UserId),
		)

		items, err := s.repo.GetReservedItems(ctx, req.RequestID)
		if err != nil {
			logger.Log.Error(fmt.Sprintf("%s Ошибка получения зарезервированных товаров", place),
				logger.WithTraceID(ctx),
				zap.Error(err),
				zap.Int64("user_id", req.UserId),
				zap.String("request_id", req.RequestID),
			)

			return nil, err
		}

		return &domain.ReserveResponse{
			Success:   true,
			RequestID: req.RequestID,
			Message:   "already reserved",
			Items:     items,
		}, nil
	}

	// Выполняем резервацию
	if err := s.repo.ReserveItems(ctx, req.RequestID, req.OrderID, req.Items); err != nil {
		logger.Log.Error(fmt.Sprintf("%s Ошибка резервации товаров", place),
			logger.WithTraceID(ctx),
			zap.Error(err),
			zap.String("request_id", req.RequestID),
			zap.Int64("user_id", req.UserId),
		)
		return &domain.ReserveResponse{
			Success:   false,
			RequestID: req.RequestID,
			Message:   err.Error(),
		}, nil
	}

	items, err := s.repo.GetReservedItems(ctx, req.RequestID)
	if err != nil {
		logger.Log.Error(fmt.Sprintf("%s Ошибка получения зарезервированных товаров", place),
			logger.WithTraceID(ctx),
			zap.Error(err),
			zap.Int64("user_id", req.UserId),
			zap.String("request_id", req.RequestID),
		)

		return nil, err
	}

	logger.Log.Info(fmt.Sprintf("%s Товары успешно зарезервированы", place),
		logger.WithTraceID(ctx),
		zap.String("request_id", req.RequestID),
		zap.Int64("user_id", req.UserId),
	)

	return &domain.ReserveResponse{
		Success:   true,
		RequestID: req.RequestID,
		OrderID:   req.OrderID,
		Message:   "reserved",
		Items:     items,
	}, nil
}

func (s *InventoryService) CancelReservation(ctx context.Context, requestID string) error {
	logger.Log.Info("Cancelling reservation",
		logger.WithTraceID(ctx),
		zap.String("request_id", requestID),
	)

	if err := s.repo.CancelReservation(ctx, requestID); err != nil {
		logger.Log.Error("Failed to cancel reservation",
			logger.WithTraceID(ctx),
			zap.Error(err),
			zap.String("request_id", requestID),
		)
		return err
	}

	logger.Log.Info("Reservation cancelled",
		logger.WithTraceID(ctx),
		zap.String("request_id", requestID),
	)
	return nil
}

func (s *InventoryService) ConfirmReservation(ctx context.Context, requestID string) error {
	logger.Log.Info("Confirming reservation",
		logger.WithTraceID(ctx),
		zap.String("request_id", requestID),
	)

	if err := s.repo.ConfirmReservation(ctx, requestID); err != nil {
		logger.Log.Error("Failed to confirm reservation",
			logger.WithTraceID(ctx),
			zap.Error(err),
			zap.String("request_id", requestID),
		)
		return err
	}

	logger.Log.Info("Reservation confirmed",
		logger.WithTraceID(ctx),
		zap.String("request_id", requestID),
	)
	return nil
}
