package storage

import (
	"testing"
	"time"

	"llm-proxy/models"
)

func TestSaveAndGetAuthCode(t *testing.T) {
	store := NewMemoryStore()

	code := "test-code"
	authCode := &models.OAuth2AuthCode{
		Code:      code,
		ClientID:  "test-client",
		Scopes:    "read",
		RedirectURI: "http://localhost/callback",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(10 * time.Minute),
		Used:      false,
	}

	if err := store.SaveAuthCode(authCode); err != nil {
		t.Fatalf("failed to save auth code: %v", err)
	}

	retrieved, err := store.GetAuthCode(code)
	if err != nil {
		t.Fatalf("failed to get auth code: %v", err)
	}

	if retrieved.Code != code {
		t.Errorf("expected code %s, got %s", code, retrieved.Code)
	}
}

func TestGetExpiredAuthCode(t *testing.T) {
	store := NewMemoryStore()

	code := "test-code"
	authCode := &models.OAuth2AuthCode{
		Code:      code,
		ClientID:  "test-client",
		Scopes:    "read",
		RedirectURI: "http://localhost/callback",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(-10 * time.Minute),
		Used:      false,
	}

	store.SaveAuthCode(authCode)

	_, err := store.GetAuthCode(code)
	if err == nil {
		t.Error("expected error for expired auth code")
	}
}

func TestGetUsedAuthCode(t *testing.T) {
	store := NewMemoryStore()

	code := "test-code"
	authCode := &models.OAuth2AuthCode{
		Code:      code,
		ClientID:  "test-client",
		Scopes:    "read",
		RedirectURI: "http://localhost/callback",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(10 * time.Minute),
		Used:      true,
	}

	store.SaveAuthCode(authCode)

	_, err := store.GetAuthCode(code)
	if err == nil {
		t.Error("expected error for used auth code")
	}
}

func TestSaveAndGetToken(t *testing.T) {
	store := NewMemoryStore()

	token := &models.OAuth2Token{
		AccessToken:  "test-token",
		ClientID:     "test-client",
		ExpiresAt:    time.Now().Add(1 * time.Hour),
		RefreshToken: "test-refresh",
		CreatedAt:    time.Now(),
	}

	if err := store.SaveToken(token); err != nil {
		t.Fatalf("failed to save token: %v", err)
	}

	retrieved, err := store.GetToken("test-token", "test-client")
	if err != nil {
		t.Fatalf("failed to get token: %v", err)
	}

	if retrieved.AccessToken != "test-token" {
		t.Errorf("expected test-token, got %s", retrieved.AccessToken)
	}
}

func TestGetExpiredToken(t *testing.T) {
	store := NewMemoryStore()

	token := &models.OAuth2Token{
		AccessToken:  "test-token",
		ClientID:     "test-client",
		ExpiresAt:    time.Now().Add(-1 * time.Hour),
		RefreshToken: "test-refresh",
		CreatedAt:    time.Now(),
	}

	store.SaveToken(token)

	_, err := store.GetToken("test-token", "test-client")
	if err == nil {
		t.Error("expected error for expired token")
	}
}

func TestCleanExpired(t *testing.T) {
	store := NewMemoryStore()

	code := "test-code"
	authCode := &models.OAuth2AuthCode{
		Code:      code,
		ClientID:  "test-client",
		Scopes:    "read",
		RedirectURI: "http://localhost/callback",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(-10 * time.Minute),
		Used:      false,
	}

	store.SaveAuthCode(authCode)
	store.CleanExpired()

	_, err := store.GetAuthCode(code)
	if err == nil {
		t.Error("expected error after cleaning expired codes")
	}
}
