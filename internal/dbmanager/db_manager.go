package dbmanager

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/talx-hub/gopher-bonus/internal/model"
)

type DBManager struct {
	log         *slog.Logger
	Pool        *pgxpool.Pool
	dsn         string
	IsConnected bool
}

func New(dsn string, log *slog.Logger) *DBManager {
	return &DBManager{
		log:         log,
		Pool:        nil,
		IsConnected: false,
		dsn:         dsn,
	}
}

func (m *DBManager) Connect(ctx context.Context) *DBManager {
	cfg, err := pgxpool.ParseConfig(m.dsn)
	if err != nil {
		m.log.LogAttrs(ctx,
			slog.LevelError,
			"failed to parse DSN",
			slog.Any(model.KeyLoggerError, err),
		)

		return m
	}
	cfg.MinConns = 1
	cfg.MaxConns = 10
	cfg.ConnConfig.Tracer = &queryTracer{m.log}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		m.log.LogAttrs(ctx,
			slog.LevelError,
			"failed to init pgxpool",
			slog.Any(model.KeyLoggerError, err),
		)

		return m
	}
	if err = pool.Ping(ctx); err != nil {
		m.log.LogAttrs(ctx,
			slog.LevelError,
			"failed to ping the DB",
			slog.Any(model.KeyLoggerError, err),
		)

		m.Pool = pool
		return m
	}

	m.IsConnected = true
	m.Pool = pool
	return m
}

func (m *DBManager) ApplyMigrations() error {
	return nil
}

func (m *DBManager) Ping(ctx context.Context) *DBManager {
	if err := m.Pool.Ping(ctx); err != nil {
		m.log.LogAttrs(ctx,
			slog.LevelError,
			"failed to ping the DB",
			slog.Any(model.KeyLoggerError, err),
		)
		m.IsConnected = false
		return m
	}

	m.IsConnected = true
	return m
}

func (m *DBManager) Close() {
	if m.Pool == nil {
		return
	}

	m.Pool.Close()
	m.log.LogAttrs(context.TODO(),
		slog.LevelInfo,
		"connection to DB closed",
	)
}
