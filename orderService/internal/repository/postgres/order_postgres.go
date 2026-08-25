package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	_ "github.com/lib/pq"
	"github.com/yourusername/order-service/internal/domain"
)

type OrderRepository struct {
	Db *sql.DB
}

func (r *OrderRepository) Create(ctx context.Context, order *domain.Order) error {
	itemsJSON, err := json.Marshal(order.Items)
	if err != nil {
		return fmt.Errorf("failed to marshal items: %w", err)
	}

	query := `INSERT INTO orders (user_id, items, status, total, created_at, updated_at) 
              VALUES ($1, $2, $3, $4, NOW(), NOW()) RETURNING id, created_at`

	err = r.Db.QueryRowContext(ctx, query, order.UserID, itemsJSON, order.Status, order.Total).
		Scan(&order.ID, &order.CreatedAt)
	return err
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
