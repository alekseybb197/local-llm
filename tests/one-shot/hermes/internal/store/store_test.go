package store

import (
	"context"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func TestInMemoryStore_SaveState(t *testing.T) {
	store := NewInMemoryStore(15 * time.Minute)

	state := &State{
		State:     "test-state",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	err := store.SaveState(context.Background(), state)
	if err != nil {
		t.Fatalf("SaveState failed: %v", err)
	}

	// Try to save duplicate state
	err = store.SaveState(context.Background(), state)
	if err != ErrStateExists {
		t.Errorf("Expected ErrStateExists, got: %v", err)
	}
}

func TestInMemoryStore_GetState(t *testing.T) {
	store := NewInMemoryStore(15 * time.Minute)

	state := &State{
		State:     "test-state",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	err := store.SaveState(context.Background(), state)
	if err != nil {
		t.Fatalf("SaveState failed: %v", err)
	}

	// Get state
	st, err := store.GetState(context.Background(), "test-state")
	if err != nil {
		t.Fatalf("GetState failed: %v", err)
	}

	if st.State != "test-state" {
		t.Errorf("Expected state 'test-state', got: %s", st.State)
	}

	// Get non-existent state
	_, err = store.GetState(context.Background(), "non-existent")
	if err != ErrStateNotFound {
		t.Errorf("Expected ErrStateNotFound, got: %v", err)
	}
}

func TestInMemoryStore_DeleteState(t *testing.T) {
	store := NewInMemoryStore(15 * time.Minute)

	state := &State{
		State:     "test-state",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	err := store.SaveState(context.Background(), state)
	if err != nil {
		t.Fatalf("SaveState failed: %v", err)
	}

	err = store.DeleteState(context.Background(), "test-state")
	if err != nil {
		t.Fatalf("DeleteState failed: %v", err)
	}

	// Try to get deleted state
	_, err = store.GetState(context.Background(), "test-state")
	if err != ErrStateNotFound {
		t.Errorf("Expected ErrStateNotFound, got: %v", err)
	}
}

func TestInMemoryStore_SaveTokenAndUserInfo(t *testing.T) {
	store := NewInMemoryStore(15 * time.Minute)

	token := &oauth2.Token{
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
		TokenType:    "Bearer",
	}

	userInfo := map[string]interface{}{
		"sub": "1234567890",
		"email": "test@example.com",
		"name": "Test User",
	}

	err := store.SaveTokenAndUserInfo(context.Background(), "test-state", token, userInfo)
	if err != nil {
		t.Fatalf("SaveTokenAndUserInfo failed: %v", err)
	}

	// Try to save duplicate token
	err = store.SaveTokenAndUserInfo(context.Background(), "test-state", token, userInfo)
	if err != ErrTokenExists {
		t.Errorf("Expected ErrTokenExists, got: %v", err)
	}
}

func TestInMemoryStore_GetTokenAndUserInfo(t *testing.T) {
	store := NewInMemoryStore(15 * time.Minute)

	token := &oauth2.Token{
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
		TokenType:    "Bearer",
	}

	userInfo := map[string]interface{}{
		"sub": "1234567890",
		"email": "test@example.com",
	}

	err := store.SaveTokenAndUserInfo(context.Background(), "test-state", token, userInfo)
	if err != nil {
		t.Fatalf("SaveTokenAndUserInfo failed: %v", err)
	}

	// Get token and user info
	tok, err := store.GetTokenAndUserInfo(context.Background(), "test-state")
	if err != nil {
		t.Fatalf("GetTokenAndUserInfo failed: %v", err)
	}

	if tok.Token.AccessToken != "test-access-token" {
		t.Errorf("Expected access token 'test-access-token', got: %s", tok.Token.AccessToken)
	}

	if tok.UserInfo["email"] != "test@example.com" {
		t.Errorf("Expected email 'test@example.com', got: %s", tok.UserInfo["email"])
	}

	// Get non-existent token
	_, err = store.GetTokenAndUserInfo(context.Background(), "non-existent")
	if err != ErrTokenNotFound {
		t.Errorf("Expected ErrTokenNotFound, got: %v", err)
	}
}

func TestInMemoryStore_StateExpiration(t *testing.T) {
	store := NewInMemoryStore(15 * time.Minute)

	state := &State{
		State:     "test-state",
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}

	err := store.SaveState(context.Background(), state)
	if err != nil {
		t.Fatalf("SaveState failed: %v", err)
	}

	// Get expired state
	_, err = store.GetState(context.Background(), "test-state")
	if err != ErrStateExpired {
		t.Errorf("Expected ErrStateExpired, got: %v", err)
	}
}

func TestInMemoryStore_TokenExpiration(t *testing.T) {
	store := NewInMemoryStore(15 * time.Minute)

	token := &oauth2.Token{
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
		TokenType:    "Bearer",
	}

	userInfo := map[string]interface{}{"email": "test@example.com"}

	err := store.SaveTokenAndUserInfo(context.Background(), "test-state", token, userInfo)
	if err != nil {
		t.Fatalf("SaveTokenAndUserInfo failed: %v", err)
	}

	// Manually expire the token
	store.mu.Lock()
	store.tokens["test-state"].ExpiresAt = time.Now().Add(-1 * time.Hour)
	store.mu.Unlock()

	// Get expired token
	_, err = store.GetTokenAndUserInfo(context.Background(), "test-state")
	if err != ErrTokenExpired {
		t.Errorf("Expected ErrTokenExpired, got: %v", err)
	}
}

func TestGenerateState(t *testing.T) {
	state, err := GenerateState()
	if err != nil {
		t.Fatalf("GenerateState failed: %v", err)
	}

	if state == "" {
		t.Error("Expected non-empty state")
	}

	// Generate another state - should be different
	state2, err := GenerateState()
	if err != nil {
		t.Fatalf("GenerateState failed: %v", err)
	}

	if state == state2 {
		t.Error("Expected different states")
	}
}

func TestGenerateCode(t *testing.T) {
	code, err := GenerateCode()
	if err != nil {
		t.Fatalf("GenerateCode failed: %v", err)
	}

	if code == "" {
		t.Error("Expected non-empty code")
	}
}

func TestVerifyCode(t *testing.T) {
	valid, err := VerifyCode("test-code")
	if !valid {
		t.Error("Expected valid code")
	}
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

func TestError(t *testing.T) {
	err := &Error{Code: "TEST_ERROR", Message: "test message"}
	if err.Error() != "TEST_ERROR" {
		t.Errorf("Expected 'TEST_ERROR', got: %s", err.Error())
	}
}
