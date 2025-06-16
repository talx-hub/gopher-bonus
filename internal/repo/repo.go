package repo

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/talx-hub/gopher-bonus/internal/model"
	"github.com/talx-hub/gopher-bonus/internal/model/order"
	"github.com/talx-hub/gopher-bonus/internal/model/user"
	"github.com/talx-hub/gopher-bonus/internal/repo/internal/db"
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
		ID:           strconv.FormatInt(int64(u.IDUser), 10),
		LoginHash:    u.HashLogin,
		PasswordHash: u.HashPassword,
	}, nil
}
func (r *UserRepository) FindByID(ctx context.Context, id string,
) (user.User, error) {
	queries := db.New(r.pool)

	idParsed, err := strconv.ParseInt(id, 10, 32)
	if err != nil {
		return user.User{},
			fmt.Errorf("failed to convert user ID to int32: %w", err)
	}

	u, err := queries.FindUserByID(ctx, int32(idParsed))
	if err != nil {
		return user.User{},
			fmt.Errorf("failed to find user by ID in DB: %w", err)
	}

	return user.User{
		ID:           strconv.FormatInt(int64(u.IDUser), 10),
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

func (r *OrderRepository) Create(ctx context.Context, o *order.Order) error {
	userInt, err := strconv.ParseInt(o.UserID, 10, 32)
	if err != nil {
		return fmt.Errorf("failed to parse userID from %s: %w", o.UserID, err)
	}

	queries := db.New(r.pool)
	if err = queries.CreateOrder(ctx, db.CreateOrderParams{
		IDUser:     int32(userInt),
		OrderNo:    o.ID,
		UploadedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		NameStatus: string(o.Status),
	}); err != nil {
		return fmt.Errorf("failed to create order in DB: %w", err)
	}

	return nil
}

func (r *OrderRepository) FindByID(ctx context.Context, orderID string,
) (order.Order, error) {
	queries := db.New(r.pool)
	userID, err := queries.FindOrderByID(ctx, orderID)
	if err != nil {
		return order.Order{},
			fmt.Errorf("failed to find userID by orderID %s: %w", orderID, err)
	}
	return order.Order{
		UserID: strconv.FormatInt(int64(userID), 10),
	}, nil
}

func (r *OrderRepository) ListByUserID(ctx context.Context, userID string,
) ([]order.Order, error) {
	userInt, err := strconv.ParseInt(userID, 10, 32)
	if err != nil {
		return nil,
			fmt.Errorf("failed to parse userID from %s: %w", userID, err)
	}
	queries := db.New(r.pool)
	ordersRaw, err := queries.ListByUserID(ctx, int32(userInt))
	if err != nil {
		return nil,
			fmt.Errorf("failed to list orders by userID %s: %w", userID, err)
	}

	orders := make([]order.Order, len(ordersRaw))
	for i, or := range ordersRaw {
		accrual, err := model.FromPGNumeric(or.Accrual)
		if err != nil {
			r.log.LogAttrs(ctx,
				slog.LevelError,
				"invalid accrual from DB",
				slog.Any("accrual", or.Accrual),
				slog.Any(model.KeyLoggerError, err),
			)
		}
		orders[i] = order.Order{
			UploadedAt: or.UploadedAt.Time,
			Status:     order.Status(or.NameStatus),
			ID:         or.OrderNo,
			UserID:     userID,
			Accrual:    accrual,
		}
	}

	return orders, nil
}

func (r *OrderRepository) UpdateOrder(ctx context.Context, o *order.Order) error {
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
	if err = queries.AddAccruedAmount(
		ctx,
		db.AddAccruedAmountParams{
			OrderNo: o.ID,
			Amount:  o.Accrual.ToPGNumeric(),
		},
	); err != nil {
		return fmt.Errorf("failed to add accrual to order %s: %w", o.ID, err)
	}

	if err = queries.UpdateStatus(
		ctx,
		db.UpdateStatusParams{
			NameStatus: string(o.Status),
			OrderNo:    o.ID,
		},
	); err != nil {
		return fmt.Errorf("failed to update status->(%s) for order %s: %w",
			string(o.Status), o.ID, err)
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit TX: %w", err)
	}

	return nil
}
