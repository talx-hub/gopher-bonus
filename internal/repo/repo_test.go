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
	"github.com/talx-hub/gopher-bonus/internal/model"
	"github.com/talx-hub/gopher-bonus/internal/model/order"
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

	err = repo.Create(ctx, &user.User{
		LoginHash:    "user2hash",
		PasswordHash: "user2password-hash",
	})
	require.NoError(t, err)

	exists = repo.Exists(ctx, "user2hash")
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

func TestOrderRepository_Create(t *testing.T) {
	repo := NewOrderRepository(pool, slog.Default())

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err := repo.Create(ctx, &order.Order{
		UploadedAt: time.Now(),
		Status:     order.StatusNew,
		ID:         "1",
		UserID:     "1",
	})
	require.NoError(t, err)

	err = repo.Create(ctx, &order.Order{
		UploadedAt: time.Now(),
		Status:     order.StatusNew,
		ID:         "2",
		UserID:     "1",
	})
	require.NoError(t, err)
	err = repo.Create(ctx, &order.Order{
		UploadedAt: time.Now(),
		Status:     order.StatusNew,
		ID:         "3",
		UserID:     "1",
	})
	require.NoError(t, err)
	err = repo.Create(ctx, &order.Order{
		UploadedAt: time.Now(),
		Status:     order.StatusNew,
		ID:         "4",
		UserID:     "1",
	})
	require.NoError(t, err)
	err = repo.Create(ctx, &order.Order{
		UploadedAt: time.Now(),
		Status:     order.StatusNew,
		ID:         "5",
		UserID:     "1",
	})
	require.NoError(t, err)
	err = repo.Create(ctx, &order.Order{
		UploadedAt: time.Now(),
		Status:     order.StatusNew,
		ID:         "6",
		UserID:     "2",
	})
	require.NoError(t, err)
}

func TestOrderRepository_FindByID(t *testing.T) {
	repo := NewOrderRepository(pool, slog.Default())

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	o, err := repo.FindByID(ctx, "1")
	require.NoError(t, err)

	assert.Equal(t, "1", o.UserID)

	o, err = repo.FindByID(ctx, "7")
	require.Error(t, err)
	assert.Equal(t, order.Order{}, o)
}

func TestOrderRepository_ListByUserID(t *testing.T) {
	repo := NewOrderRepository(pool, slog.Default())

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	orders, err := repo.ListByUserID(ctx, "1")
	require.NoError(t, err)
	assert.Equal(t, 5, len(orders))
	for _, o := range orders {
		assert.Equal(t, "1", o.UserID)
		assert.Equal(t, order.StatusNew, o.Status)
	}

	orders, err = repo.ListByUserID(ctx, "2")
	require.NoError(t, err)
	assert.Equal(t, 1, len(orders))
	assert.Equal(t, "2", orders[0].UserID)
	assert.Equal(t, order.StatusNew, orders[0].Status)

	orders, err = repo.ListByUserID(ctx, "3")
	require.NoError(t, err)
	assert.Equal(t, 0, len(orders))
}

func TestOrderRepository_UpdateOrder(t *testing.T) {
	repo := NewOrderRepository(pool, slog.Default())

	ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Second)
	defer cancel()
	err := repo.UpdateOrder(ctx, &order.Order{
		Status:  order.StatusProcessed,
		ID:      "1",
		Accrual: model.NewAmount(100, 5),
	})
	require.NoError(t, err)

	err = repo.UpdateOrder(ctx, &order.Order{
		Status:  order.StatusProcessed,
		ID:      "2",
		Accrual: model.NewAmount(0, 0),
	})
	require.NoError(t, err)

	err = repo.UpdateOrder(ctx, &order.Order{
		Status:  order.StatusProcessed,
		ID:      "3",
		Accrual: model.NewAmount(0, 51),
	})
	require.NoError(t, err)

	err = repo.UpdateOrder(ctx, &order.Order{
		Status:  order.StatusProcessed,
		ID:      "4",
		Accrual: model.NewAmount(100, 500),
	})
	require.NoError(t, err)

	err = repo.UpdateOrder(ctx, &order.Order{
		Status: order.StatusProcessing,
		ID:     "5",
	})
	require.NoError(t, err)

	err = repo.UpdateOrder(ctx, &order.Order{
		Status: order.StatusProcessing,
		ID:     "6",
	})
	require.NoError(t, err)

	err = repo.UpdateOrder(ctx, &order.Order{
		Status:  order.StatusProcessed,
		ID:      "6",
		Accrual: model.NewAmount(0, 500),
	})
	require.NoError(t, err)

	err = repo.UpdateOrder(ctx, &order.Order{
		Status: order.StatusInvalid,
		ID:     "7",
	})
	require.Error(t, err)

	err = repo.UpdateOrder(ctx, &order.Order{
		Status: order.StatusInvalid,
		ID:     "5",
	})
	require.NoError(t, err)
}
