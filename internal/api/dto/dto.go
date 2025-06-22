package dto

import (
	"encoding/json"
	"errors"
	"time"

	passwordvalidator "github.com/wagslane/go-password-validator"

	"github.com/talx-hub/gopher-bonus/internal/model/order"
)

type UserRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

func (r *UserRequest) IsValid() error {
	var invalidLoginErr error
	if r.Login == "" {
		invalidLoginErr = errors.New("login is empty")
	}

	const minEntropyBits = 50
	invalidPasswordErr := passwordvalidator.Validate(r.Password, minEntropyBits)
	return errors.Join(invalidLoginErr, invalidPasswordErr)
}

type OrderResponse struct {
	UploadedAt time.Time    `json:"uploaded_at"`
	Status     order.Status `json:"status"`
	OrderID    string       `json:"number"`
	Accrual    json.Number  `json:"accrual,omitempty"`
}

type BalanceResponse struct {
	Current   json.Number `json:"current"`
	Withdrawn json.Number `json:"withdrawn"`
}

type WithdrawRequest struct {
	OrderID string      `json:"order"`
	Sum     json.Number `json:"sum"`
}

type WithdrawalResponse struct {
	ProcessedAt time.Time   `json:"processed_at"`
	OrderID     string      `json:"order"`
	Sum         json.Number `json:"sum"`
}
