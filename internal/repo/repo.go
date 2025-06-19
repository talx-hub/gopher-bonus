package repo

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/talx-hub/gopher-bonus/internal/model"
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

type dbLogic func(ctx context.Context, tx connectionPool) (any, error)

func WithTX[T any](ctx context.Context,
	pool connectionPool, log *slog.Logger, f dbLogic,
) (T, error) {
	var zero T

	tx, err := pool.Begin(ctx)
	if err != nil {
		return zero, fmt.Errorf("failed to begin TX: %w", err)
	}
	defer func() {
		if rbErr := tx.Rollback(ctx); rbErr != nil && !errors.Is(rbErr, pgx.ErrTxClosed) {
			log.LogAttrs(ctx,
				slog.LevelError,
				"failed to rollback TX",
				slog.Any(model.KeyLoggerError, err),
			)
		}
	}()

	res, err := f(ctx, tx)
	if err != nil {
		return zero, err //nolint: wrapcheck // error from wrapped function
	}

	if err = tx.Commit(ctx); err != nil {
		return zero, fmt.Errorf("failed to commit TX: %w", err)
	}

	r, ok := res.(T)
	if !ok {
		return zero, fmt.Errorf("failed to convert any to %T", zero)
	}
	return r, nil
}
