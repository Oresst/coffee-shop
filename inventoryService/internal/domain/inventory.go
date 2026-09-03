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
	Items []OrderItem `json:"items"`
}

// OrderItem — товар из заказа
type OrderItem struct {
	ProductID int64 `json:"product_id"`
	Quantity  int   `json:"quantity"`
}

// ReserveRequest — запрос на резервацию
type ReserveRequest struct {
	RequestID string      `json:"request_id"`
	OrderID   int64       `json:"order_id"`
	Items     []OrderItem `json:"items"`
}

// ReserveResponse — ответ на резервацию
type ReserveResponse struct {
	Success   bool   `json:"success"`
	RequestID string `json:"request_id"`
	OrderID   int64  `json:"order_id"`
	Message   string `json:"message,omitempty"`
}
