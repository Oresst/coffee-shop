package domain

import "time"

type InventoryItem struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Quantity  int       `json:"quantity"`
	Price     float64   `json:"price"`
	Reserved  int       `json:"reserved"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Reservation struct {
	ID        int64             `json:"id"`
	RequestID string            `json:"request_id"`
	OrderID   int64             `json:"order_id"`
	ItemID    int64             `json:"item_id"`
	Quantity  int               `json:"quantity"`
	Status    ReservationStatus `json:"status"`
	ExpiresAt time.Time         `json:"expires_at"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

type ReservationStatus string

const (
	ReservationStatusPending   ReservationStatus = "pending"
	ReservationStatusComplete  ReservationStatus = "complete"
	ReservationStatusCancelled ReservationStatus = "cancelled"
)

// CheckAvailabilityRequest — запрос на проверку доступности
type CheckAvailabilityRequest struct {
	Items []OrderItemRequest `json:"items"`
}

// OrderItemRequest — товар из заказа
type OrderItemRequest struct {
	ProductID int64 `json:"product_id"`
	Quantity  int   `json:"quantity"`
}

// ReserveRequest — запрос на резервацию
type ReserveRequest struct {
	RequestID string             `json:"request_id"`
	Items     []OrderItemRequest `json:"items"`
	OrderID   *int64             `json:"order_id"`
	UserId    int64              `json:"user_id"`
}

// ReserveResponse — ответ на резервацию
type ReserveResponse struct {
	Success   bool            `json:"success"`
	RequestID string          `json:"request_id"`
	Message   string          `json:"message,omitempty"`
	OrderID   *int64          `json:"order_id,omitempty"`
	Items     []*ReservedItem `json:"items"`
}

type ReservedItem struct {
	ItemID   int64   `json:"item_id"`
	Quantity int     `json:"quantity"`
	Price    float64 `json:"price"`
}
