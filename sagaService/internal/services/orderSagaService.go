package services

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/yourusername/saga-service/internal/config"
	"github.com/yourusername/saga-service/internal/domains"
	"github.com/yourusername/saga-service/internal/repositories/postgres"
	"github.com/yourusername/saga-service/internal/services/client"
	"time"
)

type OrderSagaService struct {
	repo            postgres.CreateOrderSagaRepoInt
	inventoryClient *client.InventoryClient
}

func NewOrderSagaService(repo postgres.CreateOrderSagaRepoInt, cfg *config.Config) *OrderSagaService {
	inventoryClient, err := client.NewInventoryClient(cfg)
	if err != nil {
		panic(err)
	}

	return &OrderSagaService{
		repo:            repo,
		inventoryClient: inventoryClient,
	}
}

func (s *OrderSagaService) StartSaga(ctx context.Context, userID int, items []*domains.Item) (*domains.CreateOrderSagaResponse, error) {
	requestID := uuid.New().String()

	saga := domains.OrderSaga{
		UserID:    userID,
		Items:     items,
		RequestID: requestID,
		Cancelled: false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Status:    domains.StatusCreated,
	}

	err := s.repo.CreateSaga(ctx, &saga)
	if err != nil {
		return nil, fmt.Errorf("failed to insert into order_saga: %w", err)
	}

	ctx = context.WithValue(ctx, "saga", saga)

	go s.Continue(ctx)

	return &domains.CreateOrderSagaResponse{
		RequestID: requestID,
	}, nil
}

func (s *OrderSagaService) Continue(ctx context.Context) {
	saga, ok := ctx.Value("saga").(*domains.OrderSaga)
	if !ok {
		return
	}

	switch saga.Status {
	case domains.StatusCreated:
		s.Reserve(ctx, saga)
	}
}

func (s *OrderSagaService) Reserve(ctx context.Context, saga *domains.OrderSaga) {
	request := &domains.ReserveRequest{
		RequestID: saga.RequestID,
		Items:     saga.Items,
	}

	_, err := s.inventoryClient.Reserve(request)
	if err != nil {
		return
	}

	err = s.repo.ChangeStatus(ctx, saga, domains.StatusReservedFinished)
	if err != nil {
		return
	}

	ctx = context.WithValue(ctx, "saga", saga)

	go s.Continue(ctx)
}
