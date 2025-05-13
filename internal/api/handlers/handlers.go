package handlers

import (
	"net/http"

	"github.com/talx-hub/gopher-bonus/internal/model"
)

type HTTPHandler struct {
	_ model.Repository
}

func (h *HTTPHandler) Register(w http.ResponseWriter, r *http.Request) {}

func (h *HTTPHandler) Login(w http.ResponseWriter, r *http.Request) {}

func (h *HTTPHandler) GetOrders(w http.ResponseWriter, r *http.Request) {}

func (h *HTTPHandler) PostOrders(w http.ResponseWriter, r *http.Request) {}

func (h *HTTPHandler) GetBalance(w http.ResponseWriter, r *http.Request) {}

func (h *HTTPHandler) Withdraw(w http.ResponseWriter, r *http.Request) {}

func (h *HTTPHandler) GetStatistics(w http.ResponseWriter, r *http.Request) {}

func (h *HTTPHandler) Ping(w http.ResponseWriter, r *http.Request) {}
