package domain

import (
	"slices"
	"time"
)

type Order struct {
	ID        int64       `json:"id"`
	UserID    int64       `json:"user_id"`
	Items     []OrderItem `json:"items"`
	Status    OrderStatus `json:"status"` // pending, confirmed, cancelled
	Total     float64     `json:"total"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}

type OrderStatus string

const (
	StatusNew       OrderStatus = "new"
	StatusPending   OrderStatus = "pending"
	StatusComplete  OrderStatus = "complete"
	StatusCancelled OrderStatus = "cancelled"
)

var allowedOrderStatuses = []OrderStatus{
	StatusNew,
	StatusPending,
	StatusComplete,
	StatusCancelled,
}

func (s OrderStatus) IsValid() bool {
	return slices.Contains(allowedOrderStatuses, s)
}

type OrderItem struct {
	ProductID int64   `json:"product_id"`
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price"`
}

type CreateOrderRequest struct {
	UserID int64       `json:"user_id"`
	Items  []OrderItem `json:"items"`
}

type OrderResponse struct {
	ID     int64       `json:"id"`
	UserID int64       `json:"user_id"`
	Items  []OrderItem `json:"items"`
	Status OrderStatus `json:"status"`
	Total  float64     `json:"total"`
}
