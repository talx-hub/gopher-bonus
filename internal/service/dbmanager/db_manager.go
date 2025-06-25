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

func New(dsn string, log *slog.Logger) *DBManager {
	return &DBManager{
		log:  log,
		pool: nil,
		dsn:  dsn,
	}
}

type pool interface {
	Ping(context.Context) error
	Close()
}

type DBManager struct {
	log  *slog.Logger
	pool pool
	err  error
	dsn  string
}

func (m *DBManager) Error() error {
	return m.err
}

func (m *DBManager) GetPool(ctx context.Context) (*pgxpool.Pool, error) {
	p, ok := m.pool.(*pgxpool.Pool)
	if !ok {
		const errStr = "failed to convert pool to *pgxpool.Pool"
		m.log.LogAttrs(ctx,
			slog.LevelError,
			errStr,
		)
		return nil, errors.New(errStr)
	}

	return p, nil
}

func (m *DBManager) Connect(ctx context.Context) *DBManager {
	if m.err != nil {
		return m
	}

	cfg, err := pgxpool.ParseConfig(m.dsn)
	if err != nil {
		m.log.LogAttrs(ctx,
			slog.LevelError,
			"failed to parse DSN",
			slog.Any(model.KeyLoggerError, err),
		)

		m.err = err
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

		m.err = err
		return m
	}

	m.pool = pool
	return m
}

//go:embed migrations/*.sql
var migrationsDir embed.FS

func (m *DBManager) ApplyMigrations(ctx context.Context) *DBManager {
	if m.err != nil {
		return m
	}

	d, err := iofs.New(migrationsDir, "migrations")
	if err != nil {
		m.log.LogAttrs(ctx,
			slog.LevelError,
			"failed to return an iofs driver",
			slog.Any(model.KeyLoggerError, err),
		)

		m.err = err
		return m
	}

	migrations, err := migrate.NewWithSourceInstance("iofs", d, m.dsn)
	if err != nil {
		m.log.LogAttrs(ctx,
			slog.LevelError,
			"failed to get a new migrate instance",
			slog.Any(model.KeyLoggerError, err),
		)

		m.err = err
		return m
	}

	err = migrations.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		m.log.LogAttrs(ctx,
			slog.LevelError,
			"failed to apply migrations to the DB",
			slog.Any(model.KeyLoggerError, err),
		)

		m.err = err
		return m
	}
	if errors.Is(err, migrate.ErrNoChange) {
		m.log.LogAttrs(ctx,
			slog.LevelInfo,
			"no migrations to apply",
		)
		return m
	}

	m.log.LogAttrs(ctx,
		slog.LevelInfo,
		"migrations applied successfully",
	)
	return m
}

func (m *DBManager) Ping(ctx context.Context) *DBManager {
	if m.err != nil {
		return m
	}

	if err := m.pool.Ping(ctx); err != nil {
		m.log.LogAttrs(ctx,
			slog.LevelError,
			"failed to ping the DB",
			slog.Any(model.KeyLoggerError, err),
		)
		m.err = err
	}

	m.log.LogAttrs(ctx,
		slog.LevelInfo,
		"successfully ping the DB",
	)
	return m
}

func (m *DBManager) Close() {
	if m.pool == nil {
		return
	}

	m.pool.Close()
	m.log.LogAttrs(context.TODO(),
		slog.LevelInfo,
		"connection to DB closed",
	)
}
