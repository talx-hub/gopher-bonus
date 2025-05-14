package handlers

import (
	"net/http"

	"github.com/talx-hub/gopher-bonus/internal/model/bonus"
	"github.com/talx-hub/gopher-bonus/internal/model/order"
	"github.com/talx-hub/gopher-bonus/internal/model/user"
)

type GeneralHandler struct{}

type UserHandler struct {
	repo   user.Repository
	secret string
}

type OrderHandler struct {
	repo order.Repository
}

type TransactionHandler struct {
	repo bonus.Repository
}

func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {}

func (h *GeneralHandler) Login(w http.ResponseWriter, r *http.Request) {}

func (h *OrderHandler) PostOrder(w http.ResponseWriter, r *http.Request) {}

func (h *OrderHandler) GetOrders(w http.ResponseWriter, r *http.Request) {}

func (h *TransactionHandler) GetBalance(w http.ResponseWriter, r *http.Request) {}

func (h *TransactionHandler) Withdraw(w http.ResponseWriter, r *http.Request) {}

func (h *TransactionHandler) GetWithdrawals(w http.ResponseWriter, r *http.Request) {}

func (h *GeneralHandler) Ping(w http.ResponseWriter, r *http.Request) {}
