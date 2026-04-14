package model

import "time"

type OrderStatus string

const (
	StatusPending   OrderStatus = "PENDING"
	StatusConfirmed OrderStatus = "CONFIRMED"
	StatusShipped   OrderStatus = "SHIPPED"
	StatusDelivered OrderStatus = "DELIVERED"
	StatusCancelled OrderStatus = "CANCELLED"
)

type OrderItem struct {
	ProductID string  `json:"product_id"`
	Name      string  `json:"name"`
	Price     float64 `json:"price"`
	Quantity  int     `json:"quantity"`
}

type Order struct {
	ID          string      `json:"id"`
	CustomerID  string      `json:"customer_id"`
	Status      OrderStatus `json:"status"`
	Items       []OrderItem `json:"items"`
	TotalAmount float64     `json:"total_amount"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

type CreateOrderRequest struct {
	CustomerID string      `json:"customer_id" binding:"required"`
	Items      []OrderItem `json:"items" binding:"required,min=1"`
}

type UpdateStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

func (o *Order) CanCancel() bool {
	return o.Status == StatusPending || o.Status == StatusConfirmed
}

func (o *Order) CancelIsConflict() bool {
	// Cancelling a CONFIRMED (already paid) order is a conflict scenario
	return o.Status == StatusConfirmed
}
