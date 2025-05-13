package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubHandler struct {
	name string
}

func (s stubHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
func (h) PostOrders(w http.ResponseWriter, r *http.Request) {
	stubHandler{name: "post_orders"}.ServeHTTP(w, r)
}
func (h) GetBalance(w http.ResponseWriter, r *http.Request) {
	stubHandler{name: "get_balance"}.ServeHTTP(w, r)
}
func (h) Withdraw(w http.ResponseWriter, r *http.Request) {
	stubHandler{name: "withdraw"}.ServeHTTP(w, r)
}
func (h) GetStatistics(w http.ResponseWriter, r *http.Request) {
	stubHandler{name: "get_statistics"}.ServeHTTP(w, r)
}
func (h) Ping(w http.ResponseWriter, r *http.Request) {
	stubHandler{name: "ping"}.ServeHTTP(w, r)
}

func TestCustomRouter_Route_happyTests(t *testing.T) {
	r := New(nil, nil)
	r.SetRouter(h{})
	srv := httptest.NewServer(r.GetRouter())
	defer srv.Close()

	tests := []struct {
		method   string
		path     string
		wantName string
		wantCode int
	}{
		{http.MethodPost, "/api/user/register", "register", http.StatusTeapot},
		{http.MethodPost, "/api/user/login", "login", http.StatusTeapot},
		{http.MethodGet, "/api/user/orders", "get_orders", http.StatusTeapot},
		{http.MethodPost, "/api/user/orders", "post_orders", http.StatusTeapot},
		{http.MethodGet, "/api/user/balance", "get_balance", http.StatusTeapot},
		{http.MethodPost, "/api/user/balance/withdraw", "withdraw", http.StatusTeapot},
		{http.MethodGet, "/api/user/withdrawals", "get_statistics", http.StatusTeapot},
		{http.MethodGet, "/ping", "ping", http.StatusTeapot},
	}

	for _, tt := range tests {
		req, err := http.NewRequest(tt.method, srv.URL+tt.path, http.NoBody)
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		err = resp.Body.Close()
		require.NoError(t, err)

		assert.Equal(t, tt.wantCode, resp.StatusCode)
		assert.Equal(t, tt.wantName, resp.Header.Get("X-Handler"))
	}
}

func TestCustomRouter_Route_wrong_routes(t *testing.T) {
	r := New(nil, nil)
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

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			err = resp.Body.Close()
			require.NoError(t, err)

			assert.Equal(t, tt.wantCode, resp.StatusCode)
		})
	}
}
