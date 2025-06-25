package dbmanager_test

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/talx-hub/gopher-bonus/internal/model"
	"github.com/talx-hub/gopher-bonus/internal/service/dbmanager"
	"github.com/talx-hub/gopher-bonus/internal/utils/pgcontainer"
)

const testDefaultTimeout = 5 * time.Second

var (
	getDSN func() string
)

func TestMain(m *testing.M) {
	log := slog.Default()
	code, err := runMain(m, log)
	if err != nil {
		log.ErrorContext(context.TODO(),
			"unexpected test failure",
			slog.Any(model.KeyLoggerError, err),
		)
	}
	os.Exit(code)
}

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

	exitCode := m.Run()
	return exitCode, nil
}

func TestDBManager_Connect(t *testing.T) {
	dsn := getDSN()
	db := dbmanager.New(dsn, slog.Default())
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), testDefaultTimeout)
	defer cancel()

	db.Connect(ctx)
	if err := db.Error(); err != nil {
		t.Errorf("failed to connect to test DB using dsn %s: %v", dsn, err)
	}
}

func TestDBManager_Ping(t *testing.T) {
	dsn := getDSN()
	db := dbmanager.New(dsn, slog.Default())
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), testDefaultTimeout)
	defer cancel()

	db.Connect(ctx).Ping(ctx)
	if err := db.Error(); err != nil {
		t.Errorf("failed to ping test DB using dsn %s: %v", dsn, err)
	}
}

func TestDBManager_ApplyMigrations(t *testing.T) {
	dsn := getDSN()
	db := dbmanager.New(dsn, slog.Default())
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), testDefaultTimeout)
	defer cancel()

	db.Connect(ctx).Ping(ctx).ApplyMigrations(ctx).ApplyMigrations(ctx)
	if err := db.Error(); err != nil {
		t.Errorf("failed to apply mirgrations to test db using dsn %s: %v", dsn, err)
	}
}

func TestDBManager_GetPool_from_nil(t *testing.T) {
	dsn := getDSN()
	db := dbmanager.New(dsn, slog.Default())
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), testDefaultTimeout)
	defer cancel()
	p, err := db.GetPool(ctx)
	assert.Nil(t, p)
	assert.Error(t, err)
}

func TestDBManager_GetPool(t *testing.T) {
	dsn := getDSN()
	db := dbmanager.New(dsn, slog.Default())
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), testDefaultTimeout)
	defer cancel()

	db.Connect(ctx)
	if err := db.Error(); err != nil {
		t.Errorf("failed to connect to test DB using dsn %s: %v", dsn, err)
	}
	p, err := db.GetPool(ctx)
	require.NoError(t, err)
	assert.NotNil(t, p)
}
