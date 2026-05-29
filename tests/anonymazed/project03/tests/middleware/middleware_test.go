package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/oauth2-proxy/local-llm-proxy/config"
	"github.com/oauth2-proxy/local-llm-proxy/db"
	"github.com/oauth2-proxy/local-llm-proxy/models"
)

func TestMiddlewareAuth(t *testing.T) {
	t.Run("should allow request with valid token", func(t *testing.T) {
		dbPath := "./test_mw/auth_valid.db"
		dbPath = "./test_db/auth_valid.db"
		dbPath = "./test_db/mw_valid.db"
		os.Remove(dbPath + "-wal")
		os.Remove(dbPath + "-shm")

		database, err := db.NewDatabase(dbPath)
		if err != nil {
			t.Fatalf("Failed to create database: %v", err)
		}
		defer database.Close()

		// Create a valid token
		token := &models.Token{
			AccessToken:  "valid-token-123",
			ExpiresAt:    time.Now().Add(time.Hour),
			TokenType:    "Bearer",
			Scopes:       "openid profile",
			Subject:      "user",
			ClientID:     "test-client",
			CreatedAt:    time.Now(),
		}

		if err := database.CreateToken(token); err != nil {
			t.Fatalf("Failed to create token: %v", err)
		}

		cfg := &config.Config{
			Database: config.DatabaseConfig{
				Path: dbPath,
			},
		}

		mw := NewMiddleware(database, cfg)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})

		protectedHandler := mw.Auth(handler)

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer valid-token-123")

		rec := httptest.NewRecorder()
		protectedHandler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}
	})

	t.Run("should reject request without token", func(t *testing.T) {
		dbPath := "./test_db/mw_no_token.db"
		os.Remove(dbPath + "-wal")
		os.Remove(dbPath + "-shm")

		database, err := db.NewDatabase(dbPath)
		if err != nil {
			t.Fatalf("Failed to create database: %v", err)
		}
		defer database.Close()

		cfg := &config.Config{
			Database: config.DatabaseConfig{
				Path: dbPath,
			},
		}

		mw := NewMiddleware(database, cfg)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})

		protectedHandler := mw.Auth(handler)

		req := httptest.NewRequest("GET", "/test", nil)

		rec := httptest.NewRecorder()
		protectedHandler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", rec.Code)
		}
	})

	t.Run("should reject request with invalid token", func(t *testing.T) {
		dbPath := "./test_db/mw_invalid.db"
		os.Remove(dbPath + "-wal")
		os.Remove(dbPath + "-shm")

		database, err := db.NewDatabase(dbPath)
		if err != nil {
			t.Fatalf("Failed to create database: %v", err)
		}
		defer database.Close()

		cfg := &config.Config{
			Database: config.DatabaseConfig{
				Path: dbPath,
			},
		}

		mw := NewMiddleware(database, cfg)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})

		protectedHandler := mw.Auth(handler)

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")

		rec := httptest.NewRecorder()
		protectedHandler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", rec.Code)
		}
	})

	t.Run("should skip auth for public endpoints", func(t *testing.T) {
		dbPath := "./test_db/mw_public.db"
		os.Remove(dbPath + "-wal")
		os.Remove(dbPath + "-shm")

		database, err := db.NewDatabase(dbPath)
		if err != nil {
			t.Fatalf("Failed to create database: %v", err)
		}
		defer database.Close()

		cfg := &config.Config{
			Database: config.DatabaseConfig{
				Path: dbPath,
			},
		}

		mw := NewMiddleware(database, cfg)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})

		protectedHandler := mw.Auth(handler)

		req := httptest.NewRequest("GET", "/health", nil)

		rec := httptest.NewRecorder()
		protectedHandler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200 for public endpoint, got %d", rec.Code)
		}
	})
}

func TestMiddlewareRecovery(t *testing.T) {
	t.Run("should recover from panic", func(t *testing.T) {
		dbPath := "./test_db/mw_recovery.db"
		os.Remove(dbPath + "-wal")
		os.Remove(dbPath + "-shm")

		database, err := db.NewDatabase(dbPath)
		if err != nil {
			t.Fatalf("Failed to create database: %v", err)
		}
		defer database.Close()

		cfg := &config.Config{
			Database: config.DatabaseConfig{
				Path: dbPath,
			},
		}

		mw := NewMiddleware(database, cfg)

		handler := mw.Recovery(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			panic("test panic")
		}))

		req := httptest.NewRequest("GET", "/test", nil)

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		// Should not panic, should return 500
		if rec.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500 on panic recovery, got %d", rec.Code)
		}
	})
}

func TestMiddlewareLogging(t *testing.T) {
	t.Run("should log requests", func(t *testing.T) {
		dbPath := "./test_db/mw_logging.db"
		os.Remove(dbPath + "-wal")
		os.Remove(dbPath + "-shm")

		database, err := db.NewDatabase(dbPath)
		if err != nil {
			t.Fatalf("Failed to create database: %v", err)
		}
		defer database.Close()

		cfg := &config.Config{
			Database: config.DatabaseConfig{
				Path: dbPath,
			},
		}

		mw := NewMiddleware(database, cfg)

		handler := mw.Logging(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		}))

		req := httptest.NewRequest("GET", "/test", nil)

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}
	})
}

func TestGetClientIP(t *testing.T) {
	t.Run("should get IP from X-Forwarded-For", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Forwarded-For", "192.168.1.1, 10.0.0.1")

		ip := getClientIP(req)
		if ip != "192.168.1.1" {
			t.Errorf("Expected 192.168.1.1, got %s", ip)
		}
	})

	t.Run("should get IP from X-Real-IP", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Real-IP", "10.0.0.1")

		ip := getClientIP(req)
		if ip != "10.0.0.1" {
			t.Errorf("Expected 10.0.0.1, got %s", ip)
		}
	})

	t.Run("should get IP from RemoteAddr", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "127.0.0.1:12345"

		ip := getClientIP(req)
		if ip != "127.0.0.1" {
			t.Errorf("Expected 127.0.0.1, got %s", ip)
		}
	})
}

func TestResponseWriter(t *testing.T) {
	t.Run("should capture status code", func(t *testing.T) {
		rec := httptest.NewRecorder()
		rw := &responseWriter{ResponseWriter: rec}

		rw.WriteHeader(http.StatusForbidden)

		if rw.status != http.StatusForbidden {
			t.Errorf("Expected status 403, got %d", rw.status)
		}
	})
}
