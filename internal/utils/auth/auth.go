package auth

import (
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/talx-hub/gopher-bonus/internal/serviceerrs"
)

const TokenExpire = 3 * time.Hour

type Claims struct {
	jwt.RegisteredClaims
	UserID string
}

func buildJWTString(id string, secret []byte) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256,
		Claims{
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(TokenExpire)),
			},
			UserID: id,
		},
	)
	tokenString, err := token.SignedString(secret)
	if err != nil {
		return "", fmt.Errorf("JWT signing: %w", err)
	}
	return tokenString, nil
}

func Authenticate(id string, secret []byte) (http.Cookie, error) {
	jwtString, err := buildJWTString(id, secret)
	if err != nil {
		return http.Cookie{}, fmt.Errorf("authentication failed: %w", err)
	}
	return http.Cookie{
		Name:     "jwt-token",
		Value:    jwtString,
		Path:     "",
		MaxAge:   0,
		HttpOnly: true,
	}, nil
}

func CheckToken(tokenString string, secret []byte) (Claims, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(
		tokenString, claims,
		func(token *jwt.Token) (interface{}, error) {
			return secret, nil
		})
	if err != nil {
		return Claims{}, fmt.Errorf("failed to parse token %w", err)
	}
	tokenExpired := claims.ExpiresAt.Before(time.Now())
	if tokenExpired {
		return Claims{}, serviceerrs.ErrTokenExpired
	}

	return *claims, nil
}
