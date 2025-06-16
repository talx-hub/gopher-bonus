package repo

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/talx-hub/gopher-bonus/internal/model"
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

type UserRepository = DB

func NewUserRepository(pool connectionPool, log *slog.Logger) *UserRepository {
	return &UserRepository{
		pool: pool,
		log:  log,
	}
}

func (r *UserRepository) Create(ctx context.Context, u *user.User) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin TX: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
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
	u, err := queries.FindByLogin(ctx, loginHash)
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

	u, err := queries.FindByID(ctx, int32(idParsed))
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
