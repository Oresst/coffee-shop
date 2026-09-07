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
	place := "[OrderSaga.Continue]"

	saga, ok := ctx.Value("saga").(domains.OrderSaga)
	if !ok {
		return
	}

	switch saga.Status {
	case domains.StatusCreated, domains.StatusReserveStarted, domains.StatusReservedFailed:
		s.Reserve(ctx, &saga)
	case domains.StatusReservedFinished, domains.StatusCreateOrderStarted, domains.StatusCreateOrderFailed:
		s.CreateOrder(ctx, &saga)
	case domains.StatusCreateOrderFinished, domains.StatusConfirmReservationStarted, domains.StatusConfirmReservationFailed:
		s.ConfirmReservation(ctx, &saga)
	case domains.StatusConfirmReservationFinished:
		err := s.repo.ChangeStatus(ctx, &saga, domains.StatusCompleted)
		if err != nil {
			logger.Log.Error(fmt.Sprintf("%s ошибка изменения статуса саги", place),
				zap.String("request_id", saga.RequestID),
				zap.Int("saga_id", saga.ID),
				zap.Error(err),
			)
		}
	}
}

func (s *OrderSagaService) ContinueCancel(ctx context.Context) {
	place := "[OrderSaga.ContinueCancel]"

	saga, ok := ctx.Value("saga").(domains.OrderSaga)
	if !ok {
		return
	}

	switch saga.Status {
	case domains.StatusReserveStarted, domains.StatusReservedFailed, domains.StatusReservedFinished, domains.StatusCreateOrderCancelled:
		s.CancelReserve(ctx, &saga)
	case domains.StatusCreateOrderStarted, domains.StatusCreateOrderFailed, domains.StatusCreateOrderFinished, domains.StatusConfirmReservationCancelled:
		s.CancelCreateOrder(ctx, &saga)
	case domains.StatusConfirmReservationStarted, domains.StatusConfirmReservationFailed:
		s.CancelConfirmReservation(ctx, &saga)
	case domains.StatusReservedCancelled:
		err := s.repo.ChangeStatus(ctx, &saga, domains.StatusCancelled)
		if err != nil {
			logger.Log.Error(fmt.Sprintf("%s ошибка изменения статуса саги", place),
				zap.String("request_id", saga.RequestID),
				zap.Int("saga_id", saga.ID),
				zap.Error(err),
			)
		}
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

		err = s.repo.CancelSaga(ctx, saga, domains.StatusReservedFailed)
		if err != nil {
			logger.Log.Error(fmt.Sprintf("%s ошибка отмены саги", place),
				zap.String("request_id", saga.RequestID),
				zap.Int("saga_id", saga.ID),
				zap.Error(err),
			)
		}
		ctx = context.WithValue(ctx, "saga", *saga)
		go s.ContinueCancel(ctx)
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

func (s *OrderSagaService) CancelReserve(ctx context.Context, saga *domains.OrderSaga) {
	place := "[OrderSagaService.CancelReserve]"

	request := &domains.CancelReserveRequest{
		RequestID: saga.RequestID,
	}

	logger.Log.Info(fmt.Sprintf("%s Начало шага отмены резерва товаров", place),
		zap.String("request_id", saga.RequestID),
		zap.Int("user_id", saga.UserID),
		zap.Int("saga_id", saga.ID),
	)

	err := s.inventoryClient.Cancel(request)
	if err != nil {
		logger.Log.Error(fmt.Sprintf("%s Ошибка отмены резерва товаров", place),
			zap.String("request_id", saga.RequestID),
			zap.Int("user_id", saga.UserID),
			zap.Int("saga_id", saga.ID),
			zap.Error(err),
		)
		return
	}

	err = s.repo.ChangeStatus(ctx, saga, domains.StatusReservedCancelled)
	if err != nil {
		logger.Log.Error(fmt.Sprintf("%s ошибка изменения статуса саги", place),
			zap.String("request_id", saga.RequestID),
			zap.Int("saga_id", saga.ID),
			zap.Error(err),
		)
		return
	}

	ctx = context.WithValue(ctx, "saga", *saga)

	logger.Log.Info(fmt.Sprintf("%s Конец шага отмены резерва товаров", place),
		zap.String("request_id", saga.RequestID),
		zap.Int("user_id", saga.UserID),
		zap.Int("saga_id", saga.ID),
	)

	go s.ContinueCancel(ctx)
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

		err = s.repo.CancelSaga(ctx, saga, domains.StatusCreateOrderFailed)
		if err != nil {
			logger.Log.Error(fmt.Sprintf("%s ошибка изменения статуса саги", place),
				zap.String("request_id", saga.RequestID),
				zap.Int("saga_id", saga.ID),
				zap.Error(err),
			)
			return
		}

		ctx = context.WithValue(ctx, "saga", *saga)
		go s.ContinueCancel(ctx)
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

func (s *OrderSagaService) CancelCreateOrder(ctx context.Context, saga *domains.OrderSaga) {
	place := "[OrderSagaService.CancelCreateOrder]"
	
	err := s.repo.ChangeStatus(ctx, saga, domains.StatusCreateOrderCancelled)
	if err != nil {
		logger.Log.Error(fmt.Sprintf("%s Ошибка изменения статуса саги", place),
			zap.String("request_id", saga.RequestID),
			zap.Int("user_id", saga.UserID),
			zap.Int("saga_id", saga.ID),
			zap.Error(err),
		)
		return
	}

	ctx = context.WithValue(ctx, "saga", *saga)
	go s.ContinueCancel(ctx)
}

func (s *OrderSagaService) ConfirmReservation(ctx context.Context, saga *domains.OrderSaga) {
	place := "[OrderSagaService.ConfirmReservation]"

	logger.Log.Info(fmt.Sprintf("%s Начало шага подтверждение бронирования товаров", place),
		zap.String("request_id", saga.RequestID),
		zap.Int("user_id", saga.UserID),
		zap.Int("saga_id", saga.ID),
	)

	err := s.repo.ChangeStatus(ctx, saga, domains.StatusConfirmReservationStarted)
	if err != nil {
		logger.Log.Error(fmt.Sprintf("%s Ошибка изменения статуса саги", place),
			zap.String("request_id", saga.RequestID),
			zap.Int("user_id", saga.UserID),
			zap.Int("saga_id", saga.ID),
			zap.Error(err),
		)
		return
	}

	request := domains.ConfirmReserveRequest{
		RequestID: saga.RequestID,
		OrderID:   saga.OrderID,
	}

	err = s.inventoryClient.Confirm(&request)
	if err != nil {
		logger.Log.Error(fmt.Sprintf("%s Не удалось подтвердить резерв товаров", place),
			zap.String("request_id", saga.RequestID),
			zap.Int("user_id", saga.UserID),
			zap.Int("saga_id", saga.ID),
			zap.Error(err),
		)
		return
	}

	err = s.repo.ChangeStatus(ctx, saga, domains.StatusConfirmReservationFinished)
	if err != nil {
		logger.Log.Error(fmt.Sprintf("%s Ошибка изменения статуса саги", place),
			zap.String("request_id", saga.RequestID),
			zap.Int("user_id", saga.UserID),
			zap.Int("saga_id", saga.ID),
			zap.Error(err),
		)
		return
	}

	logger.Log.Info(fmt.Sprintf("%s Подтверждение бронирования товаров завершено", place),
		zap.String("request_id", saga.RequestID),
		zap.Int("user_id", saga.UserID),
		zap.Int("saga_id", saga.ID),
	)

	ctx = context.WithValue(ctx, "saga", *saga)
	go s.Continue(ctx)
}

func (s *OrderSagaService) CancelConfirmReservation(ctx context.Context, saga *domains.OrderSaga) {}
