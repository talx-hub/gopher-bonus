package order

import (
	"encoding/json"
	"errors"
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
	Status    Status       `json:"status"`
	Type      Type         `json:"type"`
	Amount    model.Amount `json:"amount"`
}

func (o *Order) MarshalJSON() ([]byte, error) {
	data := make(map[string]interface{})

	switch o.Type {
	case TypeAccrual:
		data["number"] = o.ID
		data["status"] = o.Status
		if o.Status == StatusProcessed {
			data["accrual"] = json.Number(o.Amount.String())
		}
		data["uploaded_at"] = o.CreatedAt.Local().Format(time.RFC3339)
	case TypeWithdrawal:
		data["order"] = o.ID
		data["sum"] = json.Number(o.Amount.String())
		data["processed_at"] = o.CreatedAt.Local().Format(time.RFC3339)
	default:
		return nil, errors.New("failed to Marshal order.Order: unknown order type")
	}

	return json.Marshal(data) //nolint: wrapcheck // error from wrapped func
}
