package oauth

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/oauth2-proxy/local-llm-proxy/config"
	"github.com/oauth2-proxy/local-llm-proxy/db"
	"github.com/oauth2-proxy/local-llm-proxy/models"
)

func TestOAuthServerGenerateCodeVerifier(t *testing.T) {
	t.Run("should generate valid code verifier", func(t *testing.T) {
		dbPath := "./test_db/oauth_verifier.db"
		os.Remove(dbPath + "-wal")
		os.Remove(dbPath + "-shm")

		database, err := db.NewDatabase(dbPath)
		if err != nil {
			t.Fatalf("Failed to create database: %v", err)
		}
		defer database.Close()

		cfg := &config.OAuthConfig{
			CodeChallengeMethod: "S256",
		}

		oauthServer := NewOAuthServer(database, cfg)

		verifier, err := oauthServer.GenerateCodeVerifier()
		if err != nil {
			t.Fatalf("Failed to generate code verifier: %v", err)
		}

		if verifier == "" {
			t.Error("Code verifier should not be empty")
		}

		// Verify it's valid base64
		_, err = base64.URLEncoding.DecodeString(verifier)
		if err != nil {
			t.Errorf("Code verifier should be valid base64: %v", err)
		}
	})
}

func TestOAuthServerGenerateCodeChallenge(t *testing.T) {
	t.Run("should generate S256 code challenge", func(t *testing.T) {
		dbPath := "./test_db/oauth_challenge.db"
		os.Remove(dbPath + "-wal")
		os.Remove(dbPath + "-shm")

		database, err := db.NewDatabase(dbPath)
		if err != nil {
			t.Fatalf("Failed to create database: %v", err)
		}
		defer database.Close()

		cfg := &config.OAuthConfig{
			CodeChallengeMethod: "S256",
		}

		oauthServer := NewOAuthServer(database, cfg)

		verifier := "test-verifier-123"
		challenge, err := oauthServer.GenerateCodeChallenge(verifier, "S256")
		if err != nil {
			t.Fatalf("Failed to generate code challenge: %v", err)
		}

		if challenge == "" {
			t.Error("Code challenge should not be empty")
		}

		// Verify it's valid base64 raw URL encoding
		_, err = base64.RawURLEncoding.DecodeString(challenge)
		if err != nil {
			t.Errorf("Code challenge should be valid base64 raw URL: %v", err)
		}
	})

	t.Run("should generate plain code challenge", func(t *testing.T) {
		dbPath := "./test_db/oauth_challenge2.db"
		os.Remove(dbPath + "-wal")
		os.Remove(dbPath + "-shm")

		database, err := db.NewDatabase(dbPath)
		if err != nil {
			t.Fatalf("Failed to create database: %v", err)
		}
		defer database.Close()

		cfg := &config.OAuthConfig{
			CodeChallengeMethod: "plain",
		}

		oauthServer := NewOAuthServer(database, cfg)

		verifier := "test-verifier-456"
		challenge, err := oauthServer.GenerateCodeChallenge(verifier, "plain")
		if err != nil {
			t.Fatalf("Failed to generate code challenge: %v", err)
		}

		if challenge != verifier {
			t.Errorf("Plain challenge should equal verifier, got %s", challenge)
		}
	})
}

func TestOAuthServerAuthorize(t *testing.T) {
	t.Run("should redirect to auth server", func(t *testing.T) {
		dbPath := "./test_db/oauth_authorize.db"
		os.Remove(dbPath + "-wal")
		os.Remove(dbPath + "-shm")

		database, err := db.NewDatabase(dbPath)
		if err != nil {
			t.Fatalf("Failed to create database: %v", err)
		}
		defer database.Close()

		cfg := &config.OAuthConfig{
			ClientID:            "test-client",
			RedirectURI:         "http://localhost:3000/callback",
			AuthorizationTimeout: 60 * time.Second,
			CodeChallengeMethod: "S256",
		}

		oauthServer := NewOAuthServer(database, cfg)

		req := httptest.NewRequest("GET", "/oauth/authorize?state=test-state&code_verifier=test-verifier", nil)
		rec := httptest.NewRecorder()

		oauthServer.Authorize(rec, req)

		if rec.Code != http.StatusFound {
			t.Errorf("Expected status 302, got %d", rec.Code)
		}

		// Check redirect location
		location := rec.Header().Get("Location")
		if location == "" {
			t.Error("Redirect location should be set")
		}

		if !containsString(location, "client_id=test-client") {
			t.Error("Redirect should contain client_id")
		}

		if !containsString(location, "redirect_uri=http://localhost:3000/callback") {
			t.Error("Redirect should contain redirect_uri")
		}

		if !containsString(location, "response_type=code") {
			t.Error("Redirect should contain response_type=code")
		}

		if !containsString(location, "state=test-state") {
			t.Error("Redirect should contain state")
		}

		if !containsString(location, "code_verifier=test-verifier") {
			t.Error("Redirect should contain code_verifier")
		}
	})

	t.Run("should reject without state", func(t *testing.T) {
		dbPath := "./test_db/oauth_authorize2.db"
		os.Remove(dbPath + "-wal")
		os.Remove(dbPath + "-shm")

		database, err := db.NewDatabase(dbPath)
		if err != nil {
			t.Fatalf("Failed to create database: %v", err)
		}
		defer database.Close()

		cfg := &config.OAuthConfig{
			ClientID:            "test-client",
			RedirectURI:         "http://localhost:3000/callback",
			AuthorizationTimeout: 60 * time.Second,
			CodeChallengeMethod: "S256",
		}

		oauthServer := NewOAuthServer(database, cfg)

		req := httptest.NewRequest("GET", "/oauth/authorize", nil)
		rec := httptest.NewRecorder()

		oauthServer.Authorize(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rec.Code)
		}
	})
}

func TestOAuthServerCallback(t *testing.T) {
	t.Run("should redirect to redirect URI", func(t *testing.T) {
		dbPath := "./test_db/oauth_callback.db"
		os.Remove(dbPath + "-wal")
		os.Remove(dbPath + "-shm")

		database, err := db.NewDatabase(dbPath)
		if err != nil {
			t.Fatalf("Failed to create database: %v", err)
		}
		defer database.Close()

		cfg := &config.OAuthConfig{
			ClientID:            "test-client",
			RedirectURI:         "http://localhost:3000/callback",
			AuthorizationTimeout: 60 * time.Second,
			CodeChallengeMethod: "S256",
		}

		oauthServer := NewOAuthServer(database, cfg)

		req := httptest.NewRequest("GET", "/oauth/callback?code=auth-code-123&state=test-state", nil)
		rec := httptest.NewRecorder()

		oauthServer.Callback(rec, req)

		if rec.Code != http.StatusFound {
			t.Errorf("Expected status 302, got %d", rec.Code)
		}

		location := rec.Header().Get("Location")
		if location != "http://localhost:3000/callback" {
			t.Errorf("Expected redirect to http://localhost:3000/callback, got %s", location)
		}
	})

	t.Run("should reject without code", func(t *testing.T) {
		dbPath := "./test_db/oauth_callback2.db"
		os.Remove(dbPath + "-wal")
		os.Remove(dbPath + "-shm")

		database, err := db.NewDatabase(dbPath)
		if err != nil {
			t.Fatalf("Failed to create database: %v", err)
		}
		defer database.Close()

		cfg := &config.OAuthConfig{
			ClientID:            "test-client",
			RedirectURI:         "http://localhost:3000/callback",
			AuthorizationTimeout: 60 * time.Second,
			CodeChallengeMethod: "S256",
		}

		oauthServer := NewOAuthServer(database, cfg)

		req := httptest.NewRequest("GET", "/oauth/callback?state=test-state", nil)
		rec := httptest.NewRecorder()

		oauthServer.Callback(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rec.Code)
		}
	})
}

func TestOAuthServerGetToken(t *testing.T) {
	t.Run("should return token with metadata", func(t *testing.T) {
		dbPath := "./test_db/oauth_token.db"
		os.Remove(dbPath + "-wal")
		os.Remove(dbPath + "-shm")

		database, err := db.NewDatabase(dbPath)
		if err != nil {
			t.Fatalf("Failed to create database: %v", err)
		}
		defer database.Close()

		cfg := &config.OAuthConfig{
			Scopes:               []string{"openid", "profile", "email"},
			AuthorizationTimeout: 60 * time.Second,
			CodeChallengeMethod:  "S256",
		}

		oauthServer := NewOAuthServer(database, cfg)

		ctx := context.Background()
		code := "test-auth-code"

		token, err := oauthServer.GetToken(ctx, code)
		if err != nil {
			t.Fatalf("Failed to get token: %v", err)
		}

		if token.AccessToken != code {
			t.Errorf("Expected access token %s, got %s", code, token.AccessToken)
		}

		if token.TokenType != "Bearer" {
			t.Errorf("Expected token type Bearer, got %s", token.TokenType)
		}

		// Check expiry
		if token.Expiry.Before(time.Now()) {
			t.Error("Token should not be expired")
		}
	})
}

func TestOAuthServerRefreshToken(t *testing.T) {
	t.Run("should return new token on refresh", func(t *testing.T) {
		dbPath := "./test_db/oauth_refresh.db"
		os.Remove(dbPath + "-wal")
		os.Remove(dbPath + "-shm")

		database, err := db.NewDatabase(dbPath)
		if err != nil {
			t.Fatalf("Failed to create database: %v", err)
		}
		defer database.Close()

		cfg := &config.OAuthConfig{
			Scopes:               []string{"openid", "profile"},
			AuthorizationTimeout: 60 * time.Second,
			CodeChallengeMethod:  "S256",
		}

		oauthServer := NewOAuthServer(database, cfg)

		ctx := context.Background()
		refreshToken := "test-refresh-token"

		token, err := oauthServer.RefreshToken(ctx, refreshToken)
		if err != nil {
			t.Fatalf("Failed to refresh token: %v", err)
		}

		if token.AccessToken != "new_access_token" {
			t.Errorf("Expected new access token, got %s", token.AccessToken)
		}

		if token.TokenType != "Bearer" {
			t.Errorf("Expected token type Bearer, got %s", token.TokenType)
		}
	})
}

func TestOAuthServerTokenExchange(t *testing.T) {
	t.Run("should exchange code for token", func(t *testing.T) {
		dbPath := "./test_db/oauth_exchange.db"
		os.Remove(dbPath + "-wal")
		os.Remove(dbPath + "-shm")

		database, err := db.NewDatabase(dbPath)
		if err != nil {
			t.Fatalf("Failed to create database: %v", err)
		}
		defer database.Close()

		cfg := &config.OAuthConfig{
			Scopes:               []string{"openid", "profile", "email"},
			AuthorizationTimeout: 60 * time.Second,
			CodeChallengeMethod:  "S256",
		}

		oauthServer := NewOAuthServer(database, cfg)

		ctx := context.Background()
		code := "test-exchange-code"

		token, err := oauthServer.TokenExchange(ctx, "test-client-id", code)
		if err != nil {
			t.Fatalf("Failed to exchange token: %v", err)
		}

		if token.AccessToken != code {
			t.Errorf("Expected access token %s, got %s", code, token.AccessToken)
		}
	})
}

func TestOAuthServerExchangeCode(t *testing.T) {
	t.Run("should exchange authorization code", func(t *testing.T) {
		dbPath := "./test_db/oauth_exchange2.db"
		os.Remove(dbPath + "-wal")
		os.Remove(dbPath + "-shm")

		database, err := db.NewDatabase(dbPath)
		if err != nil {
			t.Fatalf("Failed to create database: %v", err)
		}
		defer database.Close()

		cfg := &config.OAuthConfig{
			Scopes:               []string{"openid", "profile"},
			AuthorizationTimeout: 60 * time.Second,
			CodeChallengeMethod:  "S256",
		}

		oauthServer := NewOAuthServer(database, cfg)

		ctx := context.Background()
		code := "test-code-exchange"

		token, err := oauthServer.ExchangeCode(ctx, code, "http://localhost:3000/callback")
		if err != nil {
			t.Fatalf("Failed to exchange code: %v", err)
		}

		if token.AccessToken != code {
			t.Errorf("Expected access token %s, got %s", code, token.AccessToken)
		}
	})
}

func TestOAuthServerWithTx(t *testing.T) {
	t.Run("should create database transaction", func(t *testing.T) {
		dbPath := "./test_db/oauth_tx.db"
		os.Remove(dbPath + "-wal")
		os.Remove(dbPath + "-shm")

		database, err := db.NewDatabase(dbPath)
		if err != nil {
			t.Fatalf("Failed to create database: %v", err)
		}
		defer database.Close()

		tx, err := database.WithTx()
		if err != nil {
			t.Fatalf("Failed to create transaction: %v", err)
		}

		if tx == nil {
			t.Error("Transaction should not be nil")
		}
	})
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
