package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/example/oauth2-proxy/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestInitiateAuth(t *testing.T) {
	cfg := &config.OAuthConfig{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		TokenURL:     "http://localhost:8080/oauth/token",
		RedirectURL:  "http://localhost:8080/oauth/callback",
	}

	h := NewOAuthHandler(cfg)

	req := httptest.NewRequest(http.MethodGet, "/oauth/initiate", nil)
	rec := httptest.NewRecorder()
	h.InitiateAuth(rec, req)

	assert.Equal(t, http.StatusFound, rec.Code)
	assert.Contains(t, rec.Header().Get("Location"), "oauth")
	assert.Contains(t, rec.Header().Get("Location"), "state")
}

func TestHandleCallback(t *testing.T) {
	cfg := &config.OAuthConfig{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		TokenURL:     "http://localhost:8080/oauth/token",
		RedirectURL:  "http://localhost:8080/oauth/callback",
	}

	h := NewOAuthHandler(cfg)

	// Тест с валидным кодом
	req := httptest.NewRequest(http.MethodGet, "/oauth/callback?code=test-code&state=test-state", nil)
	req.AddCookie(&http.Cookie{Name: "oauth_state", Value: "test-state"})
	rec := httptest.NewRecorder()
	h.HandleCallback(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "access_token")

	// Тест с невалидным state
	req = httptest.NewRequest(http.MethodGet, "/oauth/callback?code=test-code&state=invalid-state", nil)
	req.AddCookie(&http.Cookie{Name: "oauth_state", Value: "test-state"})
	rec = httptest.NewRecorder()
	h.HandleCallback(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
