package order

import (
	"time"

	"github.com/talx-hub/gopher-bonus/internal/model"
)

type Status string

const (
	StatusNew        Status = "NEW"
	StatusProcessing Status = "PROCESSING"
	StatusInvalid    Status = "INVALID"
	StatusProcessed  Status = "PROCESSED"
)

type Type string

const (
	TypeAccrual    Type = "accrual"
	TypeWithdrawal Type = "withdrawal"
)

type Order struct {
	CreatedAt time.Time    `json:"created_at"`
	ID        string       `json:"id"`
	UserID    string       `json:"user_id"`
	OrderID   string       `json:"order_id"`
	Status    Status       `json:"status"`
	Type      Type         `json:"type"`
	Amount    model.Amount `json:"amount"`
}
