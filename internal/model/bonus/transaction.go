package bonus

import (
	"time"

	"github.com/talx-hub/gopher-bonus/internal/model"
)

type TransactionType string

const (
	TypeAccrual    TransactionType = "accrual"
	TypeWithdrawal TransactionType = "withdrawal"
)

type Transaction struct {
	ID        string          `json:"id"`
	UserID    string          `json:"user_id"`
	OrderID   string          `json:"order_id"`
	CreatedAt time.Time       `json:"created_at"`
	Type      TransactionType `json:"type"`
	Amount    model.Amount    `json:"amount"`
}
