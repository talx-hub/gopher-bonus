package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"github.com/talx-hub/gopher-bonus/internal/api/dto"
	"github.com/talx-hub/gopher-bonus/internal/model/bonus"
	"github.com/talx-hub/gopher-bonus/internal/model/order"
	"github.com/talx-hub/gopher-bonus/internal/model/user"
	"github.com/talx-hub/gopher-bonus/internal/utils/auth"
)

type AuthHandler struct {
	repo   user.Repository
	secret string
}

type OrderHandler struct {
	_ order.Repository
}

type TransactionHandler struct {
	_ bonus.Repository
}

type GeneralHandler struct{}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	data := dto.UserRequest{}
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

	u := user.User{
		ID:           uuid.NewString(),
		LoginHash:    loginHash,
		PasswordHash: passwordHash,
	}
	err = h.repo.Create(r.Context(), &u)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jwtCookie, err := auth.Authenticate(u.ID, []byte(h.secret))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &jwtCookie)
	w.WriteHeader(http.StatusCreated)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	data := dto.UserRequest{}
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	hasher := sha256.New()
	hasher.Write([]byte(data.Login))
	loginHash := hasher.Sum(nil)
	var u *user.User
	if u, err =
		h.repo.FindByLogin(r.Context(), hex.EncodeToString(loginHash)); err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	hasher.Reset()
	hasher.Write([]byte(data.Password))
	passwordHash := hasher.Sum(nil)

	storedHash, err := hex.DecodeString(u.PasswordHash)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !hmac.Equal(passwordHash, storedHash) {
		http.Error(w,
			"email or password is incorrect",
			http.StatusUnauthorized)
		return
	}

	jwtCookie, err := auth.Authenticate(u.ID, []byte(h.secret))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &jwtCookie)
	w.WriteHeader(http.StatusOK)
}

func (h *OrderHandler) PostOrder(w http.ResponseWriter, r *http.Request) {}

func (h *OrderHandler) GetOrders(w http.ResponseWriter, r *http.Request) {}

func (h *TransactionHandler) GetBalance(w http.ResponseWriter, r *http.Request) {}

func (h *TransactionHandler) Withdraw(w http.ResponseWriter, r *http.Request) {}

func (h *TransactionHandler) GetWithdrawals(w http.ResponseWriter, r *http.Request) {}

func (h *GeneralHandler) Ping(w http.ResponseWriter, r *http.Request) {}
