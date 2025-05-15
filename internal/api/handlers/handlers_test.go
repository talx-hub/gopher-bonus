package handlers

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/talx-hub/gopher-bonus/internal/api/handlers/mocks"
)

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
	const loginIdx = 1
	repo.
		On("Exists", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			login := args.String(loginIdx)
			callCounts[login]++
		}).
		Return(func(_ context.Context, login string) bool {
			count := callCounts[login]
			return count != 1
		}).
		On("Create", mock.Anything, mock.Anything).
		Return(nil)

	authHandler := AuthHandler{
		repo:   repo,
		secret: "super-secret-key",
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqBody := fmt.Sprintf(`{"login":%s, "password":%s}`,
				tt.login, tt.password)
			req := httptest.NewRequest(
				http.MethodPost, "/register", strings.NewReader(reqBody))
			rr := httptest.NewRecorder()
			authHandler.Register(rr, req)

			res := rr.Result()
			if err := res.Body.Close(); err != nil {
				require.NoError(t, err)
			}

			const cookieName = "jwt-token"
			hasToken := false
			for _, c := range res.Cookies() {
				if c.Name == cookieName && len(c.Value) != 0 {
					hasToken = true
					break
				}
			}

			assert.Equal(t, tt.wantToken, hasToken)
			assert.Equal(t, tt.wantCode, rr.Code)
		})
	}
}
