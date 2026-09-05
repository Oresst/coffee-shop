package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/yourusername/order-service/pkg/logger"
	"go.uber.org/zap"

	_ "github.com/lib/pq"
	"github.com/yourusername/order-service/internal/domain"
)

type OrderRepository struct {
	Db *sql.DB
}

func (r *OrderRepository) Create(ctx context.Context, order *domain.Order) error {
	place := "[OrderRepository.Create]"

	itemsJSON, err := json.Marshal(order.Items)
	if err != nil {
		logger.Log.Error(fmt.Sprintf("%s Ошибка сериализации order.Items", place),
			logger.WithTraceID(ctx),
			zap.Error(err),
			zap.Int64("user_id", order.UserID),
			zap.String("request_id", order.RequestID),
		)
		return err
	}

	query := `INSERT INTO orders (user_id, items, status, total, request_id, created_at, updated_at) 
              VALUES ($1, $2, $3, $4, $5, NOW(), NOW()) RETURNING id, created_at`

	err = r.Db.QueryRowContext(ctx, query, order.UserID, itemsJSON, order.Status, order.Total, order.RequestID).
		Scan(&order.ID, &order.CreatedAt)

	if err != nil {
		logger.Log.Error(fmt.Sprintf("%s Ошибка записи в таблицу orders", place),
			logger.WithTraceID(ctx),
			zap.Error(err),
			zap.Int64("user_id", order.UserID),
			zap.String("request_id", order.RequestID),
		)
		return err
	}

	return nil
}

func (r *OrderRepository) FindByRequestID(ctx context.Context, requestId string) (*domain.Order, error) {
	place := "[OrderRepository.FindByRequestID]"
	var order domain.Order

	query := `SELECT id, status, user_id FROM orders WHERE request_id = $1`
	err := r.Db.QueryRowContext(ctx, query, requestId).Scan(&order.ID, &order.Status, &order.UserID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		logger.Log.Error(fmt.Sprintf("%s Ошибка получения заказа", place),
			logger.WithTraceID(ctx),
			zap.Error(err),
			zap.Int64("user_id", order.UserID),
			zap.String("request_id", requestId),
		)
		return nil, err
	}

	return &order, nil
}

func (r *OrderRepository) FindByID(ctx context.Context, id int64) (*domain.Order, error) {
	query := `SELECT id, user_id, items, status, total, created_at, updated_at 
              FROM orders WHERE id = $1`

	var order domain.Order
	var itemsJSON []byte

	err := r.Db.QueryRowContext(ctx, query, id).Scan(
		&order.ID, &order.UserID, &itemsJSON, &order.Status,
		&order.Total, &order.CreatedAt, &order.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(itemsJSON, &order.Items); err != nil {
		return nil, fmt.Errorf("failed to unmarshal items: %w", err)
	}

	return &order, nil
}

func (r *OrderRepository) FindByUserID(ctx context.Context, userID int64) ([]domain.Order, error) {
	query := `SELECT id, user_id, items, status, total, created_at, updated_at 
              FROM orders WHERE user_id = $1 ORDER BY created_at DESC`

	rows, err := r.Db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []domain.Order
	for rows.Next() {
		var order domain.Order
		var itemsJSON []byte

		if err := rows.Scan(&order.ID, &order.UserID, &itemsJSON, &order.Status,
			&order.Total, &order.CreatedAt, &order.UpdatedAt); err != nil {
			return nil, err
		}

		if err := json.Unmarshal(itemsJSON, &order.Items); err != nil {
			return nil, fmt.Errorf("failed to unmarshal items: %w", err)
		}

		orders = append(orders, order)
	}

	return orders, nil
}

func (r *OrderRepository) UpdateStatus(ctx context.Context, id int64, status string) error {
	query := `UPDATE orders SET status = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.Db.ExecContext(ctx, query, status, id)
	return err
}

func (r *OrderRepository) Close() error {
	return r.Db.Close()
}
