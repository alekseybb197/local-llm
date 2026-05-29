package middleware

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/oauth2-proxy/local-llm-proxy/config"
	"github.com/oauth2-proxy/local-llm-proxy/db"
	"github.com/oauth2-proxy/local-llm-proxy/models"
)

type Middleware struct {
	config       *config.Config
	database     *db.Database
	rateLimiter  *RateLimiter
}

type RateLimiter struct {
	mu       sync.Mutex
	requests map[string]time.Time
	limit    int
	window   time.Duration
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		requests: make(map[string]time.Time),
		limit:    limit,
		window:   window,
	}
}

func (rl *RateLimiter) IsAllowed(clientIP string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-rl.window)

	var currentRequests []string
	for ip, timestamp := range rl.requests {
		if timestamp.After(windowStart) {
			currentRequests = append(currentRequests, ip)
		}
	}

	if len(currentRequests) >= rl.limit {
		return false
	}

	rl.requests[clientIP] = now
	return true
}

func NewMiddleware(database *db.Database, config *config.Config) *Middleware {
	return &Middleware{
		config:       config,
		database:     database,
		rateLimiter:  NewRateLimiter(config.RateLimit.RPM, config.RateLimit.Window),
	}
}

func (m *Middleware) Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for public endpoints
		skipAuth := m.skipAuth(r.URL.Path)
		if skipAuth {
			next.ServeHTTP(w, r)
			return
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Missing Authorization header", http.StatusUnauthorized)
			log.Printf("[%s] Missing Authorization header - %s", r.Method, r.URL.Path)
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "Invalid Authorization header format", http.StatusUnauthorized)
			log.Printf("[%s] Invalid Authorization header format - %s", r.Method, r.URL.Path)
			return
		}

		accessToken := parts[1]

		token, err := m.validateToken(accessToken)
		if err != nil {
			http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
			log.Printf("[%s] Invalid token: %v - %s", r.Method, err, r.URL.Path)
			return
		}

		clientIP := getClientIP(r)
		if !m.rateLimiter.IsAllowed(clientIP) {
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			log.Printf("[%s] Rate limit exceeded - %s", r.Method, r.URL.Path)
			return
		}

		ctx := context.WithValue(r.Context(), "token", token)
		ctx = context.WithValue(ctx, "clientID", token.ClientID)
		ctx = context.WithValue(ctx, "scopes", token.Scopes)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *Middleware) skipAuth(path string) bool {
	skipPaths := []string{
		"/health",
		"/ready",
		"/oauth/authorize",
		"/oauth/callback",
		"/oauth/token",
		"/oauth/client/register",
		"/oauth/client/delete",
		"/oauth/client/info",
	}
	for _, skip := range skipPaths {
		if strings.HasPrefix(path, skip) {
			return true
		}
	}
	return false
}

func (m *Middleware) validateToken(accessToken string) (*models.Token, error) {
	token, err := m.database.GetTokenByAccessToken(accessToken)
	if err != nil {
		return nil, fmt.Errorf("token not found: %w", err)
	}

	valid, err := m.database.IsTokenValid(token.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to validate token: %w", err)
	}
	if !valid {
		return nil, fmt.Errorf("token expired")
	}

	return token, nil
}

func getClientIP(r *http.Request) string {
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}

	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	ip := r.RemoteAddr
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	return ip
}

func (m *Middleware) Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		wrapped := &responseWriter{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		log.Printf("%s %s %d %.2fs", r.Method, r.URL.Path, wrapped.status, time.Since(startTime).Seconds())
	})
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func (m *Middleware) Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("Panic recovered: %v", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
