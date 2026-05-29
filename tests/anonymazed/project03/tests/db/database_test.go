package db

import (
	"os"
	"testing"
	"time"

	"github.com/oauth2-proxy/local-llm-proxy/config"
	"github.com/oauth2-proxy/local-llm-proxy/models"
)

func TestNewDatabase(t *testing.T) {
	t.Run("should create database", func(t *testing.T) {
		db, err := NewDatabase("./test_db/test1.db")
		if err != nil {
			t.Fatalf("Failed to create database: %v", err)
		}
		defer db.Close()

		if db == nil {
			t.Error("Database should not be nil")
		}
	})

	t.Run("should create data directory", func(t *testing.T) {
		// Clean up before test
		os.Remove("./test_db/test2.db")
		os.Remove("./test_db/test2.db-wal")
		os.Remove("./test_db/test2.db-shm")

		db, err := NewDatabase("./test_db/test2.db")
		if err != nil {
			t.Fatalf("Failed to create database: %v", err)
		}
		defer db.Close()

		// Check if directory was created
		if _, err := os.Stat("./test_db"); os.IsNotExist(err) {
			t.Error("Data directory should be created")
		}
	})
}

func TestCreateAndGetClient(t *testing.T) {
	t.Run("should create and get client", func(t *testing.T) {
		dbPath := "./test_db/client_test.db"
		// Clean up
		os.Remove(dbPath + "-wal")
		os.Remove(dbPath + "-shm")
		db, err := NewDatabase(dbPath)
		if err != nil {
			t.Fatalf("Failed to create database: %v", err)
		}
		defer db.Close()

		client := &models.Client{
			ID:          "test-client-1",
			ClientID:    "test-client-id",
			ClientSecret: "test-secret",
			RedirectURI: "http://localhost:3000/callback",
			Scopes:      []string{"openid", "profile", "email"},
			GrantTypes:  []string{"authorization_code", "refresh_token"},
			CreatedAt:   time.Now(),
		}

		// Create client
		if err := db.CreateClient(client); err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}

		// Get client
		got, err := db.GetClient("test-client-id")
		if err != nil {
			t.Fatalf("Failed to get client: %v", err)
		}

		if got.ClientID != client.ClientID {
			t.Errorf("Expected client ID %s, got %s", client.ClientID, got.ClientID)
		}
		if got.ClientSecret != client.ClientSecret {
			t.Errorf("Expected client secret %s, got %s", client.ClientSecret, got.ClientSecret)
		}
	})

	t.Run("should return error for non-existent client", func(t *testing.T) {
		dbPath := "./test_db/client_test2.db"
		os.Remove(dbPath + "-wal")
		os.Remove(dbPath + "-shm")
		db, err := NewDatabase(dbPath)
		if err != nil {
			t.Fatalf("Failed to create database: %v", err)
		}
		defer db.Close()

		_, err = db.GetClient("non-existent")
		if err == nil {
			t.Error("Expected error for non-existent client")
		}
	})
}

func TestCreateAndDeleteClient(t *testing.T) {
	t.Run("should delete client", func(t *testing.T) {
		dbPath := "./test_db/client_test3.db"
		os.Remove(dbPath + "-wal")
		os.Remove(dbPath + "-shm")
		db, err := NewDatabase(dbPath)
		if err != nil {
			t.Fatalf("Failed to create database: %v", err)
		}
		defer db.Close()

		client := &models.Client{
			ID:          "test-client-2",
			ClientID:    "test-client-id-2",
			ClientSecret: "test-secret-2",
			RedirectURI: "http://localhost:3000/callback",
			CreatedAt:   time.Now(),
		}

		if err := db.CreateClient(client); err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}

		// Verify client exists
		_, err = db.GetClient("test-client-id-2")
		if err != nil {
			t.Fatalf("Client should exist: %v", err)
		}

		// Delete client
		if err := db.DeleteClient("test-client-id-2"); err != nil {
			t.Fatalf("Failed to delete client: %v", err)
		}

		// Verify client is deleted
		_, err = db.GetClient("test-client-id-2")
		if err == nil {
			t.Error("Client should be deleted")
		}
	})
}

func TestCreateAndGetToken(t *testing.T) {
	t.Run("should create and get token", func(t *testing.T) {
		dbPath := "./test_db/token_test.db"
		os.Remove(dbPath + "-wal")
		os.Remove(dbPath + "-shm")
		db, err := NewDatabase(dbPath)
		if err != nil {
			t.Fatalf("Failed to create database: %v", err)
		}
		defer db.Close()

		token := &models.Token{
			AccessToken:  "test-access-token",
			RefreshToken: "test-refresh-token",
			ExpiresAt:    time.Now().Add(time.Hour),
			TokenType:    "Bearer",
			Scopes:       "openid profile email",
			Subject:      "user",
			ClientID:     "test-client",
			CreatedAt:    time.Now(),
		}

		if err := db.CreateToken(token); err != nil {
			t.Fatalf("Failed to create token: %v", err)
		}

		got, err := db.GetToken("test-access-token")
		if err != nil {
			t.Fatalf("Failed to get token: %v", err)
		}

		if got.AccessToken != token.AccessToken {
			t.Errorf("Expected access token %s, got %s", token.AccessToken, got.AccessToken)
		}
	})

	t.Run("should return error for expired token", func(t *testing.T) {
		dbPath := "./test_db/token_test2.db"
		os.Remove(dbPath + "-wal")
		os.Remove(dbPath + "-shm")
		db, err := NewDatabase(dbPath)
		if err != nil {
			t.Fatalf("Failed to create database: %v", err)
		}
		defer db.Close()

		token := &models.Token{
			AccessToken:  "test-access-token",
			ExpiresAt:    time.Now().Add(-time.Hour), // Expired
			TokenType:    "Bearer",
			Scopes:       "openid profile email",
			Subject:      "user",
			ClientID:     "test-client",
			CreatedAt:    time.Now(),
		}

		if err := db.CreateToken(token); err != nil {
			t.Fatalf("Failed to create token: %v", err)
		}

		// Wait a bit to ensure it's expired
		time.Sleep(100 * time.Millisecond)

		valid, err := db.IsTokenValid("test-access-token")
		if err != nil {
			t.Fatalf("Failed to check token validity: %v", err)
		}
		if valid {
			t.Error("Token should be invalid (expired)")
		}
	})
}

func TestCreateAndDeleteCodeByCode(t *testing.T) {
	t.Run("should create and get code", func(t *testing.T) {
		dbPath := "./test_db/code_test.db"
		os.Remove(dbPath + "-wal")
		os.Remove(dbPath + "-shm")
		db, err := NewDatabase(dbPath)
		if err != nil {
			t.Fatalf("Failed to create database: %v", err)
		}
		defer db.Close()

		code := &models.OAuthCode{
			Code:        "test-code-123",
			ClientID:    "test-client",
			RedirectURI: "http://localhost:3000/callback",
			ExpiresAt:   time.Now().Add(time.Hour),
			CreatedAt:   time.Now(),
			Scopes:      "openid profile email",
		}

		if err := db.CreateCode(code); err != nil {
			t.Fatalf("Failed to create code: %v", err)
		}

		got, err := db.GetCode("test-code-123")
		if err != nil {
			t.Fatalf("Failed to get code: %v", err)
		}

		if got.Code != code.Code {
			t.Errorf("Expected code %s, got %s", code.Code, got.Code)
		}
	})

	t.Run("should check code expiration", func(t *testing.T) {
		dbPath := "./test_db/code_test2.db"
		os.Remove(dbPath + "-wal")
		os.Remove(dbPath + "-shm")
		db, err := NewDatabase(dbPath)
		if err != nil {
			t.Fatalf("Failed to create database: %v", err)
		}
		defer db.Close()

		code := &models.OAuthCode{
			Code:        "test-code-456",
			ClientID:    "test-client",
			RedirectURI: "http://localhost:3000/callback",
			ExpiresAt:   time.Now().Add(-time.Hour), // Expired
			CreatedAt:   time.Now(),
			Scopes:      "openid profile email",
		}

		if err := db.CreateCode(code); err != nil {
			t.Fatalf("Failed to create code: %v", err)
		}

		// Check if code is expired
		isExpired, err := db.IsCodeExpired("test-code-456")
		if err != nil {
			t.Fatalf("Failed to check code expiration: %v", err)
		}
		if !isExpired {
			t.Error("Code should be expired")
		}
	})
}

func TestCreatePKCECodeVerifier(t *testing.T) {
	t.Run("should create and get PKCE verifier", func(t *testing.T) {
		dbPath := "./test_db/pkce_test.db"
		os.Remove(dbPath + "-wal")
		os.Remove(dbPath + "-shm")
		db, err := NewDatabase(dbPath)
		if err != nil {
			t.Fatalf("Failed to create database: %v", err)
		}
		defer db.Close()

		verifier := &models.PKCECodeVerifier{
			ID:                       "verifier-1",
			CodeVerifier:             "code-verifier-abc123",
			CodeChallenge:            "code-challenge-xyz789",
			CodeChallengeMethod:      "S256",
			CreatedAt:                time.Now(),
		}

		if err := db.CreatePKCECodeVerifier(verifier); err != nil {
			t.Fatalf("Failed to create verifier: %v", err)
		}

		got, err := db.GetPKCECodeVerifier("code-challenge-xyz789")
		if err != nil {
			t.Fatalf("Failed to get verifier: %v", err)
		}

		if got.CodeVerifier != verifier.CodeVerifier {
			t.Errorf("Expected verifier %s, got %s", verifier.CodeVerifier, got.CodeVerifier)
		}
		if got.CodeChallengeMethod != verifier.CodeChallengeMethod {
			t.Errorf("Expected method %s, got %s", verifier.CodeChallengeMethod, got.CodeChallengeMethod)
		}
	})
}

func TestDeleteExpiredTokens(t *testing.T) {
	t.Run("should delete expired tokens", func(t *testing.T) {
		dbPath := "./test_db/expired_test.db"
		os.Remove(dbPath + "-wal")
		os.Remove(dbPath + "-shm")
		db, err := NewDatabase(dbPath)
		if err != nil {
			t.Fatalf("Failed to create database: %v", err)
		}
		defer db.Close()

		// Create valid token
		validToken := &models.Token{
			AccessToken:  "valid-token",
			ExpiresAt:    time.Now().Add(time.Hour),
			TokenType:    "Bearer",
			Scopes:       "openid",
			Subject:      "user",
			ClientID:     "test-client",
			CreatedAt:    time.Now(),
		}

		// Create expired token
		expiredToken := &models.Token{
			AccessToken:  "expired-token",
			ExpiresAt:    time.Now().Add(-time.Hour),
			TokenType:    "Bearer",
			Scopes:       "openid",
			Subject:      "user",
			ClientID:     "test-client",
			CreatedAt:    time.Now(),
		}

		db.CreateToken(validToken)
		db.CreateToken(expiredToken)

		// Delete expired tokens
		count, err := db.DeleteExpiredTokens()
		if err != nil {
			t.Fatalf("Failed to delete expired tokens: %v", err)
		}

		if count != 1 {
			t.Errorf("Expected to delete 1 token, deleted %d", count)
		}

		// Verify valid token still exists
		_, err = db.GetToken("valid-token")
		if err != nil {
			t.Error("Valid token should still exist")
		}
	})
}

func TestDeleteExpiredCodes(t *testing.T) {
	t.Run("should delete expired codes", func(t *testing.T) {
		dbPath := "./test_db/codes_expiration_test.db"
		os.Remove(dbPath + "-wal")
		os.Remove(dbPath + "-shm")
		db, err := NewDatabase(dbPath)
		if err != nil {
			t.Fatalf("Failed to create database: %v", err)
		}
		defer db.Close()

		// Create valid code
		validCode := &models.OAuthCode{
			Code:        "valid-code",
			ClientID:    "test-client",
			RedirectURI: "http://localhost:3000/callback",
			ExpiresAt:   time.Now().Add(time.Hour),
			CreatedAt:   time.Now(),
			Scopes:      "openid",
		}

		// Create expired code
		expiredCode := &models.OAuthCode{
			Code:        "expired-code",
			ClientID:    "test-client",
			RedirectURI: "http://localhost:3000/callback",
			ExpiresAt:   time.Now().Add(-time.Hour),
			CreatedAt:   time.Now(),
			Scopes:      "openid",
		}

		db.CreateCode(validCode)
		db.CreateCode(expiredCode)

		// Delete expired codes
		count, err := db.DeleteExpiredCodes()
		if err != nil {
			t.Fatalf("Failed to delete expired codes: %v", err)
		}

		if count != 1 {
			t.Errorf("Expected to delete 1 code, deleted %d", count)
		}
	})
}

func TestCloseDatabase(t *testing.T) {
	t.Run("should close database", func(t *testing.T) {
		dbPath := "./test_db/close_test.db"
		os.Remove(dbPath + "-wal")
		os.Remove(dbPath + "-shm")
		db, err := NewDatabase(dbPath)
		if err != nil {
			t.Fatalf("Failed to create database: %v", err)
		}

		if err := db.Close(); err != nil {
			t.Fatalf("Failed to close database: %v", err)
		}
	})
}

func TestGetTokenByAccessToken(t *testing.T) {
	t.Run("should get token by access token", func(t *testing.T) {
		dbPath := "./test_db/token_by_access_test.db"
		os.Remove(dbPath + "-wal")
		os.Remove(dbPath + "-shm")
		db, err := NewDatabase(dbPath)
		if err != nil {
			t.Fatalf("Failed to create database: %v", err)
		}
		defer db.Close()

		token := &models.Token{
			AccessToken:  "my-access-token",
			RefreshToken: "my-refresh-token",
			ExpiresAt:    time.Now().Add(time.Hour),
			TokenType:    "Bearer",
			Scopes:       "openid profile",
			Subject:      "user123",
			ClientID:     "client123",
			CreatedAt:    time.Now(),
		}

		if err := db.CreateToken(token); err != nil {
			t.Fatalf("Failed to create token: %v", err)
		}

		got, err := db.GetTokenByAccessToken("my-access-token")
		if err != nil {
			t.Fatalf("Failed to get token by access token: %v", err)
		}

		if got.Subject != token.Subject {
			t.Errorf("Expected subject %s, got %s", token.Subject, got.Subject)
		}
	})
}
