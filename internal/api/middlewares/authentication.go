package middlewares

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/talx-hub/gopher-bonus/internal/model"
	"github.com/talx-hub/gopher-bonus/internal/utils/auth"
)

func Authentication(secret []byte, log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		authFunc := func(w http.ResponseWriter, r *http.Request) {
			jwtCookie, err := r.Cookie("jwt-token")
			if err != nil {
				log.LogAttrs(r.Context(),
					slog.LevelError,
					"failed to find token in request",
				)
				http.Error(w, "authentication failed", http.StatusUnauthorized)
				return
			}

			tokenStr := jwtCookie.Value
			claims, err := auth.CheckToken(tokenStr, secret)
			if err != nil {
				log.LogAttrs(r.Context(),
					slog.LevelError,
					"authentication failed",
					slog.Any(model.KeyLoggerError, err),
					slog.String("token", tokenStr),
				)
				http.Error(w, "authentication failed", http.StatusUnauthorized)
				return
			}

			initial := r.Context()
			idCtx := context.WithValue(
				initial, model.KeyContextUserID, claims.UserID)

			rWithID := r.WithContext(idCtx)
			next.ServeHTTP(w, rWithID)
		}
		return http.HandlerFunc(authFunc)
	}
}
