package repo

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/talx-hub/gopher-bonus/internal/model"
)

func TestMain(m *testing.M) {
	log := slog.Default()
	code, err := runMain(m, log)
	if err != nil {
		log.ErrorContext(context.TODO(),
			"unexpected test failure",
			slog.Any(model.KeyLoggerError, err),
		)
	}
	os.Exit(code)
}
