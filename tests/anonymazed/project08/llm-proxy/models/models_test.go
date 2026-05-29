package models

import (
	"testing"
	"time"
)

func TestAuthCodeFields(t *testing.T) {
	code := &OAuth2AuthCode{
		Code:          "test-code",
		ClientID:      "test-client",
		Scopes:        "read,write",
		RedirectURI:   "http://localhost/callback",
		Nonce:         "nonce-123",
		CreatedAt:     time.Now(),
		ExpiresAt:     time.Now().Add(10 * time.Minute),
		Used:          false,
	}

	if code.Code != "test-code" {
		t.Errorf("expected test-code, got %s", code.Code)
	}
	if code.ClientID != "test-client" {
		t.Errorf("expected test-client, got %s", code.ClientID)
	}
}

func TestTokenFields(t *testing.T) {
	token := &OAuth2Token{
		AccessToken:  "access-token-123",
		RefreshToken: "refresh-token-456",
		ExpiresAt:    time.Now().Add(1 * time.Hour),
		Scopes:       "read",
		Code:         "token-code",
		ClientID:     "test-client",
		Nonce:        "",
		CreatedAt:    time.Now(),
	}

	if token.AccessToken != "access-token-123" {
		t.Errorf("expected access-token-123, got %s", token.AccessToken)
	}
	if token.RefreshToken != "refresh-token-456" {
		t.Errorf("expected refresh-token-456, got %s", token.RefreshToken)
	}
}

func TestUserInfoFields(t *testing.T) {
	userInfo := &UserInfo{
		ID:          "user-123",
		Email:       "test@example.com",
		DisplayName: "Test User",
		Picture:     "https://example.com/avatar.jpg",
		Profile:     "https://example.com/profile",
	}

	if userInfo.ID != "user-123" {
		t.Errorf("expected user-123, got %s", userInfo.ID)
	}
	if userInfo.Email != "test@example.com" {
		t.Errorf("expected test@example.com, got %s", userInfo.Email)
	}
}

func TestTokenExpiration(t *testing.T) {
	now := time.Now()
	token := &OAuth2Token{
		AccessToken:  "test-token",
		ClientID:     "test-client",
		ExpiresAt:    now.Add(1 * time.Hour),
		RefreshToken: "test-refresh",
		CreatedAt:    now,
	}

	if token.ExpiresAt.Sub(now) != 1*time.Hour {
		t.Errorf("expected 1 hour expiration, got %v", token.ExpiresAt.Sub(now))
	}
}
