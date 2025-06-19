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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/talx-hub/gopher-bonus/internal/dbmanager"
	"github.com/talx-hub/gopher-bonus/internal/model"
	"github.com/talx-hub/gopher-bonus/internal/model/order"
	"github.com/talx-hub/gopher-bonus/internal/model/user"
	"github.com/talx-hub/gopher-bonus/internal/utils/pgcontainer"
)

const testDefaultTimeout = 5 * time.Second

var (
	getDSN       func() string
	getDBManager func() *dbmanager.DBManager
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

func TestUserRepository_Create(t *testing.T) {
	db := getDBManager()
	pool, err := db.GetPool(context.TODO())
	require.NoError(t, err)
	repo := NewUserRepository(pool, slog.Default())

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err = repo.Create(ctx, &user.User{
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
	db := getDBManager()
	pool, err := db.GetPool(context.TODO())
	require.NoError(t, err)
	repo := NewUserRepository(pool, slog.Default())

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	u, err := repo.FindByLogin(ctx, "user1hash")
	require.NoError(t, err)
	assert.Equal(t, "user1hash", u.LoginHash)
	assert.Equal(t, "user1password-hash", u.PasswordHash)
	assert.NotEmpty(t, u.ID)

	u2, err := repo.FindByLogin(ctx, "no-such-user")
	assert.Error(t, err)
	assert.Equal(t, "", u2.LoginHash)
	assert.Equal(t, "", u2.PasswordHash)
	assert.Equal(t, "", u2.ID)
}

func TestUserRepository_FindByID(t *testing.T) {
	db := getDBManager()
	pool, err := db.GetPool(context.TODO())
	require.NoError(t, err)
	repo := NewUserRepository(pool, slog.Default())
	err = loadFixtureFile(pool, "./fixtures/user_find_by_id.sql")
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	u, err := repo.FindByID(ctx, "1")
	require.NoError(t, err)
	assert.Equal(t, "user1hash", u.LoginHash)
	assert.Equal(t, "user1password-hash", u.PasswordHash)
	assert.NotEmpty(t, u.ID)

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
	db := getDBManager()
	pool, err := db.GetPool(context.TODO())
	require.NoError(t, err)
	repo := NewOrderRepository(pool, slog.Default())

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err = repo.CreateOrder(ctx, &order.Order{
		CreatedAt: time.Now(),
		Type:      order.TypeAccrual,
		Status:    order.StatusNew,
		ID:        "1",
		UserID:    "1",
	})
	require.NoError(t, err)

	err = repo.CreateOrder(ctx, &order.Order{
		CreatedAt: time.Now(),
		Type:      order.TypeAccrual,
		Status:    order.StatusNew,
		ID:        "2",
		UserID:    "1",
	})
	require.NoError(t, err)
	err = repo.CreateOrder(ctx, &order.Order{
		CreatedAt: time.Now(),
		Type:      order.TypeAccrual,
		Status:    order.StatusNew,
		ID:        "3",
		UserID:    "1",
	})
	require.NoError(t, err)
	err = repo.CreateOrder(ctx, &order.Order{
		CreatedAt: time.Now(),
		Type:      order.TypeAccrual,
		Status:    order.StatusNew,
		ID:        "4",
		UserID:    "1",
	})
	require.NoError(t, err)
	err = repo.CreateOrder(ctx, &order.Order{
		CreatedAt: time.Now(),
		Type:      order.TypeAccrual,
		Status:    order.StatusNew,
		ID:        "5",
		UserID:    "1",
	})
	require.NoError(t, err)
	err = repo.CreateOrder(ctx, &order.Order{
		CreatedAt: time.Now(),
		Type:      order.TypeAccrual,
		Status:    order.StatusNew,
		ID:        "6",
		UserID:    "2",
	})
	require.NoError(t, err)
}

func TestOrderRepository_FindUserIDByAccrualID(t *testing.T) {
	db := getDBManager()
	pool, err := db.GetPool(context.TODO())
	require.NoError(t, err)
	repo := NewOrderRepository(pool, slog.Default())

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	userID, err := repo.FindUserIDByAccrualID(ctx, "1")
	require.NoError(t, err)

	assert.Equal(t, "1", userID)

	userID, err = repo.FindUserIDByAccrualID(ctx, "7")
	require.Error(t, err)
	assert.Equal(t, "", userID)
}

func TestOrderRepository_ListAccruals(t *testing.T) {
	db := getDBManager()
	pool, err := db.GetPool(context.TODO())
	require.NoError(t, err)
	repo := NewOrderRepository(pool, slog.Default())

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	orders, err := repo.ListOrdersByUser(ctx, "1", order.TypeAccrual)
	require.NoError(t, err)
	assert.Equal(t, 5, len(orders))
	for _, o := range orders {
		assert.Equal(t, "1", o.UserID)
		assert.Equal(t, order.StatusNew, o.Status)
	}

	orders, err = repo.ListOrdersByUser(ctx, "2", order.TypeAccrual)
	require.NoError(t, err)
	assert.Equal(t, 1, len(orders))
	assert.Equal(t, "2", orders[0].UserID)
	assert.Equal(t, order.StatusNew, orders[0].Status)

	orders, err = repo.ListOrdersByUser(ctx, "3", order.TypeAccrual)
	require.NoError(t, err)
	assert.Equal(t, 0, len(orders))
}

func TestOrderRepository_UpdateOrder(t *testing.T) {
	db := getDBManager()
	pool, err := db.GetPool(context.TODO())
	require.NoError(t, err)
	repo := NewOrderRepository(pool, slog.Default())

	ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Second)
	defer cancel()
	err = repo.UpdateAccrualStatus(ctx, &order.Order{
		Status: order.StatusProcessed,
		ID:     "1",
		Amount: model.NewAmount(100, 5),
	})
	require.NoError(t, err)

	err = repo.UpdateAccrualStatus(ctx, &order.Order{
		Status: order.StatusProcessed,
		ID:     "2",
		Amount: model.NewAmount(0, 0),
	})
	require.NoError(t, err)

	err = repo.UpdateAccrualStatus(ctx, &order.Order{
		Status: order.StatusProcessed,
		ID:     "3",
		Amount: model.NewAmount(0, 51),
	})
	require.NoError(t, err)

	err = repo.UpdateAccrualStatus(ctx, &order.Order{
		Status: order.StatusProcessed,
		ID:     "4",
		Amount: model.NewAmount(100, 500),
	})
	require.NoError(t, err)

	err = repo.UpdateAccrualStatus(ctx, &order.Order{
		Status: order.StatusProcessing,
		ID:     "5",
	})
	require.NoError(t, err)

	err = repo.UpdateAccrualStatus(ctx, &order.Order{
		Status: order.StatusProcessing,
		ID:     "6",
	})
	require.NoError(t, err)

	err = repo.UpdateAccrualStatus(ctx, &order.Order{
		Status: order.StatusProcessed,
		ID:     "6",
		Amount: model.NewAmount(0, 500),
	})
	require.NoError(t, err)

	err = repo.UpdateAccrualStatus(ctx, &order.Order{
		Status: order.StatusInvalid,
		ID:     "7",
	})
	require.Error(t, err)

	err = repo.UpdateAccrualStatus(ctx, &order.Order{
		Status: order.StatusInvalid,
		ID:     "5",
	})
	require.NoError(t, err)
}
