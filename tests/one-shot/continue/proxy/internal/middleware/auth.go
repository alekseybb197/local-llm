package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/example/oauth2-proxy/internal/config"
)

var (
	ErrInvalidToken     = errors.New("invalid token")
	ErrMissingToken     = errors.New("missing token")
	ErrExpiredToken     = errors.New("token expired")
	ErrInvalidClaims    = errors.New("invalid claims")
	ErrInvalidSignature = errors.New("invalid signature")
)

type ContextKey string

const (
	CtxUserClaims ContextKey = "user_claims"
)

type Claims struct {
	UserID      string   `json:"user_id"`
	AccessToken string   `json:"access_token"`
	Exp         int64    `json:"exp"`
	Iat         int64    `json:"iat"`
	Iss         string   `json:"iss"`
	Aud         string   `json:"aud"`
	Scopes      []string `json:"scopes,omitempty"`
	jwt.RegisteredClaims
}

func NewClaimsFromToken(tokenString string, cfg *config.JWTConfig) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidSignature
		}
		return []byte(cfg.SecretKey), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		// Проверяем срок действия
		if claims.ExpiresAt.Time.Before(time.Now()) {
			return nil, ErrExpiredToken
		}
		// Проверяем audience - jwt.ClaimStrings нужно преобразовать
		if len(claims.Audience) == 0 {
			return nil, ErrInvalidClaims
		}
		// Преобразуем jwt.ClaimStrings в string для проверки
		audienceStr := string(claims.Audience[0])
		if !strings.Contains(audienceStr, cfg.Audience) {
			return nil, ErrInvalidClaims
		}
		return claims, nil
	}

	return nil, ErrInvalidToken
}

// AuthMiddleware middleware для проверки JWT токенов
func AuthMiddleware(cfg *config.JWTConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Missing Authorization header", http.StatusUnauthorized)
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				http.Error(w, "Invalid Authorization header format", http.StatusUnauthorized)
				return
			}

			tokenString := parts[1]
			claims, err := NewClaimsFromToken(tokenString, cfg)
			if err != nil {
				http.Error(w, "Invalid token: "+err.Error(), http.StatusUnauthorized)
				return
			}

			// Добавляем claims в контекст
			ctx := context.WithValue(r.Context(), CtxUserClaims, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserClaims извлекает claims из контекста
func GetUserClaims(ctx context.Context) (*Claims, bool) {
	v := ctx.Value(CtxUserClaims)
	if v == nil {
		return nil, false
	}
	claims, ok := v.(*Claims)
	return claims, ok
}
