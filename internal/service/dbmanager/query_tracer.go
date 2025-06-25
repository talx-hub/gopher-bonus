package dbmanager

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5"
)

type queryTracer struct {
	log *slog.Logger
}

func (t *queryTracer) TraceQueryStart(
	ctx context.Context,
	_ *pgx.Conn,
	data pgx.TraceQueryStartData,
) context.Context {
	t.log.LogAttrs(ctx,
		slog.LevelDebug,
		"Running query",
		slog.String("query", data.SQL),
		slog.Any("args", data.Args),
	)
	return ctx
}

func (t *queryTracer) TraceQueryEnd(
	_ context.Context,
	_ *pgx.Conn,
	_ pgx.TraceQueryEndData,
) {
}
