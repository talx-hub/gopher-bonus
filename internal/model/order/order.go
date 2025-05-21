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

type Order struct {
	UploadedAt time.Time    `json:"uploaded_at"`
	Status     Status       `json:"status"`
	ID         string       `json:"id"`
	UserID     string       `json:"user_id"`
	Accrual    model.Amount `json:"accrual,omitempty"`
}
