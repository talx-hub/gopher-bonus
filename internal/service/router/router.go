package router

import (
	"compress/gzip"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/talx-hub/gopher-bonus/internal/api/middlewares"
	"github.com/talx-hub/gopher-bonus/internal/service/config"
)

type CustomRouter struct {
	router *chi.Mux
	logger *slog.Logger
	cfg    *config.Config
}

func New(cfg *config.Config, log *slog.Logger) *CustomRouter {
	router := &CustomRouter{
		router: chi.NewRouter(),
		logger: log,
		cfg:    cfg,
	}

	return router
}

type AuthHandler interface {
	Register(w http.ResponseWriter, r *http.Request)
	Login(w http.ResponseWriter, r *http.Request)
}

type OrdersHandler interface {
	PostOrder(w http.ResponseWriter, r *http.Request)
	GetOrders(w http.ResponseWriter, r *http.Request)
	Withdraw(w http.ResponseWriter, r *http.Request)
	GetBalance(w http.ResponseWriter, r *http.Request)
	GetWithdrawals(w http.ResponseWriter, r *http.Request)
}

type HealthHandler interface {
	Ping(w http.ResponseWriter, r *http.Request)
}

type Handler interface {
	AuthHandler
	OrdersHandler
	HealthHandler
}

func (cr *CustomRouter) SetRouter(h Handler) {
	cr.router.Route("/api/user", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(middleware.Compress(gzip.DefaultCompression))

			r.Group(func(r chi.Router) {
				r.Use(middleware.AllowContentType("application/json"))
				r.Post("/register", h.Register)
				r.Post("/login", h.Login)
			})

			r.Group(func(r chi.Router) {
				r.Use(middlewares.Authentication([]byte(cr.cfg.SecretKey), cr.logger))

				r.Route("/orders", func(r chi.Router) {
					r.With(middleware.AllowContentType("text/plain")).
						Post("/", h.PostOrder)
					r.Get("/", h.GetOrders)
				})

				r.Route("/balance", func(r chi.Router) {
					r.Get("/", h.GetBalance)
					r.Route("/withdraw", func(r chi.Router) {
						r.With(middleware.AllowContentType("application/json")).
							Post("/", h.Withdraw)
					})
				})
				r.Get("/withdrawals", h.GetWithdrawals)
			})
		})
	})
	cr.router.Get("/ping", h.Ping)

	cr.router.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w,
			http.StatusText(http.StatusMethodNotAllowed),
			http.StatusMethodNotAllowed)
	})
}

func (cr *CustomRouter) GetRouter() *chi.Mux {
	return cr.router
}
