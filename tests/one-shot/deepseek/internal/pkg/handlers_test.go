package pkg

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestStartAuthorization(t *testing.T) {
	sm := NewStateMachine()
	config := Config{
		ProviderURL: "http://localhost:8081",
		EnablePKCE:  true,
	}
	handler := NewHandler(sm, config)

	req := httptest.NewRequest("GET", "/oauth2/auth?state=test123&redirect=http://example.com/callback", nil)
	rr := httptest.NewRecorder()

	handler.StartAuthorization(rr, req)

	if rr.Code != http.StatusFound {
		t.Errorf("expected status %d, got %d", http.StatusFound, rr.Code)
	}
}

func TestCallback(t *testing.T) {
	sm := NewStateMachine()
	config := Config{
		RedirectURL: "http://example.com/callback",
		ClientSecret: "secret123",
	}
	handler := NewHandler(sm, config)

	// Create state
	state, _ := NewState("auth123", "verifier123", "challenge123", "")
	sm.StoreState(state)

	req := httptest.NewRequest("GET", "/oauth2/callback?state=test123&auth_code=auth123", nil)
	rr := httptest.NewRecorder()

	handler.Callback(rr, req)

	// State should be deleted
	if _, exists := sm.GetState("test123"); exists {
		t.Error("state should be deleted after callback")
	}
}

func TestTokenEndpoint(t *testing.T) {
	sm := NewStateMachine()
	config := Config{
		ProviderURL: "http://localhost:8081",
		ClientSecret: "secret123",
	}
	handler := NewHandler(sm, config)

	req := httptest.NewRequest("POST", "/oauth2/token", nil)
	req.ParseForm()
	req.Form.Set("client_secret", "secret123")
	req.Form.Set("grant_type", "authorization_code")
	req.Form.Set("code", "auth123")

	rr := httptest.NewRecorder()

	handler.TokenEndpoint(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	// Check response
	var response map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Errorf("failed to parse response: %v", err)
	}

	if response["access_token"] == "" {
		t.Error("access_token should be in response")
	}
}

func TestTokenEndpointInvalidSecret(t *testing.T) {
	sm := NewStateMachine()
	config := Config{
		ProviderURL: "http://localhost:8081",
		ClientSecret: "secret123",
	}
	handler := NewHandler(sm, config)

	req := httptest.NewRequest("POST", "/oauth2/token", nil)
	req.ParseForm()
	req.Form.Set("client_secret", "wrongsecret")
	req.Form.Set("grant_type", "authorization_code")
	req.Form.Set("code", "auth123")

	rr := httptest.NewRecorder()

	handler.TokenEndpoint(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}

func TestRefreshEndpoint(t *testing.T) {
	sm := NewStateMachine()
	config := Config{
		ProviderURL: "http://localhost:8081",
		ClientSecret: "secret123",
	}
	handler := NewHandler(sm, config)

	req := httptest.NewRequest("POST", "/oauth2/refresh", nil)
	req.ParseForm()
	req.Form.Set("client_secret", "secret123")
	req.Form.Set("grant_type", "refresh_token")
	req.Form.Set("refresh_token", "refresh123")

	rr := httptest.NewRecorder()

	handler.RefreshEndpoint(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	// Check response
	var response map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Errorf("failed to parse response: %v", err)
	}

	if response["access_token"] == "" {
		t.Error("access_token should be in response")
	}
}

func TestInfoEndpoint(t *testing.T) {
	sm := NewStateMachine()
	config := Config{
		ProviderURL: "http://localhost:8081",
		ClientID: "test-client",
		Scopes: []string{"openid", "profile"},
	}
	handler := NewHandler(sm, config)

	req := httptest.NewRequest("GET", "/oauth2/.well-known/oauth-authorization-server", nil)
	rr := httptest.NewRecorder()

	handler.InfoEndpoint(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	// Check response
	var response map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Errorf("failed to parse response: %v", err)
	}

	if response["authorization_endpoint"] == nil {
		t.Error("authorization_endpoint should be in response")
	}

	if len(response["scopes"].([]interface{})) != 2 {
		t.Error("should have 2 scopes")
	}
}

func TestGenerateStateToken(t *testing.T) {
	token, err := GenerateStateToken()
	if err != nil {
		t.Errorf("failed to generate state token: %v", err)
	}

	if len(token) == 0 {
		t.Error("state token should not be empty")
	}

	// Verify it's base64 encoded
	if len(token) != len(token)*4/3 { // rough check
		t.Error("state token length seems incorrect")
	}
}

func TestNewState(t *testing.T) {
	state, err := NewState("auth123", "verifier123", "challenge123", "http://example.com/callback")
	if err != nil {
		t.Errorf("failed to create state: %v", err)
	}

	if state.AuthCode != "auth123" {
		t.Errorf("expected auth code 'auth123', got '%s'", state.AuthCode)
	}

	if state.CodeVerifier != "verifier123" {
		t.Errorf("expected code verifier 'verifier123', got '%s'", state.CodeVerifier)
	}

	if state.CodeChallenge != "challenge123" {
		t.Errorf("expected code challenge 'challenge123', got '%s'", state.CodeChallenge)
	}

	if state.RedirectURL != "http://example.com/callback" {
		t.Errorf("expected redirect URL 'http://example.com/callback', got '%s'", state.RedirectURL)
	}
}

func TestGetStateExpired(t *testing.T) {
	sm := NewStateMachine()
	state, _ := NewState("auth123", "verifier123", "challenge123", "")
	state.Expiry = time.Now().Add(-time.Hour) // Expired

	sm.StoreState(state)

	if _, exists := sm.GetState(state.State); exists {
		t.Error("expired state should not be found")
	}
}

func TestDeleteState(t *testing.T) {
	sm := NewStateMachine()
	state, _ := NewState("auth123", "verifier123", "challenge123", "")
	sm.StoreState(state)

	sm.DeleteState(state.State)

	if _, exists := sm.GetState(state.State); exists {
		t.Error("deleted state should not be found")
	}
}

func TestGCExpiredStates(t *testing.T) {
	sm := NewStateMachine()
	validState, _ := NewState("auth123", "verifier123", "challenge123", "")
	validState.Expiry = time.Now().Add(time.Hour)

	expiredState, _ := NewState("auth456", "verifier456", "challenge456", "")
	expiredState.Expiry = time.Now().Add(-time.Hour)

	sm.StoreState(validState)
	sm.StoreState(expiredState)

	sm.GCExpiredStates()

	if _, exists := sm.GetState(validState.State); !exists {
		t.Error("valid state should still exist")
	}

	if _, exists := sm.GetState(expiredState.State); exists {
		t.Error("expired state should be removed")
	}
}

func TestVerifyCodeChallenge(t *testing.T) {
	// Test with non-empty challenge
	if !VerifyCodeChallenge("verifier", "challenge") {
		t.Error("should return true for non-empty challenge")
	}

	// Test with empty challenge
	if !VerifyCodeChallenge("verifier", "") {
		t.Error("should return true for empty challenge")
	}
}

func TestGenerateCodeVerifier(t *testing.T) {
	verifier := generateCodeVerifier()
	if len(verifier) < 40 {
		t.Errorf("code verifier too short: %d chars", len(verifier))
	}
}

func TestGenerateCodeChallenge(t *testing.T) {
	verifier := generateCodeVerifier()
	challenge := generateCodeChallenge(verifier)

	if challenge == "" {
		t.Error("code challenge should not be empty")
	}
}
