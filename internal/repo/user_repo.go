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

	_, err := WithTX[struct{}](ctx, r.pool, r.log, createLogic)
	if err != nil {
		return err //nolint: wrapcheck // error from wrapped function
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
