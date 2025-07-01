package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/talx-hub/gopher-bonus/internal/api/handlers/mocks"
	"github.com/talx-hub/gopher-bonus/internal/model"
	"github.com/talx-hub/gopher-bonus/internal/model/order"
	"github.com/talx-hub/gopher-bonus/internal/model/user"
	"github.com/talx-hub/gopher-bonus/internal/serviceerrs"
)

func testAuthHandlers(t *testing.T,
	endpoint string,
	handlerFunc http.HandlerFunc,
	login, password string,
	wantToken bool,
	wantCode int,
) {
	t.Helper()

	reqBody := fmt.Sprintf(`{"login":%s, "password":%s}`,
		login, password)
	req := httptest.NewRequest(
		http.MethodPost, endpoint, strings.NewReader(reqBody))
	rr := httptest.NewRecorder()
	handlerFunc(rr, req)

	res := rr.Result()
	err := res.Body.Close()
	require.NoError(t, err)

	const cookieName = "jwt-token"
	hasToken := false
	for _, c := range res.Cookies() {
		if c.Name == cookieName && len(c.Value) != 0 {
			hasToken = true
			break
		}
	}

	assert.Equal(t, wantToken, hasToken)
	assert.Equal(t, wantCode, rr.Code)
}

type ResponseFixture struct {
	TestcaseName string          `json:"name"`
	Responses    json.RawMessage `json:"responses"`
}

func loadResponseFixtures(t *testing.T, file string) map[string]string {
	t.Helper()

	data, err := os.ReadFile(file)
	require.NoError(t, err)

	var temp []ResponseFixture
	require.NoError(t, json.Unmarshal(data, &temp))

	fixtures := make(map[string]string)
	for _, f := range temp {
		fixtures[f.TestcaseName] = string(f.Responses)
	}
	return fixtures
}

func TestAuthHandler_Register(t *testing.T) {
	tests := []struct {
		name      string
		login     string
		password  string
		wantCode  int
		wantToken bool
	}{
		{
			"empty login",
			`""`,
			`"very-strong-password"`,
			http.StatusBadRequest,
			false,
		},
		{
			"empty password",
			`"login0"`,
			`""`,
			http.StatusBadRequest,
			false,
		},
		{
			"empty login and password",
			`""`,
			`""`,
			http.StatusBadRequest,
			false,
		},
		{
			"weak password",
			`"login1"`,
			`"password"`,
			http.StatusBadRequest,
			false,
		},
		{
			"happy test #1",
			`"login2"`,
			`"very-strong-password"`,
			http.StatusOK,
			true,
		},
		{
			"happy test #2",
			`"login3"`,
			`"very-strong-password"`,
			http.StatusOK,
			true,
		},
		{
			"conflict",
			`"login2"`,
			`"very-strong-password"`,
			http.StatusConflict,
			false,
		},
		{
			"decoding error #1",
			`42`,
			`"very-strong-password"`,
			http.StatusBadRequest,
			false,
		},
		{
			"decoding error #2",
			`login4`,
			`3.14`,
			http.StatusBadRequest,
			false,
		},
		{
			"decoding error #3",
			`42`,
			`3.14`,
			http.StatusBadRequest,
			false,
		},
	}

	repo := mocks.NewMockUserRepository(t)
	callCounts := make(map[string]int)
	repo.EXPECT().
		Exists(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, login string) bool {
			callCounts[login]++
			count := callCounts[login]
			return count != 1
		})

	repo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil)

	authHandler := AuthHandler{
		logger: slog.Default(),
		repo:   repo,
		secret: "super-secret-key",
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testAuthHandlers(t,
				"/register",
				authHandler.Register,
				tt.login, tt.password,
				tt.wantToken, tt.wantCode)
		})
	}
}

func TestAuthHandler_Login(t *testing.T) {
	tests := []struct {
		name      string
		login     string
		password  string
		wantCode  int
		wantToken bool
	}{
		{
			"not existing user",
			`"login-not-exist"`,
			`"very-strong-password"`,
			http.StatusUnauthorized,
			false,
		},
		{
			"happy test #1",
			`"login1"`,
			`"very-strong-password"`,
			http.StatusOK,
			true,
		},
		{
			"happy test #2",
			`"login2"`,
			`"another-very-strong-password"`,
			http.StatusOK,
			true,
		},
		{
			"happy test #3",
			`"login3"`,
			`"another-very-strong-password"`,
			http.StatusOK,
			true,
		},
		{
			"wrong password #1",
			`"login3"`,
			`"another-very-WRONG-password"`,
			http.StatusUnauthorized,
			false,
		},
		{
			"wrong password #2",
			`"login3"`,
			`""`,
			http.StatusUnauthorized,
			false,
		},
	}

	hashes := map[string]string{
		"login-not-exist":              "8fd7f50c0d3558bd71df30eacde81d4d934baaab17513056bcfe41cc5e651faf",
		"login1":                       "7c8f0a693377b5f088145213e32fdfe1f48289599eea6f8af25c0445089cd875",
		"login2":                       "d7100492c03a237d810dfb65048c4a4311f879738aed63c72cc77b7b79d9ac0b",
		"login3":                       "7ac377fd43f82caf7408a581acaf1f29a90e00a3f717876966a282d07101810e",
		"very-strong-password":         "3f60d8ef18e0a446cab83d597b9ebe52d2ad0e45b720cce8466dfd29ab22c7e0",
		"another-very-strong-password": "c3914823f9d3d87225df6062220294bb965398822edce4a8a6969cabec3a6b04",
	}

	repo := mocks.NewMockUserRepository(t)
	repo.EXPECT().
		FindByLogin(mock.Anything, hashes["login-not-exist"]).
		Return(user.User{}, serviceerrs.ErrNotFound)

	repo.EXPECT().
		FindByLogin(mock.Anything, hashes["login1"]).
		Return(
			user.User{
				LoginHash:    hashes["login1"],
				PasswordHash: hashes["very-strong-password"],
			},
			nil)

	repo.EXPECT().
		FindByLogin(mock.Anything, hashes["login2"]).
		Return(
			user.User{
				LoginHash:    hashes["login2"],
				PasswordHash: hashes["another-very-strong-password"],
			},
			nil)

	repo.EXPECT().
		FindByLogin(mock.Anything, hashes["login3"]).
		Return(
			user.User{
				LoginHash:    hashes["login3"],
				PasswordHash: hashes["another-very-strong-password"],
			},
			nil)

	authHandler := AuthHandler{
		logger: slog.Default(),
		repo:   repo,
		secret: "super-secret-key",
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testAuthHandlers(t,
				"/login",
				authHandler.Login,
				tt.login, tt.password,
				tt.wantToken, tt.wantCode)
		})
	}
}

func TestAuthHandler_Login_bad_credentials(t *testing.T) {
	tests := []struct {
		name      string
		login     string
		password  string
		wantCode  int
		wantToken bool
	}{
		{
			"empty login",
			`""`,
			`"very-strong-password"`,
			http.StatusUnauthorized,
			false,
		},
		{
			"empty password",
			`"login0"`,
			`""`,
			http.StatusUnauthorized,
			false,
		},
		{
			"empty login and password",
			`""`,
			`""`,
			http.StatusUnauthorized,
			false,
		},
		{
			"decoding error #1",
			`42`,
			`"very-strong-password"`,
			http.StatusBadRequest,
			false,
		},
		{
			"decoding error #2",
			`login4`,
			`3.14`,
			http.StatusBadRequest,
			false,
		},
		{
			"decoding error #3",
			`42`,
			`3.14`,
			http.StatusBadRequest,
			false,
		},
	}

	repo := mocks.NewMockUserRepository(t)
	authHandler := AuthHandler{
		logger: slog.Default(),
		repo:   repo,
		secret: "super-secret-key",
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testAuthHandlers(t,
				"/login",
				authHandler.Login,
				tt.login, tt.password,
				tt.wantToken, tt.wantCode)
		})
	}
	repo.AssertNotCalled(
		t, "FindByLogin", mock.Anything, mock.Anything)
}

func TestOrderHandler_PostOrder(t *testing.T) {
	titleToOrderID := map[string]string{
		"not-found":              "1",
		"found":                  "2",
		"does-not-matter":        "3",
		"break-the-order-create": "4",
		"break-the-order-find":   "5",
	}
	tests := []struct {
		name     string
		userID   string
		body     string
		wantCode int
	}{
		{
			"test Accepted",
			"user",
			titleToOrderID["not-found"],
			http.StatusAccepted,
		},
		{
			"test OK",
			"correct-user",
			titleToOrderID["found"],
			http.StatusOK,
		},
		{
			"test Conflict",
			"wrong-user",
			titleToOrderID["found"],
			http.StatusConflict,
		},
		{
			"test retrieve userID from context failure",
			"dont-put-to-ctx",
			titleToOrderID["does-not-mater"],
			http.StatusInternalServerError,
		},
		{
			"test find userID in UserRepo failure",
			"user-NOT-exist",
			titleToOrderID["does-not-mater"],
			http.StatusInternalServerError,
		},
		{
			"test unexpected UserRepo failure",
			"break-the-user-repo",
			titleToOrderID["does-not-mater"],
			http.StatusInternalServerError,
		},
		{
			"test unexpected OrderRepo CreateOrder failure",
			"user",
			titleToOrderID["break-the-order-create"],
			http.StatusInternalServerError,
		},
		{
			"test unexpected OrderRepo Find failure",
			"user",
			titleToOrderID["break-the-order-find"],
			http.StatusInternalServerError,
		},
		{
			"test bad orderID",
			"user",
			"BAD",
			http.StatusBadRequest,
		},
	}

	orderRepo := mocks.NewMockOrderRepository(t)
	orderRepo.EXPECT().
		FindUserIDByAccrualID(mock.Anything, titleToOrderID["not-found"]).
		Return("", serviceerrs.ErrNotFound)
	orderRepo.EXPECT().
		FindUserIDByAccrualID(mock.Anything, titleToOrderID["found"]).
		Return("correct-user", nil)
	orderRepo.EXPECT().
		FindUserIDByAccrualID(mock.Anything, titleToOrderID["break-the-order-create"]).
		Return("", serviceerrs.ErrNotFound)
	orderRepo.EXPECT().
		FindUserIDByAccrualID(mock.Anything, titleToOrderID["break-the-order-find"]).
		Return("", serviceerrs.ErrUnexpected)
	orderRepo.EXPECT().
		CreateOrder(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, o *order.Order) error {
			if o.ID == titleToOrderID["break-the-order-create"] {
				return serviceerrs.ErrUnexpected
			}
			return nil
		})

	userRepo := mocks.NewMockUserRepository(t)
	userRepo.EXPECT().
		FindByID(mock.Anything, "user").
		Return(user.User{}, nil)
	userRepo.EXPECT().
		FindByID(mock.Anything, "correct-user").
		Return(user.User{}, nil)
	userRepo.EXPECT().
		FindByID(mock.Anything, "wrong-user").
		Return(user.User{}, nil)
	userRepo.EXPECT().
		FindByID(mock.Anything, "user-NOT-exist").
		Return(user.User{}, serviceerrs.ErrNotFound)
	userRepo.EXPECT().
		FindByID(mock.Anything, "break-the-user-repo").
		Return(user.User{}, serviceerrs.ErrUnexpected)
	userRepo.EXPECT().
		FindByID(mock.Anything, mock.Anything).
		Return(user.User{}, nil)

	h := OrderHandler{
		logger:    slog.Default(),
		orderRepo: orderRepo,
		userRepo:  userRepo,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(
				http.MethodPost, "/order", strings.NewReader(tt.body))
			if tt.userID != "dont-put-to-ctx" {
				userIDCtx := context.WithValue(
					req.Context(), model.KeyContextUserID, tt.userID)
				req = req.WithContext(userIDCtx)
			}
			rr := httptest.NewRecorder()
			h.PostOrder(rr, req)
			res := rr.Result()
			err := res.Body.Close()
			require.NoError(t, err)

			assert.Equal(t, tt.wantCode, res.StatusCode)
		})
	}

	orderRepo.AssertNotCalled(t, "FindUserIDByAccrualID", mock.Anything, "BAD")
	orderRepo.AssertNotCalled(t, "FindUserIDByAccrualID", mock.Anything,
		titleToOrderID["does-not-mater"])
	orderRepo.AssertNumberOfCalls(t, "FindUserIDByAccrualID", 5)
	orderRepo.AssertNumberOfCalls(t, "CreateOrder", 2)

	userRepo.AssertNumberOfCalls(t, "FindByID", 7)
}

func TestOrderHandler_GetOrders(t *testing.T) {
	time1, err := time.Parse(time.RFC3339, "1999-01-01T00:00:00Z")
	require.NoError(t, err)
	time2, err := time.Parse(time.RFC3339, "2025-06-21T11:58:45+03:00")
	require.NoError(t, err)
	time3, err := time.Parse(time.RFC3339, "2025-06-25T00:00:00Z")
	require.NoError(t, err)
	time4, err := time.Parse(time.RFC3339, "2025-06-26T03:00:00Z")
	require.NoError(t, err)

	wantResponses := loadResponseFixtures(t, "testdata/get_orders_response.json")

	tests := []struct {
		name                 string
		userID               string
		mockRetrieveUserID   func() (user.User, error)
		mockListOrdersByUser func() ([]order.Order, error)
		wantCode             int
		resp                 string
	}{
		{
			name:   "successful get orders",
			userID: "user-1",
			mockRetrieveUserID: func() (user.User, error) {
				return user.User{ID: "user-1"}, nil
			},
			mockListOrdersByUser: func() ([]order.Order, error) {
				return []order.Order{
					{
						ID:        "1",
						UserID:    "user-1",
						Type:      order.TypeAccrual,
						Status:    order.StatusProcessed,
						Amount:    model.NewAmount(100, 500),
						CreatedAt: time1,
					},
					{
						ID:        "2",
						UserID:    "user-1",
						Type:      order.TypeAccrual,
						Status:    order.StatusProcessed,
						Amount:    model.NewAmount(100, 5),
						CreatedAt: time2,
					},
					{
						ID:        "3",
						UserID:    "user-1",
						Type:      order.TypeAccrual,
						Status:    order.StatusProcessed,
						Amount:    model.NewAmount(100, 51),
						CreatedAt: time3,
					},
					{
						ID:        "4",
						UserID:    "user-1",
						Type:      order.TypeAccrual,
						Status:    order.StatusProcessed,
						Amount:    model.NewAmount(100, 99),
						CreatedAt: time4,
					},
					{
						ID:        "5",
						UserID:    "user-1",
						Type:      order.TypeAccrual,
						Status:    order.StatusProcessed,
						Amount:    model.NewAmount(100, 1),
						CreatedAt: time1,
					},
					{
						ID:        "6",
						UserID:    "user-1",
						Type:      order.TypeAccrual,
						Status:    order.StatusProcessed,
						Amount:    model.NewAmount(100, 100),
						CreatedAt: time2,
					},
					{ID: "7", UserID: "user-1", Type: order.TypeAccrual, Status: order.StatusProcessing, CreatedAt: time3},
					{ID: "8", UserID: "user-1", Type: order.TypeAccrual, Status: order.StatusNew, CreatedAt: time4},
					{ID: "9", UserID: "user-1", Type: order.TypeAccrual, Status: order.StatusInvalid, CreatedAt: time1},
				}, nil
			},
			wantCode: http.StatusOK,
			resp:     wantResponses["successful get orders"],
		},
		{
			name:   "user without orders",
			userID: "user-2",
			mockRetrieveUserID: func() (user.User, error) {
				return user.User{ID: "user-2"}, nil
			},
			mockListOrdersByUser: func() ([]order.Order, error) {
				return []order.Order{}, nil
			},
			wantCode: http.StatusNoContent,
		},
		{
			name:   "failListOrdersByUser",
			userID: "user-3",
			mockRetrieveUserID: func() (user.User, error) {
				return user.User{ID: "user-3"}, nil
			},
			mockListOrdersByUser: func() ([]order.Order, error) {
				return nil, serviceerrs.ErrUnexpected
			},
			wantCode: http.StatusInternalServerError,
		},
		{
			name:   "fail encoding to JSON: empty type",
			userID: "user-4",
			mockRetrieveUserID: func() (user.User, error) {
				return user.User{ID: "user-4"}, nil
			},
			mockListOrdersByUser: func() ([]order.Order, error) {
				return []order.Order{
					{ID: "unknown type", UserID: "user-4", Type: "unknown"},
					{ID: "valid", UserID: "user-4", Type: order.TypeAccrual, Status: order.StatusNew},
				}, nil
			},
			wantCode: http.StatusInternalServerError,
		},
		{
			name:   "unknown user",
			userID: "unknown",
			mockRetrieveUserID: func() (user.User, error) {
				return user.User{}, serviceerrs.ErrNotFound
			},
			wantCode: http.StatusInternalServerError,
		},
		{
			name:     "middleware failure: no user in ctx",
			userID:   "dont-put-to-ctx",
			wantCode: http.StatusInternalServerError,
		},
	}

	userRepo := mocks.NewMockUserRepository(t)
	orderRepo := mocks.NewMockOrderRepository(t)

	h := OrderHandler{
		userRetriever: userRetriever{},
		logger:        slog.Default(),
		orderRepo:     orderRepo,
		userRepo:      userRepo,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.mockRetrieveUserID != nil {
				u, err := tt.mockRetrieveUserID()
				userRepo.EXPECT().
					FindByID(mock.Anything, tt.userID).
					Return(u, err)
			} else {
				userRepo.EXPECT().
					FindByID(mock.Anything, mock.Anything).
					Times(0)
			}

			if tt.mockListOrdersByUser != nil {
				orders, err := tt.mockListOrdersByUser()
				orderRepo.EXPECT().
					ListOrdersByUser(mock.Anything, tt.userID, order.TypeAccrual).
					Return(orders, err)
			} else {
				orderRepo.EXPECT().
					ListOrdersByUser(mock.Anything, mock.Anything, mock.Anything).
					Times(0)
			}

			req := httptest.NewRequest(
				http.MethodGet, "/orders", http.NoBody)
			if tt.userID != "dont-put-to-ctx" {
				userIDCtx := context.WithValue(
					req.Context(), model.KeyContextUserID, tt.userID)
				req = req.WithContext(userIDCtx)
			}
			rr := httptest.NewRecorder()
			h.GetOrders(rr, req)
			res := rr.Result()

			assert.Equal(t, tt.wantCode, res.StatusCode)
			if tt.wantCode == http.StatusOK {
				body, err := io.ReadAll(res.Body)
				require.NoError(t, err)
				err = res.Body.Close()
				require.NoError(t, err)
				assert.JSONEq(t, tt.resp, string(body))
			}
		})
	}
}

func TestOrderHandler_GetBalance(t *testing.T) {
	tests := []struct {
		name               string
		userID             string
		mockRetrieveUserID func() (user.User, error)
		mockGetBalance     func() (model.Amount, model.Amount, error)
		wantCode           int
		resp               string
	}{
		{
			name:   "successful get balance #1",
			userID: "user-1",
			mockRetrieveUserID: func() (user.User, error) {
				return user.User{ID: "user-1"}, nil
			},
			mockGetBalance: func() (model.Amount, model.Amount, error) {
				return model.NewAmount(0, 50050),
					model.NewAmount(0, 4200),
					nil
			},
			wantCode: http.StatusOK,
			resp:     `{"current": 500.5,"withdrawn": 42}`,
		},
		{
			name:   "successful get balance #2",
			userID: "user-2",
			mockRetrieveUserID: func() (user.User, error) {
				return user.User{ID: "user-2"}, nil
			},
			mockGetBalance: func() (model.Amount, model.Amount, error) {
				return model.NewAmount(0, 0),
					model.NewAmount(0, 1),
					nil
			},
			wantCode: http.StatusOK,
			resp:     `{"current": 0.0,"withdrawn": 0.01}`,
		},
		{
			name:   "fail to get balance",
			userID: "user-3",
			mockRetrieveUserID: func() (user.User, error) {
				return user.User{ID: "user-2"}, nil
			},
			mockGetBalance: func() (model.Amount, model.Amount, error) {
				return model.NewAmount(0, 0),
					model.NewAmount(0, 0),
					serviceerrs.ErrUnexpected
			},
			wantCode: http.StatusInternalServerError,
		},
		{
			name:   "unknown user",
			userID: "unknown",
			mockRetrieveUserID: func() (user.User, error) {
				return user.User{}, serviceerrs.ErrNotFound
			},
			wantCode: http.StatusInternalServerError,
		},
		{
			name:     "middleware failure: no user in ctx",
			userID:   "dont-put-to-ctx",
			wantCode: http.StatusInternalServerError,
		},
	}

	userRepo := mocks.NewMockUserRepository(t)
	orderRepo := mocks.NewMockOrderRepository(t)

	h := OrderHandler{
		userRetriever: userRetriever{},
		logger:        slog.Default(),
		orderRepo:     orderRepo,
		userRepo:      userRepo,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.mockRetrieveUserID != nil {
				u, err := tt.mockRetrieveUserID()
				userRepo.EXPECT().
					FindByID(mock.Anything, tt.userID).
					Return(u, err)
			} else {
				userRepo.EXPECT().
					FindByID(mock.Anything, mock.Anything).
					Times(0)
			}

			if tt.mockGetBalance != nil {
				currentSum, withdrawals, err := tt.mockGetBalance()
				orderRepo.EXPECT().
					GetBalance(mock.Anything, tt.userID).
					Return(currentSum, withdrawals, err)
			} else {
				orderRepo.EXPECT().
					GetBalance(mock.Anything, mock.Anything).
					Times(0)
			}

			req := httptest.NewRequest(
				http.MethodGet, "/orders", http.NoBody)
			if tt.userID != "dont-put-to-ctx" {
				userIDCtx := context.WithValue(
					req.Context(), model.KeyContextUserID, tt.userID)
				req = req.WithContext(userIDCtx)
			}
			rr := httptest.NewRecorder()
			h.GetBalance(rr, req)
			res := rr.Result()

			assert.Equal(t, tt.wantCode, res.StatusCode)
			if tt.wantCode == http.StatusOK {
				body, err := io.ReadAll(res.Body)
				require.NoError(t, err)
				err = res.Body.Close()
				require.NoError(t, err)
				assert.JSONEq(t, tt.resp, string(body))
			}
		})
	}
}

func TestOrderHandler_Withdraw(t *testing.T) {
	tests := []struct {
		name               string
		userID             string
		body               string
		mockRetrieveUserID func() (user.User, error)
		wantCode           int
	}{
		{
			name:   "success withdrawal",
			userID: "user-1",
			body:   `{"order": "success withdrawal", "sum": 751.15}`,
			mockRetrieveUserID: func() (user.User, error) {
				return user.User{ID: "user-1"}, nil
			},
			wantCode: http.StatusOK,
		},
		{
			name:   "bad json",
			userID: "user-2",
			body:   `{"order": "2377225624", "sum":}`,
			mockRetrieveUserID: func() (user.User, error) {
				return user.User{ID: "user-2"}, nil
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name:   "invalid amount format",
			userID: "user-3",
			body:   `{"order": "2377225624", "sum": "not-a-number"}`,
			mockRetrieveUserID: func() (user.User, error) {
				return user.User{ID: "user-3"}, nil
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name:   "insufficient funds",
			userID: "user-4",
			body:   `{"order": "insufficient funds", "sum": 751.15}`,
			mockRetrieveUserID: func() (user.User, error) {
				return user.User{ID: "user-4"}, nil
			},
			wantCode: http.StatusPaymentRequired,
		},
		{
			name:   "unexpected repo error",
			userID: "user-5",
			body:   `{"order": "unexpected repo error", "sum": 751.15}`,
			mockRetrieveUserID: func() (user.User, error) {
				return user.User{ID: "user-5"}, nil
			},
			wantCode: http.StatusInternalServerError,
		},
		{
			name:   "retrieve user error",
			userID: "user-6",
			body:   `{"order": "2377225624", "sum": 751.15}`,
			mockRetrieveUserID: func() (user.User, error) {
				return user.User{}, serviceerrs.ErrNotFound
			},
			wantCode: http.StatusInternalServerError,
		},
		{
			name:               "missing user in context",
			userID:             "dont-put-to-ctx",
			body:               `{"order": "2377225624", "sum": 751.15}`,
			mockRetrieveUserID: nil,
			wantCode:           http.StatusInternalServerError,
		},
	}

	userRepo := mocks.NewMockUserRepository(t)
	orderRepo := mocks.NewMockOrderRepository(t)
	orderRepo.EXPECT().
		CreateOrder(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, o *order.Order) error {
			if o.ID == "insufficient funds" {
				return serviceerrs.ErrInsufficientFunds
			}
			if o.ID == "unexpected repo error" {
				return serviceerrs.ErrUnexpected
			}
			return nil
		})

	h := OrderHandler{
		userRetriever: userRetriever{},
		logger:        slog.Default(),
		orderRepo:     orderRepo,
		userRepo:      userRepo,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.mockRetrieveUserID != nil {
				u, err := tt.mockRetrieveUserID()
				userRepo.EXPECT().
					FindByID(mock.Anything, tt.userID).
					Return(u, err)
			} else {
				userRepo.EXPECT().
					FindByID(mock.Anything, mock.Anything).
					Times(0)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/user/balance/withdraw", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")

			if tt.userID != "dont-put-to-ctx" {
				userIDCtx := context.WithValue(req.Context(), model.KeyContextUserID, tt.userID)
				req = req.WithContext(userIDCtx)
			}

			rr := httptest.NewRecorder()
			h.Withdraw(rr, req)
			res := rr.Result()
			err := res.Body.Close()
			require.NoError(t, err)

			assert.Equal(t, tt.wantCode, res.StatusCode)
		})
	}
	orderRepo.AssertNumberOfCalls(t, "CreateOrder", 3)
}

func TestOrderHandler_GetWithdrawals(t *testing.T) {
	time1, err := time.Parse(time.RFC3339, "2025-06-21T11:58:45+03:00")
	require.NoError(t, err)
	time2, err := time.Parse(time.RFC3339, "2025-06-25T00:00:00Z")
	require.NoError(t, err)
	tests := []struct {
		name         string
		userID       string
		wantCode     int
		wantResponse string
	}{
		{
			name:     "success withdrawals list",
			userID:   "user-1",
			wantCode: http.StatusOK,
			wantResponse: `[
{"order":"w1","sum":"300.00","processed_at":"2025-06-21T11:58:45+03:00"}
{"order":"w2","sum":"1","processed_at":"2025-06-25T03:00:00+03:00"}
]`,
		},
		{
			name:     "no withdrawals",
			userID:   "no withdrawals",
			wantCode: http.StatusNoContent,
		},
		{
			name:     "not found in repository",
			userID:   "not found in repository",
			wantCode: http.StatusInternalServerError,
		},
		{
			name:     "unexpected repo error",
			userID:   "unexpected repo error",
			wantCode: http.StatusInternalServerError,
		},
		{
			name:     "missing user in context",
			userID:   "dont-put-to-ctx",
			wantCode: http.StatusInternalServerError,
		},
		{
			name:     "repo findByID error",
			userID:   "user-6",
			wantCode: http.StatusInternalServerError,
		},
	}

	userRepo := mocks.NewMockUserRepository(t)
	userRepo.EXPECT().
		FindByID(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, id string) (user.User, error) {
			if id == "user-6" {
				return user.User{}, serviceerrs.ErrNotFound
			}
			return user.User{ID: id}, nil
		})

	orderRepo := mocks.NewMockOrderRepository(t)
	orderRepo.EXPECT().
		ListOrdersByUser(mock.Anything, mock.Anything, order.TypeWithdrawal).
		RunAndReturn(func(_ context.Context, userID string, _ order.Type) ([]order.Order, error) {
			if userID == "no withdrawals" {
				return []order.Order{}, nil
			}
			if userID == "not found in repository" {
				return nil, serviceerrs.ErrNotFound
			}
			if userID == "unexpected repo error" {
				return nil, serviceerrs.ErrUnexpected
			}

			return []order.Order{
				{
					ID:        "w1",
					UserID:    "user-1",
					Type:      order.TypeWithdrawal,
					Amount:    model.NewAmount(300, 0),
					CreatedAt: time1,
				},
				{
					ID:        "w2",
					UserID:    "user-1",
					Type:      order.TypeWithdrawal,
					Amount:    model.NewAmount(0, 100),
					CreatedAt: time2,
				},
			}, nil
		})

	h := OrderHandler{
		userRetriever: userRetriever{},
		logger:        slog.Default(),
		orderRepo:     orderRepo,
		userRepo:      userRepo,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/user/balance/withdraw", http.NoBody)
			req.Header.Set("Content-Type", "application/json")

			if tt.userID != "dont-put-to-ctx" {
				userIDCtx := context.WithValue(req.Context(), model.KeyContextUserID, tt.userID)
				req = req.WithContext(userIDCtx)
			}

			rr := httptest.NewRecorder()
			h.GetWithdrawals(rr, req)
			res := rr.Result()
			err = res.Body.Close()
			require.NoError(t, err)

			assert.Equal(t, tt.wantCode, res.StatusCode)
		})
	}
	orderRepo.AssertNumberOfCalls(t, "ListOrdersByUser", 4)
}
