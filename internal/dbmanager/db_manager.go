package dbmanager

import (
	"context"
	"embed"
	"errors"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"

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

//go:embed migrations/*.sql
var migrationsDir embed.FS

func (m *DBManager) ApplyMigrations(ctx context.Context) *DBManager {
	d, err := iofs.New(migrationsDir, "migrations")
	if err != nil {
		m.log.LogAttrs(ctx,
			slog.LevelError,
			"failed to return an iofs driver",
			slog.Any(model.KeyLoggerError, err),
		)
		m.IsConnected = false
		return m
	}

	migrations, err := migrate.NewWithSourceInstance("iofs", d, m.dsn)
	if err != nil {
		m.log.LogAttrs(ctx,
			slog.LevelError,
			"failed to get a new migrate instance",
			slog.Any(model.KeyLoggerError, err),
		)
		m.IsConnected = false
		return m
	}
	if err := migrations.Up(); err != nil {
		if !errors.Is(err, migrate.ErrNoChange) {
			m.log.LogAttrs(ctx,
				slog.LevelError,
				"failed to apply migrations to the DB",
				slog.Any(model.KeyLoggerError, err),
			)
			m.IsConnected = false
			return m
		}
	}

	m.log.LogAttrs(ctx,
		slog.LevelInfo,
		"migrations applied successfully",
	)
	return m
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
