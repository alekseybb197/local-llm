package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/example/oauth2-proxy/internal/config"
	"github.com/example/oauth2-proxy/internal/middleware"
	"github.com/stretchr/testify/assert"
)

func TestBuildProxyURL(t *testing.T) {
	cfg := &config.LLMConfig{
		APIURL: "http://localhost:11434/api",
	}

	h := &ProxyHandler{cfg: cfg}

	// Тест с OpenAI-compatible эндпоинтом
	rURL, _ := url.Parse("/v1/chat/completions")
	proxyURL, err := h.buildProxyURL(rURL)
	assert.NoError(t, err)
	assert.True(t, strings.HasPrefix(proxyURL.Path, "/api/chat/completions"))

	// Тест со стандартным эндпоинтом
	rURL, _ = url.Parse("/api/completions")
	proxyURL, err = h.buildProxyURL(rURL)
	assert.NoError(t, err)
	assert.Equal(t, "/api/completions", proxyURL.Path)
}

func TestLLMHandler(t *testing.T) {
	cfg := &config.LLMConfig{
		APIURL: "http://localhost:11434/api",
	}

	// Создаем валидный токен для тестов
	claims := &middleware.Claims{
		UserID:     "test-user",
		AccessToken: "test-token",
		Exp:        9999999999,
		Iat:        1234567890,
		Iss:        "test-issuer",
		Aud:        "test-audience",
	}

	ctx := context.WithValue(context.Background(), middleware.CtxUserClaims, claims)

	h := &ProxyHandler{cfg: cfg}
	handler := h.LLMHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"result": "success"}`))
	}))

	// Тест с валидным токеном
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model": "test"}`))
	req = req.WithContext(ctx)
	req.Header.Set("Authorization", "Bearer test-token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Тест без токена
	req = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model": "test"}`))
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestHealthCheck(t *testing.T) {
	h := &ProxyHandler{}

	handler := http.HandlerFunc(h.HealthCheck)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, `{"status":"healthy","time":"2024-01-01T00:00:00Z"}`, rec.Body.String())
}
