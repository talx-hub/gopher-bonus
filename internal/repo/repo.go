package repo

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/talx-hub/gopher-bonus/internal/model"
	"github.com/talx-hub/gopher-bonus/internal/model/order"
	"github.com/talx-hub/gopher-bonus/internal/model/user"
	"github.com/talx-hub/gopher-bonus/internal/repo/internal/db"
	"github.com/talx-hub/gopher-bonus/internal/serviceerrs"
)

type connectionPool interface {
	Begin(context.Context) (pgx.Tx, error)
	Exec(context.Context, string, ...interface{}) (pgconn.CommandTag, error)
	Query(context.Context, string, ...interface{}) (pgx.Rows, error)
	QueryRow(context.Context, string, ...interface{}) pgx.Row
}

type DB struct {
	pool connectionPool
	log  *slog.Logger
}

type UserRepository struct {
	DB
}

func NewUserRepository(pool connectionPool, log *slog.Logger) *UserRepository {
	return &UserRepository{
		DB{
			pool: pool,
			log:  log,
		},
	}
}

func (r *UserRepository) Create(ctx context.Context, u *user.User) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin TX: %w", err)
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil {
			r.log.LogAttrs(ctx,
				slog.LevelError,
				"failed to rollback TX",
				slog.Any(model.KeyLoggerError, err),
			)
		}
	}()

	queries := db.New(tx)
	id, err := queries.InsertLoginHash(ctx, u.LoginHash)
	if err != nil {
		return fmt.Errorf("failed to insert user login hash: %w", err)
	}

	passParams := db.InsertPasswordHashParams{
		IDUser:       id,
		HashPassword: u.PasswordHash,
	}
	if err = queries.InsertPasswordHash(ctx, passParams); err != nil {
		return fmt.Errorf("failed to insert user password hash: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit TX: %w", err)
	}

	return nil
}

func (r *UserRepository) Exists(ctx context.Context, loginHash string) bool {
	queries := db.New(r.pool)
	exists, err := queries.Exists(ctx, loginHash)
	if err != nil {
		r.log.LogAttrs(ctx,
			slog.LevelError,
			"failed to check if loginHash exists in DB",
			slog.Any(model.KeyLoggerError, err),
		)
		return false
	}

	return exists
}

func (r *UserRepository) FindByLogin(ctx context.Context, loginHash string,
) (user.User, error) {
	queries := db.New(r.pool)
	u, err := queries.FindUserByLogin(ctx, loginHash)
	if err != nil {
		return user.User{},
			fmt.Errorf("failed to find user by login in DB: %w", err)
	}

	return user.User{
		ID:           u.IDUser,
		LoginHash:    u.HashLogin,
		PasswordHash: u.HashPassword,
	}, nil
}
func (r *UserRepository) FindByID(ctx context.Context, id string,
) (user.User, error) {
	queries := db.New(r.pool)

	u, err := queries.FindUserByID(ctx, id)
	if err != nil {
		return user.User{},
			fmt.Errorf("failed to find user by ID in DB: %w", err)
	}

	return user.User{
		ID:           u.IDUser,
		LoginHash:    u.HashLogin,
		PasswordHash: u.HashPassword,
	}, nil
}

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
	if o.Type == order.TypeAccrual {
		queries := db.New(r.pool)
		if err := queries.CreateAccrual(ctx, db.CreateAccrualParams{
			IDUser:     o.UserID,
			NameOrder:  o.ID,
			UploadedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
			NameStatus: string(o.Status),
		}); err != nil {
			return fmt.Errorf("failed to create order in DB: %w", err)
		}

		return nil
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin TX: %w", err)
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil {
			r.log.LogAttrs(ctx,
				slog.LevelError,
				"failed to rollback TX",
				slog.Any(model.KeyLoggerError, err),
			)
		}
	}()

	accrued, withdrawn, err := r.getBalanceHelper(ctx, tx, o.UserID)
	if err != nil {
		return fmt.Errorf(
			"failed to get balance for userID %s: %w", o.UserID, err)
	}
	if accrued.TotalKopecks() < o.Amount.TotalKopecks()+withdrawn.TotalKopecks() {
		return serviceerrs.ErrInsufficientFunds
	}

	queries := db.New(tx)
	if err := queries.CreateWithdrawal(ctx, db.CreateWithdrawalParams{
		IDUser:      o.UserID,
		NameOrder:   o.ID,
		ProcessedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		Amount:      o.Amount.ToPGNumeric(),
	}); err != nil {
		return fmt.Errorf("failed to withdraw in DB: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit TX: %w", err)
	}

	return nil
}

func (r *OrderRepository) FindUserIDByAccrualID(ctx context.Context, accrualID string,
) (string, error) {
	queries := db.New(r.pool)
	userID, err := queries.FindOrderByID(ctx, accrualID)
	if err != nil {
		return "",
			fmt.Errorf("failed to find userID by orderID %s: %w", accrualID, err)
	}
	return userID, nil
}

func (r *OrderRepository) ListOrdersByUser(ctx context.Context,
	userID string, tp order.Type,
) ([]order.Order, error) {
	if tp == order.TypeAccrual {
		return listAccruals(ctx, userID, r.pool, r.log)
	}
	return listWithdrawals(ctx, userID, r.pool, r.log)
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
		withdrawed, err := model.FromPGNumeric(or.Amount)
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
			Amount:    withdrawed,
		}
	}

	return orders, nil
}

func (r *OrderRepository) UpdateAccrualStatus(ctx context.Context, o *order.Order) error {
	queries := db.New(r.pool)
	res, err := queries.UpdateAccrualStatus(
		ctx,
		db.UpdateAccrualStatusParams{
			NameStatus: string(o.Status),
			NameOrder:  o.ID,
		},
	)
	if err != nil || res.RowsAffected() == 0 {
		return fmt.Errorf("failed to update status->(%s) for order %s: %w",
			string(o.Status), o.ID, err)
	}
	return nil
}

func (r *OrderRepository) GetBalance(ctx context.Context, userID string,
) (model.Amount, model.Amount, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return model.Amount{}, model.Amount{},
			fmt.Errorf("failed to begin TX: %w", err)
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil {
			r.log.LogAttrs(ctx,
				slog.LevelError,
				"failed to rollback TX",
				slog.Any(model.KeyLoggerError, err),
			)
		}
	}()

	accrued, withdrawn, err := r.getBalanceHelper(ctx, tx, userID)
	if err != nil {
		return model.Amount{}, model.Amount{},
			fmt.Errorf("failed to get balance for user %s: %w", userID, err)
	}

	if err = tx.Commit(ctx); err != nil {
		return model.Amount{}, model.Amount{},
			fmt.Errorf("failed to commit TX: %w", err)
	}

	return accrued, withdrawn, nil
}

func (r *OrderRepository) getBalanceHelper(ctx context.Context,
	tx connectionPool, userID string,
) (model.Amount, model.Amount, error) {
	queries := db.New(tx)
	a, err := queries.GetAccruedAmount(ctx, userID)
	if err != nil {
		return model.Amount{}, model.Amount{},
			fmt.Errorf("failed to get accruals %w", err)
	}
	accruedAllTime, err := model.FromPGNumeric(a)
	if err != nil {
		return model.Amount{}, model.Amount{},
			fmt.Errorf("failed to convert accruals from pgtype.Numeric: %w", err)
	}

	wd, err := queries.GetWithdrawnAmount(ctx, userID)
	if err != nil {
		return model.Amount{}, model.Amount{},
			fmt.Errorf("failed to get withdrawals: %w", err)
	}
	withdrawnAllTime, err := model.FromPGNumeric(wd)
	if err != nil {
		return model.Amount{}, model.Amount{},
			fmt.Errorf("failed to convert withdrawals from pgtype.Numeric: %w", err)
	}

	return accruedAllTime, withdrawnAllTime, nil
}
