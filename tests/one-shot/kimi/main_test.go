package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestGenerateState(t *testing.T) {
	state, err := generateState()
	if err != nil {
		t.Fatalf("generateState() error = %v", err)
	}
	if state == "" {
		t.Error("generateState() returned empty state")
	}
	if len(state) < 43 {
		t.Errorf("state too short: %d chars, expected >= 43", len(state))
	}
}

func TestValidateRedirectURI(t *testing.T) {
	tests := []struct {
		name      string
		redirectURI string
		wantErr   bool
	}{
		{"valid HTTP", "http://localhost:8080/callback", false},
		{"valid HTTPS", "https://example.com/callback", false},
		{"valid with path", "http://localhost:8080/callback?state=abc", false},
		{"valid with query", "http://localhost:8080/callback?client_id=123", false},
		{"empty", "", true},
		{"invalid scheme", "ftp://localhost/callback", true},
		{"empty string", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRedirectURI(tt.redirectURI)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateRedirectURI() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateToken(t *testing.T) {
	server := &OAuth2ProxyServer{
		tokens: make(map[string]*TokenData),
	}

	// Test invalid token
	_, err := server.validateToken("invalid-token")
	if err != errInvalidToken {
		t.Errorf("validateToken() error = %v, want %v", err, errInvalidToken)
	}

	// Test expired token
	expiresAt := time.Now().Add(-time.Hour)
	tokenData := &TokenData{
		AccessToken: "expired-token",
		ExpiresAt:   expiresAt,
	}
	server.tokens["expired-token"] = tokenData

	_, err = server.validateToken("expired-token")
	if err != errTokenExpired {
		t.Errorf("validateToken() error = %v, want %v", err, errTokenExpired)
	}

	// Test valid token
	claims := generateJWTClaims("client1", "user1", []string{"read", "write"}, time.Now().Add(time.Hour))
	validTokenData := &TokenData{
		AccessToken:  generateJWT(&claims),
		ExpiresAt:    time.Now().Add(time.Hour),
		IssuedAt:     time.Now(),
		Claims:       claims,
		Scopes:       []string{"read", "write"},
		UserInfo: UserInfo{
			ID: "user1",
		},
	}
	server.tokens["valid-token"] = validTokenData

	token, err := server.validateToken("valid-token")
	if err != nil {
		t.Errorf("validateToken() error = %v", err)
	}
	if token.ExpiresAt.Before(time.Now()) {
		t.Error("token should not be expired")
	}
}

func TestGenerateJWTClaims(t *testing.T) {
	claims := generateJWTClaims("client1", "user1", []string{"read", "write", "admin"}, time.Now().Add(2*time.Hour))

	if claims.Subject != "user1" {
		t.Errorf("claims.Subject = %v, want user1", claims.Subject)
	}
	if claims.Issuer != "oauth2-proxy" {
		t.Errorf("claims.Issuer = %v, want oauth2-proxy", claims.Issuer)
	}
	if claims.ExpiresAt.Before(time.Now().Add(time.Hour)) {
		t.Errorf("claims.ExpiresAt too early")
	}
	if claims.ExpiresAt.After(time.Now().Add(3*time.Hour)) {
		t.Errorf("claims.ExpiresAt too late")
	}
}

func TestGenerateJWT(t *testing.T) {
	claims := generateJWTClaims("client1", "user1", []string{"read"}, time.Now().Add(time.Hour))
	token := generateJWT(&claims)

	if token == "" {
		t.Error("generateJWT() returned empty token")
	}

	// Parse the token
	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return []byte(jwtSecretKey), nil
	})
	if err != nil {
		t.Fatalf("jwt.Parse() error = %v", err)
	}

	if !parsedToken.Valid {
		t.Error("parsed token is not valid")
	}
}

func TestGenerateTokenData(t *testing.T) {
	tokenData := generateTokenData("client1", "user1", []string{"read", "write"}, time.Now().Add(time.Hour))

	if tokenData.AccessToken == "" {
		t.Error("tokenData.AccessToken is empty")
	}
	if tokenData.RefreshToken == "" {
		t.Error("tokenData.RefreshToken is empty")
	}
	if len(tokenData.Scopes) != 2 {
		t.Errorf("tokenData.Scopes length = %d, want 2", len(tokenData.Scopes))
	}
	if tokenData.UserInfo.ID != "user1" {
		t.Errorf("tokenData.UserInfo.ID = %v, want user1", tokenData.UserInfo.ID)
	}
	if tokenData.ExpiresAt.Before(time.Now()) {
		t.Error("tokenData.ExpiresAt should be in the future")
	}
}

func TestAuthHandler(t *testing.T) {
	server := &OAuth2ProxyServer{
		oauth2Config: OAuth2Config{
			AuthorizationURL: "https://example.com/oauth/authorize",
			ClientID:         "test-client",
			RedirectURI:      "http://localhost:8080/callback",
			Scopes:           []string{"read", "write"},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/auth?client_id=test-client&redirect_uri=http://localhost:8080/callback", nil)
	rr := httptest.NewRecorder()

	server.authHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("authHandler status = %d, want %d", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "https://example.com/oauth/authorize") {
		t.Error("response should contain authorization URL")
	}
	if !strings.Contains(body, "test-client") {
		t.Error("response should contain client ID")
	}
}

func TestCallbackHandler(t *testing.T) {
	server := &OAuth2ProxyServer{
		oauth2Config: OAuth2Config{
			RedirectURI: "http://localhost:8080/callback",
		},
	}

	tests := []struct {
		name    string
		code    string
		state   string
		wantErr bool
	}{
		{"valid code", "test-code-123", "dGVzdC1jb2Rl", false},
		{"missing code", "", "dGVzdC1jb2Rl", true},
		{"invalid state", "test-code-123", "invalid!!!", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/callback", nil)
			if tt.code != "" {
				req.URL.RawQuery = "code=" + tt.code
			}
			if tt.state != "" {
				req.URL.RawQuery = "code=" + tt.code + "&state=" + tt.state
			}

			rr := httptest.NewRecorder()
			server.callbackHandler(rr, req)

			// For valid requests, should redirect
			if !tt.wantErr {
				if rr.Code != http.StatusFound {
					t.Errorf("callbackHandler status = %d, want %d", rr.Code, http.StatusFound)
				}
			}
		})
	}
}

func TestTokenEndpoint(t *testing.T) {
	server := &OAuth2ProxyServer{
		tokens: make(map[string]*TokenData),
	}

	tests := []struct {
		name     string
		body     map[string]interface{}
		wantCode int
	}{
		{"valid authorization_code", map[string]interface{}{
			"grant_type":    "authorization_code",
			"code":          "test-code-123",
			"redirect_uri":  "http://localhost:8080/callback",
			"client_id":     "test-client",
			"client_secret": "test-secret",
		}, http.StatusOK},
		{"missing code", map[string]interface{}{
			"grant_type":    "authorization_code",
			"redirect_uri":  "http://localhost:8080/callback",
			"client_id":     "test-client",
			"client_secret": "test-secret",
		}, http.StatusBadRequest},
		{"missing redirect_uri", map[string]interface{}{
			"grant_type":    "authorization_code",
			"code":          "test-code-123",
			"client_id":     "test-client",
			"client_secret": "test-secret",
		}, http.StatusBadRequest},
		{"invalid redirect_uri", map[string]interface{}{
			"grant_type":    "authorization_code",
			"code":          "test-code-123",
			"redirect_uri":  "ftp://localhost/callback",
			"client_id":     "test-client",
			"client_secret": "test-secret",
		}, http.StatusBadRequest},
		{"refresh_token grant", map[string]interface{}{
			"grant_type":     "refresh_token",
			"refresh_token":  "test-refresh-token",
			"client_id":      "test-client",
			"client_secret":  "test-secret",
		}, http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/oauth/token", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			server.tokenEndpoint(rr, req)

			if rr.Code != tt.wantCode {
				t.Errorf("tokenEndpoint status = %d, want %d", rr.Code, tt.wantCode)
			}
		})
	}
}

func TestUserInfoEndpoint(t *testing.T) {
	server := &OAuth2ProxyServer{
		tokens: make(map[string]*TokenData),
	}

	// Create a valid token
	claims := generateJWTClaims("client1", "user1", []string{"read", "write"}, time.Now().Add(time.Hour))
	tokenData := &TokenData{
		AccessToken:  generateJWT(&claims),
		ExpiresAt:    time.Now().Add(time.Hour),
		IssuedAt:     time.Now(),
		Claims:       claims,
		Scopes:       []string{"read", "write"},
		UserInfo: UserInfo{
			ID:       "user1",
			Email:    "user@example.com",
			Name:     "Test User",
			Username: "testuser",
		},
	}
	server.tokens["test-token"] = tokenData

	tests := []struct {
		name       string
		authHeader string
		wantCode   int
	}{
		{"valid token", "Bearer test-token", http.StatusOK},
		{"no auth header", "", http.StatusUnauthorized},
		{"invalid auth header", "Basic abc123", http.StatusUnauthorized},
		{"invalid token", "Bearer invalid-token", http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/oauth/userinfo", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			rr := httptest.NewRecorder()
			server.userInfoEndpoint(rr, req)

			if rr.Code != tt.wantCode {
				t.Errorf("userInfoEndpoint status = %d, want %d", rr.Code, tt.wantCode)
			}
		})
	}
}

func TestTokenEndpointMethods(t *testing.T) {
	server := &OAuth2ProxyServer{}

	// Test GET request
	req := httptest.NewRequest(http.MethodGet, "/oauth/token", nil)
	rr := httptest.NewRecorder()
	server.tokenEndpoint(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("tokenEndpoint with GET status = %d, want %d", rr.Code, http.StatusMethodNotAllowed)
	}
}

func TestLLMEndpoint(t *testing.T) {
	server := &OAuth2ProxyServer{
		tokens: make(map[string]*TokenData),
	}

	// Create a valid token
	claims := generateJWTClaims("client1", "user1", []string{"read", "write"}, time.Now().Add(time.Hour))
	tokenData := &TokenData{
		AccessToken:  generateJWT(&claims),
		ExpiresAt:    time.Now().Add(time.Hour),
		IssuedAt:     time.Now(),
		Claims:       claims,
		Scopes:       []string{"read", "write"},
		UserInfo: UserInfo{
			ID: "user1",
		},
	}
	server.tokens["test-token"] = tokenData

	tests := []struct {
		name       string
		authHeader string
		wantCode   int
	}{
		{"valid token", "Bearer test-token", http.StatusBadGateway}, // Mock will fail
		{"no auth header", "", http.StatusUnauthorized},
		{"invalid auth header", "Basic abc123", http.StatusUnauthorized},
		{"invalid token", "Bearer invalid-token", http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := `{"model": "llama2", "messages": [{"role": "user", "content": "Hello"}]}`
			req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			server.llmEndpoint(rr, req)

			if rr.Code != tt.wantCode {
				t.Errorf("llmEndpoint status = %d, want %d", rr.Code, tt.wantCode)
			}
		})
	}
}

func TestLLMEndpointMethods(t *testing.T) {
	server := &OAuth2ProxyServer{}

	// Test GET request
	req := httptest.NewRequest(http.MethodGet, "/v1/chat/completions", nil)
	rr := httptest.NewRecorder()
	server.llmEndpoint(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("llmEndpoint with GET status = %d, want %d", rr.Code, http.StatusMethodNotAllowed)
	}
}

func TestTokenResponseSerialization(t *testing.T) {
	tokenResponse := TokenResponse{
		AccessToken:  "test-access-token",
		TokenType:    "Bearer",
		ExpiresIn:    7200,
		RefreshToken: "test-refresh-token",
		Scope:        "read write",
	}

	data, err := json.Marshal(tokenResponse)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var result TokenResponse
	err = json.Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if result.AccessToken != tokenResponse.AccessToken {
		t.Errorf("AccessToken = %v, want %v", result.AccessToken, tokenResponse.AccessToken)
	}
	if result.TokenType != tokenResponse.TokenType {
		t.Errorf("TokenType = %v, want %v", result.TokenType, tokenResponse.TokenType)
	}
	if result.ExpiresIn != tokenResponse.ExpiresIn {
		t.Errorf("ExpiresIn = %v, want %v", result.ExpiresIn, tokenResponse.ExpiresIn)
	}
	if result.RefreshToken != tokenResponse.RefreshToken {
		t.Errorf("RefreshToken = %v, want %v", result.RefreshToken, tokenResponse.RefreshToken)
	}
	if result.Scope != tokenResponse.Scope {
		t.Errorf("Scope = %v, want %v", result.Scope, tokenResponse.Scope)
	}
}

func TestGenerateJWTSerialization(t *testing.T) {
	claims := generateJWTClaims("client1", "user1", []string{"read"}, time.Now().Add(time.Hour))
	token := generateJWT(&claims)

	if token == "" {
		t.Error("generateJWT() returned empty token")
	}

	// Parse the token
	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return []byte(jwtSecretKey), nil
	})
	if err != nil {
		t.Fatalf("jwt.Parse() error = %v", err)
	}

	if !parsedToken.Valid {
		t.Error("parsed token is not valid")
	}
}

func TestGenerateJWTClaimsSerialization(t *testing.T) {
	tests := []struct {
		name    string
		claims  jwt.RegisteredClaims
		wantErr bool
	}{
		{"valid claims", generateJWTClaims("client1", "user1", []string{"read"}, time.Now().Add(time.Hour)), false},
		{"nil claims", jwt.RegisteredClaims{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := generateJWT(&tt.claims)
			if token == "" {
				t.Error("generateJWT() returned empty token")
			}
		})
	}
}

func TestTokenDataFields(t *testing.T) {
	tokenData := generateTokenData("client1", "user1", []string{"read", "write", "admin"}, time.Now().Add(2*time.Hour))

	if tokenData.AccessToken == "" {
		t.Error("AccessToken is empty")
	}
	if tokenData.RefreshToken == "" {
		t.Error("RefreshToken is empty")
	}
	if len(tokenData.Scopes) != 3 {
		t.Errorf("Scopes length = %d, want 3", len(tokenData.Scopes))
	}
	if tokenData.UserInfo.ID != "user1" {
		t.Errorf("UserInfo.ID = %v, want user1", tokenData.UserInfo.ID)
	}
	if !tokenData.Claims.ExpiresAt.Before(time.Now().Add(3*time.Hour)) {
		t.Error("ExpiresAt should not be too far in the future")
	}
	if !tokenData.Claims.ExpiresAt.After(time.Now()) {
		t.Error("ExpiresAt should be in the future")
	}
}

func TestConfigLoadDefault(t *testing.T) {
	// Temporarily rename config file if it exists
	configFile := "config.json"
	if _, err := os.Stat(configFile); err == nil {
		os.Rename(configFile, configFile+".bak")
		defer os.Rename(configFile+".bak", configFile)
	}

	config := loadConfig()

	if config.Host != "0.0.0.0" {
		t.Errorf("Host = %v, want 0.0.0.0", config.Host)
	}
	if config.Port != defaultPort {
		t.Errorf("Port = %d, want %d", config.Port, defaultPort)
	}
}

func TestSaveToken(t *testing.T) {
	server := &OAuth2ProxyServer{
		tokens: make(map[string]*TokenData),
	}

	tokenData := generateTokenData("client1", "user1", []string{"read"}, time.Now().Add(time.Hour))
	server.saveToken(tokenData.AccessToken, tokenData)

	if _, exists := server.tokens[tokenData.AccessToken]; !exists {
		t.Error("token not saved")
	}
}

func TestTokenExpiration(t *testing.T) {
	server := &OAuth2ProxyServer{
		tokens: make(map[string]*TokenData),
	}

	// Create expired token
	expiresAt := time.Now().Add(-time.Hour)
	tokenData := &TokenData{
		AccessToken: "expired-token",
		ExpiresAt:   expiresAt,
	}
	server.tokens["expired"] = tokenData

	_, err := server.validateToken("expired")
	if err != errTokenExpired {
		t.Errorf("validateToken() error = %v, want %v", err, errTokenExpired)
	}
}

func TestOAuth2ConfigGet(t *testing.T) {
	tests := []struct {
		name    string
		query   url.Values
		wantErr bool
	}{
		{"all params", url.Values{
			"client_id":      []string{"test-client"},
			"redirect_uri":   []string{"http://localhost:8080/callback"},
			"response_type":  []string{"code"},
			"scope":          []string{"read", "write"},
		}, false},
		{"missing client_id", url.Values{
			"redirect_uri":   []string{"http://localhost:8080/callback"},
			"response_type":  []string{"code"},
			"scope":          []string{"read", "write"},
		}, true},
		{"missing redirect_uri", url.Values{
			"client_id":      []string{"test-client"},
			"response_type":  []string{"code"},
			"scope":          []string{"read", "write"},
		}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/auth", nil)
			req.URL.RawQuery = tt.query.Encode()

			_, err := getOAuth2Config(req)
			if (err != nil) != tt.wantErr {
				t.Errorf("getOAuth2Config() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateState(t *testing.T) {
	tests := []struct {
		name    string
		state   string
		wantErr bool
	}{
		{"empty state", "", false},
		{"valid base64", "dGVzdC1zdGF0ZS1zZWNyZXQ=", false},
		{"invalid base64", "!!!invalid!!!", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateState(tt.state)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateState() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}



func TestGenerateCodeChallenge(t *testing.T) {
	tests := []struct {
		name        string
		codeVerifier string
		expectedLen int
	}{
		{"short", "abc", 43},
		{"medium", "1234567890", 43},
		{"long", "abcdefghijklmnopqrstuvwxyz1234567890", 43},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			challenge := generateCodeChallenge(tt.codeVerifier)
			if len(challenge) != tt.expectedLen {
				t.Errorf("generateCodeChallenge() len = %d, want %d", len(challenge), tt.expectedLen)
			}
			if challenge == "" {
				t.Error("generateCodeChallenge() returned empty challenge")
			}
		})
	}
}

func TestHTTPMethodNotAllowed(t *testing.T) {
	server := &OAuth2ProxyServer{}

	// Test auth handler with POST
	req := httptest.NewRequest(http.MethodPost, "/auth", nil)
	rr := httptest.NewRecorder()
	server.authHandler(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("authHandler with POST status = %d, want %d", rr.Code, http.StatusMethodNotAllowed)
	}
}

func TestUserInfoEndpointJSON(t *testing.T) {
	server := &OAuth2ProxyServer{
		tokens: make(map[string]*TokenData),
	}

	// Create a valid token
	claims := generateJWTClaims("client1", "user1", []string{"read", "write"}, time.Now().Add(time.Hour))
	tokenData := &TokenData{
		AccessToken:  generateJWT(&claims),
		ExpiresAt:    time.Now().Add(time.Hour),
		IssuedAt:     time.Now(),
		Claims:       claims,
		Scopes:       []string{"read", "write"},
		UserInfo: UserInfo{
			ID:       "user1",
			Email:    "user@example.com",
			Name:     "Test User",
			Username: "testuser",
		},
	}
	server.tokens["test-token"] = tokenData

	req := httptest.NewRequest(http.MethodGet, "/oauth/userinfo", nil)
	req.Header.Set("Authorization", "Bearer test-token")

	rr := httptest.NewRecorder()
	server.userInfoEndpoint(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("userInfoEndpoint status = %d, want %d", rr.Code, http.StatusOK)
	}

	var result UserInfo
	err := json.Unmarshal(rr.Body.Bytes(), &result)
	if err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if result.ID != "user1" {
		t.Errorf("UserInfo.ID = %v, want user1", result.ID)
	}
	if result.Email != "user@example.com" {
		t.Errorf("UserInfo.Email = %v, want user@example.com", result.Email)
	}
}
