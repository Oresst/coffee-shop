package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"github.com/yourusername/inventory-service/internal/config"
	"github.com/yourusername/inventory-service/internal/domain"
)

type InventoryRepository struct {
	db *sql.DB
}

func NewInventoryRepository(cfg *config.Config) (*InventoryRepository, error) {
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &InventoryRepository{db: db}, nil
}

func (r *InventoryRepository) Close() error {
	return r.db.Close()
}

// CheckAvailability — проверяет, доступно ли количество товаров
func (r *InventoryRepository) CheckAvailability(ctx context.Context, items []domain.OrderItemRequest) (bool, error) {
	for _, item := range items {
		var available int
		query := `SELECT quantity - reserved FROM inventory WHERE id = $1`
		err := r.db.QueryRowContext(ctx, query, item.ProductID).Scan(&available)
		if err != nil {
			if err == sql.ErrNoRows {
				return false, fmt.Errorf("product %d not found", item.ProductID)
			}
			return false, err
		}
		if available < item.Quantity {
			return false, fmt.Errorf("product %d: requested %d, available %d",
				item.ProductID, item.Quantity, available)
		}
	}
	return true, nil
}

// ReserveItems — резервирует товары в рамках транзакции
func (r *InventoryRepository) ReserveItems(ctx context.Context, requestID string, orderID *int64, items []domain.OrderItemRequest) error {
	// Начинаем транзакцию
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// Для каждого товара:
	// 1. Проверяем доступность
	// 2. Увеличиваем reserved
	// 3. Создаём запись о резервации
	for _, item := range items {
		// Проверяем доступность с блокировкой строки (FOR UPDATE)
		var available int
		query := `SELECT quantity - reserved FROM inventory WHERE id = $1 FOR UPDATE`
		err = tx.QueryRowContext(ctx, query, item.ProductID).Scan(&available)
		if err != nil {
			return err
		}

		if available < item.Quantity {
			err = fmt.Errorf("product %d: requested %d, available %d", item.ProductID, item.Quantity, available)
			return err
		}

		// Обновляем reserved
		updateQuery := `UPDATE inventory SET reserved = reserved + $1, updated_at = NOW() WHERE id = $2`
		_, err = tx.ExecContext(ctx, updateQuery, item.Quantity, item.ProductID)
		if err != nil {
			return err
		}

		// Создаём запись о резервации
		reservationQuery := `
            INSERT INTO reservations (request_id, order_id, item_id, quantity, status, expires_at, created_at, updated_at)
            VALUES ($1, $2, $3, $4, '%s', $5, NOW(), NOW())
        `
		expiresAt := time.Now().Add(10 * time.Minute) // резервация на 10 минут
		_, err = tx.ExecContext(ctx, fmt.Sprintf(reservationQuery, domain.ReservationStatusPending), requestID, orderID, item.ProductID, item.Quantity, expiresAt)
		if err != nil {
			return err
		}
	}

	// Подтверждаем транзакцию
	if err = tx.Commit(); err != nil {
		return err
	}

	return nil
}

// ConfirmReservation — подтверждает резервацию (используется после оплаты)
func (r *InventoryRepository) ConfirmReservation(ctx context.Context, requestID string) error {
	// Обновляем статус резерваций и списываем товары
	query := `
        UPDATE reservations 
        SET status = '%s', updated_at = NOW()
        WHERE request_id = $1 AND status = '%s'
    `
	_, err := r.db.ExecContext(ctx, fmt.Sprintf(query, domain.ReservationStatusComplete, domain.ReservationStatusPending), requestID)
	if err != nil {
		return fmt.Errorf("failed to confirm reservation: %w", err)
	}
	return nil
}

// CancelReservation — отменяет резервацию (компенсация)
func (r *InventoryRepository) CancelReservation(ctx context.Context, requestID string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// Получаем все pending резервации по request_id
	selectQuery := `SELECT item_id, quantity FROM reservations WHERE request_id = $1 AND status = '%s'`
	rows, err := tx.QueryContext(ctx, fmt.Sprintf(selectQuery, domain.ReservationStatusPending), requestID)
	if err != nil {
		return fmt.Errorf("failed to get reservations: %w", err)
	}
	defer rows.Close()

	// Уменьшаем reserved для каждого товара
	for rows.Next() {
		var itemID, quantity int64
		if err := rows.Scan(&itemID, &quantity); err != nil {
			return err
		}
		updateQuery := `UPDATE inventory SET reserved = reserved - $1, updated_at = NOW() WHERE id = $2`
		_, err = tx.ExecContext(ctx, updateQuery, quantity, itemID)
		if err != nil {
			return fmt.Errorf("failed to update inventory: %w", err)
		}
	}

	// Обновляем статус резерваций на cancelled
	cancelQuery := `UPDATE reservations SET status = '%s', updated_at = NOW() WHERE request_id = $1 AND status = '%s'`
	_, err = tx.ExecContext(ctx, fmt.Sprintf(cancelQuery, domain.ReservationStatusCancelled, domain.ReservationStatusPending), requestID)
	if err != nil {
		return fmt.Errorf("failed to cancel reservations: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetReservationStatus — получает статус резервации по request_id
func (r *InventoryRepository) GetReservationStatus(ctx context.Context, requestID string) (string, error) {
	var status string
	query := `SELECT status FROM reservations WHERE request_id = $1 LIMIT 1`
	err := r.db.QueryRowContext(ctx, query, requestID).Scan(&status)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	return status, nil
}

// GetReservedItems - получает зарезервированные товары по request_id
func (r *InventoryRepository) GetReservedItems(ctx context.Context, requestID string) ([]*domain.ReservedItem, error) {
	var items []*domain.ReservedItem

	query := `SELECT i.id, i.quantity, i.price 
			  FROM reservations r
			  LEFT JOIN inventory i ON r.item_id = i.id
			  WHERE r.request_id = $1`

	rows, err := r.db.QueryContext(ctx, query, requestID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var item domain.ReservedItem

		err = rows.Scan(&item.ItemID, &item.Quantity, &item.Price)
		if err != nil {
			return nil, err
		}

		items = append(items, &item)
	}

	return items, nil
}
