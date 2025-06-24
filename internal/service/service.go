package service

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/talx-hub/gopher-bonus/internal/api/handlers"
	"github.com/talx-hub/gopher-bonus/internal/model"
	"github.com/talx-hub/gopher-bonus/internal/repo"
	"github.com/talx-hub/gopher-bonus/internal/service/agent"
	"github.com/talx-hub/gopher-bonus/internal/service/config"
	"github.com/talx-hub/gopher-bonus/internal/service/dbmanager"
	"github.com/talx-hub/gopher-bonus/internal/service/dto"
	"github.com/talx-hub/gopher-bonus/internal/service/router"
	"github.com/talx-hub/gopher-bonus/internal/service/watcher"
)

func initService(log *slog.Logger) (*chi.Mux, string) {
	inputCh := make(chan string)
	outputCh := make(chan dto.AccrualInfo)

	cfg := config.NewBuilder(log).
		FromEnv().
		FromFlags().
		GetConfig()

	const connectTO = 2 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), connectTO)
	defer cancel()
	dbManager := dbmanager.New(cfg.DatabaseURI, log).
		Connect(ctx).
		ApplyMigrations(ctx).
		Ping(ctx)
	if err := dbManager.Error(); err != nil {
		log.LogAttrs(context.Background(),
			slog.LevelError,
			"failed to start service: db connection error",
			slog.Any(model.KeyLoggerError, err),
		)
		return nil, ""
	}

	db, err := dbManager.GetPool(ctx)
	if err != nil {
		log.LogAttrs(context.Background(),
			slog.LevelError,
			"failed to start service: failed to get DB pool",
			slog.Any(model.KeyLoggerError, err),
		)
		return nil, ""
	}

	usersRepo := repo.NewUserRepository(db, log)
	orderRepo := repo.NewOrderRepository(db, log)

	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()
	w := watcher.New(orderRepo, inputCh, outputCh)
	w.Run(ctx)
	a := agent.New(inputCh, outputCh, cfg.AccrualAddr)
	a.Run(ctx, model.DefaultRequestCount)

	rr := router.New(cfg, log)
	rr.SetRouter(&struct {
		*handlers.AuthHandler
		*handlers.OrderHandler
		*handlers.HealthHandler
	}{
		AuthHandler:   handlers.NewAuthHandler(usersRepo, log, cfg.SecretKey),
		OrderHandler:  handlers.NewOrderHandler(usersRepo, orderRepo, log),
		HealthHandler: handlers.NewHealthHandler(dbManager),
	})

	return rr.GetRouter(), cfg.RunAddr
}

func RunServer() {
	log := slog.Default()
	mux, addr := initService(log)
	if mux == nil {
		log.LogAttrs(context.TODO(),
			slog.LevelError,
			"failed to init service",
		)
		return
	}

	err := http.ListenAndServe(addr, mux)
	if err != nil {
		log.LogAttrs(context.TODO(),
			slog.LevelError,
			"listen and serve error",
			slog.Any(model.KeyLoggerError, err),
		)
	}
}
