package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/lib/pq"
	"github.com/yourusername/saga-service/internal/config"
	"github.com/yourusername/saga-service/internal/domains"
)

type CreateOrderSagaRepoInt interface {
	CreateSaga(ctx context.Context, saga *domains.OrderSaga) error
	ChangeStatus(ctx context.Context, saga *domains.OrderSaga, status domains.OrderSagaStatus) error
	ChangeItems(ctx context.Context, sagaId int, items []*domains.Item) error
	ChangeOrderId(ctx context.Context, sagaId int, orderId int) error
	CancelSaga(ctx context.Context, saga *domains.OrderSaga, status domains.OrderSagaStatus) error
	GetNotCompleted(ctx context.Context) ([]*domains.OrderSaga, error)
	GetStatus(ctx context.Context, sagaId int) (*domains.OrderSagaStatus, error)
	GetUnfinishedSaga(ctx context.Context) ([]*domains.OrderSaga, error)
	Close()
}

type CreateOrderSagaPostRepo struct {
	db *sql.DB
}

func NewCreateOrderSagaPostRepo(cfg *config.Config) (CreateOrderSagaRepoInt, error) {
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &CreateOrderSagaPostRepo{db: db}, nil
}

func (r *CreateOrderSagaPostRepo) Close() {
	r.db.Close()
}

func (r *CreateOrderSagaPostRepo) CreateSaga(ctx context.Context, saga *domains.OrderSaga) error {
	items, err := json.Marshal(saga.Items)
	if err != nil {
		return fmt.Errorf("failed to marshal items: %w", err)
	}

	query := `INSERT INTO order_sagas
    	(request_id, user_id, status, items, cancelled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW()) RETURNING id`

	err = r.db.QueryRowContext(ctx, query, saga.RequestID, saga.UserID, saga.Status.String(), items, saga.Cancelled).Scan(&saga.ID)
	if err != nil {
		return fmt.Errorf("failed to insert into order_saga: %w", err)
	}

	return nil
}

func (r *CreateOrderSagaPostRepo) ChangeStatus(ctx context.Context, saga *domains.OrderSaga, status domains.OrderSagaStatus) error {
	query := "UPDATE order_sagas SET status = $1, updated_at = NOW() WHERE id = $2"

	_, err := r.db.ExecContext(ctx, query, status, saga.ID)
	if err != nil {
		return fmt.Errorf("failed to update order_saga: %w", err)
	}

	saga.Status = status

	return nil
}

func (r *CreateOrderSagaPostRepo) CancelSaga(ctx context.Context, saga *domains.OrderSaga, status domains.OrderSagaStatus) error {
	query := "UPDATE order_sagas SET status = $1, cancelled = true, updated_at = NOW() WHERE id = $2"

	_, err := r.db.ExecContext(ctx, query, status, saga.ID)
	if err != nil {
		return err
	}

	saga.Status = status
	saga.Cancelled = true

	return nil
}

func (r *CreateOrderSagaPostRepo) GetNotCompleted(ctx context.Context) ([]*domains.OrderSaga, error) {
	var orderSagas []*domains.OrderSaga

	query := "SELECT id, request_id, user_id, status, cancelled, created_at, updated_at FROM order_sagas WHERE status not in ($1, $2)"

	rows, err := r.db.QueryContext(ctx, query, domains.StatusCreated.String(), domains.StatusCancelled.String())
	if err != nil {
		return orderSagas, fmt.Errorf("failed to query order_saga: %w", err)
	}

	defer rows.Close()

	for rows.Next() {
		var orderSaga domains.OrderSaga

		err = rows.Scan(&orderSaga.ID, &orderSaga.RequestID, &orderSaga.UserID, &orderSaga.Status, &orderSaga.Cancelled, &orderSaga.CreatedAt, &orderSaga.UpdatedAt)
		if err != nil {
			return orderSagas, fmt.Errorf("failed to scan order_saga: %w", err)
		}

		orderSagas = append(orderSagas, &orderSaga)
	}

	if err := rows.Err(); err != nil {
		return orderSagas, fmt.Errorf("failed to scan order_saga: %w", err)
	}

	return orderSagas, nil
}

func (r *CreateOrderSagaPostRepo) GetStatus(ctx context.Context, sagaId int) (*domains.OrderSagaStatus, error) {
	type resultStruct struct {
		Status domains.OrderSagaStatus
	}

	var result resultStruct

	query := "SELECT status FROM order_sagas WHERE id = $1"
	err := r.db.QueryRowContext(ctx, query, sagaId).Scan(&result.Status)
	if err != nil {
		return nil, fmt.Errorf("failed to query order_saga: %w", err)
	}

	return &result.Status, nil
}

func (r *CreateOrderSagaPostRepo) ChangeItems(ctx context.Context, sagaId int, items []*domains.Item) error {
	preparedItems, err := json.Marshal(&items)
	if err != nil {
		return err
	}

	query := `UPDATE order_sagas SET items = $1 WHERE id = $2`

	_, err = r.db.ExecContext(ctx, query, string(preparedItems), sagaId)
	if err != nil {
		return err
	}

	return nil
}

func (r *CreateOrderSagaPostRepo) ChangeOrderId(ctx context.Context, sagaId int, orderId int) error {
	query := "UPDATE order_sagas SET order_id = $1 WHERE id = $2"

	_, err := r.db.ExecContext(ctx, query, orderId, sagaId)
	if err != nil {
		return err
	}

	return nil
}

// GetUnfinishedSaga выбирает незавершённые "зависшие" саги и сразу помечает их как
// забронированные, сдвигая updated_at на NOW(). Это нужно, чтобы следующий тик
// producer'а (через 5 секунд) не забрал те же саги повторно, пока текущая обработка
// ещё не успела сменить статус — иначе одна и та же сага уходит в канал дважды и
// обрабатывается двумя consumer'ами параллельно. FOR UPDATE SKIP LOCKED дополнительно
// защищает от гонки, если этот запрос когда-нибудь выполняется из нескольких реплик.
func (r *CreateOrderSagaPostRepo) GetUnfinishedSaga(ctx context.Context) ([]*domains.OrderSaga, error) {
	var orderSagas []*domains.OrderSaga
	query := `
		WITH claimed AS (
			SELECT id
			FROM order_sagas
			WHERE status not in ($1, $2) AND updated_at < NOW() - INTERVAL '5 minutes'
			ORDER BY id DESC
			LIMIT 10
			FOR UPDATE SKIP LOCKED
		)
		UPDATE order_sagas
		SET updated_at = NOW()
		FROM claimed
		WHERE order_sagas.id = claimed.id
		RETURNING order_sagas.id, order_sagas.request_id, order_sagas.status, order_sagas.items,
			order_sagas.cancelled, order_sagas.user_id, order_sagas.order_id, order_sagas.created_at, order_sagas.updated_at
	`

	rows, err := r.db.QueryContext(ctx, query, domains.StatusCancelled.String(), domains.StatusCompleted.String())
	if err != nil {
		return orderSagas, err
	}
	defer rows.Close()

	for rows.Next() {
		var orderSaga domains.OrderSaga
		var items []byte

		err = rows.Scan(&orderSaga.ID, &orderSaga.RequestID, &orderSaga.Status, &items, &orderSaga.Cancelled,
			&orderSaga.UserID, &orderSaga.OrderID, &orderSaga.CreatedAt, &orderSaga.UpdatedAt)
		if err != nil {
			return orderSagas, err
		}

		err = json.Unmarshal(items, &orderSaga.Items)
		if err != nil {
			return orderSagas, err
		}

		orderSagas = append(orderSagas, &orderSaga)
	}

	if err := rows.Err(); err != nil {
		return orderSagas, err
	}

	return orderSagas, nil
}
