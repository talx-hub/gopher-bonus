package repo

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/talx-hub/gopher-bonus/internal/model"
	"github.com/talx-hub/gopher-bonus/internal/model/order"
	"github.com/talx-hub/gopher-bonus/internal/serviceerrs"
)

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
		wantSum      model.Amount
		wantWithdraw model.Amount
		wantErr      bool
	}{
		{
			name:         "no accruals and no withdrawals",
			userID:       "2",
			wantSum:      model.NewAmount(0, 0),
			wantWithdraw: model.NewAmount(0, 0),
			wantErr:      false,
		},
		{
			name:         "accruals > 0, no withdrawals",
			userID:       "1",
			wantSum:      model.NewAmount(150, 50),
			wantWithdraw: model.NewAmount(0, 0),
			wantErr:      false,
		},
		{
			name:         "equal accruals and withdrawals",
			userID:       "3",
			wantSum:      model.NewAmount(0, 0),
			wantWithdraw: model.NewAmount(100, 0),
			wantErr:      false,
		},
		{
			name:         "more withdrawn than accrued",
			userID:       "4",
			wantSum:      model.NewAmount(-20, 0),
			wantWithdraw: model.NewAmount(70, 0),
			wantErr:      false,
		},
		{
			name:         "has both accruals and withdrawals",
			userID:       "5",
			wantSum:      model.NewAmount(199, 50),
			wantWithdraw: model.NewAmount(100, 50),
			wantErr:      false,
		},
		{
			name:         "user does not exist",
			userID:       "100500",
			wantSum:      model.NewAmount(0, 0),
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
				assert.Equal(t, tt.wantSum, accrued)
				assert.Equal(t, tt.wantWithdraw, withdrawn)
			}
		})
	}
}
