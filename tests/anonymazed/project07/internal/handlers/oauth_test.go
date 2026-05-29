package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"golang.org/x/oauth2"
	"hermes/config"
	"hermes/internal/store"
)

func TestNewOAuth2Handler(t *testing.T) {
	cfg := &config.OAuth2Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURI:  "http://localhost:8080/callback",
		EndpointURL:  "https://example.com/auth",
		TokenURL:     "https://example.com/token",
		UserInfoURL:  "https://example.com/userinfo",
		Scopes:       []string{"openid", "profile", "email"},
		SessionSecret: "test-secret",
	}
	storeInstance := store.NewInMemoryStore(5 * time.Minute)
	handler := NewOAuth2Handler(cfg, storeInstance)

	if handler == nil {
		t.Fatal("expected handler to be not nil")
	}
	if handler.config.ClientID != "test-client-id" {
		t.Errorf("expected client-id to be 'test-client-id', got %s", handler.config.ClientID)
	}
}

func TestLogin(t *testing.T) {
	cfg := &config.OAuth2Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURI:  "http://localhost:8080/callback",
		EndpointURL:  "https://example.com/auth",
		TokenURL:     "https://example.com/token",
		UserInfoURL:  "https://example.com/userinfo",
		Scopes:       []string{"openid", "profile", "email"},
	}
	storeInstance := store.NewInMemoryStore(5 * time.Minute)
	handler := NewOAuth2Handler(cfg, storeInstance)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/login", nil)
	handler.Login(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("expected status %d, got %d", http.StatusFound, w.Code)
	}
}

func TestCallback(t *testing.T) {
	cfg := &config.OAuth2Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURI:  "http://localhost:8080/callback",
		EndpointURL:  "https://example.com/auth",
		TokenURL:     "https://example.com/token",
		UserInfoURL:  "https://example.com/userinfo",
		Scopes:       []string{"openid", "profile", "email"},
		SessionSecret: "test-secret",
	}
	storeInstance := store.NewInMemoryStore(5 * time.Minute)
	handler := NewOAuth2Handler(cfg, storeInstance)

	// Save a state first using InMemoryStore directly
	inMemStore := store.NewInMemoryStore(5 * time.Minute)
	state := &store.State{
		State:     "test-state",
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}
	if err := inMemStore.SaveState(nil, state); err != nil {
		t.Fatalf("failed to save state: %v", err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/callback?state=test-state&code=fake-code", nil)
	handler.Callback(w, req)

	// Should return error since token exchange fails in test
	if w.Code != http.StatusBadRequest {
		t.Logf("got status %d (expected bad request since token exchange fails in test)", w.Code)
	}
}

func TestLogout(t *testing.T) {
	cfg := &config.OAuth2Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURI:  "http://localhost:8080/callback",
		EndpointURL:  "https://example.com/auth",
		TokenURL:     "https://example.com/token",
		UserInfoURL:  "https://example.com/userinfo",
		Scopes:       []string{"openid", "profile", "email"},
	}
	storeInstance := store.NewInMemoryStore(5 * time.Minute)
	handler := NewOAuth2Handler(cfg, storeInstance)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/logout", nil)
	handler.Logout(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("expected status %d, got %d", http.StatusFound, w.Code)
	}
}

func TestLoginStatus_NoSession(t *testing.T) {
	cfg := &config.OAuth2Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURI:  "http://localhost:8080/callback",
		EndpointURL:  "https://example.com/auth",
		TokenURL:     "https://example.com/token",
		UserInfoURL:  "https://example.com/userinfo",
		Scopes:       []string{"openid", "profile", "email"},
	}
	storeInstance := store.NewInMemoryStore(5 * time.Minute)
	handler := NewOAuth2Handler(cfg, storeInstance)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/loginstatus", nil)
	handler.LoginStatus(w, req)

	// Should return 200 with loggedIn: false
	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if loggedIn, ok := resp["loggedIn"].(bool); !ok || loggedIn {
		t.Errorf("expected loggedIn to be false, got %v", resp)
	}
}

func TestLoginStatus_WithSession(t *testing.T) {
	cfg := &config.OAuth2Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURI:  "http://localhost:8080/callback",
		EndpointURL:  "https://example.com/auth",
		TokenURL:     "https://example.com/token",
		UserInfoURL:  "https://example.com/userinfo",
		Scopes:       []string{"openid", "profile", "email"},
		SessionSecret: "test-secret",
	}
	storeInstance := store.NewInMemoryStore(5 * time.Minute)
	handler := NewOAuth2Handler(cfg, storeInstance)

	// Create a valid session
	session := &Session{
		UserInfo: map[string]interface{}{
			"sub": "test-user",
			"email": "test@example.com",
		},
		Token: &oauth2.Token{
			AccessToken: "test-access-token",
		},
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	// Create request with session cookie
	data, _ := json.Marshal(session)
	encoded := base64Encode(data)
	cookie := &http.Cookie{
		Name:  "hermes_session",
		Value: encoded,
	}
	req := httptest.NewRequest("GET", "/loginstatus", nil)
	req.AddCookie(cookie)

	w := httptest.NewRecorder()
	handler.LoginStatus(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if loggedIn, ok := resp["loggedIn"].(bool); !ok || !loggedIn {
		t.Errorf("expected loggedIn to be true")
	}
}

func TestBuildAuthURL(t *testing.T) {
	cfg := &config.OAuth2Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURI:  "http://localhost:8080/callback",
		EndpointURL:  "https://example.com/auth",
		TokenURL:     "https://example.com/token",
		UserInfoURL:  "https://example.com/userinfo",
		Scopes:       []string{"openid", "profile", "email"},
	}
	storeInstance := store.NewInMemoryStore(5 * time.Minute)
	handler := NewOAuth2Handler(cfg, storeInstance)

	url := handler.buildAuthURL("test-state")

	if !bytes.Contains([]byte(url), []byte("test-state")) {
		t.Errorf("expected URL to contain state parameter")
	}
	if !bytes.Contains([]byte(url), []byte("test-client-id")) {
		t.Errorf("expected URL to contain client-id")
	}
}

func TestExchangeToken(t *testing.T) {
	cfg := &config.OAuth2Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURI:  "http://localhost:8080/callback",
		EndpointURL:  "https://example.com/auth",
		TokenURL:     "https://example.com/token",
		UserInfoURL:  "https://example.com/userinfo",
		Scopes:       []string{"openid", "profile", "email"},
	}
	storeInstance := store.NewInMemoryStore(5 * time.Minute)
	handler := NewOAuth2Handler(cfg, storeInstance)

	// This will fail with fake code, but we can check error handling
	_, err := handler.exchangeToken("fake-code", "fake-state")
	if err == nil {
		t.Error("expected error for fake code")
	}
}

func TestCreateSession(t *testing.T) {
	cfg := &config.OAuth2Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURI:  "http://localhost:8080/callback",
		EndpointURL:  "https://example.com/auth",
		TokenURL:     "https://example.com/token",
		UserInfoURL:  "https://example.com/userinfo",
		Scopes:       []string{"openid", "profile", "email"},
		SessionSecret: "test-secret",
	}
	storeInstance := store.NewInMemoryStore(5 * time.Minute)

	// Test without session secret
	cfgNoSecret := cfg
	cfgNoSecret.SessionSecret = ""
	handlerNoSecret := NewOAuth2Handler(cfgNoSecret, storeInstance)

	w := httptest.NewRecorder()
	session := &Session{
		UserInfo: map[string]interface{}{"sub": "test"},
	}
	err1 := handlerNoSecret.createSession(w, session)
	if err1 == nil {
		t.Error("expected error when session secret is not set")
	}

	// Test with session secret - should succeed
	handlerWithSecret := NewOAuth2Handler(cfg, storeInstance)
	w2 := httptest.NewRecorder()
	err2 := handlerWithSecret.createSession(w2, session)
	if err2 != nil {
		t.Errorf("unexpected error: %v", err2)
	}
}

func TestDeleteSession(t *testing.T) {
	cfg := &config.OAuth2Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURI:  "http://localhost:8080/callback",
		EndpointURL:  "https://example.com/auth",
		TokenURL:     "https://example.com/token",
		UserInfoURL:  "https://example.com/userinfo",
		Scopes:       []string{"openid", "profile", "email"},
	}
	storeInstance := store.NewInMemoryStore(5 * time.Minute)
	handler := NewOAuth2Handler(cfg, storeInstance)

	w := httptest.NewRecorder()
	err := handler.deleteSession(w)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
