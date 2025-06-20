package repo

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/talx-hub/gopher-bonus/internal/model"
	"github.com/talx-hub/gopher-bonus/internal/model/user"
	"github.com/talx-hub/gopher-bonus/internal/repo/internal/db"
)

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
	createLogic := func(ctx context.Context, tx connectionPool) (any, error) {
		queries := db.New(tx)
		id, err := queries.InsertLoginHash(ctx, u.LoginHash)
		if err != nil {
			return struct{}{}, fmt.Errorf("failed to insert user login hash: %w", err)
		}

		passParams := db.InsertPasswordHashParams{
			IDUser:       id,
			HashPassword: u.PasswordHash,
		}
		if err = queries.InsertPasswordHash(ctx, passParams); err != nil {
			return struct{}{}, fmt.Errorf("failed to insert user password hash: %w", err)
		}

		return struct{}{}, nil
	}

	createWithTX := func() (struct{}, error) {
		return WithTX[struct{}](ctx, r.pool, r.log, createLogic)
	}

	_, err := WithRetry[struct{}](createWithTX, 0)
	if err != nil {
		return err //nolint: wrapcheck // error from wrapped function
	}

	return nil
}

func (r *UserRepository) Exists(ctx context.Context, loginHash string) bool {
	existsLogic := func() (bool, error) {
		queries := db.New(r.pool)
		exists, err := queries.Exists(ctx, loginHash)
		if err != nil {
			r.log.LogAttrs(ctx,
				slog.LevelError,
				"failed to check if loginHash exists in DB",
				slog.Any(model.KeyLoggerError, err),
			)
			return false, nil
		}
		return exists, nil
	}

	exists, _ := WithRetry[bool](existsLogic, 0)
	return exists
}

// nolint: dupl // ide bug, methods are different
func (r *UserRepository) FindByLogin(ctx context.Context, loginHash string,
) (user.User, error) {
	findByLoginLogic := func() (user.User, error) {
		queries := db.New(r.pool)
		u, err := findWrapper[db.FindUserByLoginRow](ctx,
			queries.FindUserByLogin, loginHash)

		return user.User{
			ID:           u.IDUser,
			LoginHash:    u.HashLogin,
			PasswordHash: u.HashPassword,
		}, err //nolint: wrapcheck // error from wrapped function
	}

	u, err := WithRetry[user.User](findByLoginLogic, 0)
	if err != nil {
		return user.User{}, err //nolint: wrapcheck // error from wrapped function
	}
	return u, nil
}

// nolint: dupl // ide bug, methods are different
func (r *UserRepository) FindByID(ctx context.Context, id string,
) (user.User, error) {
	findByIDLogic := func() (user.User, error) {
		queries := db.New(r.pool)
		u, err := findWrapper[db.FindUserByIDRow](ctx,
			queries.FindUserByID, id)

		return user.User{
			ID:           u.IDUser,
			LoginHash:    u.HashLogin,
			PasswordHash: u.HashPassword,
		}, err //nolint: wrapcheck // error from wrapped function
	}

	u, err := WithRetry[user.User](findByIDLogic, 0)
	if err != nil {
		return user.User{}, err //nolint: wrapcheck // error from wrapped function
	}
	return u, nil
}

func findWrapper[T db.FindUserByIDRow | db.FindUserByLoginRow](ctx context.Context,
	fn func(context.Context, string) (T, error),
	key string,
) (T, error) {
	var zero T

	u, err := fn(ctx, key)
	if err != nil {
		return zero,
			fmt.Errorf("failed to find user by ID in DB: %w", err)
	}

	return u, nil
}
