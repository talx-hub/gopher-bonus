package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"github.com/talx-hub/gopher-bonus/internal/api/dto"
	"github.com/talx-hub/gopher-bonus/internal/model/bonus"
	"github.com/talx-hub/gopher-bonus/internal/model/order"
	"github.com/talx-hub/gopher-bonus/internal/model/user"
)

type GeneralHandler struct{}

type UserHandler struct {
	repo user.Repository
}

type OrderHandler struct {
	_ order.Repository
}

type TransactionHandler struct {
	_ bonus.Repository
}

func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	data := dto.User{}
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	hasher := sha256.New()
	hasher.Write([]byte(data.Password))
	loginHash := hex.EncodeToString(hasher.Sum(nil))
	if _, err = h.repo.FindByLogin(r.Context(), loginHash); err == nil {
		http.Error(w, "User already exists", http.StatusConflict)
	}

	hasher.Reset()
	hasher.Write([]byte(data.Password))
	passwordHash := hex.EncodeToString(hasher.Sum(nil))

	err = h.repo.Create(r.Context(), &user.User{
		ID:           uuid.NewString(),
		LoginHash:    loginHash,
		PasswordHash: passwordHash,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (h *GeneralHandler) Login(w http.ResponseWriter, r *http.Request) {}

func (h *OrderHandler) PostOrder(w http.ResponseWriter, r *http.Request) {}

func (h *OrderHandler) GetOrders(w http.ResponseWriter, r *http.Request) {}

func (h *TransactionHandler) GetBalance(w http.ResponseWriter, r *http.Request) {}

func (h *TransactionHandler) Withdraw(w http.ResponseWriter, r *http.Request) {}

func (h *TransactionHandler) GetWithdrawals(w http.ResponseWriter, r *http.Request) {}

func (h *GeneralHandler) Ping(w http.ResponseWriter, r *http.Request) {}
