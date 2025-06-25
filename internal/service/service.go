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
	"github.com/talx-hub/gopher-bonus/internal/utils/logger"
)

func initService(log *slog.Logger) (*chi.Mux, context.CancelFunc, string) {
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
		return nil, nil, ""
	}

	db, err := dbManager.GetPool(ctx)
	if err != nil {
		log.LogAttrs(context.Background(),
			slog.LevelError,
			"failed to start service: failed to get DB pool",
			slog.Any(model.KeyLoggerError, err),
		)
		return nil, nil, ""
	}

	usersRepo := repo.NewUserRepository(db, log)
	orderRepo := repo.NewOrderRepository(db, log)

	ctx, cancel = context.WithCancel(context.Background())
	loggerCtx := logger.WithContext(ctx, log)

	inputCh := make(chan string)
	outputCh := make(chan dto.AccrualInfo)
	w := watcher.New(orderRepo, inputCh, outputCh)
	go w.Run(loggerCtx)
	a := agent.New(inputCh, outputCh, cfg.AccrualAddr)
	go a.Run(loggerCtx, model.DefaultRequestCount)

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

	return rr.GetRouter(), cancel, cfg.RunAddr
}

func RunServer() {
	log := slog.Default()
	mux, cancel, addr := initService(log)
	defer cancel()
	if mux == nil {
		log.LogAttrs(context.TODO(),
			slog.LevelError,
			"failed to init service",
		)
		return
	}

	log.LogAttrs(context.Background(),
		slog.LevelInfo,
		"starting server.....",
		slog.String("addr", addr),
	)
	err := http.ListenAndServe(addr, mux)
	if err != nil {
		log.LogAttrs(context.TODO(),
			slog.LevelError,
			"listen and serve error",
			slog.Any(model.KeyLoggerError, err),
		)
	}
}
