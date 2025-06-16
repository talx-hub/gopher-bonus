package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
			http.StatusCreated,
			true,
		},
		{
			"happy test #2",
			`"login3"`,
			`"very-strong-password"`,
			http.StatusCreated,
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

func TestAuthHandler_Login_not_use_repo(t *testing.T) {
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
			"test unexpected OrderRepo Create failure",
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
		FindByID(mock.Anything, titleToOrderID["not-found"]).
		Return(order.Order{}, serviceerrs.ErrNotFound)
	orderRepo.EXPECT().
		FindByID(mock.Anything, titleToOrderID["found"]).
		Return(order.Order{
			ID:     "found",
			UserID: "correct-user",
		}, nil)
	orderRepo.EXPECT().
		FindByID(mock.Anything, titleToOrderID["break-the-order-create"]).
		Return(order.Order{}, serviceerrs.ErrNotFound)
	orderRepo.EXPECT().
		FindByID(mock.Anything, titleToOrderID["break-the-order-find"]).
		Return(order.Order{}, serviceerrs.ErrUnexpected)
	orderRepo.EXPECT().
		Create(mock.Anything, mock.Anything).
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

	orderRepo.AssertNotCalled(t, "FindByID", mock.Anything, "BAD")
	orderRepo.AssertNotCalled(t, "FindByID", mock.Anything,
		titleToOrderID["does-not-mater"])
	orderRepo.AssertNumberOfCalls(t, "FindByID", 5)
	orderRepo.AssertNumberOfCalls(t, "Create", 2)

	userRepo.AssertNumberOfCalls(t, "FindByID", 7)
}
