package domains

import "time"

type OrderSaga struct {
	ID        int             `json:"id"`
	RequestID string          `json:"request_id"`
	Status    OrderSagaStatus `json:"status"`
	Items     []*Item         `json:"items"`
	Cancelled bool            `json:"cancelled"`
	UserID    int             `json:"user_id"`
	OrderID   int             `json:"order_id"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

type OrderSagaStatus string

const (
	StatusCreated OrderSagaStatus = "created"

	StatusReserveStarted   OrderSagaStatus = "reserve_started"
	StatusReservedFinished OrderSagaStatus = "reserved_finished"
	StatusReservedFailed   OrderSagaStatus = "reserved_failed"

	StatusCompleted OrderSagaStatus = "completed"
	StatusCancelled OrderSagaStatus = "cancelled"
)

func (s OrderSagaStatus) String() string {
	switch s {
	case StatusCreated:
		return "created"

	case StatusReserveStarted:
		return "reserve_started"
	case StatusReservedFinished:
		return "reserved_finished"
	case StatusReservedFailed:
		return "reserved_failed"

	case StatusCompleted:
		return "completed"
	case StatusCancelled:
		return "cancelled"
	default:
		return "unknown"
	}
}

type Item struct {
	ProductID int64 `json:"product_id"`
	Quantity  int   `json:"quantity"`
}

type ReserveRequest struct {
	RequestID string  `json:"request_id"`
	UserID    int     `json:"user_id"`
	Items     []*Item `json:"items"`
}

type ReserveResponse struct {
	Success   bool   `json:"success"`
	RequestID string `json:"request_id"`
	Message   string `json:"message,omitempty"`
}

type CreateOrderSagaRequest struct {
	Items  []*Item `json:"items"`
	UserId int     `json:"user_id"`
}

type CreateOrderSagaResponse struct {
	RequestID string `json:"request_id"`
}
