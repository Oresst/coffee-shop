package services

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/yourusername/saga-service/internal/config"
	"github.com/yourusername/saga-service/internal/domains"
	"github.com/yourusername/saga-service/internal/repositories/postgres"
	"github.com/yourusername/saga-service/internal/services/client"
	"github.com/yourusername/saga-service/pkg/logger"
	"go.uber.org/zap"
)

type OrderSagaService struct {
	repo            postgres.CreateOrderSagaRepoInt
	inventoryClient *client.InventoryClient
	orderClient     *client.OrderClient
}

func NewOrderSagaService(repo postgres.CreateOrderSagaRepoInt, cfg *config.Config) *OrderSagaService {
	return &OrderSagaService{
		repo:            repo,
		inventoryClient: client.NewInventoryClient(cfg),
		orderClient:     client.NewOrderClient(cfg),
	}
}

func (s *OrderSagaService) StartSaga(ctx context.Context, userID int, items []*domains.Item) (*domains.CreateOrderSagaResponse, error) {
	place := "[OrderSagaService.StartSaga]"
	requestID := uuid.New().String()

	saga := domains.OrderSaga{
		UserID:    userID,
		Items:     items,
		RequestID: requestID,
		Cancelled: false,
		Status:    domains.StatusCreated,
	}

	err := s.repo.CreateSaga(ctx, &saga)
	if err != nil {
		logger.Log.Error(fmt.Sprintf("%s Ошибка создания саги", place),
			zap.String("request_id", requestID),
			zap.Int("user_id", userID),
			zap.Error(err),
		)
		return nil, err
	}

	logger.Log.Info(fmt.Sprintf("%s Сага успешно создана", place),
		zap.String("request_id", requestID),
		zap.Int("user_id", userID),
		zap.Int("saga_id", saga.ID),
	)

	ctx = context.WithValue(ctx, "saga", saga)

	go s.Continue(ctx)

	return &domains.CreateOrderSagaResponse{
		RequestID: requestID,
	}, nil
}

func (s *OrderSagaService) Continue(ctx context.Context) {
	saga, ok := ctx.Value("saga").(domains.OrderSaga)
	if !ok {
		return
	}

	switch saga.Status {
	case domains.StatusCreated, domains.StatusReserveStarted:
		s.Reserve(ctx, &saga)
	case domains.StatusReservedFinished:
		s.CreateOrder(ctx, &saga)
	}
}

func (s *OrderSagaService) Reserve(ctx context.Context, saga *domains.OrderSaga) {
	place := "[OrderSagaService.Reserve]"

	request := &domains.ReserveRequest{
		RequestID: saga.RequestID,
		Items:     saga.Items,
	}
	var err error

	logger.Log.Info(fmt.Sprintf("%s Начало шага резервации товара", place),
		zap.String("request_id", saga.RequestID),
		zap.Int("user_id", saga.UserID),
		zap.Int("saga_id", saga.ID),
	)

	err = s.repo.ChangeStatus(ctx, saga, domains.StatusReserveStarted)
	if err != nil {
		logger.Log.Error(fmt.Sprintf("%s ошибка изменения статуса саги", place),
			zap.String("request_id", saga.RequestID),
			zap.Int("saga_id", saga.ID),
			zap.Error(err),
		)
		return
	}

	result, err := s.inventoryClient.Reserve(request)
	if err != nil {
		logger.Log.Error(fmt.Sprintf("%s ошибка резервации товаров", place),
			zap.String("request_id", saga.RequestID),
			zap.Int("saga_id", saga.ID),
			zap.Error(err),
		)
		return
	}

	err = s.repo.ChangeItems(ctx, saga.ID, result.Items)
	if err != nil {
		logger.Log.Error(fmt.Sprintf("%s ошибка изменения поля items у saga", place),
			zap.String("request_id", saga.RequestID),
			zap.Int("saga_id", saga.ID),
			zap.Error(err),
		)
		return
	}

	saga.Items = result.Items
	err = s.repo.ChangeStatus(ctx, saga, domains.StatusReservedFinished)
	if err != nil {
		logger.Log.Error(fmt.Sprintf("%s ошибка изменения статуса саги", place),
			zap.String("request_id", saga.RequestID),
			zap.Int("saga_id", saga.ID),
			zap.Error(err),
		)
		return
	}

	ctx = context.WithValue(ctx, "saga", *saga)

	logger.Log.Info(fmt.Sprintf("%s Товар успешно забронирован", place),
		zap.String("request_id", saga.RequestID),
		zap.Int("user_id", saga.UserID),
		zap.Int("saga_id", saga.ID),
	)

	go s.Continue(ctx)
}

func (s *OrderSagaService) CreateOrder(ctx context.Context, saga *domains.OrderSaga) {
	place := "[OrderSagaService.CreateOrder]"

	request := domains.CreateOrderRequest{
		Items:     saga.Items,
		UserId:    saga.UserID,
		RequestId: saga.RequestID,
	}

	logger.Log.Info(fmt.Sprintf("%s Начало шага создания заказа", place),
		zap.String("request_id", saga.RequestID),
		zap.Int("user_id", saga.UserID),
		zap.Int("saga_id", saga.ID),
	)

	err := s.repo.ChangeStatus(ctx, saga, domains.StatusCreateOrderStarted)
	if err != nil {
		logger.Log.Error(fmt.Sprintf("%s ошибка изменения статуса саги", place),
			zap.String("request_id", saga.RequestID),
			zap.Int("saga_id", saga.ID),
			zap.Error(err),
		)
		return
	}

	result, err := s.orderClient.CreateOrder(&request)
	if err != nil {
		logger.Log.Error(fmt.Sprintf("%s Ошибка вызова сервиса создания заказа", place),
			zap.String("request_id", saga.RequestID),
			zap.Int("user_id", saga.UserID),
			zap.Int("saga_id", saga.ID),
			zap.Error(err),
		)
		return
	}

	err = s.repo.ChangeOrderId(ctx, saga.ID, result.OrderID)
	if err != nil {
		logger.Log.Error(fmt.Sprintf("%s Ошибка присвояния id заказа для саги", place),
			zap.String("request_id", saga.RequestID),
			zap.Int("user_id", saga.UserID),
			zap.Int("saga_id", saga.ID),
			zap.Error(err),
		)
		return
	}

	err = s.repo.ChangeStatus(ctx, saga, domains.StatusCreateOrderFinished)
	if err != nil {
		logger.Log.Error(fmt.Sprintf("%s ошибка изменения статуса саги", place),
			zap.String("request_id", saga.RequestID),
			zap.Int("saga_id", saga.ID),
			zap.Error(err),
		)
		return
	}

	saga.OrderID = result.OrderID
	ctx = context.WithValue(ctx, "saga", *saga)

	logger.Log.Info(fmt.Sprintf("%s Создание заказа успешно завершено", place),
		zap.String("request_id", saga.RequestID),
		zap.Int("user_id", saga.UserID),
		zap.Int("saga_id", saga.ID),
	)

	go s.Continue(ctx)
}
