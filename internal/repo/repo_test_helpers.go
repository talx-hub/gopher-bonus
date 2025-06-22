package repo

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/talx-hub/gopher-bonus/internal/service/dbmanager"
	"github.com/talx-hub/gopher-bonus/internal/utils/pgcontainer"
)

const testDefaultTimeout = 5 * time.Second

var (
	getDSN       func() string
	getDBManager func() *dbmanager.DBManager
)

func runMain(m *testing.M, log *slog.Logger) (int, error) {
	pg := pgcontainer.New(log)
	getDSN = func() string {
		return pg.GetDSN()
	}
	err := pg.RunContainer()
	defer pg.Close()
	if err != nil {
		return 1, fmt.Errorf("failed to run docker container: %w", err)
	}

	if err = initGetDBManager(log); err != nil {
		return 1, fmt.Errorf("failed to init test DB: %w", err)
	}

	db := getDBManager()
	defer db.Close()

	exitCode := m.Run()
	return exitCode, nil
}

func initGetDBManager(log *slog.Logger) error {
	dsn := getDSN()
	db := dbmanager.New(dsn, log)

	ctx, cancel := context.WithTimeout(context.Background(), testDefaultTimeout)
	defer cancel()

	db.Connect(ctx).Ping(ctx).ApplyMigrations(ctx)
	if err := db.Error(); err != nil {
		return fmt.Errorf("failed to prepare test DB using dsn %s: %w", dsn, err)
	}

	getDBManager = func() *dbmanager.DBManager {
		return db
	}
	return nil
}

func loadFixtureFile(conn *pgxpool.Pool, filepath string) error {
	content, err := os.ReadFile(filepath)
	if err != nil {
		return fmt.Errorf("failed to read fixture file: %w", err)
	}

	queries := strings.Split(string(content), ";")

	for _, rawQuery := range queries {
		query := strings.TrimSpace(rawQuery)
		if query == "" {
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), testDefaultTimeout)
		_, err := conn.Exec(ctx, query)
		cancel()
		if err != nil {
			return fmt.Errorf("failed to execute query [%s]: %w", query, err)
		}
	}

	return nil
}

func setupRepo[T any](t *testing.T,
	repoConstructor func(pool connectionPool, log *slog.Logger) T,
) (T, context.Context, context.CancelFunc, *pgxpool.Pool) {
	t.Helper()

	db := getDBManager()
	pool, err := db.GetPool(context.Background())
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), testDefaultTimeout)
	return repoConstructor(pool, slog.Default()), ctx, cancel, pool
}
