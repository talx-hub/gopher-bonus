package router

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/talx-hub/gopher-bonus/internal/config"
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

type Handler interface {
	Register(w http.ResponseWriter, r *http.Request)
	Login(w http.ResponseWriter, r *http.Request)
	GetOrders(w http.ResponseWriter, r *http.Request)
	PostOrders(w http.ResponseWriter, r *http.Request)
	GetBalance(w http.ResponseWriter, r *http.Request)
	Withdraw(w http.ResponseWriter, r *http.Request)
	GetInfo(w http.ResponseWriter, r *http.Request)
	Ping(w http.ResponseWriter, r *http.Request)
}

func (cr *CustomRouter) SetRouter(h Handler) {
	cr.router.Route("/api/user", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(middleware.AllowContentType("application/json"))
			r.Post("/register", h.Register)
			r.Post("/login", h.Login)
		})

		r.Route("/orders", func(r chi.Router) {
			r.With(middleware.AllowContentType("text/plain")).
				Post("/", h.PostOrders)
			r.Get("/", h.GetOrders)
		})

		r.Route("/balance", func(r chi.Router) {
			r.Get("/", h.GetBalance)
			r.Route("/withdraw", func(r chi.Router) {
				r.With(middleware.AllowContentType("application/json")).
					Post("/", h.Withdraw)
			})
		})
		r.Get("/withdrawals", h.GetInfo)
	})
	cr.router.Get("/ping", h.Ping)
}

func (cr *CustomRouter) GetRouter() *chi.Mux {
	return cr.router
}
