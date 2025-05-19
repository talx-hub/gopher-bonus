package handlers

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"
	"unicode"

	"github.com/google/uuid"

	"github.com/talx-hub/gopher-bonus/internal/api/dto"
	"github.com/talx-hub/gopher-bonus/internal/model"
	"github.com/talx-hub/gopher-bonus/internal/model/bonus"
	"github.com/talx-hub/gopher-bonus/internal/model/order"
	"github.com/talx-hub/gopher-bonus/internal/model/user"
	"github.com/talx-hub/gopher-bonus/internal/serviceerrs"
	"github.com/talx-hub/gopher-bonus/internal/utils/auth"
)

type UserRepository interface {
	Create(ctx context.Context, u *user.User) error
	Exists(ctx context.Context, loginHash string) bool
	FindByLogin(ctx context.Context, loginHash string) (*user.User, error)
	FindByID(ctx context.Context, id string) (*user.User, error)
}

type AuthHandler struct {
	logger *slog.Logger
	repo   UserRepository
	secret string
}

type OrderRepository interface {
	Create(ctx context.Context, o order.Order) error
	FindByID(ctx context.Context, id string) (*order.Order, error)
	FindByUserID(ctx context.Context, userID string) (*order.Order, error)
}

type OrderHandler struct {
	logger *slog.Logger
	repo   OrderRepository
}

type BonusRepository interface {
	CreateTransaction(ctx context.Context, t *bonus.Transaction) error
	ListTransactionsByUser(
		ctx context.Context, userID string, tp bonus.TransactionType,
	) ([]bonus.Transaction, error)
}

type TransactionHandler struct {
	_ BonusRepository
}

type GeneralHandler struct{}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	data := dto.UserRequest{}
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		h.logger.LogAttrs(r.Context(),
			slog.LevelError,
			"failed to read the request body",
			slog.Any(model.KeyLoggerError, err),
		)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	err = r.Body.Close()
	if err != nil {
		h.logger.LogAttrs(r.Context(),
			slog.LevelError,
			"failed to close request body",
			slog.Any(model.KeyLoggerError, err),
		)
	}
	if err = data.IsValid(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	hasher := sha256.New()
	hasher.Write([]byte(data.Login))
	loginHash := hex.EncodeToString(hasher.Sum(nil))
	if h.repo.Exists(r.Context(), loginHash) {
		http.Error(w, "User already exists", http.StatusConflict)
		return
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
		h.logger.LogAttrs(r.Context(),
			slog.LevelError,
			"failed to read the request body",
			slog.Any(model.KeyLoggerError, err),
		)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	err = r.Body.Close()
	if err != nil {
		h.logger.LogAttrs(r.Context(),
			slog.LevelError,
			"failed to close request body",
			slog.Any(model.KeyLoggerError, err),
		)
	}
	unauthorizedErr := errors.New("login or password is incorrect")
	if err = data.IsValid(); err != nil {
		http.Error(w, unauthorizedErr.Error(), http.StatusUnauthorized)
		return
	}

	hasher := sha256.New()
	hasher.Write([]byte(data.Login))
	loginHash := hex.EncodeToString(hasher.Sum(nil))
	u, err := h.repo.FindByLogin(r.Context(), loginHash)
	if err != nil && errors.Is(err, serviceerrs.ErrNotFound) {
		h.logger.LogAttrs(r.Context(),
			slog.LevelError,
			"no user with given login",
			slog.String("login", data.Login),
			slog.String("loginHash", loginHash))
		http.Error(w, unauthorizedErr.Error(), http.StatusUnauthorized)
		return
	} else if err != nil {
		h.logger.LogAttrs(r.Context(),
			slog.LevelError,
			"failed to find user by ID",
			slog.String("login", data.Login),
			slog.String("loginHash", loginHash),
			slog.Any(model.KeyLoggerError, err),
		)
		http.Error(w, serviceerrs.ErrUnexpected.Error(), http.StatusInternalServerError)
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
		http.Error(w, unauthorizedErr.Error(), http.StatusUnauthorized)
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

func (h *OrderHandler) PostOrder(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.LogAttrs(r.Context(),
			slog.LevelError,
			"failed to read the request body",
			slog.Any(model.KeyLoggerError, err),
		)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = r.Body.Close()
	if err != nil {
		h.logger.LogAttrs(r.Context(),
			slog.LevelError,
			"failed to close request body",
			slog.Any(model.KeyLoggerError, err),
		)
	}
	orderID := string(body)
	for _, rn := range orderID {
		if !unicode.IsDigit(rn) {
			http.Error(w, "order ID must contain only digits", http.StatusBadRequest)
			return
		}
	}

	userID, ok := r.Context().Value(model.KeyContextUserID).(string)
	if !ok {
		h.logger.LogAttrs(r.Context(),
			slog.LevelError,
			"failed userID retrieve from context")
		http.Error(w, "unexpected server error, try to relogin", http.StatusInternalServerError)
		return
	}
	o, err := h.repo.FindByID(r.Context(), orderID)
	if err != nil && errors.Is(err, serviceerrs.ErrNotFound) {
		err = h.repo.Create(r.Context(), order.Order{
			CreatedAt: time.Now(),
			Status:    order.StatusNew,
			ID:        orderID,
			UserID:    userID,
		})
		if err != nil {
			h.logger.LogAttrs(r.Context(),
				slog.LevelError,
				"failed to create order",
				slog.String("order_id", orderID),
				slog.Any(model.KeyLoggerError, err),
			)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusAccepted)
		return
	} else if err != nil {
		h.logger.LogAttrs(r.Context(),
			slog.LevelError,
			"failed to find order by ID",
			slog.String("order_id", orderID),
			slog.Any(model.KeyLoggerError, err),
		)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if userID == o.UserID {
		w.WriteHeader(http.StatusOK)
		return
	}
	w.WriteHeader(http.StatusConflict)
}

func (h *OrderHandler) GetOrders(w http.ResponseWriter, r *http.Request) {}

func (h *TransactionHandler) GetBalance(w http.ResponseWriter, r *http.Request) {}

func (h *TransactionHandler) Withdraw(w http.ResponseWriter, r *http.Request) {}

func (h *TransactionHandler) GetWithdrawals(w http.ResponseWriter, r *http.Request) {}

func (h *GeneralHandler) Ping(w http.ResponseWriter, r *http.Request) {}
