package dto

import (
	"encoding/json"
	"errors"

	passwordvalidator "github.com/wagslane/go-password-validator"
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

type BalanceResponse struct {
	Current   json.Number `json:"current"`
	Withdrawn json.Number `json:"withdrawn"`
}

type WithdrawRequest struct {
	OrderID string      `json:"order"`
	Sum     json.Number `json:"sum"`
}
