package logger

import (
	"context"
	"log/slog"
	"os"

	"github.com/talx-hub/gopher-bonus/internal/model"
)

func New(logLevel slog.Level) *slog.Logger {
	return slog.New(
		slog.NewTextHandler(
			os.Stdout,
			&slog.HandlerOptions{Level: logLevel},
		))
}

func WithContext(ctx context.Context, log *slog.Logger) context.Context {
	ctxWithLogger := context.WithValue(ctx, model.KeyContextLogger, log)
	return ctxWithLogger
}

func FromContext(ctx context.Context) *slog.Logger {
	logRaw := ctx.Value(model.KeyContextLogger)
	if logRaw == nil {
		return slog.Default()
	}
	if log, ok := logRaw.(*slog.Logger); ok {
		return log
	}
	return slog.Default()
}
