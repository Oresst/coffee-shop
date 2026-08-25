package repository

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/yourusername/order-service/internal/repository/postgres"

	"github.com/yourusername/order-service/internal/config"
	"github.com/yourusername/order-service/internal/domain"
)

type OrderRepositoryInt interface {
	Create(ctx context.Context, order *domain.Order) error
	FindByID(ctx context.Context, id int64) (*domain.Order, error)
	FindByUserID(ctx context.Context, userID int64) ([]domain.Order, error)
	UpdateStatus(ctx context.Context, id int64, status string) error
	Close() error
}

func NewOrderRepositoryPostgres(cfg *config.Config) (OrderRepositoryInt, error) {
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &postgres.OrderRepository{Db: db}, nil
}

func NewOrderRepository(cfg *config.Config) (OrderRepositoryInt, error) {
	return NewOrderRepositoryPostgres(cfg)
}
