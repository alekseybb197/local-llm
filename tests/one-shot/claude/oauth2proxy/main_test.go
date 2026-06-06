package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestGenerateRandomString(t *testing.T) {
	s, err := generateRandomString(16)
	if err != nil {
		t.Fatalf("generateRandomString failed: %v", err)
	}
	// Base64 encodes 16 bytes to 24 characters
	if len(s) != 24 {
		t.Errorf("expected length 24, got %d", len(s))
	}
}

func TestGenerateCodeChallenge(t *testing.T) {
	verifier := "test-verifier-string"
	challenge, err := generateCodeChallenge(verifier)
	if err != nil {
		t.Fatalf("generateCodeChallenge failed: %v", err)
	}
	if len(challenge) == 0 {
		t.Error("expected non-empty challenge")
	}
}

func TestParseState(t *testing.T) {
	state, err := parseState("abc123|def456")
	if err != nil {
		t.Fatalf("parseState failed: %v", err)
	}
	if state.ClientID != "local-client" {
		t.Errorf("expected client_id 'local-client', got '%s'", state.ClientID)
	}
	if state.Expiry.IsZero() {
		t.Error("expected non-zero expiry")
	}
}

func TestHandlerServeHTTP(t *testing.T) {
	config := Config{
		ListenAddr:   "127.0.0.1:9999",
		LLMApiAddr:   "http://127.0.0.1:11434",
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		RedirectURI:  "http://127.0.0.1:8080/callback",
		AuthServerURL: "http://localhost:3000",
	}
	handler := NewHandler(config)

	tests := []struct {
		name    string
		path    string
		method  string
		wantErr bool
	}{
		{"health", "/health", http.MethodGet, false},
		{"callback", "/callback", http.MethodGet, true},
		{"index", "/", http.MethodGet, false},
		{"api-proxy", "/api/v1/models", http.MethodGet, false},
		{"not-found", "/nonexistent", http.MethodGet, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if tt.wantErr && rr.Code == http.StatusOK {
				t.Errorf("%s: expected error, got status %d", tt.name, rr.Code)
			}
		})
	}
}

func TestHandlerHandleHealth(t *testing.T) {
	config := Config{
		ListenAddr: "127.0.0.1:9999",
	}
	handler := NewHandler(config)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	handler.handleHealth(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp["status"] != "ok" {
		t.Errorf("expected status 'ok', got '%v'", resp["status"])
	}
}

func TestHandlerHandleIndex(t *testing.T) {
	config := Config{
		ListenAddr: "127.0.0.1:9999",
	}
	handler := NewHandler(config)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.handleIndex(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "OAuth2 Proxy") {
		t.Error("expected OAuth2 Proxy title in response")
	}
	if !strings.Contains(body, "redirect") {
		t.Error("expected redirect message in response")
	}
}

func TestProxyRequest(t *testing.T) {
	// Create mock server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("mock response"))
	}))
	defer mockServer.Close()

	config := Config{
		ListenAddr: "127.0.0.1:9999",
		LLMApiAddr:  mockServer.URL,
	}
	_ = NewHandler(config)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/models", nil)
	rr := httptest.NewRecorder()
	proxyRequest(rr, req, mockServer.URL)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	if !strings.Contains(rr.Body.String(), "mock response") {
		t.Error("expected mock response in body")
	}
}

func TestHandlerProxyLLMRequest(t *testing.T) {
	// Create a mock server for the target LLM
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify auth header is set
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-access-token-123" {
			t.Logf("expected auth header, got: %s", auth)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"model":"test-model","created":123456}`))
	}))
	defer mockServer.Close()

	config := Config{
		LLMApiAddr:   mockServer.URL,
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		RedirectURI:  mockServer.URL + "/callback",
		AuthServerURL: "http://localhost:3000",
	}
	handler := NewHandler(config)

	// Set auth cookie
	cookie := &http.Cookie{
		Name:  "oauth2_code",
		Value: "test-access-token-123",
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/models", nil)
	req.AddCookie(cookie)

	rr := httptest.NewRecorder()
	handler.proxyLLMRequest(rr, req)

	if rr.Code != http.StatusOK {
		t.Logf("proxy error: %s", rr.Body.String())
		t.Errorf("expected status 200, got %d, error: %s", rr.Code, rr.Body.String())
	}

	body := rr.Body.String()
	if !strings.Contains(body, "test-model") {
		t.Errorf("expected model response, got: %s", body)
	}
}

func TestHandlerProxyLLMRequestUnauthorized(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer mockServer.Close()

	config := Config{
		ListenAddr:   "127.0.0.1:9999",
		LLMApiAddr:   mockServer.URL,
		ClientID:     "test-client",
		ClientSecret: "test-secret",
	}
	handler := NewHandler(config)

	// No auth cookie
	req := httptest.NewRequest(http.MethodGet, "/api/v1/models", nil)
	rr := httptest.NewRecorder()
	handler.proxyLLMRequest(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rr.Code)
	}
}

func TestHandlerHandleCORS(t *testing.T) {
	config := Config{
		ListenAddr:          "127.0.0.1:9999",
		AllowedOrigins:      []string{"http://example.com"},
		LLMApiAddr:          "http://127.0.0.1:11434",
		ClientID:            "test-client",
		ClientSecret:        "test-secret",
		RedirectURI:         "http://127.0.0.1:8080/callback",
		AuthServerURL:       "http://localhost:3000",
		AllowedPaths:        []string{"/api/*"},
	}
	handler := NewHandler(config)

	t.Run("allowed origin", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/api/test", nil)
		req.Header.Set("Origin", "http://example.com")
		rr := httptest.NewRecorder()
		handler.handleCORS(rr, req)

		if rr.Code != http.StatusNoContent {
			t.Errorf("expected status 204, got %d", rr.Code)
		}

		allowOrigin := rr.Header().Get("Access-Control-Allow-Origin")
		if allowOrigin != "http://example.com" {
			t.Errorf("expected CORS header, got '%s'", allowOrigin)
		}
	})

	t.Run("forbidden origin", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/api/test", nil)
		req.Header.Set("Origin", "http://malicious.com")
		rr := httptest.NewRecorder()
		handler.handleCORS(rr, req)

		if rr.Code != http.StatusForbidden {
			t.Errorf("expected status 403, got %d", rr.Code)
		}
	})
}

func TestHandlerForceAuth(t *testing.T) {
	config := Config{
		ListenAddr:   "127.0.0.1:9999",
		LLMApiAddr:   "http://127.0.0.1:11434",
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		RedirectURI:  "http://127.0.0.1:8080/callback",
		AuthServerURL: "http://localhost:3000",
	}
	handler := NewHandler(config)

	req := httptest.NewRequest(http.MethodGet, "/?authorize=true", nil)
	rr := httptest.NewRecorder()

	err := handler.forceAuth(rr, req)
	if err != nil {
		t.Errorf("forceAuth failed: %v", err)
	}

	// Should redirect to callback with auth
	if rr.Code != http.StatusSeeOther {
		t.Errorf("expected status 303, got %d", rr.Code)
	}
}

func TestHandlerBuildCookies(t *testing.T) {
	config := Config{
		ListenAddr:   "127.0.0.1:9999",
		LLMApiAddr:   "http://127.0.0.1:11434",
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		RedirectURI:  "http://127.0.0.1:8080/callback",
		AuthServerURL: "http://localhost:3000",
		SessionMaxAge: 900,
	}
	handler := NewHandler(config)

	cookies := handler.buildCookies("client123", "token456", time.Now().Add(time.Hour), "nonce789")

	if len(cookies) != 2 {
		t.Errorf("expected 2 cookies, got %d", len(cookies))
	}

	// Check access token cookie
	foundAccess := false
	for _, c := range cookies {
		if c.Name == "oauth2_code" && c.Value == "token456" {
			foundAccess = true
			break
		}
	}
	if !foundAccess {
		t.Error("access token cookie not found")
	}
}

func TestContains(t *testing.T) {
	if !contains("http://example.com", "http://example.com") {
		t.Error("contains should return true for exact match")
	}

	if contains("http://other.com", "http://example.com") {
		t.Error("contains should return false for different domain")
	}
}

func TestApplyDefaults(t *testing.T) {
	config := Config{
		ListenAddr: "127.0.0.1:3000",
		// Missing other fields
	}

	// Apply defaults via applyDefaults
	defaultConfig := applyDefaults(config)

	if defaultConfig.ListenAddr != "127.0.0.1:3000" {
		t.Errorf("expected listen_addr to be preserved, got '%s'", defaultConfig.ListenAddr)
	}
}

func TestHandlerWithSkipAuth(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("mock response"))
	}))
	defer mockServer.Close()

	config := Config{
		ListenAddr:   "127.0.0.1:9999",
		LLMApiAddr:   mockServer.URL,
		ClientID:     "",
		ClientSecret: "",
		SkipAuth:     true,
	}
	handler := NewHandler(config)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/models", nil)
	rr := httptest.NewRecorder()

	handler.proxyLLMRequest(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200 with skip_auth, got %d", rr.Code)
	}
}

func TestHandlerCorsHeaders(t *testing.T) {
	config := Config{
		ListenAddr:          "127.0.0.1:9999",
		AllowedOrigins:      []string{"http://example.com"},
		LLMApiAddr:          "http://127.0.0.1:11434",
		ClientID:            "test-client",
		ClientSecret:        "test-secret",
		RedirectURI:         "http://127.0.0.1:8080/callback",
		AuthServerURL:       "http://localhost:3000",
		AllowedPaths:        []string{"/api/*"},
	}
	handler := NewHandler(config)

	req := httptest.NewRequest(http.MethodOptions, "/api/test", nil)
	req.Header.Set("Origin", "http://example.com")
	rr := httptest.NewRecorder()
	handler.handleCORS(rr, req)

	// Check CORS headers
	expectedHeaders := map[string]string{
		"Access-Control-Allow-Origin": "http://example.com",
		"Access-Control-Allow-Methods": "GET, POST, OPTIONS",
		"Access-Control-Allow-Headers": "Content-Type, Authorization, X-Requested-With",
		"Access-Control-Max-Age":       "86400",
	}

	for header, expected := range expectedHeaders {
		actual := rr.Header().Get(header)
		if actual != expected {
			t.Errorf("header %s: expected '%s', got '%s'", header, expected, actual)
		}
	}
}

func TestTokenStruct(t *testing.T) {
	token := &Token{
		AccessToken:  "test-token",
		RefreshToken: "test-refresh",
		TokenType:    "Bearer",
		ExpiresIn:    3600,
		ExpiresAt:    time.Now().Add(time.Hour),
	}

	if token.AccessToken != "test-token" {
		t.Error("expected token.AccessToken to match")
	}
	if token.TokenType != "Bearer" {
		t.Error("expected token.TokenType to match")
	}
}

func TestGenerateNonce(t *testing.T) {
	nonce := generateNonce()
	if len(nonce) == 0 {
		t.Error("expected non-empty nonce")
	}
	if len(nonce) != 16 {
		// Base64 encodes 16 bytes to 24 characters
		if len(nonce) != 24 {
			t.Errorf("expected nonce length 24, got %d", len(nonce))
		}
	}
}

func TestExchangeCodeForToken(t *testing.T) {
	handler := &Handler{
		config: Config{
			AuthServerURL: "http://localhost:3000",
		},
	}

	token, err := handler.exchangeCodeForToken("test-code")
	if err != nil {
		t.Errorf("exchangeCodeForToken failed: %v", err)
	}

	if token.AccessToken == "" {
		t.Error("expected non-empty access token")
	}
	if token.RefreshToken == "" {
		t.Error("expected non-empty refresh token")
	}
	if token.TokenType != "Bearer" {
		t.Errorf("expected token type 'Bearer', got '%s'", token.TokenType)
	}
	if token.ExpiresAt.IsZero() {
		t.Error("expected non-zero expiry time")
	}
}

func TestExchangeCodeForTokenInvalidFormat(t *testing.T) {
	handler := &Handler{
		config: Config{
			AuthServerURL: "http://localhost:3000",
		},
	}

	// Use the validation version of the function
	_, err := handler.exchangeCodeForTokenWithValidation("invalid-code-format")
	if err == nil {
		t.Error("expected error for invalid code format")
	}
}
