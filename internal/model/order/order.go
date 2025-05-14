package order

import (
	"context"
	"time"
)

type Status string

const (
	StatusNew        Status = "NEW"
	StatusProcessing Status = "PROCESSING"
	StatusInvalid    Status = "INVALID"
	StatusProcessed  Status = "PROCESSED"
)

type Order struct {
	CreatedAt time.Time `json:"created_at"`
	Status    Status    `json:"status"`
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
}

type Repository interface {
	Create(context.Context, Order) error
	FindByID(context.Context, string) (*Order, error)
	FindByUserID(context.Context, string) (*Order, error)
}
