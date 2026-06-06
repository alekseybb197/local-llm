package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/example/oauth2-proxy/internal/config"
	"github.com/example/oauth2-proxy/internal/middleware"
	"github.com/stretchr/testify/assert"
)

func TestHTTPHandler(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Port: "8080",
			Host: "localhost",
		},
		OAuth: config.OAuthConfig{
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
			TokenURL:     "http://localhost:8080/oauth/token",
			RedirectURL:  "http://localhost:8080/oauth/callback",
		},
		LLM: config.LLMConfig{
			APIURL: "http://localhost:11434/api",
		},
		JWT: config.JWTConfig{
			SecretKey: "test-secret-key",
			Issuer:    "test-issuer",
			Audience:  "test-audience",
		},
	}

	h := HTTPHandler(cfg, nil, nil)

	// Тест на root endpoint
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "OAuth2 Proxy")

	// Тест на health endpoint
	req = httptest.NewRequest(http.MethodGet, "/health", nil)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "healthy")

	// Тест на oauth/initiate endpoint
	req = httptest.NewRequest(http.MethodGet, "/oauth/initiate", nil)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusFound, rec.Code)
	assert.Contains(t, rec.Header().Get("Location"), "oauth")
}

func TestMiddlewareWithExclusions(t *testing.T) {
	cfg := &middleware.JWTConfig{
		SecretKey: "test-secret-key",
		Issuer:    "test-issuer",
		Audience:  "test-audience",
	}

	exclusions := []string{"/health", "/oauth/initiate"}

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mw := middlewareWithExclusions(h, cfg, exclusions)

	// Тест для excluded endpoint
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Тест для non-excluded endpoint без токена
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	rec = httptest.NewRecorder()
	mw.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
