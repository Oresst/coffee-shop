package service

import (
	"context"
	"fmt"
	"github.com/yourusername/order-service/internal/repository"

	"github.com/yourusername/order-service/internal/domain"
	"github.com/yourusername/order-service/pkg/logger"
	"go.uber.org/zap"
)

type OrderService struct {
	repo repository.OrderRepositoryInt
}

func NewOrderService(repo repository.OrderRepositoryInt) *OrderService {
	return &OrderService{repo: repo}
}

func (s *OrderService) CreateOrder(ctx context.Context, req *domain.CreateOrderRequest) (*domain.OrderResponse, error) {
	logger.Log.Debug("Creating order",
		logger.WithTraceID(ctx),
		zap.Int64("user_id", req.UserID),
	)

	// Рассчитываем общую сумму
	var total float64
	for _, item := range req.Items {
		total += item.Price * float64(item.Quantity)
	}

	order := &domain.Order{
		UserID: req.UserID,
		Items:  req.Items,
		Status: domain.StatusNew,
		Total:  total,
	}

	if err := s.repo.Create(ctx, order); err != nil {
		logger.Log.Error("Failed to create order",
			logger.WithTraceID(ctx),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	logger.Log.Info("Order created successfully",
		logger.WithTraceID(ctx),
		zap.Int64("order_id", order.ID),
		zap.Int64("user_id", order.UserID),
		zap.Float64("total", order.Total),
	)

	return &domain.OrderResponse{
		ID:     order.ID,
		UserID: order.UserID,
		Items:  order.Items,
		Status: order.Status,
		Total:  order.Total,
	}, nil
}

func (s *OrderService) GetOrder(ctx context.Context, id int64) (*domain.OrderResponse, error) {
	logger.Log.Debug("Getting order",
		logger.WithTraceID(ctx),
		zap.Int64("order_id", id),
	)

	order, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to find order: %w", err)
	}
	if order == nil {
		return nil, fmt.Errorf("order not found: %d", id)
	}

	return &domain.OrderResponse{
		ID:     order.ID,
		UserID: order.UserID,
		Items:  order.Items,
		Status: order.Status,
		Total:  order.Total,
	}, nil
}

func (s *OrderService) GetUserOrders(ctx context.Context, userID int64) ([]domain.OrderResponse, error) {
	logger.Log.Debug("Getting user orders",
		logger.WithTraceID(ctx),
		zap.Int64("user_id", userID),
	)

	orders, err := s.repo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to find orders: %w", err)
	}

	responses := make([]domain.OrderResponse, len(orders))
	for i, order := range orders {
		responses[i] = domain.OrderResponse{
			ID:     order.ID,
			UserID: order.UserID,
			Items:  order.Items,
			Status: order.Status,
			Total:  order.Total,
		}
	}

	return responses, nil
}
