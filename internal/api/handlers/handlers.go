package handlers

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
	"unicode"

	"github.com/ShiraazMoollatjie/goluhn"
	"github.com/google/uuid"

	"github.com/talx-hub/gopher-bonus/internal/api/dto"
	"github.com/talx-hub/gopher-bonus/internal/model"
	"github.com/talx-hub/gopher-bonus/internal/model/order"
	"github.com/talx-hub/gopher-bonus/internal/model/user"
	"github.com/talx-hub/gopher-bonus/internal/service/dbmanager"
	"github.com/talx-hub/gopher-bonus/internal/serviceerrs"
	"github.com/talx-hub/gopher-bonus/internal/utils/auth"
)

const errRetrieveUserID = "failed retrieve userID from Ctx or check it with UserRepo"

type UserRepository interface {
	Create(ctx context.Context, u *user.User) error
	Exists(ctx context.Context, loginHash string) bool
	FindByLogin(ctx context.Context, loginHash string) (user.User, error)
	FindByID(ctx context.Context, id string) (user.User, error)
}

type AuthHandler struct {
	logger *slog.Logger
	repo   UserRepository
	secret string
}

func NewAuthHandler(userRepo UserRepository, log *slog.Logger, secret string) *AuthHandler {
	return &AuthHandler{
		logger: log,
		repo:   userRepo,
		secret: secret,
	}
}

type OrderRepository interface {
	CreateOrder(ctx context.Context, o *order.Order) error
	FindUserIDByAccrualID(ctx context.Context, accrualID string) (string, error)
	ListOrdersByUser(ctx context.Context, userID string, tp order.Type) ([]order.Order, error)
	UpdateAccrualStatus(ctx context.Context, o *order.Order) error
	GetBalance(ctx context.Context, userID string) (model.Amount, model.Amount, error)
}

type userRetriever struct{}

type OrderHandler struct {
	userRetriever
	logger    *slog.Logger
	orderRepo OrderRepository
	userRepo  UserRepository
}

func NewOrderHandler(
	userRepo UserRepository, orderRepo OrderRepository, log *slog.Logger,
) *OrderHandler {
	return &OrderHandler{
		logger:    log,
		orderRepo: orderRepo,
		userRepo:  userRepo,
	}
}

type HealthHandler struct {
	db *dbmanager.DBManager
}

func NewHealthHandler(db *dbmanager.DBManager) *HealthHandler {
	return &HealthHandler{db: db}
}

const failedReadBodyMsg = "failed to read the request body"
const failedCloseBodyMsg = "failed to close the request body"

func (r *userRetriever) retrieveUserID(ctx context.Context, userRepo UserRepository,
) (string, error) {
	userID, ok := ctx.Value(model.KeyContextUserID).(string)
	if !ok {
		return "", errors.New("failed retrieve UserID from context")
	}

	_, err := userRepo.FindByID(ctx, userID)
	if err != nil && errors.Is(err, serviceerrs.ErrNotFound) {
		return "", fmt.Errorf(
			"failed to find retrieved userID in UserRepo: %w", err)
	} else if err != nil {
		return "", fmt.Errorf("unexpected UserRepo failure: %w", err)
	}

	return userID, nil
}
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	data := dto.UserRequest{}
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		h.logger.LogAttrs(r.Context(),
			slog.LevelError,
			failedReadBodyMsg,
			slog.Any(model.KeyLoggerError, err),
		)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	err = r.Body.Close()
	if err != nil {
		h.logger.LogAttrs(r.Context(),
			slog.LevelError,
			failedCloseBodyMsg,
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
	w.WriteHeader(http.StatusOK)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	data := dto.UserRequest{}
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		h.logger.LogAttrs(r.Context(),
			slog.LevelError,
			failedReadBodyMsg,
			slog.Any(model.KeyLoggerError, err),
		)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	err = r.Body.Close()
	if err != nil {
		h.logger.LogAttrs(r.Context(),
			slog.LevelError,
			failedCloseBodyMsg,
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
			failedReadBodyMsg,
			slog.Any(model.KeyLoggerError, err),
		)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = r.Body.Close()
	if err != nil {
		h.logger.LogAttrs(r.Context(),
			slog.LevelError,
			failedCloseBodyMsg,
			slog.Any(model.KeyLoggerError, err),
		)
	}
	orderID := string(body)
	if err = goluhn.Validate(orderID); err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	for _, rn := range orderID {
		if !unicode.IsDigit(rn) {
			http.Error(w, "order ID must contain only digits", http.StatusBadRequest)
			return
		}
	}

	userID, err := h.retrieveUserID(r.Context(), h.userRepo)
	if err != nil {
		h.logger.LogAttrs(r.Context(),
			slog.LevelError,
			errRetrieveUserID,
			slog.Any(model.KeyLoggerError, err),
		)
		http.Error(w, serviceerrs.ErrUnexpected.Error(),
			http.StatusInternalServerError)
		return
	}

	foundUserID, err := h.orderRepo.FindUserIDByAccrualID(r.Context(), orderID)
	if err != nil && errors.Is(err, serviceerrs.ErrNotFound) {
		err = h.orderRepo.CreateOrder(r.Context(), &order.Order{
			CreatedAt: time.Now().UTC(),
			ID:        orderID,
			UserID:    userID,
			Status:    order.StatusNew,
			Type:      order.TypeAccrual,
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
	if userID == foundUserID {
		w.WriteHeader(http.StatusOK)
		return
	}
	w.WriteHeader(http.StatusConflict)
}

func (h *OrderHandler) GetOrders(w http.ResponseWriter, r *http.Request) {
	userID, err := h.retrieveUserID(r.Context(), h.userRepo)
	if err != nil {
		h.logger.LogAttrs(r.Context(),
			slog.LevelError,
			errRetrieveUserID,
			slog.Any(model.KeyLoggerError, err),
		)
		http.Error(w, serviceerrs.ErrUnexpected.Error(),
			http.StatusInternalServerError)
		return
	}

	orders, err := h.orderRepo.ListOrdersByUser(r.Context(), userID, order.TypeAccrual)
	if err != nil {
		h.logger.LogAttrs(r.Context(),
			slog.LevelError,
			"failed to find orders by userID",
			slog.Any(model.KeyLoggerError, err),
		)
		http.Error(w, serviceerrs.ErrUnexpected.Error(), http.StatusInternalServerError)
		return
	}
	if len(orders) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set(model.HeaderContentType, "application/json")
	if err = json.NewEncoder(w).Encode(orders); err != nil {
		h.logger.LogAttrs(r.Context(),
			slog.LevelError,
			"failed to write response",
			slog.Any(model.KeyLoggerError, err),
		)
		http.Error(w, serviceerrs.ErrUnexpected.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *OrderHandler) GetBalance(w http.ResponseWriter, r *http.Request) {
	userID, err := h.retrieveUserID(r.Context(), h.userRepo)
	if err != nil {
		h.logger.LogAttrs(r.Context(),
			slog.LevelError,
			errRetrieveUserID,
			slog.Any(model.KeyLoggerError, err),
		)
		http.Error(w, serviceerrs.ErrUnexpected.Error(),
			http.StatusInternalServerError)
		return
	}

	currentSum, withdrawals, err := h.orderRepo.GetBalance(r.Context(), userID)
	if err != nil {
		h.logger.LogAttrs(r.Context(),
			slog.LevelError,
			"failed to get bonus data",
			slog.Any(model.KeyLoggerError, err),
		)
		http.Error(w, serviceerrs.ErrUnexpected.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set(model.HeaderContentType, "application/json")
	if err = json.NewEncoder(w).Encode(
		dto.BalanceResponse{
			Current:   json.Number(currentSum.String()),
			Withdrawn: json.Number(withdrawals.String()),
		}); err != nil {
		h.logger.LogAttrs(r.Context(),
			slog.LevelError,
			"failed to write response",
			slog.Any(model.KeyLoggerError, err),
		)
		http.Error(w, serviceerrs.ErrUnexpected.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *OrderHandler) Withdraw(w http.ResponseWriter, r *http.Request) {
	userID, err := h.retrieveUserID(r.Context(), h.userRepo)
	if err != nil {
		h.logger.LogAttrs(r.Context(),
			slog.LevelError,
			errRetrieveUserID,
			slog.Any(model.KeyLoggerError, err),
		)
		http.Error(w, serviceerrs.ErrUnexpected.Error(),
			http.StatusInternalServerError)
		return
	}

	var request dto.WithdrawRequest
	if err = json.NewDecoder(r.Body).Decode(&request); err != nil {
		h.logger.LogAttrs(r.Context(),
			slog.LevelError,
			failedReadBodyMsg,
			slog.Any(model.KeyLoggerError, err),
		)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err = r.Body.Close(); err != nil {
		h.logger.LogAttrs(r.Context(),
			slog.LevelError,
			failedCloseBodyMsg,
			slog.Any(model.KeyLoggerError, err),
		)
	}

	amount, err := model.FromString(request.Sum.String())
	if err != nil {
		h.logger.LogAttrs(r.Context(),
			slog.LevelError,
			"failed to convert request Sum to bonus.Amount",
			slog.Any(model.KeyLoggerError, err),
		)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	err = h.orderRepo.CreateOrder(r.Context(),
		&order.Order{
			ID:        request.OrderID,
			UserID:    userID,
			CreatedAt: time.Now().UTC(),
			Type:      order.TypeWithdrawal,
			Amount:    amount,
		},
	)
	if err == nil {
		w.WriteHeader(http.StatusOK)
	}

	if errors.Is(err, serviceerrs.ErrInsufficientFunds) {
		h.logger.LogAttrs(r.Context(),
			slog.LevelError,
			"insufficient funds",
			slog.String("order", request.OrderID),
			slog.String("requested", request.Sum.String()),
			slog.Any(model.KeyLoggerError, err),
		)
		http.Error(w, err.Error(), http.StatusPaymentRequired)
		return
	}
	h.logger.LogAttrs(r.Context(),
		slog.LevelError,
		"unexpected withdrawal error",
		slog.String("order", request.OrderID),
		slog.String("requested", request.Sum.String()),
		slog.Any(model.KeyLoggerError, err),
	)
	http.Error(w, serviceerrs.ErrUnexpected.Error(), http.StatusInternalServerError)
}

func (h *OrderHandler) GetWithdrawals(w http.ResponseWriter, r *http.Request) {
	userID, err := h.retrieveUserID(r.Context(), h.userRepo)
	if err != nil {
		h.logger.LogAttrs(r.Context(),
			slog.LevelError,
			errRetrieveUserID,
			slog.Any(model.KeyLoggerError, err),
		)
		http.Error(w, serviceerrs.ErrUnexpected.Error(),
			http.StatusInternalServerError)
		return
	}

	withdrawals, err := h.orderRepo.ListOrdersByUser(
		r.Context(), userID, order.TypeWithdrawal)
	if err != nil && errors.Is(err, serviceerrs.ErrNotFound) {
		h.logger.LogAttrs(r.Context(),
			slog.LevelError,
			"failed to find userID and it's withdrawals",
			slog.String("user_id", userID),
			slog.Any(model.KeyLoggerError, err),
		)
		http.Error(w, serviceerrs.ErrUnexpected.Error(), http.StatusInternalServerError)
		return
	} else if err != nil {
		h.logger.LogAttrs(r.Context(),
			slog.LevelError,
			"unexpected bonus Repo error",
			slog.String("user_id", userID),
			slog.Any(model.KeyLoggerError, err),
		)
		http.Error(w, serviceerrs.ErrUnexpected.Error(), http.StatusInternalServerError)
		return
	}
	if len(withdrawals) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set(model.HeaderContentType, "application/json")
	if err = json.NewEncoder(w).Encode(withdrawals); err != nil {
		h.logger.LogAttrs(r.Context(),
			slog.LevelError,
			"failed to write response",
			slog.Any(model.KeyLoggerError, err),
		)
		http.Error(w, serviceerrs.ErrUnexpected.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *HealthHandler) Ping(w http.ResponseWriter, r *http.Request) {
	h.db.Ping(r.Context())
	if err := h.db.Error(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
