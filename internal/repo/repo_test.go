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
	"github.com/talx-hub/gopher-bonus/internal/serviceerrs"
	"github.com/talx-hub/gopher-bonus/internal/utils/pgcontainer"
)

const testDefaultTimeout = 50000 * time.Second

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

func TestUserRepository_Create(t *testing.T) {
	repo, ctx, cancel, _ := setupRepo(t, NewUserRepository)
	defer cancel()

	tests := []struct {
		name       string
		loginHash  string
		password   string
		wantExists bool
		wantErr    bool
	}{
		{"create user1", "user1hash", "user1password-hash", true, false},
		{"create user2", "user2hash", "user2password-hash", true, false},
		{"duplicate login", "user1hash", "another-password", true, true},
		{"empty login", "", "some-password", false, true},
		{"empty password", "some-new-user", "", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.Create(ctx, &user.User{
				LoginHash:    tt.loginHash,
				PasswordHash: tt.password,
			})

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			exists := repo.Exists(ctx, tt.loginHash)
			assert.Equal(t, tt.wantExists, exists)
		})
	}
}

func TestUserRepository_FindByLogin(t *testing.T) {
	repo, ctx, cancel, _ := setupRepo(t, NewUserRepository)
	defer cancel()

	tests := []struct {
		name      string
		loginHash string
		wantUser  user.User
		wantErr   bool
	}{
		{
			name:      "existing user",
			loginHash: "user1hash",
			wantUser: user.User{
				LoginHash:    "user1hash",
				PasswordHash: "user1password-hash",
			},
			wantErr: false,
		},
		{
			name:      "non-existing user",
			loginHash: "no-such-user",
			wantUser:  user.User{},
			wantErr:   true,
		},
		{
			name:      "empty login",
			loginHash: "",
			wantUser:  user.User{},
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := repo.FindByLogin(ctx, tt.loginHash)
			if tt.wantErr {
				require.Error(t, err)
				assert.Equal(t, user.User{}, u)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantUser.LoginHash, u.LoginHash)
				assert.Equal(t, tt.wantUser.PasswordHash, u.PasswordHash)
				assert.NotEmpty(t, u.ID)
			}
		})
	}
}

func TestUserRepository_FindByID(t *testing.T) {
	repo, ctx, cancel, pool := setupRepo(t, NewUserRepository)
	defer cancel()
	err := loadFixtureFile(pool, "./fixtures/user_find_by_id.sql")
	require.NoError(t, err)

	tests := []struct {
		name    string
		id      string
		want    user.User
		wantErr bool
	}{
		{"existing user", "1", user.User{
			ID: "1", LoginHash: "user1hash", PasswordHash: "user1password-hash"}, false},
		{"not found", "100500", user.User{}, true},
		{"bad ID", "not-int", user.User{}, true},
		{"empty ID", "", user.User{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.FindByID(ctx, tt.id)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Equal(t, user.User{}, got)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestOrderRepository_CreateAccrual(t *testing.T) {
	repo, ctx, cancel, pool := setupRepo(t, NewOrderRepository)
	defer cancel()
	require.NoError(t, loadFixtureFile(pool, "./fixtures/order_create_accrual.sql"))

	tests := []struct {
		name    string
		order   order.Order
		wantErr bool
	}{
		{
			name: "valid accrual for user 1 - order 1",
			order: order.Order{
				CreatedAt: time.Now(),
				Type:      order.TypeAccrual,
				Status:    order.StatusNew,
				ID:        "1",
				UserID:    "1",
			},
			wantErr: false,
		},
		{
			name: "valid accrual for user 1 - order 2",
			order: order.Order{
				CreatedAt: time.Now(),
				Type:      order.TypeAccrual,
				Status:    order.StatusNew,
				ID:        "2",
				UserID:    "1",
			},
			wantErr: false,
		},
		{
			name: "valid accrual for user 1 - order 3",
			order: order.Order{
				CreatedAt: time.Now(),
				Type:      order.TypeAccrual,
				Status:    order.StatusNew,
				ID:        "3",
				UserID:    "1",
			},
			wantErr: false,
		},
		{
			name: "valid accrual for user 1 - order 4",
			order: order.Order{
				CreatedAt: time.Now(),
				Type:      order.TypeAccrual,
				Status:    order.StatusNew,
				ID:        "4",
				UserID:    "1",
			},
			wantErr: false,
		},
		{
			name: "valid accrual for user 1 - order 5",
			order: order.Order{
				CreatedAt: time.Now(),
				Type:      order.TypeAccrual,
				Status:    order.StatusNew,
				ID:        "5",
				UserID:    "1",
			},
			wantErr: false,
		},
		{
			name: "valid accrual for user 2 - order 6",
			order: order.Order{
				CreatedAt: time.Now(),
				Type:      order.TypeAccrual,
				Status:    order.StatusNew,
				ID:        "6",
				UserID:    "2",
			},
			wantErr: false,
		},
		{
			name: "missing order ID",
			order: order.Order{
				CreatedAt: time.Now(),
				Type:      order.TypeAccrual,
				Status:    order.StatusNew,
				ID:        "",
				UserID:    "1",
			},
			wantErr: true,
		},
		{
			name: "missing user ID",
			order: order.Order{
				CreatedAt: time.Now(),
				Type:      order.TypeAccrual,
				Status:    order.StatusNew,
				ID:        "7",
				UserID:    "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.CreateOrder(ctx, &tt.order)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestOrderRepository_FindUserIDByAccrualID(t *testing.T) {
	repo, ctx, cancel, _ := setupRepo(t, NewOrderRepository)
	defer cancel()

	tests := []struct {
		name      string
		accrualID string
		wantUser  string
		wantErr   bool
	}{
		{"existing accrual ID", "1", "1", false},
		{"non-existing accrual ID", "7", "", true},
		{"empty accrual ID", "", "", true},
		{"invalid accrual ID", "invalid!", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userID, err := repo.FindUserIDByAccrualID(ctx, tt.accrualID)
			if tt.wantErr {
				require.Error(t, err)
				assert.Equal(t, "", userID)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantUser, userID)
			}
		})
	}
}

func TestOrderRepository_ListAccruals(t *testing.T) {
	repo, ctx, cancel, _ := setupRepo(t, NewOrderRepository)
	defer cancel()

	tests := []struct {
		name    string
		userID  string
		wantLen int
		wantErr bool
	}{
		{"user with 5 accruals", "1", 5, false},
		{"user with 1 accrual", "2", 1, false},
		{"user with no accruals", "3", 0, false},
		{"non-existing user", "nonexistent", 0, false},
		{"empty userID", "", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orders, err := repo.ListOrdersByUser(ctx, tt.userID, order.TypeAccrual)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Len(t, orders, tt.wantLen)
				for _, o := range orders {
					assert.Equal(t, tt.userID, o.UserID)
					assert.Equal(t, order.StatusNew, o.Status)
				}
			}
		})
	}
}

func TestOrderRepository_UpdateOrder(t *testing.T) {
	repo, ctx, cancel, _ := setupRepo(t, NewOrderRepository)
	defer cancel()

	tests := []struct {
		name    string
		order   order.Order
		wantErr bool
	}{
		{
			name: "update order 1 to processed",
			order: order.Order{
				Status: order.StatusProcessed,
				ID:     "1",
				Amount: model.NewAmount(100, 5),
			},
			wantErr: false,
		},
		{
			name: "update order 2 to processed with zero amount",
			order: order.Order{
				Status: order.StatusProcessed,
				ID:     "2",
				Amount: model.NewAmount(0, 0),
			},
			wantErr: false,
		},
		{
			name: "update order 3 to processed with small amount",
			order: order.Order{
				Status: order.StatusProcessed,
				ID:     "3",
				Amount: model.NewAmount(0, 51),
			},
			wantErr: false,
		},
		{
			name: "update order 4 to processed with large amount",
			order: order.Order{
				Status: order.StatusProcessed,
				ID:     "4",
				Amount: model.NewAmount(100, 500),
			},
			wantErr: false,
		},
		{
			name: "update order 5 to processing",
			order: order.Order{
				Status: order.StatusProcessing,
				ID:     "5",
			},
			wantErr: false,
		},
		{
			name: "update order 6 to processing",
			order: order.Order{
				Status: order.StatusProcessing,
				ID:     "6",
			},
			wantErr: false,
		},
		{
			name: "update order 6 to processed with amount",
			order: order.Order{
				Status: order.StatusProcessed,
				ID:     "6",
				Amount: model.NewAmount(0, 500),
			},
			wantErr: false,
		},
		{
			name: "update order 7 to invalid status (should error)",
			order: order.Order{
				Status: order.StatusInvalid,
				ID:     "7",
			},
			wantErr: true,
		},
		{
			name: "update order 5 to invalid status",
			order: order.Order{
				Status: order.StatusInvalid,
				ID:     "5",
			},
			wantErr: false,
		},
		{
			name: "empty order ID",
			order: order.Order{
				Status: order.StatusProcessed,
				ID:     "",
			},
			wantErr: true,
		},
		{
			name: "invalid order ID",
			order: order.Order{
				Status: order.StatusProcessed,
				ID:     "invalid!",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.UpdateAccrualStatus(ctx, &tt.order)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestOrderRepository_CreateWithdrawal(t *testing.T) {
	repo, ctx, cancel, pool := setupRepo(t, NewOrderRepository)
	defer cancel()

	require.NoError(t, loadFixtureFile(pool, "./fixtures/order_create_withdrawal.sql"))

	tests := []struct {
		name      string
		order     order.Order
		wantErr   bool
		wantError error
	}{
		{
			name: "user with enough balance and no withdrawals",
			order: order.Order{
				Type:   order.TypeWithdrawal,
				ID:     "w-user1",
				UserID: "user1",
				Amount: model.NewAmount(100, 0),
			},
			wantErr: false,
		},
		{
			name: "user with zero balance and no withdrawals",
			order: order.Order{
				Type:   order.TypeWithdrawal,
				ID:     "w-user2",
				UserID: "user2",
				Amount: model.NewAmount(1, 0),
			},
			wantErr:   true,
			wantError: serviceerrs.ErrInsufficientFunds,
		},
		{
			name: "user with balance equal to total withdrawals",
			order: order.Order{
				Type:   order.TypeWithdrawal,
				ID:     "w-user3",
				UserID: "user3",
				Amount: model.NewAmount(1, 0),
			},
			wantErr:   true,
			wantError: serviceerrs.ErrInsufficientFunds,
		},
		{
			name: "user with more withdrawals than accruals (weird case)",
			order: order.Order{
				Type:   order.TypeWithdrawal,
				ID:     "w-user4",
				UserID: "user4",
				Amount: model.NewAmount(1, 0),
			},
			wantErr:   true,
			wantError: serviceerrs.ErrInsufficientFunds,
		},
		{
			name: "user with enough accruals and some withdrawals",
			order: order.Order{
				Type:   order.TypeWithdrawal,
				ID:     "w-user5",
				UserID: "user5",
				Amount: model.NewAmount(100, 0),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.CreateOrder(ctx, &tt.order)
			if tt.wantErr {
				require.Error(t, err)
				if tt.wantError != nil {
					assert.ErrorIs(t, err, tt.wantError)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestOrderRepository_ListWithdrawals(t *testing.T) {
	repo, ctx, cancel, pool := setupRepo(t, NewOrderRepository)
	defer cancel()

	err := loadFixtureFile(pool, "./fixtures/order_list_withdrawals.sql")
	require.NoError(t, err)

	tests := []struct {
		name      string
		userID    string
		wantCount int
		wantErr   bool
		wantEmpty bool
	}{
		{"user1 with 3 withdrawals", "1", 3, false, false},
		{"user2 with 1 withdrawal", "2", 1, false, false},
		{"user3 with no withdrawals", "3", 0, false, true},
		{"empty userID", "", 0, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orders, err := repo.ListOrdersByUser(ctx, tt.userID, order.TypeWithdrawal)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, orders)
			} else {
				require.NoError(t, err)
				assert.Len(t, orders, tt.wantCount)
				if tt.wantEmpty {
					assert.Empty(t, orders)
				}
			}
		})
	}
}

func TestOrderRepository_GetBalance(t *testing.T) {
	repo, ctx, cancel, pool := setupRepo(t, NewOrderRepository)
	defer cancel()
	err := loadFixtureFile(pool, "./fixtures/order_get_balance.sql")
	require.NoError(t, err)

	tests := []struct {
		name         string
		userID       string
		wantAccrued  model.Amount
		wantWithdraw model.Amount
		wantErr      bool
	}{
		{
			name:         "no accruals and no withdrawals",
			userID:       "2",
			wantAccrued:  model.NewAmount(0, 0),
			wantWithdraw: model.NewAmount(0, 0),
			wantErr:      false,
		},
		{
			name:         "accruals > 0, no withdrawals",
			userID:       "1",
			wantAccrued:  model.NewAmount(150, 50),
			wantWithdraw: model.NewAmount(0, 0),
			wantErr:      false,
		},
		{
			name:         "equal accruals and withdrawals",
			userID:       "3",
			wantAccrued:  model.NewAmount(100, 0),
			wantWithdraw: model.NewAmount(100, 0),
			wantErr:      false,
		},
		{
			name:         "more withdrawn than accrued",
			userID:       "4",
			wantAccrued:  model.NewAmount(50, 0),
			wantWithdraw: model.NewAmount(70, 0),
			wantErr:      false,
		},
		{
			name:         "has both accruals and withdrawals",
			userID:       "5",
			wantAccrued:  model.NewAmount(300, 0),
			wantWithdraw: model.NewAmount(100, 50),
			wantErr:      false,
		},
		{
			name:         "user does not exist",
			userID:       "100500",
			wantAccrued:  model.NewAmount(0, 0),
			wantWithdraw: model.NewAmount(0, 0),
			wantErr:      false, // нет записей — не ошибка
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			accrued, withdrawn, err := repo.GetBalance(ctx, tt.userID)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantAccrued, accrued)
				assert.Equal(t, tt.wantWithdraw, withdrawn)
			}
		})
	}
}
