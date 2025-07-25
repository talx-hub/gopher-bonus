package router

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/talx-hub/gopher-bonus/internal/service/config"
	"github.com/talx-hub/gopher-bonus/internal/utils/auth"
)

type stubHandler struct {
	name string
}

func (s stubHandler) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("X-Handler", s.name)
	w.WriteHeader(http.StatusTeapot)
}

type h struct{}

func (h) Register(w http.ResponseWriter, r *http.Request) {
	stubHandler{name: "register"}.ServeHTTP(w, r)
}

func (h) Login(w http.ResponseWriter, r *http.Request) {
	stubHandler{name: "login"}.ServeHTTP(w, r)
}
func (h) GetOrders(w http.ResponseWriter, r *http.Request) {
	stubHandler{name: "get_orders"}.ServeHTTP(w, r)
}
func (h) PostOrder(w http.ResponseWriter, r *http.Request) {
	stubHandler{name: "post_order"}.ServeHTTP(w, r)
}
func (h) GetBalance(w http.ResponseWriter, r *http.Request) {
	stubHandler{name: "get_balance"}.ServeHTTP(w, r)
}
func (h) Withdraw(w http.ResponseWriter, r *http.Request) {
	stubHandler{name: "withdraw"}.ServeHTTP(w, r)
}
func (h) GetWithdrawals(w http.ResponseWriter, r *http.Request) {
	stubHandler{name: "get_withdrawals"}.ServeHTTP(w, r)
}
func (h) Ping(w http.ResponseWriter, r *http.Request) {
	stubHandler{name: "ping"}.ServeHTTP(w, r)
}

func TestCustomRouter_Route_happyTests(t *testing.T) {
	tests := []struct {
		method   string
		path     string
		wantName string
		wantCode int
	}{
		{http.MethodPost, "/api/user/register", "register", http.StatusTeapot},
		{http.MethodPost, "/api/user/login", "login", http.StatusTeapot},
		{http.MethodGet, "/api/user/orders", "get_orders", http.StatusTeapot},
		{http.MethodPost, "/api/user/orders", "post_order", http.StatusTeapot},
		{http.MethodGet, "/api/user/balance", "get_balance", http.StatusTeapot},
		{http.MethodPost, "/api/user/balance/withdraw", "withdraw", http.StatusTeapot},
		{http.MethodGet, "/api/user/withdrawals", "get_withdrawals", http.StatusTeapot},
		{http.MethodGet, "/ping", "ping", http.StatusTeapot},
	}

	r := New(&config.Config{}, slog.Default())
	r.SetRouter(h{})
	srv := httptest.NewServer(r.GetRouter())
	defer srv.Close()

	for _, tt := range tests {
		req, err := http.NewRequest(tt.method, srv.URL+tt.path, http.NoBody)
		require.NoError(t, err)
		jwtCookie, err := auth.Authenticate("id", []byte(""))
		require.NoError(t, err)
		req.AddCookie(&jwtCookie)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		err = resp.Body.Close()
		require.NoError(t, err)

		assert.Equal(t, tt.wantCode, resp.StatusCode)
		assert.Equal(t, tt.wantName, resp.Header.Get("X-Handler"))
	}
}

func TestCustomRouter_Route_wrong_routes(t *testing.T) {
	r := New(&config.Config{}, slog.Default())
	r.SetRouter(h{})
	srv := httptest.NewServer(r.GetRouter())
	defer srv.Close()

	tests := []struct {
		method   string
		path     string
		wantCode int
	}{
		{http.MethodPost, "/", http.StatusNotFound},
		{http.MethodPost, "/api/user/", http.StatusNotFound},
		{http.MethodPost, "/api/user/login/", http.StatusNotFound},
		{http.MethodGet, "/api/", http.StatusNotFound},
		{http.MethodPost, "/api", http.StatusNotFound},
		{http.MethodPost, "/api/user/balance/1", http.StatusNotFound},
		{http.MethodGet, "/api/user/withdrawals/", http.StatusNotFound},
		{http.MethodGet, "/ping/", http.StatusNotFound},

		{http.MethodGet, "/api/user/register", http.StatusMethodNotAllowed},
		{http.MethodGet, "/api/user/login", http.StatusMethodNotAllowed},
		{http.MethodPut, "/api/user/orders", http.StatusMethodNotAllowed},
		{http.MethodDelete, "/api/user/orders", http.StatusMethodNotAllowed},
		{http.MethodPost, "/api/user/balance", http.StatusMethodNotAllowed},
		{http.MethodGet, "/api/user/balance/withdraw", http.StatusMethodNotAllowed},
		{http.MethodPost, "/api/user/withdrawals", http.StatusMethodNotAllowed},
		{http.MethodPost, "/ping?x=true", http.StatusMethodNotAllowed},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, srv.URL+tt.path, http.NoBody)
			require.NoError(t, err)
			jwtCookie, err := auth.Authenticate("id", []byte(""))
			require.NoError(t, err)
			req.AddCookie(&jwtCookie)

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			err = resp.Body.Close()
			require.NoError(t, err)

			assert.Equal(t, tt.wantCode, resp.StatusCode)
		})
	}
}
