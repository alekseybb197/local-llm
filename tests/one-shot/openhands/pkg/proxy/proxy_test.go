package proxy

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/openhands/oauth2-proxy/pkg/config"
)

func TestInMemorySessionStore(t *testing.T) {
	store := NewInMemorySessionStore()

	session := &Session{
		UserID:   "test-user-123",
		Username: "Test User",
		Email:    "test@example.com",
		Token:    "test-token",
		Expires:  time.Now().Add(1 * time.Hour),
	}

	err := store.Set("test-user-123", session)
	if err != nil {
		t.Fatalf("Failed to set session: %v", err)
	}

	retrieved, err := store.Get("test-user-123")
	if err != nil {
		t.Fatalf("Failed to get session: %v", err)
	}

	if retrieved.UserID != session.UserID {
		t.Errorf("Expected UserID %s, got %s", session.UserID, retrieved.UserID)
	}

	err = store.Delete("test-user-123")
	if err != nil {
		t.Fatalf("Failed to delete session: %v", err)
	}

	_, err = store.Get("test-user-123")
	if err == nil {
		t.Error("Expected error after deletion, got nil")
	}
}

func TestProxy_New(t *testing.T) {
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	cfg.OAuth2Config.ClientID = "test-client"
	cfg.OAuth2Config.ClientSecret = "test-secret"
	cfg.OAuth2Config.RedirectURI = "http://localhost:8080/callback"
	cfg.OAuth2Config.AuthURL = "http://localhost:8080/auth"
	cfg.OAuth2Config.TokenURL = "http://localhost:8080/token"
	cfg.OAuth2Config.UserInfoURL = "http://localhost:8080/userinfo"
	cfg.ProxyConfig.LLMEndpoint = "http://localhost:8080/llm"

	p, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create proxy: %v", err)
	}

	if p == nil {
		t.Error("Expected proxy to be non-nil")
	}
}

func TestAuthHandler(t *testing.T) {
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	cfg.OAuth2Config.ClientID = "test-client"
	cfg.OAuth2Config.ClientSecret = "test-secret"
	cfg.OAuth2Config.RedirectURI = "http://localhost:8080/callback"
	cfg.OAuth2Config.AuthURL = "http://example.com/oauth/authorize"
	cfg.OAuth2Config.Scope = "openid email profile"

	p, _ := New(cfg)

	req := httptest.NewRequest("GET", "/oauth2/auth", nil)
	rr := httptest.NewRecorder()

	p.AuthHandler(rr, req)

	if rr.Code != http.StatusFound {
		t.Errorf("Expected status %d, got %d", http.StatusFound, rr.Code)
	}
}

func TestCallbackHandler(t *testing.T) {
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	cfg.OAuth2Config.ClientID = "test-client"
	cfg.OAuth2Config.ClientSecret = "test-secret"
	cfg.OAuth2Config.RedirectURI = "http://localhost:8080/callback"
	cfg.OAuth2Config.AuthURL = "http://example.com/oauth/authorize"
	cfg.OAuth2Config.TokenURL = "http://localhost:8080/token"
	cfg.OAuth2Config.UserInfoURL = "http://example.com/userinfo"

	p, _ := New(cfg)

	req := httptest.NewRequest("GET", "/oauth2/callback", nil)
	rr := httptest.NewRecorder()

	p.CallbackHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d for missing state, got %d", http.StatusBadRequest, rr.Code)
	}

	req = httptest.NewRequest("GET", "/oauth2/callback?state=invalid", nil)
	rr = httptest.NewRecorder()

	p.CallbackHandler(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("Expected status %d for invalid state, got %d", http.StatusForbidden, rr.Code)
	}
}

func TestProtectedHandler(t *testing.T) {
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	p, _ := New(cfg)

	req := httptest.NewRequest("GET", "/api/protected", nil)
	rr := httptest.NewRecorder()

	p.ProtectedHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d for missing auth header, got %d", http.StatusUnauthorized, rr.Code)
	}

	req = httptest.NewRequest("GET", "/api/protected", nil)
	req.Header.Set("Authorization", "Invalid format")
	rr = httptest.NewRecorder()

	p.ProtectedHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d for invalid auth header, got %d", http.StatusUnauthorized, rr.Code)
	}

	req = httptest.NewRequest("GET", "/api/protected", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rr = httptest.NewRecorder()

	p.ProtectedHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d for valid auth header, got %d", http.StatusOK, rr.Code)
	}
}
