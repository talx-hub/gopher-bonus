package repo

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/talx-hub/gopher-bonus/internal/dbmanager"
	"github.com/talx-hub/gopher-bonus/internal/model/user"
)

var pool *pgxpool.Pool

func TestMain(m *testing.M) {
	code, err := runMain(m)
	if err != nil {
		slog.Error("main failed", "reason", err.Error())
	}
	os.Exit(code)
}

func runMain(m *testing.M) (int, error) {
	logger := slog.Default()

	dsn := "postgresql://gophermart:gophermart@localhost:5432/gophermart"
	mn := dbmanager.New(dsn, logger)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	mn.Connect(ctx).Ping(ctx)

	var err error
	pool, err = mn.GetPool(ctx)
	if err != nil {
		logger.Error("failed to get pgxpool", "reason", err.Error())
		return 1, err
	}

	exitCode := m.Run()
	return exitCode, nil
}

func TestUserRepository_Create(t *testing.T) {
	repo := NewUserRepository(pool, slog.Default())

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err := repo.Create(ctx, &user.User{
		LoginHash:    "user1hash",
		PasswordHash: "user1password-hash",
	})
	require.NoError(t, err)

	exists := repo.Exists(ctx, "user1hash")
	assert.True(t, exists)
}

func TestUserRepository_FindByLogin(t *testing.T) {
	repo := NewUserRepository(pool, slog.Default())

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	u, err := repo.FindByLogin(ctx, "user1hash")
	require.NoError(t, err)
	assert.Equal(t, "user1hash", u.LoginHash)
	assert.Equal(t, "user1password-hash", u.PasswordHash)
	assert.Equal(t, "1", u.ID)

	u2, err := repo.FindByLogin(ctx, "no-such-user")
	assert.Error(t, err)
	assert.Equal(t, "", u2.LoginHash)
	assert.Equal(t, "", u2.PasswordHash)
	assert.Equal(t, "", u2.ID)
}

func TestUserRepository_FindByID(t *testing.T) {
	repo := NewUserRepository(pool, slog.Default())

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	u, err := repo.FindByID(ctx, "1")
	require.NoError(t, err)
	assert.Equal(t, "user1hash", u.LoginHash)
	assert.Equal(t, "user1password-hash", u.PasswordHash)
	assert.Equal(t, "1", u.ID)

	u2, err := repo.FindByID(ctx, "100500")
	assert.Error(t, err)
	assert.Equal(t, "", u2.LoginHash)
	assert.Equal(t, "", u2.PasswordHash)
	assert.Equal(t, "", u2.ID)

	u3, err := repo.FindByID(ctx, "not-int")
	assert.Error(t, err)
	assert.Equal(t, "", u3.LoginHash)
	assert.Equal(t, "", u3.PasswordHash)
	assert.Equal(t, "", u3.ID)
}
