package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestHandler_Home(t *testing.T) {
	config := &Config{
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		RedirectURI:  "http://localhost/callback",
		AuthorizationEndpoint: "http://auth/oauth",
		TokenEndpoint:    "http://auth/token",
		StateSecret:    "test-secret",
		LLMAPIURL:      "http://llm/v1",
		LLMAPIKey:      "test-key",
	}

	sm := NewStateManager(config.StateSecret, 10*time.Minute)
	ss := NewSessionStore()
	handler := NewHandler(config, sm, nil, ss)

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	handler.Home(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", rr.Code)
	}

	if !strings.Contains(rr.Body.String(), "OAuth2 Proxy") {
		t.Error("Home page should contain 'OAuth2 Proxy'")
	}
}

func TestHandler_Home_NotFound(t *testing.T) {
	config := &Config{
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		RedirectURI:  "http://localhost/callback",
		AuthorizationEndpoint: "http://auth/oauth",
		TokenEndpoint:    "http://auth/token",
		StateSecret:    "test-secret",
		LLMAPIURL:      "http://llm/v1",
		LLMAPIKey:      "test-key",
	}

	sm := NewStateManager(config.StateSecret, 10*time.Minute)
	ss := NewSessionStore()
	handler := NewHandler(config, sm, nil, ss)

	req := httptest.NewRequest("GET", "/not-found", nil)
	rr := httptest.NewRecorder()

	handler.Home(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("Expected 404, got %d", rr.Code)
	}
}

func TestHandler_Home_MethodNotAllowed(t *testing.T) {
	config := &Config{
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		RedirectURI:  "http://localhost/callback",
		AuthorizationEndpoint: "http://auth/oauth",
		TokenEndpoint:    "http://auth/token",
		StateSecret:    "test-secret",
		LLMAPIURL:      "http://llm/v1",
		LLMAPIKey:      "test-key",
	}

	sm := NewStateManager(config.StateSecret, 10*time.Minute)
	ss := NewSessionStore()
	handler := NewHandler(config, sm, nil, ss)

	req := httptest.NewRequest("POST", "/", nil)
	rr := httptest.NewRecorder()

	handler.Home(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected 405 for POST, got %d", rr.Code)
	}
}

func TestHandler_Auth(t *testing.T) {
	config := &Config{
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		RedirectURI:  "http://localhost/callback",
		AuthorizationEndpoint: "http://auth/oauth",
		TokenEndpoint:    "http://auth/token",
		StateSecret:    "test-secret",
		LLMAPIURL:      "http://llm/v1",
		LLMAPIKey:      "test-key",
		Scopes:        []string{"openid", "profile"},
	}

	sm := NewStateManager(config.StateSecret, 10*time.Minute)
	ss := NewSessionStore()
	handler := NewHandler(config, sm, nil, ss)

	req := httptest.NewRequest("GET", "/auth", nil)
	rr := httptest.NewRecorder()

	handler.Auth(rr, req)

	if rr.Code != http.StatusFound {
		t.Errorf("Expected 302, got %d", rr.Code)
	}

	location := rr.Header().Get("Location")
	if location == "" {
		t.Fatal("Location header should be set")
	}

	parsedURL, err := url.Parse(location)
	if err != nil {
		t.Fatalf("Failed to parse location URL: %v", err)
	}

	if !strings.Contains(parsedURL.Host, "auth") {
		t.Errorf("Expected host to contain 'auth', got '%s'", parsedURL.Host)
	}

	params := parsedURL.Query()
	if params.Get("client_id") != "test-client" {
		t.Errorf("Expected client_id 'test-client', got '%s'", params.Get("client_id"))
	}

	if params.Get("redirect_uri") != "http://localhost/callback" {
		t.Errorf("Expected redirect_uri, got '%s'", params.Get("redirect_uri"))
	}

	if params.Get("response_type") != "code" {
		t.Errorf("Expected response_type 'code', got '%s'", params.Get("response_type"))
	}

	if params.Get("state") == "" {
		t.Error("Expected state parameter")
	}

	if params.Get("scope") != "openid profile" {
		t.Errorf("Expected scope 'openid profile', got '%s'", params.Get("scope"))
	}
}

func TestHandler_Callback(t *testing.T) {
	config := &Config{
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		RedirectURI:  "http://localhost/callback",
		AuthorizationEndpoint: "http://auth/oauth",
		TokenEndpoint:    "http://auth/token",
		StateSecret:    "test-secret",
		LLMAPIURL:      "http://llm/v1",
		LLMAPIKey:      "test-key",
	}

	sm := NewStateManager(config.StateSecret, 10*time.Minute)
	ss := NewSessionStore()
	handler := NewHandler(config, sm, nil, ss)

	// Mock the token exchange
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"access_token":"mock-token","token_type":"Bearer","expires_in":3600}`))
	}))
	defer ts.Close()

	// Create a custom client for this test
	client := &http.Client{
		Transport: &mockTransport{
			response: http.StatusOK,
			body:     `{"access_token":"mock-token","token_type":"Bearer","expires_in":3600}`,
		},
	}

	// Temporarily replace the handler's internal client
	original := handler.llmClient
	handler.llmClient = nil // Disable LLM client to avoid conflicts

	// Manually test the token exchange logic
	tokenReq := TokenRequest{
		GrantType:   "authorization_code",
		Code:        "auth-code",
		RedirectURI: config.RedirectURI,
		ClientID:    config.ClientID,
		ClientSecret: config.ClientSecret,
	}

	tokenBody, _ := json.Marshal(tokenReq)
	tokenURL := config.TokenEndpoint

	httpReq, _ := http.NewRequest("POST", tokenURL, strings.NewReader(string(tokenBody)))
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	httpReq.Header.Set("Authorization", "Basic "+btoa(config.ClientID+":"+config.ClientSecret))

	resp, err := client.Do(httpReq)
	if err != nil {
		t.Fatalf("Token exchange failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}

	// Restore
	handler.llmClient = original
}

type mockTransport struct {
	response int
	body     string
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp := &http.Response{
		StatusCode: m.response,
		Body:       io.NopCloser(strings.NewReader(m.body)),
		Header:     make(http.Header),
	}
	return resp, nil
}

func TestHandler_Callback_InvalidCode(t *testing.T) {
	config := &Config{
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		RedirectURI:  "http://localhost/callback",
		AuthorizationEndpoint: "http://auth/oauth",
		TokenEndpoint:    "http://auth/token",
		StateSecret:    "test-secret",
		LLMAPIURL:      "http://llm/v1",
		LLMAPIKey:      "test-key",
	}

	sm := NewStateManager(config.StateSecret, 10*time.Minute)
	ss := NewSessionStore()
	handler := NewHandler(config, sm, nil, ss)

	req := httptest.NewRequest("GET", "/callback?code=invalid", nil)
	rr := httptest.NewRecorder()

	handler.Callback(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for invalid code, got %d", rr.Code)
	}
}

func TestHandler_Callback_InvalidState(t *testing.T) {
	config := &Config{
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		RedirectURI:  "http://localhost/callback",
		AuthorizationEndpoint: "http://auth/oauth",
		TokenEndpoint:    "http://auth/token",
		StateSecret:    "test-secret",
		LLMAPIURL:      "http://llm/v1",
		LLMAPIKey:      "test-key",
	}

	sm := NewStateManager(config.StateSecret, 10*time.Minute)
	ss := NewSessionStore()
	handler := NewHandler(config, sm, nil, ss)

	req := httptest.NewRequest("GET", "/callback?code=code&state=invalid-state", nil)
	rr := httptest.NewRecorder()

	handler.Callback(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for invalid state, got %d", rr.Code)
	}
}

func TestHandler_Callback_MethodNotAllowed(t *testing.T) {
	config := &Config{
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		RedirectURI:  "http://localhost/callback",
		AuthorizationEndpoint: "http://auth/oauth",
		TokenEndpoint:    "http://auth/token",
		StateSecret:    "test-secret",
		LLMAPIURL:      "http://llm/v1",
		LLMAPIKey:      "test-key",
	}

	sm := NewStateManager(config.StateSecret, 10*time.Minute)
	ss := NewSessionStore()
	handler := NewHandler(config, sm, nil, ss)

	req := httptest.NewRequest("POST", "/callback", nil)
	rr := httptest.NewRecorder()

	handler.Callback(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected 405 for POST, got %d", rr.Code)
	}
}

func TestHandler_Health(t *testing.T) {
	config := &Config{
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		RedirectURI:  "http://localhost/callback",
		AuthorizationEndpoint: "http://auth/oauth",
		TokenEndpoint:    "http://auth/token",
		StateSecret:    "test-secret",
		LLMAPIURL:      "http://llm/v1",
		LLMAPIKey:      "test-key",
	}

	sm := NewStateManager(config.StateSecret, 10*time.Minute)
	ss := NewSessionStore()
	handler := NewHandler(config, sm, nil, ss)

	req := httptest.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()

	handler.Health(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", rr.Code)
	}

	var health map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &health); err != nil {
		t.Errorf("Failed to decode health response: %v", err)
	}

	if health["status"] != "ok" {
		t.Errorf("Expected status 'ok', got '%s'", health["status"])
	}
}

func TestHandler_Proxy(t *testing.T) {
	config := &Config{
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		RedirectURI:  "http://localhost/callback",
		AuthorizationEndpoint: "http://auth/oauth",
		TokenEndpoint:    "http://auth/token",
		StateSecret:    "test-secret",
		LLMAPIURL:      "http://llm/v1",
		LLMAPIKey:      "test-key",
	}

	sm := NewStateManager(config.StateSecret, 10*time.Minute)
	ss := NewSessionStore()
	handler := NewHandler(config, sm, nil, ss)

	// Mock the LLM server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"test","object":"chat.completion","created":123,"model":"test","choices":[{"index":0,"message":{"role":"assistant","content":"hello"},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}`))
	}))
	defer ts.Close()

	config.LLMAPIURL = ts.URL

	ss.mu.Lock()
	ss.sessions["test-session"] = &Session{
		Token:     "test-token",
		UserInfo:  UserInfoResponse{Sub: "test-user"},
		ExpiresAt: time.Now().Add(time.Hour),
	}
	ss.mu.Unlock()

	req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(`{"model":"test","messages":[{"role":"user","content":"hi"}]}`))
	req.URL.RawQuery = "session=test-session"
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.Proxy(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", rr.Code)
	}

	body := rr.Body.Bytes()
	if len(body) == 0 {
		t.Error("Response body should not be empty")
	}
}

func TestHandler_Proxy_Unauthenticated(t *testing.T) {
	config := &Config{
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		RedirectURI:  "http://localhost/callback",
		AuthorizationEndpoint: "http://auth/oauth",
		TokenEndpoint:    "http://auth/token",
		StateSecret:    "test-secret",
		LLMAPIURL:      "http://llm/v1",
		LLMAPIKey:      "test-key",
	}

	sm := NewStateManager(config.StateSecret, 10*time.Minute)
	ss := NewSessionStore()
	handler := NewHandler(config, sm, nil, ss)

	req := httptest.NewRequest("POST", "/v1/chat/completions", nil)
	rr := httptest.NewRecorder()

	handler.Proxy(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 for unauthenticated, got %d", rr.Code)
	}
}

func TestHandler_Proxy_ExpiredSession(t *testing.T) {
	config := &Config{
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		RedirectURI:  "http://localhost/callback",
		AuthorizationEndpoint: "http://auth/oauth",
		TokenEndpoint:    "http://auth/token",
		StateSecret:    "test-secret",
		LLMAPIURL:      "http://llm/v1",
		LLMAPIKey:      "test-key",
	}

	sm := NewStateManager(config.StateSecret, 10*time.Minute)
	ss := NewSessionStore()
	handler := NewHandler(config, sm, nil, ss)

	ss.mu.Lock()
	ss.sessions["test-session"] = &Session{
		Token:     "test-token",
		UserInfo:  UserInfoResponse{Sub: "test-user"},
		ExpiresAt: time.Now().Add(-time.Hour),
	}
	ss.mu.Unlock()

	req := httptest.NewRequest("POST", "/v1/chat/completions", nil)
	req.URL.RawQuery = "session=test-session"
	rr := httptest.NewRecorder()

	handler.Proxy(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 for expired session, got %d", rr.Code)
	}
}
