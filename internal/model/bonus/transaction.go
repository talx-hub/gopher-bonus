package bonus

import (
	"time"
)

type TransactionType string

const (
	TypeAccrual    TransactionType = "accrual"
	TypeWithdrawal TransactionType = "withdrawal"
)

type Transaction struct {
	ID        string          `json:"id"`
	UserID    string          `json:"user_id"`
	CreatedAt time.Time       `json:"created_at"`
	Type      TransactionType `json:"type"`
	Amount    int             `json:"amount"`
}
