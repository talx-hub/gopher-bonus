package repo

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/talx-hub/gopher-bonus/internal/model"
	"github.com/talx-hub/gopher-bonus/internal/model/order"
	"github.com/talx-hub/gopher-bonus/internal/repo/internal/db"
	"github.com/talx-hub/gopher-bonus/internal/serviceerrs"
)

type OrderRepository struct {
	DB
}

func NewOrderRepository(pool connectionPool, log *slog.Logger) *OrderRepository {
	return &OrderRepository{
		DB{
			pool: pool,
			log:  log,
		},
	}
}

func (r *OrderRepository) CreateOrder(ctx context.Context, o *order.Order) error {
	createOrderCb := func() (struct{}, error) {
		if o.Type == order.TypeAccrual {
			queries := db.New(r.pool)
			if err := queries.CreateAccrual(ctx, db.CreateAccrualParams{
				IDUser:     o.UserID,
				NameOrder:  o.ID,
				UploadedAt: pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
				NameStatus: string(o.Status),
			}); err != nil {
				return struct{}{}, fmt.Errorf("failed to create order in DB: %w", err)
			}

			return struct{}{}, nil
		}

		withdraw := func(_ context.Context, tx connectionPool) (any, error) {
			accrued, withdrawn, err := r.getBalanceTX(ctx, tx, o.UserID)
			if err != nil {
				return struct{}{}, fmt.Errorf(
					"failed to get balance for userID %s: %w", o.UserID, err)
			}
			if accrued.TotalKopecks() < o.Amount.TotalKopecks()+withdrawn.TotalKopecks() {
				return struct{}{}, serviceerrs.ErrInsufficientFunds
			}

			queries := db.New(tx)
			if err := queries.CreateWithdrawal(ctx, db.CreateWithdrawalParams{
				IDUser:      o.UserID,
				NameOrder:   o.ID,
				ProcessedAt: pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
				Amount:      o.Amount.ToPGNumeric(),
			}); err != nil {
				return struct{}{}, fmt.Errorf("failed to withdraw in DB: %w", err)
			}
			return struct{}{}, nil
		}

		_, err := WithTX[struct{}](ctx, r.pool, r.log, withdraw)
		if err != nil {
			return struct{}{}, err //nolint: wrapcheck // error from wrapped function
		}
		return struct{}{}, nil
	}

	_, err := WithRetry[struct{}](createOrderCb, 0)
	return err
}

func (r *OrderRepository) FindUserIDByAccrualID(ctx context.Context, accrualID string,
) (string, error) {
	findLogic := func() (string, error) {
		queries := db.New(r.pool)
		userID, err := queries.FindOrderByID(ctx, accrualID)
		if err != nil {
			return "", fmt.Errorf("failed to find userID by orderID %s: %w", accrualID, err)
		}
		return userID, nil
	}

	return WithRetry[string](findLogic, 0) //nolint: wrapcheck // error from wrapped function
}

func (r *OrderRepository) ListOrdersByUser(ctx context.Context,
	userID string, tp order.Type,
) ([]order.Order, error) {
	if len(userID) == 0 {
		return nil, errors.New("failed to list orders for empty user: userID must be not empty")
	}

	listLogic := func() ([]order.Order, error) {
		if tp == order.TypeAccrual {
			return listAccruals(ctx, userID, r.pool, r.log)
		}
		return listWithdrawals(ctx, userID, r.pool, r.log)
	}

	return WithRetry[[]order.Order](listLogic, 0) //nolint: wrapcheck // error from wrapped function
}

func listAccruals(ctx context.Context,
	userID string, pool connectionPool, log *slog.Logger,
) ([]order.Order, error) {
	queries := db.New(pool)
	ordersRaw, err := queries.ListAccrualsByUserID(ctx, userID)
	if err != nil {
		return nil,
			fmt.Errorf("failed to list orders by userID %s: %w", userID, err)
	}

	orders := make([]order.Order, len(ordersRaw))
	for i, or := range ordersRaw {
		accrual, err := model.FromPGNumeric(or.Accrual)
		if err != nil {
			log.LogAttrs(ctx,
				slog.LevelError,
				"invalid accrual from DB",
				slog.Any("accrual", or.Accrual),
				slog.Any(model.KeyLoggerError, err),
			)
		}
		orders[i] = order.Order{
			CreatedAt: or.UploadedAt.Time,
			Status:    order.Status(or.NameStatus),
			ID:        or.NameOrder,
			UserID:    userID,
			Amount:    accrual,
		}
	}

	return orders, nil
}

func listWithdrawals(ctx context.Context,
	userID string, pool connectionPool, log *slog.Logger,
) ([]order.Order, error) {
	queries := db.New(pool)
	ordersRaw, err := queries.ListWithdrawalsByUser(ctx, userID)
	if err != nil {
		return nil,
			fmt.Errorf("failed to list withdrawals by userID %s: %w", userID, err)
	}

	orders := make([]order.Order, len(ordersRaw))
	for i, or := range ordersRaw {
		withdrew, err := model.FromPGNumeric(or.Amount)
		if err != nil {
			log.LogAttrs(ctx,
				slog.LevelError,
				"invalid withdrawal from DB",
				slog.Any("withdrawal", or.Amount),
				slog.Any(model.KeyLoggerError, err),
			)
		}
		orders[i] = order.Order{
			CreatedAt: or.ProcessedAt.Time,
			ID:        or.NameOrder,
			UserID:    userID,
			Amount:    withdrew,
		}
	}

	return orders, nil
}

func (r *OrderRepository) UpdateAccrualStatus(ctx context.Context, o *order.Order) error {
	updateFn := func() (struct{}, error) {
		queries := db.New(r.pool)
		res, err := queries.UpdateAccrualStatus(
			ctx,
			db.UpdateAccrualStatusParams{
				NameStatus: string(o.Status),
				NameOrder:  o.ID,
			},
		)
		if err != nil || res.RowsAffected() == 0 {
			return struct{}{}, fmt.Errorf("failed to update status->(%s) for order %s: %w",
				string(o.Status), o.ID, err)
		}
		return struct{}{}, nil
	}

	_, err := WithRetry[struct{}](updateFn, 0)
	return err //nolint: wrapcheck // error from wrapped function
}

func (r *OrderRepository) GetBalance(ctx context.Context, userID string,
) (model.Amount, model.Amount, error) {
	type Balance struct {
		accrued   model.Amount
		withdrawn model.Amount
	}
	getBalance := func(ctx context.Context, tx connectionPool) (any, error) {
		accrued, withdrawn, err := r.getBalanceTX(ctx, tx, userID)
		if err != nil {
			return Balance{}, fmt.Errorf("failed to get balance for user %s: %w", userID, err)
		}
		return Balance{
			accrued:   accrued,
			withdrawn: withdrawn,
		}, nil
	}

	runWithTX := func() (Balance, error) {
		return WithTX[Balance](ctx, r.pool, r.log, getBalance)
	}

	balance, err := WithRetry[Balance](runWithTX, 0)
	if err != nil {
		return model.Amount{}, model.Amount{}, err //nolint: wrapcheck // error from wrapped function
	}

	currentSum := model.NewAmount(
		0,
		balance.accrued.TotalKopecks()-balance.withdrawn.TotalKopecks(),
	)
	return currentSum, balance.withdrawn, nil
}

func (r *OrderRepository) getBalanceTX(ctx context.Context,
	tx connectionPool, userID string,
) (model.Amount, model.Amount, error) {
	queries := db.New(tx)

	accrued, err := getAmount(ctx, queries.GetAccruedAmount, userID)
	if err != nil {
		return model.Amount{}, model.Amount{},
			fmt.Errorf("failed to get accruals %w", err)
	}
	withdrawn, err := getAmount(ctx, queries.GetWithdrawnAmount, userID)
	if err != nil {
		return model.Amount{}, model.Amount{},
			fmt.Errorf("failed to get withdrawals: %w", err)
	}

	return accrued, withdrawn, nil
}

type amountQuery func(context.Context, string) (pgtype.Numeric, error)

func getAmount(ctx context.Context, amountQuery amountQuery, userID string) (model.Amount, error) {
	val, err := amountQuery(ctx, userID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return model.Amount{}, err
	}
	if err == nil {
		return model.FromPGNumeric(val) //nolint: wrapcheck // error from wrapped function
	}
	return model.NewAmount(0, 0), nil
}
